package application

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
)

type RetryUntilReadyField string

const (
	DelayedFieldSourceCode    RetryUntilReadyField = "source_code"
	DelayedFieldSourceCodeABI RetryUntilReadyField = "source_code_abi"
)

type RetryUntilReadyFieldKey struct {
	ProjectID uuid.UUID
	Field     RetryUntilReadyField
}

func (k RetryUntilReadyFieldKey) DedupeKey() string {
	return k.ProjectID.String() + ":" + string(k.Field)
}

type RetryUntilReadyPool interface {
	Add(ctx context.Context, key RetryUntilReadyFieldKey, nextAttemptAt time.Time) error
	Remove(ctx context.Context, key RetryUntilReadyFieldKey) error
	ListDue(ctx context.Context, now time.Time, limit int) ([]RetryUntilReadyFieldKey, error)
}

type retryUntilReadyPoolImpl struct {
	mu    sync.RWMutex
	items map[RetryUntilReadyFieldKey]time.Time
}

func NewRetryUntilReadyPool() RetryUntilReadyPool {
	return &retryUntilReadyPoolImpl{
		items: make(map[RetryUntilReadyFieldKey]time.Time),
	}
}

func (p *retryUntilReadyPoolImpl) Add(ctx context.Context, key RetryUntilReadyFieldKey, nextAttemptAt time.Time) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.items[key] = nextAttemptAt
	return nil
}

func (p *retryUntilReadyPoolImpl) Remove(ctx context.Context, key RetryUntilReadyFieldKey) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.items, key)
	return nil
}

func (p *retryUntilReadyPoolImpl) ListDue(ctx context.Context, now time.Time, limit int) ([]RetryUntilReadyFieldKey, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if limit <= 0 {
		limit = 1000
	}

	result := make([]RetryUntilReadyFieldKey, 0, limit)

	for key, nextAttemptAt := range p.items {
		if nextAttemptAt.IsZero() || !now.Before(nextAttemptAt) {
			result = append(result, key)
			if len(result) >= limit {
				return result, nil
			}
		}
	}

	return result, nil
}
