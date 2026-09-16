//go:build integration

package main

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/useryege/athena/internal/testutil/pgtest"
)

func TestMigrationToolOnlyPreparesTradingAndVerifyNeverInitializes(t *testing.T) {
	db := pgtest.NewUnmigrated(t)
	t.Setenv("ATHENA_WORM_TRADING_POSTGRES_DSN", db.DSN)
	cmd := newCommand()
	cmd.SetArgs([]string{"verify"})
	require.Error(t, cmd.Execute())
	var exists bool
	require.NoError(t, db.Pool.QueryRow(context.Background(), `SELECT to_regclass('public.goose_db_version') IS NOT NULL`).Scan(&exists))
	require.False(t, exists)
	cmd = newCommand()
	cmd.SetArgs([]string{"up"})
	require.NoError(t, cmd.Execute())
	cmd = newCommand()
	cmd.SetArgs([]string{"verify"})
	require.NoError(t, cmd.Execute())
	require.NoError(t, db.Pool.QueryRow(context.Background(), `SELECT to_regclass('public.accounts') IS NOT NULL`).Scan(&exists))
	require.False(t, exists)
}
