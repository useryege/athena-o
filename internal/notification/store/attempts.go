package store

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/useryege/athena/internal/accountstate/txgate"
	"github.com/useryege/athena/internal/notification/delivery"
	q "github.com/useryege/athena/internal/notification/store/sqlc"
)

var ErrDeliveryNotEligible = errors.New("notification delivery is not eligible")
var ErrStalePermit = errors.New("notification permit is stale or invalid")

type permitWork struct {
	payload  []byte
	owner    string
	digest   []byte
	status   string
	attempts int
	eligible bool
	next     time.Time
	current  pgtype.UUID
	chat     int64
}

func lockPermitWork(ctx context.Context, queries *q.Queries, ref delivery.WorkRef) (permitWork, error) {
	switch ref.Kind {
	case "account":
		r, err := queries.GetAccountDeliveryForPermit(ctx, ref.ID)
		if err != nil {
			return permitWork{}, err
		}
		return permitWork{owner: uuidString(r.AccountID), digest: r.PayloadDigest, payload: r.Payload, status: r.Status, attempts: int(r.Attempts), eligible: r.BindingEligible.Valid && r.BindingEligible.Bool && !r.EligibilityRevokedAt.Valid, next: r.NextAttemptAt.Time, current: r.CurrentAttemptID, chat: r.TelegramChatID}, nil
	case "reply":
		r, err := queries.GetReplyDeliveryForPermit(ctx, ref.ID)
		if err != nil {
			return permitWork{}, err
		}
		return permitWork{owner: uuidString(r.AccountID), digest: r.PayloadDigest, payload: r.Payload, status: r.Status, attempts: int(r.Attempts), eligible: r.BindingEligible.Valid && r.BindingEligible.Bool && !r.EligibilityRevokedAt.Valid, next: r.NextAttemptAt.Time, current: r.CurrentAttemptID, chat: r.TelegramChatID}, nil
	case "system":
		r, err := queries.GetSystemDeliveryForPermit(ctx, ref.ID)
		if err != nil {
			return permitWork{}, err
		}
		return permitWork{digest: r.PayloadDigest, payload: r.Payload, status: r.Status, attempts: int(r.Attempts), eligible: true, next: r.NextAttemptAt.Time, current: r.CurrentAttemptID}, nil

	default:
		return permitWork{}, ErrDeliveryNotEligible
	}
}

