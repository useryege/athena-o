package commands

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/useryege/athena/cmd/tokenworker"
	"github.com/useryege/athena/common"
	"github.com/useryege/athena/internal/token/adapters/evm"
	tokenpostgres "github.com/useryege/athena/internal/token/adapters/postgres"
	discoveryapp "github.com/useryege/athena/internal/token/discovery/application"
	"github.com/useryege/athena/internal/token/telemetry"
	"github.com/useryege/athena/internal/token/workerhost"
	"github.com/useryege/athena/util/cli"
)

const cliName = "athena-token-scanner"

func NewCommand() *cobra.Command {
	var flags tokenworker.CommonFlags
	command := &cobra.Command{Use: cliName, Short: "Scan configured token chains", DisableAutoGenTag: true, RunE: func(cmd *cobra.Command, _ []string) error {
		connection, registry, host, err := flags.Open(cmd.Context(), "scanner")
		if err != nil {
			return err
		}
		clients := evm.NewChainClientRegistry(registry)
		host.AddClose(clients.Close)
		repository := tokenpostgres.NewChainRepository(connection)
		application := discoveryapp.NewScanner(repository, evm.NewBlockSource(clients))
		jobs := make([]workerhost.PeriodicJob, 0, len(registry.EnabledChains()))
		for _, chain := range registry.EnabledChains() {
			chain := chain
			jobs = append(jobs, workerhost.PeriodicJob{Name: fmt.Sprintf("chain-scanner-%d", chain.ID), Interval: chain.ScannerPollInterval, Scope: telemetry.Scope{Component: "chain_scanner", ChainID: chain.ID}, Initialize: func(ctx context.Context) error { return application.StartChain(ctx, chain.ID) }, Shutdown: func(ctx context.Context) error { return application.StopChain(ctx, chain.ID) }, RunOnce: func(ctx context.Context) (workerhost.JobResult, error) {
				result, err := application.RunOnce(ctx, discoveryapp.ScanChainCommand{ChainID: chain.ID, BlockFetchConcurrency: chain.BlockFetchConcurrency})
				return workerhost.JobResult{Processed: int(result.Blocks)}, err
			}})
		}
		if len(jobs) == 0 {
			return fmt.Errorf("token scanner has no enabled chains")
		}
		host.SetWorker(workerhost.NewPeriodicWorker(jobs, host.Reporter()))
		common.GetVersion().LogStartupInfo("Athena Token Scanner", map[string]any{"chains": len(registry.EnabledChains())})
		return host.Run(cmd.Context())
	}}
	flags.Bind(command, "127.0.0.1:8110")
	command.AddCommand(cli.NewVersionCmd(cliName))
	return command
}
