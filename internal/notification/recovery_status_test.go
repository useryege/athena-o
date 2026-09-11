package notification

import (
	"testing"
	"time"
)

func TestRecoverySnapshotRetainsUnknownRemainingAndFrozenTerminalElapsed(t *testing.T) {
	c := &manualDispatchClock{now: time.Now()}
	p := &recoveryProgress{}
	if p.snapshot() != nil {
		t.Fatal("invented unstarted recovery")
	}
	p.begin(c)
	s := p.snapshot()
	if s == nil || s.State != "initializing" || s.RemainingMillis != "" || s.ElapsedMillis != "0" || s.ClockSource != "sender_monotonic" {
		t.Fatalf("initial: %+v", s)
	}
	p.waiting(c.Now().Add(time.Minute), "non_first_start_barrier")
	c.advance(5 * time.Second)
	s = p.snapshot()
	if s.RemainingMillis != "55000" || s.ElapsedMillis != "5000" || s.State != "waiting" {
		t.Fatal(s)
	}
	p.finish("cancelled", "cancelled")
	c.advance(8 * time.Second)
	s = p.snapshot()
	if s.State != "cancelled" || s.ElapsedMillis != "5000" {
		t.Fatalf("API kept timing a stopped recovery: %+v", s)
	}
	p.begin(c)
	p.finish("completed", "first_start_empty_history")
	s = p.snapshot()
	if s.State != "completed" || s.RemainingMillis != "0" || s.ElapsedMillis != "0" {
		t.Fatal(s)
	}
	p.begin(c)
	p.finish("failed", "retry_after_read_failed")
	s = p.snapshot()
	if s.State != "failed" || s.RemainingMillis != "" {
		t.Fatal(s)
	}
}
