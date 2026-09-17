package devruntime

import (
	"context"
	"net"
	"strings"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

type slowGatewayHealth struct {
	healthpb.UnimplementedHealthServer
}

func (slowGatewayHealth) Check(ctx context.Context, _ *healthpb.HealthCheckRequest) (*healthpb.HealthCheckResponse, error) {
	<-ctx.Done()
	return nil, ctx.Err()
}
func TestFiveGatewayHealthChecksRunInParallelWithThreeSecondBudget(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	server := grpc.NewServer()
	healthpb.RegisterHealthServer(server, slowGatewayHealth{})
	go server.Serve(listener)
	defer server.Stop()
	addresses := []string{listener.Addr().String(), listener.Addr().String(), listener.Addr().String(), listener.Addr().String(), listener.Addr().String()}
	start := time.Now()
	results := probeGatewayAddresses(context.Background(), addresses)
	if elapsed := time.Since(start); elapsed > 5*time.Second || elapsed < 2*time.Second {
		t.Fatalf("health budget not 3s parallel: %v", elapsed)
	}
	if len(results) != 5 {
		t.Fatal("missing gateway result", results)
	}
	for _, result := range results {
		if result.Error == "" || result.Serving || !strings.Contains(result.Error, "DeadlineExceeded") {
			t.Fatal("timed out gateway reported serving", result)
		}
	}
}

func TestGatewayHealthDoesNotRequireReflection(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	server := grpc.NewServer()
	healthServer := health.NewServer()
	healthServer.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	healthpb.RegisterHealthServer(server, healthServer)
	go server.Serve(listener)
	defer server.Stop()
	results := probeGatewayAddresses(context.Background(), []string{listener.Addr().String()})
	if len(results) != 1 || !results[0].Serving || results[0].Error != "" {
		t.Fatal(results)
	}
}
