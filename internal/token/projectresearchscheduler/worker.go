package projectresearchscheduler

import (
	"context"
	"fmt"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	tokenstore "github.com/useryege/athena/internal/token/store"
)

type Options struct {
	Store                  *tokenstore.SQLStore
	ChainStateInterval     time.Duration
	WalletAssetInterval    time.Duration
	SimulationInterval     time.Duration
	AveInterval            time.Duration
	ContractSourceInterval time.Duration
	ResearchTTL            time.Duration
	PollInterval           time.Duration
}
type Worker struct {
	opts   Options
	mu     sync.Mutex
	cancel context.CancelFunc
	done   chan struct{}
}

func NewWorker(opts Options) *Worker {
	if opts.ChainStateInterval <= 0 {
		opts.ChainStateInterval = 15 * time.Second
	}
	if opts.WalletAssetInterval <= 0 {
		opts.WalletAssetInterval = time.Minute
	}
	if opts.SimulationInterval <= 0 {
		opts.SimulationInterval = time.Minute
	}
	if opts.AveInterval <= 0 {
		opts.AveInterval = 5 * time.Minute
	}
	if opts.ContractSourceInterval <= 0 {
		opts.ContractSourceInterval = 10 * time.Minute
	}
	if opts.ResearchTTL <= 0 {
		opts.ResearchTTL = 168 * time.Hour
	}
	if opts.PollInterval <= 0 {
		opts.PollInterval = time.Second
	}
	return &Worker{opts: opts}
}
func (w *Worker) Start(ctx context.Context) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.cancel != nil {
		return nil
	}
	if w.opts.Store == nil {
		return fmt.Errorf("token research scheduler store is required")
	}
	policies := map[string]time.Duration{tokenstore.ProjectDataCollectionTypeChainState: w.opts.ChainStateInterval, tokenstore.ProjectDataCollectionTypeWalletAssetState: w.opts.WalletAssetInterval, tokenstore.ProjectDataCollectionTypeSimulationResult: w.opts.SimulationInterval, tokenstore.ProjectDataCollectionTypeAve: w.opts.AveInterval, tokenstore.ProjectDataCollectionTypeContractCodeSource: w.opts.ContractSourceInterval}
	if err := w.opts.Store.ApplyResearchPolicy(ctx, policies, w.opts.ResearchTTL); err != nil {
		return err
	}
	runCtx, cancel := context.WithCancel(ctx)
	w.cancel = cancel
	w.done = make(chan struct{})
	go w.run(runCtx)
	return nil
}
func (w *Worker) Stop(ctx context.Context) error {
	w.mu.Lock()
	cancel, done := w.cancel, w.done
	w.cancel = nil
	w.done = nil
	w.mu.Unlock()
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
	ticker := time.NewTicker(w.opts.PollInterval)
	defer ticker.Stop()
	for {
		if err := w.opts.Store.MaintainResearchLifecycle(ctx); err != nil {
			log.WithError(err).Error("token research lifecycle maintenance failed")
		}
		if n, err := w.opts.Store.ScheduleDueProjectDataCollectionTasks(ctx, 100); err != nil {
			log.WithError(err).Error("token research scheduler failed")
		} else if n > 0 {
			log.WithField("task_count", n).Debug("token research scheduler created tasks")
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
