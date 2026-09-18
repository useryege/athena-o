// Package ingest provides bounded, best-effort collection independent of the
// business request's cancellation. Sink implementations must respect deadlines.
package ingest

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/useryege/athena/internal/operationlog/event"
)

const AttemptTimeout = 100 * time.Millisecond
const PhaseBudget = 200 * time.Millisecond
const RequestBudget = 400 * time.Millisecond
const Capacity = 16
const StatusInterval = 5 * time.Second

var ErrConflict = errors.New("operation log event conflict")
var ErrUnconfirmed = errors.New("operation log submission unconfirmed")
var ErrCapacity = errors.New("operation log capacity exceeded")
var ErrClosed = errors.New("operation log producer closed")
var ErrBudget = errors.New("operation log request budget exhausted")

type Sink interface {
	Append(context.Context, event.Event) error
	PublishStatus(context.Context, Status) error
}
type Budget struct {
	mu        sync.Mutex
	remaining time.Duration
}

func NewBudget() *Budget                   { return &Budget{remaining: RequestBudget} }
func (b *Budget) Remaining() time.Duration { b.mu.Lock(); defer b.mu.Unlock(); return b.remaining }

// A budget reserves at most one phase's allowance and refunds unused time.
// Concurrent children cannot reserve the same allowance; business execution
// between calls does not consume it.
func (b *Budget) reserve() (time.Duration, func(time.Duration)) {
	if b == nil {
		return PhaseBudget, func(time.Duration) {}
	}
	b.mu.Lock()
	n := min(b.remaining, PhaseBudget)
	b.remaining -= n
	b.mu.Unlock()
	return n, func(used time.Duration) { b.mu.Lock(); b.remaining += max(0, n-used); b.mu.Unlock() }
}

type Producer struct {
	id             string
	sink           Sink
	mu             sync.Mutex
	status         Status
	closed         bool
	slots          chan struct{}
	stop           chan struct{}
	done           chan struct{}
	workers        sync.WaitGroup
	records        sync.WaitGroup
	statusWorkers  sync.WaitGroup
	statusSlot     chan struct{}
	drained        chan struct{}
	reporterDone   chan struct{}
	shutdown       context.Context
	shutdownCancel context.CancelFunc
	lastLog        map[string]time.Time
}

func New(s Sink) *Producer {
	p := &Producer{id: uuid.NewString(), sink: s, slots: make(chan struct{}, Capacity), stop: make(chan struct{}), done: make(chan struct{}), lastLog: map[string]time.Time{}, statusSlot: make(chan struct{}, 1), drained: make(chan struct{}), reporterDone: make(chan struct{})}
	p.status.ProducerID = p.id
	p.status.StartedAt = time.Now().UTC()
	go p.reportLoop()
	go func() {
		<-p.stop
		p.records.Wait()
		p.workers.Wait()
		close(p.drained)
		<-p.reporterDone
		p.statusWorkers.Wait()
		p.mu.Lock()
		stopped := time.Now().UTC()
		p.status.StoppedAt = &stopped
		p.mu.Unlock()
		p.shutdownCancel()
		close(p.done)
	}()
	return p
}

func (p *Producer) ID() string {
	if p == nil {
		return ""
	}
	return p.id
}

