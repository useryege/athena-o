package polymarket

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	notificationapiclient "github.com/useryege/athena/internal/notification/apiclient"
	polymarketstore "github.com/useryege/athena/internal/polymarket/store"
	"github.com/useryege/athena/pkg/apis/application/v1alpha1"
	utilio "github.com/useryege/athena/util/io"
	utilpolymarket "github.com/useryege/athena/util/polymarket"
	"google.golang.org/grpc"
)

type fakeMoverNotificationClientset struct {
	client *fakeMoverNotificationClient
	err    error
	opens  int
}

func (f *fakeMoverNotificationClientset) NewNotificationServiceClient() (utilio.Closer, notificationapiclient.NotificationServiceClient, error) {
	f.opens++
	if f.err != nil {
		return nil, nil, f.err
	}
	if f.client == nil {
		f.client = &fakeMoverNotificationClient{}
	}
	return utilio.NopCloser, f.client, nil
}

type fakeMoverNotificationClient struct {
	err   error
	sends []*notificationapiclient.SendNotificationRequest
}

func (f *fakeMoverNotificationClient) GetNotificationStatus(context.Context, *notificationapiclient.GetNotificationStatusRequest, ...grpc.CallOption) (*v1alpha1.NotificationStatus, error) {
	return &v1alpha1.NotificationStatus{}, nil
}

func (f *fakeMoverNotificationClient) SendNotification(_ context.Context, req *notificationapiclient.SendNotificationRequest, _ ...grpc.CallOption) (*notificationapiclient.SendNotificationResponse, error) {
	f.sends = append(f.sends, req)
	if f.err != nil {
		return nil, f.err
	}
	return &notificationapiclient.SendNotificationResponse{
		NotificationId: 1,
		Status:         notificationapiclient.NotificationDeliveryStatus_NOTIFICATION_DELIVERY_STATUS_SENT,
	}, nil
}

func (f *fakeMoverNotificationClient) ListNotificationDeliveries(context.Context, *notificationapiclient.ListNotificationDeliveriesRequest, ...grpc.CallOption) (*notificationapiclient.ListNotificationDeliveriesResponse, error) {
	return &notificationapiclient.ListNotificationDeliveriesResponse{}, nil
}

func (f *fakeMoverNotificationClient) GetNotificationDelivery(context.Context, *notificationapiclient.GetNotificationDeliveryRequest, ...grpc.CallOption) (*notificationapiclient.GetNotificationDeliveryResponse, error) {
	return &notificationapiclient.GetNotificationDeliveryResponse{}, nil
}

func TestMoverAlertsDoNotSendBelowThreshold(t *testing.T) {
	now := time.Date(2026, 6, 3, 12, 0, 0, 0, time.UTC)
	svc, client := newMoverAlertTestService(now)
	item := moverAlertHotMarket(t, "cond-quiet", 12000, 5000, 0.62)
	seedMoverAlertMarket(svc, item, now, 0.60)

	collectAndSendMoverAlerts(svc, now)

	if len(client.sends) != 0 {
		t.Fatalf("sends = %#v, want none", client.sends)
	}
}

func TestMoverAlertsSendWarningNotification(t *testing.T) {
	now := time.Date(2026, 6, 3, 12, 0, 0, 0, time.UTC)
	svc, client := newMoverAlertTestService(now)
	item := moverAlertHotMarket(t, "cond-warning", 12000, 5000, 0.66)
	seedMoverAlertMarket(svc, item, now, 0.60)

	collectAndSendMoverAlerts(svc, now)

	if len(client.sends) != 1 {
		t.Fatalf("sends = %d, want 1", len(client.sends))
	}
	req := client.sends[0]
	if req.GetSource() != moverAlertSource || req.GetTopic() != "poly" {
		t.Fatalf("source/topic = %q/%v, want polymarket mover poly", req.GetSource(), req.GetTopic())
	}
	if req.GetSeverity() != notificationapiclient.NotificationSeverity_NOTIFICATION_SEVERITY_WARNING {
		t.Fatalf("severity = %v, want warning", req.GetSeverity())
	}
	if !strings.Contains(req.GetTitle(), "UP Yes 66.0%") || !strings.Contains(req.GetTitle(), "Question cond-warning") {
		t.Fatalf("title = %q, want mover summary", req.GetTitle())
	}
	for _, want := range []string{"Score: 6.00", "1m +6.00pp", "24h volume: 12000.00", "Liquidity: 5000.00", "Condition ID: cond-warning"} {
		if !strings.Contains(req.GetBody(), want) {
			t.Fatalf("body = %q, want %q", req.GetBody(), want)
		}
	}
	if req.GetLink() != "https://polymarket.com/event/event-cond-warning" {
		t.Fatalf("link = %q, want event link", req.GetLink())
	}
}

