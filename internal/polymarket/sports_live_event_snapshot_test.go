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

func TestSportsLiveEventSnapshotRefreshAndMapping(t *testing.T) {
	now := time.Date(2026, 6, 2, 9, 0, 0, 0, time.UTC)
	liveSlug := "live-event-1"
	soonSlug := "soon-event-1"
	marketType := "moneyline"
	groupTitle := "Moneyline"
	outcomesJSON := "[\"Team A\",\"Team B\"]"
	pricesJSON := "[0.62,0.38]"
	invalidJSON := "oops"

	gamma := &fakeGammaClient{responses: []*utilpolymarket.EventKeysetResponse{
		{
			Events: []utilpolymarket.Event{
				{
					Slug:       &liveSlug,
					Title:      strPtr("Team A vs Team B"),
					Live:       boolPtr(true),
					Ended:      boolPtr(false),
					Score:      strPtr("1-0"),
					Period:     strPtr("Q2"),
					Elapsed:    strPtr("11:12"),
					GameStatus: strPtr("In Progress"),
					StartTime:  timePtr(now.Add(-20 * time.Minute)),
					UpdatedAt:  timePtr(now),
					Markets: []utilpolymarket.Market{
						{
							ConditionID:      strPtr("cond-1"),
							Slug:             strPtr("market-1"),
							Question:         strPtr("Who wins?"),
							SportsMarketType: &marketType,
							GroupItemTitle:   &groupTitle,
							Outcomes:         &outcomesJSON,
							OutcomePrices:    &pricesJSON,
							BestBid:          floatPtr(0.61),
							BestAsk:          floatPtr(0.63),
							LastTradePrice:   floatPtr(0.62),
							VolumeNum:        floatPtr(1200),
							LiquidityNum:     floatPtr(15000),
							Image:            strPtr("https://img.example/1.png"),
						},
						{
							ConditionID:      strPtr("cond-2"),
							Slug:             strPtr("market-2"),
							Question:         strPtr("Alt line"),
							SportsMarketType: &marketType,
							GroupItemTitle:   &groupTitle,
							Outcomes:         &invalidJSON,
							OutcomePrices:    &invalidJSON,
						},
					},
				},
			},
		},
		{
			Events: []utilpolymarket.Event{
				{
					Slug:      &soonSlug,
					StartTime: timePtr(now.Add(2 * time.Hour)),
					Markets: []utilpolymarket.Market{
						{Slug: strPtr("soon-market-1"), SportsMarketType: &marketType},
					},
				},
			},
		},
	}}

	svc := NewService(polymarketstore.NewSQLStore(nil), WithGammaClient(gamma), WithSportsWSClient(&fakeSportsWSClient{}))
	svc.started = true
	svc.nowFn = func() time.Time { return now }

	if err := svc.refreshSportsLiveEventSnapshot(context.Background()); err != nil {
		t.Fatalf("refreshSportsLiveEventSnapshot: %v", err)
	}

	resp := svc.currentSportsLiveEventResponse(10)
	if resp == nil {
		t.Fatal("response is nil")
	}
	if len(resp.GetEvents()) != 1 {
		t.Fatalf("events len = %d, want 1", len(resp.GetEvents()))
	}

	event := resp.GetEvents()[0]
	if event.EventSlug != liveSlug || event.Score != "1-0" || event.Period != "Q2" || event.Elapsed != "11:12" {
		t.Fatalf("event mapping mismatch: %#v", event)
	}
	if event.Image != "https://img.example/1.png" {
		t.Fatalf("image = %q, want mapped market image", event.Image)
	}
	if event.LastUpdate != now.Format(time.RFC3339) {
		t.Fatalf("last_update = %q, want %q", event.LastUpdate, now.Format(time.RFC3339))
	}
	if len(event.Markets) != 1 || len(event.Markets[0].Markets) != 2 {
		t.Fatalf("market groups mapping mismatch: %#v", event.Markets)
	}
	if len(event.Markets[0].Markets[0].Outcomes) != 2 || len(event.Markets[0].Markets[0].OutcomePrices) != 2 {
		t.Fatalf("market json arrays not parsed: %#v", event.Markets[0].Markets[0])
	}
	if len(event.Markets[0].Markets[1].Outcomes) != 0 || len(event.Markets[0].Markets[1].OutcomePrices) != 0 {
		t.Fatalf("invalid market json should fallback to empty arrays: %#v", event.Markets[0].Markets[1])
	}
}

func TestSportsLiveEventSnapshotStaleFallbackOnRefreshError(t *testing.T) {
	liveSlug := "live-event-1"
	marketType := "moneyline"
	gamma := &fakeGammaClient{responses: []*utilpolymarket.EventKeysetResponse{
		{
			Events: []utilpolymarket.Event{
				{
					Slug: &liveSlug,
					Live: boolPtr(true),
					Markets: []utilpolymarket.Market{
						{Slug: strPtr("market-1"), SportsMarketType: &marketType},
					},
				},
			},
		},
		{Events: []utilpolymarket.Event{}},
	}}
	svc := NewService(polymarketstore.NewSQLStore(nil), WithGammaClient(gamma), WithSportsWSClient(&fakeSportsWSClient{}))
	svc.started = true

	if err := svc.refreshSportsLiveEventSnapshot(context.Background()); err != nil {
		t.Fatalf("refreshSportsLiveEventSnapshot: %v", err)
	}

	gamma.err = errors.New("boom")
	if err := svc.refreshSportsLiveEventSnapshot(context.Background()); err == nil {
		t.Fatal("expected refresh error")
	}

	resp := svc.currentSportsLiveEventResponse(10)
	if resp == nil {
		t.Fatal("response is nil")
	}
	if !resp.GetStale() {
		t.Fatal("stale = false, want true")
	}
	if len(resp.GetEvents()) != 1 {
		t.Fatalf("events len = %d, want cached 1", len(resp.GetEvents()))
	}
}

func TestGetSportsLiveSnapshotColdStartFailure(t *testing.T) {
	gamma := &fakeGammaClient{err: errors.New("unavailable")}
	svc := NewService(polymarketstore.NewSQLStore(nil), WithGammaClient(gamma), WithSportsWSClient(&fakeSportsWSClient{}))
	svc.started = true

	_, err := svc.GetPolymarketSportsLiveSnapshot(context.Background(), &apiclient.GetPolymarketSportsLiveSnapshotRequest{Limit: 1})
	if status.Code(err) != codes.Unavailable {
		t.Fatalf("GetPolymarketSportsLiveSnapshot error = %v, want Unavailable", err)
	}
}

func TestGetSportsLiveSnapshotLimitValidation(t *testing.T) {
	svc := NewService(polymarketstore.NewSQLStore(nil), WithGammaClient(&fakeGammaClient{}), WithSportsWSClient(&fakeSportsWSClient{}))
	svc.started = true

	_, err := svc.GetPolymarketSportsLiveSnapshot(context.Background(), &apiclient.GetPolymarketSportsLiveSnapshotRequest{Limit: 1001})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("GetPolymarketSportsLiveSnapshot error = %v, want InvalidArgument", err)
	}
}

func timePtr(value time.Time) *time.Time {
	return &value
}
