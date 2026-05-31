package apiclient

import (
	"context"
	"fmt"
	"math"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/useryege/athena/common"
	"github.com/useryege/athena/util/env"
	utilio "github.com/useryege/athena/util/io"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health/grpc_health_v1"
)

var MaxGRPCMessageSize = env.ParseNumFromEnv(common.EnvGRPCMaxSizeMB, 100, 0, math.MaxInt32) * 1024 * 1024

// Clientset represents polymarket api clients.
type Clientset interface {
	NewPolymarketServiceClient() (utilio.Closer, PolymarketServiceClient, error)
}

type clientSet struct {
	address string
}

// NewPolymarketServiceClient creates a new polymarket client.
func (c *clientSet) NewPolymarketServiceClient() (utilio.Closer, PolymarketServiceClient, error) {
	conn, err := NewConnection(c.address)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open a new connection to polymarket service: %w", err)
	}
	return conn, NewPolymarketServiceClient(conn), nil
}

// NewPolymarketClientset creates a new instance of polymarket Clientset.
func NewPolymarketClientset(address string) Clientset {
	return &clientSet{address: address}
}

func NewConnection(address string) (*grpc.ClientConn, error) {
	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Errorf("Unable to connect to polymarket service with address %s", address)
		return nil, err
	}
	return conn, nil
}

func WaitForPolymarketService(ctx context.Context, address string) error {
	return waitForPolymarketService(ctx, address, time.Second)
}

func waitForPolymarketService(ctx context.Context, address string, retryInterval time.Duration) error {
	for {
		checkCtx, cancel := context.WithTimeout(ctx, retryInterval)
		err := checkPolymarketServiceHealth(checkCtx, address)
		cancel()
		if err == nil {
			return nil
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		log.WithError(err).Infof("waiting for athena polymarket grpc service at %s", address)

		timer := time.NewTimer(retryInterval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}

func checkPolymarketServiceHealth(ctx context.Context, address string) error {
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
		return fmt.Errorf("polymarket grpc health status is %s", resp.GetStatus())
	}
	return nil
}
