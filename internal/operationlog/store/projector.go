package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/useryege/athena/internal/operationlog/event"
	"github.com/useryege/athena/internal/operationlog/store/sqlc"
	"sort"
	"strings"
	"time"
)

type rawEvent struct {
	e        event.Event
	received time.Time
	ingestID int64
	id       pgtype.UUID
	pending  bool
}

func (s *Store) Project(ctx context.Context) (out Projection, err error) {
	ctx, cancel := context.WithTimeout(ctx, ProjectTimeout)
	defer cancel()
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return out, err
	}
	defer tx.Rollback(ctx)
	q := sqlc.New(tx)
	row, err := q.LockPublication(ctx)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Projection{Busy: true}, nil
		}
		return out, err
	}
	out.PublishedSeq = row
	batch, err := q.PendingBatch(ctx)
	if err != nil {
		return out, err
	}
	if len(batch) == 0 {
		return out, tx.Commit(ctx)
	}
	grouped := map[string][]rawEvent{}
	for _, r := range batch {
		e, decodeErr := event.Decode(r.Payload)
		if decodeErr != nil {
			if qerr := q.Quarantine(ctx, sqlc.QuarantineParams{EventID: r.EventID, ReasonCode: pgtype.Text{String: "invalid_event", Valid: true}}); qerr != nil {
				return out, qerr
			}
			out.Quarantined++
			continue
		}
		grouped[e.OperationID] = append(grouped[e.OperationID], rawEvent{e: e, received: r.ReceivedAt.Time, ingestID: r.IngestID.Int64, id: r.EventID, pending: true})
	}
	changed := false
	for op, rs := range grouped {
		if _, err = tx.Exec(ctx, `SAVEPOINT operation_log_projection`); err != nil {
			return out, err
		}
		before := out
		// A later phase is allowed to arrive after its counterpart was already
		// published. Fold against all previously processed source facts rather
		// than treating each inbox batch as a standalone operation.
		processed, qerr := q.ProcessedEvents(ctx, id(op))
		if qerr != nil {
			if _, rbErr := tx.Exec(ctx, `ROLLBACK TO SAVEPOINT operation_log_projection`); rbErr != nil {
				return out, qerr
			}
			return out, qerr
		}
		for _, r := range processed {
			e, derr := event.Decode(r.Payload)
			if derr != nil {
				// Processed rows originated from a validated envelope. Keep the
				// transaction conservative if a database operator tampered with it.
				return out, fmt.Errorf("processed event %s: %w", r.EventID, derr)
			}
			rs = append(rs, rawEvent{e: e, received: r.ReceivedAt.Time, ingestID: r.IngestID.Int64, id: r.EventID})
		}
		if c, perr := projectOperation(ctx, q, op, rs, &out); perr != nil {
			if _, rbErr := tx.Exec(ctx, `ROLLBACK TO SAVEPOINT operation_log_projection`); rbErr != nil {
				return out, perr
			}
			if !isSingleRowProjectionError(perr) {
				return out, perr
			}
			out = before
			for _, r := range rs {
				if r.pending {
					if qerr := q.Quarantine(ctx, sqlc.QuarantineParams{EventID: r.id, ReasonCode: pgtype.Text{String: "projection_error", Valid: true}}); qerr != nil {
						return out, qerr
					}
					out.Quarantined++
				}
			}
			if _, relErr := tx.Exec(ctx, `RELEASE SAVEPOINT operation_log_projection`); relErr != nil {
				return out, relErr
			}
			continue
		} else if c {
			changed = true
		}
		if _, err = tx.Exec(ctx, `RELEASE SAVEPOINT operation_log_projection`); err != nil {
			return out, err
		}
	}
	if !changed {
		if err = tx.Commit(ctx); err != nil {
			return out, err
		}
		return out, nil
	}
	if err = q.Publish(ctx, out.PublishedSeq+1); err != nil {
		return out, err
	}
	out.PublishedSeq++
	if err = tx.Commit(ctx); err != nil {
		return out, err
	}
	return out, nil
}
func projectOperation(ctx context.Context, q *sqlc.Queries, op string, rs []rawEvent, out *Projection) (bool, error) {
	// Keep arrival order (received_at, ingest id already supplied by SQL). A bad
	// stage is quarantined by savepoint and never participates in later folds.
	sort.SliceStable(rs, func(i, j int) bool {
		if rs[i].received.Equal(rs[j].received) {
			return rs[i].ingestID < rs[j].ingestID
		}
		return rs[i].received.Before(rs[j].received)
	})
	var start, finish *rawEvent
	var rejected *rawEvent
	changed := false
	for i := range rs {
		r := rs[i]
		if r.e.OperationID != op {
			continue
		}
		switch r.e.Phase {
		case event.Start:
			if start != nil {
				if err := q.Quarantine(ctx, sqlc.QuarantineParams{EventID: r.id, ReasonCode: pgtype.Text{String: "event_conflict", Valid: true}}); err != nil {
					return false, err
				}
				if r.pending {
					out.Quarantined++
				}
				continue
			}
			start = &r
		case event.Finish:
			if finish != nil {
				if err := q.Quarantine(ctx, sqlc.QuarantineParams{EventID: r.id, ReasonCode: pgtype.Text{String: "event_conflict", Valid: true}}); err != nil {
					return false, err
				}
				if r.pending {
					out.Quarantined++
				}
				continue
			}
			finish = &r
		}
	}
	if start == nil && finish == nil {
		return false, nil
	}
	base := start
	if finish != nil {
		base = finish
	}
	view := NormalizedView{Event: base.e, FirstReceivedAt: base.received, LastReceivedAt: base.received}
	for _, r := range rs {
		if r.e.OperationID == op && !containsString(view.PhasesReceived, string(r.e.Phase)) {
			view.PhasesReceived = append(view.PhasesReceived, string(r.e.Phase))
		}
	}
	if start != nil && finish != nil {
		if start.e.ProducerID != finish.e.ProducerID || start.e.ActionCode != finish.e.ActionCode || start.e.ModuleCode != finish.e.ModuleCode || start.e.StartedAt.UTC() != finish.e.StartedAt.UTC() || !actorsCompatible(start.e.Actor, finish.e.Actor) || !sameOptional(start.e.RequestID, finish.e.RequestID) || !sameOptionalPtr(start.e.ParentOperationID, finish.e.ParentOperationID) || !sameOptionalPtr(start.e.TargetAccountID, finish.e.TargetAccountID) {
			keep, reject := start, finish
			if start.pending && !finish.pending {
				keep, reject = finish, start
			} else if start.pending && finish.pending && later(start, finish) {
				keep, reject = finish, start
			}
			if err := q.Quarantine(ctx, sqlc.QuarantineParams{EventID: reject.id, ReasonCode: pgtype.Text{String: "event_conflict", Valid: true}}); err != nil {
				return false, err
			}
			if reject.pending {
				out.Quarantined++
			}
			rejected = reject
			base = keep
			view.Event = keep.e
			if keep.e.Phase == event.Start {
				view.Observation = event.StartOnly
			} else {
				view.Observation = event.FinishOnly
			}
		} else {
			view.Event = finish.e
			view.Actor = mergeActors(start.e.Actor, finish.e.Actor)
			view.Observation = event.Complete
		}
	}
	acceptedPending := (start != nil && start.pending && start != rejected) || (finish != nil && finish.pending && finish != rejected)
	if !acceptedPending {
		return false, nil
	}
	if rejected != nil {
		view.FirstReceivedAt = base.received
		view.LastReceivedAt = base.received
	} else if start != nil && finish != nil {
		view.FirstReceivedAt = start.received
		view.LastReceivedAt = finish.received
		if finish.received.Before(start.received) {
			view.FirstReceivedAt = finish.received
			view.LastReceivedAt = start.received
		}
	}
	// Explicitly encode the normalized view so filter columns and detail agree.
	detail, err := json.Marshal(view)
	if err != nil {
		return false, err
	}
	seq := out.PublishedSeq + 1
	if err = q.CloseVersion(ctx, sqlc.CloseVersionParams{OperationID: id(op), VisibleToSeq: pgtype.Int8{Int64: seq, Valid: true}}); err != nil {
		return false, err
	}
	actor := view.Actor
	var primaryType, primaryID *string
	for _, r := range view.Resources {
		if r.Primary {
			primaryType = &r.Type
			primaryID = &r.ID
			break
		}
	}
	if err = q.InsertVersion(ctx, sqlc.InsertVersionParams{OperationID: id(op), VisibleFromSeq: seq, StartedAt: timestamp(view.StartedAt), FinishedAt: timestampOrNull(view.OccurredAt, view.Phase == event.Finish), ActorAccountID: optionalID(actor.AccountID), ActorUsername: textValue(actor.UsernameSnapshot), ActorRole: actor.Role, Realm: actor.Realm, CredentialKind: actor.CredentialKind, ModuleCode: view.ModuleCode, ActionCode: view.ActionCode, Outcome: string(view.Outcome), Observation: string(view.Observation), TargetAccountID: optionalID(view.TargetAccountID), PrimaryResourceType: textValue(primaryType), PrimaryResourceID: textValue(primaryID), RequestID: id(view.RequestID), ParentOperationID: optionalID(view.ParentOperationID), BusinessRequestID: textValue(view.BusinessRequestID), DurationMs: intValue(view.DurationMs), Detail: detail, SourceEventIds: sourceEventIDs(start, finish, rejected)}); err != nil {
		return false, err
	}
	if start != nil && start != rejected {
		if start.pending {
			if err = q.MarkProcessed(ctx, start.id); err != nil {
				return false, err
			}
			out.Processed++
			changed = true
		}
	}
	if finish != nil && finish != rejected {
		if finish.pending {
			if err = q.MarkProcessed(ctx, finish.id); err != nil {
				return false, err
			}
			out.Processed++
			changed = true
		}
	}
	return changed, nil
}

