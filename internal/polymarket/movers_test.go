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
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestMoversScoreLeaderDirectionAndSorting(t *testing.T) {
	now := time.Date(2026, 6, 3, 12, 0, 0, 0, time.UTC)
	high, ok := mapHotMarket(validHotMarket("cond-high", 100, 50))
	if !ok {
		t.Fatal("valid high hot market rejected")
	}
	low, ok := mapHotMarket(validHotMarket("cond-low", 200, 50))
	if !ok {
		t.Fatal("valid low hot market rejected")
	}

	highYes := high.Tokens[0].TokenID
	highNo := high.Tokens[1].TokenID
	lowYes := low.Tokens[0].TokenID
	svc := NewService(polymarketstore.NewSQLStore(nil), WithGammaClient(&fakeGammaClient{}), WithSportsWSClient(&fakeSportsWSClient{}), WithCLOBMarketWSClient(&fakeCLOBMarketWSClient{}))
	svc.started = true
	svc.nowFn = func() time.Time { return now }
	svc.hotMarketItems = []*v1alpha1.PolymarketHotMarketItem{low, high}
	svc.hotMarketFetched = now.Unix()
	svc.realtimeConnected = true
	svc.realtimeLastEventAt = now.Unix()
	svc.realtimeFetched = now.Unix()
	svc.realtimeStates[highYes] = &realtimeTokenState{tokenID: highYes, outcome: "Yes", price: 0.62, bestBid: 0.61, bestAsk: 0.63, spread: 0.02, lastTradePrice: 0.62, lastTradeSize: 4, lastTradeSide: "BUY", lastEventAt: now.Unix()}
	svc.realtimeStates[highNo] = &realtimeTokenState{tokenID: highNo, outcome: "No", price: 0.29, bestBid: 0.28, bestAsk: 0.3, spread: 0.02, lastTradePrice: 0.29, lastTradeSize: 6, lastTradeSide: "SELL", lastEventAt: now.Unix()}
	svc.realtimeStates[lowYes] = &realtimeTokenState{tokenID: lowYes, outcome: "Yes", price: 0.53, bestBid: 0.52, bestAsk: 0.54, spread: 0.02, lastTradePrice: 0.53, lastTradeSize: 2, lastTradeSide: "BUY", lastEventAt: now.Unix()}
	svc.realtimeSamples[highYes] = []realtimeSample{{at: now.Add(-time.Minute).Unix(), price: 0.60}}
	svc.realtimeSamples[highNo] = []realtimeSample{{at: now.Add(-5 * time.Minute).Unix(), price: 0.39}, {at: now.Add(-time.Minute).Unix(), price: 0.34}}
	svc.realtimeSamples[lowYes] = []realtimeSample{{at: now.Add(-time.Minute).Unix(), price: 0.50}}

	resp := svc.currentMoverMarketResponse(10)
	if resp == nil || len(resp.GetItems()) != 2 {
		t.Fatalf("response = %#v, want two movers", resp)
	}
	if resp.GetItems()[0].ConditionID != "cond-high" {
		t.Fatalf("first mover = %s, want cond-high", resp.GetItems()[0].ConditionID)
	}
	leader := resp.GetItems()[0].Leader
	if leader == nil || leader.TokenID != highNo || leader.Direction != "down" {
		t.Fatalf("leader = %#v, want high No down leader", leader)
	}
	if math.Abs(leader.Score-11) > 0.000001 {
		t.Fatalf("leader score = %.6f, want 11", leader.Score)
	}
	if resp.GetItems()[0].Score != leader.Score || resp.GetItems()[0].Direction != "down" || len(resp.GetItems()[0].Tokens) != 2 {
		t.Fatalf("market item = %#v, want leader score and all displayable tokens", resp.GetItems()[0])
	}
	if resp.GetMonitoredMarkets() != 2 || resp.GetMonitoredTokens() != 4 || !resp.GetConnected() || resp.GetFetchedAt() != now.Unix() || resp.GetLastEventAt() != now.Unix() {
		t.Fatalf("metadata = %#v, want realtime metadata", resp)
	}
}

