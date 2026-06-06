package token

import (
	"context"
	"testing"

	"github.com/useryege/athena/internal/token/apiclient"
)

func TestTokenStatusTransitions(t *testing.T) {
	service := NewService()

	resp, err := service.GetTokenStatus(context.Background(), &apiclient.GetTokenStatusRequest{})
	if err != nil {
		t.Fatalf("GetTokenStatus before start: %v", err)
	}
	if resp.Started || resp.Status != "stopped" {
		t.Fatalf("status before start = %#v, want stopped", resp)
	}

	if err := service.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if err := service.Start(); err != nil {
		t.Fatalf("second Start: %v", err)
	}
	resp, err = service.GetTokenStatus(context.Background(), &apiclient.GetTokenStatusRequest{})
	if err != nil {
		t.Fatalf("GetTokenStatus after start: %v", err)
	}
	if !resp.Started || resp.Status != "running" {
		t.Fatalf("status after start = %#v, want running", resp)
	}

	if err := service.Stop(); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if err := service.Stop(); err != nil {
		t.Fatalf("second Stop: %v", err)
	}
	resp, err = service.GetTokenStatus(context.Background(), &apiclient.GetTokenStatusRequest{})
	if err != nil {
		t.Fatalf("GetTokenStatus after stop: %v", err)
	}
	if resp.Started || resp.Status != "stopped" {
		t.Fatalf("status after stop = %#v, want stopped", resp)
	}
}
