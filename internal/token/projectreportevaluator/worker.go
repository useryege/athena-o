package projectreportevaluator

import (
	"context"
	"sync"
	"time"

	tokenstore "github.com/useryege/athena/internal/token/store"
)

const (
	pollInterval = time.Second
	taskLimit    = int32(20)
)

type Options struct {
	Store *tokenstore.SQLStore
}

type Worker struct {
	opts Options

	startStopMu sync.Mutex
	cancel      context.CancelFunc
	done        chan struct{}
}

func NewWorker(opts Options) *Worker {
	return &Worker{opts: opts}
}

func (w *Worker) Start(ctx context.Context) error {
	w.startStopMu.Lock()
	defer w.startStopMu.Unlock()
	if w.cancel != nil {
		return nil
	}
	if w.opts.Store == nil {
		return errStoreRequired()
	}
	runCtx, cancel := context.WithCancel(ctx)
	w.cancel = cancel
	w.done = make(chan struct{})
	go w.run(runCtx)
	return nil
}

func (w *Worker) Stop(ctx context.Context) error {
	w.startStopMu.Lock()
	cancel := w.cancel
	done := w.done
	w.cancel = nil
	w.done = nil
	w.startStopMu.Unlock()

	if cancel != nil {
		cancel()
	}
	if done != nil {
		select {
		case <-done:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}

func (w *Worker) run(ctx context.Context) {
	defer close(w.done)
	runner := reportEvaluatorRunner{
		store:        w.opts.Store,
		pollInterval: pollInterval,
		taskLimit:    taskLimit,
	}
	runner.run(ctx)
}
