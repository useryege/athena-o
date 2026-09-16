package commands

import (
	"context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"io"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestCatalogBudgetRejectsInvalidSettingsBeforeDependencies(t *testing.T) {
	for _, tc := range []struct{ name, value string }{{"zero", "0s"}, {"negative", "-1s"}, {"malformed", "bad"}, {"below attempt", "4s"}} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("ATHENA_WORM_TRADING_CATALOG_BUDGET", tc.value)
			cmd := NewCommand()
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			defer cancel()
			cmd.SetContext(ctx)
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)
			cmd.SetArgs([]string{})
			err := cmd.Execute()
			require.ErrorContains(t, err, "catalog budget")
		})
	}
}
func TestCatalogBudgetFlagOverridesEnvironment(t *testing.T) {
	t.Setenv("ATHENA_WORM_TRADING_CATALOG_BUDGET", "45s")
	cmd := NewCommand()
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	cmd.SetContext(ctx)
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{"--worm-catalog-budget", "1s"})
	require.ErrorContains(t, cmd.Execute(), "catalog budget")
}

// A long-lived health stream must not prevent the command from releasing its workers and pools.
func TestGracefulStopBoundsLongLivedRPC(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	server := grpc.NewServer()
	healthpb.RegisterHealthServer(server, health.NewServer())
	go server.Serve(listener)
	defer server.Stop()
	conn, err := grpc.NewClient(listener.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	defer conn.Close()
	stream, err := healthpb.NewHealthClient(conn).Watch(context.Background(), &healthpb.HealthCheckRequest{})
	require.NoError(t, err)
	_, err = stream.Recv()
	require.NoError(t, err)
	done := make(chan struct{})
	go func() { stopGRPC(server, 30*time.Millisecond); close(done) }()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("gRPC shutdown exceeded its bound")
	}
	_, err = stream.Recv()
	require.Error(t, err)
}

func TestGracefulStopDoesNotWaitForUncooperativeHandler(t *testing.T) {
	entered, release := make(chan struct{}), make(chan struct{})

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	server := grpc.NewServer(grpc.UnaryInterceptor(func(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		close(entered)
		<-release
		return handler(ctx, req)
	}))
	healthpb.RegisterHealthServer(server, health.NewServer())
	go server.Serve(listener)
	defer func() { close(release); server.Stop() }()
	conn, err := grpc.NewClient(listener.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	defer conn.Close()
	go healthpb.NewHealthClient(conn).Check(context.Background(), &healthpb.HealthCheckRequest{})
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("handler did not start")
	}
	done := make(chan struct{})
	go func() { stopGRPC(server, 30*time.Millisecond); close(done) }()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("shutdown waited for a handler that ignores transport cancellation")
	}
}
