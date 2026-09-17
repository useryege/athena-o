package util

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	"google.golang.org/grpc"
)

const ServiceShutdownTimeout = 30 * time.Second

// ServeGRPC owns the listener and the supplied shutdown operation. Shutdown
// must cancel workers, wait for them, and close resources owned by the command.
// It runs concurrently with transport draining under one shared budget.
func ServeGRPC(ctx context.Context, listener net.Listener, server *grpc.Server, shutdown func() error) error {
	return serveGRPC(ctx, listener, server, shutdown, ServiceShutdownTimeout)
}
func serveGRPC(ctx context.Context, listener net.Listener, server *grpc.Server, shutdown func() error, budget time.Duration) error {
	rpcDone := make(chan error, 1)
	go func() { rpcDone <- server.Serve(listener) }()
	var result error
	select {
	case <-ctx.Done():
	case result = <-rpcDone:
	}
	result = cleanServeError(result)
	// Close the listener before canceling work so no new transport can enter.
	_ = listener.Close()
	drained := make(chan struct{})
	go func() { server.GracefulStop(); close(drained) }()
	workDone := make(chan error, 1)
	go func() { workDone <- shutdown() }()
	timer := time.NewTimer(budget)
	defer timer.Stop()
	for drained != nil || workDone != nil {
		select {
		case <-drained:
			drained = nil
		case err := <-workDone:
			result = errors.Join(result, err)
			workDone = nil
		case <-timer.C:
			server.Stop()
			return errors.Join(result, fmt.Errorf("service shutdown exceeded %s", budget))
		}
	}
	return result
}
func cleanServeError(err error) error {
	if errors.Is(err, grpc.ErrServerStopped) || errors.Is(err, net.ErrClosed) {
		return nil
	}
	return err
}
