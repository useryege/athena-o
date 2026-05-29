package store

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/jackc/pgx/v5/pgtype"
	appsqlc "github.com/useryege/athena/internal/application/store/sqlc"
)

func (s *SQLStore) AddProjectEventLog(ctx context.Context, item ProjectEventLog) error {
	if item.EventType <= 0 {
		return fmt.Errorf("project event log event type %d must be positive", item.EventType)
	}
	if strings.TrimSpace(item.IdempotencyKey) == "" {
		return fmt.Errorf("project event log idempotency key is empty")
	}

	occurredAt := item.OccurredAt.UTC()
	if occurredAt.IsZero() {
		occurredAt = time.Now().UTC()
	}
	payload := defaultJSONPayload(item.Payload)
	if !json.Valid([]byte(payload)) {
		return fmt.Errorf("project event log payload is not valid JSON")
	}

	if s.queries == nil {
		return fmt.Errorf("application postgres database is not configured")
	}
	if err := s.queries.AddProjectEventLog(ctx, appsqlc.AddProjectEventLogParams{
		Contract:       item.Contract.Bytes(),
		EventType:      item.EventType,
		OccurredAt:     pgtype.Timestamptz{Time: occurredAt, Valid: true},
		Message:        nullablePgText(item.Message),
		Column5:        []byte(payload),
		IdempotencyKey: item.IdempotencyKey,
	}); err != nil {
		return fmt.Errorf("add project event log: %w", err)
	}
	return nil
}

func (s *SQLStore) ListProjectEventLogsByContract(ctx context.Context, contract common.Address) ([]ProjectEventLog, error) {
	if s.queries == nil {
		return nil, fmt.Errorf("application postgres database is not configured")
	}
	rows, err := s.queries.ListProjectEventLogsByContract(ctx, contract.Bytes())
	if err != nil {
		return nil, fmt.Errorf("list project event logs: %w", err)
	}

	items := make([]ProjectEventLog, 0, len(rows))
	for _, row := range rows {
		items = append(items, projectEventLogFromSQLC(row))
	}
	return items, nil
}

func nullablePgText(value string) pgtype.Text {
	if strings.TrimSpace(value) == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: value, Valid: true}
}

func projectEventLogFromSQLC(row appsqlc.ProjectEventLog) ProjectEventLog {
	return ProjectEventLog{
		ID:             row.ID,
		Contract:       common.BytesToAddress(row.Contract),
		EventType:      row.EventType,
		OccurredAt:     row.OccurredAt.Time,
		Message:        row.Message.String,
		Payload:        string(row.Payload),
		IdempotencyKey: row.IdempotencyKey,
		CreatedAt:      row.CreatedAt.Time,
	}
}

func defaultJSONPayload(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "{}"
	}
	return trimmed
}
