package e2etest

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health/grpc_health_v1"
)

const (
	defaultHealthCheckTimeout = 3 * time.Second
	defaultHealthPollInterval = time.Second
)

func NewInsecureGRPCConn(t testing.TB, addr string, callOptions ...grpc.CallOption) *grpc.ClientConn {
	t.Helper()

	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	if len(callOptions) > 0 {
		opts = append(opts, grpc.WithDefaultCallOptions(callOptions...))
	}
	conn, err := grpc.NewClient(addr, opts...)
	if err != nil {
		t.Fatalf("create grpc client for %s: %v", addr, err)
	}
	t.Cleanup(func() {
		_ = conn.Close()
	})
	return conn
}

func WaitForGRPCServing(ctx context.Context, addr string, failureHint string) error {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("create grpc health client for %s: %w", addr, err)
	}
	defer conn.Close()

	client := grpc_health_v1.NewHealthClient(conn)
	ticker := time.NewTicker(defaultHealthPollInterval)
	defer ticker.Stop()

	lastStatus := grpc_health_v1.HealthCheckResponse_UNKNOWN
	var lastErr error
	for {
		checkCtx, cancel := context.WithTimeout(ctx, defaultHealthCheckTimeout)
		resp, err := client.Check(checkCtx, &grpc_health_v1.HealthCheckRequest{})
		cancel()
		if err == nil {
			lastStatus = resp.GetStatus()
			if lastStatus == grpc_health_v1.HealthCheckResponse_SERVING {
				return nil
			}
			lastErr = nil
		} else {
			lastErr = err
		}

		select {
		case <-ctx.Done():
			message := fmt.Sprintf(
				"grpc service did not become SERVING at %s before timeout: last_status=%s last_error=%v",
				addr,
				lastStatus.String(),
				lastErr,
			)
			if strings.TrimSpace(failureHint) != "" {
				message = fmt.Sprintf("%s; %s", message, failureHint)
			}
			return fmt.Errorf("%s", message)
		case <-ticker.C:
		}
	}
}
