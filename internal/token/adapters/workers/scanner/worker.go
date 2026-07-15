package scanner

import (
	"context"
	"fmt"
	"sync"

	log "github.com/sirupsen/logrus"
	"github.com/useryege/athena/internal/token/adapters/evm"
	"github.com/useryege/athena/internal/token/chainregistry"
	"github.com/useryege/athena/internal/token/discovery"
	"github.com/useryege/athena/internal/token/telemetry"
	"github.com/useryege/athena/util/ethws"
)

type Options struct {
	Store     discovery.ScannerRepository
	Chains    []chainregistry.Chain
	Clients   *evm.ChainClientRegistry
	Telemetry telemetry.Reporter
}

type Worker struct {
	opts Options

	startStopMu sync.Mutex
	cancel      context.CancelFunc
	done        chan struct{}
	runners     map[int64]*chainRunner
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
	if w.opts.Clients == nil {
		return fmt.Errorf("token chain scanner EVM client registry is required")
	}

	runCtx, cancel := context.WithCancel(ctx)
	runners := make(map[int64]*chainRunner)
	for _, cfg := range w.opts.Chains {
		if !cfg.Enabled {
			log.WithFields(log.Fields{
				"chain_id":   cfg.ID,
				"chain_name": cfg.Name,
			}).Info("token chain scanner chain disabled by runtime config")
			continue
		}
		nodeWSURLs := ethws.NormalizeEndpoints(cfg.NodeWSURLs)
		if len(nodeWSURLs) == 0 {
			cancel()
			return errNodeWSURLRequired(cfg.ID)
		}
		checkpoint, err := w.opts.Store.GetChainIngestCheckpoint(ctx, cfg.ID)
		if err != nil {
			cancel()
			return err
		}
		if checkpoint == nil {
			cancel()
			return fmt.Errorf("token chain scanner chain %d is not configured", cfg.ID)
		}
		if !checkpoint.Enabled {
			log.WithFields(log.Fields{
				"chain_id":   checkpoint.ChainID,
				"chain_name": checkpoint.ChainName,
			}).Info("token chain scanner chain is disabled")
			continue
		}
		checkpoint.Status = discovery.ChainIngestStatusRunning
		if _, err := w.opts.Store.UpsertChainIngestCheckpoint(ctx, *checkpoint); err != nil {
			cancel()
			return err
		}
		runner := newChainRunner(chainRunnerOptions{
			store:                 w.opts.Store,
			clients:               w.opts.Clients,
			chainID:               cfg.ID,
			chainName:             cfg.Name,
			pollInterval:          cfg.ScannerPollInterval,
			blockFetchConcurrency: cfg.BlockFetchConcurrency,
			telemetry:             w.opts.Telemetry,
		})
		runners[cfg.ID] = runner
		telemetry.Register(w.opts.Telemetry, telemetry.Scope{Component: "chain_scanner", ChainID: cfg.ID})
		log.WithFields(log.Fields{
			"chain_id":            checkpoint.ChainID,
			"chain_name":          checkpoint.ChainName,
			"cursor_block_number": checkpoint.CursorBlockNumber,
			"poll_interval":       cfg.ScannerPollInterval.String(),
			"fetch_concurrency":   cfg.BlockFetchConcurrency,
			"checkpoint_status":   checkpoint.Status,
		}).Info("token chain scanner runner started")
	}
	if len(runners) == 0 {
		log.Info("token chain scanner has no enabled chains")
	}

	w.cancel = cancel
	w.done = make(chan struct{})
	w.runners = runners
	go w.run(runCtx, runners)
	return nil
}

func (w *Worker) Stop(ctx context.Context) error {
	w.startStopMu.Lock()
	cancel := w.cancel
	done := w.done
	runners := w.runners
	w.cancel = nil
	w.done = nil
	w.runners = nil
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

	if w.opts.Store == nil {
		return nil
	}
	var result error
	for chainID := range runners {
		if _, err := w.opts.Store.UpdateChainIngestCheckpointStatus(ctx, chainID, discovery.ChainIngestStatusStopped); err != nil {
			if result == nil {
				result = err
			}
			log.WithError(err).WithField("chain_id", chainID).Error("token chain scanner failed to stop checkpoint")
			continue
		}
		log.WithField("chain_id", chainID).Info("token chain scanner checkpoint stopped")
	}
	return result
}

func (w *Worker) run(ctx context.Context, runners map[int64]*chainRunner) {
	defer close(w.done)
	var wg sync.WaitGroup
	for _, runner := range runners {
		wg.Add(1)
		go func(runner *chainRunner) {
			defer wg.Done()
			runner.run(ctx)
		}(runner)
	}
	wg.Wait()
}
