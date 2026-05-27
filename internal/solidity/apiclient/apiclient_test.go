package apiclient

import (
	"context"
	"net"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
)

func TestWaitForSolidityServiceReturnsWhenServing(t *testing.T) {
	healthServer := health.NewServer()
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)

	grpcServer := grpc.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		if err := grpcServer.Serve(listener); err != nil {
			t.Logf("solidity health test server stopped: %v", err)
		}
	}()
	defer func() {
		grpcServer.Stop()
		<-done
	}()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := waitForSolidityService(ctx, listener.Addr().String(), time.Millisecond); err != nil {
		t.Fatalf("waitForSolidityService: %v", err)
	}
}
