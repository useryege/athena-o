package polymarket

import (
	"context"
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

type fakeCLOBMarketWSClient struct {
	runFn func(context.Context, utilpolymarket.CLOBMarketWSSubscription, utilpolymarket.CLOBMarketWSHandler) error
}

func (f *fakeCLOBMarketWSClient) Run(ctx context.Context, sub utilpolymarket.CLOBMarketWSSubscription, handler utilpolymarket.CLOBMarketWSHandler) error {
	if f.runFn != nil {
		return f.runFn(ctx, sub, handler)
	}
	<-ctx.Done()
	return nil
}

func TestRealtimeSubscriptionSnapshotUsesTop500AndDedupesTokens(t *testing.T) {
	svc := NewService(polymarketstore.NewSQLStore(nil), WithGammaClient(&fakeGammaClient{}), WithSportsWSClient(&fakeSportsWSClient{}), WithCLOBMarketWSClient(&fakeCLOBMarketWSClient{}))
	svc.started = true
	for i := 0; i < hotMarketTargetLimit+5; i++ {
		item, ok := mapHotMarket(validHotMarket(fmt.Sprintf("cond-%03d", i), float64(1000-i), float64(500-i)))
		if !ok {
			t.Fatalf("valid hot market %d was rejected", i)
		}
		if i == 1 {
			item.Tokens[0].TokenID = "token-cond-000-yes"
		}
		svc.hotMarketItems = append(svc.hotMarketItems, item)
	}

	snapshot := svc.currentRealtimeSubscriptionSnapshot()
	if len(snapshot.markets) != hotMarketTargetLimit {
		t.Fatalf("markets = %d, want %d", len(snapshot.markets), hotMarketTargetLimit)
	}
	if len(snapshot.tokenIDs) != hotMarketTargetLimit*2-1 {
		t.Fatalf("token ids = %d, want deduped %d", len(snapshot.tokenIDs), hotMarketTargetLimit*2-1)
	}
	if snapshot.hash == "" {
		t.Fatal("hash is empty")
	}
}

func TestRealtimeEventsAndWindowChanges(t *testing.T) {
	now := time.Date(2026, 6, 3, 12, 0, 0, 0, time.UTC)
	item, ok := mapHotMarket(validHotMarket("cond-1", 100, 50))
	if !ok {
		t.Fatal("valid hot market rejected")
	}
	item.EventSlug = "event-cond-1"
	tokenID := item.Tokens[0].TokenID
	svc := NewService(polymarketstore.NewSQLStore(nil), WithGammaClient(&fakeGammaClient{}), WithSportsWSClient(&fakeSportsWSClient{}), WithCLOBMarketWSClient(&fakeCLOBMarketWSClient{}))
	svc.started = true
	svc.nowFn = func() time.Time { return now }
	svc.hotMarketItems = []*v1alpha1.PolymarketHotMarketItem{item}
	svc.hotMarketFetched = now.Unix()

	svc.applyRealtimeBook(utilpolymarket.CLOBMarketBookEvent{
		AssetID:   tokenID,
		Bids:      []utilpolymarket.CLOBOrderSummary{{Price: "0.52"}, {Price: "0.51"}},
		Asks:      []utilpolymarket.CLOBOrderSummary{{Price: "0.55"}, {Price: "0.56"}},
		Timestamp: now.Format(time.RFC3339),
	})
	svc.applyRealtimePriceChange(utilpolymarket.CLOBMarketPriceChangeEvent{
		Timestamp: now.Add(10 * time.Second).Format(time.RFC3339),
		PriceChanges: []utilpolymarket.CLOBMarketPriceChange{{
			AssetID: tokenID,
			Price:   "0.57",
			BestBid: strPtr("0.56"),
			BestAsk: strPtr("0.58"),
		}},
	})
	svc.applyRealtimeLastTrade(utilpolymarket.CLOBMarketLastTradePriceEvent{
		AssetID:   tokenID,
		Price:     "0.59",
		Size:      "10",
		Side:      "BUY",
		Timestamp: now.Add(20 * time.Second).Format(time.RFC3339),
	})

	svc.cacheMu.Lock()
	svc.realtimeSamples[tokenID] = []realtimeSample{{at: now.Add(-time.Minute).Unix(), price: 0.52}}
	svc.cacheMu.Unlock()

	resp := svc.currentRealtimeMarketResponse(1)
	if resp == nil || len(resp.GetItems()) != 1 || len(resp.GetItems()[0].Tokens) == 0 {
		t.Fatalf("response missing token: %#v", resp)
	}
	if resp.GetItems()[0].EventSlug != "event-cond-1" {
		t.Fatalf("event slug = %q, want inherited event slug", resp.GetItems()[0].EventSlug)
	}
	token := resp.GetItems()[0].Tokens[0]
	if token.Price != 0.57 || token.BestBid != 0.56 || token.BestAsk != 0.58 || math.Abs(token.Spread-0.02) > 0.000001 || token.LastTradePrice != 0.59 || token.LastTradeSize != 10 || token.LastTradeSide != "BUY" {
		t.Fatalf("token state = %#v, want ws fields applied", token)
	}
	if len(token.Windows) != 3 || token.Windows[0].Warmup || fmt.Sprintf("%.1f", token.Windows[0].PriceChangePp) != "5.0" {
		t.Fatalf("1m window = %#v, want +5.0pp and not warmup", token.Windows)
	}
	if !token.Windows[1].Warmup || !token.Windows[2].Warmup {
		t.Fatalf("longer windows should be warmup: %#v", token.Windows)
	}
	if !resp.GetConnected() || resp.GetLastEventAt() != now.Add(20*time.Second).Unix() {
		t.Fatalf("connection metadata connected=%v last=%d", resp.GetConnected(), resp.GetLastEventAt())
	}
}

func TestRealtimeSamplingPrunesAndWarmup(t *testing.T) {
	now := time.Date(2026, 6, 3, 12, 0, 0, 0, time.UTC)
	svc := NewService(polymarketstore.NewSQLStore(nil), WithGammaClient(&fakeGammaClient{}), WithSportsWSClient(&fakeSportsWSClient{}), WithCLOBMarketWSClient(&fakeCLOBMarketWSClient{}))
	svc.realtimeStates["token-1"] = &realtimeTokenState{tokenID: "token-1", price: 0.6}
	svc.realtimeSamples["token-1"] = []realtimeSample{{at: now.Add(-20 * time.Minute).Unix(), price: 0.4}}

	svc.sampleRealtime(now)
	samples := svc.realtimeSamples["token-1"]
	if len(samples) != 1 || samples[0].price != 0.6 || samples[0].at != now.Unix() {
		t.Fatalf("samples = %#v, want old sample pruned and current sample kept", samples)
	}
	window := svc.realtimeWindowItemLocked("token-1", now.Unix(), realtimeWindow1m, 0.6)
	if !window.Warmup {
		t.Fatalf("window warmup = false, want true for insufficient samples")
	}
}

func TestRealtimeStaleFallbackAndLimitValidation(t *testing.T) {
	item, ok := mapHotMarket(validHotMarket("cond-1", 100, 50))
	if !ok {
		t.Fatal("valid hot market rejected")
	}
	svc := NewService(polymarketstore.NewSQLStore(nil), WithGammaClient(&fakeGammaClient{}), WithSportsWSClient(&fakeSportsWSClient{}), WithCLOBMarketWSClient(&fakeCLOBMarketWSClient{}))
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
