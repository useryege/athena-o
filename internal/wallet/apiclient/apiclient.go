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

// Clientset represents wallet api clients.
type Clientset interface {
	Wallet() WalletServiceClient
	CheckHealth(ctx context.Context) (grpc_health_v1.HealthCheckResponse_ServingStatus, error)
	Close() error
}

// WormExecutionSignerClientset exposes only the capability-scoped signer
// service and health. It must be constructed with the signer-specific token,
// never the WalletService credential.
type WormExecutionSignerClientset interface {
	Signer() WormExecutionSignerServiceClient
	CheckHealth(ctx context.Context) (grpc_health_v1.HealthCheckResponse_ServingStatus, error)
	Close() error
}

type clientSet struct {
	connection *utilgrpc.ClientConnection
	client     WalletServiceClient
}

type wormExecutionSignerClientSet struct {
	connection *utilgrpc.ClientConnection
	client     WormExecutionSignerServiceClient
}

// NewWalletServiceClient creates a new wallet client.
func (c *clientSet) Wallet() WalletServiceClient {
	return c.client
}

// NewWalletClientset creates a Wallet client whose every RPC carries the
// required internal service bearer credential.
func NewWalletClientset(address, internalAuthToken string) (Clientset, error) {
	token, err := NormalizeInternalAuthToken(internalAuthToken)
	if err != nil {
		return nil, fmt.Errorf("validate Wallet internal authentication: %w", err)
	}
	connection, err := utilgrpc.NewClientConnection(
		address,
		grpc.WithPerRPCCredentials(internalBearerCredentials{authorization: "Bearer " + token}),
	)
	if err != nil {
		return nil, err
	}
	return &clientSet{connection: connection, client: NewWalletServiceClient(connection.ClientConn())}, nil
}

func NewWormExecutionSignerClientset(address, signerAuthToken string) (WormExecutionSignerClientset, error) {
	token, err := NormalizeWormExecutionSignerAuthToken(signerAuthToken)
	if err != nil {
		return nil, fmt.Errorf("validate Wallet Worm execution signer authentication: %w", err)
	}
	connection, err := utilgrpc.NewClientConnection(
		address,
		grpc.WithPerRPCCredentials(internalBearerCredentials{authorization: "Bearer " + token}),
	)
	if err != nil {
		return nil, err
	}
	return &wormExecutionSignerClientSet{
		connection: connection,
		client:     NewWormExecutionSignerServiceClient(connection.ClientConn()),
	}, nil
}

func (c *clientSet) CheckHealth(ctx context.Context) (grpc_health_v1.HealthCheckResponse_ServingStatus, error) {
	return c.connection.CheckHealth(ctx)
}

func (c *clientSet) Close() error {
	return c.connection.Close()
}

func (c *wormExecutionSignerClientSet) Signer() WormExecutionSignerServiceClient {
	return c.client
}

func (c *wormExecutionSignerClientSet) CheckHealth(ctx context.Context) (grpc_health_v1.HealthCheckResponse_ServingStatus, error) {
	return c.connection.CheckHealth(ctx)
}

func (c *wormExecutionSignerClientSet) Close() error {
	return c.connection.Close()
}