func TestMoverAlertsSuppressRepeatDuringCooldown(t *testing.T) {
	now := time.Date(2026, 6, 3, 12, 0, 0, 0, time.UTC)
	svc, client := newMoverAlertTestService(now)
	item := moverAlertHotMarket(t, "cond-cooldown", 12000, 5000, 0.66)
	seedMoverAlertMarket(svc, item, now, 0.60)

	collectAndSendMoverAlerts(svc, now)
	collectAndSendMoverAlerts(svc, now.Add(time.Minute))

	if len(client.sends) != 1 {
		t.Fatalf("sends = %d, want only first alert during cooldown", len(client.sends))
	}
}

func TestMoverAlertsAllowCriticalUpgradeDuringCooldown(t *testing.T) {
	now := time.Date(2026, 6, 3, 12, 0, 0, 0, time.UTC)
	svc, client := newMoverAlertTestService(now)
	item := moverAlertHotMarket(t, "cond-upgrade", 12000, 5000, 0.66)
	seedMoverAlertMarket(svc, item, now, 0.60)

	collectAndSendMoverAlerts(svc, now)
	svc.hotMarketItems[0].Tokens[0].Price = 0.73
	collectAndSendMoverAlerts(svc, now.Add(time.Minute))

	if len(client.sends) != 2 {
		t.Fatalf("sends = %d, want warning and critical upgrade", len(client.sends))
	}
	if client.sends[1].GetSeverity() != notificationapiclient.NotificationSeverity_NOTIFICATION_SEVERITY_CRITICAL {
		t.Fatalf("second severity = %v, want critical", client.sends[1].GetSeverity())
	}
}

func TestMoverAlertsFilterWarmupStaleAndLowVolume(t *testing.T) {
	now := time.Date(2026, 6, 3, 12, 0, 0, 0, time.UTC)
	svc, client := newMoverAlertTestService(now)
	lowVolume := moverAlertHotMarket(t, "cond-low-volume", 9999, 5000, 0.80)
	warmup := moverAlertHotMarket(t, "cond-warmup", 12000, 5000, 0.80)
	stale := moverAlertHotMarket(t, "cond-stale", 12000, 5000, 0.80)
	svc.hotMarketItems = []*v1alpha1.PolymarketHotMarketItem{lowVolume, warmup, stale}
	seedMoverAlertToken(svc, lowVolume, now, 0.60)
	seedMoverAlertToken(svc, warmup, now, 0)
	seedMoverAlertToken(svc, stale, now.Add(-121*time.Second), 0.60)

	collectAndSendMoverAlerts(svc, now)

	if len(client.sends) != 0 {
		t.Fatalf("sends = %#v, want none", client.sends)
	}
}

func TestMoverAlertsLimitAndSortPerRefresh(t *testing.T) {
	now := time.Date(2026, 6, 3, 12, 0, 0, 0, time.UTC)
	svc, _ := newMoverAlertTestService(now)
	for _, market := range []*v1alpha1.PolymarketHotMarketItem{
		moverAlertHotMarket(t, "warning-low", 12000, 5000, 0.66),
		moverAlertHotMarket(t, "critical-high", 12000, 5000, 0.74),
		moverAlertHotMarket(t, "warning-high", 12000, 5000, 0.69),
		moverAlertHotMarket(t, "critical-low", 12000, 5000, 0.72),
		moverAlertHotMarket(t, "critical-mid", 12000, 5000, 0.73),
	} {
		svc.hotMarketItems = append(svc.hotMarketItems, market)
		seedMoverAlertToken(svc, market, now, 0.60)
	}

	svc.cacheMu.Lock()
	alerts := svc.collectMoverAlertsLocked(now.Unix())
	svc.cacheMu.Unlock()

	got := make([]string, 0, len(alerts))
	for _, alert := range alerts {
		got = append(got, alert.condition)
	}
	want := []string{"critical-high", "critical-mid", "critical-low"}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("alert order = %v, want %v", got, want)
	}
}

