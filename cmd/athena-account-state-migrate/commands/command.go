package commands

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/useryege/athena/internal/accountstate/schema"
)

func NewCommand() *cobra.Command {
	command := &cobra.Command{Use: "athena-account-state-migrate", Short: "Manage the shared account-state schema", SilenceUsage: true, SilenceErrors: true}
	for _, action := range []string{"up", "verify"} {
		timeout := schema.DefaultTimeout
		child := &cobra.Command{Use: action, Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
			if timeout <= 0 {
				return fmt.Errorf("timeout must be positive")
			}
			ctx, cancel := context.WithTimeout(cmd.Context(), timeout)
			defer cancel()
			dsn, err := schema.LoadDSN(os.LookupEnv)
			if err != nil {
				return err
			}
			if action == "up" {
				err = schema.Up(ctx, dsn)
			} else {
				var poolErr error
				pool, poolErr := schema.ConnectVerified(ctx, dsn)
				err = poolErr
				if pool != nil {
					pool.Close()
				}
			}
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "account-state schema %s succeeded\n", action)
			return nil
		}}
		child.Flags().DurationVar(&timeout, "timeout", schema.DefaultTimeout, "Total connection, lock and operation timeout")
		command.AddCommand(child)
	}
	return command
}
