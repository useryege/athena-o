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

type Clientset interface {
	SportsHistory() SportsHistoryServiceClient
	CheckHealth(context.Context) (grpc_health_v1.HealthCheckResponse_ServingStatus, error)
	Close() error
}

type clientSet struct {
	connection *utilgrpc.ClientConnection
	client     SportsHistoryServiceClient
}

func (c *clientSet) SportsHistory() SportsHistoryServiceClient {
	return c.client
}

func NewSportsHistoryClientset(address string) (Clientset, error) {
	connection, err := utilgrpc.NewClientConnection(address)
	if err != nil {
		return nil, err
	}
	return &clientSet{connection: connection, client: NewSportsHistoryServiceClient(connection.ClientConn())}, nil
}

func (c *clientSet) CheckHealth(ctx context.Context) (grpc_health_v1.HealthCheckResponse_ServingStatus, error) {
	return c.connection.CheckHealth(ctx)
}

func (c *clientSet) Close() error {
	return c.connection.Close()
}