func TestMoverAlertsSendFailureDoesNotFailHotMarketRefresh(t *testing.T) {
	now := time.Date(2026, 6, 3, 12, 0, 0, 0, time.UTC)
	market := validHotMarket("cond-send-failure", 12000, 5000)
	market.OutcomePrices = strPtr(`["0.66","0.34"]`)
	gamma := &fakeGammaClient{marketResponses: []*utilpolymarket.MarketKeysetResponse{{
		Markets: []utilpolymarket.Market{market},
	}}}
	client := &fakeMoverNotificationClient{err: errors.New("telegram unavailable")}
	svc := NewService(
		polymarketstore.NewSQLStore(nil),
		WithGammaClient(gamma),
		WithSportsWSClient(&fakeSportsWSClient{}),
		WithNotificationClientset(&fakeMoverNotificationClientset{client: client}),
		WithMoverAlertsConfig(MoverAlertsConfig{Enabled: true}),
	)
	svc.started = true
	svc.nowFn = func() time.Time { return now }
	svc.realtimeStates["token-cond-send-failure-yes"] = trustedRealtimeTestState("token-cond-send-failure-yes", "Yes", 0.60, now)
	svc.realtimeSamples["token-cond-send-failure-yes"] = []realtimeSample{{at: now.Add(-time.Minute).Unix(), price: 0.60}}

	if err := svc.refreshHotMarkets(context.Background()); err != nil {
		t.Fatalf("refreshHotMarkets: %v", err)
	}
	if len(client.sends) != 1 {
		t.Fatalf("sends = %d, want attempted send despite provider failure", len(client.sends))
	}
}

func newMoverAlertTestService(now time.Time) (*Service, *fakeMoverNotificationClient) {
	client := &fakeMoverNotificationClient{}
	svc := NewService(
		polymarketstore.NewSQLStore(nil),
		WithGammaClient(&fakeGammaClient{}),
		WithSportsWSClient(&fakeSportsWSClient{}),
		WithNotificationClientset(&fakeMoverNotificationClientset{client: client}),
		WithMoverAlertsConfig(MoverAlertsConfig{Enabled: true}),
	)
	svc.started = true
	svc.nowFn = func() time.Time { return now }
	svc.hotMarketFetched = now.Unix()
	svc.realtimeConnected = true
	svc.realtimeFetched = now.Unix()
	svc.realtimeLastEventAt = now.Unix()
	return svc, client
}

func collectAndSendMoverAlerts(svc *Service, now time.Time) {
	svc.cacheMu.Lock()
	alerts := svc.collectMoverAlertsLocked(now.Unix())
	svc.cacheMu.Unlock()
	svc.sendMoverAlerts(context.Background(), alerts)
}

func moverAlertHotMarket(t *testing.T, conditionID string, volume24hr, liquidity, price float64) *v1alpha1.PolymarketHotMarketItem {
	t.Helper()
	item, ok := mapHotMarket(validHotMarket(conditionID, volume24hr, liquidity))
	if !ok {
		t.Fatalf("valid hot market %s rejected", conditionID)
	}
	item.EventSlug = "event-" + conditionID
	item.Tokens[0].Price = price
	item.Tokens[1].Price = 1 - price
	return item
}

func seedMoverAlertMarket(svc *Service, item *v1alpha1.PolymarketHotMarketItem, now time.Time, samplePrice float64) {
	svc.hotMarketItems = []*v1alpha1.PolymarketHotMarketItem{item}
	seedMoverAlertToken(svc, item, now, samplePrice)
}

func seedMoverAlertToken(svc *Service, item *v1alpha1.PolymarketHotMarketItem, lastEventAt time.Time, samplePrice float64) {
	token := item.Tokens[0]
	svc.realtimeStates[token.TokenID] = trustedRealtimeTestState(token.TokenID, token.Outcome, token.Price, lastEventAt)
	if samplePrice > 0 {
		svc.realtimeSamples[token.TokenID] = []realtimeSample{{at: lastEventAt.Add(-time.Minute).Unix(), price: samplePrice}}
	}
}

var _ notificationapiclient.NotificationServiceClient = (*fakeMoverNotificationClient)(nil)
var _ notificationapiclient.Clientset = (*fakeMoverNotificationClientset)(nil)
