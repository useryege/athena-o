package solidity

import (
	"context"
	"testing"

	"github.com/useryege/athena/internal/solidity/apiclient"
	soliditystore "github.com/useryege/athena/internal/solidity/store"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestSolidityStatusTransitions(t *testing.T) {
	service := NewService(soliditystore.NewSQLStore(nil))

	resp, err := service.GetSolidityStatus(context.Background(), &apiclient.GetSolidityStatusRequest{})
	if err != nil {
		t.Fatalf("GetSolidityStatus before start: %v", err)
	}
	if resp.Started || resp.Status != "stopped" {
		t.Fatalf("status before start = %#v, want stopped", resp)
	}

	if err := service.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	resp, err = service.GetSolidityStatus(context.Background(), &apiclient.GetSolidityStatusRequest{})
	if err != nil {
		t.Fatalf("GetSolidityStatus after start: %v", err)
	}
	if !resp.Started || resp.Status != "running" {
		t.Fatalf("status after start = %#v, want running", resp)
	}

	if err := service.Stop(); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	resp, err = service.GetSolidityStatus(context.Background(), &apiclient.GetSolidityStatusRequest{})
	if err != nil {
		t.Fatalf("GetSolidityStatus after stop: %v", err)
	}
	if resp.Started || resp.Status != "stopped" {
		t.Fatalf("status after stop = %#v, want stopped", resp)
	}
}

func TestSolidityStartRequiresStore(t *testing.T) {
	err := NewService(nil).Start()
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("Start error = %v, want FailedPrecondition", err)
	}
}
