package apiclient

import (
	"context"
	fmt "fmt"
	math "math"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/useryege/athena/common"
	"github.com/useryege/athena/util/env"
	utilio "github.com/useryege/athena/util/io"
	grpc "google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health/grpc_health_v1"
)

// MaxGRPCMessageSize contains max grpc message size
var MaxGRPCMessageSize = env.ParseNumFromEnv(common.EnvGRPCMaxSizeMB, 100, 0, math.MaxInt32) * 1024 * 1024

// Clientset represents project controller api clients.
type Clientset interface {
	NewApplicationServiceClient() (utilio.Closer, ApplicationServiceClient, error)
}

type clientSet struct {
	address string
}

// NewApplicationServiceClient creates a new application client.
func (c *clientSet) NewApplicationServiceClient() (utilio.Closer, ApplicationServiceClient, error) {
	conn, err := NewConnection(c.address)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open a new connection to chain watcher: %w", err)
	}
	return conn, NewApplicationServiceClient(conn), nil
}

// NewConnection creates new connection to project controller.
func NewConnection(address string) (*grpc.ClientConn, error) {
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))

	conn, err := grpc.NewClient(address, opts...)
	if err != nil {
		log.Errorf("Unable to connect to chain watcher service with address %s", address)
		return nil, err
	}
	return conn, nil
}

// NewApplicationClientset creates new instance of application Clientset.
func NewApplicationClientset(address string) Clientset {
	return &clientSet{address: address}
}

// WaitForApplicationService blocks until the application gRPC health service is serving.
func WaitForApplicationService(ctx context.Context, address string) error {
	return waitForApplicationService(ctx, address, time.Second)
}

func waitForApplicationService(ctx context.Context, address string, retryInterval time.Duration) error {
	for {
		checkCtx, cancel := context.WithTimeout(ctx, retryInterval)
		err := checkApplicationServiceHealth(checkCtx, address)
		cancel()
		if err == nil {
			return nil
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		log.WithError(err).Infof("waiting for athena application grpc service at %s", address)

		timer := time.NewTimer(retryInterval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}

func checkApplicationServiceHealth(ctx context.Context, address string) error {
	conn, err := NewConnection(address)
	if err != nil {
		return err
	}
	defer utilio.Close(conn)

	client := grpc_health_v1.NewHealthClient(conn)
	resp, err := client.Check(ctx, &grpc_health_v1.HealthCheckRequest{})
	if err != nil {
		return err
	}
	if resp.GetStatus() != grpc_health_v1.HealthCheckResponse_SERVING {
		return fmt.Errorf("application grpc health status is %s", resp.GetStatus())
	}
	return nil
}
