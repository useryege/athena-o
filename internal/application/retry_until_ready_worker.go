package application

import "context"

type RetryUntilReadyWorker struct {
	queue    RetryUntilReadyQueue
	resolver RetryUntilReadyResolver
}

func NewRetryUntilReadyWorker(queue RetryUntilReadyQueue, resolver RetryUntilReadyResolver) *RetryUntilReadyWorker {
	return &RetryUntilReadyWorker{
		queue:    queue,
		resolver: resolver,
	}
}

func (w *RetryUntilReadyWorker) Run(ctx context.Context) {
	for {
		req, err := w.queue.Dequeue(ctx)
		if err != nil {
			return
		}

		resolveErr := w.resolver.Resolve(ctx, req)

		_ = w.queue.Done(ctx, req, resolveErr)
	}
}
