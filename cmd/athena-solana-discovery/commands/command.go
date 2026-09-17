package commands

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/useryege/athena/internal/accountstate/schema"
	accountstore "github.com/useryege/athena/internal/accountstate/store"
	"github.com/useryege/athena/internal/serviceschema"
	"github.com/useryege/athena/internal/solanadiscovery"
	"github.com/useryege/athena/internal/solanadiscovery/rpcservice"
	solanapb "github.com/useryege/athena/pkg/apiclient/solana"
	"github.com/useryege/athena/util/db/postgres"
	"github.com/useryege/athena/util/env"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

func NewCommand() *cobra.Command {
	c := defaultConfig()
	command := &cobra.Command{Use: "athena-solana-discovery", Short: "Discover newly initialized Solana Mainnet mints", SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := c.validate(); err != nil {
				return err
			}
			ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
			defer stop()
			pool, err := postgres.OpenPool(ctx, "solana-discovery", c.DSN)
			if err != nil {
				return err
			}
			cleanupOwned := true
			defer func() {
				if cleanupOwned {
					pool.Close()
				}
			}()
			store := solanadiscovery.NewStore(pool)
			if err := schema.Verify(ctx, pool); err != nil {
				return fmt.Errorf("verify Solana account schema: %w", err)
			}
			if err := store.Verify(ctx); err != nil {
				return fmt.Errorf("verify Solana discovery: %w", err)
			}
			service, err := rpcservice.NewServer(store, accountstore.NewSQLStore(pool), c.InternalToken)
			if err != nil {
				return err
			}
			listener, err := (&net.ListenConfig{}).Listen(ctx, "tcp", net.JoinHostPort(c.Address, strconv.Itoa(c.Port)))
			if err != nil {
				return err
			}
			server := grpc.NewServer()
			solanapb.RegisterSolanaServiceServer(server, service)
			h := health.NewServer()
			healthpb.RegisterHealthServer(server, h)
			h.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
			defer h.Shutdown()
			client := &http.Client{Timeout: c.RequestTimeout}
			defer client.CloseIdleConnections()
			scanner := solanadiscovery.NewScanner(store, solanadiscovery.NewRPCClient(c.RPCURL, client), solanadiscovery.ScannerConfig{
				StartSlot: c.StartSlot, InitialLookback: 32, RangeSize: c.RangeSize, Concurrency: c.Concurrency, RequestsPerSecond: float64(c.RequestsPerSecond), RequestTimeout: c.RequestTimeout, PollInterval: c.PollInterval,
			})
			cmd.Printf("Solana discovery listening on %s; finalized scan progress is available via GetDiscoveryStatus\n", listener.Addr())
			enricher := solanadiscovery.NewEnricher(scanner)
			cleanupOwned = false
			return serveWithCleanup(ctx, listener, server, func(ctx context.Context) error {
				return runDiscovery(ctx, scanner.Run, enricher.Run)
			}, pool.Close)
		},
	}
	f := command.Flags()
	f.StringVar(&c.Address, "address", env.StringFromEnv("ATHENA_SOLANA_DISCOVERY_LISTEN_ADDRESS", c.Address), "gRPC listen address")
	// String defaults parsed by pflag reject invalid environment values, instead of silently falling back.
	f.IntVar(&c.Port, "port", c.Port, "gRPC listen port")
	f.StringVar(&c.RPCURL, "rpc-url", env.StringFromEnv("ATHENA_SOLANA_DISCOVERY_RPC_URL", c.RPCURL), "Solana Mainnet HTTP RPC URL")
	f.StringVar(&c.DSN, "postgres-dsn", env.StringFromEnv("ATHENA_SOLANA_DISCOVERY_POSTGRES_DSN", env.StringFromEnv(schema.DSNEnv, "")), "PostgreSQL database containing Solana schema and account access")
	c.InternalToken = os.Getenv(rpcservice.InternalAuthTokenEnv)
	f.Uint64Var(&c.StartSlot, "start-slot", c.StartSlot, "First scan slot; 0 starts 32 slots behind finalized head, ignored after initial checkpoint")
	f.Uint64Var(&c.RangeSize, "range-size", c.RangeSize, "Slots per committed range (1..32)")
	f.IntVar(&c.Concurrency, "concurrency", c.Concurrency, "Maximum concurrent block requests")
	f.IntVar(&c.RequestsPerSecond, "requests-per-second", c.RequestsPerSecond, "Node request budget per second")
	f.DurationVar(&c.RequestTimeout, "request-timeout", c.RequestTimeout, "Timeout for each node request")
	f.DurationVar(&c.PollInterval, "poll-interval", c.PollInterval, "Poll interval after catching up")
	var environmentError error
	for name, variable := range map[string]string{"port": "ATHENA_SOLANA_DISCOVERY_PORT", "start-slot": "ATHENA_SOLANA_DISCOVERY_START_SLOT", "range-size": "ATHENA_SOLANA_DISCOVERY_RANGE_SIZE", "concurrency": "ATHENA_SOLANA_DISCOVERY_CONCURRENCY", "requests-per-second": "ATHENA_SOLANA_DISCOVERY_REQUESTS_PER_SECOND", "request-timeout": "ATHENA_SOLANA_DISCOVERY_REQUEST_TIMEOUT", "poll-interval": "ATHENA_SOLANA_DISCOVERY_POLL_INTERVAL"} {
		if value, ok := os.LookupEnv(variable); ok {
			if err := f.Set(name, value); err != nil {
				environmentError = fmt.Errorf("invalid %s: %w", variable, err)
			}
		}
	}
	command.PreRunE = func(*cobra.Command, []string) error { return environmentError }
	command.AddCommand(serviceschema.NewCommand(solanadiscovery.Schema()))
	return command
}
