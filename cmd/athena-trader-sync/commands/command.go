package commands

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/useryege/athena/internal/tradersync"
	"github.com/useryege/athena/internal/tradersync/apiclient"
	"github.com/useryege/athena/internal/tradersync/rpcconfig"
	"github.com/useryege/athena/internal/tradersync/transport"
	"google.golang.org/grpc"
)

func NewCommand() *cobra.Command {
	cmd := &cobra.Command{Use: "athena-trader-sync", Short: "Run the independent Trader Sync service", SilenceUsage: true, SilenceErrors: true, Args: cobra.NoArgs}
	cmd.RunE = func(cmd *cobra.Command, _ []string) error {
		cfg, err := tradersync.LoadConfigFromEnv()
		if err != nil {
			return err
		}
		rpcCfg, err := rpcconfig.LoadServer(os.LookupEnv)
		if err != nil {
			return err
		}
		creds, err := rpcconfig.ServerCredentials(rpcCfg)
		if err != nil {
			return err
		}
		cfg.OnError = func(err error) {
			log.WithFields(log.Fields{"service": "trader-sync", "phase": "collector_or_source_degraded"}).WithError(err).Warn("Trader Sync source unavailable")
		}
		r, err := tradersync.NewRuntime(cfg, rpcCfg, tradersync.RuntimeDependencies{NewRPCServer: func(s *tradersync.Service, ready func() bool) *grpc.Server {
			server := grpc.NewServer(grpc.Creds(creds), grpc.MaxRecvMsgSize(rpcCfg.MaxMessageBytes), grpc.MaxSendMsgSize(rpcCfg.MaxMessageBytes), grpc.UnaryInterceptor(transport.NewUnaryInterceptor(rpcCfg.Token, ready)))
			apiclient.RegisterTraderSyncServiceServer(server, transport.NewServer(s))
			return server
		}})
		if err != nil {
			return err
		}
		ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		remaining := cfg.ShutdownTimeout
		started := make(chan error, 1)
		go func() { started <- r.Start(ctx) }()
		select {
		case err = <-started:
			if err != nil {
				return err
			}
		case <-ctx.Done():
			// Include cancellation during schema verification/recovery in the same
			// process-wide budget; no defer may free resources behind live work.
			deadline := time.Now().Add(cfg.ShutdownTimeout)
			timer := time.NewTimer(cfg.ShutdownTimeout)
			defer timer.Stop()
			select {
			case err = <-started:
				if err != nil {
					return err
				}
			case <-timer.C:
				shutdownDeadline(r.ShutdownLogFields())
			}
			remaining = time.Until(deadline)
		}
		return runLifecycle(ctx, r, remaining)
	}
	cmd.AddCommand(newHealthCommand())
	return cmd
}

type lifecycle interface {
	Wait() error
	Shutdown(context.Context) error
	ShutdownLogFields() tradersync.RuntimeLogFields
}

func runLifecycle(ctx context.Context, r lifecycle, budget time.Duration) error {
	stopped := make(chan error, 1)
	go func() { stopped <- r.Wait() }()
	var runErr error
	select {
	case runErr = <-stopped:
	case <-ctx.Done():
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), budget)
	defer cancel()
	finished := make(chan error, 1)
	go func() { finished <- r.Shutdown(shutdownCtx) }()
	select {
	case err := <-finished:
		if errors.Is(err, context.DeadlineExceeded) || shutdownCtx.Err() != nil {
			shutdownDeadline(r.ShutdownLogFields())
		}
		return errors.Join(runErr, err)
	case <-shutdownCtx.Done():
		shutdownDeadline(r.ShutdownLogFields())
	}
	return nil
}
func shutdownDeadline(fields tradersync.RuntimeLogFields) {
	fmt.Fprintf(os.Stderr, "service=trader-sync phase=shutdown instance=%q run=%q runtime_generation=%s collector_epoch=%s shutdown deadline exceeded\n", fields.Instance, fields.Run, fields.RuntimeGeneration, fields.CollectorEpoch)
	os.Exit(1)
}
