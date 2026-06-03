package polymarket

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/useryege/athena/internal/notification"
	polymarketstore "github.com/useryege/athena/internal/polymarket/store"
	utilpolymarket "github.com/useryege/athena/util/polymarket"
)

func TestSportsKickoffDoesNotAlertFirstSeenLive(t *testing.T) {
	now := time.Date(2026, 6, 3, 12, 0, 0, 0, time.UTC)
	event := sportsKickoffEvent("event-live-first", true, false, now, now.Add(-10*time.Minute))
	gamma := &fakeGammaClient{responses: []*utilpolymarket.EventKeysetResponse{
		{Events: []utilpolymarket.Event{event}},
		{Events: []utilpolymarket.Event{}},
	}}
	client := &fakeMoverNotificationClient{}
	svc := newSportsKickoffTestService(now, gamma, client)

	if err := svc.refreshSportsLiveEventSnapshot(context.Background()); err != nil {
		t.Fatalf("refreshSportsLiveEventSnapshot: %v", err)
	}

	if len(client.sends) != 0 {
		t.Fatalf("sends = %#v, want none for first-seen live event", client.sends)
	}
	state := svc.sportsKickoffStates["event-live-first"]
	if !state.WasLive || !state.Alerted {
		t.Fatalf("state = %#v, want first-seen live suppressed", state)
	}
}

func TestSportsKickoffAlertsSoonToLiveOncePerEvent(t *testing.T) {
	now := time.Date(2026, 6, 3, 12, 0, 0, 0, time.UTC)
	soon := sportsKickoffEvent("event-kickoff", false, false, now, now.Add(30*time.Minute))
	live := sportsKickoffEvent("event-kickoff", true, false, now.Add(time.Minute), now.Add(-time.Minute))
	live.Markets = append(live.Markets, utilpolymarket.Market{
		ConditionID:      strPtr("cond-event-kickoff-2"),
		Slug:             strPtr("market-event-kickoff-2"),
		Question:         strPtr("Alternate moneyline"),
		SportsMarketType: strPtr(sportsLiveMoneylineMarketType),
		GroupItemTitle:   strPtr("Moneyline"),
		Outcomes:         strPtr(`["Team C","Team D"]`),
		OutcomePrices:    strPtr(`["0.48","0.52"]`),
	})
	gamma := &fakeGammaClient{responses: []*utilpolymarket.EventKeysetResponse{
		{Events: []utilpolymarket.Event{}},
		{Events: []utilpolymarket.Event{soon}},
		{Events: []utilpolymarket.Event{live}},
		{Events: []utilpolymarket.Event{}},
		{Events: []utilpolymarket.Event{live}},
		{Events: []utilpolymarket.Event{}},
	}}
	client := &fakeMoverNotificationClient{}
	svc := newSportsKickoffTestService(now, gamma, client)

	if err := svc.refreshSportsLiveEventSnapshot(context.Background()); err != nil {
		t.Fatalf("first refreshSportsLiveEventSnapshot: %v", err)
	}
	if len(client.sends) != 0 {
		t.Fatalf("first sends = %#v, want none while event is soon", client.sends)
	}

	svc.nowFn = func() time.Time { return now.Add(time.Minute) }
	if err := svc.refreshSportsLiveEventSnapshot(context.Background()); err != nil {
		t.Fatalf("second refreshSportsLiveEventSnapshot: %v", err)
	}
	if len(client.sends) != 1 {
		t.Fatalf("sends = %d, want one kickoff alert", len(client.sends))
	}
	req := client.sends[0]
	if req.GetSource() != sportsKickoffAlertSource || req.GetTopic() != notification.NotificationTopicPolyKickoff {
		t.Fatalf("source/topic = %q/%q, want sports kickoff topic", req.GetSource(), req.GetTopic())
	}
	if req.GetLink() != "https://polymarket.com/event/event-kickoff" {
		t.Fatalf("link = %q, want event link", req.GetLink())
	}
	for _, want := range []string{"Polymarket kickoff", "Team A vs Team B", "Score: 0-0", "Game status: In Progress", "Event slug: event-kickoff", "cond-event-kickoff", "cond-event-kickoff-2"} {
		if !strings.Contains(req.GetTitle()+req.GetBody(), want) {
			t.Fatalf("notification = title %q body %q, want %q", req.GetTitle(), req.GetBody(), want)
		}
	}

	svc.nowFn = func() time.Time { return now.Add(2 * time.Minute) }
	if err := svc.refreshSportsLiveEventSnapshot(context.Background()); err != nil {
		t.Fatalf("third refreshSportsLiveEventSnapshot: %v", err)
	}
	if len(client.sends) != 1 {
		t.Fatalf("sends after repeat live = %d, want still one", len(client.sends))
	}
}

