package notification

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	log "github.com/sirupsen/logrus"
	"github.com/useryege/athena/internal/accountstate/txgate"
	"github.com/useryege/athena/internal/notification/delivery"
	ns "github.com/useryege/athena/internal/notification/store"
	ts "github.com/useryege/athena/internal/tradersync/store"
	"sync"
	"time"
)

type summaryReady struct {
	candidate           delivery.Candidate
	batchID, deliveryID int64
	revision            uint64
}
type summaryObserved struct {
	permit        delivery.Permit
	batchID       int64
	at, monotonic time.Time
	missing       bool
}

// SummarySource owns no pool or background loop: the shared Dispatcher owns its
// Dispatch invocations and joins them before the CLI closes notification storage.
type SummarySource struct {
	service  *Service
	pool     *pgxpool.Pool
	trader   *ts.SQLStore
	beginTx  func(context.Context, *pgxpool.Conn) (pgx.Tx, error)
	mu       sync.Mutex
	ready    map[delivery.WorkRef]summaryReady
	observed map[string]summaryObserved
}

// ConfigureSummaries is startup-only; pool must be the notification store's exact
// borrowed pool. Task12 supplies the validated site URL here. No second pool opens.
func (s *Service) ConfigureSummaries(pool *pgxpool.Pool, siteURL string) error {
	s.startStopMu.Lock()
	defer s.startStopMu.Unlock()
	if s.started {
		return fmt.Errorf("summaries must be configured before Start")
	}
	if s.store == nil {
		return fmt.Errorf("notification store is required")
	}
	existing, e := s.store.BorrowPool()
	if e != nil {
		return e
	}
	if pool != existing {
		return fmt.Errorf("summaries require the shared notification pool")
	}
	trader := ts.NewSQLStore(pool)
	if e = trader.ConfigureActivities(siteURL); e != nil {
		return e
	}
	s.summarySource = &SummarySource{service: s, pool: pool, trader: trader, beginTx: func(ctx context.Context, c *pgxpool.Conn) (pgx.Tx, error) {
		return c.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	}, ready: map[delivery.WorkRef]summaryReady{}, observed: map[string]summaryObserved{}}
	return nil
}
func (s *SummarySource) Ready(ctx context.Context, now time.Time) ([]delivery.Candidate, error) {
	// A failed start write never drops the in-process monotonic evidence. Retry its
	// fact write under a bounded gate without keeping the original HTTP call locked.
	s.mu.Lock()
	var missing []summaryObserved
	for _, o := range s.observed {
		if o.missing {
			missing = append(missing, o)
		}
	}
	s.mu.Unlock()
	for _, o := range missing {
		recordCtx, cancel := context.WithTimeout(ctx, time.Second)
		e := txgate.WithAccountTx(recordCtx, s.pool, o.permit.OwnerID, func(tx pgx.Tx) error {
			return s.service.store.RecordSummaryStartedTx(recordCtx, tx, o.batchID, o.permit, o.at)
		})
		cancel()
		if e == nil {
			s.mu.Lock()
			current := s.observed[o.permit.OwnerID]
			if current.permit.AttemptID == o.permit.AttemptID {
				current.missing = false
				s.observed[o.permit.OwnerID] = current
			}
			s.mu.Unlock()
		}
	}
	rows, e := s.service.store.SummaryHeads(ctx)
	if e != nil {
		return nil, e
	}
	out := make([]delivery.Candidate, 0, len(rows))
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ready = map[delivery.WorkRef]summaryReady{}
	for _, r := range rows {
		owner := uuid.UUID(r.OwnerID.Bytes).String()
		start := r.NotBefore.Time
		deadline := r.OldestAt.Time.Add(time.Minute)
		if !r.CurrentBatchID.Valid && r.PreviousBasisAt.Valid && r.PreviousBasisAt.Time.Add(time.Minute).After(start) {
			start = r.PreviousBasisAt.Time.Add(time.Minute)
		}
		if o, ok := s.observed[owner]; ok && !r.CurrentBatchID.Valid && o.monotonic.Add(time.Minute).After(start) {
			start = o.monotonic.Add(time.Minute)
		}
		// This is a scheduler identity only. Every persisted attempt is still account
		// work, and onStarted consumes this original head reservation, never its part ID.
		c := delivery.Candidate{Ref: delivery.WorkRef{Kind: "summary_head", ID: r.ID}, OwnerID: owner, ChatID: r.ChatID, NotBefore: start, Deadline: &deadline}
		s.ready[c.Ref] = summaryReady{candidate: c, batchID: r.CurrentBatchID.Int64, deliveryID: r.DeliveryID, revision: uint64(r.BindingRevision)}
		out = append(out, c)
	}
	return out, nil
}

