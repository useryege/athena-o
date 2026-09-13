//go:build integration

package store

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/useryege/athena/internal/accountaccess"
	"github.com/useryege/athena/internal/accountstate/txgate"
	"github.com/useryege/athena/internal/notification/delivery"
	ns "github.com/useryege/athena/internal/notification/store"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	et "github.com/ethereum/go-ethereum/core/types"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"github.com/useryege/athena/internal/accountcredentials"
	ac "github.com/useryege/athena/internal/accountstate/store"
	"github.com/useryege/athena/internal/accountstate/store/migrations"
	"github.com/useryege/athena/internal/testutil/pgtest"
	tm "github.com/useryege/athena/internal/tradersync/types"
	pm "github.com/useryege/athena/util/polymarket"
)

func runtimeWriteSnapshot(t *testing.T, pool *pgxpool.Pool) string {
	t.Helper()
	var snapshot string
	require.NoError(t, pool.QueryRow(context.Background(), `SELECT jsonb_build_array(
 (SELECT jsonb_agg(to_jsonb(t) ORDER BY id) FROM trader_sync_source_records t),
 (SELECT jsonb_agg(to_jsonb(t) ORDER BY id) FROM trader_sync_activities t),
 (SELECT jsonb_agg(to_jsonb(t) ORDER BY id) FROM trader_sync_baseline_attempts t),
 (SELECT jsonb_agg(to_jsonb(t) ORDER BY id) FROM trader_sync_monitor_intervals t),
 (SELECT jsonb_agg(to_jsonb(t) ORDER BY id) FROM trader_sync_subscriptions t),
 (SELECT jsonb_agg(to_jsonb(t)) FROM trader_sync_directory_refresh t),
 (SELECT jsonb_agg(to_jsonb(t)) FROM trader_sync_market_metadata t),
 (SELECT jsonb_agg(to_jsonb(t)) FROM trader_sync_collector_epochs t),
 (SELECT jsonb_agg(to_jsonb(t)) FROM account_notification_deliveries t)
 )::text`).Scan(&snapshot))
	return snapshot
}

func TestRuntimeWriteRejectsOldProjectionMetadata(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	ctx := context.Background()
	account, err := ac.NewSQLStore(db.Pool).EnsureDevelopmentAccount(ctx, accountcredentials.DevelopmentRoleMember)
	require.NoError(t, err)
	source := activityFixture(t, db.Pool, account.ID, 1)
	base := NewSQLStore(db.Pool)
	a, err := base.AcquireRuntimeSession(ctx)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, a.CloseAfterWorkers(ctx)) })
	stale, err := base.WithRuntime(a.RuntimeToken())
	require.NoError(t, err)
	require.NoError(t, a.CloseAfterWorkers(ctx))
	b, err := base.AcquireRuntimeSession(ctx)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, b.CloseAfterWorkers(ctx)) })
	err = stale.CompleteProjectionMetadata(ctx, source.Candidate.SourceID, true)
	require.ErrorIs(t, err, ErrRuntimeFenced)
	var complete bool
	require.NoError(t, db.Pool.QueryRow(ctx, "SELECT metadata_complete FROM trader_sync_source_records WHERE id=$1", source.Candidate.SourceID).Scan(&complete))
	require.False(t, complete)
}