// Record returns collection diagnostics only. Callers must never substitute
// them for the original business response or retry the business operation.
func (p *Producer) Record(ctx context.Context, e event.Event, b *Budget) (result error) {
	if p == nil {
		return ErrClosed
	}
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return ErrClosed
	}
	p.records.Add(1)
	defer p.records.Done()
	p.status.AttemptedEvents++
	p.status.InFlightEvents++
	p.mu.Unlock()
	defer func() {
		if recover() != nil {
			result = ErrUnconfirmed
		}
		p.mu.Lock()
		p.status.InFlightEvents--
		switch {
		case result == nil:
			p.status.ConfirmedEvents++
		case errors.Is(result, event.ErrInvalid) || errors.Is(result, ErrConflict):
			p.status.InvalidEvents++
		case errors.Is(result, ErrCapacity):
			p.status.CapacityRejectedEvents++
		default:
			p.status.UnconfirmedEvents++
		}
		now := time.Now().UTC()
		if result == nil {
			p.status.LastConfirmedAt = &now
			p.persistenceLocked(true, now)
		} else {
			p.failureLocked(failureCode(result), now)
			if errors.Is(result, ErrUnconfirmed) {
				p.persistenceLocked(false, now)
			}
		}
		p.mu.Unlock()
		if result != nil {
			p.logFailure(failureCode(result))
		}
	}()
	if e.ProducerID != p.id {
		return event.ErrInvalid
	}
	encoded, err := event.Canonical(e)
	if err != nil {
		return err
	}
	e, err = event.Decode(encoded)
	if err != nil {
		return err
	} // detached immutable snapshot
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return ErrClosed
	}
	select {
	case p.slots <- struct{}{}:
	default:
		p.mu.Unlock()
		return ErrCapacity
	}
	n, refund := b.reserve()
	if n <= 0 {
		<-p.slots
		p.mu.Unlock()
		return ErrBudget
	}
	if ctx == nil {
		ctx = context.Background()
	}
	phaseCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), n)
	start := time.Now()
	defer func() { cancel(); refund(time.Since(start)) }()
	resultCh := make(chan error, 1)
	p.workers.Go(func() {
		defer func() { <-p.slots }()
		var last error
		for attempt := 0; attempt < 2; attempt++ {
			if phaseCtx.Err() != nil {
				break
			}
			attemptCtx, stop := context.WithTimeout(phaseCtx, AttemptTimeout)
			last = p.append(attemptCtx, e)
			stop()
			if last == nil || errors.Is(last, event.ErrInvalid) || errors.Is(last, ErrConflict) {
				resultCh <- last
				return
			}
		}
		resultCh <- ErrUnconfirmed
	})
	p.mu.Unlock()
	select {
	case err := <-resultCh:
		return err
	case <-phaseCtx.Done():
		return ErrUnconfirmed
	}
}
func (p *Producer) append(ctx context.Context, e event.Event) (err error) {
	defer func() {
		if recover() != nil {
			err = ErrUnconfirmed
		}
	}()
	if p.sink == nil {
		return ErrUnconfirmed
	}
	return p.sink.Append(ctx, e)
}
func (p *Producer) Snapshot() Status {
	if p == nil {
		return Status{}
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.status.SnapshotNo++
	p.status.ObservedAt = time.Now().UTC()
	out := p.status
	out.StoppedAt = copyStatusPointer(out.StoppedAt)
	out.LastFailureAt = copyStatusPointer(out.LastFailureAt)
	out.LastFailureCode = copyStatusPointer(out.LastFailureCode)
	out.LastRecoveredAt = copyStatusPointer(out.LastRecoveredAt)
	out.LastConfirmedAt = copyStatusPointer(out.LastConfirmedAt)
	out.PersistenceReachable = copyStatusPointer(out.PersistenceReachable)
	return out
}
func copyStatusPointer[T any](v *T) *T {
	if v == nil {
		return nil
	}
	out := *v
	return &out
}

// These observations share the counter mutex. Recovery requires a prior
// observed persistence failure; initial success and invalid local input cannot
// manufacture a recovery event. Old unconfirmed event counters never change.
func (p *Producer) persistenceLocked(reachable bool, now time.Time) {
	if reachable && p.status.PersistenceReachable != nil && !*p.status.PersistenceReachable {
		p.status.LastRecoveredAt = &now
	}
	p.status.PersistenceReachable = &reachable
}
func (p *Producer) failureLocked(code string, now time.Time) {
	p.status.LastFailureAt = &now
	p.status.LastFailureCode = &code
}
func (p *Producer) publish(ctx context.Context, s Status) (err error) {
	defer func() {
		if recover() != nil {
			err = ErrUnconfirmed
		}
	}()
	if p.sink == nil {
		return ErrUnconfirmed
	}
	return p.sink.PublishStatus(ctx, s)
}

// publishSnapshot permits only one actual Sink call. A timed-out call retains
// its slot until it returns, so neither a later tick nor shutdown starts a
// replacement publisher alongside a non-cooperative sink.
func (p *Producer) publishSnapshot(ctx context.Context, s Status) {
	if ctx.Err() != nil {
		return
	}
	select {
	case p.statusSlot <- struct{}{}:
	default:
		return
	}
	result := make(chan error, 1)
	p.statusWorkers.Go(func() { err := p.publish(ctx, s); <-p.statusSlot; result <- err })
	var err error
	select {
	case err = <-result:
	case <-ctx.Done():
		err = ctx.Err()
	}
	p.mu.Lock()
	now := time.Now().UTC()
	p.persistenceLocked(err == nil, now)
	if err != nil {
		p.failureLocked("status_unconfirmed", now)
	}
	p.mu.Unlock()
	if err != nil {
		p.logFailure("status_unconfirmed")
	}
}
func (p *Producer) reportLoop() {
	defer close(p.reporterDone)
	ticker := time.NewTicker(StatusInterval)
	defer ticker.Stop()
	for {
		select {
		case <-p.stop:
			p.reportStopped()
			return
		default:
		}
		select {
		case <-p.stop:
			p.reportStopped()
			return
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), AttemptTimeout)
			p.publishSnapshot(ctx, p.Snapshot())
			cancel()
		}
	}
}
func (p *Producer) reportStopped() {
	p.mu.Lock()
	shutdown := p.shutdown
	p.mu.Unlock()
	select {
	case <-p.drained:
	case <-shutdown.Done():
		return
	}
	if shutdown.Err() != nil {
		return
	}
	// All admitted records and their actual workers have stopped. This final
	// payload announces that fact; its acknowledgement must not be pre-assumed
	// in the payload's reachability or recovery observations.
	s := p.Snapshot()
	stopped := time.Now().UTC()
	s.StoppedAt = &stopped
	ctx, cancel := context.WithTimeout(shutdown, AttemptTimeout)
	defer cancel()
	p.publishSnapshot(ctx, s)
}

