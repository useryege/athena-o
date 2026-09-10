//go:build integration

package txgate_test

import (
	"context"
	"github.com/jackc/pgx/v5"
	migrations "github.com/useryege/athena/internal/accountstate/store/migrations"
	"github.com/useryege/athena/internal/accountstate/txgate"
	"github.com/useryege/athena/internal/testutil/pgtest"
	"testing"
	"time"
)

func TestAccountSessionHoldsAcrossCommitAndDiscardsUncertainUnlock(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	const owner = "5b0ddd23-bfc2-4c86-af84-28a984188f5c"
	gate, e := txgate.AcquireAccountSession(ctx, db.Pool, owner)
	if e != nil {
		t.Fatal(e)
	}
	defer func() {
		if gate.Conn != nil {
			_ = gate.Release(context.Background())
		}
	}()
	tx, e := gate.Conn.Begin(ctx)
	if e != nil {
		t.Fatal(e)
	}
	if e = tx.Commit(ctx); e != nil {
		t.Fatal(e)
	}
	short, stop := context.WithTimeout(ctx, 100*time.Millisecond)
	e = txgate.WithAccountTx(short, db.Pool, owner, func(pgx.Tx) error { t.Error("activity entered before session released"); return nil })
	stop()
	if e == nil {
		t.Fatal("session did not exclude transaction after commit")
	}
	if e = txgate.WithAccountTx(ctx, db.Pool, "47f3cbb5-8388-49c4-9c67-13bd2e36583b", func(pgx.Tx) error { return nil }); e != nil {
		t.Fatal(e)
	}
	pid := gate.Conn.Conn().PgConn().PID()
	dead, abort := context.WithCancel(ctx)
	abort()
	if e = gate.Release(dead); e == nil {
		t.Fatal("cancelled unlock must report failure")
	}
	if e = txgate.WithAccountTx(ctx, db.Pool, owner, func(pgx.Tx) error { return nil }); e != nil {
		t.Fatal("discarded connection left lock", e)
	}
	var exists bool
	if e = db.Pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM pg_stat_activity WHERE pid=$1)", pid).Scan(&exists); e != nil {
		t.Fatal(e)
	}
	if exists {
		t.Fatal("uncertain locked connection returned to pool")
	}
}

func TestAccountSessionCancelledAcquireDoesNotLeak(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	ctx := context.Background()
	const owner = "5b0ddd23-bfc2-4c86-af84-28a984188f5c"
	first, e := txgate.AcquireAccountSession(ctx, db.Pool, owner)
	if e != nil {
		t.Fatal(e)
	}
	defer func() {
		if first.Conn != nil {
			_ = first.Release(ctx)
		}
	}()
	short, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	second, e := txgate.AcquireAccountSession(short, db.Pool, owner)
	cancel()
	if second != nil {
		_ = second.Release(ctx)
	}
	if e == nil {
		t.Fatal("second session ignored held account lock")
	}
	if e = first.Release(ctx); e != nil {
		t.Fatal(e)
	}
	last, e := txgate.AcquireAccountSession(ctx, db.Pool, owner)
	if e != nil {
		t.Fatal(e)
	}
	if e = last.Release(ctx); e != nil {
		t.Fatal(e)
	}
}
