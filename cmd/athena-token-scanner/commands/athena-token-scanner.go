package commands

import (
	"github.com/spf13/cobra"
	"github.com/useryege/athena/cmd/tokenworker"
	"github.com/useryege/athena/common"
	"github.com/useryege/athena/internal/token/adapters/evm"
	"github.com/useryege/athena/internal/token/adapters/workers/scanner"
	"github.com/useryege/athena/util/cli"
)

const cliName = "athena-token-scanner"

func NewCommand() *cobra.Command {
	var flags tokenworker.CommonFlags
	command := &cobra.Command{Use: cliName, Short: "Scan configured token chains", DisableAutoGenTag: true, RunE: func(cmd *cobra.Command, _ []string) error {
		database, registry, host, err := flags.Open(cmd.Context(), "scanner")
		if err != nil {
			return err
		}
		clients := evm.NewChainClientRegistry(registry)
		host.AddClose(clients.Close)
		host.SetWorker(scanner.NewWorker(scanner.Options{Store: database.Discovery(), Chains: registry.Chains(), Clients: clients, Telemetry: host.Reporter()}))
		common.GetVersion().LogStartupInfo("Athena Token Scanner", map[string]any{"chains": len(registry.EnabledChains())})
		return host.Run(cmd.Context())
	}}
	flags.Bind(command, "127.0.0.1:8110")
	command.AddCommand(cli.NewVersionCmd(cliName))
	return command
}
