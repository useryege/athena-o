package notification

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/useryege/athena/internal/notification/apiclient"
	"github.com/useryege/athena/internal/notification/delivery"
	notificationstore "github.com/useryege/athena/internal/notification/store"
	"strconv"
	"sync"
	"time"
)

func restoreBudget(ctx context.Context, store *notificationstore.SQLStore, budget *Budget, now time.Time) error {
	evidence, err := store.BudgetEvidence(ctx, now.Add(-time.Minute))
	if err != nil {
		return err
	}
	for _, a := range evidence {
		c := delivery.Candidate{Ref: delivery.WorkRef{Kind: a.WorkKind, ID: a.WorkID}, ChatID: a.TelegramChatID, Group: a.TelegramGroup}
		if a.StartedAt.Valid {
			budget.Start(c, a.StartedAt.Time)
		} else if a.ResultAt.Valid {
			log.WithField("attempt_id", a.ID).WithField("result_at", a.ResultAt.Time).Warn("notification recovery lacks actual HTTP start; restoring conservative budget window")
			// Result time is only an upper bound for a missing start, never repaired historical evidence.
			if a.Outcome.String == "unknown" {
				budget.Tighten(a.ResultAt.Time.Add(time.Minute))
			} else {
				budget.Start(c, a.ResultAt.Time)
			}
		} else {
			return fmt.Errorf("unrecovered sender attempt %s blocks budget restoration", a.ID)
		}
		if a.ResultAt.Valid && a.RetryAfter.Valid && !a.RetryAfterReleasedAt.Valid {
			budget.Tighten(a.ResultAt.Time.Add(time.Duration(a.RetryAfter.Microseconds) * time.Microsecond))
		}
	}
	return nil
}

// retryBudget tracks only monotonic waiting. UTC age is never proof of Retry-After expiry.
type retryBudget struct {
	store   *notificationstore.SQLStore
	budget  *Budget
	clock   Clock
	mu      sync.Mutex
	pending map[uuid.UUID]time.Time
	wake    chan struct{}
}
type retryBudgetKey struct{}

func (r *retryBudget) track(id uuid.UUID, wait time.Duration) {
	if wait <= 0 {
		return
	}
	until := r.clock.Now().Add(wait)
	r.mu.Lock()
	r.pending[id] = until
	r.mu.Unlock()
	r.budget.Tighten(until)
	select {
	case r.wake <- struct{}{}:
	default:
	}
}
func (r *retryBudget) releaseElapsed(ctx context.Context) error {
	now := r.clock.Now()
	r.mu.Lock()
	var due []uuid.UUID
	for id, until := range r.pending {
		if !until.After(now) {
			due = append(due, id)
		}
	}
	r.mu.Unlock()
	for _, id := range due {
		if err := r.store.ReleaseRetryAfter(ctx, id); err != nil {
			return err
		}
		r.mu.Lock()
		delete(r.pending, id)
		r.mu.Unlock()
	}
	return nil
}
func (r *retryBudget) run(ctx context.Context) error {
	for {
		if err := r.releaseElapsed(ctx); err != nil {
			return err
		}
		next := r.clock.Now().Add(time.Hour)
		r.mu.Lock()
		for _, until := range r.pending {
			if until.Before(next) {
				next = until
			}
		}
		r.mu.Unlock()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-r.wake:
		case <-r.clock.After(max(0, next.Sub(r.clock.Now()))):
		}
	}
}

// recoverStartupBudget is called after acquiring the singleton session. firstStart is proven
// there only when both instance registrations and all attempt history were empty.
func recoverStartupBudget(ctx context.Context, store *notificationstore.SQLStore, budget *Budget, clock Clock, firstStart bool, progress *recoveryProgress) (r *retryBudget, err error) {
	progress.ensureStarted(clock)
	failureReason := "retry_after_read_failed"
	completionReason := "first_start_empty_history"
	defer func() {
		if err != nil {
			if ctx.Err() != nil {
				progress.finish("cancelled", "cancelled")
			} else {
				progress.finish("failed", failureReason)
			}
		} else {
			progress.finish("completed", completionReason)
		}
	}()
	r = &retryBudget{store: store, budget: budget, clock: clock, pending: make(map[uuid.UUID]time.Time), wake: make(chan struct{}, 1)}
	evidence, err := store.UnreleasedRetryAfters(ctx)
	if err != nil {
		return nil, err
	}
	wait := time.Duration(0)
	if !firstStart {
		wait = time.Minute
	}
	for _, item := range evidence {
		r.track(item.AttemptID, item.Wait)
		wait = max(wait, item.Wait)
	}
	until := clock.Now().Add(wait)
	if !firstStart {
		completionReason = "non_first_start_barrier"
	}
	if len(evidence) > 0 {
		completionReason = "persistent_retry_after"
		if !firstStart {
			completionReason = "barrier_and_persistent_retry_after"
		}
	}
	if wait > 0 {
		progress.waiting(until, completionReason)
	}
	failureReason = "recovery_wait_failed"
	budget.Tighten(until)
	if wait > 0 {
		began := clock.Now()
		log.WithField("recovery_wait", wait).Warn("notification local recovery delay: monotonic startup budget barrier; included in overall latency, not provider time")
		defer func() {
			log.WithField("local_recovery_delay", clock.Now().Sub(began)).WithField("cancelled", ctx.Err() != nil).Info("notification startup budget barrier ended; retain this delay in overall delivery latency")
		}()
	}
	for clock.Now().Before(until) {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-clock.After(until.Sub(clock.Now())):
		}
	}
	failureReason = "retry_after_release_failed"
	if err = r.releaseElapsed(ctx); err != nil {
		return nil, err
	}
	return r, nil
}

// recoveryProgress is owned by the existing recovery wait, not an API timer.
type recoveryProgress struct {
	mu                  sync.Mutex
	clock               Clock
	began, until, ended time.Time
	state, reason       string
}

func (p *recoveryProgress) begin(clock Clock) {
	if p == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.clock = clock
	p.began = clock.Now()
	p.until = time.Time{}
	p.ended = time.Time{}
	p.state = "initializing"
	p.reason = "initializing"
}
func (p *recoveryProgress) ensureStarted(clock Clock) {
	if p == nil {
		return
	}
	p.mu.Lock()
	empty := p.state == ""
	p.mu.Unlock()
	if empty {
		p.begin(clock)
	}
}
func (p *recoveryProgress) waiting(until time.Time, reason string) {
	if p == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.until = until
	p.state = "waiting"
	p.reason = reason
}
func (p *recoveryProgress) finish(state, reason string) {
	if p == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.clock == nil {
		return
	}
	p.ended = p.clock.Now()
	p.state = state
	p.reason = reason
}
func (p *recoveryProgress) snapshot() *apiclient.NotificationRecoveryStatus {
	if p == nil {
		return nil
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.state == "" {
		return nil
	}
	now := p.ended
	if now.IsZero() {
		now = p.clock.Now()
	}
	out := &apiclient.NotificationRecoveryStatus{State: p.state, Reason: p.reason, StartedAt: p.began.UTC().Format(time.RFC3339Nano), ElapsedMillis: strconv.FormatInt(now.Sub(p.began).Milliseconds(), 10), ClockSource: "sender_monotonic"}
	if p.state == "completed" {
		out.RemainingMillis = "0"
	} else if !p.until.IsZero() {
		out.RemainingMillis = strconv.FormatInt(max(0, p.until.Sub(now).Milliseconds()), 10)
	}
	return out
}
