//go:build integration

package store

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/google/uuid"
	"github.com/useryege/athena/internal/accountcredentials"
	ac "github.com/useryege/athena/internal/accountstate/store"
	"github.com/useryege/athena/internal/accountstate/store/migrations"
	"github.com/useryege/athena/internal/testutil/pgtest"
	tm "github.com/useryege/athena/internal/tradersync/types"
	"testing"
	"time"
)

func TestTimingUTCAnomalyDenominators(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	ctx := context.Background()
	accounts := ac.NewSQLStore(db.Pool)
	owner, e := accounts.EnsureDevelopmentAccount(ctx, accountcredentials.DevelopmentRoleMember)
	if e != nil {
		t.Fatal(e)
	}
	admin, e := accounts.EnsureDevelopmentAccount(ctx, accountcredentials.DevelopmentRoleAdministrator)
	if e != nil {
		t.Fatal(e)
	}
	s := runtimeTestStore(t, db.Pool)
	if e = s.ConfigureActivities("https://athena.test"); e != nil {
		t.Fatal(e)
	}
	if _, e = db.Pool.Exec(ctx, `INSERT INTO telegram_bindings(account_id,telegram_user_id,telegram_chat_id,telegram_display_name,revision)VALUES($1,123,123,'fixture',1)`, owner.ID); e != nil {
		t.Fatal(e)
	}
	first := activityFixture(t, db.Pool, owner.ID, 2001)
	for i := 0; i < 11; i++ {
		in := first
		if i > 0 {
			in = activitySource(t, db.Pool, owner.ID, first.Candidate.SubscriptionID, first.Candidate.AttemptID, first.Trade.Wallet, first.Confirmation.SettledAt, 2001+i)
		}
		id, created, e := s.Project(ctx, in)
		if e != nil || !created {
			t.Fatal(e)
		}
		if i >= 3 {
			continue
		}
		var delivery int64
		if e = db.Pool.QueryRow(ctx, `SELECT id FROM account_notification_deliveries WHERE activity_id=$1`, id).Scan(&delivery); e != nil {
			t.Fatal(e)
		}
		// The anomalous sample has recorded < authorized < started < result < sender return.
		if _, e = db.Pool.Exec(ctx, `INSERT INTO notification_delivery_attempts(id,work_kind,work_id,owner_id,sender_incarnation,telegram_chat_id,telegram_group,payload_digest,authorized_at,started_at,result_at,sender_returned_at,outcome)
   SELECT $2,'account',$3,$4,$5,123,false,decode(repeat('ab',32),'hex'),recorded_at+interval '1 second',recorded_at+interval '2 seconds',
   CASE WHEN $6::int<2 THEN recorded_at+interval '3 seconds' END,
   CASE WHEN $6::int=0 THEN recorded_at+interval '4 seconds' WHEN $6::int=1 THEN recorded_at+interval '2.5 seconds' END,
   CASE WHEN $6::int=0 THEN 'failed' WHEN $6::int=1 THEN 'unknown' END
   FROM trader_sync_activities WHERE id=$1`, id, uuid.NewString(), delivery, owner.ID, uuid.NewString(), i); e != nil {
			t.Fatal(e)
		}
	}
	oldest := time.Now().Add(-time.Minute)
	for i, offset := range []int{10, 5, 0, 20, -1, 30} {
		var started any = oldest.Add(time.Duration(offset) * time.Second)
		if i == 2 {
			started = nil
		}
		if _, e = db.Pool.Exec(ctx, `INSERT INTO trader_sync_summary_batches(owner_id,binding_revision,chat_id,oldest_at,first_started_at)VALUES($1,1,123,$2,$3)`, owner.ID, oldest, started); e != nil {
			t.Fatal(e)
		}
	}
	r, e := s.ReadRuntimeStatus(ctx, admin.ID, tm.ObservationClock{})
	if e != nil {
		t.Fatal(e)
	}
	got := map[string]string{}
	for _, v := range r.Metrics {
		got[v.Name] = v.Value
	}
	for name, want := range map[string]string{
		"timing_attempts_total": "3", "timing_attempts_failed": "1", "timing_attempts_unknown": "1", "timing_attempts_result_unconfirmed": "1",
		"timing_worker_result_wait_usable": "1", "timing_worker_result_wait_clock_anomalies": "1", "timing_worker_result_wait_missing": "1",
		"timing_worker_result_wait_p95_utc_seconds": "0.5",
		"timing_summary_batches_total":              "6", "timing_summary_oldest_start_usable": "4", "timing_summary_oldest_start_clock_anomalies": "1", "timing_summary_oldest_start_missing": "1",
		"timing_summary_adjacent_start_usable": "1", "timing_summary_adjacent_start_clock_anomalies": "2", "timing_summary_adjacent_start_missing": "2", "timing_summary_adjacent_start_no_predecessor": "1",
		"timing_summary_waiting_members": "1",
	} {
		if got[name] != want {
			t.Errorf("%s=%q want %q", name, got[name], want)
		}
	}
	if got["timing_activity_all_clock_anomalies"] != "0" {
		t.Error("worker anomaly mislabelled as activity anomaly")
	}
	t.Log(fmt.Sprintf("attempt and summary UTC denominators: %v", got))
}

