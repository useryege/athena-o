package chainingestor

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	tokenstore "github.com/useryege/athena/internal/token/store"
)

const (
	ChainIDEthereumMainnet int64 = 1
	ChainIDBSCMainnet      int64 = 56

	ethereumPollInterval = 30 * time.Second
	bscPollInterval      = 3 * time.Second
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
	if strings.TrimSpace(w.opts.EthNodeWSURL) == "" {
		return errNodeWSURLRequired(ChainIDEthereumMainnet)
	}
	if strings.TrimSpace(w.opts.BSCNodeWSURL) == "" {
		return errNodeWSURLRequired(ChainIDBSCMainnet)
	}

	runCtx, cancel := context.WithCancel(ctx)
	runners := make(map[int64]*chainRunner)
	for _, cfg := range []chainConfig{
		{chainID: ChainIDEthereumMainnet, name: "Ethereum Mainnet", nodeWSURL: w.opts.EthNodeWSURL, pollInterval: ethereumPollInterval},
		{chainID: ChainIDBSCMainnet, name: "BSC Mainnet", nodeWSURL: w.opts.BSCNodeWSURL, pollInterval: bscPollInterval},
	} {
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
			store:          w.opts.Store,
			chainID:        cfg.chainID,
			chainName:      cfg.name,
			nodeWSURL:      cfg.nodeWSURL,
			nodeWSUseProxy: w.opts.NodeWSUseProxy,
			pollInterval:   cfg.pollInterval,
		})
		runners[cfg.chainID] = runner
		log.WithFields(log.Fields{
			"chain_id":            checkpoint.ChainID,
			"chain_name":          checkpoint.ChainName,
			"cursor_block_number": checkpoint.CursorBlockNumber,
			"poll_interval":       cfg.pollInterval.String(),
		}).Info("token chain ingestor checkpoint activated")
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
	chainID      int64
	name         string
	nodeWSURL    string
	pollInterval time.Duration
}
