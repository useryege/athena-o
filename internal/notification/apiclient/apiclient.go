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

// Clientset represents notification api clients.
type Clientset interface {
	NewNotificationServiceClient() (utilio.Closer, NotificationServiceClient, error)
	CheckHealth(ctx context.Context) (grpc_health_v1.HealthCheckResponse_ServingStatus, error)
}

type clientSet struct {
	address string
}

// NewNotificationServiceClient creates a new notification client.
func (c *clientSet) NewNotificationServiceClient() (utilio.Closer, NotificationServiceClient, error) {
	conn, err := NewConnection(c.address)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open a new connection to notification service: %w", err)
	}
	return conn, NewNotificationServiceClient(conn), nil
}

// NewNotificationClientset creates a new instance of notification Clientset.
func NewNotificationClientset(address string) Clientset {
	return &clientSet{address: address}
}

func NewConnection(address string) (*grpc.ClientConn, error) {
	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Errorf("Unable to connect to notification service with address %s", address)
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
