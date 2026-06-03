package polymarket

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/useryege/athena/internal/polymarket/apiclient"
	polymarketstore "github.com/useryege/athena/internal/polymarket/store"
	"github.com/useryege/athena/pkg/apis/application/v1alpha1"
	utilpolymarket "github.com/useryege/athena/util/polymarket"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestHotMarketDiscoveryFiltersSortsAndParses(t *testing.T) {
	cursor := "next"
	updated := time.Date(2026, 6, 3, 1, 2, 3, 0, time.UTC)
	gamma := &fakeGammaClient{marketResponses: []*utilpolymarket.MarketKeysetResponse{
		{
			Markets: []utilpolymarket.Market{
				validHotMarket("cond-low", 10, 4),
				invalidHotMarketNoToken("cond-no-token"),
				validHotMarket("cond-high", 20, 3),
			},
			NextCursor: &cursor,
		},
		{
			Markets: []utilpolymarket.Market{
				invalidHotMarketInactive("cond-inactive"),
				validHotMarketWithUpdatedAt("cond-tie-a", 20, 9, updated),
				validHotMarket("cond-tie-b", 20, 8),
			},
		},
	}}
	svc := NewService(polymarketstore.NewSQLStore(nil), WithGammaClient(gamma), WithSportsWSClient(&fakeSportsWSClient{}))
	svc.started = true

	if err := svc.refreshHotMarkets(context.Background()); err != nil {
		t.Fatalf("refreshHotMarkets: %v", err)
	}
	if gamma.marketCalls != 2 {
		t.Fatalf("market calls = %d, want 2", gamma.marketCalls)
	}
	if gamma.marketOpts[0].Order != "volume24hr" || gamma.marketOpts[0].Ascending == nil || *gamma.marketOpts[0].Ascending || gamma.marketOpts[0].Closed == nil || *gamma.marketOpts[0].Closed {
		t.Fatalf("unexpected market opts: %#v", gamma.marketOpts[0])
	}
	if gamma.marketOpts[1].AfterCursor != cursor {
		t.Fatalf("after cursor = %q, want %q", gamma.marketOpts[1].AfterCursor, cursor)
	}

	resp := svc.currentHotMarketResponse(10)
	if resp == nil {
		t.Fatal("response is nil")
	}
	if resp.GetCandidateCount() != 4 || resp.GetMonitoredMarkets() != 4 || resp.GetMonitoredTokens() != 8 {
		t.Fatalf("response counts = candidates:%d markets:%d tokens:%d", resp.GetCandidateCount(), resp.GetMonitoredMarkets(), resp.GetMonitoredTokens())
	}
	got := hotMarketIDs(resp.GetItems())
	want := []string{"cond-tie-a", "cond-tie-b", "cond-high", "cond-low"}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("order = %v, want %v", got, want)
	}
	first := resp.GetItems()[0]
	if first.UpdatedAt != updated.Format(time.RFC3339) {
		t.Fatalf("updated_at = %q, want %q", first.UpdatedAt, updated.Format(time.RFC3339))
	}
	if len(first.Tokens) != 2 || first.Tokens[0].Outcome != "Yes" || first.Tokens[0].Price != 0.6 {
		t.Fatalf("tokens not parsed: %#v", first.Tokens)
	}
}

