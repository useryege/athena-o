package polymarket

import (
	"context"
	"testing"
	"time"
)

type fakeSportsSnapshotGammaClient struct {
	listEventsKeysetFn func(context.Context, ListEventsKeysetOptions) (*EventKeysetResponse, error)
}

func (f *fakeSportsSnapshotGammaClient) ListEventsKeyset(ctx context.Context, options ListEventsKeysetOptions) (*EventKeysetResponse, error) {
	if f.listEventsKeysetFn == nil {
		return &EventKeysetResponse{}, nil
	}
	return f.listEventsKeysetFn(ctx, options)
}

func TestBuildSportsLiveSnapshotMergeAndDedupe(t *testing.T) {
	now := time.Date(2026, 6, 2, 0, 0, 0, 0, time.UTC)
	liveUpdated := now.Add(-time.Minute)
	liveOlder := now.Add(-5 * time.Minute)
	soonStart := now.Add(2 * time.Hour)
	soonLater := now.Add(3 * time.Hour)

	liveSlug := "live-event-1"
	sharedSlug := "shared-event"
	soonSlug := "soon-event-1"
	sportsTypeMoneyline := "moneyline"
	sportsTypeTotals := "totals"
	groupTitle := "O/U 2.5"

	calls := 0
	client := &fakeSportsSnapshotGammaClient{
		listEventsKeysetFn: func(_ context.Context, options ListEventsKeysetOptions) (*EventKeysetResponse, error) {
			calls++
			if options.Live != nil && *options.Live {
				return &EventKeysetResponse{
					Events: []Event{
						{
							Slug:      &liveSlug,
							UpdatedAt: &liveUpdated,
							Markets: []Market{
								{Slug: strPtr("m-live-1"), SportsMarketType: &sportsTypeMoneyline},
								{Slug: strPtr("m-live-2"), SportsMarketType: &sportsTypeTotals, GroupItemTitle: &groupTitle},
							},
						},
						{
							Slug:      &sharedSlug,
							UpdatedAt: &liveOlder,
							Markets: []Market{
								{Slug: strPtr("m-shared-live"), SportsMarketType: &sportsTypeMoneyline},
							},
						},
					},
				}, nil
			}

			return &EventKeysetResponse{
				Events: []Event{
					{
						Slug:      &sharedSlug,
						StartTime: &soonLater,
						Markets: []Market{
							{Slug: strPtr("m-shared-soon"), SportsMarketType: &sportsTypeMoneyline},
						},
					},
					{
						Slug:      &soonSlug,
						StartTime: &soonStart,
						Markets: []Market{
							{Slug: strPtr("m-soon-1"), SportsMarketType: &sportsTypeTotals, GroupItemTitle: &groupTitle},
						},
					},
				},
			}, nil
		},
	}

	got, err := buildSportsLiveSnapshotWithNow(context.Background(), client, (SportsLiveSnapshotOptions{}).withDefaults(), now)
	if err != nil {
		t.Fatalf("buildSportsLiveSnapshotWithNow: %v", err)
	}
	if calls != 2 {
		t.Fatalf("calls = %d, want 2", calls)
	}
	if !got.FetchedAt.Equal(now) {
		t.Fatalf("fetchedAt = %s, want %s", got.FetchedAt, now)
	}
	if len(got.Live) != 2 {
		t.Fatalf("len(live) = %d, want 2", len(got.Live))
	}
	if got.Live[0].Slug != liveSlug {
		t.Fatalf("live[0].slug = %q, want %q", got.Live[0].Slug, liveSlug)
	}
	if len(got.Live[0].Markets) != 2 {
		t.Fatalf("live[0] group count = %d, want 2", len(got.Live[0].Markets))
	}
	if len(got.Soon) != 1 {
		t.Fatalf("len(soon) = %d, want 1", len(got.Soon))
	}
	if got.Soon[0].Slug != soonSlug {
		t.Fatalf("soon[0].slug = %q, want %q", got.Soon[0].Slug, soonSlug)
	}
}

