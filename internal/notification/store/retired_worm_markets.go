package store

import (
	"context"
	"fmt"
	"time"

	q "github.com/useryege/athena/internal/notification/store/sqlc"
)

// WormMarketsRetirementCounts describes Worm Markets retirement progress.
// RetireWormMarketsNotifications returns Cancelled as the rows newly changed by
// that batch. CountWormMarketsNotifications returns Cancelled as the current
// total whose status and exact retirement reason both identify this retirement.
// Pending and Sending are a transaction-consistent snapshot in either method.
type WormMarketsRetirementCounts struct {
	Cancelled int64 `json:"cancelled"`
	Pending   int64 `json:"pending"`
	Sending   int64 `json:"sending"`
}

// RetireWormMarketsNotifications cancels one bounded batch. Its candidate row
// lock is the same row lock used by delivery authorization: an authorization
// that commits first remains sending, while a cancellation that commits first
// makes that delivery ineligible for a permit.
func (s *SQLStore) RetireWormMarketsNotifications(ctx context.Context, batchSize int32) (WormMarketsRetirementCounts, error) {
	if err := s.transactional(); err != nil {
		return WormMarketsRetirementCounts{}, err
	}
	if batchSize <= 0 {
		return WormMarketsRetirementCounts{}, fmt.Errorf("retirement batch size must be positive")
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return WormMarketsRetirementCounts{}, err
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	if _, err = tx.Exec(ctx, `SET LOCAL lock_timeout='2s'; SET LOCAL statement_timeout='5s'`); err != nil {
		return WormMarketsRetirementCounts{}, err
	}
	queries := q.New(tx)
	cancelled, err := queries.CancelWormMarketsNotificationsPending(ctx, batchSize)
	if err != nil {
		return WormMarketsRetirementCounts{}, err
	}
	snapshot, err := queries.CountWormMarketsNotifications(ctx)
	if err != nil {
		return WormMarketsRetirementCounts{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return WormMarketsRetirementCounts{}, err
	}
	return WormMarketsRetirementCounts{
		Cancelled: cancelled,
		Pending:   snapshot.Pending,
		Sending:   snapshot.Sending,
	}, nil
}

// CountWormMarketsNotifications returns a fresh snapshot for the fixed five
// sources. Cancelled counts only rows carrying WORM_MARKETS_RETIRED.
func (s *SQLStore) CountWormMarketsNotifications(ctx context.Context) (WormMarketsRetirementCounts, error) {
	if err := s.configured(); err != nil {
		return WormMarketsRetirementCounts{}, err
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	r, err := s.queries.CountWormMarketsNotifications(ctx)
	if err != nil {
		return WormMarketsRetirementCounts{}, err
	}
	return WormMarketsRetirementCounts{Cancelled: r.Cancelled, Pending: r.Pending, Sending: r.Sending}, nil
}
