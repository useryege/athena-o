package grpc

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health/grpc_health_v1"
)

// ClientConnection owns one reusable plaintext gRPC channel for an internal
// service dependency. ClientConn is safe for concurrent RPCs and reconnects
// automatically when the remote process restarts.
type ClientConnection struct {
	conn      *grpc.ClientConn
	closeOnce sync.Once
	closeErr  error
}

// NewClientConnection creates an internal plaintext gRPC channel and starts
// connecting without blocking service startup on dependency availability.
func NewClientConnection(target string, opts ...grpc.DialOption) (*ClientConnection, error) {
	target = strings.TrimSpace(target)
	if target == "" {
		return nil, fmt.Errorf("gRPC target is required")
	}

	dialOpts := make([]grpc.DialOption, 0, len(opts)+1)
	dialOpts = append(dialOpts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	dialOpts = append(dialOpts, opts...)
	conn, err := grpc.NewClient(target, dialOpts...)
	if err != nil {
		return nil, fmt.Errorf("create gRPC client for %s: %w", target, err)
	}
	conn.Connect()
	return &ClientConnection{conn: conn}, nil
}

// ClientConn returns the shared channel used to construct generated clients.
func (c *ClientConnection) ClientConn() *grpc.ClientConn {
	if c == nil {
		return nil
	}
	return c.conn
}

// CheckHealth queries the dependency's standard gRPC health service through
// the shared channel.
func (c *ClientConnection) CheckHealth(ctx context.Context) (grpc_health_v1.HealthCheckResponse_ServingStatus, error) {
	if c == nil || c.conn == nil {
		return grpc_health_v1.HealthCheckResponse_UNKNOWN, fmt.Errorf("gRPC client connection is unavailable")
	}
	response, err := grpc_health_v1.NewHealthClient(c.conn).Check(ctx, &grpc_health_v1.HealthCheckRequest{})
	if err != nil {
		return grpc_health_v1.HealthCheckResponse_UNKNOWN, err
	}
	return response.GetStatus(), nil
}

// Close releases the shared channel once. Callers must stop request handlers
// and background loops before closing their owned clientsets.
func (c *ClientConnection) Close() error {
	if c == nil {
		return nil
	}
	c.closeOnce.Do(func() {
		if c.conn != nil {
			c.closeErr = c.conn.Close()
		}
	})
	return c.closeErr
}