func TestBuildSportsLiveSnapshotOptionsAndFiltering(t *testing.T) {
	now := time.Date(2026, 6, 2, 12, 0, 0, 0, time.UTC)
	liveSlug := "event-live"
	soonSlug := "event-soon"
	moneyline := "moneyline"
	totals := "totals"
	liveStart := now.Add(30 * time.Minute)
	soonStart := now.Add(90 * time.Minute)

	callOpts := make([]ListEventsKeysetOptions, 0, 2)
	client := &fakeSportsSnapshotGammaClient{
		listEventsKeysetFn: func(_ context.Context, options ListEventsKeysetOptions) (*EventKeysetResponse, error) {
			callOpts = append(callOpts, options)
			if options.Live != nil && *options.Live {
				return &EventKeysetResponse{
					Events: []Event{
						{
							Slug:      &liveSlug,
							StartTime: &liveStart,
							Markets: []Market{
								{Slug: strPtr("live-moneyline"), SportsMarketType: &moneyline},
								{Slug: strPtr("live-totals"), SportsMarketType: &totals},
							},
						},
					},
				}, nil
			}
			return &EventKeysetResponse{
				Events: []Event{
					{
						Slug:      &soonSlug,
						StartTime: &soonStart,
						Markets: []Market{
							{Slug: strPtr("soon-totals"), SportsMarketType: &totals},
						},
					},
				},
			}, nil
		},
	}

	got, err := buildSportsLiveSnapshotWithNow(context.Background(), client, SportsLiveSnapshotOptions{
		LiveLimit:   1,
		SoonLimit:   1,
		SoonWindow:  2 * time.Hour,
		MarketTypes: []string{"moneyline"},
	}, now)
	if err != nil {
		t.Fatalf("buildSportsLiveSnapshotWithNow: %v", err)
	}
	if len(callOpts) != 2 {
		t.Fatalf("call count = %d, want 2", len(callOpts))
	}
	if callOpts[0].Limit == nil || *callOpts[0].Limit != 1 || callOpts[0].Live == nil || !*callOpts[0].Live || callOpts[0].Closed == nil || *callOpts[0].Closed || callOpts[0].TagSlug != "sports" || !hasOnlyEsportsExcludeTag(callOpts[0].ExcludeTagID) {
		t.Fatalf("live query options = %#v", callOpts[0])
	}
	if callOpts[1].Limit == nil || *callOpts[1].Limit != 1 || callOpts[1].Closed == nil || *callOpts[1].Closed || callOpts[1].TagSlug != "sports" || !hasOnlyEsportsExcludeTag(callOpts[1].ExcludeTagID) {
		t.Fatalf("soon query options = %#v", callOpts[1])
	}
	if callOpts[1].StartTimeMin == nil || !callOpts[1].StartTimeMin.Equal(now) {
		t.Fatalf("start_time_min = %v, want %v", callOpts[1].StartTimeMin, now)
	}
	wantSoonMax := now.Add(2 * time.Hour)
	if callOpts[1].StartTimeMax == nil || !callOpts[1].StartTimeMax.Equal(wantSoonMax) {
		t.Fatalf("start_time_max = %v, want %v", callOpts[1].StartTimeMax, wantSoonMax)
	}

	if len(got.Live) != 1 || got.Live[0].Slug != liveSlug {
		t.Fatalf("live = %#v", got.Live)
	}
	if len(got.Live[0].Markets) != 1 || got.Live[0].Markets[0].Type != "moneyline" {
		t.Fatalf("live market groups = %#v, want only moneyline", got.Live[0].Markets)
	}
	if len(got.Soon) != 0 {
		t.Fatalf("soon = %#v, want empty after market type filter", got.Soon)
	}
}

func TestBuildSportsLiveSnapshotHandlesSparseData(t *testing.T) {
	badSlug := "   "
	goodSlug := "good-soon-event"
	soonStart := time.Date(2026, 6, 2, 16, 0, 0, 0, time.UTC)
	moneyline := "moneyline"

	client := &fakeSportsSnapshotGammaClient{
		listEventsKeysetFn: func(_ context.Context, options ListEventsKeysetOptions) (*EventKeysetResponse, error) {
			if options.Live != nil && *options.Live {
				return &EventKeysetResponse{
					Events: []Event{
						{Slug: &badSlug},
					},
				}, nil
			}
			return &EventKeysetResponse{
				Events: []Event{
					{Slug: &badSlug},
					{
						Slug:      &goodSlug,
						StartTime: &soonStart,
						Markets: []Market{
							{Slug: strPtr("good-market"), SportsMarketType: &moneyline},
						},
					},
				},
			}, nil
		},
	}

	got, err := BuildSportsLiveSnapshot(context.Background(), client, SportsLiveSnapshotOptions{
		LiveLimit: 10,
		SoonLimit: 10,
	})
	if err != nil {
		t.Fatalf("BuildSportsLiveSnapshot: %v", err)
	}
	if len(got.Live) != 0 {
		t.Fatalf("live = %#v, want empty", got.Live)
	}
	if len(got.Soon) != 1 || got.Soon[0].Slug != goodSlug {
		t.Fatalf("soon = %#v, want only good slug", got.Soon)
	}
}

func strPtr(value string) *string {
	return &value
}

func hasOnlyEsportsExcludeTag(values []int64) bool {
	return len(values) == 1 && values[0] == PolymarketEsportsTagID
}
