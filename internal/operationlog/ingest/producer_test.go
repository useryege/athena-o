package ingest

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"github.com/useryege/athena/internal/operationlog/event"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"
)

type testSink struct {
	append  func(context.Context, event.Event) error
	publish func(context.Context, Status) error
}

func (s testSink) Append(c context.Context, e event.Event) error {
	if s.append != nil {
		return s.append(c, e)
	}
	return nil
}
func (s testSink) PublishStatus(c context.Context, v Status) error {
	if s.publish != nil {
		return s.publish(c, v)
	}
	return nil
}
func testEvent(p *Producer) event.Event {
	d := int64(0)
	return event.Event{SchemaVersion: 1, EventID: uuid.NewString(), OperationID: uuid.NewString(), RequestID: uuid.NewString(), ProducerID: p.ID(), Phase: event.Finish, StartedAt: time.Now().UTC(), OccurredAt: time.Now().UTC(), Actor: event.Actor{Role: "UNKNOWN", Realm: "UNKNOWN", CredentialKind: "UNAUTHENTICATED"}, ActionCode: "identity.login", ModuleCode: "identity", Outcome: event.Unknown, Observation: event.FinishOnly, DurationMs: &d, ResourcesComplete: true}
}
func balanced(t *testing.T, s Status) {
	t.Helper()
	if s.AttemptedEvents != s.ConfirmedEvents+s.UnconfirmedEvents+s.InvalidEvents+s.CapacityRejectedEvents+s.InFlightEvents {
		t.Fatalf("non-atomic counters: %+v", s)
	}
}
func TestRetrySameEventAndDetachedContext(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var calls int
		var first string
		p := New(testSink{append: func(ctx context.Context, e event.Event) error {
			calls++
			if ctx.Err() != nil {
				t.Error("inherited cancellation")
			}
			deadline, ok := ctx.Deadline()
			if !ok || time.Until(deadline) > 100*time.Millisecond {
				t.Error("unbounded attempt")
			}
			if calls == 1 {
				first = e.EventID
				return errors.New("ack lost")
			}
			if first != e.EventID {
				t.Error("retry changed identity")
			}
			return nil
		}})
		defer p.Close(context.Background())
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if err := p.Record(ctx, testEvent(p), NewBudget()); err != nil {
			t.Fatal(err)
		}
		s := p.Snapshot()
		balanced(t, s)
		if calls != 2 || s.ConfirmedEvents != 1 || s.AttemptedEvents != 1 || s.UnconfirmedEvents != 0 {
			t.Fatalf("calls=%d status=%+v", calls, s)
		}
	})
}
func TestSharedBudgetIsCumulativeWaitNotBusinessDeadline(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var calls atomic.Int64
		p := New(testSink{append: func(ctx context.Context, _ event.Event) error { calls.Add(1); <-ctx.Done(); return ctx.Err() }})
		defer p.Close(context.Background())
		budget := NewBudget()
		before := time.Now()
		if p.Record(context.Background(), testEvent(p), budget) == nil {
			t.Fatal("missing unconfirmed")
		}
		if time.Since(before) != 200*time.Millisecond {
			t.Fatalf("phase wait %s", time.Since(before))
		}
		time.Sleep(time.Hour) // Business work does not consume logging budget.
		before = time.Now()
		_ = p.Record(context.Background(), testEvent(p), budget)
		if time.Since(before) != 200*time.Millisecond {
			t.Fatal("budget was a wall deadline")
		}
		before = time.Now()
		if p.Record(context.Background(), testEvent(p), budget) == nil {
			t.Fatal("budget exhaustion accepted")
		}
		if time.Since(before) != 0 || calls.Load() != 4 {
			t.Fatalf("extra wait/calls: %s %d", time.Since(before), calls.Load())
		}
		s := p.Snapshot()
		balanced(t, s)
		if s.AttemptedEvents != 3 || s.UnconfirmedEvents != 3 {
			t.Fatalf("status %+v", s)
		}
	})
}
func TestCapacityAndNonCooperativeSinkRemainBounded(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		block := make(chan struct{})
		var calls atomic.Int64
		p := New(testSink{append: func(context.Context, event.Event) error { calls.Add(1); <-block; return nil }})
		var wg sync.WaitGroup
		for i := 0; i < 16; i++ {
			wg.Go(func() { _ = p.Record(context.Background(), testEvent(p), NewBudget()) })
		}
		synctest.Wait()
		before := time.Now()
		if err := p.Record(context.Background(), testEvent(p), NewBudget()); !errors.Is(err, ErrCapacity) {
			t.Fatalf("expected capacity: %v", err)
		}
		if time.Since(before) != 0 {
			t.Fatal("capacity queued")
		}
		time.Sleep(200 * time.Millisecond)
		wg.Wait()
		if calls.Load() != 16 {
			t.Fatal("worker bound exceeded")
		}
		s := p.Snapshot()
		balanced(t, s)
		if s.CapacityRejectedEvents != 1 || s.UnconfirmedEvents != 16 {
			t.Fatalf("status %+v", s)
		}
		before = time.Now()
		if p.Close(context.Background()) == nil || time.Since(before) > 200*time.Millisecond {
			t.Fatal("close not bounded")
		}
		close(block)
		synctest.Wait()
		if err := p.Close(context.Background()); err != nil {
			t.Fatal(err)
		}
	})
}
func TestInvalidConflictAndPanicAreSeparateTerminalCounts(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		calls := 0
		p := New(testSink{append: func(context.Context, event.Event) error {
			calls++
			if calls == 1 {
				return ErrConflict
			}
			panic("secret payload")
		}})
		defer p.Close(context.Background())
		invalid := testEvent(p)
		invalid.SchemaVersion = 2
		if p.Record(context.Background(), invalid, nil) == nil {
			t.Fatal("accepted invalid")
		}
		if p.Record(context.Background(), testEvent(p), nil) == nil {
			t.Fatal("accepted conflict")
		}
		if p.Record(context.Background(), testEvent(p), nil) == nil {
			t.Fatal("accepted panic")
		}
		s := p.Snapshot()
		balanced(t, s)
		if s.AttemptedEvents != 3 || s.InvalidEvents != 2 || s.UnconfirmedEvents != 1 {
			t.Fatalf("status %+v", s)
		}
	})
}
func TestStatusPublishesCumulativeSnapshotsAndStops(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var snapshots []Status
		p := New(testSink{publish: func(ctx context.Context, s Status) error {
			if _, ok := ctx.Deadline(); !ok {
				t.Error("unbounded status")
			}
			snapshots = append(snapshots, s)
			if len(snapshots) == 1 {
				return errors.New("offline")
			}
			return nil
		}})
		_ = p.Record(context.Background(), testEvent(p), nil)
		time.Sleep(5 * time.Second)
		synctest.Wait()
		_ = p.Record(context.Background(), testEvent(p), nil)
		time.Sleep(5 * time.Second)
		synctest.Wait()
		if len(snapshots) != 2 || snapshots[0].ConfirmedEvents != 1 || snapshots[1].ConfirmedEvents != 2 || snapshots[1].SnapshotNo <= snapshots[0].SnapshotNo {
			t.Fatalf("snapshots %+v", snapshots)
		}
		if err := p.Close(context.Background()); err != nil {
			t.Fatal(err)
		}
		n := len(snapshots)
		time.Sleep(10 * time.Second)
		if len(snapshots) != n {
			t.Fatal("status continued after close")
		}
	})
}
