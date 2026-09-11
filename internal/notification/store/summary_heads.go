package store

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/useryege/athena/internal/accountstate/txgate"
	"github.com/useryege/athena/internal/notification/delivery"
	q "github.com/useryege/athena/internal/notification/store/sqlc"
	"time"
)

var ErrSummaryNotReady = errors.New("summary head is not ready")

// LockSummaryHeadTx requires the caller's owner gate. No independent connection
// or recursive account lock is taken while the summary session owns that gate.
func (s *SQLStore) LockSummaryHeadTx(ctx context.Context, tx pgx.Tx, owner string) (q.TraderSyncSummaryHead, error) {
	id, e := uuidValue(owner)
	if e != nil {
		return q.TraderSyncSummaryHead{}, e
	}
	queries := q.New(tx)
	if e = queries.EnsureSummaryHead(ctx, id); e != nil {
		return q.TraderSyncSummaryHead{}, e
	}
	return queries.LockSummaryHead(ctx, id)
}
func (s *SQLStore) SummaryHeads(ctx context.Context) ([]q.ListSummaryHeadsRow, error) {
	if e := s.queries.EnsureSummaryHeads(ctx); e != nil {
		return nil, e
	}
	owners, e := s.queries.SummaryHeadOwners(ctx)
	if e != nil {
		return nil, e
	}
	for _, owner := range owners {
		if e = txgate.WithAccountTx(ctx, s.pool, uuidString(owner), func(tx pgx.Tx) error { return q.New(tx).ResolveNeverStartedSummaryHeads(ctx, owner) }); e != nil {
			return nil, e
		}
	}
	return s.queries.ListSummaryHeads(ctx)
}
func (s *SQLStore) SetSummaryHeadTx(ctx context.Context, tx pgx.Tx, owner string, batch int64) error {
	id, e := uuidValue(owner)
	if e != nil {
		return e
	}
	n, e := q.New(tx).SetSummaryHeadBatch(ctx, q.SetSummaryHeadBatchParams{OwnerID: id, CurrentBatchID: pgtype.Int8{Int64: batch, Valid: true}})
	if e == nil && n != 1 {
		return ErrSummaryNotReady
	}
	return e
}
func (s *SQLStore) AttachSummaryPermitTx(ctx context.Context, tx pgx.Tx, batch int64, p delivery.Permit) error {
	head, e := s.LockSummaryHeadTx(ctx, tx, p.OwnerID)
	if e != nil {
		return e
	}
	if !head.CurrentBatchID.Valid || head.CurrentBatchID.Int64 != batch {
		return ErrStalePermit
	}
	a, e := q.New(tx).GetDeliveryAttemptForUpdate(ctx, uuidPG(p.AttemptID))
	if e != nil {
		return e
	}
	if e = validatePermit(p, a); e != nil {
		return e
	}
	n, e := q.New(tx).SetSummaryHeadPermit(ctx, q.SetSummaryHeadPermitParams{OwnerID: head.OwnerID, CurrentBatchID: head.CurrentBatchID, CurrentAttemptID: uuidPG(p.AttemptID)})
	if e == nil && n != 1 {
		return ErrStalePermit
	}
	return e
}

// ValidateSummaryPermitTx resumes only a still-live same invocation. It checks
// durable permit/head facts, not current grant/binding eligibility; it cannot
// authorize another attempt. Recovery never calls this to send after a crash.
func (s *SQLStore) ValidateSummaryPermitTx(ctx context.Context, tx pgx.Tx, batch int64, p delivery.Permit) error {
	head, e := s.LockSummaryHeadTx(ctx, tx, p.OwnerID)
	if e != nil {
		return e
	}
	if head.CurrentBatchID.Int64 != batch || !head.CurrentBatchID.Valid || head.CurrentAttemptID != uuidPG(p.AttemptID) {
		return ErrStalePermit
	}
	work, e := lockPermitWork(ctx, q.New(tx), p.Work)
	if e != nil {
		return e
	}
	a, e := q.New(tx).GetDeliveryAttemptForUpdate(ctx, uuidPG(p.AttemptID))
	if e != nil {
		return e
	}
	if work.status != "sending" || work.current != a.ID || a.ResultAt.Valid {
		return ErrStalePermit
	}
	return validatePermit(p, a)
}