func isSingleRowProjectionError(err error) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return false
	}
	state := pgErr.SQLState()
	return strings.HasPrefix(state, "22") || strings.HasPrefix(state, "23") || state == "P0001"
}

func later(a, b *rawEvent) bool {
	if a.received.Equal(b.received) {
		return a.ingestID > b.ingestID
	}
	return a.received.After(b.received)
}

func sameOptional(a, b string) bool { return a == b }

func sameOptionalPtr(a, b *string) bool {
	return a == nil || b == nil || *a == *b
}
func timestampOrNull(t time.Time, ok bool) pgtype.Timestamptz {
	if !ok {
		return pgtype.Timestamptz{}
	}
	return timestamp(t)
}
func sourceEventIDs(a, b, rejected *rawEvent) []pgtype.UUID {
	ids := make([]pgtype.UUID, 0, 2)
	if a != nil && a != rejected {
		ids = append(ids, a.id)
	}
	if b != nil && b != rejected {
		ids = append(ids, b.id)
	}
	return ids
}
func actorsCompatible(a, b event.Actor) bool {
	if !a.IdentityVerified || !b.IdentityVerified {
		return true
	}
	// A verified account may have a partial snapshot when its username
	// projection is unavailable. Let a later verified event complete that
	// snapshot; any fields present in both events are still compared below.
	if a.AccountID != nil && b.AccountID != nil && *a.AccountID != *b.AccountID {
		return false
	}
	if a.UsernameSnapshot != nil && b.UsernameSnapshot != nil && *a.UsernameSnapshot != *b.UsernameSnapshot {
		return false
	}
	if a.Provider != nil && b.Provider != nil && *a.Provider != *b.Provider {
		return false
	}
	if a.Role != "UNKNOWN" && b.Role != "UNKNOWN" && a.Role != b.Role {
		return false
	}
	if a.Realm != "UNKNOWN" && b.Realm != "UNKNOWN" && a.Realm != b.Realm {
		return false
	}
	if a.CredentialKind != "UNAUTHENTICATED" && b.CredentialKind != "UNAUTHENTICATED" && a.CredentialKind != b.CredentialKind {
		return false
	}
	return true
}

