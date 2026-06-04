package polymarket

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/useryege/athena/internal/notification"
	notificationapiclient "github.com/useryege/athena/internal/notification/apiclient"
	polymarketstore "github.com/useryege/athena/internal/polymarket/store"
	utilpolymarket "github.com/useryege/athena/util/polymarket"
)

func TestSportsLiveMarketAlertsBaselineDoesNotNotify(t *testing.T) {
	now := time.Date(2026, 6, 3, 12, 0, 0, 0, time.UTC)
	gamma := &fakeGammaClient{responses: []*utilpolymarket.EventKeysetResponse{{
		Events: []utilpolymarket.Event{
			sportsLiveAlertEvent("event-baseline", sportsLiveAlertMarket("cond-baseline", "market-baseline", "Baseline moneyline")),
		},
	}}}
	client := &fakeMoverNotificationClient{}
	svc := newSportsKickoffTestService(now, gamma, client)

	if err := svc.refreshSportsLiveSnapshot(context.Background()); err != nil {
		t.Fatalf("refreshSportsLiveSnapshot: %v", err)
	}

	if len(client.sends) != 0 {
		t.Fatalf("sends = %#v, want none for first live sports baseline", client.sends)
	}
	if !svc.sportsLiveMarketAlertsInitialized {
		t.Fatal("sportsLiveMarketAlertsInitialized = false, want true")
	}
	if state, ok := svc.sportsLiveMarketAlertStates["cond-baseline"]; !ok || state.LastSeenAt != now.Unix() {
		t.Fatalf("baseline state = %#v ok=%v, want current condition recorded", state, ok)
	}
}

func TestSportsLiveMarketAlertsNotifyNewConditionOnce(t *testing.T) {
	now := time.Date(2026, 6, 3, 12, 0, 0, 0, time.UTC)
	gamma := &fakeGammaClient{responses: []*utilpolymarket.EventKeysetResponse{
		{Events: []utilpolymarket.Event{
			sportsLiveAlertEvent("event-baseline", sportsLiveAlertMarket("cond-baseline", "market-baseline", "Baseline moneyline")),
		}},
		{Events: []utilpolymarket.Event{
			sportsLiveAlertEvent("event-baseline", sportsLiveAlertMarket("cond-baseline", "market-baseline", "Baseline moneyline")),
			sportsLiveAlertEvent("event-new", sportsLiveAlertMarket("cond-new", "market-new", "New moneyline")),
		}},
		{Events: []utilpolymarket.Event{
			sportsLiveAlertEvent("event-baseline", sportsLiveAlertMarket("cond-baseline", "market-baseline", "Baseline moneyline")),
			sportsLiveAlertEvent("event-new", sportsLiveAlertMarket("cond-new", "market-new", "New moneyline")),
		}},
	}}
	client := &fakeMoverNotificationClient{}
	svc := newSportsKickoffTestService(now, gamma, client)

	if err := svc.refreshSportsLiveSnapshot(context.Background()); err != nil {
		t.Fatalf("baseline refreshSportsLiveSnapshot: %v", err)
	}

	lastUpdate := "2026-06-03T12:01:00Z"
	svc.applySportsWSUpdate(utilpolymarket.SportsWSUpdate{
		Slug:       "event-new",
		Live:       boolPtr(true),
		Score:      strPtr("0-0"),
		Period:     strPtr("Q1"),
		Elapsed:    strPtr("0:15"),
		LastUpdate: &lastUpdate,
	})
	svc.nowFn = func() time.Time { return now.Add(time.Minute) }
	if err := svc.refreshSportsLiveSnapshot(context.Background()); err != nil {
		t.Fatalf("new market refreshSportsLiveSnapshot: %v", err)
	}

	if len(client.sends) != 1 {
		t.Fatalf("sends = %d, want one new live market notification", len(client.sends))
	}
	req := client.sends[0]
	if req.GetSource() != sportsKickoffAlertSource || req.GetTopic() != notification.NotificationTopicPolyKickoff {
		t.Fatalf("source/topic = %q/%q, want sports kickoff topic", req.GetSource(), req.GetTopic())
	}
	if req.GetSeverity() != notificationapiclient.NotificationSeverity_NOTIFICATION_SEVERITY_INFO {
		t.Fatalf("severity = %v, want info", req.GetSeverity())
	}
	if req.GetLink() != "https://polymarket.com/event/event-new" {
		t.Fatalf("link = %q, want event link", req.GetLink())
	}
	for _, want := range []string{
		"Polymarket live market: New moneyline",
		"Event slug: event-new",
		"Market slug: market-new",
		"Condition ID: cond-new",
		"Score: 0-0",
		"Period: Q1",
		"Elapsed: 0:15",
		"Last update: 2026-06-03T12:01:00Z",
		"Volume: 222.00",
		"Liquidity: 333.00",
	} {
		if !strings.Contains(req.GetTitle()+req.GetBody(), want) {
			t.Fatalf("notification = title %q body %q, want %q", req.GetTitle(), req.GetBody(), want)
		}
	}

	svc.nowFn = func() time.Time { return now.Add(2 * time.Minute) }
	if err := svc.refreshSportsLiveSnapshot(context.Background()); err != nil {
		t.Fatalf("repeat market refreshSportsLiveSnapshot: %v", err)
	}
	if len(client.sends) != 1 {
		t.Fatalf("sends after repeat = %d, want still one", len(client.sends))
	}
}

