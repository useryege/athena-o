package apiclient

import (
	"context"
	"fmt"
	"math"

	"github.com/useryege/athena/common"
	"github.com/useryege/athena/util/env"
	utilgrpc "github.com/useryege/athena/util/grpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health/grpc_health_v1"
)

var MaxGRPCMessageSize = env.ParseNumFromEnv(common.EnvGRPCMaxSizeMB, 100, 0, math.MaxInt32) * 1024 * 1024

// Clientset represents authenticated Worm Trading API clients.
type Clientset interface {
	WormTrading() WormTradingServiceClient
	CheckHealth(ctx context.Context) (grpc_health_v1.HealthCheckResponse_ServingStatus, error)
	Close() error
}

type clientSet struct {
	connection *utilgrpc.ClientConnection
	client     WormTradingServiceClient
}

func (c *clientSet) WormTrading() WormTradingServiceClient {
	return c.client
}

// NewWormTradingClientset creates a client whose every business RPC carries
// the Worm Trading service's independent internal Bearer.
func NewWormTradingClientset(address, internalAuthToken string) (Clientset, error) {
	token, err := NormalizeInternalAuthToken(internalAuthToken)
	if err != nil {
		return nil, fmt.Errorf("validate Worm Trading internal authentication: %w", err)
	}
	connection, err := utilgrpc.NewClientConnection(
		address,
		grpc.WithPerRPCCredentials(internalBearerCredentials{authorization: "Bearer " + token}),
	)
	if err != nil {
		return nil, err
	}
	return &clientSet{
		connection: connection,
		client:     NewWormTradingServiceClient(connection.ClientConn()),
	}, nil
}

func (c *clientSet) CheckHealth(ctx context.Context) (grpc_health_v1.HealthCheckResponse_ServingStatus, error) {
	return c.connection.CheckHealth(ctx)
}

func (c *clientSet) Close() error {
	return c.connection.Close()
}
