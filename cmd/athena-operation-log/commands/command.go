package commands

import "github.com/spf13/cobra"

// NewCommand exposes the readiness probe used by the production Compose
// healthcheck. The service process itself remains owned by cmd/athena-operation-log.
func NewCommand() *cobra.Command {
	command := &cobra.Command{Use: "athena-operation-log", SilenceUsage: true, SilenceErrors: true}
	command.AddCommand(newHealthCommand())
	return command
}
