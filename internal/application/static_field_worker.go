package application

import "context"

type StaticFieldWorker interface {
	Run(ctx context.Context)
}

type staticFieldWorkerImpl struct {
	queue    StaticFieldQueue
	resolver StaticFieldResolver
}

var _ StaticFieldWorker = &staticFieldWorkerImpl{}

func NewStaticFieldWorker(queue StaticFieldQueue, resolver StaticFieldResolver) StaticFieldWorker {
	return &staticFieldWorkerImpl{
		queue:    queue,
		resolver: resolver,
	}
}

func (w *staticFieldWorkerImpl) Run(ctx context.Context) {
	for {
		req, err := w.queue.Dequeue(ctx)
		if err != nil {
			return
		}

		resolveErr := w.resolver.ResolveField(
			ctx,
			req.ProjectID,
			req.Field,
			req.Force,
		)

		_ = w.queue.Done(ctx, req, resolveErr)
	}
}
