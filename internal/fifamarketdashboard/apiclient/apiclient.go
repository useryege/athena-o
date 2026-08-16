package apiclient

import (
	"context"
	"math"

	"github.com/useryege/athena/common"
	"github.com/useryege/athena/util/env"
	utilgrpc "github.com/useryege/athena/util/grpc"
	"google.golang.org/grpc/health/grpc_health_v1"
)

var MaxGRPCMessageSize = env.ParseNumFromEnv(common.EnvGRPCMaxSizeMB, 100, 0, math.MaxInt32) * 1024 * 1024

// Clientset represents FIFA Market Dashboard API clients.
type Clientset interface {
	FIFAMarketDashboard() FIFAMarketDashboardServiceClient
	CheckHealth(ctx context.Context) (grpc_health_v1.HealthCheckResponse_ServingStatus, error)
	Close() error
}

type clientSet struct {
	connection *utilgrpc.ClientConnection
	client     FIFAMarketDashboardServiceClient
}

// NewFIFAMarketDashboardServiceClient creates a new FIFA Market Dashboard client.
func (c *clientSet) FIFAMarketDashboard() FIFAMarketDashboardServiceClient {
	return c.client
}

// NewFIFAMarketDashboardClientset creates a new instance of FIFA Market Dashboard Clientset.
func NewFIFAMarketDashboardClientset(address string) (Clientset, error) {
	connection, err := utilgrpc.NewClientConnection(address)
	if err != nil {
		return nil, err
	}
	return &clientSet{connection: connection, client: NewFIFAMarketDashboardServiceClient(connection.ClientConn())}, nil
}

func (c *clientSet) CheckHealth(ctx context.Context) (grpc_health_v1.HealthCheckResponse_ServingStatus, error) {
	return c.connection.CheckHealth(ctx)
}

func (c *clientSet) Close() error {
	return c.connection.Close()
}
