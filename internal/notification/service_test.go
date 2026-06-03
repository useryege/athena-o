package notification

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/useryege/athena/internal/notification/apiclient"
	notificationstore "github.com/useryege/athena/internal/notification/store"
	notificationsqlc "github.com/useryege/athena/internal/notification/store/sqlc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type fakeSender struct {
	messageID string
	err       error
	topic     string
	text      string
	calls     int
}

func (f *fakeSender) Send(_ context.Context, request SendRequest) (string, error) {
	f.calls++
	f.topic = request.Topic
	f.text = request.Text
	if f.err != nil {
		return "", f.err
	}
	if f.messageID != "" {
		return f.messageID, nil
	}
	return "1", nil
}

type fakeProfileSyncer struct {
	calls int
	err   error
}

func (f *fakeProfileSyncer) SyncProfile(context.Context) error {
	f.calls++
	return f.err
}

type fakeNotificationQuerier struct {
	createDeliveryResult notificationsqlc.CreateDeliveryRow
	createDeliveryParams notificationsqlc.CreateDeliveryParams
	markSentParams       notificationsqlc.MarkDeliverySentParams
	markFailedParams     notificationsqlc.MarkDeliveryFailedParams
	scheduleRetryParams  notificationsqlc.ScheduleDeliveryRetryParams
}

func (f *fakeNotificationQuerier) ClaimPendingDeliveries(context.Context, notificationsqlc.ClaimPendingDeliveriesParams) ([]notificationsqlc.ClaimPendingDeliveriesRow, error) {
	return nil, nil
}

func (f *fakeNotificationQuerier) CountDeliveries(context.Context, notificationsqlc.CountDeliveriesParams) (int64, error) {
	return 0, nil
}

func (f *fakeNotificationQuerier) CreateDelivery(_ context.Context, arg notificationsqlc.CreateDeliveryParams) (notificationsqlc.CreateDeliveryRow, error) {
	f.createDeliveryParams = arg
	return f.createDeliveryResult, nil
}

func (f *fakeNotificationQuerier) GetDelivery(context.Context, int64) (notificationsqlc.GetDeliveryRow, error) {
	return notificationsqlc.GetDeliveryRow{}, nil
}

func (f *fakeNotificationQuerier) ListDeliveries(context.Context, notificationsqlc.ListDeliveriesParams) ([]notificationsqlc.ListDeliveriesRow, error) {
	return nil, nil
}

func (f *fakeNotificationQuerier) MarkDeliveryFailed(_ context.Context, arg notificationsqlc.MarkDeliveryFailedParams) error {
	f.markFailedParams = arg
	return nil
}

func (f *fakeNotificationQuerier) MarkDeliverySent(_ context.Context, arg notificationsqlc.MarkDeliverySentParams) error {
	f.markSentParams = arg
	return nil
}

func (f *fakeNotificationQuerier) ScheduleDeliveryRetry(_ context.Context, arg notificationsqlc.ScheduleDeliveryRetryParams) error {
	f.scheduleRetryParams = arg
	return nil
}

func TestNotificationStatusTransitions(t *testing.T) {
	profileSyncer := &fakeProfileSyncer{}
	service := NewService(notificationstore.NewSQLStore(nil), &fakeSender{}, profileSyncer)

	resp, err := service.GetNotificationStatus(context.Background(), &apiclient.GetNotificationStatusRequest{})
	if err != nil {
		t.Fatalf("GetNotificationStatus before start: %v", err)
	}
	if resp.Started || resp.Status != "stopped" {
		t.Fatalf("status before start = %#v, want stopped", resp)
	}

	if err := service.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer service.Stop()
	if profileSyncer.calls != 1 {
		t.Fatalf("profile sync calls = %d, want 1", profileSyncer.calls)
	}
	resp, err = service.GetNotificationStatus(context.Background(), &apiclient.GetNotificationStatusRequest{})
	if err != nil {
		t.Fatalf("GetNotificationStatus after start: %v", err)
	}
	if !resp.Started || resp.Status != "running" {
		t.Fatalf("status after start = %#v, want running", resp)
	}

	if err := service.Stop(); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	resp, err = service.GetNotificationStatus(context.Background(), &apiclient.GetNotificationStatusRequest{})
	if err != nil {
		t.Fatalf("GetNotificationStatus after stop: %v", err)
	}
	if resp.Started || resp.Status != "stopped" {
		t.Fatalf("status after stop = %#v, want stopped", resp)
	}
}