func (s *SummarySource) Dispatch(ctx context.Context, c delivery.Candidate, onStarted func(time.Time)) error {
	s.mu.Lock()
	work, ok := s.ready[c.Ref]
	o := s.observed[c.OwnerID]
	s.mu.Unlock()
	if !ok || work.candidate.OwnerID != c.OwnerID || work.candidate.ChatID != c.ChatID {
		return ErrDispatchDeferred
	}
	clock := Clock(wallClock{})
	var budget *Budget
	if slot, ok := ctx.Value(dispatchSlotKey{}).(dispatchSlot); ok {
		clock, budget = slot.clock, slot.budget
	}
	if work.batchID == 0 && !o.monotonic.IsZero() && o.monotonic.Add(time.Minute).After(clock.Now()) {
		return ErrDispatchDeferred
	}
	begun := clock.Now()
	gate, e := txgate.AcquireAccountSession(ctx, s.pool, c.OwnerID)
	if e != nil {
		return e
	}
	a := &summaryAttempt{source: s, gate: gate, clock: clock, budget: budget, batchID: work.batchID, gateWait: clock.Now().Sub(begun)}
	defer a.releaseGate()
	tx, e := s.beginTx(ctx, gate.Conn)
	if e != nil {
		return e
	}
	var finish func(bool)
	committed := false
	defer func() {
		_ = tx.Rollback(context.Background())
		if finish != nil {
			finish(committed)
		}
	}()
	if work.batchID == 0 {
		batch, e := s.trader.FreezeSummaryTx(ctx, tx, c.OwnerID, work.revision, c.ChatID)
		if e != nil {
			if errors.Is(e, ns.ErrSummaryNotReady) {
				return ErrDispatchDeferred
			}
			return e
		}
		a.batchID = batch.ID
		work.deliveryID = batch.Parts[0].DeliveryID
	} else {
		h, e := s.service.store.LockSummaryHeadTx(ctx, tx, c.OwnerID)
		if e != nil {
			return e
		}
		if !h.CurrentBatchID.Valid || h.CurrentBatchID.Int64 != work.batchID {
			return ErrDispatchDeferred
		}
	}
	actual := delivery.Candidate{Ref: delivery.WorkRef{Kind: "account", ID: work.deliveryID}, OwnerID: c.OwnerID, ChatID: c.ChatID}
	p, e := s.service.store.AuthorizeTx(ctx, tx, actual, s.service.senderIncarnation, func() error {
		if s.service.senderSession != nil {
			if e := s.service.senderSession.Check(ctx); e != nil {
				return e
			}
		}
		var e error
		finish, e = beginDispatchAuthorization(ctx)
		return e
	})
	if e != nil {
		if errors.Is(e, ns.ErrDeliveryNotEligible) {
			return ErrDispatchDeferred
		}
		return e
	}
	a.permit = p
	if e = s.service.store.AttachSummaryPermitTx(ctx, tx, a.batchID, p); e != nil {
		return e
	}
	e = tx.Commit(ctx)
	commitUncertain := e != nil
	if commitUncertain {
		// A lost acknowledgement is not permission to try another attempt. Read the
		// exact original durable identity in a bounded independent transaction. This
		// proves the permit, not that the old physical session still owns its lock.
		_ = tx.Rollback(context.Background())
		checkCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), time.Second)
		confirmed := s.service.store.ConfirmSummaryPermit(checkCtx, a.batchID, p)
		cancel()
		if confirmed != nil {
			return fmt.Errorf("summary permit commit unconfirmed: %w (readback: %v)", e, confirmed)
		}
	}
	committed = true
	if finish != nil {
		finish(true)
		finish = nil
	}
	if commitUncertain {
		// End the authorization budget guard before any new account acquisition.
		// Even a live-looking pooled object may have lost its PostgreSQL session.
		// Release/discard it and let this same invocation reenter and validate the
		// original permit under a new gate. No new authorization or slot is created.
		a.releaseGate()
		if a.releaseErr != nil {
			log.WithError(a.releaseErr).WithField("batch_id", a.batchID).Warn("discarded uncertain summary commit session before reentry")
			a.releaseErr = nil // Release already discarded the physical connection.
		}
	}
	// The single invocation owns gate/admission until its Started signal or a
	// definite pre-start return. Existing grant revocation cannot replace this permit.
	outcome, e := s.service.executePermit(ctx, p, onStarted, a)
	a.releaseGate()
	recordCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), time.Second)
	waitErr := s.service.store.RecordSummaryWait(recordCtx, a.batchID, a.waitStart, a.waitEnd, a.budgetWait, a.gateWait, a.waitReason)
	cancel()
	if waitErr != nil {
		return waitErr
	}
	if e != nil {
		return e
	}
	if a.releaseErr != nil {
		log.WithError(a.releaseErr).WithField("batch_id", a.batchID).Warn("summary session discarded after failed unlock")
	}
	if outcome.Code == "recipient_unreachable" {
		if e = s.service.store.MarkTelegramBindingUnreachable(ctx, p.OwnerID, p.ChatID, int64(work.revision), outcome.Code); e != nil {
			return e
		}
	}
	if outcome.RetryAfter > 0 {
		return &RateLimitError{RetryAfter: outcome.RetryAfter}
	}
	return nil
}