func TestRuntimeWriteRejectsAllOwnedPathsAfterTakeover(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	ctx := context.Background()
	account, err := ac.NewSQLStore(db.Pool).EnsureDevelopmentAccount(ctx, accountcredentials.DevelopmentRoleMember)
	require.NoError(t, err)
	source := activityFixture(t, db.Pool, account.ID, 1)
	pendingID := uuid.NewString()
	_, err = db.Pool.Exec(ctx, `INSERT INTO trader_sync_baseline_attempts(id,owner_id,subscription_id,activation_generation,expected_revision,state) VALUES($1,$2,$3,1,1,'pending')`, pendingID, account.ID, source.Candidate.SubscriptionID)
	require.NoError(t, err)
	stoppedSub := uuid.NewString()
	_, err = db.Pool.Exec(ctx, `INSERT INTO trader_sync_subscriptions(id,owner_id,wallet,desired_state,observation_state,target_display) VALUES($1,$2,decode(repeat('22',20),'hex'),'enabled','pending_baseline','{}')`, stoppedSub, account.ID)
	require.NoError(t, err)
	_, err = db.Pool.Exec(ctx, `INSERT INTO trader_sync_baseline_attempts(owner_id,subscription_id,activation_generation,expected_revision,state,collector_epoch) VALUES($1,$2,1,1,'pending',1)`, account.ID, stoppedSub)
	require.NoError(t, err)
	_, err = db.Pool.Exec(ctx, `UPDATE trader_sync_collector_epochs SET ended_at=clock_timestamp(),reason='fixture_stopped' WHERE id=1`)
	require.NoError(t, err)
	base := NewSQLStore(db.Pool)
	a, err := base.AcquireRuntimeSession(ctx)
	require.NoError(t, err)
	stale, err := base.WithRuntime(a.RuntimeToken())
	require.NoError(t, err)
	require.NoError(t, stale.ConfigureActivities("https://athena.test"))
	require.NoError(t, a.CloseAfterWorkers(ctx))
	b, err := base.AcquireRuntimeSession(ctx)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, b.CloseAfterWorkers(ctx)) })
	token := a.CollectorToken()
	var raw et.Log
	var data []byte
	require.NoError(t, db.Pool.QueryRow(ctx, `SELECT raw_json FROM trader_sync_source_records WHERE id=$1`, source.Candidate.SourceID).Scan(&data))
	require.NoError(t, json.Unmarshal(data, &raw))
	raw.Topics = []common.Hash{{}, {}, common.BytesToHash(source.Trade.Wallet.Bytes()), {}}
	received := tm.ReceivedLog{Raw: raw, Sequence: 2, ReceivedAt: time.Now()}
	sub := tm.Subscription{ID: source.Candidate.SubscriptionID, OwnerID: account.ID, Wallet: source.Trade.Wallet, Revision: 1, Generation: 1, DesiredState: "enabled"}
	var interval string
	require.NoError(t, db.Pool.QueryRow(ctx, `SELECT id FROM trader_sync_monitor_intervals WHERE subscription_id=$1`, sub.ID).Scan(&interval))
	cases := []struct {
		name string
		call func() error
	}{
		{"intake", func() error { return stale.PersistReceived(ctx, token, 1, received) }},
		{"finality", func() error {
			return stale.RecordFinalityObservation(ctx, source.Candidate.SourceID, tm.FinalityRoundObservation{})
		}},
		{"projection_evidence", func() error {
			return stale.SaveProjectionEvidence(ctx, source.Candidate.SourceID, source.Confirmation, &source.Trade)
		}},
		{"activity_and_delivery", func() error { _, _, e := stale.Project(ctx, source); return e }},
		{"metadata_cache", func() error { return stale.SaveMetadata(ctx, "stale", tm.TradeMetadata{}) }},
		{"directory_admission", func() error {
			_, e := stale.RefreshComboPage(ctx, func(context.Context, string, int) (pm.ComboMarketPage, error) {
				t.Error("stale runtime sent HTTP")
				return pm.ComboMarketPage{Markets: []pm.ComboMarket{}}, nil
			})
			return e
		}},
		{"baseline_registration", func() error {
			return stale.RegisterNeededBaseline(ctx, sub, func(context.Context, pgx.Tx, tm.Subscription) error {
				t.Error("stale runtime registered baseline")
				return nil
			})
		}},
		{"baseline_bind", func() error {
			return stale.BindBaseline(ctx, token, pendingID, 1, func() tm.WalletObservation { return tm.WalletObservation{} })
		}},
		{"baseline_boundary", func() error { return stale.SaveBaselineBoundary(ctx, token, pendingID, 1, time.Now()) }},
		{"baseline_complete", func() error { return stale.CompleteBaseline(ctx, token, pendingID) }},
		{"baseline_fail", func() error { return stale.FailBaseline(ctx, token, pendingID, "stale") }},
		{"baseline_recovery", func() error {
			return stale.abandonUnboundBaseline(ctx, token, pgtype.UUID{Bytes: uuid.MustParse(pendingID), Valid: true})
		}},
		{"stopped_baseline_cleanup", func() error { return stale.CleanupStoppedBaselines(ctx, token) }},
		{"session_recovery", func() error { return a.RecoverPending(ctx) }},
		{"epoch_start", func() error { _, e := stale.StartCollectorEpoch(ctx, token); return e }},
		{"epoch_close", func() error { return stale.CloseCollectorEpoch(ctx, token, 1, "stale") }},
		{"filter_checkpoint", func() error { return stale.AckFilters(ctx, token, 1, 1) }},
		{"observation_checkpoint", func() error {
			return stale.SaveObservationCheckpoint(ctx, tm.CheckpointInterval{ID: interval, OwnerID: account.ID}, tm.ObservationCheckpoint{At: time.Now(), ExpiresAt: time.Now().Add(time.Minute)}, func() bool { return true })
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			before := runtimeWriteSnapshot(t, db.Pool)
			require.ErrorIs(t, tc.call(), ErrRuntimeFenced)
			require.Equal(t, before, runtimeWriteSnapshot(t, db.Pool))
		})
	}
}

