//go:build integration

package acceptance

import (
	"context"
	facade "github.com/useryege/athena/internal/server/tradersync"
	ts "github.com/useryege/athena/internal/tradersync"
	trpc "github.com/useryege/athena/internal/tradersync/apiclient"
	"github.com/useryege/athena/internal/tradersync/transport"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/test/bufconn"
	"net"
	"os"
	"sync/atomic"
	"testing"
)

func newInternalFacade(t *testing.T, service *ts.Service, actors facade.ActorResolver) *facade.Server {
	t.Helper()
	const token = "acceptance-internal-token-0123456789"
	listener := bufconn.Listen(1024 * 1024)
	var calls atomic.Uint64
	server := grpc.NewServer(grpc.ChainUnaryInterceptor(transport.NewUnaryInterceptor(token, func() bool { return true }), func(ctx context.Context, request any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		calls.Add(1)
		return handler(ctx, request)
	}))
	trpc.RegisterTraderSyncServiceServer(server, transport.NewServer(service))
	go server.Serve(listener)
	t.Cleanup(func() {
		server.Stop()
		if calls.Load() == 0 && os.Getenv("ATHENA_UI_E2E_FIXTURE_ONLY") != "1" {
			t.Error("acceptance facade bypassed the authenticated internal gRPC handler")
		}
		t.Logf("acceptance authenticated internal gRPC handler calls: %d", calls.Load())
	})
	conn, err := grpc.DialContext(context.Background(), "passthrough:///internal", grpc.WithInsecure(), grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return listener.Dial() }), grpc.WithUnaryInterceptor(func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoke grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		return invoke(metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+token), method, req, reply, cc, opts...)
	}))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	return facade.New(trpc.NewTraderSyncServiceClient(conn), actors)
}
