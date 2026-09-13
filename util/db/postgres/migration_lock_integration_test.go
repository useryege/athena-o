//go:build integration

package postgres_test

import (
	"context"
	"errors"
	"net"
	"net/url"
	"testing"
	"testing/fstest"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/useryege/athena/internal/accountstate/store/migrations"
	"github.com/useryege/athena/internal/testutil/pgtest"
	"github.com/useryege/athena/util/db/postgres"
)

func TestMigrationEntrypointsSerializeEquivalentDSNs(t *testing.T) {
	for _, operation := range []struct {
		name string
		run  func(context.Context, string) error
	}{
		{"up", func(ctx context.Context, dsn string) error {
			return postgres.Migrate(ctx, dsn, migrations.FS, migrations.Dir)
		}},
		{"status", func(ctx context.Context, dsn string) error {
			return postgres.MigrationStatus(ctx, dsn, migrations.FS, migrations.Dir)
		}},
	} {
		t.Run(operation.name, func(t *testing.T) {
			db := pgtest.NewUnmigrated(t)
			ctx := context.Background()
			conn, err := db.Pool.Acquire(ctx)
			require.NoError(t, err)
			defer conn.Release()
			_, err = conn.Exec(ctx, `SELECT pg_advisory_lock(hashtext('athena:postgres:schema:v1')::bigint)`)
			require.NoError(t, err)
			parsed, err := url.Parse(db.DSN)
			require.NoError(t, err)
			parsed.Host = net.JoinHostPort("localhost", parsed.Port())
			short, cancel := context.WithTimeout(ctx, 150*time.Millisecond)
			started := time.Now()
			err = operation.run(short, parsed.String())
			cancel()
			require.Error(t, err, "migration must not bypass another connection's database lock")
			require.True(t, errors.Is(err, context.DeadlineExceeded), "expected deadline, got %v", err)
			require.Less(t, time.Since(started), time.Second)
			var exists bool
			require.NoError(t, conn.QueryRow(ctx, `SELECT to_regclass('public.goose_db_version') IS NOT NULL`).Scan(&exists))
			require.False(t, exists, "a timed-out lock waiter must not execute schema initialization")
			_, err = conn.Exec(ctx, `SELECT pg_advisory_unlock(hashtext('athena:postgres:schema:v1')::bigint)`)
			require.NoError(t, err)
			long, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()
			require.NoError(t, operation.run(long, parsed.String()))
		})
	}
}

func TestInProcessGooseWaitHonorsCallDeadline(t *testing.T) {
	db := pgtest.NewUnmigrated(t)
	ctx := context.Background()
	owner, err := db.Pool.Acquire(ctx)
	require.NoError(t, err)
	defer owner.Release()
	_, err = owner.Exec(ctx, `SELECT pg_advisory_lock(hashtext($1)::bigint)`, postgres.MigrationLockName)
	require.NoError(t, err)
	defer owner.Exec(ctx, `SELECT pg_advisory_unlock(hashtext($1)::bigint)`, postgres.MigrationLockName)
	firstCtx, cancelFirst := context.WithTimeout(ctx, 5*time.Second)
	defer cancelFirst()
	firstDone := make(chan error, 1)
	go func() { firstDone <- postgres.Migrate(firstCtx, db.DSN, migrations.FS, migrations.Dir) }()
	require.Eventually(t, func() bool {
		var waiting int
		err := db.Pool.QueryRow(ctx, `SELECT count(*) FROM pg_stat_activity WHERE datname=current_database() AND wait_event='advisory'`).Scan(&waiting)
		return err == nil && waiting == 1
	}, 2*time.Second, 10*time.Millisecond, "first migration must hold the Go gate and wait for the database lock")
	secondCtx, cancelSecond := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancelSecond()
	secondDone := make(chan error, 1)
	go func() { secondDone <- postgres.MigrationStatus(secondCtx, db.DSN, migrations.FS, migrations.Dir) }()
	select {
	case err := <-secondDone:
		require.ErrorIs(t, err, context.DeadlineExceeded)
	case <-time.After(time.Second):
		t.Error("waiting for the in-process goose gate ignored the caller deadline")
	}
	cancelFirst()
	require.Error(t, <-firstDone)
}

func TestMigrationDeadlineReleasesAcquiredSessionLock(t *testing.T) {
	db := pgtest.NewUnmigrated(t)
	slow := fstest.MapFS{"000001_slow.sql": {Data: []byte("-- +goose Up\nSELECT pg_sleep(5);\n-- +goose Down\nSELECT 1;\n")}}
	short, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()
	require.Error(t, postgres.Migrate(short, db.DSN, slow, "."))
	var acquired bool
	conn, err := db.Pool.Acquire(context.Background())
	require.NoError(t, err)
	defer conn.Release()
	require.NoError(t, conn.QueryRow(context.Background(), `SELECT pg_try_advisory_lock(hashtext($1)::bigint)`, postgres.MigrationLockName).Scan(&acquired))
	require.True(t, acquired, "expired migration must release its session lock")
	_, err = conn.Exec(context.Background(), `SELECT pg_advisory_unlock(hashtext($1)::bigint)`, postgres.MigrationLockName)
	require.NoError(t, err)
}
