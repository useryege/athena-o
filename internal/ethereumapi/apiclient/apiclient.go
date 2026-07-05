package apiclient

import (
	"context"
	"fmt"
	"math"

	log "github.com/sirupsen/logrus"
	"github.com/useryege/athena/common"
	"github.com/useryege/athena/util/env"
	utilio "github.com/useryege/athena/util/io"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health/grpc_health_v1"
)

var MaxGRPCMessageSize = env.ParseNumFromEnv(common.EnvGRPCMaxSizeMB, 100, 0, math.MaxInt32) * 1024 * 1024

// Clientset represents ethereum-api clients.
type Clientset interface {
	NewEthereumAPIServiceClient() (utilio.Closer, EthereumAPIServiceClient, error)
	CheckHealth(ctx context.Context) (grpc_health_v1.HealthCheckResponse_ServingStatus, error)
}

type clientSet struct {
	address string
}

// NewEthereumAPIServiceClient creates a new ethereum-api client.
func (c *clientSet) NewEthereumAPIServiceClient() (utilio.Closer, EthereumAPIServiceClient, error) {
	conn, err := NewConnection(c.address)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open a new connection to ethereum-api service: %w", err)
	}
	return conn, NewEthereumAPIServiceClient(conn), nil
}

// NewEthereumAPIClientset creates a new ethereum-api Clientset.
func NewEthereumAPIClientset(address string) Clientset {
	return &clientSet{address: address}
}

func NewConnection(address string) (*grpc.ClientConn, error) {
	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Errorf("Unable to connect to ethereum-api service with address %s", address)
		return nil, err
	}
	return conn, nil
}

func (c *clientSet) CheckHealth(ctx context.Context) (grpc_health_v1.HealthCheckResponse_ServingStatus, error) {
	conn, err := NewConnection(c.address)
	if err != nil {
		return grpc_health_v1.HealthCheckResponse_UNKNOWN, err
	}
	defer utilio.Close(conn)

	client := grpc_health_v1.NewHealthClient(conn)
	resp, err := client.Check(ctx, &grpc_health_v1.HealthCheckRequest{})
	if err != nil {
		return grpc_health_v1.HealthCheckResponse_UNKNOWN, err
	}
	return resp.GetStatus(), nil
}
