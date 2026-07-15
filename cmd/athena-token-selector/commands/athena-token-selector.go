package commands

import (
	"context"
	"time"

	"github.com/spf13/cobra"
	"github.com/useryege/athena/cmd/tokenworker"
	"github.com/useryege/athena/common"
	tokenpostgres "github.com/useryege/athena/internal/token/adapters/postgres"
	"github.com/useryege/athena/internal/token/selection"
	selectionapp "github.com/useryege/athena/internal/token/selection/application"
	"github.com/useryege/athena/internal/token/telemetry"
	"github.com/useryege/athena/internal/token/workerhost"
	"github.com/useryege/athena/util/cli"
	"github.com/useryege/athena/util/env"
)

const cliName = "athena-token-selector"

func NewCommand() *cobra.Command {
	var flags tokenworker.CommonFlags
	var strategyKey, strategyVersion string
	command := &cobra.Command{Use: cliName, Short: "Evaluate token project selections", DisableAutoGenTag: true, RunE: func(cmd *cobra.Command, _ []string) error {
		connection, _, host, err := flags.Open(cmd.Context(), "selector")
		if err != nil {
			return err
		}
		registry, err := selection.NewStrategyRegistry(strategyKey, strategyVersion)
		if err != nil {
			return err
		}
		application := selectionapp.NewEvaluator(tokenpostgres.NewSelectionRepository(connection), registry, selectionapp.EvaluatorOptions{})
		job := workerhost.PeriodicJob{Name: "selection-evaluator", Interval: time.Second, Scope: telemetry.Scope{Component: "selection_evaluator"}, RunOnce: func(ctx context.Context) (workerhost.JobResult, error) {
			count, err := application.RunOnce(ctx)
			return workerhost.JobResult{Processed: count}, err
		}}
		host.SetWorker(workerhost.NewPeriodicWorker([]workerhost.PeriodicJob{job}, host.Reporter()))
		common.GetVersion().LogStartupInfo("Athena Token Selector", map[string]any{"strategy": strategyKey, "version": strategyVersion})
		return host.Run(cmd.Context())
	}}
	flags.Bind(command, "127.0.0.1:8119")
	command.Flags().StringVar(&strategyKey, "selection-strategy-key", env.StringFromEnv("ATHENA_TOKEN_SELECTION_STRATEGY_KEY", "default"), "Active selection strategy key")
	command.Flags().StringVar(&strategyVersion, "selection-strategy-version", env.StringFromEnv("ATHENA_TOKEN_SELECTION_STRATEGY_VERSION", "1"), "Active selection strategy version")
	command.AddCommand(cli.NewVersionCmd(cliName))
	return command
}
