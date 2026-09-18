package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spf13/cobra"
	"github.com/useryege/athena/internal/operationlog/schema"
)

const dsnEnv = "ATHENA_ACCOUNT_STATE_POSTGRES_DSN"

func main() {
	if err := newCommand().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func newCommand() *cobra.Command {
	command := &cobra.Command{Use: "athena-operation-log-migrate", Short: "Manage only the operation-log schema", SilenceUsage: true, SilenceErrors: true}
	for _, action := range []string{"up", "verify"} {
		action := action
		timeout := schema.DefaultTimeout
		child := &cobra.Command{Use: action, Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
			if timeout <= 0 {
				return fmt.Errorf("timeout must be positive")
			}
			dsn := strings.TrimSpace(os.Getenv(dsnEnv))
			if dsn == "" {
				return fmt.Errorf("%s is required", dsnEnv)
			}
			ctx, cancel := context.WithTimeout(cmd.Context(), timeout)
			defer cancel()
			if action == "up" {
				if err := schema.Up(ctx, dsn); err != nil {
					return err
				}
			} else {
				pool, err := pgxpool.New(ctx, dsn)
				if err != nil {
					return err
				}
				defer pool.Close()
				if err := pool.Ping(ctx); err != nil {
					return err
				}
				if err := schema.Verify(ctx, pool); err != nil {
					return err
				}
			}
			fmt.Fprintf(cmd.OutOrStdout(), "operation-log schema %s succeeded\n", action)
			return nil
		}}
		child.Flags().DurationVar(&timeout, "timeout", schema.DefaultTimeout, "Total connection, lock and operation timeout")
		command.AddCommand(child)
	}
	return command
}
