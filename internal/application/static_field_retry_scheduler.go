package application

import (
	"context"
	"time"
)

type StaticFieldRetryScheduler interface {
	Run(ctx context.Context)
}

type staticFieldRetrySchedulerImpl struct {
	store    staticFieldDueLister
	queue    StaticFieldQueue
	interval time.Duration
	limit    int
}

type staticFieldDueLister interface {
	ListDueStaticFields(ctx context.Context, now time.Time, limit int) ([]StaticFieldResolveRequest, error)
}

func NewStaticFieldRetryScheduler(
	store staticFieldDueLister,
	queue StaticFieldQueue,
	interval time.Duration,
	limit int,
) StaticFieldRetryScheduler {
	if interval <= 0 {
		interval = 3 * time.Second
	}
	if limit <= 0 {
		limit = 500
	}

	return &staticFieldRetrySchedulerImpl{
		store:    store,
		queue:    queue,
		interval: interval,
		limit:    limit,
	}
}

func (s *staticFieldRetrySchedulerImpl) Run(ctx context.Context) {
	s.enqueueDueFields(ctx)

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.enqueueDueFields(ctx)
		}
	}
}

func (s *staticFieldRetrySchedulerImpl) enqueueDueFields(ctx context.Context) {
	now := time.Now()
	requests, err := s.store.ListDueStaticFields(ctx, now, s.limit)
	if err != nil {
		return
	}

	for _, req := range requests {
		req.Reason = StaticResolveReasonRetry
		req.CreatedAt = now
		req.Force = false
		_ = s.queue.Enqueue(ctx, req)
	}
}
