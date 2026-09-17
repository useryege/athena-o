package commands

import (
	"context"
	"errors"
	"net"

	cmdutil "github.com/useryege/athena/cmd/util"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
)

// serve owns both loops; cancellation reaches node calls before draining RPC.
func serve(ctx context.Context, listener net.Listener, server *grpc.Server, scan func(context.Context) error) error {
	return serveWithCleanup(ctx, listener, server, scan, func() {})
}

func serveWithCleanup(ctx context.Context, listener net.Listener, server *grpc.Server, scan func(context.Context) error, cleanup func()) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	scanDone := make(chan error, 1)
	go func() { scanDone <- scan(ctx); cancel() }()
	return cmdutil.ServeGRPC(ctx, listener, server, func() error {
		cancel()
		err := <-scanDone
		cleanup()
		if errors.Is(err, context.Canceled) {
			return nil
		}
		return err
	})
}

// runDiscovery shares shutdown across the live scanner and persisted enrichment queue.
func runDiscovery(ctx context.Context, scan, enrich func(context.Context) error) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	group, ctx := errgroup.WithContext(ctx)
	for _, run := range []func(context.Context) error{scan, enrich} {
		run := run
		group.Go(func() error { defer cancel(); return run(ctx) })
	}
	return group.Wait()
}