func (s *SQLStore) withPermitTx(ctx context.Context, owner string, fn func(*q.Queries) error) error {
	if err := s.transactional(); err != nil {
		return err
	}
	if owner != "" {
		return txgate.WithAccountTx(ctx, s.pool, owner, func(tx pgx.Tx) error { return fn(q.New(tx)) })
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	if err = fn(q.New(tx)); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// Authorize consumes a work item before any HTTP call. A failed/uncertain commit never permits a send.
func (s *SQLStore) Authorize(ctx context.Context, candidate delivery.Candidate, incarnation uuid.UUID, guard func() error) (delivery.Permit, error) {
	ref := candidate.Ref
	if candidate.ChatID == 0 {
		return delivery.Permit{}, ErrDeliveryNotEligible
	}
	if err := s.transactional(); err != nil {
		return delivery.Permit{}, err
	}
	if ref.ID <= 0 || incarnation == uuid.Nil {
		return delivery.Permit{}, ErrDeliveryNotEligible
	}
	owner := ""
	if ref.Kind == "account" {
		id, err := s.queries.GetAccountDeliveryOwner(ctx, ref.ID)
		if err != nil {
			return delivery.Permit{}, err
		}
		owner = uuidString(id)
	}
	if ref.Kind == "reply" {
		id, err := s.queries.GetReplyDeliveryOwner(ctx, ref.ID)
		if err != nil {
			return delivery.Permit{}, err
		}
		owner = uuidString(id)
	}
	var permit delivery.Permit
	err := s.withPermitTx(ctx, owner, func(queries *q.Queries) error {
		work, err := lockPermitWork(ctx, queries, ref)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrDeliveryNotEligible
		}
		if err != nil {
			return err
		}
		if _, err := delivery.DecodePayload(work.payload); err != nil {
			return err
		}
		if !bytes.Equal(delivery.PayloadDigest(work.payload), work.digest) {
			return ErrStalePermit
		}
		if work.owner != owner || work.status != "pending" || !work.eligible || work.attempts >= 5 {
			return ErrDeliveryNotEligible
		}
		if work.chat != 0 && (work.chat != candidate.ChatID || candidate.Group) {
			return ErrDeliveryNotEligible
		}
		if ref.Kind == "system" && (!candidate.Group || candidate.ChatID >= 0) {
			return ErrDeliveryNotEligible
		}
		if guard != nil {
			if err := guard(); err != nil {
				return err
			}
		}
		// Evaluate the deadline against database time, not a worker's clock.
		var now time.Time
		// The attempt INSERT supplies the database timestamp used by all permit transitions.
		r, err := queries.CreateDeliveryAttempt(ctx, q.CreateDeliveryAttemptParams{ID: uuidPG(uuid.New()), WorkKind: ref.Kind, WorkID: ref.ID, OwnerID: optionalUUID(owner), SenderIncarnation: uuidPG(incarnation), PayloadDigest: work.digest, TelegramChatID: candidate.ChatID, TelegramGroup: candidate.Group})
		if err != nil {
			return err
		}
		now = r.AuthorizedAt.Time
		if work.next.After(now) {
			return ErrDeliveryNotEligible
		}
		var rows int64
		if ref.Kind == "account" {
			rows, err = queries.AuthorizeAccountDelivery(ctx, q.AuthorizeAccountDeliveryParams{ID: ref.ID, CurrentAttemptID: r.ID, LastAttemptAt: r.AuthorizedAt})
		} else if ref.Kind == "reply" {
			rows, err = queries.AuthorizeReplyDelivery(ctx, q.AuthorizeReplyDeliveryParams{ID: ref.ID, CurrentAttemptID: r.ID, LastAttemptAt: r.AuthorizedAt})
		} else {
			rows, err = queries.AuthorizeSystemDelivery(ctx, q.AuthorizeSystemDeliveryParams{ID: ref.ID, CurrentAttemptID: r.ID, LastAttemptAt: r.AuthorizedAt})
		}
		if err != nil {
			return err
		}
		if rows != 1 {
			return ErrDeliveryNotEligible
		}
		permit = delivery.Permit{ChatID: candidate.ChatID, Group: candidate.Group, Work: ref, AttemptID: uuid.UUID(r.ID.Bytes), OwnerID: owner, SenderIncarnation: incarnation, PayloadDigest: append([]byte(nil), work.digest...), Payload: append([]byte(nil), work.payload...), AuthorizedAt: now}
		return nil
	})
	if err != nil {
		return delivery.Permit{}, err
	}
	return permit, nil
}

func validatePermit(p delivery.Permit, r q.NotificationDeliveryAttempt) error {
	if p.ChatID != r.TelegramChatID || p.Group != r.TelegramGroup || p.AttemptID == uuid.Nil || r.ID != uuidPG(p.AttemptID) || r.WorkKind != p.Work.Kind || r.WorkID != p.Work.ID || uuidString(r.OwnerID) != p.OwnerID || r.SenderIncarnation != uuidPG(p.SenderIncarnation) || !bytes.Equal(r.PayloadDigest, p.PayloadDigest) || !r.AuthorizedAt.Time.Equal(p.AuthorizedAt) {
		return ErrStalePermit
	}
	return nil
}

// RecordStarted preserves missing evidence and accepts delayed evidence after a receipt.
func (s *SQLStore) RecordStarted(ctx context.Context, p delivery.Permit, at time.Time) error {
	if at.IsZero() {
		return fmt.Errorf("notification start timestamp is required")
	}
	return s.withPermitTx(ctx, p.OwnerID, func(queries *q.Queries) error {
		r, err := queries.GetDeliveryAttemptForUpdate(ctx, uuidPG(p.AttemptID))
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrStalePermit
		}
		if err != nil {
			return err
		}
		if err = validatePermit(p, r); err != nil {
			return err
		}
		_, err = queries.RecordDeliveryAttemptStarted(ctx, q.RecordDeliveryAttemptStartedParams{ID: r.ID, StartedAt: timestamptzValue(at)})
		return err
	})
}

func sameOutcome(r q.NotificationDeliveryAttempt, o delivery.Outcome) bool {
	return r.ResultAt.Valid && r.Outcome.String == o.Kind && r.MessageID.String == o.MessageID && r.OutcomeCode.String == o.Code && r.RetryAfter.Microseconds == o.RetryAfter.Microseconds()
}

