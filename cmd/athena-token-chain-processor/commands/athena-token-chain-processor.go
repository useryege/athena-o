package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/useryege/athena/cmd/tokenworker"
	"github.com/useryege/athena/common"
	"github.com/useryege/athena/internal/token/adapters/evm"
	tokenpostgres "github.com/useryege/athena/internal/token/adapters/postgres"
	discoveryapp "github.com/useryege/athena/internal/token/discovery/application"
	"github.com/useryege/athena/internal/token/telemetry"
	"github.com/useryege/athena/internal/token/workerhost"
	"github.com/useryege/athena/util/cli"
	"github.com/useryege/athena/util/env"
)

const cliName = "athena-token-chain-processor"

func NewCommand() *cobra.Command {
	var flags tokenworker.CommonFlags
	var researchTTL time.Duration
	command := &cobra.Command{Use: cliName, Short: "Process configured token chains", DisableAutoGenTag: true, RunE: func(cmd *cobra.Command, _ []string) error {
		connection, registry, host, err := flags.Open(cmd.Context(), "chain-processor")
		if err != nil {
			return err
		}
		clients := evm.NewChainClientRegistry(registry, flags.NodeWSProxyURL)
		host.AddClose(clients.Close)
		repository := tokenpostgres.NewChainRepository(connection)
		application := discoveryapp.NewChainProcessor(repository, evm.NewBlockSource(clients), evm.NewCandidateInspector(registry, clients), discoveryapp.ChainProcessorOptions{ResearchTTL: researchTTL})
		jobs := make([]workerhost.PeriodicJob, 0, len(registry.EnabledChains()))
		for _, chain := range registry.EnabledChains() {
			chain := chain
			jobs = append(jobs, workerhost.PeriodicJob{Name: fmt.Sprintf("chain-processor-%d", chain.ID), Interval: chain.ProcessorPollInterval, Scope: telemetry.Scope{Component: "chain_processor", ChainID: chain.ID}, Initialize: func(ctx context.Context) error { return application.StartChain(ctx, chain.ID) }, Shutdown: func(ctx context.Context) error { return application.StopChain(ctx, chain.ID) }, RunOnce: func(ctx context.Context) (workerhost.JobResult, error) {
				result, err := application.RunOnce(ctx, discoveryapp.ProcessChainCommand{ChainID: chain.ID, InitialLookbackDuration: chain.ProcessorInitialLookbackDuration})
				return workerhost.JobResult{Processed: int(result.Blocks)}, err
			}})
		}
		if len(jobs) == 0 {
			return fmt.Errorf("token chain processor has no enabled chains")
		}
		host.SetWorker(workerhost.NewPeriodicWorker(jobs, host.Reporter()))
		common.GetVersion().LogStartupInfo("Athena Token Chain Processor", map[string]any{"chains": len(registry.EnabledChains())})
		return host.Run(cmd.Context())
	}}
	flags.Bind(command, "127.0.0.1:8110")
	command.Flags().DurationVar(&researchTTL, "research-ttl", env.ParseDurationFromEnv("ATHENA_TOKEN_RESEARCH_TTL", 24*time.Hour, time.Hour, 30*24*time.Hour), "Project research attention duration measured in chain block time")
	command.AddCommand(cli.NewVersionCmd(cliName))
	return command
}
