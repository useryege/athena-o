//go:build integration

package store

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
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

func TestIntakeFirstSequenceFreezesOriginalCandidatesAndRemovedEvidence(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	ctx := context.Background()
	s := NewSQLStore(db.Pool)
	owner, err := accountstore.NewSQLStore(db.Pool).EnsureDevelopmentAccount(ctx, accountcredentials.DevelopmentRoleMember)
	if err != nil {
		t.Fatal(err)
	}
	wallet := common.HexToAddress("0x1111111111111111111111111111111111111111")
	var subID string
	if err = db.Pool.QueryRow(ctx, `INSERT INTO trader_sync_subscriptions(owner_id,wallet,desired_state,observation_state,target_display) VALUES($1,$2,'enabled','pending_baseline','{"displayName":{"availability":"unavailable","reasonCode":"fixture_not_queried","source":"fixture"},"avatar":{"availability":"unavailable","reasonCode":"fixture_not_queried","source":"fixture"},"profileURL":{"availability":"unavailable","reasonCode":"fixture_not_queried","source":"fixture"}}'::jsonb) RETURNING id`, owner.ID, wallet.Bytes()).Scan(&subID); err != nil {
		t.Fatal(err)
	}
	session, err := s.AcquireRuntimeSession(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer session.CloseAfterWorkers(ctx)
	epoch, err := s.StartCollectorEpoch(ctx, session.CollectorToken())
	if err != nil {
		t.Fatal(err)
	}
	sub := tm.Subscription{ID: subID, OwnerID: owner.ID, Wallet: wallet, Generation: 1, Revision: 1, DesiredState: "enabled"}
	err = txgate.WithAccountTx(ctx, db.Pool, owner.ID, func(tx pgx.Tx) error {
		if e := txgate.LockWallet(ctx, tx, wallet); e != nil {
			return e
		}
		return s.RegisterBaselineTx(ctx, tx, sub, session.CollectorToken(), epoch, tm.WalletObservation{High: 100, Sequence: 10})
	})
	if err != nil {
		t.Fatal(err)
	}
	raw := ethtypes.Log{Address: common.HexToAddress("0xe111180000d2663c0091e4f400237545b87b996b"), Topics: []common.Hash{common.HexToHash("0xd543adfd945773f1a62f74f0ee55a5e3b9b1a28262980ba90b1a89f2ea84d8ee"), {}, common.BytesToHash(wallet.Bytes()), {}}, Data: make([]byte, 224), BlockNumber: 101, BlockHash: common.HexToHash("0xaa"), TxHash: common.HexToHash("0xbb"), Index: 2}
	received := tm.ReceivedLog{Raw: raw, Sequence: 10, ReceivedAt: time.Now().UTC()}
	if err = s.PersistReceived(ctx, session.CollectorToken(), epoch, received); err != nil {
		t.Fatal(err)
	}
	count := func() int {
		var n int
		if e := db.Pool.QueryRow(ctx, `SELECT count(*) FROM trader_sync_source_candidates`).Scan(&n); e != nil {
			t.Fatal(e)
		}
		return n
	}
	if n := count(); n != 0 {
		t.Fatal("queued pre-registration log acquired new owner", n)
	}
	received.Sequence = 11 // A duplicate delivered after registration must not create candidates.
	if err = s.PersistReceived(ctx, session.CollectorToken(), epoch, received); err != nil {
		t.Fatal(err)
	}
	if n := count(); n != 0 {
		t.Fatal("duplicate gained candidates", n)
	}
	received.Raw.Index++
	received.Sequence++
	if err = s.PersistReceived(ctx, session.CollectorToken(), epoch, received); err != nil {
		t.Fatal(err)
	}
	if n := count(); n != 1 {
		t.Fatal("new received fact missing original attempt", n)
	}
	original, _ := json.Marshal(received.Raw)
	received.Raw.Removed = true
	received.Sequence++
	if err = s.PersistReceived(ctx, session.CollectorToken(), epoch, received); err != nil {
		t.Fatal(err)
	}
	var stored []byte
	var removed bool
	if err = db.Pool.QueryRow(ctx, `SELECT raw_json,removed FROM trader_sync_source_records WHERE log_index=$1`, received.Raw.Index).Scan(&stored, &removed); err != nil {
		t.Fatal(err)
	}
	var first, got ethtypes.Log
	_ = json.Unmarshal(original, &first)
	_ = json.Unmarshal(stored, &got)
	if !removed || got.Removed || got.Index != first.Index || count() != 1 {
		t.Fatal("removed rewrote original fact or candidates", removed, got)
	}
	if err = s.CloseCollectorEpoch(ctx, session.CollectorToken(), epoch, "test interruption"); err != nil {
		t.Fatal(err)
	}
	received.Raw.Index++
	if err = s.PersistReceived(ctx, session.CollectorToken(), epoch, received); err == nil {
		t.Fatal("closed epoch accepted write")
	}
}

func TestIntakeRemovedFirstAndCounterpartyPushNeverBindNewAttempt(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	ctx := context.Background()
	s := NewSQLStore(db.Pool)
	owner, err := accountstore.NewSQLStore(db.Pool).EnsureDevelopmentAccount(ctx, accountcredentials.DevelopmentRoleMember)
	if err != nil {
		t.Fatal(err)
	}
	wallet := common.HexToAddress("0x1111111111111111111111111111111111111111")
	var subID string
	if err = db.Pool.QueryRow(ctx, `INSERT INTO trader_sync_subscriptions(owner_id,wallet,desired_state,observation_state,target_display) VALUES($1,$2,'enabled','pending_baseline','{"displayName":{"availability":"unavailable","reasonCode":"fixture_not_queried","source":"fixture"},"avatar":{"availability":"unavailable","reasonCode":"fixture_not_queried","source":"fixture"},"profileURL":{"availability":"unavailable","reasonCode":"fixture_not_queried","source":"fixture"}}'::jsonb)RETURNING id`, owner.ID, wallet.Bytes()).Scan(&subID); err != nil {
		t.Fatal(err)
	}
	ownerSession, err := s.AcquireRuntimeSession(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer ownerSession.CloseAfterWorkers(ctx)
	epoch, err := s.StartCollectorEpoch(ctx, ownerSession.CollectorToken())
	if err != nil {
		t.Fatal(err)
	}
	err = txgate.WithAccountTx(ctx, db.Pool, owner.ID, func(tx pgx.Tx) error {
		if e := txgate.LockWallet(ctx, tx, wallet); e != nil {
			return e
		}
		return s.RegisterBaselineTx(ctx, tx, tm.Subscription{ID: subID, OwnerID: owner.ID, Wallet: wallet, Generation: 1, Revision: 1}, ownerSession.CollectorToken(), epoch, tm.WalletObservation{})
	})
	if err != nil {
		t.Fatal(err)
	}
	raw := ethtypes.Log{Address: common.HexToAddress("0xe111180000d2663c0091e4f400237545b87b996b"), Topics: []common.Hash{{}, {}, common.BytesToHash(wallet.Bytes()), {}}, Data: make([]byte, 224), BlockNumber: 1, BlockHash: common.HexToHash("0xaa"), TxHash: common.HexToHash("0xbb"), Removed: true}
	if err = s.PersistReceived(ctx, ownerSession.CollectorToken(), epoch, tm.ReceivedLog{Raw: raw, ReceivedAt: time.Now(), Sequence: 1}); err != nil {
		t.Fatal(err)
	}
	var n int
	if err = db.Pool.QueryRow(ctx, `SELECT count(*) FROM trader_sync_source_candidates`).Scan(&n); err != nil || n != 0 {
		t.Fatal("removed-only fact gained new attempt", n, err)
	}
	raw.Index++
	raw.Removed = false
	raw.Topics[3] = raw.Topics[2]
	raw.Topics[2] = common.BytesToHash(common.HexToAddress("0x2222222222222222222222222222222222222222").Bytes())
	if err = s.PersistReceived(ctx, ownerSession.CollectorToken(), epoch, tm.ReceivedLog{Raw: raw, ReceivedAt: time.Now(), Sequence: 2}); err != nil {
		t.Fatal(err)
	}
	if err = db.Pool.QueryRow(ctx, `SELECT count(*) FROM trader_sync_source_records`).Scan(&n); err != nil || n != 1 {
		t.Fatal("counterparty-only push registered an unobserved wallet", n, err)
	}
}

func TestIntakeWaitingWalletCannotCrossClosedEpochFence(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	ctx := context.Background()
	s := NewSQLStore(db.Pool)
	session, err := s.AcquireRuntimeSession(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer session.CloseAfterWorkers(ctx)
	epoch, err := s.StartCollectorEpoch(ctx, session.CollectorToken())
	if err != nil {
		t.Fatal(err)
	}
	wallet := common.HexToAddress("0x1111111111111111111111111111111111111111")
	blocker, err := db.Pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer blocker.Rollback(ctx)
	if err = txgate.LockWallet(ctx, blocker, wallet); err != nil {
		t.Fatal(err)
	}
	raw := ethtypes.Log{Address: common.HexToAddress("0xe111180000d2663c0091e4f400237545b87b996b"), Topics: []common.Hash{{}, {}, common.BytesToHash(wallet.Bytes()), {}}, Data: make([]byte, 224), BlockNumber: 1, BlockHash: common.HexToHash("0xaa"), TxHash: common.HexToHash("0xbb")}
	persisted := make(chan error, 1)
	go func() {
		persisted <- s.PersistReceived(ctx, session.CollectorToken(), epoch, tm.ReceivedLog{Raw: raw, ReceivedAt: time.Now(), Sequence: 1})
	}()
	var waiting bool
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if err = db.Pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM pg_locks WHERE locktype='advisory' AND NOT granted AND database=(SELECT oid FROM pg_database WHERE datname=current_database()) AND objid=(hashtextextended('athena:wallet:' || $1::text,0)&4294967295)::oid)`, wallet.Hex()).Scan(&waiting); err != nil {
			t.Fatal(err)
		}
		if waiting {
			break
		}
		time.Sleep(time.Millisecond)
	}
	if !waiting {
		t.Fatal("intake did not reach wallet wait")
	}
	ended := make(chan error, 1)
	go func() { ended <- s.CloseCollectorEpoch(ctx, session.CollectorToken(), epoch, "test disconnect") }()
	select {
	case err = <-ended:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("epoch close waited for wallet: lock order inverted")
	}
	if err = blocker.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	select {
	case err = <-persisted:
		if !errors.Is(err, ErrCollectorFenced) {
			t.Fatal("stale intake crossed epoch close", err)
		}
	case <-time.After(time.Second):
		t.Fatal("stale intake did not finish")
	}
}

func TestIntakeRemovedAfterLastTargetDisabledPreservesOriginalFact(t *testing.T) {
	for _, state := range []string{"paused", "cancelled", "permission_disabled"} {
		t.Run(state, func(t *testing.T) {
			db := pgtest.New(t, migrations.FS, migrations.Dir)
			ctx := context.Background()
			s := NewSQLStore(db.Pool)
			owner, err := accountstore.NewSQLStore(db.Pool).EnsureDevelopmentAccount(ctx, accountcredentials.DevelopmentRoleMember)
			if err != nil {
				t.Fatal(err)
			}
			wallet := common.HexToAddress("0x1111111111111111111111111111111111111111")
			var sub string
			if err = db.Pool.QueryRow(ctx, `INSERT INTO trader_sync_subscriptions(owner_id,wallet,desired_state,observation_state,target_display) VALUES($1,$2,'enabled','pending_baseline','{"displayName":{"availability":"unavailable","reasonCode":"fixture_not_queried","source":"fixture"},"avatar":{"availability":"unavailable","reasonCode":"fixture_not_queried","source":"fixture"},"profileURL":{"availability":"unavailable","reasonCode":"fixture_not_queried","source":"fixture"}}'::jsonb)RETURNING id`, owner.ID, wallet.Bytes()).Scan(&sub); err != nil {
				t.Fatal(err)
			}
			session, err := s.AcquireRuntimeSession(ctx)
			if err != nil {
				t.Fatal(err)
			}
			defer session.CloseAfterWorkers(ctx)
			epoch, err := s.StartCollectorEpoch(ctx, session.CollectorToken())
			if err != nil {
				t.Fatal(err)
			}
			err = txgate.WithAccountTx(ctx, db.Pool, owner.ID, func(tx pgx.Tx) error {
				if e := txgate.LockWallet(ctx, tx, wallet); e != nil {
					return e
				}
				return s.RegisterBaselineTx(ctx, tx, tm.Subscription{ID: sub, OwnerID: owner.ID, Wallet: wallet, Revision: 1, Generation: 1}, session.CollectorToken(), epoch, tm.WalletObservation{})
			})
			if err != nil {
				t.Fatal(err)
			}
			raw := ethtypes.Log{Address: common.HexToAddress("0xe111180000d2663c0091e4f400237545b87b996b"), Topics: []common.Hash{{}, {}, common.BytesToHash(wallet.Bytes()), {}}, Data: make([]byte, 224), BlockNumber: 10, BlockHash: common.HexToHash("0xaa"), TxHash: common.HexToHash("0xbb"), Index: 1}
			first := time.Now().UTC().Truncate(time.Microsecond)
			if err = s.PersistReceived(ctx, session.CollectorToken(), epoch, tm.ReceivedLog{Raw: raw, ReceivedAt: first, Sequence: 1}); err != nil {
				t.Fatal(err)
			}
			if _, err = db.Pool.Exec(ctx, `UPDATE trader_sync_source_records SET confirmation_state='confirmed'`); err != nil {
				t.Fatal(err)
			}
			err = txgate.WithAccountTx(ctx, db.Pool, owner.ID, func(tx pgx.Tx) error {
				_, e := tx.Exec(ctx, `UPDATE trader_sync_subscriptions SET desired_state=$2,revision=revision+1 WHERE id=$1`, sub, state)
				return e
			})
			if err != nil {
				t.Fatal(err)
			}
			raw.Removed = true
			if err = s.PersistReceived(ctx, session.CollectorToken(), epoch, tm.ReceivedLog{Raw: raw, ReceivedAt: first.Add(time.Second), Sequence: 2}); err != nil {
				t.Fatal(err)
			}
			var removed bool
			var confirmation, reason string
			var original []byte
			var received time.Time
			var sequence int64
			if err = db.Pool.QueryRow(ctx, `SELECT removed,confirmation_state,confirmation_reason,raw_json,received_at,read_sequence FROM trader_sync_source_records`).Scan(&removed, &confirmation, &reason, &original, &received, &sequence); err != nil {
				t.Fatal(err)
			}
			var persisted ethtypes.Log
			_ = json.Unmarshal(original, &persisted)
			if !removed || confirmation != "invalid" || reason != "removed" {
				t.Fatal("disabled target lost received removed evidence", removed, confirmation, reason)
			}
			if persisted.Removed || !received.Equal(first) || sequence != 1 {
				t.Fatal("removed rewrote original receive", persisted.Removed, received, sequence)
			}
			raw.Index++
			raw.Removed = false
			if err = s.PersistReceived(ctx, session.CollectorToken(), epoch, tm.ReceivedLog{Raw: raw, ReceivedAt: first.Add(2 * time.Second), Sequence: 3}); err != nil {
				t.Fatal(err)
			}
			var records, candidates int
			_ = db.Pool.QueryRow(ctx, `SELECT count(*) FROM trader_sync_source_records`).Scan(&records)
			_ = db.Pool.QueryRow(ctx, `SELECT count(*) FROM trader_sync_source_candidates`).Scan(&candidates)
			if records != 1 || candidates != 1 {
				t.Fatal("disabled target gained first source/candidate", records, candidates)
			}
		})
	}
}