func TestRuntimeWriteRequiresExplicitRuntimeStore(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	ctx := context.Background()
	base := NewSQLStore(db.Pool)
	require.ErrorIs(t, base.SaveMetadata(ctx, "offline", tm.TradeMetadata{}), ErrRuntimeRequired)
	require.ErrorIs(t, base.CompleteProjectionMetadata(ctx, 1, true), ErrRuntimeRequired)
	_, err := base.RefreshComboPage(ctx, func(context.Context, string, int) (pm.ComboMarketPage, error) {
		t.Error("offline store called provider")
		return pm.ComboMarketPage{}, nil
	})
	require.ErrorIs(t, err, ErrRuntimeRequired)
	require.ErrorIs(t, base.RecordFinalityObservation(ctx, 1, tm.FinalityRoundObservation{}), ErrRuntimeRequired)
	_, err = base.StartCollectorEpoch(ctx, 1)
	require.ErrorIs(t, err, ErrRuntimeRequired)
	// A live owner elsewhere does not implicitly authorize the read adapter.
	owner, err := base.AcquireRuntimeSession(ctx)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, owner.CloseAfterWorkers(ctx)) })
	require.ErrorIs(t, base.SaveMetadata(ctx, "offline", tm.TradeMetadata{}), ErrRuntimeRequired)
	_, _, err = base.LoadMetadata(ctx, "offline")
	require.NoError(t, err)
}

