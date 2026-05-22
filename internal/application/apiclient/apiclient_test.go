package apiclient

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
)

func TestWaitForApplicationServiceReturnsWhenServing(t *testing.T) {
	address, healthServer, stop := startHealthServer(t)
	defer stop()
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_NOT_SERVING)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	resultCh := make(chan error, 1)
	go func() {
		resultCh <- waitForApplicationService(ctx, address, 5*time.Millisecond)
	}()

	select {
	case err := <-resultCh:
		t.Fatalf("wait returned before health became SERVING: %v", err)
	case <-time.After(30 * time.Millisecond):
	}

	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)

	select {
	case err := <-resultCh:
		if err != nil {
			t.Fatalf("waitForApplicationService returned error: %v", err)
		}
	case <-ctx.Done():
		t.Fatalf("waitForApplicationService did not return after SERVING: %v", ctx.Err())
	}
}

func TestWaitForApplicationServiceReturnsContextErrorWhenUnavailable(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	address := listener.Addr().String()
	if err := listener.Close(); err != nil {
		t.Fatalf("close listener: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()

	err = waitForApplicationService(ctx, address, 5*time.Millisecond)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("waitForApplicationService error = %v, want DeadlineExceeded", err)
	}
}

func startHealthServer(t *testing.T) (string, *health.Server, func()) {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)

	done := make(chan struct{})
	go func() {
		defer close(done)
		if err := grpcServer.Serve(listener); err != nil {
			t.Logf("health test server stopped: %v", err)
		}
	}()

	stop := func() {
		grpcServer.Stop()
		<-done
	}
	return listener.Addr().String(), healthServer, stop
}
