package runtime

import (
	"context"
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"
	appoutbox "github.com/useryege/athena/internal/application/outbox"
	appworkflows "github.com/useryege/athena/internal/application/workflows"
)

func runOutboxWorkerMode(ctx context.Context, opts Options) error {
	if opts.Store == nil {
		return fmt.Errorf("application store is required for outbox-worker mode")
	}
	temporalClient, err := dialTemporalWithRetry(ctx, opts, ModeOutboxWorker)
	if err != nil {
		return err
	}
	defer temporalClient.Close()
	dispatcher, err := appoutbox.NewDispatcher(appoutbox.Options{
		Store:        opts.Store,
		Starter:      appworkflows.NewTemporalStarter(temporalClient),
		LockedBy:     fmt.Sprintf("application-outbox-%d", os.Getpid()),
		PollInterval: opts.OutboxPollInterval,
	})
	if err != nil {
		return err
	}
	if err := dispatcher.Start(ctx); err != nil {
		dispatcher.Stop()
		return err
	}
	log.Info("athena-application outbox worker started")
	defer dispatcher.Stop()
	return waitForShutdown(ctx)
}
