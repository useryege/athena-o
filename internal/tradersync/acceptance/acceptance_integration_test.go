//go:build integration

package acceptance

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	tm "github.com/useryege/athena/internal/tradersync/types"
	"testing"
	"time"
)

func TestRecordedPayloadTraversesActualPipeline(t *testing.T) {
	h := newHarness(t, 1, 1)
	defer h.Close()
	h.Advance(1100 * time.Millisecond)
	raw := h.Replay(h.wallets[0], 0)
	h.Push(raw)
	h.wait("one actual successful activity", 10*time.Second, func() bool {
		s := h.Stats()
		return s.Activities == 1 && s.HTTPCalls == 1 && s.Owners[h.ownerIDs[0]].Sent == 1
	})
	h.Push(raw)
	h.Advance(100 * time.Millisecond)
	s := h.Stats()
	if s.Activities != 1 || s.Sources != 1 || s.HTTPCalls != 1 || s.SideEffects != 0 || s.HistoricalRangeCalls != 0 {
		t.Fatal(s)
	}
	t.Logf("actual pipeline: %+v", s)
	metrics := h.captureRuntime("tradersync-fix1-ordinary")
	if metrics["finality_sources_completed"] != "1" {
		t.Fatalf("actual first confirmation missing: %v", metrics)
	}

}

