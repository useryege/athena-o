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
	text      string
}

func (f *fakeSender) Send(_ context.Context, text string) (string, error) {
	f.text = text
	if f.err != nil {
		return "", f.err
	}
	if f.messageID != "" {
		return f.messageID, nil
	}
	return "1", nil
}

type fakeNotificationQuerier struct {
	createDeliveryResult notificationsqlc.CreateDeliveryRow
	markSentParams       notificationsqlc.MarkDeliverySentParams
	markFailedParams     notificationsqlc.MarkDeliveryFailedParams
}

func (f *fakeNotificationQuerier) CountDeliveries(context.Context, notificationsqlc.CountDeliveriesParams) (int64, error) {
	return 0, nil
}

func (f *fakeNotificationQuerier) CreateDelivery(context.Context, notificationsqlc.CreateDeliveryParams) (notificationsqlc.CreateDeliveryRow, error) {
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

func TestNotificationStatusTransitions(t *testing.T) {
	service := NewService(notificationstore.NewSQLStore(nil), &fakeSender{})

	resp, err := service.GetNotificationStatus(context.Background(), &apiclient.GetNotificationStatusRequest{})
	if err != nil {
		t.Fatalf("GetNotificationStatus before start: %v", err)
	}
	if resp.Started || resp.Status != "stopped" {
		t.Fatalf("status before start = %#v, want stopped", resp)
	}

	if err := service.Start(); err != nil {
		t.Fatalf("Start: %v", err)
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
	err := NewService(nil, &fakeSender{}).Start()
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("Start error = %v, want FailedPrecondition", err)
	}
}

func TestNotificationStartRequiresSender(t *testing.T) {
	err := NewService(notificationstore.NewSQLStore(nil), nil).Start()
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("Start error = %v, want FailedPrecondition", err)
	}
}

func TestSendNotificationSuccessRecordsDelivery(t *testing.T) {
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
			CreatedAt: pgtype.Timestamptz{Time: createdAt, Valid: true},
		},
	}

	sender := &fakeSender{messageID: "123"}
	resp, err := NewService(notificationstore.NewSQLStoreWithQuerier(querier), sender).SendNotification(context.Background(), &apiclient.SendNotificationRequest{
		Source:   "worm",
		Severity: apiclient.NotificationSeverity_NOTIFICATION_SEVERITY_WARNING,
		Title:    "Scan finished",
		Body:     "Contract risk changed",
		Link:     "https://example.com",
	})
	if err != nil {
		t.Fatalf("SendNotification: %v", err)
	}
	if resp.NotificationId != 7 || resp.Status != apiclient.NotificationDeliveryStatus_NOTIFICATION_DELIVERY_STATUS_SENT || resp.ProviderMessageId != "123" {
		t.Fatalf("response = %#v, want sent delivery", resp)
	}
	if querier.markSentParams.ID != 7 || querier.markSentParams.ProviderMessageID.String != "123" {
		t.Fatalf("mark sent params = %#v", querier.markSentParams)
	}
	if !strings.Contains(sender.text, "Severity: WARNING") || !strings.Contains(sender.text, "Contract risk changed") {
		t.Fatalf("telegram text = %q, want rendered notification text", sender.text)
	}
}

func TestSendNotificationFailureRecordsDelivery(t *testing.T) {
	createdAt := time.Date(2026, time.May, 27, 12, 0, 0, 0, time.UTC)
	querier := &fakeNotificationQuerier{
		createDeliveryResult: notificationsqlc.CreateDeliveryRow{
			ID:        9,
			Source:    "application",
			Severity:  "error",
			Title:     "",
			Body:      "Deploy failed",
			Link:      "",
			Channel:   "telegram",
			Status:    "pending",
			CreatedAt: pgtype.Timestamptz{Time: createdAt, Valid: true},
		},
	}

	resp, err := NewService(notificationstore.NewSQLStoreWithQuerier(querier), &fakeSender{err: errors.New("telegram unavailable")}).SendNotification(context.Background(), &apiclient.SendNotificationRequest{
		Source:   "application",
		Severity: apiclient.NotificationSeverity_NOTIFICATION_SEVERITY_ERROR,
		Body:     "Deploy failed",
	})
	if status.Code(err) != codes.Unavailable {
		t.Fatalf("SendNotification error = %v, want Unavailable", err)
	}
	if resp == nil || resp.NotificationId != 9 || resp.Status != apiclient.NotificationDeliveryStatus_NOTIFICATION_DELIVERY_STATUS_FAILED {
		t.Fatalf("response = %#v, want failed delivery response", resp)
	}
	if querier.markFailedParams.ID != 9 || querier.markFailedParams.ErrorMessage.String != "telegram unavailable" {
		t.Fatalf("mark failed params = %#v", querier.markFailedParams)
	}
}
