package store

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/useryege/athena/internal/notification/delivery"
	q "github.com/useryege/athena/internal/notification/store/sqlc"
	"time"
)

// RecoverSender requires durable external or graceful stop confirmation for this exact incarnation.
func (s *SQLStore) RecoverSender(ctx context.Context, stoppedIncarnation uuid.UUID) error {
	instance, err := s.queries.GetSenderInstance(ctx, uuidPG(stoppedIncarnation))
	if err != nil {
		return err
	}
	if !instance.StoppedAt.Valid {
		return ErrSenderActive
	}
	attempts, err := s.queries.ListSenderRecoveryAttempts(ctx, uuidPG(stoppedIncarnation))
	if err != nil {
		return err
	}
	for _, a := range attempts {
		p := permitFromAttempt(a)
		outcome := delivery.Outcome{Kind: "unknown", Code: "sender_stopped"}
		at := instance.StoppedAt.Time
		if a.ResultAt.Valid {
			outcome = delivery.Outcome{Kind: a.Outcome.String, Code: a.OutcomeCode.String, MessageID: a.MessageID.String, RetryAfter: time.Duration(a.RetryAfter.Microseconds) * time.Microsecond}
			at = a.ResultAt.Time
		}
		if err = s.RecordOutcome(ctx, p, outcome, at); err != nil {
			return fmt.Errorf("recover attempt %s: %w", p.AttemptID, err)
		}
	}
	return s.recoverSummaryHeads(ctx, uuidPG(stoppedIncarnation), instance.StoppedAt.Time)
}
func permitFromAttempt(a q.NotificationDeliveryAttempt) delivery.Permit {
	return delivery.Permit{Work: delivery.WorkRef{Kind: a.WorkKind, ID: a.WorkID}, AttemptID: uuid.UUID(a.ID.Bytes), OwnerID: uuidString(a.OwnerID), SenderIncarnation: uuid.UUID(a.SenderIncarnation.Bytes), PayloadDigest: a.PayloadDigest, AuthorizedAt: a.AuthorizedAt.Time, ChatID: a.TelegramChatID, Group: a.TelegramGroup}
}
func (s *SQLStore) BudgetEvidence(ctx context.Context, since time.Time) ([]q.NotificationDeliveryAttempt, error) {
	return s.queries.ListBudgetEvidence(ctx, timestamptzValue(since))
}

type RetryBudgetEvidence struct {
	AttemptID uuid.UUID
	Wait      time.Duration
}

func (s *SQLStore) UnreleasedRetryAfters(ctx context.Context) ([]RetryBudgetEvidence, error) {
	rows, err := s.queries.ListUnreleasedRetryAfters(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]RetryBudgetEvidence, 0, len(rows))
	for _, r := range rows {
		items = append(items, RetryBudgetEvidence{AttemptID: uuid.UUID(r.ID.Bytes), Wait: time.Duration(r.RetryAfter.Microseconds) * time.Microsecond})
	}
	return items, nil
}
func (s *SQLStore) ReleaseRetryAfter(ctx context.Context, id uuid.UUID) error {
	_, err := s.queries.ReleaseRetryAfter(ctx, uuidPG(id))
	return err
}
