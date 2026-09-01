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

type Clientset interface {
	System() SystemNotificationServiceClient
	Account() AccountNotificationServiceClient
	Runtime() NotificationRuntimeServiceClient
	CheckHealth(ctx context.Context) (grpc_health_v1.HealthCheckResponse_ServingStatus, error)
	Close() error
}

type clientSet struct {
	connection *utilgrpc.ClientConnection
	system     SystemNotificationServiceClient
	account    AccountNotificationServiceClient
	runtime    NotificationRuntimeServiceClient
}

func (c *clientSet) System() SystemNotificationServiceClient {
	return c.system
}

func (c *clientSet) Account() AccountNotificationServiceClient {
	return c.account
}

func (c *clientSet) Runtime() NotificationRuntimeServiceClient {
	return c.runtime
}

func NewNotificationClientset(address, internalAuthToken string) (Clientset, error) {
	token, err := NormalizeInternalAuthToken(internalAuthToken)
	if err != nil {
		return nil, fmt.Errorf("validate notification internal authentication: %w", err)
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
		system:     NewSystemNotificationServiceClient(connection.ClientConn()),
		account:    NewAccountNotificationServiceClient(connection.ClientConn()),
		runtime:    NewNotificationRuntimeServiceClient(connection.ClientConn()),
	}, nil
}

func (c *clientSet) CheckHealth(ctx context.Context) (grpc_health_v1.HealthCheckResponse_ServingStatus, error) {
	return c.connection.CheckHealth(ctx)
}

func (c *clientSet) Close() error {
	return c.connection.Close()
}
