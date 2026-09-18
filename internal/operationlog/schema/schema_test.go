//go:build integration

package schema_test

import (
	"bytes"
	"context"
	"github.com/jackc/pgx/v5"
	account "github.com/useryege/athena/internal/accountstate/schema"
	"github.com/useryege/athena/internal/accountstate/schema/catalog"
	"github.com/useryege/athena/internal/operationlog/schema"
	"github.com/useryege/athena/internal/testutil/pgtest"
	"github.com/useryege/athena/util/db/postgres"
	"testing"
	"time"
)

func TestVerifyEmptyIsReadOnly(t *testing.T) {
	db := pgtest.NewUnmigrated(t)
	ctx := context.Background()
	if schema.Verify(ctx, db.Pool) == nil {
		t.Fatal("empty schema accepted")
	}
	var n int
	if err := db.Pool.QueryRow(ctx, `SELECT count(*) FROM pg_namespace WHERE nspname='operation_log'`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("verify created schema: %d %v", n, err)
	}
}
func TestUpPreservesPublicAndDetectsDrift(t *testing.T) {
	db := pgtest.NewUnmigrated(t)
	ctx := context.Background()
	if err := account.Up(ctx, db.DSN); err != nil {
		t.Fatal(err)
	}
	snapshot := func() []byte {
		tx, err := db.Pool.BeginTx(ctx, pgx.TxOptions{AccessMode: pgx.ReadOnly})
		if err != nil {
			t.Fatal(err)
		}
		defer tx.Rollback(ctx)
		b, err := catalog.Read(ctx, tx)
		if err != nil {
			t.Fatal(err)
		}
		var versions []byte
		if err = tx.QueryRow(ctx, `SELECT jsonb_agg(v ORDER BY id)::text FROM public.goose_db_version v`).Scan(&versions); err != nil {
			t.Fatal(err)
		}
		return append(b, versions...)
	}
	before := snapshot()
	for i := 0; i < 2; i++ {
		if err := schema.Up(ctx, db.DSN); err != nil {
			t.Fatal(err)
		}
	}
	if !bytes.Equal(before, snapshot()) {
		t.Fatal("public catalog/version mutated")
	}
	if err := account.Verify(ctx, db.Pool); err != nil {
		t.Fatal(err)
	}
	if err := schema.Verify(ctx, db.Pool); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Pool.Exec(ctx, `DROP INDEX operation_log.entry_outcome_idx`); err != nil {
		t.Fatal(err)
	}
	if schema.Verify(ctx, db.Pool) == nil {
		t.Fatal("index drift accepted")
	}
}
func TestUpSharesBoundedMigrationLock(t *testing.T) {
	db := pgtest.NewUnmigrated(t)
	ctx := context.Background()
	conn, err := db.Pool.Acquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Release()
	if _, err = conn.Exec(ctx, `SELECT pg_advisory_lock(hashtext($1)::bigint)`, postgres.MigrationLockName); err != nil {
		t.Fatal(err)
	}
	defer conn.Exec(ctx, `SELECT pg_advisory_unlock(hashtext($1)::bigint)`, postgres.MigrationLockName)
	short, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()
	if schema.Up(short, db.DSN) == nil {
		t.Fatal("migration bypassed shared lock")
	}
	var found bool
	if err = db.Pool.QueryRow(ctx, `SELECT EXISTS(SELECT FROM pg_namespace WHERE nspname='operation_log')`).Scan(&found); err != nil || found {
		t.Fatalf("created before lock: %t %v", found, err)
	}
}
