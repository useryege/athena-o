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

// ServeGRPC owns the listener and the supplied shutdown operations. Background
// cancellation/join runs concurrently with RPC draining. Shared resources are
// released only after both finish, under the same total shutdown budget.
func ServeGRPC(ctx context.Context, listener net.Listener, server *grpc.Server, stopBackground, releaseResources func() error) error {
	return serveGRPC(ctx, listener, server, stopBackground, releaseResources, ServiceShutdownTimeout)
}
func serveGRPC(ctx context.Context, listener net.Listener, server *grpc.Server, stopBackground, releaseResources func() error, budget time.Duration) error {
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
	go func() { workDone <- stopBackground() }()
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

	// The timer is deliberately not reset: resource release consumes only the
	// budget remaining after accepted RPCs and background work have completed.
	releaseDone := make(chan error, 1)
	go func() { releaseDone <- releaseResources() }()
	select {
	case err := <-releaseDone:
		return errors.Join(result, err)
	case <-timer.C:
		server.Stop()
		return errors.Join(result, fmt.Errorf("service shutdown exceeded %s while releasing resources", budget))
	}
}
func cleanServeError(err error) error {
	if errors.Is(err, grpc.ErrServerStopped) || errors.Is(err, net.ErrClosed) {
		return nil
	}
	return err
}
