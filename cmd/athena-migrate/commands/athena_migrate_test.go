package commands

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/useryege/athena/internal/accountstate/schema"
	"github.com/useryege/athena/internal/migration"
)

func TestOperationLogMigrationRouteUsesItsSchemaOwner(t *testing.T) {
	modules, err := migration.Select("operation-log")
	require.NoError(t, err)
	var got string
	original := operationLogUp
	operationLogUp = func(_ context.Context, dsn string) error {
		got = dsn
		return nil
	}
	t.Cleanup(func() { operationLogUp = original })
	t.Setenv(schema.DSNEnv, "postgres://fixture")
	require.NoError(t, migrateUp(context.Background(), modules[0]))
	require.Equal(t, "postgres://fixture", got)
}

func TestOperationLogStatusRouteUsesItsSchemaOwner(t *testing.T) {
	modules, err := migration.Select("operation-log")
	require.NoError(t, err)
	var got string
	original := operationLogStatus
	operationLogStatus = func(_ context.Context, dsn string) error {
		got = dsn
		return nil
	}
	t.Cleanup(func() { operationLogStatus = original })
	t.Setenv(schema.DSNEnv, "postgres://fixture")
	require.NoError(t, migrationStatus(context.Background(), modules[0]))
	require.Equal(t, "postgres://fixture", got)
}
