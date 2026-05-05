package application

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

type StaticResolveReason string

const (
	StaticResolveReasonProjectCreated StaticResolveReason = "project_created"
	StaticResolveReasonRetry          StaticResolveReason = "retry"
	StaticResolveReasonManual         StaticResolveReason = "manual"
)

type StaticFieldResolveRequest struct {
	ProjectID uuid.UUID
	Field     StaticField
	Force     bool
	Reason    StaticResolveReason
	CreatedAt time.Time
}

type StaticFieldQueue interface {
	Enqueue(ctx context.Context, req StaticFieldResolveRequest) error
	Dequeue(ctx context.Context) (StaticFieldResolveRequest, error)
	Done(ctx context.Context, req StaticFieldResolveRequest, err error) error
	Close() error
	Len() int
}

type staticFieldQueueImpl struct {
	ch chan StaticFieldResolveRequest
	mu sync.Mutex

	closed   bool
	queued   map[string]StaticFieldResolveRequest
	inFlight map[string]StaticFieldResolveRequest
	pending  map[string]StaticFieldResolveRequest
}

var (
	ErrStaticFieldQueueClosed = errors.New("static field queue is closed")
	ErrStaticFieldDequeueStop = errors.New("static field queue dequeue stopped")
)

func NewStaticFieldQueue(buffer int) StaticFieldQueue {
	if buffer <= 0 {
		buffer = 1024
	}

	return &staticFieldQueueImpl{
		ch:       make(chan StaticFieldResolveRequest, buffer),
		queued:   make(map[string]StaticFieldResolveRequest),
		inFlight: make(map[string]StaticFieldResolveRequest),
		pending:  make(map[string]StaticFieldResolveRequest),
	}
}

func (q *staticFieldQueueImpl) Enqueue(ctx context.Context, req StaticFieldResolveRequest) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	q.mu.Lock()
	defer q.mu.Unlock()

	if q.closed {
		return ErrStaticFieldQueueClosed
	}

	key := staticFieldQueueKey(req)
	if _, ok := q.queued[key]; ok {
		return nil
	}
	if _, ok := q.inFlight[key]; ok {
		return nil
	}
	if _, ok := q.pending[key]; ok {
		return nil
	}

	select {
	case q.ch <- req:
		q.queued[key] = req
	default:
		q.pending[key] = req
	}

	return nil
}

func (q *staticFieldQueueImpl) Dequeue(ctx context.Context) (StaticFieldResolveRequest, error) {
	select {
	case <-ctx.Done():
		return StaticFieldResolveRequest{}, ctx.Err()
	case req, ok := <-q.ch:
		if !ok {
			return StaticFieldResolveRequest{}, ErrStaticFieldDequeueStop
		}

		key := staticFieldQueueKey(req)
		q.mu.Lock()
		delete(q.queued, key)
		q.inFlight[key] = req
		q.mu.Unlock()

		return req, nil
	}
}

func (q *staticFieldQueueImpl) Done(ctx context.Context, req StaticFieldResolveRequest, err error) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	q.mu.Lock()
	defer q.mu.Unlock()

	delete(q.inFlight, staticFieldQueueKey(req))
	q.flushPendingLocked()
	return nil
}

func (q *staticFieldQueueImpl) Close() error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.closed {
		return nil
	}

	q.closed = true
	close(q.ch)
	return nil
}

func (q *staticFieldQueueImpl) Len() int {
	q.mu.Lock()
	defer q.mu.Unlock()

	return len(q.queued) + len(q.inFlight) + len(q.pending)
}

func (q *staticFieldQueueImpl) flushPendingLocked() {
	if q.closed {
		return
	}

	for {
		moved := false
		for key, req := range q.pending {
			select {
			case q.ch <- req:
				q.queued[key] = req
				delete(q.pending, key)
				moved = true
			default:
				return
			}
		}

		if !moved {
			return
		}
	}
}

func staticFieldQueueKey(req StaticFieldResolveRequest) string {
	return fmt.Sprintf("%s:%s", req.ProjectID.String(), req.Field)
}
