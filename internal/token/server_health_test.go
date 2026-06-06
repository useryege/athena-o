package token

import (
	"context"
	"net"
	"testing"
	"time"

	tokenpkg "github.com/useryege/athena/internal/token/apiclient"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
)

func TestTokenServerHealthStatusTransitions(t *testing.T) {
	healthServer := health.NewServer()
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_NOT_SERVING)
	server := &Server{healthService: healthServer}

	address, stop := serveTokenGRPC(t, server.CreateGRPC())
	defer stop()

	if status := checkHealthStatus(t, address); status != grpc_health_v1.HealthCheckResponse_NOT_SERVING {
		t.Fatalf("initial health status = %s, want NOT_SERVING", status)
	}

	server.setHealthStatus(grpc_health_v1.HealthCheckResponse_SERVING)
	if status := checkHealthStatus(t, address); status != grpc_health_v1.HealthCheckResponse_SERVING {
		t.Fatalf("serving health status = %s, want SERVING", status)
	}

	server.setHealthStatus(grpc_health_v1.HealthCheckResponse_NOT_SERVING)
	if status := checkHealthStatus(t, address); status != grpc_health_v1.HealthCheckResponse_NOT_SERVING {
		t.Fatalf("stopped health status = %s, want NOT_SERVING", status)
	}
}

func serveTokenGRPC(t *testing.T, grpcServer *grpc.Server) (string, func()) {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		if err := grpcServer.Serve(listener); err != nil {
			t.Logf("token health test server stopped: %v", err)
		}
	}()

	stop := func() {
		grpcServer.Stop()
		<-done
	}
	return listener.Addr().String(), stop
}

func checkHealthStatus(t *testing.T, address string) grpc_health_v1.HealthCheckResponse_ServingStatus {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	conn, err := tokenpkg.NewConnection(address)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer conn.Close()

	resp, err := grpc_health_v1.NewHealthClient(conn).Check(ctx, &grpc_health_v1.HealthCheckRequest{})
	if err != nil {
		t.Fatalf("health check: %v", err)
	}
	return resp.GetStatus()
}
