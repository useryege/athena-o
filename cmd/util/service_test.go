package util

import (
	"context"
	"errors"
	"net"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
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
		done <- serveGRPC(ctx, listener, server, func() error { <-release; return nil }, func() error { return nil }, 100*time.Millisecond)
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
	require.ErrorIs(t, serveGRPC(ctx, listener, grpc.NewServer(), func() error { return failure }, func() error { return nil }, time.Second), failure)
}

type resourceHealth struct {
	healthpb.UnimplementedHealthServer
	entered, resume chan struct{}
	file            *os.File
}

func (h *resourceHealth) Check(ctx context.Context, _ *healthpb.HealthCheckRequest) (*healthpb.HealthCheckResponse, error) {
	close(h.entered)
	select {
	case <-h.resume:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	var b [1]byte
	if _, err := h.file.ReadAt(b[:], 0); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &healthpb.HealthCheckResponse{Status: healthpb.HealthCheckResponse_SERVING}, nil
}

// A handler accepted before shutdown must retain access to shared resources
// until it returns, even when all background work has already stopped.
func TestServeGRPCKeepsResourcesUntilAcceptedRPCReturns(t *testing.T) {
	resource, err := os.CreateTemp(t.TempDir(), "rpc-resource")
	require.NoError(t, err)
	defer resource.Close()
	_, err = resource.WriteString("ready")
	require.NoError(t, err)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	server := grpc.NewServer()
	handler := &resourceHealth{entered: make(chan struct{}), resume: make(chan struct{}), file: resource}
	healthpb.RegisterHealthServer(server, handler)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	backgroundDone := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		done <- serveGRPC(ctx, listener, server, func() error { close(backgroundDone); return nil }, resource.Close, time.Second)
	}()
	client, err := grpc.NewClient(listener.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	defer client.Close()
	rpcResult := make(chan error, 1)
	go func() {
		_, err := healthpb.NewHealthClient(client).Check(context.Background(), &healthpb.HealthCheckRequest{})
		rpcResult <- err
	}()
	select {
	case <-handler.entered:
	case <-time.After(time.Second):
		t.Fatal("handler never entered")
	}
	cancel()
	select {
	case <-backgroundDone:
	case <-time.After(time.Second):
		t.Fatal("background did not stop")
	}
	close(handler.resume)
	select {
	case err := <-rpcResult:
		require.NoError(t, err, "accepted RPC lost its resource during drain")
	case <-time.After(time.Second):
		t.Fatal("handler never returned")
	}
	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("shutdown never returned")
	}
	_, err = resource.Stat()
	require.ErrorIs(t, err, os.ErrClosed, "resource was not released after drain")
}

func TestServeGRPCPreservesResourceReleaseFailure(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	stopErr, releaseErr := errors.New("background stop failed"), errors.New("resource release failed")
	err = serveGRPC(ctx, listener, grpc.NewServer(), func() error { return stopErr }, func() error { return releaseErr }, time.Second)
	require.ErrorIs(t, err, stopErr)
	require.ErrorIs(t, err, releaseErr)
}

func TestServeGRPCResourceReleaseSharesTotalBudget(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	unblock := make(chan struct{})
	defer close(unblock)
	start := time.Now()
	err = serveGRPC(ctx, listener, grpc.NewServer(), func() error { time.Sleep(300 * time.Millisecond); return nil }, func() error { <-unblock; return nil }, 400*time.Millisecond)
	require.ErrorContains(t, err, "releasing resources")
	require.Less(t, time.Since(start), 600*time.Millisecond, "resource release restarted the shutdown budget")
}
