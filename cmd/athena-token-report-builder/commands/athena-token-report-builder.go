package commands

import (
	"github.com/spf13/cobra"
	"github.com/useryege/athena/cmd/tokenworker"
	"github.com/useryege/athena/common"
	"github.com/useryege/athena/internal/token/reporting/builder"
	"github.com/useryege/athena/util/cli"
)

const cliName = "athena-token-report-builder"

func NewCommand() *cobra.Command {
	var flags tokenworker.CommonFlags
	command := &cobra.Command{Use: cliName, Short: "Build token project reports", DisableAutoGenTag: true, RunE: func(cmd *cobra.Command, _ []string) error {
		database, _, host, err := flags.Open(cmd.Context(), "report-builder")
		if err != nil {
			return err
		}
		host.SetWorker(builder.NewWorker(builder.Options{Store: database.Reporting(), Telemetry: host.Reporter()}))
		common.GetVersion().LogStartupInfo("Athena Token Report Builder", nil)
		return host.Run(cmd.Context())
	}}
	flags.Bind(command, "127.0.0.1:8118")
	command.AddCommand(cli.NewVersionCmd(cliName))
	return command
}
