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
		if p.Snapshot().PersistenceReachable != nil {
			t.Fatal("capacity rejection invented persistence failure")
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

func TestProducerLifecycleObservesFailureRecoveryWithoutRewritingHistory(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		offline := true
		p := New(testSink{append: func(context.Context, event.Event) error {
			if offline {
				return errors.New("private database detail")
			}
			return nil
		}})
		defer p.Close(context.Background())
		initial := p.Snapshot()
		if initial.StartedAt.IsZero() || initial.PersistenceReachable != nil || initial.LastConfirmedAt != nil || initial.LastRecoveredAt != nil || initial.StoppedAt != nil {
			t.Fatalf("invented initial facts %+v", initial)
		}
		started := initial.StartedAt
		if err := p.Record(context.Background(), testEvent(p), nil); err == nil {
			t.Fatal("offline accepted")
		}
		failed := p.Snapshot()
		if failed.PersistenceReachable == nil || *failed.PersistenceReachable || failed.LastFailureAt == nil || failed.LastFailureCode == nil || *failed.LastFailureCode != "event_unconfirmed" || failed.LastRecoveredAt != nil || failed.UnconfirmedEvents != 1 {
			t.Fatalf("missing failure facts %+v", failed)
		}
		time.Sleep(time.Second)
		offline = false
		if err := p.Record(context.Background(), testEvent(p), nil); err != nil {
			t.Fatal(err)
		}
		recovered := p.Snapshot()
		balanced(t, recovered)
		if recovered.StartedAt != started || recovered.PersistenceReachable == nil || !*recovered.PersistenceReachable || recovered.LastRecoveredAt == nil || !recovered.LastRecoveredAt.After(*failed.LastFailureAt) || recovered.LastConfirmedAt == nil || recovered.UnconfirmedEvents != 1 || recovered.ConfirmedEvents != 1 {
			t.Fatalf("missing recovery or erased history %+v", recovered)
		}
		// Snapshots may be mutated by a caller without changing producer state.
		*recovered.LastRecoveredAt = time.Time{}
		*recovered.PersistenceReachable = false
		next := p.Snapshot()
		if next.LastRecoveredAt.IsZero() || !*next.PersistenceReachable {
			t.Fatal("snapshot pointers alias producer")
		}
	})
}
func TestProducerInitialSuccessIsNotRecoveryAndInvalidDoesNotClaimUnreachable(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		p := New(testSink{})
		defer p.Close(context.Background())
		bad := testEvent(p)
		bad.SchemaVersion = 2
		_ = p.Record(context.Background(), bad, nil)
		s := p.Snapshot()
		if s.PersistenceReachable != nil || s.LastFailureCode == nil || *s.LastFailureCode != "invalid_event" {
			t.Fatalf("invalid was treated as connectivity %+v", s)
		}
		_ = p.Record(context.Background(), testEvent(p), nil)
		s = p.Snapshot()
		if s.PersistenceReachable == nil || !*s.PersistenceReachable || s.LastRecoveredAt != nil || s.LastConfirmedAt == nil {
			t.Fatalf("initial success fabricated recovery %+v", s)
		}
	})
}
func TestClosePublishesFinalStoppedSnapshotAfterDrain(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var reports []Status
		p := New(testSink{append: func(ctx context.Context, _ event.Event) error { time.Sleep(50 * time.Millisecond); return nil }, publish: func(ctx context.Context, s Status) error {
			deadline, ok := ctx.Deadline()
			if !ok || time.Until(deadline) > 100*time.Millisecond {
				t.Error("unbounded final report")
			}
			reports = append(reports, s)
			return nil
		}})
		done := make(chan struct{})
		go func() { _ = p.Record(context.Background(), testEvent(p), nil); close(done) }()
		synctest.Wait()
		before := time.Now()
		if err := p.Close(context.Background()); err != nil {
			t.Fatal(err)
		}
		<-done
		if time.Since(before) > 200*time.Millisecond || len(reports) != 1 || reports[0].StoppedAt == nil || reports[0].ConfirmedEvents != 1 || reports[0].InFlightEvents != 0 {
			t.Fatalf("missing final stopped report %+v", reports)
		}
		s := p.Snapshot()
		if s.StoppedAt == nil || s.PersistenceReachable == nil || !*s.PersistenceReachable {
			t.Fatalf("missing local stop %+v", s)
		}
		n := len(reports)
		_ = p.Close(context.Background())
		time.Sleep(10 * time.Second)
		if len(reports) != n {
			t.Fatal("close repeated final report")
		}
	})
}
func TestStatusFailureAndRecoveryAreObservedWithoutConfirmingEvents(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		calls := 0
		p := New(testSink{publish: func(context.Context, Status) error {
			calls++
			if calls == 1 {
				return errors.New("offline")
			}
			return nil
		}})
		defer p.Close(context.Background())
		time.Sleep(5 * time.Second)
		synctest.Wait()
		s := p.Snapshot()
		if s.PersistenceReachable == nil || *s.PersistenceReachable || s.LastFailureCode == nil || *s.LastFailureCode != "status_unconfirmed" {
			t.Fatalf("missing status failure %+v", s)
		}
		time.Sleep(5 * time.Second)
		synctest.Wait()
		s = p.Snapshot()
		if s.PersistenceReachable == nil || !*s.PersistenceReachable || s.LastRecoveredAt == nil || s.LastConfirmedAt != nil || s.ConfirmedEvents != 0 {
			t.Fatalf("status ack fabricated event ack %+v", s)
		}
	})
}
func TestNonCooperativeStatusIsSingleFlightAndCloseBounded(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		block := make(chan struct{})
		var calls atomic.Int64
		p := New(testSink{publish: func(context.Context, Status) error { calls.Add(1); <-block; return nil }})
		defer func() {
			select {
			case <-block:
			default:
				close(block)
			}
			_ = p.Close(context.Background())
		}()
		time.Sleep(5200 * time.Millisecond)
		synctest.Wait()
		s := p.Snapshot()
		if s.PersistenceReachable == nil || *s.PersistenceReachable {
			t.Fatalf("status timeout not observed %+v", s)
		}
		time.Sleep(10 * time.Second)
		synctest.Wait()
		if calls.Load() != 1 {
			t.Fatal("spawned concurrent status publishers")
		}
		before := time.Now()
		if p.Close(context.Background()) == nil || time.Since(before) > 200*time.Millisecond {
			t.Fatal("close of stuck status not bounded")
		}
		if p.Snapshot().StoppedAt != nil {
			t.Fatal("claimed stopped while status worker still running")
		}
		if calls.Load() != 1 {
			t.Fatal("final report overlaps stuck report")
		}
		close(block)
		synctest.Wait()
		if err := p.Close(context.Background()); err != nil {
			t.Fatal(err)
		}
		if p.Snapshot().StoppedAt == nil {
			t.Fatal("missing actual eventual stop")
		}
	})
}
func TestFinalReportFailureIsNotPersistenceOrRecoverySuccess(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		p := New(testSink{publish: func(context.Context, Status) error { return errors.New("offline") }})
		if err := p.Close(context.Background()); err != nil {
			t.Fatal(err)
		}
		s := p.Snapshot()
		if s.StoppedAt == nil || s.PersistenceReachable == nil || *s.PersistenceReachable || s.LastRecoveredAt != nil || s.LastFailureCode == nil || *s.LastFailureCode != "status_unconfirmed" {
			t.Fatalf("fabricated successful stop reporting %+v", s)
		}
	})
}

