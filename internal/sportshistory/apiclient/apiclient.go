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

type Clientset interface {
	NewSportsHistoryServiceClient() (utilio.Closer, SportsHistoryServiceClient, error)
	CheckHealth(context.Context) (grpc_health_v1.HealthCheckResponse_ServingStatus, error)
}

type clientSet struct {
	address string
}

func (c *clientSet) NewSportsHistoryServiceClient() (utilio.Closer, SportsHistoryServiceClient, error) {
	conn, err := NewConnection(c.address)
	if err != nil {
		return nil, nil, fmt.Errorf("connect to sports history service: %w", err)
	}
	return conn, NewSportsHistoryServiceClient(conn), nil
}

func NewSportsHistoryClientset(address string) Clientset {
	return &clientSet{address: address}
}

func NewConnection(address string) (*grpc.ClientConn, error) {
	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.WithError(err).WithField("address", address).Error("connect to sports history service")
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
	response, err := grpc_health_v1.NewHealthClient(conn).Check(ctx, &grpc_health_v1.HealthCheckRequest{})
	if err != nil {
		return grpc_health_v1.HealthCheckResponse_UNKNOWN, err
	}
	return response.GetStatus(), nil
}
