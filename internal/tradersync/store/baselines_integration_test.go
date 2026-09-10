//go:build integration

package store

import (
	"context"
	"errors"
	"github.com/ethereum/go-ethereum/common"
	"github.com/jackc/pgx/v5"
	"github.com/useryege/athena/internal/accountcredentials"
	accountstore "github.com/useryege/athena/internal/accountstate/store"
	"github.com/useryege/athena/internal/accountstate/store/migrations"
	"github.com/useryege/athena/internal/accountstate/txgate"
	"github.com/useryege/athena/internal/testutil/pgtest"
	tm "github.com/useryege/athena/internal/tradersync/types"
	"testing"
	"time"
)

func TestBaselineOwnershipIsExclusiveAndTokenPersistsAcrossInstances(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	ctx := context.Background()
	s := NewSQLStore(db.Pool)
	first, err := s.AcquireCollectorSession(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer first.Close(ctx)
	if second, err := s.AcquireCollectorSession(ctx); err == nil {
		_ = second.Close(ctx)
		t.Fatal("two collectors own same database")
	}
	epoch, err := s.StartCollectorEpoch(ctx, first.Token)
	if err != nil {
		t.Fatal(err)
	}
	if err = first.Check(ctx); err != nil {
		t.Fatal(err)
	}
	if err = first.Close(ctx); err != nil {
		t.Fatal(err)
	}
	next, err := s.AcquireCollectorSession(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer next.Close(ctx)
	if next.Token <= first.Token {
		t.Fatal("fencing token did not advance")
	}
	if _, err = s.StartCollectorEpoch(ctx, first.Token); err == nil {
		t.Fatal("old instance wrote after replacement")
	}
	var ended bool
	if err = db.Pool.QueryRow(ctx, `SELECT ended_at IS NOT NULL FROM trader_sync_collector_epochs WHERE id=$1`, epoch).Scan(&ended); err != nil || !ended {
		t.Fatal("lost epoch not closed", ended, err)
	}
}

func TestBaselineBoundaryRequiresACKFutureTimeAndCurrentIntent(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	ctx := context.Background()
	s := NewSQLStore(db.Pool)
	owner, err := accountstore.NewSQLStore(db.Pool).EnsureDevelopmentAccount(ctx, accountcredentials.DevelopmentRoleMember)
	if err != nil {
		t.Fatal(err)
	}
	session, err := s.AcquireCollectorSession(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close(ctx)
	epoch, err := s.StartCollectorEpoch(ctx, session.Token)
	if err != nil {
		t.Fatal(err)
	}
	register := func(wallet string) string {
		t.Helper()
		var id string
		address := common.HexToAddress(wallet)
		if err := db.Pool.QueryRow(ctx, `INSERT INTO trader_sync_subscriptions(owner_id,wallet,desired_state,observation_state)VALUES($1,$2,'enabled','pending_baseline')RETURNING id`, owner.ID, address.Bytes()).Scan(&id); err != nil {
			t.Fatal(err)
		}
		err := txgate.WithAccountTx(ctx, db.Pool, owner.ID, func(tx pgx.Tx) error {
			if err := txgate.LockWallet(ctx, tx, address); err != nil {
				return err
			}
			return s.RegisterBaselineTx(ctx, tx, tm.Subscription{ID: id, OwnerID: owner.ID, Wallet: address, Generation: 1, Revision: 1}, session.Token, epoch, tm.WalletObservation{High: 20, Sequence: 5})
		})
		if err != nil {
			t.Fatal(err)
		}
		return id
	}
	first := register("0x1111111111111111111111111111111111111111")
	pending, err := s.PendingBaselines(ctx)
	if err != nil || len(pending) != 1 {
		t.Fatal("committed attempt not visible", pending, err)
	}
	attempt := pending[0].ID
	future := time.Now().Add(300 * time.Millisecond)
	if err = s.SaveBaselineBoundary(ctx, session.Token, attempt, 1, future); err == nil {
		t.Fatal("accepted unacknowledged filter")
	}
	if err = s.AckFilters(ctx, session.Token, epoch, 1); err != nil {
		t.Fatal(err)
	}
	if err = s.SaveBaselineBoundary(ctx, session.Token, attempt, 1, time.Now().Add(-time.Second)); err == nil {
		t.Fatal("saved expired boundary")
	}
	future = time.Now().Add(300 * time.Millisecond)
	if err = s.SaveBaselineBoundary(ctx, session.Token, attempt, 1, future); err != nil {
		t.Fatal(err)
	}
	if err = s.CompleteBaseline(ctx, session.Token, attempt); err == nil {
		t.Fatal("succeeded before boundary")
	}
	time.Sleep(time.Until(future) + 10*time.Millisecond)
	if err = s.CompleteBaseline(ctx, session.Token, attempt); err != nil {
		t.Fatal(err)
	}
	if err = s.CompleteBaseline(ctx, session.Token, attempt); err != nil {
		t.Fatal("idempotent success read failed", err)
	}
	var intervals int
	var effective time.Time
	if err = db.Pool.QueryRow(ctx, `SELECT count(*),min(effective_at) FROM trader_sync_monitor_intervals WHERE subscription_id=$1`, first).Scan(&intervals, &effective); err != nil || intervals != 1 || effective.Sub(future).Abs() > time.Microsecond {
		t.Fatal(intervals, effective, future, err)
	}
	second := register("0x2222222222222222222222222222222222222222")
	pending, err = s.PendingBaselines(ctx)
	if err != nil || len(pending) != 1 {
		t.Fatal(pending, err)
	}
	future = time.Now().Add(100 * time.Millisecond)
	if err = s.SaveBaselineBoundary(ctx, session.Token, pending[0].ID, 1, future); err != nil {
		t.Fatal(err)
	}
	err = txgate.WithAccountTx(ctx, db.Pool, owner.ID, func(tx pgx.Tx) error {
		_, e := tx.Exec(ctx, `UPDATE trader_sync_subscriptions SET desired_state='paused',revision=revision+1 WHERE id=$1`, second)
		return e
	})
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Until(future) + time.Millisecond)
	if err = s.CompleteBaseline(ctx, session.Token, pending[0].ID); err == nil {
		t.Fatal("stale revision became healthy")
	}
}

func TestBaselineOwnershipSnapshotSerializesRegistrationAndPreservesNewPending(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	ctx := context.Background()
	s := NewSQLStore(db.Pool)
	owner, err := accountstore.NewSQLStore(db.Pool).EnsureDevelopmentAccount(ctx, accountcredentials.DevelopmentRoleMember)
	if err != nil {
		t.Fatal(err)
	}
	insert := func(tx pgx.Tx, wallet string) (string, string) {
		t.Helper()
		address := common.HexToAddress(wallet)
		var id string
		if err := tx.QueryRow(ctx, `INSERT INTO trader_sync_subscriptions(owner_id,wallet,desired_state,observation_state)VALUES($1,$2,'enabled','pending_baseline')RETURNING id`, owner.ID, address.Bytes()).Scan(&id); err != nil {
			t.Fatal(err)
		}
		if err := txgate.LockWallet(ctx, tx, address); err != nil {
			t.Fatal(err)
		}
		if err := s.RegisterBaselineTx(ctx, tx, tm.Subscription{ID: id, OwnerID: owner.ID, Wallet: address, Generation: 1, Revision: 1}, 0, 0, tm.WalletObservation{}); err != nil {
			t.Fatal(err)
		}
		var attempt string
		if err := tx.QueryRow(ctx, `SELECT id FROM trader_sync_baseline_attempts WHERE subscription_id=$1`, id).Scan(&attempt); err != nil {
			t.Fatal(err)
		}
		return id, attempt
	}
	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended('athena:account:' || $1::text,0))`, owner.ID); err != nil {
		t.Fatal(err)
	}
	oldSub, oldID := insert(tx, "0x1111111111111111111111111111111111111111")
	enabledSub, enabledOldID := insert(tx, "0x3333333333333333333333333333333333333333")
	acquired := make(chan *CollectorSession, 1)
	acquireError := make(chan error, 1)
	go func() {
		session, e := s.AcquireCollectorSession(ctx)
		if e != nil {
			acquireError <- e
		} else {
			acquired <- session
		}
	}()
	select {
	case session := <-acquired:
		session.Close(ctx)
		t.Fatal("ownership passed uncommitted registration control lock")
	case e := <-acquireError:
		t.Fatal(e)
	case <-time.After(100 * time.Millisecond):
	}
	if err = tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	var session *CollectorSession
	select {
	case session = <-acquired:
	case e := <-acquireError:
		t.Fatal(e)
	case <-time.After(2 * time.Second):
		t.Fatal("ownership remained blocked")
	}
	defer session.Close(ctx)
	var newID string
	err = txgate.WithAccountTx(ctx, db.Pool, owner.ID, func(tx pgx.Tx) error { _, newID = insert(tx, "0x2222222222222222222222222222222222222222"); return nil })
	if err != nil {
		t.Fatal(err)
	}
	// Pausing between the snapshot and cleanup must remain paused after recovery.
	err = txgate.WithAccountTx(ctx, db.Pool, owner.ID, func(tx pgx.Tx) error {
		_, e := tx.Exec(ctx, `UPDATE trader_sync_subscriptions SET desired_state='paused',revision=revision+1 WHERE id=$1`, oldSub)
		return e
	})
	if err != nil {
		t.Fatal(err)
	}
	if err = session.RecoverPending(ctx); err != nil {
		t.Fatal(err)
	}
	var oldState, newState, desired string
	if err = db.Pool.QueryRow(ctx, `SELECT state FROM trader_sync_baseline_attempts WHERE id=$1`, oldID).Scan(&oldState); err != nil {
		t.Fatal(err)
	}
	if err = db.Pool.QueryRow(ctx, `SELECT state FROM trader_sync_baseline_attempts WHERE id=$1`, newID).Scan(&newState); err != nil {
		t.Fatal(err)
	}
	if err = db.Pool.QueryRow(ctx, `SELECT desired_state FROM trader_sync_subscriptions WHERE id=$1`, oldSub).Scan(&desired); err != nil {
		t.Fatal(err)
	}
	if oldState != "failed" || newState != "pending" || desired != "paused" {
		t.Fatal("ownership snapshot affected wrong intent", oldState, newState, desired)
	}
	needed, err := s.SubscriptionsNeedingBaseline(ctx)
	if err != nil || len(needed) != 1 || needed[0].ID != enabledSub {
		t.Fatal("restart did not request only the enabled replacement", needed, err)
	}
	err = s.RegisterNeededBaseline(ctx, needed[0], func(ctx context.Context, tx pgx.Tx, sub tm.Subscription) error {
		if e := txgate.LockWallet(ctx, tx, sub.Wallet); e != nil {
			return e
		}
		return s.RegisterBaselineTx(ctx, tx, sub, session.Token, 0, tm.WalletObservation{})
	})
	if err != nil {
		t.Fatal(err)
	}
	var replacement string
	if err = db.Pool.QueryRow(ctx, `SELECT id FROM trader_sync_baseline_attempts WHERE subscription_id=$1 AND state='pending'`, enabledSub).Scan(&replacement); err != nil || replacement == enabledOldID {
		t.Fatal("reused failed preparation attempt", replacement, enabledOldID, err)
	}

}

type baselineCommitReplyLost struct {
	txgate.Beginner
	calls    int
	rollback bool
}
type baselineCommittedTx struct {
	pgx.Tx
	parent *baselineCommitReplyLost
}

func (b *baselineCommitReplyLost) BeginTx(ctx context.Context, options pgx.TxOptions) (pgx.Tx, error) {
	tx, err := b.Beginner.BeginTx(ctx, options)
	if err != nil {
		return nil, err
	}
	return &baselineCommittedTx{Tx: tx, parent: b}, nil
}
func (tx *baselineCommittedTx) Commit(ctx context.Context) error {
	if tx.parent.rollback {
		return errors.Join(errors.New("injected unknown rolled-back COMMIT"), tx.Tx.Rollback(ctx))
	}
	if err := tx.Tx.Commit(ctx); err != nil {
		return err
	}
	tx.parent.calls++
	return errors.New("injected lost successful COMMIT reply")
}

func TestBaselineUnknownCommitReadsPersistedSuccessBeforeAnyRetry(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	ctx := context.Background()
	s := NewSQLStore(db.Pool)
	owner, err := accountstore.NewSQLStore(db.Pool).EnsureDevelopmentAccount(ctx, accountcredentials.DevelopmentRoleMember)
	if err != nil {
		t.Fatal(err)
	}
	session, err := s.AcquireCollectorSession(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close(ctx)
	epoch, err := s.StartCollectorEpoch(ctx, session.Token)
	if err != nil {
		t.Fatal(err)
	}
	wallet := common.HexToAddress("0x1111111111111111111111111111111111111111")
	var subID, attempt string
	if err = db.Pool.QueryRow(ctx, `INSERT INTO trader_sync_subscriptions(owner_id,wallet,desired_state,observation_state)VALUES($1,$2,'enabled','pending_baseline')RETURNING id`, owner.ID, wallet.Bytes()).Scan(&subID); err != nil {
		t.Fatal(err)
	}
	err = txgate.WithAccountTx(ctx, db.Pool, owner.ID, func(tx pgx.Tx) error {
		if e := txgate.LockWallet(ctx, tx, wallet); e != nil {
			return e
		}
		return s.RegisterBaselineTx(ctx, tx, tm.Subscription{ID: subID, OwnerID: owner.ID, Wallet: wallet, Revision: 1, Generation: 1}, session.Token, epoch, tm.WalletObservation{})
	})
	if err != nil {
		t.Fatal(err)
	}
	if err = db.Pool.QueryRow(ctx, `SELECT id FROM trader_sync_baseline_attempts WHERE subscription_id=$1`, subID).Scan(&attempt); err != nil {
		t.Fatal(err)
	}
	if err = s.AckFilters(ctx, session.Token, epoch, 1); err != nil {
		t.Fatal(err)
	}
	at := time.Now().Add(20 * time.Millisecond)
	if err = s.SaveBaselineBoundary(ctx, session.Token, attempt, 1, at); err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Until(at) + time.Millisecond)
	lost := &baselineCommitReplyLost{Beginner: db.Pool}
	if err = s.completeBaselineUsing(ctx, lost, session.Token, attempt); err != nil {
		t.Fatal("successful COMMIT treated as failed", err)
	}
	if lost.calls != 1 {
		t.Fatal("test did not lose real COMMIT reply", lost.calls)
	}
	if err = s.CompleteBaseline(ctx, session.Token, attempt); err != nil {
		t.Fatal(err)
	}
	var n int
	if err = db.Pool.QueryRow(ctx, `SELECT count(*) FROM trader_sync_monitor_intervals WHERE baseline_attempt_id=$1`, attempt).Scan(&n); err != nil || n != 1 {
		t.Fatal("unknown reply duplicated interval", n, err)
	}
}

func TestBaselineEpochUnknownCommitRecoversDurableActiveEpoch(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	ctx := context.Background()
	s := NewSQLStore(db.Pool)
	owner, err := s.AcquireCollectorSession(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer owner.Close(ctx)
	lost := &baselineCommitReplyLost{Beginner: db.Pool}
	epoch, err := s.startCollectorEpochUsing(ctx, lost, owner.Token)
	if err != nil || epoch == 0 {
		t.Fatal("committed active epoch stranded after lost reply", epoch, err)
	}
	if lost.calls != 1 {
		t.Fatal("test did not commit and lose response", lost.calls)
	}
	var active int64
	if err = db.Pool.QueryRow(ctx, `SELECT active_epoch FROM trader_sync_collector_control`).Scan(&active); err != nil || active != int64(epoch) {
		t.Fatal("recovery chose wrong epoch", active, epoch, err)
	}
	if err = owner.Check(ctx); err != nil {
		t.Fatal("ownership connection was not independently healthy", err)
	}
	if err = s.CloseCollectorEpoch(ctx, owner.Token, epoch, "test recovered startup"); err != nil {
		t.Fatal(err)
	}
	next, err := s.StartCollectorEpoch(ctx, owner.Token)
	if err != nil || next <= epoch {
		t.Fatal("same owner could not advance after recovered startup", next, err)
	}
}

func TestBaselineEpochUnknownRollbackCannotBecomeActiveEvidence(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	ctx := context.Background()
	s := NewSQLStore(db.Pool)
	owner, err := s.AcquireCollectorSession(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer owner.Close(ctx)
	lost := &baselineCommitReplyLost{Beginner: db.Pool, rollback: true}
	epoch, err := s.startCollectorEpochUsing(ctx, lost, owner.Token)
	if epoch == 0 || err == nil || !errors.Is(err, ErrCollectorFenced) {
		t.Fatal("uncommitted epoch adopted as durable active", epoch, err)
	}
	next, err := s.StartCollectorEpoch(ctx, owner.Token)
	if err != nil || next <= epoch {
		t.Fatal("rolled-back epoch leaked control state", next, err)
	}
}
