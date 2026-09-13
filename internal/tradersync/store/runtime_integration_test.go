//go:build integration

package store

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
	accountstatemigrations "github.com/useryege/athena/internal/accountstate/store/migrations"
	"github.com/useryege/athena/internal/accountstate/txgate"
	"github.com/useryege/athena/internal/testutil/pgtest"
)

func TestRuntimeRejectsStaleGeneration(t *testing.T) {
	db := pgtest.New(t, accountstatemigrations.FS, accountstatemigrations.Dir)
	ctx := context.Background()
	s := NewSQLStore(db.Pool)
	owner, err := s.AcquireRuntimeSession(ctx)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, owner.CloseAfterWorkers(ctx)) })
	token := owner.RuntimeToken()
	token.Generation++
	gate, err := NewRuntimeWriteGate(db.Pool, token)
	require.NoError(t, err)
	tx, err := gate.BeginTx(ctx, pgx.TxOptions{})
	require.ErrorIs(t, err, ErrRuntimeFenced)
	require.Nil(t, tx)
	called := false
	err = txgate.WithAccountTx(ctx, gate, uuid.NewString(), func(tx pgx.Tx) error {
		called = true
		_, err := tx.Exec(ctx, "INSERT INTO trader_sync_collector_epochs(fencing_token) VALUES(99)")
		return err
	})
	require.ErrorIs(t, err, ErrRuntimeFenced)
	require.False(t, called)
	var count int
	require.NoError(t, db.Pool.QueryRow(ctx, "SELECT count(*) FROM trader_sync_collector_epochs").Scan(&count))
	require.Zero(t, count)
	token = owner.RuntimeToken()
	token.OwnerID = uuid.New()
	gate, err = NewRuntimeWriteGate(db.Pool, token)
	require.NoError(t, err)
	tx, err = gate.BeginTx(ctx, pgx.TxOptions{})
	require.ErrorIs(t, err, ErrRuntimeFenced)
	require.Nil(t, tx)
}