func TestRuntimeWriteOfflineSummaryPermitAndRevocationAdapters(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	ctx := context.Background()
	accounts := ac.NewSQLStore(db.Pool)
	account, err := accounts.EnsureDevelopmentAccount(ctx, accountcredentials.DevelopmentRoleMember)
	require.NoError(t, err)
	_, err = db.Pool.Exec(ctx, `INSERT INTO telegram_bindings(account_id,telegram_user_id,telegram_chat_id,telegram_display_name,revision)VALUES($1,123,123,'offline',1)`, account.ID)
	require.NoError(t, err)
	base := NewSQLStore(db.Pool)
	owner, err := base.AcquireRuntimeSession(ctx)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, owner.CloseAfterWorkers(ctx)) })
	runtime := bindTestRuntime(t, base, owner)
	require.NoError(t, runtime.ConfigureActivities("https://athena.test"))
	first := activityFixture(t, db.Pool, account.ID, 1)
	for i := 1; i <= 12; i++ {
		in := first
		if i > 1 {
			in = activitySource(t, db.Pool, account.ID, first.Candidate.SubscriptionID, first.Candidate.AttemptID, first.Trade.Wallet, first.Confirmation.SettledAt, i)
		}
		_, created, e := runtime.Project(ctx, in)
		require.NoError(t, e)
		require.True(t, created)
	}
	require.NoError(t, owner.CloseAfterWorkers(ctx))
	blocker, err := db.Pool.Begin(ctx)
	require.NoError(t, err)
	defer blocker.Rollback(ctx)
	_, err = blocker.Exec(ctx, `SELECT 1 FROM trader_sync_runtime_control FOR UPDATE`)
	require.NoError(t, err)
	bounded, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	require.NoError(t, base.ConfigureActivities("https://athena.test"))
	notifications := ns.NewSQLStore(db.Pool)
	var batch tm.SummaryBatch
	var permit delivery.Permit
	// Summary freeze, permit and attach borrow exactly one account transaction.
	require.NoError(t, txgate.WithAccountTx(bounded, db.Pool, account.ID, func(tx pgx.Tx) error {
		batch, err = base.FreezeSummaryTx(bounded, tx, account.ID, 1, 123)
		if err != nil {
			return err
		}
		permit, err = notifications.AuthorizeTx(bounded, tx, delivery.Candidate{Ref: delivery.WorkRef{Kind: "account", ID: batch.Parts[0].DeliveryID}, OwnerID: account.ID, ChatID: 123}, uuid.New(), nil)
		if err != nil {
			return err
		}
		return notifications.AttachSummaryPermitTx(bounded, tx, batch.ID, permit)
	}))
	require.NotEmpty(t, batch.Parts)
	before, err := accounts.GetAccountAccess(bounded, account.ID)
	require.NoError(t, err)
	revoked := before.Clone()
	revoked.Modules[accountaccess.ModuleTraderSync] = accountaccess.AccessLevelNone
	fail := true
	forced := errors.New("force atomic revocation rollback")
	adapter := NewAccessRevocationAdapter()
	accounts.SetAccessChangeHook(func(c context.Context, tx pgx.Tx, id string, previous, next accountaccess.Access) error {
		if e := adapter.ApplyAccessChangeTx(c, tx, id, previous, next); e != nil {
			return e
		}
		if fail {
			return forced
		}
		return nil
	})
	snapshot := runtimeWriteSnapshot(t, db.Pool)
	_, err = accounts.UpdateAccountAccess(bounded, account.ID, revoked, before.Revision)
	require.ErrorIs(t, err, forced)
	require.Equal(t, snapshot, runtimeWriteSnapshot(t, db.Pool))
	fail = false
	disabled, err := accounts.UpdateAccountAccess(bounded, account.ID, revoked, before.Revision)
	require.NoError(t, err)
	var activeIntervals, eligible, unlicensed int
	require.NoError(t, db.Pool.QueryRow(ctx, `SELECT count(*) FROM trader_sync_monitor_intervals WHERE ended_at IS NULL`).Scan(&activeIntervals))
	require.Zero(t, activeIntervals)
	require.NoError(t, db.Pool.QueryRow(ctx, `SELECT count(*) FROM trader_sync_alert_memberships WHERE eligibility_revoked_at IS NULL`).Scan(&eligible))
	require.Zero(t, eligible)
	require.NoError(t, db.Pool.QueryRow(ctx, `SELECT count(*) FROM account_notification_deliveries WHERE status='pending'`).Scan(&unlicensed))
	require.Zero(t, unlicensed)
	// A granted request keeps its actual result even after product revocation.
	require.NoError(t, notifications.RecordOutcome(bounded, permit, delivery.Outcome{Kind: "sent", MessageID: "42"}, time.Now()))
	disabled.Modules[accountaccess.ModuleTraderSync] = accountaccess.AccessLevelReadWrite
	_, err = accounts.UpdateAccountAccess(bounded, account.ID, disabled, disabled.Revision)
	require.NoError(t, err)
	_, err = notifications.Authorize(bounded, delivery.Candidate{Ref: permit.Work, OwnerID: account.ID, ChatID: 123}, uuid.New(), nil)
	require.ErrorIs(t, err, ns.ErrDeliveryNotEligible)
	var state string
	require.NoError(t, db.Pool.QueryRow(ctx, `SELECT status FROM account_notification_deliveries WHERE id=$1`, permit.Work.ID).Scan(&state))
	require.Equal(t, "sent", state)
}

func TestRuntimeWriteCompletedBaselineStillRequiresOwner(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	ctx := context.Background()
	account, err := ac.NewSQLStore(db.Pool).EnsureDevelopmentAccount(ctx, accountcredentials.DevelopmentRoleMember)
	require.NoError(t, err)
	source := activityFixture(t, db.Pool, account.ID, 1)
	base := NewSQLStore(db.Pool)
	a, err := base.AcquireRuntimeSession(ctx)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, a.CloseAfterWorkers(ctx)) })
	stale := bindTestRuntime(t, base, a)
	require.NoError(t, stale.CompleteBaseline(ctx, a.CollectorToken(), source.Candidate.AttemptID))
	require.NoError(t, a.CloseAfterWorkers(ctx))
	b, err := base.AcquireRuntimeSession(ctx)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, b.CloseAfterWorkers(ctx)) })
	require.ErrorIs(t, stale.CompleteBaseline(ctx, a.CollectorToken(), source.Candidate.AttemptID), ErrRuntimeFenced)
	require.ErrorIs(t, base.CompleteBaseline(ctx, a.CollectorToken(), source.Candidate.AttemptID), ErrRuntimeRequired)
}