func TestActualConfirmationFailureKeepsWSSAndClosedEpochCandidate(t *testing.T) {
	for _, fault := range []string{"403", "null"} {
		t.Run(fault, func(t *testing.T) {
			h := newHarness(t, 1, 1)
			defer h.Close()
			h.Advance(1100 * time.Millisecond)
			h.mu.Lock()
			h.httpFault = fault
			h.mu.Unlock()
			raw := h.Replay(h.wallets[0], 0)
			h.Push(raw)
			h.wait("actual failed finalized RPC retained raw", 6*time.Second, func() bool {
				s := h.Stats()
				return s.Sources == 1 && s.Unverified == 1 && s.Costs["block_finalized"] > 0
			})
			h.wait("first finality attempt persistently waiting", time.Second, func() bool {
				var state string
				e := h.db.Pool.QueryRow(h.ctx, `SELECT finality_timing->>'state' FROM trader_sync_source_records`).Scan(&state)
				return e == nil && state == "waiting"
			})
			waitingMetrics := h.captureRuntime("tradersync-fix1-finality-waiting-" + fault)
			if waitingMetrics["finality_sources_waiting"] != "1" || waitingMetrics["finality_pending_oldest_monotonic_seconds"] == "" {
				t.Fatal(waitingMetrics)
			}
			var epoch int64
			var attempt string
			var state string
			if e := h.db.Pool.QueryRow(h.ctx, `SELECT r.collector_epoch,c.baseline_attempt_id::text,s.observation_state FROM trader_sync_source_records r JOIN trader_sync_source_candidates c ON c.source_record_id=r.id JOIN trader_sync_subscriptions s ON s.id=c.subscription_id`).Scan(&epoch, &attempt, &state); e != nil || state != "healthy" {
				t.Fatal("confirmation outage contaminated WSS", state, e)
			}
			if s := h.Stats(); s.Activities != 0 || s.HTTPCalls != 0 {
				t.Fatal(s)
			}
			h.mu.Lock()
			sockets := append([]*socket(nil), h.sockets...)
			h.mu.Unlock()
			for _, s := range sockets {
				s.conn.Close()
			}
			h.wait("old epoch persistently closed", 5*time.Second, func() bool {
				var ended bool
				e := h.db.Pool.QueryRow(h.ctx, `SELECT ended_at IS NOT NULL FROM trader_sync_collector_epochs WHERE id=$1`, epoch).Scan(&ended)
				return e == nil && ended
			})
			h.mu.Lock()
			h.httpFault = ""
			h.mu.Unlock()
			h.wait("old successful interval candidate still projects", 8*time.Second, func() bool { s := h.Stats(); return s.Activities == 1 && s.Owners[h.ownerIDs[0]].Sent == 1 })
			var retained string
			if e := h.db.Pool.QueryRow(h.ctx, `SELECT baseline_attempt_id::text FROM trader_sync_source_candidates`).Scan(&retained); e != nil || retained != attempt {
				t.Fatal("candidate rebound to new attempt", retained, attempt, e)
			}
			var timingJSON []byte
			if e := h.db.Pool.QueryRow(h.ctx, `SELECT finality_timing FROM trader_sync_source_records`).Scan(&timingJSON); e != nil {
				t.Fatal(e)
			}
			var timing tm.FinalityTiming
			if e := json.Unmarshal(timingJSON, &timing); e != nil {
				t.Fatal(e)
			}
			if timing.State != "completed" || timing.FirstStartedNS == nil || timing.FirstConfirmedNS == nil || timing.FirstRoundNS == nil || *timing.FirstConfirmedNS-*timing.FirstStartedNS <= *timing.FirstRoundNS {
				t.Fatalf("cross-retry interval lost: %s", timingJSON)
			}
			metrics := h.captureRuntime("tradersync-fix1-finality-completed-" + fault)
			if metrics["finality_sources_completed"] != "1" || metrics["finality_first_attempt_to_first_confirmed_usable"] != "1" || metrics["finality_after_first_attempt_to_first_confirmed_p95_monotonic_seconds"] == "" {
				t.Fatal(metrics)
			}
			t.Logf("actual persisted finality retry: %s", timingJSON)
			t.Logf("%s restored original candidate: %+v", fault, h.Stats())
		})
	}
}
func TestActualVersionReadFailureIsRetriedWithoutDiscardingRaw(t *testing.T) {
	h := newHarness(t, 1, 1)
	defer h.Close()
	h.Advance(1100 * time.Millisecond)
	h.mu.Lock()
	h.codeFault = true
	h.mu.Unlock()
	raw := h.Replay(h.wallets[0], 8)
	h.Push(raw)
	h.wait("unknown implementation evidence", 6*time.Second, func() bool { s := h.Stats(); return s.Unverified == 1 && s.Costs["rpc_eth_getCode"] > 0 })
	if s := h.Stats(); s.Activities != 0 || s.HTTPCalls != 0 {
		t.Fatal(s)
	}
	h.mu.Lock()
	h.codeFault = false
	h.mu.Unlock()
	h.wait("same candidate after version recovery", 8*time.Second, func() bool { s := h.Stats(); return s.Activities == 1 && s.Owners[h.ownerIDs[0]].Sent == 1 })
	if s := h.Stats(); s.Sources != 1 || s.SideEffects != 0 || s.HistoricalRangeCalls != 0 {
		t.Fatal(s)
	}
}
func TestActualRemovedCannotBeClearedByOriginalReceipt(t *testing.T) {
	h := newHarness(t, 1, 1)
	defer h.Close()
	h.Advance(1100 * time.Millisecond)
	h.mu.Lock()
	h.httpFault = "403"
	h.mu.Unlock()
	raw := h.Replay(h.wallets[0], 4)
	h.Push(raw)
	h.wait("persisted original receipt", 4*time.Second, func() bool { return h.Stats().Sources == 1 })
	raw.Removed = true
	h.Push(raw)
	h.wait("durable removed evidence", 3*time.Second, func() bool {
		var removed bool
		e := h.db.Pool.QueryRow(h.ctx, `SELECT removed FROM trader_sync_source_records`).Scan(&removed)
		return e == nil && removed
	})
	h.mu.Lock()
	h.httpFault = ""
	h.mu.Unlock()
	h.wait("removed candidate invalidated", 6*time.Second, func() bool {
		var state string
		e := h.db.Pool.QueryRow(h.ctx, `SELECT confirmation_state FROM trader_sync_source_records`).Scan(&state)
		return e == nil && state == "invalid"
	})
	if s := h.Stats(); s.Sources != 1 || s.Activities != 0 || s.HTTPCalls != 0 {
		t.Fatal(s)
	}
}
func TestActualSenderFailuresRetainAllAttemptEvidence(t *testing.T) {
	for _, fault := range []string{"429", "timeout"} {
		t.Run(fault, func(t *testing.T) {
			h := newHarness(t, 1, 1)
			defer h.Close()
			h.Advance(1100 * time.Millisecond)
			h.mu.Lock()
			h.sendFault = fault
			h.mu.Unlock()
			h.Push(h.Replay(h.wallets[0], 0))
			attempts := 1
			want := "unknown"
			if fault == "429" {
				attempts = 5
				want = "failed"
			}
			h.wait("definite sender terminal state", 18*time.Second, func() bool {
				var state string
				e := h.db.Pool.QueryRow(h.ctx, `SELECT status FROM account_notification_deliveries WHERE activity_id IS NOT NULL`).Scan(&state)
				return e == nil && state == want
			})
			var total, starts, returns, elapsed int
			if e := h.db.Pool.QueryRow(h.ctx, `SELECT count(*),count(started_at),count(sender_returned_at),count(sender_elapsed_ns) FROM notification_delivery_attempts`).Scan(&total, &starts, &returns, &elapsed); e != nil || total != attempts || starts != attempts || returns != attempts || elapsed != attempts {
				t.Fatal(total, starts, returns, elapsed, e)
			}
			s := h.Stats()
			if s.HTTPCalls != attempts || s.Activities != 1 || s.SideEffects != 0 {
				t.Fatal(s)
			}
			var metric string
			if e := h.db.Pool.QueryRow(context.Background(), `SELECT formation_evidence->>'cohort' FROM trader_sync_activities`).Scan(&metric); e != nil || metric != "ordinary_default" {
				t.Fatal("external result rewrote cohort", metric, e)
			}
			t.Logf("%s %s attempts=%d snapshot=%+v", fault, want, attempts, s)
			h.captureRuntime("tradersync-fault-" + fault)
			h.captureTiming("tradersync-fault-" + fault)
		})
	}
}
func TestActualPauseAndNewGenerationCannotReviveOriginalCandidate(t *testing.T) {
	h := newHarness(t, 1, 1)
	defer h.Close()
	h.Advance(1100 * time.Millisecond)
	h.mu.Lock()
	h.httpFault = "403"
	h.mu.Unlock()
	h.Push(h.Replay(h.wallets[0], 0))
	h.wait("original pending confirmation", 5*time.Second, func() bool { return h.Stats().Sources == 1 })
	owner := h.ownerIDs[0]
	var id string
	var revision uint64
	if e := h.db.Pool.QueryRow(h.ctx, `SELECT id::text,revision FROM trader_sync_subscriptions WHERE owner_id=$1`, owner).Scan(&id, &revision); e != nil {
		t.Fatal(e)
	}
	paused, e := h.service.ChangeSubscription(h.ctx, owner, "pause", tm.ChangeInput{SubscriptionID: id, RequestID: uuid.NewString(), ExpectedRevision: revision})
	if e != nil {
		t.Fatal(e)
	}
	_ = paused
	if e = h.db.Pool.QueryRow(h.ctx, `SELECT revision FROM trader_sync_subscriptions WHERE id=$1`, id).Scan(&revision); e != nil {
		t.Fatal(e)
	}
	if _, e = h.service.ChangeSubscription(h.ctx, owner, "resume", tm.ChangeInput{SubscriptionID: id, RequestID: uuid.NewString(), ExpectedRevision: revision}); e != nil {
		t.Fatal(e)
	}
	h.mu.Lock()
	h.httpFault = ""
	h.mu.Unlock()
	h.wait("old generation made permanently ineligible", 8*time.Second, func() bool {
		var state string
		e := h.db.Pool.QueryRow(h.ctx, `SELECT disposition FROM trader_sync_source_candidates`).Scan(&state)
		return e == nil && state == "ineligible"
	})
	if s := h.Stats(); s.Activities != 0 || s.HTTPCalls != 0 || s.Sources != 1 {
		t.Fatal(fmt.Sprint(s))
	}
}

