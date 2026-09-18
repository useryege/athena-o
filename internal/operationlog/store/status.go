package store

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/useryege/athena/internal/operationlog/event"
	"github.com/useryege/athena/internal/operationlog/ingest"
	"github.com/useryege/athena/internal/operationlog/store/sqlc"
	"math"
	"time"
)

func (s *Store) PublishStatus(ctx context.Context, v ingest.Status) error {
	ctx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()
	u, err := uuid.Parse(v.ProducerID)
	if err != nil || u == uuid.Nil || u.String() != v.ProducerID || v.StartedAt.IsZero() || v.ObservedAt.Before(v.StartedAt) {
		return fmt.Errorf("%w: producer status", event.ErrInvalid)
	}
	counts := []uint64{v.ConfirmedEvents, v.UnconfirmedEvents, v.InvalidEvents, v.CapacityRejectedEvents, v.InFlightEvents}
	var sum uint64
	for _, n := range append(append([]uint64{}, counts...), v.SnapshotNo, v.AttemptedEvents) {
		if n > math.MaxInt64 {
			return fmt.Errorf("%w: producer counter overflow", event.ErrInvalid)
		}
	}
	for _, n := range counts {
		if n > math.MaxInt64-sum {
			return fmt.Errorf("%w: producer counter overflow", event.ErrInvalid)
		}
		sum += n
	}
	if sum != v.AttemptedEvents {
		return fmt.Errorf("%w: producer counters", event.ErrInvalid)
	}
	var reachable pgtype.Bool
	if v.PersistenceReachable != nil {
		reachable = pgtype.Bool{Bool: *v.PersistenceReachable, Valid: true}
	}
	return sqlc.New(s.pool).PublishStatus(ctx, sqlc.PublishStatusParams{ProducerID: id(v.ProducerID), StartedAt: timestamp(v.StartedAt), LastSeenAt: timestamp(v.ObservedAt), StoppedAt: optionalTime(v.StoppedAt), SnapshotNo: int64(v.SnapshotNo), AttemptedEvents: int64(v.AttemptedEvents), ConfirmedEvents: int64(v.ConfirmedEvents), UnconfirmedEvents: int64(v.UnconfirmedEvents), InvalidEvents: int64(v.InvalidEvents), CapacityRejectedEvents: int64(v.CapacityRejectedEvents), InFlightEvents: int64(v.InFlightEvents), LastFailureAt: optionalTime(v.LastFailureAt), LastFailureCode: textValue(v.LastFailureCode), LastRecoveredAt: optionalTime(v.LastRecoveredAt), LastConfirmedAt: optionalTime(v.LastConfirmedAt), PersistenceReachable: reachable})
}
