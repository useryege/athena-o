package commands

import (
	"context"
	"errors"
	"fmt"
	"google.golang.org/grpc"
	"net"
	"time"
)

// serve owns both loops; cancellation reaches node calls before draining RPC.
func serve(ctx context.Context, listener net.Listener, server *grpc.Server, scan func(context.Context) error) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	defer listener.Close()
	scanDone := make(chan error, 1)
	rpcDone := make(chan error, 1)
	go func() { scanDone <- scan(ctx) }()
	go func() { rpcDone <- server.Serve(listener) }()
	var result error
	scannerExited := false
	select {
	case <-ctx.Done():
	case result = <-scanDone:
		scannerExited = true
	case result = <-rpcDone:
	}
	cancel()
	drained := make(chan struct{})
	go func() { server.GracefulStop(); close(drained) }()
	select {
	case <-drained:
	case <-time.After(5 * time.Second):
		server.Stop()
		<-drained
	}
	if !scannerExited {
		select {
		case err := <-scanDone:
			if result == nil {
				result = err
			}
		case <-time.After(5 * time.Second):
			return fmt.Errorf("Solana scanner did not stop within 5 seconds")
		}
	}
	if errors.Is(result, context.Canceled) || errors.Is(result, grpc.ErrServerStopped) {
		return nil
	}
	return result
}
