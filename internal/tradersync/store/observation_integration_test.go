//go:build integration

package store

import (
	"context"
	"github.com/ethereum/go-ethereum/common"
	"github.com/google/uuid"
	"github.com/useryege/athena/internal/accountcredentials"
	ac "github.com/useryege/athena/internal/accountstate/store"
	"github.com/useryege/athena/internal/accountstate/store/migrations"
	"github.com/useryege/athena/internal/testutil/pgtest"
	tm "github.com/useryege/athena/internal/tradersync/types"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestObservationCheckpointRequiresCurrentCoveredInterval(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	ctx := context.Background()
	owner, e := ac.NewSQLStore(db.Pool).EnsureDevelopmentAccount(ctx, accountcredentials.DevelopmentRoleMember)
	if e != nil {
		t.Fatal(e)
	}
	in := activityFixture(t, db.Pool, owner.ID, 1)
	if _, e = db.Pool.Exec(ctx, `UPDATE trader_sync_collector_control SET fencing_token=1,owner_id=$1,active_epoch=1`, uuid.NewString()); e != nil {
		t.Fatal(e)
	}
	s := NewSQLStore(db.Pool)
	rows, e := s.CheckpointIntervals(ctx, 1, "", 10)
	if e != nil {
		t.Fatal(e)
	}
	if len(rows) != 1 || rows[0].LastReliableAt != nil {
		t.Fatalf("initial checkpoint: %+v", rows)
	}
	at := time.Now().Add(-time.Millisecond)
	evidence := tm.ObservationCheckpoint{Token: 1, Epoch: 1, FilterRevision: 1, At: at, ExpiresAt: time.Now().Add(time.Second), CoveredAt: at.Add(-time.Second), Wallets: []common.Address{in.Trade.Wallet}}
	if e = s.SaveObservationCheckpoint(ctx, rows[0], evidence, func() bool { return true }); e != nil {
		t.Fatal(e)
	}
	rows, e = s.CheckpointIntervals(ctx, 1, "", 10)
	if e != nil {
		t.Fatal(e)
	}
	if len(rows) != 1 || rows[0].LastReliableAt == nil || !rows[0].LastReliableAt.Equal(at.Truncate(time.Microsecond)) {
		t.Fatalf("real observed instant not persisted: %+v", rows)
	}
	old := *rows[0].LastReliableAt
	evidence.At = time.Now()
	evidence.CoveredAt = evidence.At.Add(time.Second)
	if e = s.SaveObservationCheckpoint(ctx, rows[0], evidence, func() bool { return true }); e == nil {
		t.Fatal("old health point accepted for newly acknowledged coverage")
	}
	var stored time.Time
	if e = db.Pool.QueryRow(ctx, `SELECT last_reliable_at FROM trader_sync_monitor_intervals WHERE id=$1`, rows[0].ID).Scan(&stored); e != nil || !stored.Equal(old) {
		t.Fatal("invalid checkpoint changed durable fact", stored, e)
	}
}

func TestSubscriptionCurrentIntervalUsesCausalEpochAcrossClockRegression(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	ctx := context.Background()
	owner, e := ac.NewSQLStore(db.Pool).EnsureDevelopmentAccount(ctx, accountcredentials.DevelopmentRoleMember)
	if e != nil {
		t.Fatal(e)
	}
	in := activityFixture(t, db.Pool, owner.ID, 1)
	attempt := uuid.NewString()
	interval := uuid.NewString()
	if _, e = db.Pool.Exec(ctx, `UPDATE trader_sync_monitor_intervals SET state='closed',ended_at=effective_at+interval '1 second' WHERE subscription_id=$1`, in.Candidate.SubscriptionID); e != nil {
		t.Fatal(e)
	}
	if _, e = db.Pool.Exec(ctx, `INSERT INTO trader_sync_baseline_attempts(id,owner_id,subscription_id,activation_generation,collector_epoch,filter_revision,expected_revision,state,effective_at) SELECT $1,owner_id,subscription_id,activation_generation,2,filter_revision,expected_revision,'succeeded',effective_at-interval '1 hour' FROM trader_sync_baseline_attempts WHERE id=$2`, attempt, in.Candidate.AttemptID); e != nil {
		t.Fatal(e)
	}
	if _, e = db.Pool.Exec(ctx, `INSERT INTO trader_sync_monitor_intervals(id,owner_id,subscription_id,baseline_attempt_id,activation_generation,collector_epoch,filter_revision,expected_revision,registered_high,candidate_effective_at,effective_at) SELECT $1,owner_id,subscription_id,$2,activation_generation,2,filter_revision,expected_revision,registered_high,candidate_effective_at-interval '1 hour',effective_at-interval '1 hour' FROM trader_sync_monitor_intervals WHERE subscription_id=$3`, interval, attempt, in.Candidate.SubscriptionID); e != nil {
		t.Fatal(e)
	}
	result, e := NewSQLStore(db.Pool).ReadSubscriptions(ctx, owner.ID, tm.SubscriptionFilter{ID: in.Candidate.SubscriptionID}, tm.ReadPage{Limit: 1})
	if e != nil {
		t.Fatal(e)
	}
	if len(result.Subscriptions) != 1 || result.Subscriptions[0].CurrentInterval == nil || result.Subscriptions[0].CurrentInterval.ID != interval {
		t.Fatalf("clock regression selected a closed predecessor as current: %+v", result.Subscriptions)
	}
}

func TestCheckpointRejectsExpiredClosedUncoveredAndClockRegression(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	ctx := context.Background()
	owner, e := ac.NewSQLStore(db.Pool).EnsureDevelopmentAccount(ctx, accountcredentials.DevelopmentRoleMember)
	if e != nil {
		t.Fatal(e)
	}
	in := activityFixture(t, db.Pool, owner.ID, 1)
	if _, e = db.Pool.Exec(ctx, `UPDATE trader_sync_collector_control SET fencing_token=1,owner_id=$1,active_epoch=1`, uuid.NewString()); e != nil {
		t.Fatal(e)
	}
	s := NewSQLStore(db.Pool)
	rows, e := s.CheckpointIntervals(ctx, 1, "", 100)
	if e != nil || len(rows) != 1 {
		t.Fatal(e)
	}
	target := rows[0]
	at := time.Now().Add(-time.Millisecond)
	good := tm.ObservationCheckpoint{Token: 1, Epoch: 1, FilterRevision: 1, At: at, ExpiresAt: time.Now().Add(time.Minute), CoveredAt: at.Add(-time.Second), Wallets: []common.Address{in.Trade.Wallet}}
	if e = s.SaveObservationCheckpoint(ctx, target, good, func() bool { return true }); e != nil {
		t.Fatal(e)
	}
	for _, mode := range []string{"expired", "session_closed", "uncovered", "old_filter", "old_generation", "clock_regressed", "future_wall", "wrong_fence"} {
		t.Run(mode, func(t *testing.T) {
			evidence := good
			row := target
			alive := true
			switch mode {
			case "expired":
				evidence.ExpiresAt = time.Now().Add(-time.Second)
			case "session_closed":
				alive = false
			case "uncovered":
				evidence.Wallets = []common.Address{common.HexToAddress("0xff")}
			case "old_filter":
				evidence.FilterRevision = 0
			case "old_generation":
				row.Generation++
			case "clock_regressed":
				evidence.At = at.Add(-time.Second)
				evidence.CoveredAt = evidence.At.Add(-time.Second)
			case "future_wall":
				evidence.At = time.Now().Add(time.Hour)
			case "wrong_fence":
				evidence.Token++
			}
			err := s.SaveObservationCheckpoint(ctx, row, evidence, func() bool { return alive })
			if err == nil {
				t.Fatal("invalid health evidence advanced checkpoint")
			}
			var stored time.Time
			if e := db.Pool.QueryRow(ctx, `SELECT last_reliable_at FROM trader_sync_monitor_intervals WHERE id=$1`, target.ID).Scan(&stored); e != nil || !stored.Equal(at.Truncate(time.Microsecond)) {
				t.Fatal("invalid evidence altered existing fact", stored, e)
			}
		})
	}
	// A real pause closes the coverage; a still-alive session cannot revive it.
	if _, e = db.Pool.Exec(ctx, `UPDATE trader_sync_subscriptions SET desired_state='paused' WHERE id=$1`, target.SubscriptionID); e != nil {
		t.Fatal(e)
	}
	if e = s.SaveObservationCheckpoint(ctx, target, good, func() bool { return true }); e == nil {
		t.Fatal("paused subscription accepted health")
	}
}

func TestObservationHistoryRecoveryUsesSameGenerationAndRealBoundaries(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	ctx := context.Background()
	owner, e := ac.NewSQLStore(db.Pool).EnsureDevelopmentAccount(ctx, accountcredentials.DevelopmentRoleMember)
	if e != nil {
		t.Fatal(e)
	}
	in := tm.Projection{Candidate: tm.Candidate{SubscriptionID: uuid.NewString(), AttemptID: uuid.NewString()}}
	if _, e = db.Pool.Exec(ctx, `INSERT INTO trader_sync_subscriptions(id,owner_id,wallet,desired_state,observation_state,target_display) VALUES($1,$2,$3,'enabled','healthy','{"DisplayName":{"Availability":"unavailable"},"Avatar":{"Availability":"unavailable"},"ProfileURL":{"Availability":"unavailable"}}')`, in.Candidate.SubscriptionID, owner.ID, common.HexToAddress("0x11").Bytes()); e != nil {
		t.Fatal(e)
	}
	if _, e = db.Pool.Exec(ctx, `INSERT INTO trader_sync_collector_epochs(fencing_token) VALUES(1)`); e != nil {
		t.Fatal(e)
	}
	if _, e = db.Pool.Exec(ctx, `INSERT INTO trader_sync_baseline_attempts(id,owner_id,subscription_id,activation_generation,collector_epoch,filter_revision,expected_revision,state,effective_at) VALUES($1,$2,$3,1,1,1,1,'succeeded',clock_timestamp()-interval '1 minute')`, in.Candidate.AttemptID, owner.ID, in.Candidate.SubscriptionID); e != nil {
		t.Fatal(e)
	}
	if _, e = db.Pool.Exec(ctx, `INSERT INTO trader_sync_monitor_intervals(owner_id,subscription_id,baseline_attempt_id,activation_generation,collector_epoch,filter_revision,expected_revision,registered_high,candidate_effective_at,effective_at) VALUES($1,$2,$3,1,1,1,1,0,clock_timestamp()-interval '1 minute',clock_timestamp()-interval '1 minute')`, owner.ID, in.Candidate.SubscriptionID, in.Candidate.AttemptID); e != nil {
		t.Fatal(e)
	}
	s := NewSQLStore(db.Pool)
	session, e := s.AcquireRuntimeSession(ctx)
	if e != nil {
		t.Fatal(e)
	}
	defer session.CloseAfterWorkers(ctx)
	if _, e = db.Pool.Exec(ctx, `UPDATE trader_sync_collector_control SET active_epoch=1 WHERE singleton`); e != nil {
		t.Fatal(e)
	}
	checkpoint := time.Now().Add(-time.Second).Truncate(time.Microsecond)
	if _, e = db.Pool.Exec(ctx, `UPDATE trader_sync_monitor_intervals SET last_reliable_at=$1 WHERE subscription_id=$2`, checkpoint, in.Candidate.SubscriptionID); e != nil {
		t.Fatal(e)
	}
	if e = s.CloseCollectorEpoch(ctx, session.CollectorToken(), 1, "connection_lost"); e != nil {
		t.Fatal(e)
	}
	before, e := s.ReadSubscriptions(ctx, owner.ID, tm.SubscriptionFilter{ID: in.Candidate.SubscriptionID}, tm.ReadPage{Limit: 1})
	if e != nil {
		t.Fatal(e)
	}
	observed := before.Subscriptions[0]
	admin, e := ac.NewSQLStore(db.Pool).EnsureDevelopmentAccount(ctx, accountcredentials.DevelopmentRoleAdministrator)
	if e != nil {
		t.Fatal(e)
	}
	assertShared := func(state string, count int64) {
		t.Helper()
		memberPage, err := s.ReadSubscriptions(ctx, owner.ID, tm.SubscriptionFilter{ID: in.Candidate.SubscriptionID, State: state}, tm.ReadPage{Limit: 1})
		if err != nil || len(memberPage.Subscriptions) != 1 {
			t.Fatal("member shared state filter", memberPage, err)
		}
		adminPage, err := s.ReadSubscriptionSummaries(ctx, admin.ID, tm.AdminSubscriptionFilter{ID: in.Candidate.SubscriptionID, State: state}, tm.ReadPage{Limit: 1})
		if err != nil || len(adminPage.Summaries) != 1 {
			t.Fatal("admin shared state filter", adminPage, err)
		}
		left, right := memberPage.Subscriptions[0].Observation, adminPage.Summaries[0].Observation
		if !reflect.DeepEqual(left, right) || left.InterruptionCount != count {
			t.Fatal("member/admin observation rules diverged", left, right)
		}
		history, err := s.ReadHistory(ctx, owner.ID, in.Candidate.SubscriptionID, tm.ReadPage{Limit: 51})
		if err != nil {
			t.Fatal(err)
		}
		var total int64
		for _, entry := range history.Entries {
			if entry.Interruption != nil {
				total++
				if left.LatestInterruption != nil && entry.Interruption.ID == left.LatestInterruption.ID && !reflect.DeepEqual(entry.Interruption, left.LatestInterruption) {
					t.Fatal("latest/history recovery diverged")
				}
			}
		}
		if total != count {
			t.Fatal("history/count related epoch divergence", total, count)
		}
	}
	assertShared("interrupted", 1)
	if observed.CurrentInterval.EndedAt == nil || observed.Observation.State != "interrupted" || observed.Observation.LatestInterruption == nil || observed.Observation.LatestInterruption.Start != nil || observed.Observation.LatestInterruption.End != nil {
		t.Fatal("logical close manufactured observation times", observed)
	}
	if e = s.CleanupStoppedBaselines(ctx, session.CollectorToken()); e != nil {
		t.Fatal(e)
	}
	assertShared("interrupted", 1)
	epoch2, e := s.StartCollectorEpoch(ctx, session.CollectorToken())
	if e != nil {
		t.Fatal(e)
	}
	if _, e = db.Pool.Exec(ctx, `INSERT INTO trader_sync_baseline_attempts(owner_id,subscription_id,activation_generation,collector_epoch,filter_revision,expected_revision,state) VALUES($1,$2,1,$3,1,1,'pending')`, owner.ID, in.Candidate.SubscriptionID, epoch2); e != nil {
		t.Fatal(e)
	}
	if e = s.CloseCollectorEpoch(ctx, session.CollectorToken(), epoch2, "connection_lost"); e != nil {
		t.Fatal(e)
	}
	if e = s.CleanupStoppedBaselines(ctx, session.CollectorToken()); e != nil {
		t.Fatal(e)
	}
	epoch3, e := s.StartCollectorEpoch(ctx, session.CollectorToken())
	if e != nil {
		t.Fatal(e)
	}
	attempt := uuid.NewString()
	recovered := checkpoint.Add(-time.Hour)
	if _, e = db.Pool.Exec(ctx, `INSERT INTO trader_sync_baseline_attempts(id,owner_id,subscription_id,activation_generation,collector_epoch,filter_revision,expected_revision,state,effective_at) VALUES($1,$2,$3,1,$4,1,1,'succeeded',$5)`, attempt, owner.ID, in.Candidate.SubscriptionID, epoch3, recovered); e != nil {
		t.Fatal(e)
	}
	if _, e = db.Pool.Exec(ctx, `INSERT INTO trader_sync_monitor_intervals(owner_id,subscription_id,baseline_attempt_id,activation_generation,collector_epoch,filter_revision,expected_revision,registered_high,candidate_effective_at,effective_at) VALUES($1,$2,$3,1,$4,1,1,0,$5,$5)`, owner.ID, in.Candidate.SubscriptionID, attempt, epoch3, recovered); e != nil {
		t.Fatal(e)
	}
	if _, e = db.Pool.Exec(ctx, `UPDATE trader_sync_subscriptions SET observation_state='healthy' WHERE id=$1`, in.Candidate.SubscriptionID); e != nil {
		t.Fatal(e)
	}
	assertShared("healthy", 2)
	history, e := s.ReadHistory(ctx, owner.ID, in.Candidate.SubscriptionID, tm.ReadPage{Limit: 51})
	if e != nil {
		t.Fatal(e)
	}
	interruptions := 0
	for _, entry := range history.Entries {
		if entry.Interruption != nil {
			interruptions++
			v := entry.Interruption
			if v.Start != nil || v.End == nil || v.RecoveredAt == nil || !v.End.Equal(recovered) || !v.RecoveredAt.Equal(recovered) || !strings.Contains(v.Uncertainty, "clock_order_uncertain") {
				t.Fatal("consecutive epochs did not share causal recovery", v)
			}
		}
	}
	if interruptions != 2 {
		t.Fatal("related interruption count", interruptions)
	}
	same, e := s.ReadSubscriptions(ctx, owner.ID, tm.SubscriptionFilter{ID: in.Candidate.SubscriptionID}, tm.ReadPage{Limit: 1})
	if e != nil || same.Subscriptions[0].Observation.LastReliableAt == nil || !same.Subscriptions[0].Observation.LastReliableAt.Equal(checkpoint) {
		t.Fatal("same generation lost original confirmed checkpoint", same, e)
	}
	if _, e = db.Pool.Exec(ctx, `UPDATE trader_sync_subscriptions SET activation_generation=2,observation_state='pending_baseline' WHERE id=$1`, in.Candidate.SubscriptionID); e != nil {
		t.Fatal(e)
	}
	next, e := s.ReadSubscriptions(ctx, owner.ID, tm.SubscriptionFilter{ID: in.Candidate.SubscriptionID}, tm.ReadPage{Limit: 1})
	if e != nil || next.Subscriptions[0].Observation.LastReliableAt != nil || next.Subscriptions[0].CurrentInterval != nil {
		t.Fatal("new generation inherited old coverage/health", next, e)
	}
}

func TestObservationExcludesAlreadyStoppedIntervalsAndPreparationFailure(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	ctx := context.Background()
	owner, e := ac.NewSQLStore(db.Pool).EnsureDevelopmentAccount(ctx, accountcredentials.DevelopmentRoleMember)
	if e != nil {
		t.Fatal(e)
	}
	s := NewSQLStore(db.Pool)
	session, e := s.AcquireRuntimeSession(ctx)
	if e != nil {
		t.Fatal(e)
	}
	defer session.CloseAfterWorkers(ctx)
	epoch, e := s.StartCollectorEpoch(ctx, session.CollectorToken())
	if e != nil {
		t.Fatal(e)
	}
	stopped := time.Now().Add(-time.Second).Truncate(time.Microsecond)
	subscriptions := map[string]string{}
	for i, state := range []string{"paused", "cancelled", "permission_disabled", "enabled"} {
		sub, attempt := uuid.NewString(), uuid.NewString()
		subscriptions[state] = sub
		if _, e = db.Pool.Exec(ctx, `INSERT INTO trader_sync_subscriptions(id,owner_id,wallet,desired_state,observation_state,ended_at,target_display) VALUES($1,$2,$3,$4,'pending_baseline',$5,'{"DisplayName":{"Availability":"unavailable"},"Avatar":{"Availability":"unavailable"},"ProfileURL":{"Availability":"unavailable"}}')`, sub, owner.ID, common.BytesToAddress([]byte{byte(i + 1)}).Bytes(), state, stopped); e != nil {
			t.Fatal(e)
		}
		if state == "enabled" {
			if _, e = db.Pool.Exec(ctx, `INSERT INTO trader_sync_baseline_attempts(id,owner_id,subscription_id,activation_generation,collector_epoch,filter_revision,expected_revision,state,ended_at,reason) VALUES($1,$2,$3,1,$4,1,1,'failed',$5,'baseline_head_invalid')`, attempt, owner.ID, sub, epoch, stopped); e != nil {
				t.Fatal(e)
			}
			continue
		}
		if _, e = db.Pool.Exec(ctx, `INSERT INTO trader_sync_baseline_attempts(id,owner_id,subscription_id,activation_generation,collector_epoch,filter_revision,expected_revision,state,effective_at) VALUES($1,$2,$3,1,$4,1,1,'succeeded',$5)`, attempt, owner.ID, sub, epoch, stopped.Add(-time.Minute)); e != nil {
			t.Fatal(e)
		}
		if _, e = db.Pool.Exec(ctx, `INSERT INTO trader_sync_monitor_intervals(owner_id,subscription_id,baseline_attempt_id,activation_generation,collector_epoch,filter_revision,expected_revision,registered_high,candidate_effective_at,effective_at,state,ended_at,reason) VALUES($1,$2,$3,1,$4,1,1,0,$5,$5,'closed',$6,$7)`, owner.ID, sub, attempt, epoch, stopped.Add(-time.Minute), stopped, state); e != nil {
			t.Fatal(e)
		}
	}
	if e = s.CloseCollectorEpoch(ctx, session.CollectorToken(), epoch, "connection_lost"); e != nil {
		t.Fatal(e)
	}
	if e = s.CleanupStoppedBaselines(ctx, session.CollectorToken()); e != nil {
		t.Fatal(e)
	}
	for state, sub := range subscriptions {
		page, err := s.ReadSubscriptions(ctx, owner.ID, tm.SubscriptionFilter{ID: sub}, tm.ReadPage{Limit: 1})
		if err != nil || len(page.Subscriptions) != 1 {
			t.Fatal(err)
		}
		v := page.Subscriptions[0]
		if v.Observation.InterruptionCount != 0 || v.Observation.LatestInterruption != nil {
			t.Fatal("unrelated epoch created interruption", state, v)
		}
		history, err := s.ReadHistory(ctx, owner.ID, sub, tm.ReadPage{Limit: 51})
		if err != nil {
			t.Fatal(err)
		}
		if state == "enabled" {
			if v.Observation.Reason != "baseline_head_invalid" || v.CurrentInterval != nil || len(history.Entries) != 0 {
				t.Fatal("preparation failure fabricated interval/interruption", v, history)
			}
			continue
		}
		if v.CurrentInterval == nil || v.CurrentInterval.EndedAt == nil || !v.CurrentInterval.EndedAt.Equal(stopped) || len(history.Entries) != 1 {
			t.Fatal("epoch close overwrote earlier user stop", v, history)
		}
	}
}

func TestObservationViewsExposeOnlySafeSharedFacts(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	ctx := context.Background()
	var count int
	if e := db.Pool.QueryRow(ctx, `SELECT count(*) FROM pg_class WHERE relnamespace='public'::regnamespace AND relkind='v' AND relname IN ('trader_sync_subscription_interruptions','trader_sync_subscription_observations')`).Scan(&count); e != nil {
		t.Fatal(e)
	}
	if count != 2 {
		t.Fatal("shared safe observation interfaces missing", count)
	}
	rows, e := db.Pool.Query(ctx, `SELECT table_name,column_name FROM information_schema.columns WHERE table_schema='public' AND table_name IN ('trader_sync_subscription_interruptions','trader_sync_subscription_observations')`)
	if e != nil {
		t.Fatal(e)
	}
	defer rows.Close()
	allowed := map[string]bool{"owner_id": true, "subscription_id": true, "activation_generation": true, "epoch_id": true, "id": true, "recorded_at": true, "ended_at": true, "reason": true, "awaiting_cleanup": true, "interruption_json": true, "status": true, "observation_json": true}
	for rows.Next() {
		var table, column string
		if e = rows.Scan(&table, &column); e != nil {
			t.Fatal(e)
		}
		if !allowed[column] {
			t.Fatal("private or unrelated field in shared observation view", table, column)
		}
	}
	if e = rows.Err(); e != nil {
		t.Fatal(e)
	}
}
