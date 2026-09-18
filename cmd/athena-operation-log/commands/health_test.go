package commands

import (
	"context"
	"net"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

func TestHealthCommandReportsServingOperationLog(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()
	grpcServer := grpc.NewServer()
	healthServer := health.NewServer()
	healthServer.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	healthpb.RegisterHealthServer(grpcServer, healthServer)
	go grpcServer.Serve(listener)
	defer grpcServer.Stop()

	command := NewCommand()
	command.SetArgs([]string{"health", "--target", listener.Addr().String(), "--transport", "plaintext"})
	require.NoError(t, command.ExecuteContext(context.Background()))
}