// RecordOutcome is the only delivery result transition. Retries repeat this CAS, never the send.
func (s *SQLStore) RecordOutcome(ctx context.Context, p delivery.Permit, o delivery.Outcome, at time.Time) error {
	if err := s.transactional(); err != nil {
		return err
	}
	if at.IsZero() || o.RetryAfter < 0 {
		return fmt.Errorf("invalid notification outcome timestamp or retry delay")
	}
	switch o.Kind {
	case "sent", "retryable", "failed", "unknown":
	default:
		return fmt.Errorf("invalid notification outcome kind %q", o.Kind)
	}
	if o.Kind == "sent" && o.MessageID == "" {
		return fmt.Errorf("sent notification requires a provider message ID")
	}
	hadOutcome := false
	err := s.withPermitTx(ctx, p.OwnerID, func(queries *q.Queries) error {
		work, err := lockPermitWork(ctx, queries, p.Work)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrStalePermit
		}
		if err != nil {
			return err
		}
		r, err := queries.GetDeliveryAttemptForUpdate(ctx, uuidPG(p.AttemptID))
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrStalePermit
		}
		if err != nil {
			return err
		}
		if err = validatePermit(p, r); err != nil {
			return err
		}
		hadOutcome = r.ResultAt.Valid
		if r.ResultAt.Valid {
			if !sameOutcome(r, o) {
				return ErrStalePermit
			}
			if work.status != "sending" {
				return nil
			}
		}
		if work.owner != p.OwnerID || work.current != r.ID || work.status != "sending" || !bytes.Equal(work.digest, p.PayloadDigest) {
			return ErrStalePermit
		}
		state, wait := delivery.NextState(o, work.attempts, work.eligible)
		var rows int64
		if p.Work.Kind == "account" {
			rows, err = queries.RecordAccountDeliveryOutcome(ctx, q.RecordAccountDeliveryOutcomeParams{ID: p.Work.ID, CurrentAttemptID: r.ID, Status: state, ProviderMessageID: nullableText(o.MessageID), ErrorMessage: nullableText(o.Code), ResultAt: timestamptzValue(at), NextAttemptAt: timestamptzValue(at.Add(wait))})
		} else if p.Work.Kind == "reply" {
			rows, err = queries.RecordReplyDeliveryOutcome(ctx, q.RecordReplyDeliveryOutcomeParams{ID: p.Work.ID, CurrentAttemptID: r.ID, Status: state, ProviderMessageID: nullableText(o.MessageID), ErrorMessage: nullableText(o.Code), ResultAt: timestamptzValue(at), NextAttemptAt: timestamptzValue(at.Add(wait))})
		} else {
			rows, err = queries.RecordSystemDeliveryOutcome(ctx, q.RecordSystemDeliveryOutcomeParams{ID: p.Work.ID, CurrentAttemptID: r.ID, Status: state, ProviderMessageID: nullableText(o.MessageID), ErrorMessage: nullableText(o.Code), ResultAt: timestamptzValue(at), NextAttemptAt: timestamptzValue(at.Add(wait))})
		}
		if err != nil {
			return err
		}
		if rows != 1 {
			return ErrStalePermit
		}
		if r.ResultAt.Valid {
			return nil
		}
		rows, err = queries.RecordDeliveryAttemptOutcome(ctx, q.RecordDeliveryAttemptOutcomeParams{ID: r.ID, Outcome: textValue(o.Kind), ResultAt: timestamptzValue(at), MessageID: nullableText(o.MessageID), OutcomeCode: nullableText(o.Code), RetryAfter: intervalValue(o.RetryAfter)})
		if err != nil {
			return err
		}
		if rows != 1 {
			return ErrStalePermit
		}
		return nil
	})
	if err == nil || errors.Is(err, ErrStalePermit) {
		return err
	}
	// A lost COMMIT acknowledgement can already have committed the receipt. Read by UUID;
	// if it cannot be confirmed, leave the consumed permit untouched and report the error.
	r, readErr := s.queries.GetDeliveryAttempt(ctx, uuidPG(p.AttemptID))
	if !hadOutcome && readErr == nil && validatePermit(p, r) == nil && sameOutcome(r, o) {
		return nil
	}
	return err
}
func uuidPG(id uuid.UUID) pgtype.UUID { return pgtype.UUID{Bytes: [16]byte(id), Valid: true} }
func optionalUUID(owner string) pgtype.UUID {
	if owner == "" {
		return pgtype.UUID{}
	}
	id, _ := uuidValue(owner)
	return id
}
