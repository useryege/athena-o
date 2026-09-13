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
	"testing"
)

func newInternalFacade(t *testing.T, service *ts.Service, actors facade.ActorResolver) *facade.Server {
	t.Helper()
	const token = "acceptance-internal-token-0123456789"
	listener := bufconn.Listen(1024 * 1024)
	server := grpc.NewServer(grpc.UnaryInterceptor(transport.NewUnaryInterceptor(token, func() bool { return true })))
	trpc.RegisterTraderSyncServiceServer(server, transport.NewServer(service))
	go server.Serve(listener)
	t.Cleanup(server.Stop)
	conn, err := grpc.DialContext(context.Background(), "passthrough:///internal", grpc.WithInsecure(), grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return listener.Dial() }), grpc.WithUnaryInterceptor(func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoke grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		return invoke(metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+token), method, req, reply, cc, opts...)
	}))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	return facade.New(trpc.NewTraderSyncServiceClient(conn), actors)
}