func TestNotificationStartRequiresStore(t *testing.T) {
	err := NewService(nil, &fakeSender{}, &fakeProfileSyncer{}).Start(context.Background())
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("Start error = %v, want FailedPrecondition", err)
	}
}

func TestNotificationStartRequiresSender(t *testing.T) {
	err := NewService(notificationstore.NewSQLStore(nil), nil, &fakeProfileSyncer{}).Start(context.Background())
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("Start error = %v, want FailedPrecondition", err)
	}
}

func TestNotificationStartRequiresProfileSyncer(t *testing.T) {
	err := NewService(notificationstore.NewSQLStore(nil), &fakeSender{}, nil).Start(context.Background())
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("Start error = %v, want FailedPrecondition", err)
	}
}

func TestNotificationStartProfileSyncFailureLeavesStopped(t *testing.T) {
	service := NewService(notificationstore.NewSQLStore(nil), &fakeSender{}, &fakeProfileSyncer{err: errors.New("telegram profile unavailable")})

	err := service.Start(context.Background())
	if status.Code(err) != codes.Unavailable {
		t.Fatalf("Start error = %v, want Unavailable", err)
	}
	resp, statusErr := service.GetNotificationStatus(context.Background(), &apiclient.GetNotificationStatusRequest{})
	if statusErr != nil {
		t.Fatalf("GetNotificationStatus: %v", statusErr)
	}
	if resp.Started || resp.Status != "stopped" {
		t.Fatalf("status after failed start = %#v, want stopped", resp)
	}
}

func TestNotificationStartIsIdempotentAfterProfileSync(t *testing.T) {
	profileSyncer := &fakeProfileSyncer{}
	service := NewService(notificationstore.NewSQLStore(nil), &fakeSender{}, profileSyncer)

	if err := service.Start(context.Background()); err != nil {
		t.Fatalf("first Start: %v", err)
	}
	defer service.Stop()
	if err := service.Start(context.Background()); err != nil {
		t.Fatalf("second Start: %v", err)
	}
	if profileSyncer.calls != 1 {
		t.Fatalf("profile sync calls = %d, want 1", profileSyncer.calls)
	}
}

func TestSendNotificationQueuesDelivery(t *testing.T) {
	createdAt := time.Date(2026, time.May, 27, 12, 0, 0, 0, time.UTC)
	querier := &fakeNotificationQuerier{
		createDeliveryResult: notificationsqlc.CreateDeliveryRow{
			ID:        7,
			Source:    "worm",
			Severity:  "warning",
			Title:     "Scan finished",
			Body:      "Contract risk changed",
			Link:      "https://example.com",
			Channel:   "telegram",
			Status:    "pending",
			Topic:     "token",
			CreatedAt: pgtype.Timestamptz{Time: createdAt, Valid: true},
		},
	}

	sender := &fakeSender{messageID: "123"}
	resp, err := NewService(notificationstore.NewSQLStoreWithQuerier(querier), sender, &fakeProfileSyncer{}).SendNotification(context.Background(), &apiclient.SendNotificationRequest{
		Source:   "worm",
		Severity: apiclient.NotificationSeverity_NOTIFICATION_SEVERITY_WARNING,
		Title:    "Scan finished",
		Body:     "Contract risk changed",
		Link:     "https://example.com",
		Topic:    apiclient.NotificationTopic_NOTIFICATION_TOPIC_TOKEN,
	})
	if err != nil {
		t.Fatalf("SendNotification: %v", err)
	}
	if resp.NotificationId != 7 || resp.Status != apiclient.NotificationDeliveryStatus_NOTIFICATION_DELIVERY_STATUS_PENDING || resp.ProviderMessageId != "" {
		t.Fatalf("response = %#v, want pending delivery", resp)
	}
	if sender.calls != 0 {
		t.Fatalf("sender calls = %d, want notification queued without synchronous send", sender.calls)
	}
	if querier.createDeliveryParams.Topic != "token" {
		t.Fatalf("topic create = %q, want token", querier.createDeliveryParams.Topic)
	}
}

