package runtime

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"syscall"

	log "github.com/sirupsen/logrus"
)

func runUntilShutdown(ctx context.Context, run func(context.Context) error) error {
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	errCh := make(chan error, 1)
	go func() {
		errCh <- run(runCtx)
	}()
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigCh)
	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		cancel()
		return ignoreContextCanceled(<-errCh)
	case sig := <-sigCh:
		log.Printf("got signal %v, attempting graceful shutdown", sig)
		cancel()
		return ignoreContextCanceled(<-errCh)
	}
}

func ignoreContextCanceled(err error) error {
	if errors.Is(err, context.Canceled) {
		return nil
	}
	return err
}

func waitForShutdown(ctx context.Context) error {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigCh)
	select {
	case <-ctx.Done():
		return ctx.Err()
	case sig := <-sigCh:
		log.Printf("got signal %v, attempting graceful shutdown", sig)
		return nil
	}
}