// Close stops acceptance and periodic publication. Even a misbehaving sink
// cannot make shutdown wait longer than a phase; its slot remains occupied until
// it returns, preventing replacement workers from growing without a bound.
func (p *Producer) Close(ctx context.Context) error {
	if p == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	p.mu.Lock()
	if !p.closed {
		p.closed = true
		p.shutdown, p.shutdownCancel = context.WithTimeout(ctx, PhaseBudget)
		close(p.stop)
	}
	p.mu.Unlock()
	bounded, cancel := context.WithTimeout(ctx, PhaseBudget)
	defer cancel()
	select {
	case <-p.done:
		return nil
	case <-bounded.Done():
		return bounded.Err()
	}
}
func failureCode(err error) string {
	switch {
	case errors.Is(err, event.ErrInvalid):
		return "invalid_event"
	case errors.Is(err, ErrConflict):
		return "event_conflict"
	case errors.Is(err, ErrCapacity):
		return "capacity_exceeded"
	case errors.Is(err, ErrBudget):
		return "request_budget_exhausted"
	case errors.Is(err, ErrClosed):
		return "producer_closed"
	default:
		return "event_unconfirmed"
	}
}
func (p *Producer) logFailure(code string) {
	defer func() { _ = recover() }()
	p.mu.Lock()
	now := time.Now()
	last := p.lastLog[code]
	if !last.IsZero() && now.Sub(last) < 30*time.Second {
		p.mu.Unlock()
		return
	}
	p.lastLog[code] = now
	p.mu.Unlock()
	// Never expose sink errors, panic values, request fields, or identities.
	slog.Warn("operation_log_collection", "code", code)
}
