package wallet

import (
	"context"
	"testing"

	"github.com/useryege/athena/internal/wallet/apiclient"
	walletstore "github.com/useryege/athena/internal/wallet/store"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestWalletStatusTransitions(t *testing.T) {
	service := NewService(walletstore.NewSQLStore(nil))

	resp, err := service.GetWalletStatus(context.Background(), &apiclient.GetWalletStatusRequest{})
	if err != nil {
		t.Fatalf("GetWalletStatus before start: %v", err)
	}
	if resp.Started || resp.Status != "stopped" {
		t.Fatalf("status before start = %#v, want stopped", resp)
	}

	if err := service.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	resp, err = service.GetWalletStatus(context.Background(), &apiclient.GetWalletStatusRequest{})
	if err != nil {
		t.Fatalf("GetWalletStatus after start: %v", err)
	}
	if !resp.Started || resp.Status != "running" {
		t.Fatalf("status after start = %#v, want running", resp)
	}

	if err := service.Stop(); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	resp, err = service.GetWalletStatus(context.Background(), &apiclient.GetWalletStatusRequest{})
	if err != nil {
		t.Fatalf("GetWalletStatus after stop: %v", err)
	}
	if resp.Started || resp.Status != "stopped" {
		t.Fatalf("status after stop = %#v, want stopped", resp)
	}
}

func TestWalletStartRequiresStore(t *testing.T) {
	err := NewService(nil).Start()
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("Start error = %v, want FailedPrecondition", err)
	}
}
