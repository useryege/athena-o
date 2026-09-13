package solana

import (
	"context"
	"testing"
	"time"

	"github.com/useryege/athena/internal/accountcredentials"
	api "github.com/useryege/athena/pkg/apiclient/solana"
	utilsession "github.com/useryege/athena/util/session"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const accountID = "a04e3483-d207-44ea-a815-4dc392d954dd"

type downstream struct {
	api.SolanaServiceClient
	id      []string
	timeout time.Duration
	err     error
}

func (d *downstream) ListProjects(ctx context.Context, _ *api.ListProjectsRequest, _ ...grpc.CallOption) (*api.ListProjectsResponse, error) {
	md, _ := metadata.FromOutgoingContext(ctx)
	d.id = md.Get("x-athena-account-id")
	deadline, _ := ctx.Deadline()
	d.timeout = time.Until(deadline)
	return &api.ListProjectsResponse{TotalSize: 1}, d.err
}
func (d *downstream) GetDiscoveryStatus(ctx context.Context, _ *api.GetDiscoveryStatusRequest, _ ...grpc.CallOption) (*api.GetDiscoveryStatusResponse, error) {
	md, _ := metadata.FromOutgoingContext(ctx)
	d.id = md.Get("x-athena-account-id")
	deadline, _ := ctx.Deadline()
	d.timeout = time.Until(deadline)
	return &api.GetDiscoveryStatusResponse{Status: "ready"}, d.err
}

func authenticatedContext() context.Context {
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-athena-account-id", "spoofed"))
	ctx = metadata.NewOutgoingContext(ctx, metadata.Pairs("x-athena-account-id", "spoofed"))
	return utilsession.WithAuthenticatedCredential(ctx, accountcredentials.AuthenticatedCredential{AccountID: accountID, JTI: "test-session"})
}

func TestProxyInjectsAuthenticatedAccountAndDeadline(t *testing.T) {
	client := &downstream{}
	server := NewServer(client)
	if _, err := server.ListProjects(authenticatedContext(), &api.ListProjectsRequest{}); err != nil {
		t.Fatal(err)
	}
	if len(client.id) != 1 || client.id[0] != accountID {
		t.Fatalf("forwarded account metadata = %v", client.id)
	}
	if client.timeout <= 0 || client.timeout > 10*time.Second {
		t.Fatalf("deadline remaining = %s", client.timeout)
	}
	if _, err := server.GetDiscoveryStatus(authenticatedContext(), &api.GetDiscoveryStatusRequest{}); err != nil {
		t.Fatal(err)
	}
	if len(client.id) != 1 || client.id[0] != accountID {
		t.Fatalf("status account metadata = %v", client.id)
	}
}

func TestProxyRejectsMissingCredentialAndExplainsDependencyFailure(t *testing.T) {
	client := &downstream{}
	server := NewServer(client)
	_, err := server.ListProjects(context.Background(), &api.ListProjectsRequest{})
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("missing credential = %v", err)
	}
	client.err = status.Error(codes.Unavailable, "connection refused")
	_, err = server.ListProjects(authenticatedContext(), &api.ListProjectsRequest{})
	if status.Code(err) != codes.Unavailable {
		t.Fatalf("dependency failure = %v", err)
	}
	missingClient := NewServer(nil)
	_, err = missingClient.GetDiscoveryStatus(authenticatedContext(), &api.GetDiscoveryStatusRequest{})
	if status.Code(err) != codes.Unavailable {
		t.Fatalf("unconfigured dependency = %v", err)
	}
}
