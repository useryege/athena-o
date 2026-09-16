package store

import (
	"context"
	"fmt"
	"time"

	q "github.com/useryege/athena/internal/notification/store/sqlc"
)

type RetiredSportsCounts struct {
	Pending int64 `json:"pending"`
	Sending int64 `json:"sending"`
}

func (s *SQLStore) CountRetiredSports(ctx context.Context) (RetiredSportsCounts, error) {
	if err := s.configured(); err != nil {
		return RetiredSportsCounts{}, err
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	r, err := s.queries.CountRetiredSports(ctx)
	return RetiredSportsCounts{Pending: r.Pending, Sending: r.Sending}, err
}

// CancelRetiredSportsPending shares the delivery row lock with Authorize and
// RecordOutcome. A consumed permit is never revoked or rewritten by retirement.
func (s *SQLStore) CancelRetiredSportsPending(ctx context.Context, limit int32) (int64, error) {
	if err := s.transactional(); err != nil {
		return 0, err
	}
	if limit < 1 || limit > 100 {
		return 0, fmt.Errorf("retirement batch must contain 1 to 100 deliveries")
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `SET LOCAL lock_timeout='2s'; SET LOCAL statement_timeout='5s'`); err != nil {
		return 0, err
	}
	n, err := q.New(tx).CancelRetiredSportsPending(ctx, limit)
	if err != nil {
		return 0, err
	}
	if err = tx.Commit(ctx); err != nil {
		return 0, err
	}
	return n, nil
}
