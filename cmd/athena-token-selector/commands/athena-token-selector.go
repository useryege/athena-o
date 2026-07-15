package commands

import (
	"github.com/spf13/cobra"
	"github.com/useryege/athena/cmd/tokenworker"
	"github.com/useryege/athena/common"
	"github.com/useryege/athena/internal/token/selection/evaluator"
	"github.com/useryege/athena/util/cli"
	"github.com/useryege/athena/util/env"
)

const cliName = "athena-token-selector"

func NewCommand() *cobra.Command {
	var flags tokenworker.CommonFlags
	var strategyKey, strategyVersion string
	command := &cobra.Command{Use: cliName, Short: "Evaluate token project selections", DisableAutoGenTag: true, RunE: func(cmd *cobra.Command, _ []string) error {
		database, _, host, err := flags.Open(cmd.Context(), "selector")
		if err != nil {
			return err
		}
		registry, err := evaluator.NewRegistry(strategyKey, strategyVersion)
		if err != nil {
			return err
		}
		host.SetWorker(evaluator.NewWorker(evaluator.Options{Store: database.Selection(), Registry: registry, Telemetry: host.Reporter()}))
		common.GetVersion().LogStartupInfo("Athena Token Selector", map[string]any{"strategy": strategyKey, "version": strategyVersion})
		return host.Run(cmd.Context())
	}}
	flags.Bind(command, "127.0.0.1:8119")
	command.Flags().StringVar(&strategyKey, "selection-strategy-key", env.StringFromEnv("ATHENA_TOKEN_SELECTION_STRATEGY_KEY", "default"), "Active selection strategy key")
	command.Flags().StringVar(&strategyVersion, "selection-strategy-version", env.StringFromEnv("ATHENA_TOKEN_SELECTION_STRATEGY_VERSION", "1"), "Active selection strategy version")
	command.AddCommand(cli.NewVersionCmd(cliName))
	return command
}