func TestRuntimeReplacementWaitsForOldWritesAndStaleClosePreservesOwner(t *testing.T) {
	db := pgtest.New(t, accountstatemigrations.FS, accountstatemigrations.Dir)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	s := NewSQLStore(db.Pool)
	a, err := s.AcquireRuntimeSession(ctx)
	require.NoError(t, err)
	defer a.CloseAfterWorkers(context.Background())
	aToken := a.RuntimeToken()
	gateA, err := NewRuntimeWriteGate(db.Pool, aToken)
	require.NoError(t, err)
	oldTx, err := gateA.BeginTx(ctx, pgx.TxOptions{})
	require.NoError(t, err)
	defer oldTx.Rollback(context.Background())
	_, err = oldTx.Exec(ctx, "INSERT INTO trader_sync_collector_epochs(fencing_token) VALUES(41)")
	require.NoError(t, err)
	// Kill only A's dedicated advisory connection, leaving its guarded write alive.
	pid := a.conn.Conn().PgConn().PID()
	var killed bool
	require.NoError(t, db.Pool.QueryRow(ctx, "SELECT pg_terminate_backend($1)", pid).Scan(&killed))
	require.True(t, killed)
	require.Eventually(t, func() bool {
		var exists bool
		err := db.Pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM pg_locks WHERE pid=$1 AND locktype='advisory' AND granted)", pid).Scan(&exists)
		return err == nil && !exists
	}, 3*time.Second, 10*time.Millisecond)
	type result struct {
		session *RuntimeSession
		err     error
	}
	acquired := make(chan result, 1)
	go func() { b, err := s.AcquireRuntimeSession(ctx); acquired <- result{b, err} }()
	// Prove B passed the advisory lock and is waiting for the runtime row lock.
	require.Eventually(t, func() bool {
		var waiting bool
		err := db.Pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM pg_stat_activity a WHERE a.datname=current_database() AND a.wait_event_type='Lock' AND a.query LIKE '%trader_sync_runtime_control%FOR UPDATE%' AND EXISTS(SELECT 1 FROM pg_locks l WHERE l.pid=a.pid AND l.locktype='advisory' AND l.granted))`).Scan(&waiting)
		return err == nil && waiting
	}, 3*time.Second, 10*time.Millisecond)
	select {
	case r := <-acquired:
		t.Fatalf("replacement passed an open runtime write: %v", r.err)
	default:
	}
	require.NoError(t, oldTx.Commit(ctx))
	var b *RuntimeSession
	select {
	case r := <-acquired:
		require.NoError(t, r.err)
		b = r.session
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	defer b.CloseAfterWorkers(context.Background())
	require.Greater(t, b.RuntimeToken().Generation, aToken.Generation)
	require.Greater(t, b.CollectorToken(), a.CollectorToken())
	var count int
	require.NoError(t, db.Pool.QueryRow(ctx, "SELECT count(*) FROM trader_sync_collector_epochs WHERE fencing_token=41").Scan(&count))
	require.Equal(t, 1, count)
	tx, err := gateA.BeginTx(ctx, pgx.TxOptions{})
	require.ErrorIs(t, err, ErrRuntimeFenced)
	require.Nil(t, tx)
	require.Error(t, a.CloseAfterWorkers(ctx))
	require.NoError(t, b.Check(ctx))
	gateB, err := NewRuntimeWriteGate(db.Pool, b.RuntimeToken())
	require.NoError(t, err)
	tx, err = gateB.BeginTx(ctx, pgx.TxOptions{})
	require.NoError(t, err)
	require.NoError(t, tx.Commit(ctx))
}

func TestRuntimeSecondInstanceFailsBeforeAdvancingTokens(t *testing.T) {
	db := pgtest.New(t, accountstatemigrations.FS, accountstatemigrations.Dir)
	ctx := context.Background()
	a, err := NewSQLStore(db.Pool).AcquireRuntimeSession(ctx)
	require.NoError(t, err)
	defer a.CloseAfterWorkers(ctx)
	b, err := NewSQLStore(db.Pool).AcquireRuntimeSession(ctx)
	require.Error(t, err)
	require.Nil(t, b)
	require.NoError(t, a.Check(ctx))
	var generation, collector int64
	require.NoError(t, db.Pool.QueryRow(ctx, "SELECT r.generation,c.fencing_token FROM trader_sync_runtime_control r CROSS JOIN trader_sync_collector_control c").Scan(&generation, &collector))
	require.Equal(t, int64(1), generation)
	require.Equal(t, int64(1), collector)
}

func TestRuntimeFailedReplacementRollsBackBothTokensAndEpoch(t *testing.T) {
	db := pgtest.New(t, accountstatemigrations.FS, accountstatemigrations.Dir)
	ctx := context.Background()
	_, err := db.Pool.Exec(ctx, `UPDATE trader_sync_runtime_control SET generation=7; INSERT INTO trader_sync_collector_epochs(id,fencing_token) VALUES(1,9223372036854775807); UPDATE trader_sync_collector_control SET fencing_token=9223372036854775807,active_epoch=1`)
	require.NoError(t, err)
	a, err := NewSQLStore(db.Pool).AcquireRuntimeSession(ctx)
	require.Error(t, err)
	require.Nil(t, a)
	var generation, collector int64
	var ended bool
	require.NoError(t, db.Pool.QueryRow(ctx, "SELECT r.generation,c.fencing_token,e.ended_at IS NOT NULL FROM trader_sync_runtime_control r CROSS JOIN trader_sync_collector_control c CROSS JOIN trader_sync_collector_epochs e").Scan(&generation, &collector, &ended))
	require.Equal(t, int64(7), generation)
	require.Equal(t, int64(math.MaxInt64), collector)
	require.False(t, ended)
	_, err = db.Pool.Exec(ctx, `UPDATE trader_sync_runtime_control SET generation=9223372036854775807; UPDATE trader_sync_collector_control SET fencing_token=3`)
	require.NoError(t, err)
	a, err = NewSQLStore(db.Pool).AcquireRuntimeSession(ctx)
	require.Error(t, err)
	require.Nil(t, a)
	require.NoError(t, db.Pool.QueryRow(ctx, "SELECT r.generation,c.fencing_token FROM trader_sync_runtime_control r CROSS JOIN trader_sync_collector_control c").Scan(&generation, &collector))
	require.Equal(t, int64(math.MaxInt64), generation)
	require.Equal(t, int64(3), collector)
	// Failure also gives back advisory ownership rather than leaking it in the pool.
	_, err = db.Pool.Exec(ctx, "UPDATE trader_sync_runtime_control SET generation=7")
	require.NoError(t, err)
	a, err = NewSQLStore(db.Pool).AcquireRuntimeSession(ctx)
	require.NoError(t, err)
	require.NoError(t, a.CloseAfterWorkers(ctx))
}

func TestRuntimeGateRejectsInvalidOwnership(t *testing.T) {
	db := pgtest.New(t, accountstatemigrations.FS, accountstatemigrations.Dir)
	for _, token := range []RuntimeToken{
		{}, {OwnerID: uuid.New()}, {Generation: 1}, {OwnerID: uuid.New(), Generation: math.MaxUint64},
	} {
		gate, err := NewRuntimeWriteGate(db.Pool, token)
		require.Error(t, err)
		require.Nil(t, gate)
	}
	gate, err := NewRuntimeWriteGate(nil, RuntimeToken{OwnerID: uuid.New(), Generation: 1})
	require.Error(t, err)
	require.Nil(t, gate)
}

func TestRuntimeStaleCloseCannotClearSuccessorOnLiveConnection(t *testing.T) {
	db := pgtest.New(t, accountstatemigrations.FS, accountstatemigrations.Dir)
	ctx := context.Background()
	s := NewSQLStore(db.Pool)
	a, err := s.AcquireRuntimeSession(ctx)
	require.NoError(t, err)
	defer a.CloseAfterWorkers(ctx)
	// Simulate a session losing its advisory lock while the connection survives.
	_, err = a.conn.Exec(ctx, "SELECT pg_advisory_unlock(hashtextextended($1,0))", collectorLock)
	require.NoError(t, err)
	require.ErrorIs(t, a.Check(ctx), ErrRuntimeFenced)
	b, err := s.AcquireRuntimeSession(ctx)
	require.NoError(t, err)
	defer b.CloseAfterWorkers(ctx)
	epoch, err := s.StartCollectorEpoch(ctx, b.CollectorToken())
	require.NoError(t, err)
	require.ErrorIs(t, a.CloseAfterWorkers(ctx), ErrRuntimeFenced)
	require.NoError(t, a.CloseAfterWorkers(ctx))
	require.NoError(t, b.Check(ctx))
	var ended bool
	require.NoError(t, db.Pool.QueryRow(ctx, "SELECT ended_at IS NOT NULL FROM trader_sync_collector_epochs WHERE id=$1", epoch).Scan(&ended))
	require.False(t, ended)
	gate, err := NewRuntimeWriteGate(db.Pool, b.RuntimeToken())
	require.NoError(t, err)
	require.NoError(t, b.CloseAfterWorkers(ctx))
	tx, err := gate.BeginTx(ctx, pgx.TxOptions{})
	require.ErrorIs(t, err, ErrRuntimeFenced)
	require.Nil(t, tx)
	require.ErrorIs(t, b.Check(ctx), ErrRuntimeFenced)
}

func TestRuntimeSharePrecedesAccountGate(t *testing.T) {
	db := pgtest.New(t, accountstatemigrations.FS, accountstatemigrations.Dir)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	owner, err := NewSQLStore(db.Pool).AcquireRuntimeSession(ctx)
	require.NoError(t, err)
	defer owner.CloseAfterWorkers(context.Background())
	gate, err := NewRuntimeWriteGate(db.Pool, owner.RuntimeToken())
	require.NoError(t, err)
	accountID := uuid.NewString()
	blocker, err := txgate.AcquireAccountSession(ctx, db.Pool, accountID)
	require.NoError(t, err)
	defer blocker.Release(context.Background())
	done := make(chan error, 1)
	go func() { done <- txgate.WithAccountTx(ctx, gate, accountID, func(pgx.Tx) error { return nil }) }()
	require.Eventually(t, func() bool {
		var waiting bool
		err := db.Pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM pg_stat_activity a WHERE a.datname=current_database() AND a.wait_event_type='Lock' AND a.query LIKE '%pg_advisory_xact_lock%' AND EXISTS(SELECT 1 FROM pg_locks l WHERE l.pid=a.pid AND l.relation='trader_sync_runtime_control'::regclass AND l.mode='RowShareLock' AND l.granted))`).Scan(&waiting)
		return err == nil && waiting
	}, 3*time.Second, 10*time.Millisecond)
	// The waiting account transaction must already prevent runtime takeover.
	probe, err := db.Pool.Begin(ctx)
	require.NoError(t, err)
	_, err = probe.Exec(ctx, "SELECT * FROM trader_sync_runtime_control FOR UPDATE NOWAIT")
	require.Error(t, err)
	require.NoError(t, probe.Rollback(ctx))
	require.NoError(t, blocker.Release(ctx))
	select {
	case err := <-done:
		require.NoError(t, err)
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
}

func TestRuntimeGateAllowsConcurrentCurrentTransactions(t *testing.T) {
	db := pgtest.New(t, accountstatemigrations.FS, accountstatemigrations.Dir)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	owner, err := NewSQLStore(db.Pool).AcquireRuntimeSession(ctx)
	require.NoError(t, err)
	defer owner.CloseAfterWorkers(context.Background())
	gate, err := NewRuntimeWriteGate(db.Pool, owner.RuntimeToken())
	require.NoError(t, err)
	first, err := gate.BeginTx(ctx, pgx.TxOptions{})
	require.NoError(t, err)
	defer first.Rollback(context.Background())
	second, err := gate.BeginTx(ctx, pgx.TxOptions{})
	require.NoError(t, err)
	require.NoError(t, second.Commit(ctx))
	require.NoError(t, first.Commit(ctx))
}