func TestMoversFiltersWarmupStaleAndInvalidPrices(t *testing.T) {
	now := time.Date(2026, 6, 3, 12, 0, 0, 0, time.UTC)
	fresh, _ := mapHotMarket(validHotMarket("cond-fresh", 100, 50))
	warmup, _ := mapHotMarket(validHotMarket("cond-warmup", 90, 50))
	stale, _ := mapHotMarket(validHotMarket("cond-stale", 80, 50))
	invalid, _ := mapHotMarket(validHotMarket("cond-invalid", 70, 50))
	svc := NewService(polymarketstore.NewSQLStore(nil), WithGammaClient(&fakeGammaClient{}), WithSportsWSClient(&fakeSportsWSClient{}), WithCLOBMarketWSClient(&fakeCLOBMarketWSClient{}))
	svc.started = true
	svc.nowFn = func() time.Time { return now }
	svc.hotMarketItems = []*v1alpha1.PolymarketHotMarketItem{fresh, warmup, stale, invalid}
	svc.hotMarketFetched = now.Unix()
	svc.realtimeStale = true

	freshToken := fresh.Tokens[0].TokenID
	warmupToken := warmup.Tokens[0].TokenID
	staleToken := stale.Tokens[0].TokenID
	invalidToken := invalid.Tokens[0].TokenID
	svc.realtimeStates[freshToken] = &realtimeTokenState{tokenID: freshToken, price: 0.55, lastEventAt: now.Unix()}
	svc.realtimeStates[warmupToken] = &realtimeTokenState{tokenID: warmupToken, price: 0.55, lastEventAt: now.Unix()}
	svc.realtimeStates[staleToken] = &realtimeTokenState{tokenID: staleToken, price: 0.55, lastEventAt: now.Add(-121 * time.Second).Unix()}
	svc.realtimeStates[invalidToken] = &realtimeTokenState{tokenID: invalidToken, price: 1.01, lastEventAt: now.Unix()}
	svc.realtimeSamples[freshToken] = []realtimeSample{{at: now.Add(-time.Minute).Unix(), price: 0.50}}

	resp := svc.currentMoverMarketResponse(10)
	if resp == nil || len(resp.GetItems()) != 1 || resp.GetItems()[0].ConditionID != "cond-fresh" {
		t.Fatalf("response = %#v, want only fresh mover", resp)
	}
	if !resp.GetStale() {
		t.Fatal("stale = false, want stale metadata to be preserved")
	}
}

func TestListPolymarketMoversLimitValidation(t *testing.T) {
	svc := NewService(polymarketstore.NewSQLStore(nil), WithGammaClient(&fakeGammaClient{}), WithSportsWSClient(&fakeSportsWSClient{}), WithCLOBMarketWSClient(&fakeCLOBMarketWSClient{}))
	svc.started = true
	_, err := svc.ListPolymarketMovers(context.Background(), &apiclient.ListPolymarketMoversRequest{Limit: maxMoverListLimit + 1})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("ListPolymarketMovers error = %v, want InvalidArgument", err)
	}
}

func TestMoversStableSortingFallback(t *testing.T) {
	now := time.Date(2026, 6, 3, 12, 0, 0, 0, time.UTC)
	svc := NewService(polymarketstore.NewSQLStore(nil), WithGammaClient(&fakeGammaClient{}), WithSportsWSClient(&fakeSportsWSClient{}), WithCLOBMarketWSClient(&fakeCLOBMarketWSClient{}))
	svc.started = true
	svc.nowFn = func() time.Time { return now }
	for _, conditionID := range []string{"cond-b", "cond-a"} {
		item, ok := mapHotMarket(validHotMarket(conditionID, 100, 50))
		if !ok {
			t.Fatalf("valid market %s rejected", conditionID)
		}
		tokenID := item.Tokens[0].TokenID
		svc.hotMarketItems = append(svc.hotMarketItems, item)
		svc.realtimeStates[tokenID] = &realtimeTokenState{tokenID: tokenID, price: 0.55, lastEventAt: now.Unix()}
		svc.realtimeSamples[tokenID] = []realtimeSample{{at: now.Add(-time.Minute).Unix(), price: 0.50}}
	}

	resp := svc.currentMoverMarketResponse(10)
	if len(resp.GetItems()) != 2 {
		t.Fatalf("items = %d, want 2", len(resp.GetItems()))
	}
	if got := fmt.Sprintf("%s,%s", resp.GetItems()[0].ConditionID, resp.GetItems()[1].ConditionID); got != "cond-a,cond-b" {
		t.Fatalf("order = %s, want condition id fallback", got)
	}
}
