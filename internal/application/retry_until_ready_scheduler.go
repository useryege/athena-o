package application

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type RetryScheduler struct {
	pool     RetryUntilReadyPool
	queue    RetryUntilReadyResolveQueue
	interval time.Duration
	limit    int
}

type ResolveReason string

const (
	ResolveReasonRetry          ResolveReason = "retry"
	ResolveReasonManual         ResolveReason = "manual"
	ResolveReasonProjectCreated ResolveReason = "project_created"
)

type RetryUntilReadyResolveRequest struct {
	ProjectID uuid.UUID
	Field     RetryUntilReadyField
	Reason    ResolveReason
	CreatedAt time.Time
}

func (r RetryUntilReadyResolveRequest) DedupeKey() string {
	return r.ProjectID.String() + ":" + string(r.Field)
}

type RetryUntilReadyResolveQueue interface {
	Enqueue(ctx context.Context, req RetryUntilReadyResolveRequest) error
}

func NewRetryScheduler(
	pool RetryUntilReadyPool,
	queue RetryUntilReadyResolveQueue,
	interval time.Duration,
	limit int,
) *RetryScheduler {
	if interval <= 0 {
		interval = 1 * time.Minute
	}

	return &RetryScheduler{
		pool:     pool,
		queue:    queue,
		interval: interval,
		limit:    limit,
	}
}

func (s *RetryScheduler) Start(ctx context.Context) {
	s.run(ctx)

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.run(ctx)
		}
	}
}

func (s *RetryScheduler) run(ctx context.Context) {
	now := time.Now()
	requests, err := s.pool.ListDue(ctx, now, s.limit)
	if err != nil {
		return
	}

	for _, key := range requests {
		_ = s.queue.Enqueue(ctx, RetryUntilReadyResolveRequest{
			ProjectID: key.ProjectID,
			Field:     key.Field,
			Reason:    ResolveReasonRetry,
			CreatedAt: now,
		})
	}
}
