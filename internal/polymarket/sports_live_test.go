package polymarket

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/useryege/athena/internal/polymarket/apiclient"
	polymarketstore "github.com/useryege/athena/internal/polymarket/store"
	utilpolymarket "github.com/useryege/athena/util/polymarket"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type fakeGammaClient struct {
	responses       []*utilpolymarket.EventKeysetResponse
	marketResponses []*utilpolymarket.MarketKeysetResponse
	err             error
	marketErr       error
	calls           int
	marketCalls     int
	opts            []utilpolymarket.ListEventsKeysetOptions
	marketOpts      []utilpolymarket.ListMarketsKeysetOptions
}

func (f *fakeGammaClient) ListEventsKeyset(_ context.Context, options utilpolymarket.ListEventsKeysetOptions) (*utilpolymarket.EventKeysetResponse, error) {
	f.calls++
	f.opts = append(f.opts, options)
	if f.err != nil {
		return nil, f.err
	}
	if len(f.responses) == 0 {
		return &utilpolymarket.EventKeysetResponse{}, nil
	}
	idx := f.calls - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(f.responses) {
		idx = len(f.responses) - 1
	}
	return f.responses[idx], nil
}

func (f *fakeGammaClient) ListMarketsKeyset(_ context.Context, options utilpolymarket.ListMarketsKeysetOptions) (*utilpolymarket.MarketKeysetResponse, error) {
	f.marketCalls++
	f.marketOpts = append(f.marketOpts, options)
	if f.marketErr != nil {
		return nil, f.marketErr
	}
	if len(f.marketResponses) == 0 {
		return &utilpolymarket.MarketKeysetResponse{}, nil
	}
	idx := f.marketCalls - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(f.marketResponses) {
		idx = len(f.marketResponses) - 1
	}
	return f.marketResponses[idx], nil
}

type fakeSportsWSClient struct {
	runFn func(context.Context, utilpolymarket.SportsWSHandler) error
}

func (f *fakeSportsWSClient) Run(ctx context.Context, handler utilpolymarket.SportsWSHandler) error {
	if f.runFn != nil {
		return f.runFn(ctx, handler)
	}
	<-ctx.Done()
	return nil
}

func TestSportsLiveRefreshAndWSMerge(t *testing.T) {
	cursor := "next-page"
	gamma := &fakeGammaClient{responses: []*utilpolymarket.EventKeysetResponse{
		{
			Events: []utilpolymarket.Event{
				{
					Slug:  strPtr("event-a"),
					Title: strPtr("Event A"),
					Image: strPtr("event-a.png"),
					Markets: []utilpolymarket.Market{
						{ConditionID: strPtr("cond-a1"), Slug: strPtr("market-a1"), Question: strPtr("A1"), LiquidityNum: floatPtr(100), VolumeNum: floatPtr(10)},
						{ConditionID: strPtr("cond-a2"), Slug: strPtr("market-a2"), Question: strPtr("A2"), LiquidityNum: floatPtr(80), VolumeNum: floatPtr(8)},
					},
				},
			},
			NextCursor: &cursor,
		},
		{
			Events: []utilpolymarket.Event{
				{
					Slug:  strPtr("event-b"),
					Title: strPtr("Event B"),
					Image: strPtr("event-b.png"),
					Markets: []utilpolymarket.Market{
						{ConditionID: strPtr("cond-b1"), Slug: strPtr("market-b1"), Question: strPtr("B1"), LiquidityNum: floatPtr(1), VolumeNum: floatPtr(1)},
					},
				},
			},
		},
	}}

	svc := NewService(
		polymarketstore.NewSQLStore(nil),
		WithGammaClient(gamma),
		WithSportsWSClient(&fakeSportsWSClient{}),
		WithSportsLiveEventPageLimit(500),
	)
	svc.started = true

	if err := svc.refreshSportsLiveSnapshot(context.Background()); err != nil {
		t.Fatalf("refreshSportsLiveSnapshot: %v", err)
	}
	if gamma.calls != 2 {
		t.Fatalf("gamma calls = %d, want 2", gamma.calls)
	}
	if gamma.opts[0].TagSlug != "sports" || gamma.opts[0].Live == nil || !*gamma.opts[0].Live || gamma.opts[0].Closed == nil || *gamma.opts[0].Closed || !hasOnlyEsportsExcludeTag(gamma.opts[0].ExcludeTagID) {
		t.Fatalf("unexpected options[0] = %#v", gamma.opts[0])
	}
	if gamma.opts[1].AfterCursor != cursor {
		t.Fatalf("after_cursor = %q, want %q", gamma.opts[1].AfterCursor, cursor)
	}

	updateTs := "2026-06-01T22:30:00Z"
	svc.applySportsWSUpdate(utilpolymarket.SportsWSUpdate{Slug: "event-b", Score: strPtr("2-1"), LastUpdate: &updateTs, Live: boolPtr(true)})
	resp := svc.currentSportsLiveResponse(10)
	if resp == nil {
		t.Fatal("response is nil")
	}
	if len(resp.GetItems()) != 3 {
		t.Fatalf("items len = %d, want 3", len(resp.GetItems()))
	}
	if resp.GetItems()[0].ConditionID != "cond-b1" {
		t.Fatalf("first item condition = %q, want cond-b1", resp.GetItems()[0].ConditionID)
	}
	if resp.GetItems()[0].Score != "2-1" || resp.GetItems()[0].LastUpdate != updateTs {
		t.Fatalf("event-b ws fields not applied: %#v", resp.GetItems()[0])
	}
}

