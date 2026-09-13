//go:build integration

package main

import (
	"context"
	"encoding/json"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/useryege/athena/internal/testutil/pgtest"
)

func TestToolMigratesOnlyItsTemporaryDatabase(t *testing.T) {
	admin := pgtest.NewUnmigrated(t)
	t.Setenv("ATHENA_TEST_PG_ADMIN_DSN", admin.DSN)
	output, err := exec.Command("go", "run", ".").Output()
	require.NoError(t, err)
	require.True(t, json.Valid(output))
	require.Contains(t, string(output), `"name": "trader_sync_runtime_control"`)
	require.Contains(t, string(output), "trader_sync_guard_attempt_terminal()")
	var count int
	require.NoError(t, admin.Pool.QueryRow(context.Background(), `SELECT count(*) FROM pg_class c JOIN pg_namespace n ON n.oid=c.relnamespace WHERE n.nspname='public'`).Scan(&count))
	require.Zero(t, count, "admin connection must never be migrated")
	require.NoError(t, admin.Pool.QueryRow(context.Background(), `SELECT count(*) FROM pg_database WHERE datname LIKE 'athena_contract_%'`).Scan(&count))
	require.Zero(t, count, "tool must remove its random database")
}
