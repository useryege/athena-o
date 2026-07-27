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
)

const cliName = "athena-token-validator"

func NewCommand() *cobra.Command {
	var flags tokenworker.CommonFlags
	command := &cobra.Command{Use: cliName, Short: "Validate token project candidates", DisableAutoGenTag: true, RunE: func(cmd *cobra.Command, _ []string) error {
		connection, registry, host, err := flags.Open(cmd.Context(), "validator")
		if err != nil {
			return err
		}
		clients := evm.NewChainClientRegistry(registry, flags.NodeWSProxyURL)
		host.AddClose(clients.Close)
		repository := tokenpostgres.NewCandidateRepository(connection)
		application := discoveryapp.NewValidator(repository, evm.NewCandidateInspector(registry, clients), repository, discoveryapp.ValidatorOptions{})
		jobs := make([]workerhost.PeriodicJob, 0, len(registry.EnabledChains()))
		for _, chain := range registry.EnabledChains() {
			chain := chain
			jobs = append(jobs, workerhost.PeriodicJob{Name: fmt.Sprintf("project-validator-%d", chain.ID), Interval: 3 * time.Second, Scope: telemetry.Scope{Component: "project_validator", ChainID: chain.ID}, RunOnce: func(ctx context.Context) (workerhost.JobResult, error) {
				count, err := application.RunOnce(ctx, chain.ID)
				return workerhost.JobResult{Processed: count}, err
			}})
		}
		if len(jobs) == 0 {
			return fmt.Errorf("token validator has no enabled chains")
		}
		host.SetWorker(workerhost.NewPeriodicWorker(jobs, host.Reporter()))
		common.GetVersion().LogStartupInfo("Athena Token Validator", map[string]any{"chains": len(registry.EnabledChains())})
		return host.Run(cmd.Context())
	}}
	flags.Bind(command, "127.0.0.1:8111")
	command.AddCommand(cli.NewVersionCmd(cliName))
	return command
}
