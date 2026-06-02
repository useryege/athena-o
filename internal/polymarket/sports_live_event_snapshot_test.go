package polymarket

import (
	"context"
	"encoding/json"
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
	spreadType := "spread"
	groupTitle := "Moneyline"
	spreadTitle := "Spread -1.5"
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
					Teams: []json.RawMessage{
						rawJSON(`{"name":"Team A","logo":"https://flags.example/a.png","ordering":"home"}`),
						rawJSON(`{"name":"Team B","logo":"https://flags.example/b.png","ordering":"away"}`),
					},
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
						{
							ConditionID:      strPtr("cond-spread"),
							Slug:             strPtr("market-spread"),
							Question:         strPtr("Spread"),
							SportsMarketType: &spreadType,
							GroupItemTitle:   &spreadTitle,
							VolumeNum:        floatPtr(9999),
						},
					},
				},
				{
					Slug:  strPtr("spread-only-event"),
					Live:  boolPtr(true),
					Ended: boolPtr(false),
					Markets: []utilpolymarket.Market{
						{
							ConditionID:      strPtr("cond-spread-only"),
							Slug:             strPtr("market-spread-only"),
							Question:         strPtr("Spread only"),
							SportsMarketType: &spreadType,
							GroupItemTitle:   &spreadTitle,
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
	if event.Markets[0].Type != sportsLiveMoneylineMarketType {
		t.Fatalf("market group type = %q, want %q", event.Markets[0].Type, sportsLiveMoneylineMarketType)
	}
	if len(event.Markets[0].Markets[0].Outcomes) != 2 || len(event.Markets[0].Markets[0].OutcomePrices) != 2 {
		t.Fatalf("market json arrays not parsed: %#v", event.Markets[0].Markets[0])
	}
	if len(event.Markets[0].Markets[1].Outcomes) != 0 || len(event.Markets[0].Markets[1].OutcomePrices) != 0 {
		t.Fatalf("invalid market json should fallback to empty arrays: %#v", event.Markets[0].Markets[1])
	}
	if len(event.Teams) != 2 || event.Teams[0].Name != "Team A" || event.Teams[0].Logo != "https://flags.example/a.png" || event.Teams[0].Ordering != "home" {
		t.Fatalf("teams mapping mismatch: %#v", event.Teams)
	}
}

func TestSportsLiveEventSnapshotSortsByVolumeThenLastUpdateThenSlug(t *testing.T) {
	now := time.Date(2026, 6, 2, 12, 0, 0, 0, time.UTC)
	marketType := "moneyline"
	spreadType := "spread"
	groupTitle := "Moneyline"

	makeLiveEvent := func(slug string, updatedAt time.Time, volume *float64) utilpolymarket.Event {
		return utilpolymarket.Event{
			Slug:      strPtr(slug),
			Title:     strPtr("Event " + slug),
			Live:      boolPtr(true),
			Ended:     boolPtr(false),
			UpdatedAt: timePtr(updatedAt),
			Markets: []utilpolymarket.Market{
				{
					ConditionID:      strPtr("cond-" + slug),
					Slug:             strPtr("market-" + slug),
					Question:         strPtr("Who wins?"),
					SportsMarketType: &marketType,
					GroupItemTitle:   &groupTitle,
					VolumeNum:        volume,
				},
				{
					ConditionID:      strPtr("spread-cond-" + slug),
					Slug:             strPtr("spread-market-" + slug),
					Question:         strPtr("Spread"),
					SportsMarketType: &spreadType,
					VolumeNum:        floatPtr(99999),
				},
			},
		}
	}

	vol40 := 40.0
	vol10 := 10.0
	updatedLatest := now
	updatedMid := now.Add(-1 * time.Minute)
	updatedOld := now.Add(-2 * time.Minute)

	gamma := &fakeGammaClient{responses: []*utilpolymarket.EventKeysetResponse{
		{
			Events: []utilpolymarket.Event{
				makeLiveEvent("event-b", updatedOld, &vol40),
				makeLiveEvent("event-c", updatedMid, &vol40),
				makeLiveEvent("event-a", updatedMid, &vol40),
				makeLiveEvent("event-d", updatedLatest, &vol10),
				makeLiveEvent("event-e", updatedLatest, nil),
			},
		},
		{Events: []utilpolymarket.Event{}},
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
	got := make([]string, 0, len(resp.GetEvents()))
	for i := range resp.GetEvents() {
		got = append(got, resp.GetEvents()[i].EventSlug)
	}
	want := []string{"event-a", "event-c", "event-b", "event-d", "event-e"}
	if len(got) != len(want) {
		t.Fatalf("event count = %d, want %d (got=%v)", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("order mismatch at %d: got=%v want=%v", i, got, want)
		}
	}

	topTwo := svc.currentSportsLiveEventResponse(2)
	if topTwo == nil {
		t.Fatal("topTwo response is nil")
	}
	if len(topTwo.GetEvents()) != 2 || topTwo.GetEvents()[0].EventSlug != "event-a" || topTwo.GetEvents()[1].EventSlug != "event-c" {
		t.Fatalf("top two = %#v, want [event-a event-c]", topTwo.GetEvents())
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

func rawJSON(value string) json.RawMessage {
	return json.RawMessage(value)
}
