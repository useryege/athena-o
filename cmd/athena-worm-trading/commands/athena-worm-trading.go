package commands

import (
	"context"
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
	accountstore "github.com/useryege/athena/internal/accountstate/store"
	walletapiclient "github.com/useryege/athena/internal/wallet/apiclient"
	"github.com/useryege/athena/internal/wormtrading"
	wormtradingapiclient "github.com/useryege/athena/internal/wormtrading/apiclient"
	wormtradingstore "github.com/useryege/athena/internal/wormtrading/store"
	"github.com/useryege/athena/util/cli"
	"github.com/useryege/athena/util/env"
	utilio "github.com/useryege/athena/util/io"
	utilworm "github.com/useryege/athena/util/worm"
)

const (
	cliName                    = "athena-worm-trading"
	solanaRPCEndpointEnv       = "ATHENA_WORM_TRADING_SOLANA_RPC_URL"
	rpcAttemptTimeoutEnv       = "ATHENA_WORM_TRADING_RPC_ATTEMPT_TIMEOUT"
	balanceBudgetEnv           = "ATHENA_WORM_TRADING_BALANCE_BUDGET"
	rpcRateLimitEnv            = "ATHENA_WORM_TRADING_RPC_RATE_LIMIT"
	rpcRateBurstEnv            = "ATHENA_WORM_TRADING_RPC_RATE_BURST"
	credentialKeyEnv           = "ATHENA_WORM_TRADING_CREDENTIAL_ENCRYPTION_KEY"
	wormAttemptTimeoutEnv      = "ATHENA_WORM_TRADING_WORM_API_ATTEMPT_TIMEOUT"
	wormCatalogBudgetEnv       = "ATHENA_WORM_TRADING_CATALOG_BUDGET"
	wormPositionBudgetEnv      = "ATHENA_WORM_TRADING_POSITION_BUDGET"
	wormPositionConcurrencyEnv = "ATHENA_WORM_TRADING_POSITION_CONCURRENCY"
	walletServerAddressEnv     = "ATHENA_WALLET_SERVER_ADDRESS"
)

func NewCommand() *cobra.Command {
	var (
		listenHost                 string
		listenPort                 int
		solanaRPCURL               string
		rpcAttemptTimeoutRaw       string
		balanceBudgetRaw           string
		rpcRateLimitRaw            string
		rpcRateBurstRaw            string
		wormAttemptTimeoutRaw      string
		wormCatalogBudgetRaw       string
		wormPositionBudgetRaw      string
		wormPositionConcurrencyRaw string
		walletServerAddress        string
		storeSource                func(context.Context) (*wormtradingstore.SQLStore, error)
	)

	command := &cobra.Command{
		Use:   cliName,
		Short: "Run the Athena Worm Trading service",
		Long: "Worm Trading exposes trusted mainnet Solana balances, official Worm HMAC activity and preview reads, and the Worm Web JWT live execution engine. " +
			"Wallet private keys remain behind the capability-scoped Wallet execution signer. Every non-health RPC requires an independent internal Bearer of at least 32 bytes.",
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
			wormAttemptTimeout, err := parseDurationSetting("Worm API attempt timeout", wormAttemptTimeoutRaw)
			if err != nil {
				return err
			}
			wormCatalogBudget, err := parseDurationSetting("Worm catalog budget", wormCatalogBudgetRaw)
			if err != nil {
				return err
			}
			if wormCatalogBudget < wormAttemptTimeout {
				return fmt.Errorf("Worm catalog budget must be at least the API attempt timeout")
			}
			wormPositionBudget, err := parseDurationSetting("Worm position budget", wormPositionBudgetRaw)
			if err != nil {
				return err
			}
			wormPositionConcurrency, err := parseIntSetting("Worm position concurrency", wormPositionConcurrencyRaw)
			if err != nil {
				return err
			}
			credentialStore, err := storeSource(cmd.Context())
			if err != nil {
				return err
			}
			defer utilio.Close(credentialStore)
			accountStore, err := accountstore.NewSQLStoreSource()(cmd.Context())
			if err != nil {
				return err
			}
			defer utilio.Close(accountStore)
			walletSignerClientset, err := walletapiclient.NewWormExecutionSignerClientset(
				walletServerAddress,
				env.StringFromEnv(walletapiclient.WormExecutionSignerAuthTokenEnv, ""),
			)
			if err != nil {
				return fmt.Errorf("configure Wallet Worm execution signer client: %w", err)
			}
			defer utilio.Close(walletSignerClientset)
			credentialEncryptionKey, err := wormtrading.CredentialEncryptionKeyFromPassphrase(env.StringFromEnv(credentialKeyEnv, ""))
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
				AccountAccessReader:     accountStore,
				WormCatalogBudget:       wormCatalogBudget,
				BalanceAdapter:          adapter,
				CredentialStore:         credentialStore,
				CredentialEncryptionKey: credentialEncryptionKey,
				WormAPIAttemptTimeout:   wormAttemptTimeout,
				WormPositionBudget:      wormPositionBudget,
				WormPositionConcurrency: wormPositionConcurrency,
				WormWebClient:           utilworm.NewWebClient(utilworm.WebClientConfig{Timeout: wormAttemptTimeout}),
				WalletSignerClientset:   walletSignerClientset,
				InternalAuthToken:       env.StringFromEnv(wormtradingapiclient.InternalAuthTokenEnv, ""),
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
				stopGRPC(grpcServer, 10*time.Second)
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
	command.Flags().StringVar(
		&walletServerAddress,
		"wallet-server-address",
		env.StringFromEnv(walletServerAddressEnv, fmt.Sprintf("%s:%d", common.DefaultLocalGRPCHost, common.DefaultPortWallet)),
		"Athena Wallet server address used only by the Worm execution signer capability",
	)
	command.Flags().StringVar(
		&wormAttemptTimeoutRaw,
		"worm-api-attempt-timeout",
		env.StringFromEnv(wormAttemptTimeoutEnv, wormtrading.DefaultWormAPIAttemptTimeout.String()),
		"Timeout for one official Worm HMAC API attempt",
	)
	command.Flags().StringVar(
		&wormPositionBudgetRaw,
		"worm-position-budget",
		env.StringFromEnv(wormPositionBudgetEnv, wormtrading.DefaultWormPositionBudget.String()),
		"Total budget for one wallet position snapshot page",
	)
	command.Flags().StringVar(
		&wormPositionConcurrencyRaw,
		"worm-position-concurrency",
		env.StringFromEnv(wormPositionConcurrencyEnv, strconv.Itoa(wormtrading.DefaultWormPositionConcurrency)),
		"Maximum concurrent official Worm position requests",
	)
	command.Flags().StringVar(&wormCatalogBudgetRaw, "worm-catalog-budget", env.StringFromEnv(wormCatalogBudgetEnv, wormtrading.DefaultWormCatalogBudget.String()), "Total budget for one Worm order event catalog")
	storeSource = wormtradingstore.NewSQLStoreSource()
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

// stopGRPC bounds draining RPCs; stopping transport does not undo external requests.
func stopGRPC(server *grpc.Server, timeout time.Duration) {
	done := make(chan struct{})
	go func() { server.GracefulStop(); close(done) }()
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case <-done:
	case <-timer.C:
		server.Stop()
	}
}
