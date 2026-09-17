package util

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

type blockingHealth struct {
	healthpb.UnimplementedHealthServer
	started, canceled chan struct{}
}

func (h *blockingHealth) Check(ctx context.Context, _ *healthpb.HealthCheckRequest) (*healthpb.HealthCheckResponse, error) {
	close(h.started)
	<-ctx.Done()
	close(h.canceled)
	return nil, ctx.Err()
}

// A stuck RPC and a stuck background worker must not hold the standalone
// command forever; cancellation must close the listener and force RPC drain.
func TestServeGRPCBoundsBlockedWork(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	server := grpc.NewServer()
	h := &blockingHealth{started: make(chan struct{}), canceled: make(chan struct{})}
	healthpb.RegisterHealthServer(server, h)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	release := make(chan struct{})
	defer close(release)
	done := make(chan error, 1)
	go func() {
		done <- serveGRPC(ctx, listener, server, func() error { <-release; return nil }, 100*time.Millisecond)
	}()
	client, err := grpc.NewClient(listener.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	defer client.Close()
	go func() {
		_, _ = healthpb.NewHealthClient(client).Check(context.Background(), &healthpb.HealthCheckRequest{})
	}()
	select {
	case <-h.started:
	case <-time.After(time.Second):
		t.Fatal("RPC not started")
	}
	cancel()
	select {
	case err := <-done:
		require.ErrorContains(t, err, "shutdown")
	case <-time.After(time.Second):
		t.Fatal("shutdown exceeded budget")
	}
	select {
	case <-h.canceled:
	case <-time.After(time.Second):
		t.Fatal("RPC context was not canceled")
	}
	conn, err := net.DialTimeout("tcp", listener.Addr().String(), time.Millisecond*50)
	if err == nil {
		conn.Close()
		t.Fatal("listener still open")
	}
}
func TestServeGRPCPreservesCleanupFailure(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	failure := errors.New("worker failed")
	require.ErrorIs(t, serveGRPC(ctx, listener, grpc.NewServer(), func() error { return failure }, time.Second), failure)
}