func TestActualSummaryFreezesWSSActivitiesIntoOneDelivery(t *testing.T) {
	h := newHarness(t, 1, 1)
	defer h.Close()
	h.Advance(1100 * time.Millisecond)
	release := make(chan struct{})
	defer func() {
		select {
		case <-release:
		default:
			close(release)
		}
	}()
	h.mu.Lock()
	h.sendHold = release
	h.mu.Unlock()
	for i := 0; i < 13; i++ {
		h.Push(h.Replay(h.wallets[0], 0))
	}
	h.wait("13 formed while first real request held", 8*time.Second, func() bool { s := h.Stats(); return s.Activities == 13 && s.HTTPCalls == 1 })
	close(release)
	h.wait("ten ordinary and one multi-activity summary sent", 18*time.Second, func() bool {
		var sent int
		e := h.db.Pool.QueryRow(h.ctx, `SELECT count(*) FROM account_notification_deliveries WHERE status='sent'`).Scan(&sent)
		return e == nil && sent == 11
	})
	var members, parts, batches int
	var started bool
	if e := h.db.Pool.QueryRow(h.ctx, `SELECT (SELECT count(*) FROM trader_sync_summary_part_items),(SELECT count(*) FROM trader_sync_summary_parts),(SELECT count(*) FROM trader_sync_summary_batches),(SELECT first_started_at IS NOT NULL FROM trader_sync_summary_batches)`).Scan(&members, &parts, &batches, &started); e != nil || members != 3 || parts != 1 || batches != 1 || !started {
		t.Fatal(members, parts, batches, started, e)
	}
	if s := h.Stats(); s.HTTPCalls != 11 || s.Activities != 13 || s.SideEffects != 0 {
		t.Fatal(s)
	}
	metrics := h.captureRuntime("tradersync-summary-three-members")
	for name, want := range map[string]string{"timing_activity_all_total": "13", "timing_activity_summary_total": "3", "timing_logical_sent": "11", "timing_attempts_total": "11", "timing_activity_all_no_ack": "0"} {
		if metrics[name] != want {
			t.Fatalf("%s=%s want %s", name, metrics[name], want)
		}
	}
}
