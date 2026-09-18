package commands

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spf13/cobra"
	"github.com/useryege/athena/internal/accountstate/schema"
	"github.com/useryege/athena/internal/migration"
	operationlogschema "github.com/useryege/athena/internal/operationlog/schema"
	"github.com/useryege/athena/util/cli"
	"github.com/useryege/athena/util/db/postgres"
)

const cliName = "athena-migrate"

var (
	selectModules      = migration.Select
	runUp              = migrateUp
	runStatus          = migrationStatus
	operationLogUp     = operationlogschema.Up
	operationLogStatus = verifyOperationLogSchema
)

func NewCommand() *cobra.Command {
	command := &cobra.Command{
		Use:          cliName,
		Short:        "Manage Athena PostgreSQL migrations",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	command.AddCommand(newUpCommand())
	command.AddCommand(newStatusCommand())
	command.AddCommand(cli.NewVersionCmd(cliName))
	return command
}

func newUpCommand() *cobra.Command {
	var module string
	command := &cobra.Command{
		Use:          "up",
		Short:        "Apply PostgreSQL migrations",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			modules, err := selectModules(module)
			if err != nil {
				return err
			}
			for _, item := range modules {
				fmt.Fprintf(cmd.OutOrStdout(), "running %s postgres migrations\n", item.Name)
				if err := runUp(cmd.Context(), item); err != nil {
					return fmt.Errorf("%s postgres migration up: %w", item.Name, err)
				}
			}
			return nil
		},
	}
	command.Flags().StringVar(&module, "module", migration.AllModules, "Module to migrate, or all")
	return command
}

func newStatusCommand() *cobra.Command {
	var module string
	command := &cobra.Command{
		Use:          "status",
		Short:        "Show PostgreSQL migration status",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			modules, err := selectModules(module)
			if err != nil {
				return err
			}
			for _, item := range modules {
				fmt.Fprintf(cmd.OutOrStdout(), "%s postgres migration status\n", item.Name)
				if err := runStatus(cmd.Context(), item); err != nil {
					return fmt.Errorf("%s postgres migration status: %w", item.Name, err)
				}
			}
			return nil
		},
	}
	command.Flags().StringVar(&module, "module", migration.AllModules, "Module to inspect, or all")
	return command
}

func migrateUp(ctx context.Context, module migration.Module) error {
	if module.Name == "account-state" {
		dsn, err := schema.LoadDSN(os.LookupEnv)
		if err != nil {
			return err
		}
		return schema.Up(ctx, dsn)
	}
	if module.Name == "operation-log" {
		dsn, err := schema.LoadDSN(os.LookupEnv)
		if err != nil {
			return err
		}
		return operationLogUp(ctx, dsn)
	}
	return postgres.Migrate(ctx, postgres.DSN(module.DSNEnv, module.Database), module.Migrations, module.Dir)
}

func migrationStatus(ctx context.Context, module migration.Module) error {
	if module.Name == "account-state" {
		dsn, err := schema.LoadDSN(os.LookupEnv)
		if err != nil {
			return err
		}
		pool, err := schema.ConnectVerified(ctx, dsn)
		if pool != nil {
			pool.Close()
		}
		return err
	}
	if module.Name == "operation-log" {
		dsn, err := schema.LoadDSN(os.LookupEnv)
		if err != nil {
			return err
		}
		return operationLogStatus(ctx, dsn)
	}
	return postgres.MigrationStatus(ctx, postgres.DSN(module.DSNEnv, module.Database), module.Migrations, module.Dir)
}

func verifyOperationLogSchema(ctx context.Context, dsn string) error {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return err
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		return err
	}
	return operationlogschema.Verify(ctx, pool)
}