func TestSportsLiveMarketAlertsNotifyMultipleNewConditions(t *testing.T) {
	now := time.Date(2026, 6, 3, 12, 0, 0, 0, time.UTC)
	gamma := &fakeGammaClient{responses: []*utilpolymarket.EventKeysetResponse{
		{Events: []utilpolymarket.Event{}},
		{Events: []utilpolymarket.Event{
			sportsLiveAlertEvent(
				"event-multi",
				sportsLiveAlertMarket("cond-new-1", "market-new-1", "New moneyline 1"),
				sportsLiveAlertMarket("cond-new-2", "market-new-2", "New moneyline 2"),
			),
		}},
	}}
	client := &fakeMoverNotificationClient{}
	svc := newSportsKickoffTestService(now, gamma, client)

	if err := svc.refreshSportsLiveSnapshot(context.Background()); err != nil {
		t.Fatalf("baseline refreshSportsLiveSnapshot: %v", err)
	}
	svc.nowFn = func() time.Time { return now.Add(time.Minute) }
	if err := svc.refreshSportsLiveSnapshot(context.Background()); err != nil {
		t.Fatalf("new markets refreshSportsLiveSnapshot: %v", err)
	}

	if len(client.sends) != 2 {
		t.Fatalf("sends = %d, want one notification per new condition", len(client.sends))
	}
	got := client.sends[0].GetBody() + "\n" + client.sends[1].GetBody()
	for _, want := range []string{"Condition ID: cond-new-1", "Condition ID: cond-new-2"} {
		if !strings.Contains(got, want) {
			t.Fatalf("notifications = %q, want %q", got, want)
		}
	}
}

