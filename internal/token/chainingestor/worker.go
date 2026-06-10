package chainingestor

import (
	"context"
	"fmt"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/useryege/athena/common"
	tokenstore "github.com/useryege/athena/internal/token/store"
	"github.com/useryege/athena/util/ethws"
)

const (
	ethereumPollInterval = 30 * time.Second
	bscPollInterval      = 3 * time.Second

	ethereumBlockFetchConcurrency = 5
	bscBlockFetchConcurrency      = 10
)

type Options struct {
	Store          *tokenstore.SQLStore
	EthNodeWSURLs  []string
	BSCNodeWSURLs  []string
	EthEnabled     bool
	BSCEnabled     bool
	NodeWSUseProxy bool
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

	runCtx, cancel := context.WithCancel(ctx)
	runners := make(map[int64]*chainRunner)
	for _, cfg := range w.chainConfigs() {
		if !cfg.enabled {
			log.WithFields(log.Fields{
				"chain_id":   cfg.chainID,
				"chain_name": cfg.name,
			}).Info("token chain ingestor chain disabled by runtime config")
			continue
		}
		nodeWSURLs := ethws.NormalizeEndpoints(cfg.nodeWSURLs)
		if len(nodeWSURLs) == 0 {
			cancel()
			return errNodeWSURLRequired(cfg.chainID)
		}
		checkpoint, err := w.opts.Store.GetChainIngestCheckpoint(ctx, cfg.chainID)
		if err != nil {
			cancel()
			return err
		}
		if checkpoint == nil {
			cancel()
			return fmt.Errorf("token chain ingestor chain %d is not configured", cfg.chainID)
		}
		if !checkpoint.Enabled {
			log.WithFields(log.Fields{
				"chain_id":   checkpoint.ChainID,
				"chain_name": checkpoint.ChainName,
			}).Info("token chain ingestor chain is disabled")
			continue
		}
		checkpoint.Status = tokenstore.ChainIngestStatusRunning
		if _, err := w.opts.Store.UpsertChainIngestCheckpoint(ctx, *checkpoint); err != nil {
			cancel()
			return err
		}
		runner := newChainRunner(chainRunnerOptions{
			store:                 w.opts.Store,
			chainID:               cfg.chainID,
			chainName:             cfg.name,
			nodeWSURLs:            nodeWSURLs,
			nodeWSUseProxy:        w.opts.NodeWSUseProxy,
			pollInterval:          cfg.pollInterval,
			blockFetchConcurrency: cfg.blockFetchConcurrency,
		})
		runners[cfg.chainID] = runner
		log.WithFields(log.Fields{
			"chain_id":            checkpoint.ChainID,
			"chain_name":          checkpoint.ChainName,
			"cursor_block_number": checkpoint.CursorBlockNumber,
			"poll_interval":       cfg.pollInterval.String(),
			"fetch_concurrency":   cfg.blockFetchConcurrency,
		}).Info("token chain ingestor checkpoint activated")
	}
	if len(runners) == 0 {
		log.Info("token chain ingestor has no enabled chains")
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
	for _, runner := range runners {
		runner.close()
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
		if _, err := w.opts.Store.UpdateChainIngestCheckpointStatus(ctx, chainID, tokenstore.ChainIngestStatusStopped); err != nil {
			if result == nil {
				result = err
			}
			log.WithError(err).WithField("chain_id", chainID).Error("token chain ingestor failed to stop checkpoint")
			continue
		}
		log.WithField("chain_id", chainID).Info("token chain ingestor checkpoint stopped")
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

type chainConfig struct {
	chainID               int64
	name                  string
	nodeWSURLs            []string
	pollInterval          time.Duration
	blockFetchConcurrency int
	enabled               bool
}

func (w *Worker) chainConfigs() []chainConfig {
	return []chainConfig{
		{
			chainID:               common.ChainIDEthereumMainnet,
			name:                  common.ChainNameEthereumMainnet,
			nodeWSURLs:            w.opts.EthNodeWSURLs,
			pollInterval:          ethereumPollInterval,
			blockFetchConcurrency: ethereumBlockFetchConcurrency,
			enabled:               w.opts.EthEnabled,
		},
		{
			chainID:               common.ChainIDBSCMainnet,
			name:                  common.ChainNameBSCMainnet,
			nodeWSURLs:            w.opts.BSCNodeWSURLs,
			pollInterval:          bscPollInterval,
			blockFetchConcurrency: bscBlockFetchConcurrency,
			enabled:               w.opts.BSCEnabled,
		},
	}
}
