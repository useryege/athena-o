package notification

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/useryege/athena/internal/notification/apiclient"
	notificationstore "github.com/useryege/athena/internal/notification/store"
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
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new: %v", err)
	}
	defer db.Close()

	createdAt := time.Date(2026, time.May, 27, 12, 0, 0, 0, time.UTC)
	mock.ExpectQuery("INSERT INTO notification_deliveries").
		WithArgs("worm", "warning", "Scan finished", "Contract risk changed", "https://example.com", "telegram", "pending").
		WillReturnRows(sqlmock.NewRows([]string{"id", "source", "severity", "title", "body", "link", "channel", "status", "created_at"}).
			AddRow(int64(7), "worm", "warning", "Scan finished", "Contract risk changed", "https://example.com", "telegram", "pending", createdAt))
	mock.ExpectExec("UPDATE notification_deliveries").
		WithArgs(int64(7), "123").
		WillReturnResult(sqlmock.NewResult(0, 1))

	sender := &fakeSender{messageID: "123"}
	resp, err := NewService(notificationstore.NewSQLStore(db), sender).SendNotification(context.Background(), &apiclient.SendNotificationRequest{
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
	if !strings.Contains(sender.text, "Severity: WARNING") || !strings.Contains(sender.text, "Contract risk changed") {
		t.Fatalf("telegram text = %q, want rendered notification text", sender.text)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestSendNotificationFailureRecordsDelivery(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new: %v", err)
	}
	defer db.Close()

	createdAt := time.Date(2026, time.May, 27, 12, 0, 0, 0, time.UTC)
	mock.ExpectQuery("INSERT INTO notification_deliveries").
		WithArgs("application", "error", "", "Deploy failed", "", "telegram", "pending").
		WillReturnRows(sqlmock.NewRows([]string{"id", "source", "severity", "title", "body", "link", "channel", "status", "created_at"}).
			AddRow(int64(9), "application", "error", "", "Deploy failed", "", "telegram", "pending", createdAt))
	mock.ExpectExec("UPDATE notification_deliveries").
		WithArgs(int64(9), "telegram unavailable").
		WillReturnResult(sqlmock.NewResult(0, 1))

	resp, err := NewService(notificationstore.NewSQLStore(db), &fakeSender{err: errors.New("telegram unavailable")}).SendNotification(context.Background(), &apiclient.SendNotificationRequest{
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
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}