func mergeActors(start, finish event.Actor) event.Actor {
	if actorTrust(finish) > actorTrust(start) {
		return finish
	}
	if actorTrust(start) > actorTrust(finish) {
		return start
	}
	merged := start
	if merged.AccountID == nil {
		merged.AccountID = finish.AccountID
	}
	if merged.UsernameSnapshot == nil {
		merged.UsernameSnapshot = finish.UsernameSnapshot
	}
	if merged.Provider == nil {
		merged.Provider = finish.Provider
	}
	if merged.Role == "UNKNOWN" {
		merged.Role = finish.Role
	}
	if merged.Realm == "UNKNOWN" {
		merged.Realm = finish.Realm
	}
	if merged.CredentialKind == "UNAUTHENTICATED" {
		merged.CredentialKind = finish.CredentialKind
	}
	merged.IdentityVerified = merged.IdentityVerified || finish.IdentityVerified
	merged.IdentitySnapshotComplete = merged.IdentitySnapshotComplete || finish.IdentitySnapshotComplete
	return merged
}

func actorTrust(a event.Actor) int {
	if a.IdentitySnapshotComplete && a.IdentityVerified && a.AccountID != nil && a.UsernameSnapshot != nil {
		return 3
	}
	if a.IdentityVerified && a.AccountID != nil {
		return 2
	}
	if a.IdentityVerified {
		return 1
	}
	return 0
}

