package polymarket

import (
	"context"
	"testing"

	"github.com/useryege/athena/internal/polymarket/apiclient"
	polymarketstore "github.com/useryege/athena/internal/polymarket/store"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestPolymarketStatusTransitions(t *testing.T) {
	service := NewService(polymarketstore.NewSQLStore(nil))

	resp, err := service.GetPolymarketStatus(context.Background(), &apiclient.GetPolymarketStatusRequest{})
	if err != nil {
		t.Fatalf("GetPolymarketStatus before start: %v", err)
	}
	if resp.Started || resp.Status != "stopped" {
		t.Fatalf("status before start = %#v, want stopped", resp)
	}

	if err := service.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	resp, err = service.GetPolymarketStatus(context.Background(), &apiclient.GetPolymarketStatusRequest{})
	if err != nil {
		t.Fatalf("GetPolymarketStatus after start: %v", err)
	}
	if !resp.Started || resp.Status != "running" {
		t.Fatalf("status after start = %#v, want running", resp)
	}

	if err := service.Stop(); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	resp, err = service.GetPolymarketStatus(context.Background(), &apiclient.GetPolymarketStatusRequest{})
	if err != nil {
		t.Fatalf("GetPolymarketStatus after stop: %v", err)
	}
	if resp.Started || resp.Status != "stopped" {
		t.Fatalf("status after stop = %#v, want stopped", resp)
	}
}

func TestPolymarketStartRequiresStore(t *testing.T) {
	err := NewService(nil).Start()
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("Start error = %v, want FailedPrecondition", err)
	}
}
