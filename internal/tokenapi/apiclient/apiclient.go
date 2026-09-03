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
	Catalog() TokenCatalogServiceClient
	Collection() TokenCollectionServiceClient
	Policy() TokenPolicyServiceClient
	Operations() TokenOperationsServiceClient
	CheckHealth(context.Context) (grpc_health_v1.HealthCheckResponse_ServingStatus, error)
	Close() error
}

type clientSet struct {
	connection *utilgrpc.ClientConnection
	catalog    TokenCatalogServiceClient
	collection TokenCollectionServiceClient
	policy     TokenPolicyServiceClient
	operations TokenOperationsServiceClient
}

func NewTokenAPIClientset(address string) (Clientset, error) {
	connection, err := utilgrpc.NewClientConnection(address)
	if err != nil {
		return nil, err
	}
	conn := connection.ClientConn()
	return &clientSet{
		connection: connection,
		catalog:    NewTokenCatalogServiceClient(conn),
		collection: NewTokenCollectionServiceClient(conn),
		policy:     NewTokenPolicyServiceClient(conn),
		operations: NewTokenOperationsServiceClient(conn),
	}, nil
}

func (c *clientSet) Catalog() TokenCatalogServiceClient {
	return c.catalog
}

func (c *clientSet) Collection() TokenCollectionServiceClient {
	return c.collection
}

func (c *clientSet) Policy() TokenPolicyServiceClient {
	return c.policy
}

func (c *clientSet) Operations() TokenOperationsServiceClient {
	return c.operations
}

func (c *clientSet) CheckHealth(ctx context.Context) (grpc_health_v1.HealthCheckResponse_ServingStatus, error) {
	return c.connection.CheckHealth(ctx)
}

func (c *clientSet) Close() error {
	return c.connection.Close()
}