// Projector runs the serialized projection loop. It keeps polling at the
// contract interval and backs off temporary database/schema failures without
// terminating the service, allowing recovery after dependencies return.
type Projector struct {
	Store  *Store
	Verify func(context.Context) error
	Now    func() time.Time
}

const (
	initialProjectorBackoff = time.Second
	maxProjectorBackoff     = 30 * time.Second
)

func nextProjectorBackoff(current time.Duration) time.Duration {
	if current <= 0 {
		return initialProjectorBackoff
	}
	if current >= maxProjectorBackoff {
		return maxProjectorBackoff
	}
	next := current * 2
	if next > maxProjectorBackoff {
		return maxProjectorBackoff
	}
	return next
}

func (p *Projector) Run(ctx context.Context) error {
	if p == nil || p.Store == nil {
		return errors.New("operation-log projector store is required")
	}
	interval := ProjectInterval
	backoff := initialProjectorBackoff
	timer := time.NewTimer(0)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-timer.C:
		}
		_, err := p.Store.Project(ctx)
		if err != nil {
			p.Store.setProcessingError(err)
			if p.Verify != nil {
				// Reverification is bounded independently from the projector
				// transaction. A failed database operation must not make shutdown
				// wait for schema verification's longer migration budget.
				verifyCtx, cancel := context.WithTimeout(ctx, ProjectTimeout)
				verifyErr := p.Verify(verifyCtx)
				cancel()
				p.Store.SetQueryReady(verifyErr == nil)
			}
			timer.Reset(backoff)
			backoff = nextProjectorBackoff(backoff)
			continue
		}
		if err := p.refreshQueryReadiness(ctx); err != nil {
			p.Store.setProcessingError(err)
			timer.Reset(backoff)
			backoff = nextProjectorBackoff(backoff)
			continue
		}
		p.Store.setProcessingError(nil)
		backoff = interval
		timer.Reset(interval)
	}
}

// refreshQueryReadiness closes the recovery gap after a transient schema or
// account-store failure. A successful projection alone proves only that the
// operation-log schema is writable; the independent verification must succeed
// before health can return to SERVING.
func (p *Projector) refreshQueryReadiness(ctx context.Context) error {
	if p.Store.QueryReady() || p.Verify == nil {
		return nil
	}
	verifyCtx, cancel := context.WithTimeout(ctx, ProjectTimeout)
	err := p.Verify(verifyCtx)
	cancel()
	if err == nil {
		p.Store.SetQueryReady(true)
	}
	return err
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
