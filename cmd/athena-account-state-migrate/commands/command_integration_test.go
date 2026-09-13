//go:build integration

package commands

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/useryege/athena/internal/accountstate/schema"
	"github.com/useryege/athena/internal/testutil/pgtest"
)

func TestCLIUpThenReadOnlyVerify(t *testing.T) {
	db := pgtest.NewUnmigrated(t)
	t.Setenv(schema.DSNEnv, db.DSN)
	run := func(args ...string) error {
		cmd := NewCommand()
		cmd.SetArgs(args)
		cmd.SetOut(&bytes.Buffer{})
		return cmd.Execute()
	}
	require.ErrorIs(t, run("verify"), schema.ErrVersions)
	var exists bool
	require.NoError(t, db.Pool.QueryRow(context.Background(), `SELECT to_regclass('public.goose_db_version') IS NOT NULL`).Scan(&exists))
	require.False(t, exists)
	require.NoError(t, run("up", "--timeout=10s"))
	require.NoError(t, run("verify", "--timeout=10s"))
}
func TestCLITotalTimeoutIncludesLockWait(t *testing.T) {
	db := pgtest.NewUnmigrated(t)
	t.Setenv(schema.DSNEnv, db.DSN)
	ctx := context.Background()
	conn, err := db.Pool.Acquire(ctx)
	require.NoError(t, err)
	defer conn.Release()
	_, err = conn.Exec(ctx, `SELECT pg_advisory_lock(hashtext($1)::bigint)`, schema.MigrationLockName)
	require.NoError(t, err)
	defer conn.Exec(ctx, `SELECT pg_advisory_unlock(hashtext($1)::bigint)`, schema.MigrationLockName)
	cmd := NewCommand()
	cmd.SetArgs([]string{"up", "--timeout=100ms"})
	started := time.Now()
	require.Error(t, cmd.Execute())
	require.Less(t, time.Since(started), time.Second)
}
