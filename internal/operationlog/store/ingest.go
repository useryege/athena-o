package store

import (
	"bytes"
	"context"
	"crypto/sha256"
	"github.com/useryege/athena/internal/operationlog/event"
	"github.com/useryege/athena/internal/operationlog/ingest"
	"github.com/useryege/athena/internal/operationlog/store/sqlc"
	"time"
)

// Append commits the immutable event and its delivery together. A commit error
// is deliberately not converted to success; a retry resolves ambiguous commits.
func (s *Store) Append(ctx context.Context, e event.Event) error {
	ctx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()
	payload, err := event.Canonical(e)
	if err != nil {
		return err
	}
	hash := sha256.Sum256(payload)
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	q := sqlc.New(tx)
	n, err := q.InsertEvent(ctx, sqlc.InsertEventParams{EventID: id(e.EventID), OperationID: id(e.OperationID), Phase: string(e.Phase), ProducerID: id(e.ProducerID), SchemaVersion: int32(e.SchemaVersion), OccurredAt: timestamp(e.OccurredAt), Payload: payload, PayloadHash: hash[:]})
	if err != nil {
		return err
	}
	if n == 0 {
		existing, err := q.FindConflictingEvents(ctx, sqlc.FindConflictingEventsParams{EventID: id(e.EventID), OperationID: id(e.OperationID), Phase: string(e.Phase)})
		if err != nil {
			return err
		}
		if len(existing) != 1 || existing[0].EventID != id(e.EventID) || existing[0].OperationID != id(e.OperationID) || existing[0].Phase != string(e.Phase) || !bytes.Equal(existing[0].PayloadHash, hash[:]) {
			return ingest.ErrConflict
		}
	} else if err = q.InsertDelivery(ctx, id(e.EventID)); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
