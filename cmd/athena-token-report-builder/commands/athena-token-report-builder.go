package commands

import (
	"context"
	"time"

	"github.com/spf13/cobra"
	"github.com/useryege/athena/cmd/tokenworker"
	"github.com/useryege/athena/common"
	tokenpostgres "github.com/useryege/athena/internal/token/adapters/postgres"
	reportingapp "github.com/useryege/athena/internal/token/reporting/application"
	"github.com/useryege/athena/internal/token/telemetry"
	"github.com/useryege/athena/internal/token/workerhost"
	"github.com/useryege/athena/util/cli"
)

const cliName = "athena-token-report-builder"

func NewCommand() *cobra.Command {
	var flags tokenworker.CommonFlags
	command := &cobra.Command{Use: cliName, Short: "Build token project reports", DisableAutoGenTag: true, RunE: func(cmd *cobra.Command, _ []string) error {
		connection, _, host, err := flags.Open(cmd.Context(), "report-builder")
		if err != nil {
			return err
		}
		application := reportingapp.NewBuilder(tokenpostgres.NewReportingRepository(connection), reportingapp.BuilderOptions{})
		job := workerhost.PeriodicJob{Name: "report-builder", Interval: time.Second, Scope: telemetry.Scope{Component: "report_builder"}, RunOnce: func(ctx context.Context) (workerhost.JobResult, error) {
			count, err := application.RunOnce(ctx)
			return workerhost.JobResult{Processed: count}, err
		}}
		host.SetWorker(workerhost.NewPeriodicWorker([]workerhost.PeriodicJob{job}, host.Reporter()))
		common.GetVersion().LogStartupInfo("Athena Token Report Builder", nil)
		return host.Run(cmd.Context())
	}}
	flags.Bind(command, "127.0.0.1:8118")
	command.AddCommand(cli.NewVersionCmd(cliName))
	return command
}
