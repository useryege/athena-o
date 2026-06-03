package polymarket

import (
	"context"
	"errors"
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/useryege/athena/internal/polymarket/apiclient"
	polymarketstore "github.com/useryege/athena/internal/polymarket/store"
	"github.com/useryege/athena/pkg/apis/application/v1alpha1"
	utilpolymarket "github.com/useryege/athena/util/polymarket"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestRealtimeHotMarketRefreshSamplesGammaPrices(t *testing.T) {
	now := time.Date(2026, 6, 3, 12, 0, 0, 0, time.UTC)
	gamma := &fakeGammaClient{marketResponses: []*utilpolymarket.MarketKeysetResponse{{
		Markets: []utilpolymarket.Market{validHotMarket("cond-1", 100, 50)},
	}}}
	svc := NewService(polymarketstore.NewSQLStore(nil), WithGammaClient(gamma), WithSportsWSClient(&fakeSportsWSClient{}))
	svc.started = true
	svc.nowFn = func() time.Time { return now }

	if err := svc.refreshHotMarkets(context.Background()); err != nil {
		t.Fatalf("refreshHotMarkets: %v", err)
	}

	tokenID := svc.hotMarketItems[0].Tokens[0].TokenID
	state := svc.realtimeStates[tokenID]
	samples := svc.realtimeSamples[tokenID]
	if state == nil || state.price != 0.6 || state.lastEventAt != now.Unix() {
		t.Fatalf("state = %#v, want sampled Gamma token price", state)
	}
	if len(samples) != 1 || samples[0].at != now.Unix() || samples[0].price != 0.6 {
		t.Fatalf("samples = %#v, want one Gamma snapshot sample", samples)
	}
	resp := svc.currentRealtimeMarketResponse(10)
	if resp == nil || !resp.GetConnected() || resp.GetFetchedAt() != now.Unix() || resp.GetLastEventAt() != now.Unix() {
		t.Fatalf("metadata = %#v, want connected snapshot metadata", resp)
	}
	token := resp.GetItems()[0].Tokens[0]
	if token.Price != 0.6 || token.BestBid != 0.59 || token.BestAsk != 0.61 || token.Spread != 0.02 || token.LastTradePrice != 0.6 {
		t.Fatalf("token = %#v, want Gamma fields", token)
	}
}

func TestRealtimeWindowChangesFromGammaSamples(t *testing.T) {
	now := time.Date(2026, 6, 3, 12, 0, 0, 0, time.UTC)
	item, ok := mapHotMarket(validHotMarket("cond-1", 100, 50))
	if !ok {
		t.Fatal("valid hot market rejected")
	}
	item.EventSlug = "event-cond-1"
	tokenID := item.Tokens[0].TokenID
	item.Tokens[0].Price = 0.60
	svc := NewService(polymarketstore.NewSQLStore(nil), WithGammaClient(&fakeGammaClient{}), WithSportsWSClient(&fakeSportsWSClient{}))
	svc.started = true
	svc.nowFn = func() time.Time { return now }
	svc.hotMarketItems = []*v1alpha1.PolymarketHotMarketItem{item}
	svc.hotMarketFetched = now.Unix()
	svc.realtimeFetched = now.Unix()
	svc.realtimeConnected = true
	svc.realtimeLastEventAt = now.Unix()
	svc.realtimeStates[tokenID] = &realtimeTokenState{tokenID: tokenID, outcome: "Yes", price: 0.60, lastEventAt: now.Unix()}
	svc.realtimeSamples[tokenID] = []realtimeSample{
		{at: now.Add(-15 * time.Minute).Unix(), price: 0.30},
		{at: now.Add(-5 * time.Minute).Unix(), price: 0.40},
		{at: now.Add(-time.Minute).Unix(), price: 0.50},
	}

	resp := svc.currentRealtimeMarketResponse(1)
	if resp == nil || len(resp.GetItems()) != 1 || len(resp.GetItems()[0].Tokens) == 0 {
		t.Fatalf("response missing token: %#v", resp)
	}
	if resp.GetItems()[0].EventSlug != "event-cond-1" {
		t.Fatalf("event slug = %q, want inherited event slug", resp.GetItems()[0].EventSlug)
	}
	windows := resp.GetItems()[0].Tokens[0].Windows
	if len(windows) != 3 {
		t.Fatalf("windows = %#v, want 3", windows)
	}
	if windows[0].Warmup || math.Abs(windows[0].PriceChangePp-10) > 0.000001 {
		t.Fatalf("1m window = %#v, want +10pp", windows[0])
	}
	if windows[1].Warmup || math.Abs(windows[1].PriceChangePp-20) > 0.000001 {
		t.Fatalf("5m window = %#v, want +20pp", windows[1])
	}
	if windows[2].Warmup || math.Abs(windows[2].PriceChangePp-30) > 0.000001 {
		t.Fatalf("15m window = %#v, want +30pp", windows[2])
	}
}

func TestRealtimeWindowWarmupForInsufficientSamples(t *testing.T) {
	now := time.Date(2026, 6, 3, 12, 0, 0, 0, time.UTC)
	item, ok := mapHotMarket(validHotMarket("cond-1", 100, 50))
	if !ok {
		t.Fatal("valid hot market rejected")
	}
	tokenID := item.Tokens[0].TokenID
	svc := NewService(polymarketstore.NewSQLStore(nil), WithGammaClient(&fakeGammaClient{}), WithSportsWSClient(&fakeSportsWSClient{}))
	svc.nowFn = func() time.Time { return now }
	svc.hotMarketItems = []*v1alpha1.PolymarketHotMarketItem{item}
	svc.hotMarketFetched = now.Unix()
	svc.realtimeStates[tokenID] = &realtimeTokenState{tokenID: tokenID, price: 0.6, lastEventAt: now.Unix()}
	svc.realtimeSamples[tokenID] = []realtimeSample{{at: now.Unix(), price: 0.6}}

	resp := svc.currentRealtimeMarketResponse(1)
	windows := resp.GetItems()[0].Tokens[0].Windows
	if len(windows) != 3 || !windows[0].Warmup || !windows[1].Warmup || !windows[2].Warmup {
		t.Fatalf("windows = %#v, want all warmup", windows)
	}
}

func TestRealtimeRefreshErrorDoesNotSampleAndMarksStale(t *testing.T) {
	now := time.Date(2026, 6, 3, 12, 0, 0, 0, time.UTC)
	gamma := &fakeGammaClient{marketResponses: []*utilpolymarket.MarketKeysetResponse{{
		Markets: []utilpolymarket.Market{validHotMarket("cond-1", 100, 50)},
	}}}
	svc := NewService(polymarketstore.NewSQLStore(nil), WithGammaClient(gamma), WithSportsWSClient(&fakeSportsWSClient{}))
	svc.started = true
	svc.nowFn = func() time.Time { return now }
	if err := svc.refreshHotMarkets(context.Background()); err != nil {
		t.Fatalf("initial refreshHotMarkets: %v", err)
	}
	tokenID := svc.hotMarketItems[0].Tokens[0].TokenID
	if len(svc.realtimeSamples[tokenID]) != 1 {
		t.Fatalf("initial samples = %#v, want one sample", svc.realtimeSamples[tokenID])
	}

	gamma.marketErr = errors.New("boom")
	svc.nowFn = func() time.Time { return now.Add(time.Minute) }
	if err := svc.refreshHotMarkets(context.Background()); err == nil {
		t.Fatal("expected refresh error")
	}
	if len(svc.realtimeSamples[tokenID]) != 1 {
		t.Fatalf("samples = %#v, want no new sample on refresh failure", svc.realtimeSamples[tokenID])
	}
	resp := svc.currentRealtimeMarketResponse(1)
	if resp == nil || !resp.GetStale() || resp.GetConnected() {
		t.Fatalf("response = %#v, want stale disconnected snapshot", resp)
	}
}

func TestRealtimeMissingHysteresisMarketIsNotResampled(t *testing.T) {
	now := time.Date(2026, 6, 3, 12, 0, 0, 0, time.UTC)
	oldItem, _ := mapHotMarket(validHotMarket("cond-old", 100, 50))
	newItem, _ := mapHotMarket(validHotMarket("cond-new", 200, 50))
	svc := NewService(polymarketstore.NewSQLStore(nil), WithGammaClient(&fakeGammaClient{}), WithSportsWSClient(&fakeSportsWSClient{}))
	svc.hotMarketItems = []*v1alpha1.PolymarketHotMarketItem{oldItem}
	svc.sampleHotMarketCandidatesLocked([]hotMarketCandidate{{item: oldItem, rank: 0}}, now.Unix())

	oldToken := oldItem.Tokens[0].TokenID
	svc.applyHotMarketCandidatesLocked([]hotMarketCandidate{{item: newItem, rank: 0}})
	svc.sampleHotMarketCandidatesLocked([]hotMarketCandidate{{item: newItem, rank: 0}}, now.Add(time.Minute).Unix())

	if !hasHotMarketID(svc.hotMarketItems, "cond-old") {
		t.Fatal("cond-old should be retained by hysteresis")
	}
	if len(svc.realtimeSamples[oldToken]) != 1 || svc.realtimeStates[oldToken].lastEventAt != now.Unix() {
		t.Fatalf("old samples=%#v state=%#v, want retained market not resampled", svc.realtimeSamples[oldToken], svc.realtimeStates[oldToken])
	}
}

func TestRealtimeStaleFallbackAndLimitValidation(t *testing.T) {
	item, ok := mapHotMarket(validHotMarket("cond-1", 100, 50))
	if !ok {
		t.Fatal("valid hot market rejected")
	}
	svc := NewService(polymarketstore.NewSQLStore(nil), WithGammaClient(&fakeGammaClient{}), WithSportsWSClient(&fakeSportsWSClient{}))
	svc.started = true
	svc.hotMarketItems = []*v1alpha1.PolymarketHotMarketItem{item}
	svc.hotMarketFetched = 1717000000
	svc.ensureRealtimeStatesForMarketsLocked(svc.hotMarketItems)
	svc.markRealtimeStale()

	resp := svc.currentRealtimeMarketResponse(1)
	if resp == nil || !resp.GetStale() || len(resp.GetItems()) != 1 {
		t.Fatalf("response = %#v, want stale fallback with cached market", resp)
	}
	_, err := svc.ListPolymarketRealtimeMarkets(context.Background(), &apiclient.ListPolymarketRealtimeMarketsRequest{Limit: maxRealtimeListLimit + 1})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("ListPolymarketRealtimeMarkets error = %v, want InvalidArgument", err)
	}
}

func TestRealtimeTopHotMarketsUsesTop500(t *testing.T) {
	svc := NewService(polymarketstore.NewSQLStore(nil), WithGammaClient(&fakeGammaClient{}), WithSportsWSClient(&fakeSportsWSClient{}))
	for i := 0; i < hotMarketTargetLimit+5; i++ {
		item, ok := mapHotMarket(validHotMarket(fmt.Sprintf("cond-%03d", i), float64(1000-i), float64(500-i)))
		if !ok {
			t.Fatalf("valid hot market %d was rejected", i)
		}
		svc.hotMarketItems = append(svc.hotMarketItems, item)
	}

	markets := realtimeTopHotMarketsLocked(svc.hotMarketItems)
	if len(markets) != hotMarketTargetLimit {
		t.Fatalf("markets = %d, want %d", len(markets), hotMarketTargetLimit)
	}
}
