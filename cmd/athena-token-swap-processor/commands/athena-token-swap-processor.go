package commands

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/useryege/athena/cmd/tokenworker"
	"github.com/useryege/athena/common"
	"github.com/useryege/athena/internal/token/adapters/evm"
	tokenpostgres "github.com/useryege/athena/internal/token/adapters/postgres"
	swapapp "github.com/useryege/athena/internal/token/swap/application"
	"github.com/useryege/athena/internal/token/telemetry"
	"github.com/useryege/athena/internal/token/workerhost"
	"github.com/useryege/athena/util/cli"
)

const cliName = "athena-token-swap-processor"

func NewCommand() *cobra.Command {
	var flags tokenworker.CommonFlags
	command := &cobra.Command{Use: cliName, Short: "Collect project Pair Swap events", DisableAutoGenTag: true, RunE: func(cmd *cobra.Command, _ []string) error {
		connection, registry, host, err := flags.Open(cmd.Context(), "swap-processor")
		if err != nil {
			return err
		}
		clients := evm.NewChainClientRegistry(registry, flags.NodeWSProxyURL)
		source, err := evm.NewSwapBlockSource(clients)
		if err != nil {
			_ = clients.Close()
			_ = connection.Close()
			return err
		}
		host.AddClose(clients.Close)
		application := swapapp.NewSwapProcessor(tokenpostgres.NewSwapRepository(connection), source)
		jobs := make([]workerhost.PeriodicJob, 0, len(registry.EnabledChains()))
		for _, chain := range registry.EnabledChains() {
			chain := chain
			jobs = append(jobs, workerhost.PeriodicJob{
				Name:       fmt.Sprintf("swap-processor-%d", chain.ID),
				Interval:   chain.SwapPollInterval,
				Scope:      telemetry.Scope{Component: "swap_processor", ChainID: chain.ID},
				Initialize: func(ctx context.Context) error { return application.StartChain(ctx, chain.ID) },
				Shutdown:   func(ctx context.Context) error { return application.StopChain(ctx, chain.ID) },
				RunOnce: func(ctx context.Context) (workerhost.JobResult, error) {
					result, err := application.RunOnce(ctx, chain.ID)
					return workerhost.JobResult{Processed: int(result.Blocks)}, err
				},
			})
		}
		if len(jobs) == 0 {
			return fmt.Errorf("token Swap processor has no enabled chains")
		}
		host.SetWorker(workerhost.NewPeriodicWorker(jobs, host.Reporter()))
		common.GetVersion().LogStartupInfo("Athena Token Swap Processor", map[string]any{"chains": len(registry.EnabledChains())})
		return host.Run(cmd.Context())
	}}
	flags.Bind(command, "127.0.0.1:8111")
	command.AddCommand(cli.NewVersionCmd(cliName))
	return command
}