// RecordSummaryStartedTx runs on the held summary connection. Callback callers
// must send an in-memory signal; only the coordinator calls this bounded I/O.
func (s *SQLStore) RecordSummaryStartedTx(ctx context.Context, tx pgx.Tx, batch int64, p delivery.Permit, at time.Time) error {
	if at.IsZero() {
		return fmt.Errorf("actual summary start required")
	}
	head, e := s.LockSummaryHeadTx(ctx, tx, p.OwnerID)
	if e != nil {
		return e
	}
	queries := q.New(tx)
	a, e := queries.GetDeliveryAttemptForUpdate(ctx, uuidPG(p.AttemptID))
	if e != nil {
		return e
	}
	if e = validatePermit(p, a); e != nil {
		return e
	}
	b, e := queries.ReadSummaryBatch(ctx, q.ReadSummaryBatchParams{OwnerID: head.OwnerID, ID: batch})
	if e != nil {
		return e
	}
	if head.CurrentBatchID.Int64 != batch && !b.FirstStartedAt.Valid {
		return ErrStalePermit
	}
	if _, e = queries.RecordDeliveryAttemptStarted(ctx, q.RecordDeliveryAttemptStartedParams{ID: a.ID, StartedAt: timestamptzValue(at)}); e != nil {
		return e
	}
	if _, e = queries.RecordSummaryStarted(ctx, q.RecordSummaryStartedParams{ID: batch, FirstStartedAt: timestamptzValue(at)}); e != nil {
		return e
	}
	if b.FirstStartedAt.Valid {
		at = b.FirstStartedAt.Time
	}
	_, e = queries.ResolveStartedSummaryHead(ctx, q.ResolveStartedSummaryHeadParams{OwnerID: head.OwnerID, CurrentBatchID: pgtype.Int8{Int64: batch, Valid: true}, PreviousBasisAt: timestamptzValue(at)})
	return e
}

func (s *SQLStore) RecordSummaryWait(ctx context.Context, batch int64, start, end time.Time, wait, gate time.Duration, reason string) error {
	if !start.IsZero() {
		if e := s.queries.RecordSummaryBudgetWait(ctx, q.RecordSummaryBudgetWaitParams{BatchID: batch, StartedAt: timestamptzValue(start), EndedAt: timestamptzValue(end), WaitMs: max(0, wait.Milliseconds()), Reason: reason}); e != nil {
			return e
		}
	}
	if gate > 0 {
		return s.queries.RecordSummaryGateWait(ctx, q.RecordSummaryGateWaitParams{ID: batch, LocalGateWaitMs: gate.Milliseconds()})
	}
	return nil
}

func (s *SQLStore) recoverSummaryHeads(ctx context.Context, incarnation pgtype.UUID, stoppedAt time.Time) error {
	heads, e := s.queries.RecoverableSummaryHeads(ctx, incarnation)
	if e != nil {
		return e
	}
	for _, old := range heads {
		e = txgate.WithAccountTx(ctx, s.pool, uuidString(old.OwnerID), func(tx pgx.Tx) error {
			queries := q.New(tx)
			h, e := queries.LockSummaryHead(ctx, old.OwnerID)
			if e != nil {
				return e
			}
			if h.CurrentBatchID != old.CurrentBatchID || h.CurrentAttemptID != old.CurrentAttemptID {
				return nil
			}
			actual, e := queries.SummaryBatchStartEvidence(ctx, h.CurrentBatchID.Int64)
			if e != nil {
				return e
			}
			basis, reason := stoppedAt, "sender_stopped_missing_start"
			if actual.Valid {
				basis, reason = actual.Time, "recovered_actual_start"
				if _, e = queries.RecordSummaryStarted(ctx, q.RecordSummaryStartedParams{ID: h.CurrentBatchID.Int64, FirstStartedAt: actual}); e != nil {
					return e
				}
			}
			if e = queries.RecoverSummaryBatchBasis(ctx, q.RecoverSummaryBatchBasisParams{ID: h.CurrentBatchID.Int64, RecoveryBasisAt: timestamptzValue(basis), RecoveryReason: reason}); e != nil {
				return e
			}
			_, e = queries.ResolveStartedSummaryHead(ctx, q.ResolveStartedSummaryHeadParams{OwnerID: h.OwnerID, CurrentBatchID: h.CurrentBatchID, PreviousBasisAt: timestamptzValue(basis)})
			return e
		})
		if e != nil {
			return e
		}
	}
	return nil
}

// ConfirmSummaryPermit is bounded by its caller after uncertain Commit. Row
// reads validate only the exact original attempt; it never grants permission.
func (s *SQLStore) ConfirmSummaryPermit(ctx context.Context, batch int64, p delivery.Permit) error {
	tx, e := s.pool.Begin(ctx)
	if e != nil {
		return e
	}
	defer tx.Rollback(context.Background())
	if e = s.ValidateSummaryPermitTx(ctx, tx, batch, p); e != nil {
		return e
	}
	return tx.Commit(ctx)
}
