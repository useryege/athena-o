package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
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

	_, err := s.db.ExecContext(ctx, `
INSERT INTO project_event_log (
  contract,
  event_type,
  occurred_at,
  message,
  payload,
  idempotency_key
) VALUES ($1, $2, $3, $4, $5::jsonb, $6)
ON CONFLICT (contract, idempotency_key) DO NOTHING
`, item.Contract.Bytes(), item.EventType, occurredAt, nullIfEmpty(item.Message), payload, item.IdempotencyKey)
	if err != nil {
		return fmt.Errorf("add project event log: %w", err)
	}
	return nil
}

func (s *SQLStore) ListProjectEventLogsByContract(ctx context.Context, contract common.Address) ([]ProjectEventLog, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT
  id,
  contract,
  event_type,
  occurred_at,
  message,
  payload,
  idempotency_key,
  created_at
FROM project_event_log
WHERE contract = $1
ORDER BY occurred_at DESC, id DESC
`, contract.Bytes())
	if err != nil {
		return nil, fmt.Errorf("list project event logs: %w", err)
	}
	defer rows.Close()

	items := make([]ProjectEventLog, 0)
	for rows.Next() {
		item, err := scanProjectEventLogRow(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate project event logs: %w", err)
	}
	return items, nil
}

func scanProjectEventLogRow(scanner rowScanner) (ProjectEventLog, error) {
	var (
		item           ProjectEventLog
		contract       []byte
		message        sql.NullString
		payload        []byte
		idempotencyKey string
	)
	if err := scanner.Scan(&item.ID, &contract, &item.EventType, &item.OccurredAt, &message, &payload, &idempotencyKey, &item.CreatedAt); err != nil {
		return ProjectEventLog{}, fmt.Errorf("scan project event log: %w", err)
	}
	item.Contract = common.BytesToAddress(contract)
	item.Message = message.String
	item.Payload = string(payload)
	item.IdempotencyKey = idempotencyKey
	return item, nil
}

func nullIfEmpty(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}

func defaultJSONPayload(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "{}"
	}
	return trimmed
}