func TestHotMarketEventSlugMapping(t *testing.T) {
	sameSlugMarket := validHotMarket("cond-same", 20, 5)
	sameSlugMarket.Events = []utilpolymarket.Event{{Slug: sameSlugMarket.Slug}}
	sameSlug, ok := mapHotMarket(sameSlugMarket)
	if !ok {
		t.Fatal("same slug market rejected")
	}
	if sameSlug.EventSlug != sameSlug.MarketSlug {
		t.Fatalf("event slug = %q, want market slug %q", sameSlug.EventSlug, sameSlug.MarketSlug)
	}

	eventSlug := "us-x-iran-permanent-peace-deal-by"
	differentSlugMarket := validHotMarket("cond-different", 20, 5)
	differentSlugMarket.Slug = strPtr("us-x-iran-permanent-peace-deal-by-june-15-2026-734-856-129")
	differentSlugMarket.Events = []utilpolymarket.Event{{Slug: &eventSlug}}
	differentSlug, ok := mapHotMarket(differentSlugMarket)
	if !ok {
		t.Fatal("different slug market rejected")
	}
	if differentSlug.MarketSlug != "us-x-iran-permanent-peace-deal-by-june-15-2026-734-856-129" || differentSlug.EventSlug != eventSlug {
		t.Fatalf("slugs = market:%q event:%q, want market slug preserved and event slug mapped", differentSlug.MarketSlug, differentSlug.EventSlug)
	}

	noEvent, ok := mapHotMarket(validHotMarket("cond-no-event", 20, 5))
	if !ok {
		t.Fatal("no event market rejected")
	}
	if noEvent.EventSlug != "" {
		t.Fatalf("event slug = %q, want empty fallback", noEvent.EventSlug)
	}
}

func TestHotMarketDiscoveryHysteresis(t *testing.T) {
	first := makeHotMarketBatch("old", 500, 1000)
	second := makeHotMarketBatch("new", 500, 2000)
	second = append(second, validHotMarket("old-000", 1, 1))
	gamma := &fakeGammaClient{marketResponses: []*utilpolymarket.MarketKeysetResponse{
		{Markets: first},
		{Markets: second},
		{Markets: makeHotMarketBatch("new", 500, 2000)},
		{Markets: makeHotMarketBatch("new", 500, 2000)},
		{Markets: makeHotMarketBatch("new", 500, 2000)},
	}}
	svc := NewService(polymarketstore.NewSQLStore(nil), WithGammaClient(gamma), WithSportsWSClient(&fakeSportsWSClient{}))
	svc.started = true

	if err := svc.refreshHotMarkets(context.Background()); err != nil {
		t.Fatalf("first refreshHotMarkets: %v", err)
	}
	if len(svc.hotMarketItems) != 500 {
		t.Fatalf("first hot markets = %d, want 500", len(svc.hotMarketItems))
	}
	if err := svc.refreshHotMarkets(context.Background()); err != nil {
		t.Fatalf("second refreshHotMarkets: %v", err)
	}
	if len(svc.hotMarketItems) != hotMarketCandidateLimit {
		t.Fatalf("second hot markets = %d, want capped retained %d", len(svc.hotMarketItems), hotMarketCandidateLimit)
	}

	for i := 0; i < hotMarketMissingThreshold-1; i++ {
		if err := svc.refreshHotMarkets(context.Background()); err != nil {
			t.Fatalf("missing refresh %d: %v", i, err)
		}
		if !hasHotMarketID(svc.hotMarketItems, "old-000") {
			t.Fatalf("old-000 removed too early after %d missing refreshes", i+1)
		}
	}
	if err := svc.refreshHotMarkets(context.Background()); err != nil {
		t.Fatalf("final missing refresh: %v", err)
	}
	if hasHotMarketID(svc.hotMarketItems, "old-000") {
		t.Fatal("old-000 retained after missing threshold")
	}
}

func TestHotMarketStaleFallbackOnRefreshError(t *testing.T) {
	gamma := &fakeGammaClient{marketResponses: []*utilpolymarket.MarketKeysetResponse{{
		Markets: []utilpolymarket.Market{validHotMarket("cond-1", 10, 1)},
	}}}
	svc := NewService(polymarketstore.NewSQLStore(nil), WithGammaClient(gamma), WithSportsWSClient(&fakeSportsWSClient{}))
	svc.started = true
	if err := svc.refreshHotMarkets(context.Background()); err != nil {
		t.Fatalf("refreshHotMarkets: %v", err)
	}
	gamma.marketErr = errors.New("boom")
	if err := svc.refreshHotMarkets(context.Background()); err == nil {
		t.Fatal("expected refresh error")
	}
	resp := svc.currentHotMarketResponse(10)
	if resp == nil {
		t.Fatal("response is nil")
	}
	if !resp.GetStale() {
		t.Fatal("stale = false, want true")
	}
	if len(resp.GetItems()) != 1 || resp.GetItems()[0].ConditionID != "cond-1" {
		t.Fatalf("cached items = %#v", resp.GetItems())
	}
}

