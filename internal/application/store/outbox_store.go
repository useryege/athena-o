package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	appsqlc "github.com/useryege/athena/internal/application/store/sqlc"
)

const (
	OutboxTypeProjectCollectionRequested = "project.collection_requested"
	OutboxTypeKafkaProjectEventPublish   = "kafka.project_event_publish_requested"
)

const (
	OutboxStatusPending    = "pending"
	OutboxStatusProcessing = "processing"
	OutboxStatusProcessed  = "processed"
	OutboxStatusDiscarded  = "discarded"
)

type OutboxEvent struct {
	ID            int64
	Type          string
	AggregateType string
	AggregateID   string
	ChainID       int64
	DedupKey      string
	Payload       json.RawMessage
	Status        string
	Attempts      int32
	NextAttemptAt time.Time
	LockedAt      time.Time
	LockedBy      string
	LastError     string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type CreateOutboxEventParams struct {
	Type          string
	AggregateType string
	AggregateID   string
	ChainID       int64
	DedupKey      string
	Payload       any
	NextAttemptAt time.Time
}

func (s *SQLStore) InsertOutboxEvent(ctx context.Context, item CreateOutboxEventParams) (*OutboxEvent, error) {
	queries, err := s.querier()
	if err != nil {
		return nil, err
	}
	return insertOutboxEvent(ctx, queries, item)
}

func insertOutboxEvent(ctx context.Context, queries appsqlc.Querier, item CreateOutboxEventParams) (*OutboxEvent, error) {
	payload, err := json.Marshal(item.Payload)
	if err != nil {
		return nil, fmt.Errorf("marshal outbox payload: %w", err)
	}
	if queries == nil {
		return nil, fmt.Errorf("application postgres database is not configured")
	}
	row, err := queries.InsertOutboxEvent(ctx, appsqlc.InsertOutboxEventParams{
		Type:          item.Type,
		AggregateType: item.AggregateType,
		AggregateID:   item.AggregateID,
		ChainID:       item.ChainID,
		DedupKey:      item.DedupKey,
		Payload:       payload,
		NextAttemptAt: pgTime(item.NextAttemptAt),
	})
	if err != nil {
		return nil, fmt.Errorf("insert outbox event: %w", err)
	}
	event := outboxEventFromSQLC(row.ID, row.Type, row.AggregateType, row.AggregateID, row.ChainID, row.DedupKey, row.Payload, row.Status, row.Attempts, row.NextAttemptAt, row.LockedAt, row.LockedBy, row.LastError, row.CreatedAt, row.UpdatedAt)
	return &event, nil
}

func (s *SQLStore) ClaimOutboxEvents(ctx context.Context, lockedBy string, now time.Time, limit int32) ([]OutboxEvent, error) {
	if limit <= 0 {
		return nil, nil
	}
	queries, err := s.querier()
	if err != nil {
		return nil, err
	}
	rows, err := queries.ClaimOutboxEvents(ctx, appsqlc.ClaimOutboxEventsParams{
		Now:        pgTime(now),
		LockedBy:   pgText(lockedBy),
		LimitCount: limit,
	})
	if err != nil {
		return nil, fmt.Errorf("claim outbox events: %w", err)
	}
	items := make([]OutboxEvent, 0, len(rows))
	for _, row := range rows {
		items = append(items, outboxEventFromSQLC(row.ID, row.Type, row.AggregateType, row.AggregateID, row.ChainID, row.DedupKey, row.Payload, row.Status, row.Attempts, row.NextAttemptAt, row.LockedAt, row.LockedBy, row.LastError, row.CreatedAt, row.UpdatedAt))
	}
	return items, nil
}

func (s *SQLStore) ClaimOutboxEventsByTypes(ctx context.Context, lockedBy string, now time.Time, limit int32, types []string) ([]OutboxEvent, error) {
	if limit <= 0 || len(types) == 0 {
		return nil, nil
	}
	queries, err := s.querier()
	if err != nil {
		return nil, err
	}
	rows, err := queries.ClaimOutboxEventsByTypes(ctx, appsqlc.ClaimOutboxEventsByTypesParams{
		Now:        pgTime(now),
		LockedBy:   pgText(lockedBy),
		Types:      types,
		LimitCount: limit,
	})
	if err != nil {
		return nil, fmt.Errorf("claim outbox events by types: %w", err)
	}
	items := make([]OutboxEvent, 0, len(rows))
	for _, row := range rows {
		items = append(items, outboxEventFromSQLC(row.ID, row.Type, row.AggregateType, row.AggregateID, row.ChainID, row.DedupKey, row.Payload, row.Status, row.Attempts, row.NextAttemptAt, row.LockedAt, row.LockedBy, row.LastError, row.CreatedAt, row.UpdatedAt))
	}
	return items, nil
}

func (s *SQLStore) MarkOutboxEventProcessed(ctx context.Context, id int64) error {
	queries, err := s.querier()
	if err != nil {
		return err
	}
	return queries.MarkOutboxEventProcessed(ctx, id)
}

func (s *SQLStore) MarkOutboxEventDiscarded(ctx context.Context, id int64, lastError string) error {
	queries, err := s.querier()
	if err != nil {
		return err
	}
	return queries.MarkOutboxEventDiscarded(ctx, appsqlc.MarkOutboxEventDiscardedParams{
		ID:        id,
		LastError: pgText(lastError),
	})
}

func (s *SQLStore) MarkOutboxEventFailed(ctx context.Context, id int64, nextAttemptAt time.Time, lastError string) error {
	queries, err := s.querier()
	if err != nil {
		return err
	}
	return queries.MarkOutboxEventFailed(ctx, appsqlc.MarkOutboxEventFailedParams{
		ID:            id,
		NextAttemptAt: pgTime(nextAttemptAt),
		LastError:     pgText(lastError),
	})
}

func outboxEventFromSQLC(id int64, eventType string, aggregateType string, aggregateID string, chainID int64, dedupKey string, payload []byte, status string, attempts int32, nextAttemptAt pgtype.Timestamptz, lockedAt pgtype.Timestamptz, lockedBy pgtype.Text, lastError pgtype.Text, createdAt pgtype.Timestamptz, updatedAt pgtype.Timestamptz) OutboxEvent {
	item := OutboxEvent{
		ID:            id,
		Type:          eventType,
		AggregateType: aggregateType,
		AggregateID:   aggregateID,
		ChainID:       chainID,
		DedupKey:      dedupKey,
		Payload:       append(json.RawMessage(nil), payload...),
		Status:        status,
		Attempts:      attempts,
		LockedBy:      lockedBy.String,
		LastError:     lastError.String,
	}
	if nextAttemptAt.Valid {
		item.NextAttemptAt = nextAttemptAt.Time
	}
	if lockedAt.Valid {
		item.LockedAt = lockedAt.Time
	}
	if createdAt.Valid {
		item.CreatedAt = createdAt.Time
	}
	if updatedAt.Valid {
		item.UpdatedAt = updatedAt.Time
	}
	return item
}
