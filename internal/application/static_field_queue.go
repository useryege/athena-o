package application

import (
	"context"
	"sync"
)

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

// func NewStaticFieldQueue() StaticFieldQueue {
// 	return &staticFieldQueueImpl{}
// }