func TestSportsKickoffSkipsEndedEvent(t *testing.T) {
	now := time.Date(2026, 6, 3, 12, 0, 0, 0, time.UTC)
	soon := sportsKickoffEvent("event-ended", false, false, now, now.Add(30*time.Minute))
	ended := sportsKickoffEvent("event-ended", true, true, now.Add(time.Minute), now.Add(-time.Minute))
	gamma := &fakeGammaClient{responses: []*utilpolymarket.EventKeysetResponse{
		{Events: []utilpolymarket.Event{}},
		{Events: []utilpolymarket.Event{soon}},
		{Events: []utilpolymarket.Event{ended}},
		{Events: []utilpolymarket.Event{}},
	}}
	client := &fakeMoverNotificationClient{}
	svc := newSportsKickoffTestService(now, gamma, client)

	if err := svc.refreshSportsLiveEventSnapshot(context.Background()); err != nil {
		t.Fatalf("first refreshSportsLiveEventSnapshot: %v", err)
	}
	svc.nowFn = func() time.Time { return now.Add(time.Minute) }
	if err := svc.refreshSportsLiveEventSnapshot(context.Background()); err != nil {
		t.Fatalf("second refreshSportsLiveEventSnapshot: %v", err)
	}

	if len(client.sends) != 0 {
		t.Fatalf("sends = %#v, want none for ended event", client.sends)
	}
}

func TestSportsKickoffSendFailureDoesNotFailRefresh(t *testing.T) {
	now := time.Date(2026, 6, 3, 12, 0, 0, 0, time.UTC)
	soon := sportsKickoffEvent("event-send-failure", false, false, now, now.Add(30*time.Minute))
	live := sportsKickoffEvent("event-send-failure", true, false, now.Add(time.Minute), now.Add(-time.Minute))
	gamma := &fakeGammaClient{responses: []*utilpolymarket.EventKeysetResponse{
		{Events: []utilpolymarket.Event{}},
		{Events: []utilpolymarket.Event{soon}},
		{Events: []utilpolymarket.Event{live}},
		{Events: []utilpolymarket.Event{}},
	}}
	client := &fakeMoverNotificationClient{err: errors.New("telegram unavailable")}
	svc := newSportsKickoffTestService(now, gamma, client)

	if err := svc.refreshSportsLiveEventSnapshot(context.Background()); err != nil {
		t.Fatalf("first refreshSportsLiveEventSnapshot: %v", err)
	}
	svc.nowFn = func() time.Time { return now.Add(time.Minute) }
	if err := svc.refreshSportsLiveEventSnapshot(context.Background()); err != nil {
		t.Fatalf("second refreshSportsLiveEventSnapshot: %v", err)
	}

	if len(client.sends) != 1 {
		t.Fatalf("sends = %d, want attempted send despite provider failure", len(client.sends))
	}
}

func newSportsKickoffTestService(now time.Time, gamma *fakeGammaClient, client *fakeMoverNotificationClient) *Service {
	svc := NewService(
		polymarketstore.NewSQLStore(nil),
		WithGammaClient(gamma),
		WithSportsWSClient(&fakeSportsWSClient{}),
		WithNotificationClientset(&fakeMoverNotificationClientset{client: client}),
	)
	svc.started = true
	svc.nowFn = func() time.Time { return now }
	return svc
}

func sportsKickoffEvent(slug string, live bool, ended bool, updatedAt time.Time, startTime time.Time) utilpolymarket.Event {
	return utilpolymarket.Event{
		Slug:       strPtr(slug),
		Title:      strPtr("Team A vs Team B"),
		Live:       boolPtr(live),
		Ended:      boolPtr(ended),
		Score:      strPtr("0-0"),
		Period:     strPtr("Q1"),
		Elapsed:    strPtr("0:15"),
		GameStatus: strPtr("In Progress"),
		StartTime:  timePtr(startTime),
		UpdatedAt:  timePtr(updatedAt),
		Markets: []utilpolymarket.Market{
			{
				ConditionID:      strPtr("cond-" + slug),
				Slug:             strPtr("market-" + slug),
				Question:         strPtr("Who wins?"),
				SportsMarketType: strPtr(sportsLiveMoneylineMarketType),
				GroupItemTitle:   strPtr("Moneyline"),
				Outcomes:         strPtr(`["Team A","Team B"]`),
				OutcomePrices:    strPtr(`["0.51","0.49"]`),
				VolumeNum:        floatPtr(1000),
				LiquidityNum:     floatPtr(5000),
			},
		},
	}
}
