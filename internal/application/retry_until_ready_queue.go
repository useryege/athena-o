package application

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
)

type RetryUntilReadyQueue interface {
	Enqueue(ctx context.Context, req RetryUntilReadyResolveRequest) error
	Dequeue(ctx context.Context) (RetryUntilReadyResolveRequest, error)
	Done(ctx context.Context, req RetryUntilReadyResolveRequest, err error) error
	Close() error
	Stats() RetryUntilReadyQueueStats
}

var _ RetryUntilReadyQueue = &retryUntilReadyQueueImpl{}

type retryUntilReadyQueueImpl struct {
	ch chan RetryUntilReadyResolveRequest

	mu sync.RWMutex

	closed bool

	queued   map[string]RetryUntilReadyResolveRequest // waiting to be dequeued
	inFlight map[string]RetryUntilReadyResolveRequest // being processed
	pending  map[string]RetryUntilReadyResolveRequest // waiting to be re-enqueued
}

func NewRetryUntilReadyQueue(bufferSize int) RetryUntilReadyQueue {
	if bufferSize <= 0 {
		bufferSize = 1024
	}

	return &retryUntilReadyQueueImpl{
		ch:       make(chan RetryUntilReadyResolveRequest, bufferSize),
		queued:   make(map[string]RetryUntilReadyResolveRequest),
		inFlight: make(map[string]RetryUntilReadyResolveRequest),
		pending:  make(map[string]RetryUntilReadyResolveRequest),
	}
}

var ErrRetryUntilReadyQueueClosed = errors.New("retry until ready queue closed")

func (q *retryUntilReadyQueueImpl) Enqueue(
	ctx context.Context,
	req RetryUntilReadyResolveRequest,
) error {
	if req.CreatedAt.IsZero() {
		req.CreatedAt = time.Now()
	}

	key := req.DedupeKey()

	q.mu.Lock()

	if q.closed {
		q.mu.Unlock()
		return ErrRetryUntilReadyQueueClosed
	}

	if _, ok := q.inFlight[key]; ok {
		q.pending[key] = mergeDelayedFieldResolveRequest(q.pending[key], req)
		q.mu.Unlock()
		return nil
	}

	if _, ok := q.queued[key]; ok {
		q.queued[key] = mergeDelayedFieldResolveRequest(q.queued[key], req)
		q.mu.Unlock()
		return nil
	}

	q.queued[key] = req

	select {
	case q.ch <- req:
		q.mu.Unlock()
		return nil
	case <-ctx.Done():
		delete(q.queued, key)
		q.mu.Unlock()
		return ctx.Err()
	}
}

func mergeDelayedFieldResolveRequest(
	oldReq RetryUntilReadyResolveRequest,
	newReq RetryUntilReadyResolveRequest,
) RetryUntilReadyResolveRequest {
	if oldReq.ProjectID == uuid.Nil {
		return newReq
	}

	merged := oldReq

	if isResolveReasonHigherPriority(newReq.Reason, oldReq.Reason) {
		merged.Reason = newReq.Reason
	}

	if merged.CreatedAt.IsZero() || newReq.CreatedAt.Before(merged.CreatedAt) {
		merged.CreatedAt = newReq.CreatedAt
	}

	return merged
}

func isResolveReasonHigherPriority(newReason, oldReason ResolveReason) bool {
	return resolveReasonPriority(newReason) > resolveReasonPriority(oldReason)
}

func resolveReasonPriority(reason ResolveReason) int {
	switch reason {
	case ResolveReasonManual:
		return 100
	case ResolveReasonProjectCreated:
		return 80
	case ResolveReasonRetry:
		return 50
	default:
		return 0
	}
}

func (q *retryUntilReadyQueueImpl) Dequeue(
	ctx context.Context,
) (RetryUntilReadyResolveRequest, error) {
	select {
	case req, ok := <-q.ch:
		if !ok {
			return RetryUntilReadyResolveRequest{}, ErrRetryUntilReadyQueueClosed
		}

		key := req.DedupeKey()

		q.mu.Lock()
		delete(q.queued, key)

		if q.closed {
			q.mu.Unlock()
			return RetryUntilReadyResolveRequest{}, ErrRetryUntilReadyQueueClosed
		}

		q.inFlight[key] = req
		q.mu.Unlock()

		return req, nil
	case <-ctx.Done():
		return RetryUntilReadyResolveRequest{}, ctx.Err()
	}
}

func (q *retryUntilReadyQueueImpl) Done(
	ctx context.Context,
	req RetryUntilReadyResolveRequest,
	err error,
) error {
	key := req.DedupeKey()

	q.mu.Lock()

	delete(q.inFlight, key)

	pendingReq, hasPending := q.pending[key]
	if hasPending {
		delete(q.pending, key)

		if q.closed {
			q.mu.Unlock()
			return ErrRetryUntilReadyQueueClosed
		}

		q.queued[key] = pendingReq
	}

	if !hasPending {
		q.mu.Unlock()
		return nil
	}

	select {
	case q.ch <- pendingReq:
		q.mu.Unlock()
		return nil

	case <-ctx.Done():
		delete(q.queued, key)
		q.mu.Unlock()
		return ctx.Err()
	}
}

func (q *retryUntilReadyQueueImpl) Close() error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.closed {
		return nil
	}

	q.closed = true
	close(q.ch)

	return nil
}

type RetryUntilReadyQueueStats struct {
	Queued   int
	InFlight int
	Pending  int
	Closed   bool
}

func (q *retryUntilReadyQueueImpl) Stats() RetryUntilReadyQueueStats {
	q.mu.RLock()
	defer q.mu.RUnlock()

	return RetryUntilReadyQueueStats{
		Queued:   len(q.queued),
		InFlight: len(q.inFlight),
		Pending:  len(q.pending),
		Closed:   q.closed,
	}
}
