package projectqualifier

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/useryege/athena/common"
	tokenstore "github.com/useryege/athena/internal/token/store"
)

const (
	pollInterval   = 3 * time.Second
	candidateLimit = int32(100)
)

type Options struct {
	Store          *tokenstore.SQLStore
	EthNodeWSURL   string
	BSCNodeWSURL   string
	NodeWSUseProxy bool
}

type Worker struct {
	opts Options

	startStopMu sync.Mutex
	cancel      context.CancelFunc
	done        chan struct{}
	runner      *qualifierRunner
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
	if strings.TrimSpace(w.opts.EthNodeWSURL) == "" {
		return errNodeWSURLRequired(common.ChainIDEthereumMainnet)
	}
	if strings.TrimSpace(w.opts.BSCNodeWSURL) == "" {
		return errNodeWSURLRequired(common.ChainIDBSCMainnet)
	}

	runCtx, cancel := context.WithCancel(ctx)
	runner := newQualifierRunner(qualifierRunnerOptions{
		store:          w.opts.Store,
		nodeWSURLs:     w.nodeWSURLs(),
		nodeWSUseProxy: w.opts.NodeWSUseProxy,
		pollInterval:   pollInterval,
		candidateLimit: candidateLimit,
	})
	w.cancel = cancel
	w.done = make(chan struct{})
	w.runner = runner
	go w.run(runCtx, runner)
	return nil
}

func (w *Worker) Stop(ctx context.Context) error {
	w.startStopMu.Lock()
	cancel := w.cancel
	done := w.done
	runner := w.runner
	w.cancel = nil
	w.done = nil
	w.runner = nil
	w.startStopMu.Unlock()

	if cancel != nil {
		cancel()
	}
	if runner != nil {
		runner.close()
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

func (w *Worker) run(ctx context.Context, runner *qualifierRunner) {
	defer close(w.done)
	runner.run(ctx)
}

func (w *Worker) nodeWSURLs() map[int64]string {
	return map[int64]string{
		common.ChainIDEthereumMainnet: w.opts.EthNodeWSURL,
		common.ChainIDBSCMainnet:      w.opts.BSCNodeWSURL,
	}
}