func TestFinalNonCooperativeReportCannotClaimStopped(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		block := make(chan struct{})
		calls := atomic.Int64{}
		p := New(testSink{publish: func(_ context.Context, s Status) error {
			calls.Add(1)
			if s.StoppedAt == nil {
				t.Error("final payload omitted stop fact")
			}
			<-block
			return nil
		}})
		defer func() {
			select {
			case <-block:
			default:
				close(block)
			}
			_ = p.Close(context.Background())
		}()
		start := time.Now()
		if p.Close(context.Background()) == nil || time.Since(start) > 200*time.Millisecond {
			t.Fatal("final publish escaped close budget")
		}
		s := p.Snapshot()
		if calls.Load() != 1 || s.StoppedAt != nil || s.PersistenceReachable == nil || *s.PersistenceReachable || s.LastRecoveredAt != nil {
			t.Fatalf("false final success %+v calls=%d", s, calls.Load())
		}
		close(block)
		synctest.Wait()
		if err := p.Close(context.Background()); err != nil {
			t.Fatal(err)
		}
		s = p.Snapshot()
		if s.StoppedAt == nil || *s.PersistenceReachable || s.LastRecoveredAt != nil {
			t.Fatalf("late out-of-budget acknowledgement invented recovery %+v", s)
		}
	})
}

func TestBudgetAndClosedRejectionsDoNotClaimPersistenceFailure(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		offline := true
		p := New(testSink{append: func(ctx context.Context, _ event.Event) error {
			if offline {
				<-ctx.Done()
				return ctx.Err()
			}
			return nil
		}})
		defer p.Close(context.Background())
		budget := NewBudget()
		_ = p.Record(context.Background(), testEvent(p), budget)
		_ = p.Record(context.Background(), testEvent(p), budget)
		offline = false
		_ = p.Record(context.Background(), testEvent(p), nil)
		before := p.Snapshot()
		if err := p.Record(context.Background(), testEvent(p), budget); !errors.Is(err, ErrBudget) {
			t.Fatalf("budget err=%v", err)
		}
		after := p.Snapshot()
		if after.PersistenceReachable == nil || !*after.PersistenceReachable || after.LastConfirmedAt == nil || !after.LastConfirmedAt.Equal(*before.LastConfirmedAt) || *after.LastFailureCode != "request_budget_exhausted" || after.UnconfirmedEvents != before.UnconfirmedEvents+1 {
			t.Fatalf("budget changed persistence facts %+v", after)
		}
		if err := p.Close(context.Background()); err != nil {
			t.Fatal(err)
		}
		stopped := p.Snapshot()
		if err := p.Record(context.Background(), testEvent(p), nil); !errors.Is(err, ErrClosed) {
			t.Fatalf("closed err=%v", err)
		}
		final := p.Snapshot()
		if final.PersistenceReachable == nil || !*final.PersistenceReachable || final.AttemptedEvents != stopped.AttemptedEvents || final.LastFailureCode == nil || *final.LastFailureCode != *stopped.LastFailureCode {
			t.Fatal("closed producer accepted accounting or invented persistence failure")
		}
	})
}
