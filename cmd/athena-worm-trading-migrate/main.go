package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	wormstore "github.com/useryege/athena/internal/wormtrading/store"
	"github.com/useryege/athena/util/db/postgres"
)

func main() {
	if err := newCommand().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func newCommand() *cobra.Command {
	command := &cobra.Command{Use: "athena-worm-trading-migrate", Short: "Manage only the Worm Trading schema", SilenceUsage: true, SilenceErrors: true}
	for _, action := range []string{"up", "verify"} {
		timeout := wormstore.DefaultTimeout
		child := &cobra.Command{Use: action, Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
			if timeout <= 0 {
				return fmt.Errorf("timeout must be positive")
			}
			dsn := strings.TrimSpace(os.Getenv(wormstore.DSNEnv))
			if dsn == "" {
				return fmt.Errorf("%s is required", wormstore.DSNEnv)
			}
			ctx, cancel := context.WithTimeout(cmd.Context(), timeout)
			defer cancel()
			if action == "up" {
				if err := postgres.Migrate(ctx, dsn, wormstore.Migrations(), "migrations"); err != nil {
					return err
				}
			} else {
				store, err := wormstore.NewSQLStoreSource()(ctx)
				if err != nil {
					return err
				}
				defer store.Close()
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Worm Trading schema %s succeeded\n", action)
			return nil
		}}
		child.Flags().DurationVar(&timeout, "timeout", wormstore.DefaultTimeout, "Total connection, lock and operation timeout")
		command.AddCommand(child)
	}
	return command
}
