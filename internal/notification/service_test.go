package notification

import (
	"context"
	"testing"

	"github.com/useryege/athena/internal/notification/apiclient"
	notificationstore "github.com/useryege/athena/internal/notification/store"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestNotificationStatusTransitions(t *testing.T) {
	service := NewService(notificationstore.NewSQLStore(nil))

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
	err := NewService(nil).Start()
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("Start error = %v, want FailedPrecondition", err)
	}
}