func TestSportsLiveMarketAlertsSkipEndedAndNonLiveMarkets(t *testing.T) {
	now := time.Date(2026, 6, 3, 12, 0, 0, 0, time.UTC)
	gamma := &fakeGammaClient{responses: []*utilpolymarket.EventKeysetResponse{
		{Events: []utilpolymarket.Event{}},
		{Events: []utilpolymarket.Event{
			sportsLiveAlertEvent("event-ws-non-live", sportsLiveAlertMarket("cond-ws-non-live", "market-ws-non-live", "WS non-live")),
			sportsLiveAlertEvent("event-ended", sportsLiveAlertMarket("cond-ended", "market-ended", "Ended event")),
			sportsLiveAlertEvent("event-explicit-non-live", sportsLiveAlertMarket("cond-explicit-non-live", "market-explicit-non-live", "Explicit non-live")),
		}},
	}}
	gamma.responses[1].Events[1].Ended = boolPtr(true)
	gamma.responses[1].Events[2].Live = boolPtr(false)
	client := &fakeMoverNotificationClient{}
	svc := newSportsKickoffTestService(now, gamma, client)

	if err := svc.refreshSportsLiveSnapshot(context.Background()); err != nil {
		t.Fatalf("baseline refreshSportsLiveSnapshot: %v", err)
	}
	svc.applySportsWSUpdate(utilpolymarket.SportsWSUpdate{Slug: "event-ws-non-live", Live: boolPtr(false)})
	svc.nowFn = func() time.Time { return now.Add(time.Minute) }
	if err := svc.refreshSportsLiveSnapshot(context.Background()); err != nil {
		t.Fatalf("filtered refreshSportsLiveSnapshot: %v", err)
	}

	if len(client.sends) != 0 {
		t.Fatalf("sends = %#v, want none for non-live/ended markets", client.sends)
	}
	for _, conditionID := range []string{"cond-ws-non-live", "cond-ended", "cond-explicit-non-live"} {
		if _, ok := svc.sportsLiveMarketAlertStates[conditionID]; ok {
			t.Fatalf("condition %q recorded despite not entering live sports list", conditionID)
		}
	}
}

func TestSportsLiveMarketAlertSendFailureDoesNotFailRefresh(t *testing.T) {
	now := time.Date(2026, 6, 3, 12, 0, 0, 0, time.UTC)
	gamma := &fakeGammaClient{responses: []*utilpolymarket.EventKeysetResponse{
		{Events: []utilpolymarket.Event{}},
		{Events: []utilpolymarket.Event{
			sportsLiveAlertEvent("event-send-failure", sportsLiveAlertMarket("cond-send-failure", "market-send-failure", "Send failure")),
		}},
	}}
	client := &fakeMoverNotificationClient{err: errors.New("telegram unavailable")}
	svc := newSportsKickoffTestService(now, gamma, client)

	if err := svc.refreshSportsLiveSnapshot(context.Background()); err != nil {
		t.Fatalf("baseline refreshSportsLiveSnapshot: %v", err)
	}
	svc.nowFn = func() time.Time { return now.Add(time.Minute) }
	if err := svc.refreshSportsLiveSnapshot(context.Background()); err != nil {
		t.Fatalf("new market refreshSportsLiveSnapshot: %v", err)
	}

	if len(client.sends) != 1 {
		t.Fatalf("sends = %d, want attempted send despite provider failure", len(client.sends))
	}
}

func TestSportsLiveEventSnapshotDoesNotSendMarketAlerts(t *testing.T) {
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
		t.Fatalf("sends = %#v, want no notifications from event snapshot refresh", client.sends)
	}
	if len(svc.sportsLiveMarketAlertStates) != 0 {
		t.Fatalf("sports live market alert states = %#v, want unchanged by event snapshot", svc.sportsLiveMarketAlertStates)
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

func sportsLiveAlertEvent(slug string, markets ...utilpolymarket.Market) utilpolymarket.Event {
	return utilpolymarket.Event{
		Slug:    strPtr(slug),
		Title:   strPtr("Team A vs Team B"),
		Live:    boolPtr(true),
		Ended:   boolPtr(false),
		Markets: markets,
	}
}

func sportsLiveAlertMarket(conditionID, slug, question string) utilpolymarket.Market {
	return utilpolymarket.Market{
		ConditionID:  strPtr(conditionID),
		Slug:         strPtr(slug),
		Question:     strPtr(question),
		VolumeNum:    floatPtr(222),
		LiquidityNum: floatPtr(333),
	}
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
