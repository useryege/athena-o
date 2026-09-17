//go:build integration

package schema

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/useryege/athena/internal/accountstate/store/migrations"
	"github.com/useryege/athena/internal/testutil/pgtest"
)

func TestVerifyDoesNotInitializeDatabase(t *testing.T) {
	db := pgtest.NewUnmigrated(t)
	ctx := context.Background()
	require.ErrorIs(t, Verify(ctx, db.Pool), ErrVersions)
	var exists bool
	require.NoError(t, db.Pool.QueryRow(ctx, `SELECT to_regclass('public.goose_db_version') IS NOT NULL`).Scan(&exists))
	require.False(t, exists)
}
func TestVerifyRejectsDamagedSchemaAndVersions(t *testing.T) {
	fixtures := []struct {
		name, sql string
		versions  bool
	}{
		{"missing module access table", `DROP TABLE athena_module_access_setting`, false},
		{"missing module access column", `ALTER TABLE athena_module_access_setting DROP COLUMN is_open`, false},
		{"missing column", `ALTER TABLE trader_sync_runtime_control DROP COLUMN owner_id`, false},
		{"wrong column type", `ALTER TABLE trader_sync_runtime_control ALTER COLUMN generation TYPE numeric`, false},
		{"wrong nullability", `ALTER TABLE trader_sync_runtime_control ALTER COLUMN generation DROP NOT NULL`, false},
		{"wrong default", `ALTER TABLE trader_sync_runtime_control ALTER COLUMN generation SET DEFAULT 9`, false},
		{"missing constraint", `ALTER TABLE trader_sync_runtime_control DROP CONSTRAINT trader_sync_runtime_control_generation_check`, false},
		{"missing index", `DROP INDEX trader_sync_activity_owner_time`, false},
		{"missing trigger", `DROP TRIGGER trader_sync_attempt_terminal ON trader_sync_baseline_attempts`, false},
		{"disabled trigger", `ALTER TABLE trader_sync_baseline_attempts DISABLE TRIGGER trader_sync_attempt_terminal`, false},
		{"changed function", `CREATE OR REPLACE FUNCTION trader_sync_guard_attempt_terminal() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RETURN NEW; END $$`, false},
		{"missing table with matching versions", `DROP TABLE trader_sync_runtime_control`, false},
		{"unknown version", `INSERT INTO goose_db_version(version_id,is_applied) VALUES (99,true)`, true},
		{"missing older version", `DELETE FROM goose_db_version WHERE version_id=1`, true},
		{"reverted version", `INSERT INTO goose_db_version(version_id,is_applied) VALUES (1,false)`, true},
	}
	for _, fixture := range fixtures {
		t.Run(fixture.name, func(t *testing.T) {
			db := pgtest.New(t, migrations.FS, migrations.Dir)
			ctx := context.Background()
			require.NoError(t, Verify(ctx, db.Pool))
			_, err := db.Pool.Exec(ctx, fixture.sql)
			require.NoError(t, err)
			if fixture.versions {
				require.ErrorIs(t, Verify(ctx, db.Pool), ErrVersions)
			} else {
				require.ErrorIs(t, Verify(ctx, db.Pool), ErrCatalog)
			}
		})
	}
}
func TestConnectVerifiedReturnsIndependentPoolsAndNeverMigrates(t *testing.T) {
	db := pgtest.NewUnmigrated(t)
	ctx := context.Background()
	_, err := ConnectVerified(ctx, db.DSN)
	require.ErrorIs(t, err, ErrVersions)
	require.NoError(t, Up(ctx, db.DSN))
	a, err := ConnectVerified(ctx, db.DSN)
	require.NoError(t, err)
	defer a.Close()
	b, err := ConnectVerified(ctx, db.DSN)
	require.NoError(t, err)
	defer b.Close()
	require.NotSame(t, a, b)
	a.Close()
	require.NoError(t, b.Ping(ctx))
}
func TestUpHonorsTotalLockDeadline(t *testing.T) {
	db := pgtest.NewUnmigrated(t)
	ctx := context.Background()
	conn, err := db.Pool.Acquire(ctx)
	require.NoError(t, err)
	defer conn.Release()
	_, err = conn.Exec(ctx, `SELECT pg_advisory_lock(hashtext($1)::bigint)`, MigrationLockName)
	require.NoError(t, err)
	short, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()
	require.Error(t, Up(short, db.DSN))
	var exists bool
	require.NoError(t, conn.QueryRow(ctx, `SELECT to_regclass('public.goose_db_version') IS NOT NULL`).Scan(&exists))
	require.False(t, exists)
	_, err = conn.Exec(ctx, `SELECT pg_advisory_unlock(hashtext($1)::bigint)`, MigrationLockName)
	require.NoError(t, err)
	require.NoError(t, Up(ctx, db.DSN))
	require.NoError(t, Verify(ctx, db.Pool))
}