func TestFinalityObservationPersistence(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	ctx := context.Background()
	owner, e := ac.NewSQLStore(db.Pool).EnsureDevelopmentAccount(ctx, accountcredentials.DevelopmentRoleMember)
	if e != nil {
		t.Fatal(e)
	}
	source := activityFixture(t, db.Pool, owner.ID, 3001).Candidate.SourceID
	s := runtimeTestStore(t, db.Pool)
	cutoff, e := s.FinalityObservationCutoff(ctx)
	if e != nil {
		t.Fatal(e)
	}
	if cutoff != source {
		t.Errorf("committed startup cutoff=%d want %d", cutoff, source)
	}
	at := time.Now()
	first := tm.FinalityRoundObservation{ClockID: "one", StartedAt: at, ReturnedAt: at.Add(time.Millisecond), StartedNS: 100, ReturnedNS: 200, Reliable: true, SourceAfterCutoff: true}
	if e = s.RecordFinalityObservation(ctx, source, first); e != nil {
		t.Fatal(e)
	}
	read := func() tm.FinalityTiming {
		t.Helper()
		var raw []byte
		if e := db.Pool.QueryRow(ctx, `SELECT to_jsonb(r)->'finality_timing' FROM trader_sync_source_records r WHERE id=$1`, source).Scan(&raw); e != nil {
			t.Fatal(e)
		}
		var v tm.FinalityTiming
		if len(raw) > 0 {
			if e = json.Unmarshal(raw, &v); e != nil {
				t.Fatal(e)
			}
		}
		return v
	}
	waiting := read()
	if waiting.State != "waiting" || waiting.FirstStartedNS == nil || *waiting.FirstStartedNS != 100 {
		t.Fatalf("first actual observation not persisted: %+v", waiting)
	}
	next := first
	next.StartedNS = 800
	next.ReturnedNS = 850
	next.Confirmed = true
	if e = s.RecordFinalityObservation(ctx, source, next); e != nil {
		t.Fatal(e)
	}
	if e = s.RecordFinalityObservation(ctx, source, next); e != nil {
		t.Fatal(e)
	}
	next.ClockID = "two"
	next.ReturnedNS = 9000
	if e = s.RecordFinalityObservation(ctx, source, next); e != nil {
		t.Fatal(e)
	}
	done := read()
	if done.State != "completed" || *done.FirstConfirmedNS != 850 || *done.FirstStartedNS != 100 {
		t.Fatalf("first interval overwritten: %+v", done)
	}
	admin, e := ac.NewSQLStore(db.Pool).EnsureDevelopmentAccount(ctx, accountcredentials.DevelopmentRoleAdministrator)
	if e != nil {
		t.Fatal(e)
	}
	ids := []int64{source}
	for i := 0; i < 5; i++ {
		// Independent source identities, same original subscription/attempt.
		var sub, attempt string
		if e = db.Pool.QueryRow(ctx, `SELECT subscription_id,baseline_attempt_id FROM trader_sync_source_candidates WHERE source_record_id=$1`, source).Scan(&sub, &attempt); e != nil {
			t.Fatal(e)
		}
		in := activitySource(t, db.Pool, owner.ID, sub, attempt, common.HexToAddress("0x1111111111111111111111111111111111111111"), at, 3002+i)
		ids = append(ids, in.Candidate.SourceID)
	}
	old := first
	old.SourceAfterCutoff = false
	if e = s.RecordFinalityObservation(ctx, ids[1], old); e != nil {
		t.Fatal(e)
	}
	if e = s.RecordFinalityObservation(ctx, ids[2], first); e != nil {
		t.Fatal(e)
	}
	other := first
	other.ClockID = "two"
	if e = s.RecordFinalityObservation(ctx, ids[3], other); e != nil {
		t.Fatal(e)
	}
	if e = s.RecordFinalityObservation(ctx, ids[4], first); e != nil {
		t.Fatal(e)
	}
	gap := first
	gap.Reliable = false
	gap.Confirmed = true
	if e = s.RecordFinalityObservation(ctx, ids[4], gap); e != nil {
		t.Fatal(e)
	}
	clock := tm.ObservationClock{ID: "one", ElapsedNS: 1000, Valid: true, Cutoff: ids[1]}
	assertMetrics := func(clock tm.ObservationClock, wants map[string]string) map[string]string {
		t.Helper()
		r, e := s.ReadRuntimeStatus(ctx, admin.ID, clock)
		if e != nil {
			t.Fatal(e)
		}
		values := map[string]string{}
		for _, v := range r.Metrics {
			values[v.Name] = v.Value
		}
		for name, want := range wants {
			if values[name] != want {
				t.Errorf("%s=%q want %q", name, values[name], want)
			}
		}
		return values
	}
	assertMetrics(clock, map[string]string{"finality_sources_total": "6", "finality_sources_completed": "1", "finality_sources_waiting": "1", "finality_sources_unavailable": "3", "finality_sources_not_started": "1",
		"finality_unavailable_first_observation_missing": "1", "finality_unavailable_clock_changed": "1", "finality_unavailable_observation_gap": "1",
		"finality_first_attempt_round_usable": "2", "finality_first_attempt_to_first_confirmed_usable": "1", "finality_after_first_attempt_to_first_confirmed_usable": "1",
		"finality_first_attempt_to_first_confirmed_p95_monotonic_seconds": "7.5e-07", "finality_after_first_attempt_to_first_confirmed_p95_monotonic_seconds": "6.5e-07", "finality_pending_oldest_monotonic_seconds": "9e-07"})
	clock.Valid = false
	values := assertMetrics(clock, map[string]string{"finality_sources_completed": "1", "finality_sources_waiting": "0", "finality_sources_unavailable": "5", "finality_sources_not_started": "0"})
	if _, ok := values["finality_pending_oldest_monotonic_seconds"]; ok {
		t.Fatal("gap produced usable pending age")
	}
	clock.Valid = true
	clock.ID = "new-process"
	assertMetrics(clock, map[string]string{"finality_sources_completed": "1", "finality_sources_waiting": "0", "finality_unavailable_clock_changed": "2"})
	// JSONB storage retains values beyond IEEE754 integer precision exactly.
	exact := first
	exact.StartedNS = 9007199254740993
	exact.ReturnedNS = 9007199254740999
	if e = s.RecordFinalityObservation(ctx, ids[5], exact); e != nil {
		t.Fatal(e)
	}
	var raw []byte
	if e = db.Pool.QueryRow(ctx, `SELECT finality_timing FROM trader_sync_source_records WHERE id=$1`, ids[5]).Scan(&raw); e != nil {
		t.Fatal(e)
	}
	var precise tm.FinalityTiming
	if e = json.Unmarshal(raw, &precise); e != nil || precise.FirstStartedNS == nil || *precise.FirstStartedNS != exact.StartedNS {
		t.Fatalf("precision lost: %s err=%v", raw, e)
	}
}
