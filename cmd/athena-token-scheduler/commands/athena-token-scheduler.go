package commands

import (
	"context"
	"time"

	"github.com/spf13/cobra"
	"github.com/useryege/athena/cmd/tokenworker"
	"github.com/useryege/athena/common"
	tokenpostgres "github.com/useryege/athena/internal/token/adapters/postgres"
	"github.com/useryege/athena/internal/token/research"
	researchapp "github.com/useryege/athena/internal/token/research/application"
	"github.com/useryege/athena/internal/token/telemetry"
	"github.com/useryege/athena/internal/token/workerhost"
	"github.com/useryege/athena/util/cli"
	"github.com/useryege/athena/util/env"
)

const cliName = "athena-token-scheduler"

func NewCommand() *cobra.Command {
	var flags tokenworker.CommonFlags
	var chainStateInterval, walletAssetInterval, simulationInterval, aveInterval, contractSourceInterval, walletNormalTransactionsInterval, researchTTL time.Duration
	command := &cobra.Command{Use: cliName, Short: "Schedule token research collection", DisableAutoGenTag: true, RunE: func(cmd *cobra.Command, _ []string) error {
		connection, _, host, err := flags.Open(cmd.Context(), "scheduler")
		if err != nil {
			return err
		}
		application := researchapp.NewScheduler(tokenpostgres.NewSchedulerRepository(connection), researchapp.SchedulerOptions{Intervals: map[research.DataCollectionType]time.Duration{
			research.DataCollectionTypeChainState: chainStateInterval, research.DataCollectionTypeWalletAssetState: walletAssetInterval,
			research.DataCollectionTypeSimulationResult: simulationInterval, research.DataCollectionTypeAve: aveInterval,
			research.DataCollectionTypeContractCodeSource:       contractSourceInterval,
			research.DataCollectionTypeWalletNormalTransactions: walletNormalTransactionsInterval,
		}, TTL: researchTTL})
		job := workerhost.PeriodicJob{Name: "research-scheduler", Interval: time.Second, Scope: telemetry.Scope{Component: "research_scheduler"}, Initialize: application.Initialize, RunOnce: func(ctx context.Context) (workerhost.JobResult, error) {
			count, err := application.RunOnce(ctx)
			return workerhost.JobResult{Processed: count}, err
		}}
		host.SetWorker(workerhost.NewPeriodicWorker([]workerhost.PeriodicJob{job}, host.Reporter()))
		common.GetVersion().LogStartupInfo("Athena Token Scheduler", nil)
		return host.Run(cmd.Context())
	}}
	flags.Bind(command, "127.0.0.1:8112")
	command.Flags().DurationVar(&chainStateInterval, "chain-state-interval", env.ParseDurationFromEnv("ATHENA_TOKEN_CHAIN_STATE_INTERVAL", 15*time.Second, time.Second, time.Hour), "Chain state refresh interval")
	command.Flags().DurationVar(&walletAssetInterval, "wallet-asset-interval", env.ParseDurationFromEnv("ATHENA_TOKEN_WALLET_ASSET_INTERVAL", time.Minute, time.Second, time.Hour), "Wallet asset refresh interval")
	command.Flags().DurationVar(&simulationInterval, "simulation-interval", env.ParseDurationFromEnv("ATHENA_TOKEN_SIMULATION_INTERVAL", time.Minute, time.Second, time.Hour), "Simulation refresh interval")
	command.Flags().DurationVar(&aveInterval, "ave-interval", env.ParseDurationFromEnv("ATHENA_TOKEN_AVE_INTERVAL", 5*time.Minute, time.Second, 24*time.Hour), "Ave refresh interval")
	command.Flags().DurationVar(&contractSourceInterval, "contract-source-interval", env.ParseDurationFromEnv("ATHENA_TOKEN_CONTRACT_SOURCE_INTERVAL", 10*time.Minute, time.Second, 24*time.Hour), "Contract source refresh interval")
	command.Flags().DurationVar(&walletNormalTransactionsInterval, "wallet-normal-transactions-interval", env.ParseDurationFromEnv("ATHENA_TOKEN_WALLET_NORMAL_TRANSACTIONS_INTERVAL", 10*time.Minute, time.Second, 24*time.Hour), "Wallet normal transactions retry interval")
	command.Flags().DurationVar(&researchTTL, "research-ttl", env.ParseDurationFromEnv("ATHENA_TOKEN_RESEARCH_TTL", 24*time.Hour, time.Hour, 30*24*time.Hour), "Research expiration duration")
	command.AddCommand(cli.NewVersionCmd(cliName))
	return command
}
