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
	"github.com/useryege/athena/internal/bscswap"
	bscswapstore "github.com/useryege/athena/internal/bscswap/store"
	"github.com/useryege/athena/util/cli"
	"github.com/useryege/athena/util/env"
	"google.golang.org/grpc"
)

const cliName = "athena-bsc-swap-indexer"

func NewCommand() *cobra.Command {
	var (
		nodeRPCURL       string
		grpcListen       string
		telemetryListen  string
		fetchConcurrency int
		pollInterval     time.Duration
	)
	command := &cobra.Command{
		Use: cliName, Short: "Index and query BSC transactions containing V2 Swap topic logs", DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cli.SetLogFormat(cmdutil.LogFormat)
			cli.SetLogLevel(cmdutil.LogLevel)
			if strings.TrimSpace(nodeRPCURL) == "" {
				return fmt.Errorf("ATHENA_BSC_SWAP_NODE_RPC_URL is required")
			}
			ctx, stop := signal.NotifyContext(cmd.Context(), syscall.SIGINT, syscall.SIGTERM)
			defer stop()

			repository, err := bscswapstore.Open(ctx)
			if err != nil {
				return err
			}
			defer repository.Close()
			node, err := bscswap.DialNode(ctx, nodeRPCURL)
			if err != nil {
				return err
			}
			defer node.Close()

			metrics := bscswap.NewMetrics()
			scanner, err := bscswap.NewScanner(repository, node, metrics, bscswap.ScannerConfig{
				FetchConcurrency: fetchConcurrency, PollInterval: pollInterval,
			})
			if err != nil {
				return err
			}
			service := bscswap.NewService(repository, metrics)
			grpcServer := bscswap.NewGRPCServer(service)
			listener, err := net.Listen("tcp", grpcListen)
			if err != nil {
				return fmt.Errorf("listen BSC swap gRPC on %s: %w", grpcListen, err)
			}
			defer listener.Close()
			telemetry := bscswap.NewTelemetryServer(telemetryListen, metrics, repository.Ping)
			if err := telemetry.Start(); err != nil {
				return err
			}

			metrics.SetRunning(true)
			grpcServer.SetServing(true)
			go scanner.Run(ctx)
			serveError := make(chan error, 1)
			go func() { serveError <- grpcServer.Server().Serve(listener) }()

			common.GetVersion().LogStartupInfo("Athena BSC Swap Indexer", map[string]any{
				"grpc_listen": grpcListen, "telemetry_listen": telemetryListen,
				"scan_batch_size": bscswap.ScanBatchSize, "fetch_concurrency": fetchConcurrency,
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
			log.Info("BSC swap indexer stopped")
			return result
		},
	}
	command.Flags().StringVar(&cmdutil.LogFormat, "logformat", env.StringFromEnv(common.EnvLogFormat, "json"), "Set the logging format. One of: json|text")
	command.Flags().StringVar(&cmdutil.LogLevel, "loglevel", env.StringFromEnv(common.EnvLogLevel, "info"), "Set the logging level. One of: debug|info|warn|error")
	command.Flags().StringVar(&nodeRPCURL, "node-rpc-url", env.StringFromEnv("ATHENA_BSC_SWAP_NODE_RPC_URL", ""), "Private BSC Mainnet JSON-RPC URL")
	command.Flags().StringVar(&grpcListen, "grpc-listen-address", env.StringFromEnv("ATHENA_BSC_SWAP_GRPC_LISTEN_ADDRESS", "127.0.0.1:8130"), "gRPC listen address")
	command.Flags().StringVar(&telemetryListen, "telemetry-listen-address", env.StringFromEnv("ATHENA_BSC_SWAP_TELEMETRY_LISTEN_ADDRESS", "127.0.0.1:8131"), "Health and metrics listen address")
	command.Flags().IntVar(&fetchConcurrency, "fetch-concurrency", env.ParseNumFromEnv("ATHENA_BSC_SWAP_FETCH_CONCURRENCY", bscswap.DefaultFetchConcurrency, 1, 512), "Maximum concurrent BSC block requests")
	command.Flags().DurationVar(&pollInterval, "poll-interval", env.ParseDurationFromEnv("ATHENA_BSC_SWAP_POLL_INTERVAL", bscswap.DefaultPollInterval, 100*time.Millisecond, time.Minute), "Delay between caught-up or failed scan iterations")
	command.AddCommand(cli.NewVersionCmd(cliName))
	return command
}
