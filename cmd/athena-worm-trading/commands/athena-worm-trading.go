package commands

import (
	"errors"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"

	cmdutil "github.com/useryege/athena/cmd/util"
	"github.com/useryege/athena/common"
	"github.com/useryege/athena/internal/wormtrading"
	wormtradingapiclient "github.com/useryege/athena/internal/wormtrading/apiclient"
	"github.com/useryege/athena/util/cli"
	"github.com/useryege/athena/util/env"
)

const (
	cliName              = "athena-worm-trading"
	solanaRPCEndpointEnv = "ATHENA_WORM_TRADING_SOLANA_RPC_URL"
	rpcAttemptTimeoutEnv = "ATHENA_WORM_TRADING_RPC_ATTEMPT_TIMEOUT"
	balanceBudgetEnv     = "ATHENA_WORM_TRADING_BALANCE_BUDGET"
	rpcRateLimitEnv      = "ATHENA_WORM_TRADING_RPC_RATE_LIMIT"
	rpcRateBurstEnv      = "ATHENA_WORM_TRADING_RPC_RATE_BURST"
)

func NewCommand() *cobra.Command {
	var (
		listenHost           string
		listenPort           int
		solanaRPCURL         string
		rpcAttemptTimeoutRaw string
		balanceBudgetRaw     string
		rpcRateLimitRaw      string
		rpcRateBurstRaw      string
	)

	command := &cobra.Command{
		Use:   cliName,
		Short: "Run the Athena Worm Trading service",
		Long: "Worm Trading exposes trusted, read-only mainnet Solana wallet balances. " +
			"It never loads Wallet storage or private keys. Every non-health RPC requires an independent internal Bearer of at least 32 bytes.",
		DisableAutoGenTag: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cli.SetLogFormat(cmdutil.LogFormat)
			cli.SetLogLevel(cmdutil.LogLevel)
			common.GetVersion().LogStartupInfo("Athena Worm Trading", map[string]any{
				"network": wormtrading.SolanaNetwork,
				"port":    listenPort,
			})

			rpcAttemptTimeout, err := parseDurationSetting("RPC attempt timeout", rpcAttemptTimeoutRaw)
			if err != nil {
				return err
			}
			balanceBudget, err := parseDurationSetting("balance budget", balanceBudgetRaw)
			if err != nil {
				return err
			}
			rpcRateLimit, err := parseFloatSetting("RPC rate limit", rpcRateLimitRaw)
			if err != nil {
				return err
			}
			rpcRateBurst, err := parseIntSetting("RPC rate burst", rpcRateBurstRaw)
			if err != nil {
				return err
			}
			adapter, err := wormtrading.NewSolanaBalanceAdapter(wormtrading.SolanaBalanceAdapterConfig{
				RPCURL:            solanaRPCURL,
				RPCAttemptTimeout: rpcAttemptTimeout,
				BalanceBudget:     balanceBudget,
				RPCRateLimit:      rpcRateLimit,
				RPCRateBurst:      rpcRateBurst,
			})
			if err != nil {
				return fmt.Errorf("configure Solana balance adapter: %w", err)
			}
			server, err := wormtrading.NewServer(wormtrading.ServerOpts{
				BalanceAdapter:    adapter,
				InternalAuthToken: env.StringFromEnv(wormtradingapiclient.InternalAuthTokenEnv, ""),
			})
			if err != nil {
				return fmt.Errorf("configure Worm Trading server: %w", err)
			}

			listener, err := (&net.ListenConfig{}).Listen(
				cmd.Context(),
				"tcp",
				fmt.Sprintf("%s:%d", listenHost, listenPort),
			)
			if err != nil {
				return fmt.Errorf("listen for Worm Trading gRPC: %w", err)
			}
			defer listener.Close()
			if err := server.Start(); err != nil {
				return err
			}

			grpcServer := server.CreateGRPC()
			signalContext, stopSignals := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
			defer stopSignals()
			serveErrors := make(chan error, 1)
			go func() {
				log.Info("starting Worm Trading gRPC server")
				serveErrors <- grpcServer.Serve(listener)
			}()

			select {
			case <-signalContext.Done():
				grpcServer.GracefulStop()
				err = <-serveErrors
			case err = <-serveErrors:
			}
			stopErr := server.Stop()
			if err != nil && !errors.Is(err, grpc.ErrServerStopped) {
				return fmt.Errorf("serve Worm Trading gRPC: %w", err)
			}
			if stopErr != nil {
				return stopErr
			}
			log.Info("Worm Trading service stopped cleanly")
			return nil
		},
		Example: "Start the Athena Worm Trading service:\n  athena-worm-trading",
	}

	command.Flags().StringVar(
		&cmdutil.LogFormat,
		"logformat",
		env.StringFromEnv(common.EnvLogFormat, "json"),
		"Set logging format",
	)
	command.Flags().StringVar(
		&cmdutil.LogLevel,
		"loglevel",
		env.StringFromEnv(common.EnvLogLevel, "info"),
		"Set logging level",
	)
	command.Flags().StringVar(
		&listenHost,
		"address",
		env.StringFromEnv("ATHENA_WORM_TRADING_LISTEN_ADDRESS", common.DefaultLocalGRPCHost),
		"Listen address",
	)
	command.Flags().IntVar(
		&listenPort,
		"port",
		env.ParseNumFromEnv("ATHENA_WORM_TRADING_PORT", common.DefaultPortWormTrading, 1, 65535),
		"Listen port",
	)
	command.Flags().StringVar(
		&solanaRPCURL,
		"solana-rpc-url",
		env.StringFromEnv(solanaRPCEndpointEnv, wormtrading.DefaultSolanaMainnetRPCEndpoint),
		"Solana mainnet RPC URL (verified by genesis hash at runtime)",
	)
	command.Flags().StringVar(
		&rpcAttemptTimeoutRaw,
		"rpc-attempt-timeout",
		env.StringFromEnv(rpcAttemptTimeoutEnv, wormtrading.DefaultSolanaRPCAttemptTimeout.String()),
		"Timeout for one Solana JSON-RPC attempt",
	)
	command.Flags().StringVar(
		&balanceBudgetRaw,
		"balance-budget",
		env.StringFromEnv(balanceBudgetEnv, wormtrading.DefaultSolanaBalanceBudget.String()),
		"Total budget for one balance batch",
	)
	command.Flags().StringVar(
		&rpcRateLimitRaw,
		"rpc-rate-limit",
		env.StringFromEnv(rpcRateLimitEnv, strconv.FormatFloat(wormtrading.DefaultSolanaRPCRateLimit, 'f', -1, 64)),
		"Logical Solana JSON-RPC subrequests allowed per second",
	)
	command.Flags().StringVar(
		&rpcRateBurstRaw,
		"rpc-rate-burst",
		env.StringFromEnv(rpcRateBurstEnv, strconv.Itoa(wormtrading.DefaultSolanaRPCRateBurst)),
		"Logical Solana JSON-RPC subrequest burst",
	)
	command.AddCommand(cli.NewVersionCmd(cliName))
	return command
}

func parseDurationSetting(name, raw string) (time.Duration, error) {
	value, err := time.ParseDuration(raw)
	if err != nil || value <= 0 {
		return 0, fmt.Errorf("%s must be a positive duration", name)
	}
	return value, nil
}

func parseFloatSetting(name, raw string) (float64, error) {
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil || value <= 0 {
		return 0, fmt.Errorf("%s must be a positive number", name)
	}
	return value, nil
}

func parseIntSetting(name, raw string) (int, error) {
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return 0, fmt.Errorf("%s must be a positive integer", name)
	}
	return value, nil
}
