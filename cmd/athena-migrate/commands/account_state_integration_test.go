//go:build integration

package commands

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/useryege/athena/internal/accountstate/schema"
	"github.com/useryege/athena/internal/migration"
	"github.com/useryege/athena/internal/testutil/pgtest"
)

func TestAggregateAccountStateDelegatesToExplicitSchema(t *testing.T) {
	db := pgtest.NewUnmigrated(t)
	t.Setenv(schema.DSNEnv, db.DSN)
	modules, err := migration.Select("account-state")
	require.NoError(t, err)
	require.ErrorIs(t, migrationStatus(context.Background(), modules[0]), schema.ErrVersions)
	var exists bool
	require.NoError(t, db.Pool.QueryRow(context.Background(), `SELECT to_regclass('public.goose_db_version') IS NOT NULL`).Scan(&exists))
	require.False(t, exists)
	require.NoError(t, migrateUp(context.Background(), modules[0]))
	require.NoError(t, migrationStatus(context.Background(), modules[0]))
	t.Setenv(schema.DSNEnv, "")
	t.Setenv("ATHENA_SERVER_POSTGRES_DSN", db.DSN)
	require.ErrorIs(t, migrateUp(context.Background(), modules[0]), schema.ErrConfiguration)
}
