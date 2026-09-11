package notification

import (
	"context"
	"testing"
	"time"
)

func TestSummaryFinalAdmissionNeverWaitsWithSessionGate(t *testing.T) {
	clock := &manualDispatchClock{now: time.Now()}
	b := NewBudget(20, time.Second, 20, time.Minute)
	b.Tighten(clock.Now().Add(120 * time.Second))
	release, wait, _, e := b.tryAdmitStart(context.Background(), clock)
	if release != nil {
		release()
		t.Fatal("final admission ignored active Retry-After")
	}
	if e != nil || wait != 120*time.Second {
		t.Fatal(wait, e)
	}
	clock.advance(120 * time.Second)
	b.authorizationMu.Lock()
	done := make(chan struct{})
	go func() {
		defer close(done)
		release, wait, _, e := b.tryAdmitStart(context.Background(), clock)
		if release != nil {
			release()
			t.Error("final admission ignored writer")
		}
		if e != nil || wait <= 0 {
			t.Error(wait, e)
		}
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		b.authorizationMu.Unlock()
		t.Fatal("try admission waited while account gate could be held")
	}
	b.authorizationMu.Unlock()
	release, wait, _, e = b.tryAdmitStart(context.Background(), clock)
	if e != nil || release == nil || wait != 0 {
		t.Fatal("expired cooldown unavailable", wait, e)
	}
	release()
}
