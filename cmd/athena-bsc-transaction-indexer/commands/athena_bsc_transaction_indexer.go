package commands

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os/signal"
	"strings"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	cmdutil "github.com/useryege/athena/cmd/util"
	"github.com/useryege/athena/common"
	"github.com/useryege/athena/internal/bscinbound"
	bscstore "github.com/useryege/athena/internal/bscinbound/store"
	"github.com/useryege/athena/util/cli"
	"github.com/useryege/athena/util/env"
	"google.golang.org/grpc"
)

const cliName = "athena-bsc-transaction-indexer"

func NewCommand() *cobra.Command {
	var (
		nodeRPCURL       string
		grpcListen       string
		telemetryListen  string
		batchSize        int
		fetchConcurrency int
		pollInterval     time.Duration
	)
	command := &cobra.Command{
		Use: cliName, Short: "Index and query finalized inbound BSC transactions", DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cli.SetLogFormat(cmdutil.LogFormat)
			cli.SetLogLevel(cmdutil.LogLevel)
			if strings.TrimSpace(nodeRPCURL) == "" {
				return fmt.Errorf("ATHENA_BSC_INBOUND_NODE_RPC_URL is required")
			}
			ctx, stop := signal.NotifyContext(cmd.Context(), syscall.SIGINT, syscall.SIGTERM)
			defer stop()

			repository, err := bscstore.Open(ctx)
			if err != nil {
				return err
			}
			defer repository.Close()
			node, err := bscinbound.DialNode(ctx, nodeRPCURL)
			if err != nil {
				return err
			}
			defer node.Close()

			metrics := bscinbound.NewMetrics()
			scanner, err := bscinbound.NewScanner(repository, node, metrics, bscinbound.ScannerConfig{
				BatchSize: batchSize, FetchConcurrency: fetchConcurrency, PollInterval: pollInterval,
			})
			if err != nil {
				return err
			}
			service := bscinbound.NewService(repository, metrics)
			grpcServer := bscinbound.NewGRPCServer(service)
			listener, err := net.Listen("tcp", grpcListen)
			if err != nil {
				return fmt.Errorf("listen BSC inbound gRPC on %s: %w", grpcListen, err)
			}
			defer listener.Close()
			telemetry := bscinbound.NewTelemetryServer(telemetryListen, metrics, repository.Ping)
			if err := telemetry.Start(); err != nil {
				return err
			}

			metrics.SetRunning(true)
			grpcServer.SetServing(true)
			go scanner.Run(ctx)
			serveError := make(chan error, 1)
			go func() { serveError <- grpcServer.Server().Serve(listener) }()

			common.GetVersion().LogStartupInfo("Athena BSC Transaction Indexer", map[string]any{
				"grpc_listen": grpcListen, "telemetry_listen": telemetryListen,
				"batch_size": batchSize, "fetch_concurrency": fetchConcurrency,
			})

			var result error
			select {
			case <-ctx.Done():
			case err := <-serveError:
				if err != nil && !errors.Is(err, grpc.ErrServerStopped) {
					result = err
				}
				stop()
			}

			metrics.SetRunning(false)
			grpcServer.SetServing(false)
			gracefulDone := make(chan struct{})
			go func() {
				grpcServer.Server().GracefulStop()
				close(gracefulDone)
			}()
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			select {
			case <-gracefulDone:
			case <-shutdownCtx.Done():
				grpcServer.Server().Stop()
				if result == nil {
					result = shutdownCtx.Err()
				}
			}
			if err := telemetry.Stop(shutdownCtx); err != nil && result == nil {
				result = err
			}
			log.Info("BSC transaction indexer stopped")
			return result
		},
	}
	command.Flags().StringVar(&cmdutil.LogFormat, "logformat", env.StringFromEnv(common.EnvLogFormat, "json"), "Set the logging format. One of: json|text")
	command.Flags().StringVar(&cmdutil.LogLevel, "loglevel", env.StringFromEnv(common.EnvLogLevel, "info"), "Set the logging level. One of: debug|info|warn|error")
	command.Flags().StringVar(&nodeRPCURL, "node-rpc-url", env.StringFromEnv("ATHENA_BSC_INBOUND_NODE_RPC_URL", ""), "Private BSC Mainnet JSON-RPC URL")
	command.Flags().StringVar(&grpcListen, "grpc-listen-address", env.StringFromEnv("ATHENA_BSC_INBOUND_GRPC_LISTEN_ADDRESS", "127.0.0.1:8130"), "gRPC listen address")
	command.Flags().StringVar(&telemetryListen, "telemetry-listen-address", env.StringFromEnv("ATHENA_BSC_INBOUND_TELEMETRY_LISTEN_ADDRESS", "127.0.0.1:8131"), "Health and metrics listen address")
	command.Flags().IntVar(&batchSize, "scan-batch-size", env.ParseNumFromEnv("ATHENA_BSC_INBOUND_SCAN_BATCH_SIZE", bscinbound.DefaultScanBatchSize, 1, 10000), "Maximum finalized blocks committed per batch")
	command.Flags().IntVar(&fetchConcurrency, "fetch-concurrency", env.ParseNumFromEnv("ATHENA_BSC_INBOUND_FETCH_CONCURRENCY", bscinbound.DefaultFetchConcurrency, 1, 512), "Maximum concurrent BSC RPC requests")
	command.Flags().DurationVar(&pollInterval, "poll-interval", env.ParseDurationFromEnv("ATHENA_BSC_INBOUND_POLL_INTERVAL", bscinbound.DefaultPollInterval, 100*time.Millisecond, time.Minute), "Delay between caught-up or failed scan iterations")
	command.AddCommand(cli.NewVersionCmd(cliName))
	return command
}