type summaryAttempt struct {
	source               *SummarySource
	gate                 *txgate.AccountSession
	permit               delivery.Permit
	batchID              int64
	budget               *Budget
	clock                Clock
	waitStart, waitEnd   time.Time
	budgetWait, gateWait time.Duration
	waitReason           string
	releaseErr           error
}

func (a *summaryAttempt) releaseGate() {
	if a.gate == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if e := a.gate.Release(ctx); e != nil && a.releaseErr == nil {
		a.releaseErr = e
	}
	a.gate = nil
}
func (a *summaryAttempt) admit(ctx context.Context) (func(), error) {
	for {
		if e := ctx.Err(); e != nil {
			return nil, e
		}
		if a.gate == nil {
			began := a.clock.Now()
			gate, e := txgate.AcquireAccountSession(ctx, a.source.pool, a.permit.OwnerID)
			a.gateWait += a.clock.Now().Sub(began)
			if e != nil {
				return nil, e
			}
			a.gate = gate
			if a.source.service.senderSession != nil {
				if e = a.source.service.senderSession.Check(ctx); e != nil {
					return nil, e
				}
			}
			tx, e := gate.Conn.Begin(ctx)
			if e != nil {
				return nil, e
			}
			e = a.source.service.store.ValidateSummaryPermitTx(ctx, tx, a.batchID, a.permit)
			if e != nil {
				_ = tx.Rollback(context.Background())
				return nil, e
			}
			if e = tx.Commit(ctx); e != nil {
				return nil, e
			}
		}
		if a.budget == nil {
			return func() {}, nil
		}
		release, _, _, e := a.budget.tryAdmitStart(ctx, a.clock)
		if e != nil {
			return nil, e
		}
		if release != nil {
			return release, nil
		}
		a.releaseGate()
		if a.releaseErr != nil {
			return nil, a.releaseErr
		}
		start := a.clock.Now()
		if a.waitStart.IsZero() {
			a.waitStart = start.UTC()
		}
		// Do not carry its read lock into account acquisition: final admission is
		// checked again after that cancellable acquisition and same-permit readback.
		release, e = a.budget.admitStartObserved(ctx, a.clock, func(reason string) {
			if a.waitReason == "" {
				a.waitReason = reason
			} else if a.waitReason != reason {
				a.waitReason = "telegram_retry_after_and_budget_coordination"
			}
		})
		end := a.clock.Now()
		a.waitEnd = end.UTC()
		a.budgetWait += end.Sub(start)
		if release != nil {
			release()
		}
		if e != nil {
			return nil, e
		}
	}
}
func (a *summaryAttempt) recordStart(ctx context.Context, at time.Time) error {
	o := summaryObserved{permit: a.permit, batchID: a.batchID, at: at, monotonic: a.clock.Now(), missing: true}
	a.source.mu.Lock()
	a.source.observed[a.permit.OwnerID] = o
	a.source.mu.Unlock()
	defer a.releaseGate()
	if a.gate == nil {
		return fmt.Errorf("summary start lacks held session")
	}
	tx, e := a.gate.Conn.Begin(ctx)
	if e != nil {
		return e
	}
	defer tx.Rollback(context.Background())
	if e = a.source.service.store.RecordSummaryStartedTx(ctx, tx, a.batchID, a.permit, at); e != nil {
		return e
	}
	if e = tx.Commit(ctx); e != nil {
		return e
	}
	a.source.mu.Lock()
	o.missing = false
	a.source.observed[a.permit.OwnerID] = o
	a.source.mu.Unlock()
	return nil
}