func TestSportsLiveEndedRemovesEventMarkets(t *testing.T) {
	gamma := &fakeGammaClient{responses: []*utilpolymarket.EventKeysetResponse{{
		Events: []utilpolymarket.Event{{
			Slug:    strPtr("event-a"),
			Markets: []utilpolymarket.Market{{ConditionID: strPtr("cond-a1"), Slug: strPtr("market-a1")}},
		}},
	}}}
	svc := NewService(polymarketstore.NewSQLStore(nil), WithGammaClient(gamma), WithSportsWSClient(&fakeSportsWSClient{}))
	svc.started = true
	if err := svc.refreshSportsLiveSnapshot(context.Background()); err != nil {
		t.Fatalf("refreshSportsLiveSnapshot: %v", err)
	}
	svc.applySportsWSUpdate(utilpolymarket.SportsWSUpdate{Slug: "event-a", Ended: boolPtr(true)})
	resp := svc.currentSportsLiveResponse(10)
	if resp == nil {
		t.Fatal("response is nil")
	}
	if len(resp.GetItems()) != 0 {
		t.Fatalf("items len = %d, want 0", len(resp.GetItems()))
	}
}

func TestSportsLiveStaleFallbackOnRefreshError(t *testing.T) {
	gamma := &fakeGammaClient{responses: []*utilpolymarket.EventKeysetResponse{{
		Events: []utilpolymarket.Event{{Slug: strPtr("event-a"), Markets: []utilpolymarket.Market{{ConditionID: strPtr("cond-a1"), Slug: strPtr("market-a1")}}}},
	}}}
	svc := NewService(polymarketstore.NewSQLStore(nil), WithGammaClient(gamma), WithSportsWSClient(&fakeSportsWSClient{}))
	svc.started = true
	if err := svc.refreshSportsLiveSnapshot(context.Background()); err != nil {
		t.Fatalf("refreshSportsLiveSnapshot: %v", err)
	}
	gamma.err = errors.New("boom")
	if err := svc.refreshSportsLiveSnapshot(context.Background()); err == nil {
		t.Fatal("expected refresh error")
	}
	resp := svc.currentSportsLiveResponse(10)
	if resp == nil {
		t.Fatal("response is nil")
	}
	if !resp.GetStale() {
		t.Fatal("stale = false, want true")
	}
	if len(resp.GetItems()) != 1 || resp.GetItems()[0].ConditionID != "cond-a1" {
		t.Fatalf("cached items = %#v", resp.GetItems())
	}
}

func TestListSportsLiveColdStartFailure(t *testing.T) {
	gamma := &fakeGammaClient{err: errors.New("unavailable")}
	svc := NewService(polymarketstore.NewSQLStore(nil), WithGammaClient(gamma), WithSportsWSClient(&fakeSportsWSClient{}))
	svc.started = true
	_, err := svc.ListPolymarketSportsLiveMarkets(context.Background(), &apiclient.ListPolymarketSportsLiveMarketsRequest{Limit: 1})
	if status.Code(err) != codes.Unavailable {
		t.Fatalf("ListPolymarketSportsLiveMarkets error = %v, want Unavailable", err)
	}
}

func TestListSportsLiveLimitValidation(t *testing.T) {
	svc := NewService(polymarketstore.NewSQLStore(nil), WithGammaClient(&fakeGammaClient{}), WithSportsWSClient(&fakeSportsWSClient{}))
	svc.started = true
	_, err := svc.ListPolymarketSportsLiveMarkets(context.Background(), &apiclient.ListPolymarketSportsLiveMarketsRequest{Limit: 5001})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("ListPolymarketSportsLiveMarkets error = %v, want InvalidArgument", err)
	}
}

func strPtr(value string) *string     { return &value }
func floatPtr(value float64) *float64 { return &value }
func boolPtr(value bool) *bool        { return &value }

func hasOnlyEsportsExcludeTag(values []int64) bool {
	return len(values) == 1 && values[0] == utilpolymarket.PolymarketEsportsTagID
}

var _ = time.Second
