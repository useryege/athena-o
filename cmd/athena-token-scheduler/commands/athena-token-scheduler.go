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
	var chainStateRetryInterval, walletAssetRetryInterval, simulationRetryInterval, aveRetryInterval, contractSourceRetryInterval, walletNormalTransactionsRetryInterval, researchTTL time.Duration
	command := &cobra.Command{Use: cliName, Short: "Schedule token research collection", DisableAutoGenTag: true, RunE: func(cmd *cobra.Command, _ []string) error {
		connection, _, host, err := flags.Open(cmd.Context(), "scheduler")
		if err != nil {
			return err
		}
		application := researchapp.NewScheduler(tokenpostgres.NewSchedulerRepository(connection), researchapp.SchedulerOptions{RetryIntervals: map[research.DataCollectionType]time.Duration{
			research.DataCollectionTypeChainState: chainStateRetryInterval, research.DataCollectionTypeWalletAssetState: walletAssetRetryInterval,
			research.DataCollectionTypeSimulationResult: simulationRetryInterval, research.DataCollectionTypeAve: aveRetryInterval,
			research.DataCollectionTypeContractCodeSource:       contractSourceRetryInterval,
			research.DataCollectionTypeWalletNormalTransactions: walletNormalTransactionsRetryInterval,
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
	command.Flags().DurationVar(&chainStateRetryInterval, "chain-state-retry-interval", env.ParseDurationFromEnv("ATHENA_TOKEN_CHAIN_STATE_RETRY_INTERVAL", 15*time.Second, time.Second, time.Hour), "Chain state retry interval")
	command.Flags().DurationVar(&walletAssetRetryInterval, "wallet-asset-retry-interval", env.ParseDurationFromEnv("ATHENA_TOKEN_WALLET_ASSET_RETRY_INTERVAL", time.Minute, time.Second, time.Hour), "Wallet asset retry interval")
	command.Flags().DurationVar(&simulationRetryInterval, "simulation-retry-interval", env.ParseDurationFromEnv("ATHENA_TOKEN_SIMULATION_RETRY_INTERVAL", time.Minute, time.Second, time.Hour), "Simulation retry interval")
	command.Flags().DurationVar(&aveRetryInterval, "ave-retry-interval", env.ParseDurationFromEnv("ATHENA_TOKEN_AVE_RETRY_INTERVAL", 5*time.Minute, time.Second, 24*time.Hour), "Ave retry interval")
	command.Flags().DurationVar(&contractSourceRetryInterval, "contract-source-retry-interval", env.ParseDurationFromEnv("ATHENA_TOKEN_CONTRACT_SOURCE_RETRY_INTERVAL", 10*time.Minute, time.Second, 24*time.Hour), "Contract source retry interval")
	command.Flags().DurationVar(&walletNormalTransactionsRetryInterval, "wallet-normal-transactions-retry-interval", env.ParseDurationFromEnv("ATHENA_TOKEN_WALLET_NORMAL_TRANSACTIONS_RETRY_INTERVAL", 10*time.Minute, time.Second, 24*time.Hour), "Wallet normal transactions retry interval")
	command.Flags().DurationVar(&researchTTL, "research-ttl", env.ParseDurationFromEnv("ATHENA_TOKEN_RESEARCH_TTL", 24*time.Hour, time.Hour, 30*24*time.Hour), "Research expiration duration")
	command.AddCommand(cli.NewVersionCmd(cliName))
	return command
}
