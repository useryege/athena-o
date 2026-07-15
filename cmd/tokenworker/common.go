package tokenworker

import (
	"context"
	"time"

	"github.com/spf13/cobra"
	cmdutil "github.com/useryege/athena/cmd/util"
	"github.com/useryege/athena/common"
	tokenpostgres "github.com/useryege/athena/internal/token/adapters/postgres"
	"github.com/useryege/athena/internal/token/chainregistry"
	"github.com/useryege/athena/internal/token/discovery"
	"github.com/useryege/athena/internal/token/workerhost"
	"github.com/useryege/athena/util/cli"
	"github.com/useryege/athena/util/env"
)

type CommonFlags struct {
	ChainsJSON          string
	HealthListenAddress string
	HealthStaleAfter    time.Duration
}

func (f *CommonFlags) Bind(command *cobra.Command, defaultHealthAddress string) {
	command.Flags().StringVar(&cmdutil.LogFormat, "logformat", env.StringFromEnv(common.EnvLogFormat, "json"), "Set log format: json|text")
	command.Flags().StringVar(&cmdutil.LogLevel, "loglevel", env.StringFromEnv(common.EnvLogLevel, "info"), "Set log level")
	command.Flags().StringVar(&f.ChainsJSON, "chains-json", env.StringFromEnv(chainregistry.EnvironmentVariable, ""), "Token chain registry JSON")
	command.Flags().StringVar(&f.HealthListenAddress, "health-listen-address", env.StringFromEnv("ATHENA_TOKEN_HEALTH_LISTEN_ADDRESS", defaultHealthAddress), "Health, readiness, and metrics listen address")
	command.Flags().DurationVar(&f.HealthStaleAfter, "health-stale-after", env.ParseDurationFromEnv("ATHENA_TOKEN_HEALTH_STALE_AFTER", 2*time.Minute, time.Minute, time.Hour), "Maximum successful loop age before readiness fails")
}

func (f *CommonFlags) Open(ctx context.Context, name string) (*tokenpostgres.Connection, *chainregistry.Registry, *workerhost.Host, error) {
	cli.SetLogFormat(cmdutil.LogFormat)
	cli.SetLogLevel(cmdutil.LogLevel)
	registry, err := chainregistry.Parse(f.ChainsJSON)
	if err != nil {
		return nil, nil, nil, err
	}
	connection, err := tokenpostgres.NewConnectionSource()(ctx)
	if err != nil {
		return nil, nil, nil, err
	}
	chains := make([]discovery.Chain, 0, len(registry.Chains()))
	for _, chain := range registry.Chains() {
		chains = append(chains, discovery.Chain{ID: chain.ID, Name: chain.Name, Enabled: chain.Enabled})
	}
	chainRepository := tokenpostgres.NewChainRepository(connection)
	if err := chainRepository.SyncChains(ctx, chains); err != nil {
		_ = connection.Close()
		return nil, nil, nil, err
	}
	diagnostics := tokenpostgres.NewDiagnosticsRepository(connection)
	host, _, err := workerhost.New(workerhost.Options{
		Name:                name,
		HealthListenAddress: f.HealthListenAddress,
		HealthStaleAfter:    f.HealthStaleAfter,
		Ping:                connection.Ping,
		Diagnostics: func(ctx context.Context) ([]workerhost.QueueMetric, error) {
			metrics, err := diagnostics.ListPipelineQueueMetrics(ctx)
			if err != nil {
				return nil, err
			}
			result := make([]workerhost.QueueMetric, 0, len(metrics))
			for _, metric := range metrics {
				result = append(result, workerhost.QueueMetric{Queue: metric.Queue, Status: metric.Status, Count: metric.Count, OldestAvailableAt: metric.OldestAvailableAt})
			}
			return result, nil
		},
		Close: connection.Close,
	})
	if err != nil {
		_ = connection.Close()
		return nil, nil, nil, err
	}
	return connection, registry, host, nil
}