func TestSendNotificationPolyTopic(t *testing.T) {
	querier := &fakeNotificationQuerier{
		createDeliveryResult: notificationsqlc.CreateDeliveryRow{
			ID:        8,
			Source:    "polymarket",
			Severity:  "info",
			Body:      "Market updated",
			Channel:   "telegram",
			Status:    "pending",
			Topic:     "poly",
			CreatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
		},
	}
	sender := &fakeSender{}

	_, err := NewService(notificationstore.NewSQLStoreWithQuerier(querier), sender, &fakeProfileSyncer{}).SendNotification(context.Background(), &apiclient.SendNotificationRequest{
		Source:   "polymarket",
		Severity: apiclient.NotificationSeverity_NOTIFICATION_SEVERITY_INFO,
		Body:     "Market updated",
		Topic:    apiclient.NotificationTopic_NOTIFICATION_TOPIC_POLY,
	})
	if err != nil {
		t.Fatalf("SendNotification: %v", err)
	}
	if querier.createDeliveryParams.Topic != "poly" || sender.calls != 0 {
		t.Fatalf("topic create/sender calls = %q/%d, want queued poly", querier.createDeliveryParams.Topic, sender.calls)
	}
}

func TestSendNotificationRequiresTopic(t *testing.T) {
	_, err := NewService(notificationstore.NewSQLStoreWithQuerier(&fakeNotificationQuerier{}), &fakeSender{}, &fakeProfileSyncer{}).SendNotification(context.Background(), &apiclient.SendNotificationRequest{
		Source:   "worm",
		Severity: apiclient.NotificationSeverity_NOTIFICATION_SEVERITY_INFO,
		Body:     "Contract risk changed",
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("SendNotification error = %v, want InvalidArgument", err)
	}
}

func TestSendNotificationRejectsUnknownTopic(t *testing.T) {
	_, err := NewService(notificationstore.NewSQLStoreWithQuerier(&fakeNotificationQuerier{}), &fakeSender{}, &fakeProfileSyncer{}).SendNotification(context.Background(), &apiclient.SendNotificationRequest{
		Source:   "worm",
		Severity: apiclient.NotificationSeverity_NOTIFICATION_SEVERITY_INFO,
		Body:     "Contract risk changed",
		Topic:    apiclient.NotificationTopic(99),
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("SendNotification error = %v, want InvalidArgument", err)
	}
}

func TestWorkerMarksDeliverySent(t *testing.T) {
	querier := &fakeNotificationQuerier{}
	sender := &fakeSender{messageID: "123"}
	service := NewServiceWithWorkerConfig(notificationstore.NewSQLStoreWithQuerier(querier), sender, &fakeProfileSyncer{}, WorkerConfig{Disabled: true})

	service.processClaimedDelivery(context.Background(), notificationstore.ClaimedDelivery{
		ID:       9,
		Source:   "application",
		Severity: "error",
		Body:     "Deploy failed",
		Topic:    "poly",
		Attempts: 1,
	})

	if querier.markSentParams.ID != 9 || querier.markSentParams.ProviderMessageID.String != "123" {
		t.Fatalf("mark sent params = %#v, want sent delivery", querier.markSentParams)
	}
	if sender.calls != 1 || sender.topic != "poly" || !strings.Contains(sender.text, "Severity: ERROR") || !strings.Contains(sender.text, "Deploy failed") {
		t.Fatalf("sender = %#v, want rendered poly send", sender)
	}
}

func TestWorkerSchedulesRetry(t *testing.T) {
	querier := &fakeNotificationQuerier{}
	service := NewServiceWithWorkerConfig(notificationstore.NewSQLStoreWithQuerier(querier), &fakeSender{err: errors.New("telegram unavailable")}, &fakeProfileSyncer{}, WorkerConfig{
		Disabled:    true,
		MaxAttempts: 5,
	})

	before := time.Now().UTC().Add(1500 * time.Millisecond)
	service.processClaimedDelivery(context.Background(), notificationstore.ClaimedDelivery{
		ID:       10,
		Source:   "application",
		Severity: "error",
		Body:     "Deploy failed",
		Topic:    "poly",
		Attempts: 1,
	})
	after := time.Now().UTC().Add(2500 * time.Millisecond)

	if querier.scheduleRetryParams.ID != 10 || querier.scheduleRetryParams.ErrorMessage.String != "telegram unavailable" {
		t.Fatalf("retry params = %#v, want retry scheduled", querier.scheduleRetryParams)
	}
	if querier.scheduleRetryParams.NextAttemptAt.Time.Before(before) || querier.scheduleRetryParams.NextAttemptAt.Time.After(after) {
		t.Fatalf("next attempt = %s, want about 2s from now", querier.scheduleRetryParams.NextAttemptAt.Time)
	}
}

func TestWorkerUsesTelegramRetryAfter(t *testing.T) {
	querier := &fakeNotificationQuerier{}
	service := NewServiceWithWorkerConfig(notificationstore.NewSQLStoreWithQuerier(querier), &fakeSender{err: &RateLimitError{RetryAfter: 3 * time.Second, Err: errors.New("rate limited")}}, &fakeProfileSyncer{}, WorkerConfig{
		Disabled:    true,
		MaxAttempts: 5,
	})

	before := time.Now().UTC().Add(2500 * time.Millisecond)
	service.processClaimedDelivery(context.Background(), notificationstore.ClaimedDelivery{
		ID:       11,
		Source:   "application",
		Severity: "error",
		Body:     "Deploy failed",
		Topic:    "poly",
		Attempts: 1,
	})
	after := time.Now().UTC().Add(3500 * time.Millisecond)

	if querier.scheduleRetryParams.ID != 11 {
		t.Fatalf("retry params = %#v, want retry scheduled", querier.scheduleRetryParams)
	}
	if querier.scheduleRetryParams.NextAttemptAt.Time.Before(before) || querier.scheduleRetryParams.NextAttemptAt.Time.After(after) {
		t.Fatalf("next attempt = %s, want about retry_after 3s from now", querier.scheduleRetryParams.NextAttemptAt.Time)
	}
}

func TestWorkerMarksDeliveryFailedAfterMaxAttempts(t *testing.T) {
	querier := &fakeNotificationQuerier{}
	service := NewServiceWithWorkerConfig(notificationstore.NewSQLStoreWithQuerier(querier), &fakeSender{err: errors.New("telegram unavailable")}, &fakeProfileSyncer{}, WorkerConfig{
		Disabled:    true,
		MaxAttempts: 5,
	})

	service.processClaimedDelivery(context.Background(), notificationstore.ClaimedDelivery{
		ID:       9,
		Source:   "application",
		Severity: "error",
		Body:     "Deploy failed",
		Topic:    "poly",
		Attempts: 5,
	})

	if querier.markFailedParams.ID != 9 || querier.markFailedParams.ErrorMessage.String != "telegram unavailable" {
		t.Fatalf("mark failed params = %#v", querier.markFailedParams)
	}
}