func TestListHotMarketsValidation(t *testing.T) {
	svc := NewService(polymarketstore.NewSQLStore(nil), WithGammaClient(&fakeGammaClient{}), WithSportsWSClient(&fakeSportsWSClient{}))
	svc.started = true
	_, err := svc.ListPolymarketHotMarkets(context.Background(), &apiclient.ListPolymarketHotMarketsRequest{Limit: maxHotMarketListLimit + 1})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("ListPolymarketHotMarkets error = %v, want InvalidArgument", err)
	}
}

func TestListHotMarketsColdStartFailure(t *testing.T) {
	gamma := &fakeGammaClient{marketErr: errors.New("unavailable")}
	svc := NewService(polymarketstore.NewSQLStore(nil), WithGammaClient(gamma), WithSportsWSClient(&fakeSportsWSClient{}))
	svc.started = true
	_, err := svc.ListPolymarketHotMarkets(context.Background(), &apiclient.ListPolymarketHotMarketsRequest{Limit: 1})
	if status.Code(err) != codes.Unavailable {
		t.Fatalf("ListPolymarketHotMarkets error = %v, want Unavailable", err)
	}
}

func validHotMarket(conditionID string, volume24hr, liquidity float64) utilpolymarket.Market {
	return validHotMarketWithUpdatedAt(conditionID, volume24hr, liquidity, time.Time{})
}

func validHotMarketWithUpdatedAt(conditionID string, volume24hr, liquidity float64, updatedAt time.Time) utilpolymarket.Market {
	active := true
	closed := false
	enableOrderBook := true
	tokens := `["token-` + conditionID + `-yes","token-` + conditionID + `-no"]`
	outcomes := `["Yes","No"]`
	prices := `["0.6","0.4"]`
	market := utilpolymarket.Market{
		ConditionID:     strPtr(conditionID),
		Slug:            strPtr("market-" + conditionID),
		Question:        strPtr("Question " + conditionID),
		Active:          &active,
		Closed:          &closed,
		EnableOrderBook: &enableOrderBook,
		Volume24hr:      &volume24hr,
		LiquidityNum:    &liquidity,
		ClobTokenIDs:    &tokens,
		Outcomes:        &outcomes,
		OutcomePrices:   &prices,
		Spread:          floatPtr(0.02),
		BestBid:         floatPtr(0.59),
		BestAsk:         floatPtr(0.61),
		LastTradePrice:  floatPtr(0.6),
	}
	if !updatedAt.IsZero() {
		market.UpdatedAt = &updatedAt
	}
	return market
}

func invalidHotMarketNoToken(conditionID string) utilpolymarket.Market {
	market := validHotMarket(conditionID, 100, 100)
	market.ClobTokenIDs = strPtr("")
	return market
}

func invalidHotMarketInactive(conditionID string) utilpolymarket.Market {
	market := validHotMarket(conditionID, 100, 100)
	market.Active = boolPtr(false)
	return market
}

func makeHotMarketBatch(prefix string, count int, startVolume float64) []utilpolymarket.Market {
	out := make([]utilpolymarket.Market, 0, count)
	for i := 0; i < count; i++ {
		out = append(out, validHotMarket(fmt.Sprintf("%s-%03d", prefix, i), startVolume-float64(i), float64(count-i)))
	}
	return out
}

func hotMarketIDs(items []*v1alpha1.PolymarketHotMarketItem) []string {
	out := make([]string, 0, len(items))
	for i := range items {
		out = append(out, items[i].ConditionID)
	}
	return out
}

func hasHotMarketID(items []*v1alpha1.PolymarketHotMarketItem, id string) bool {
	for i := range items {
		if items[i] != nil && items[i].ConditionID == id {
			return true
		}
	}
	return false
}
