package projectqualifier

import (
	"context"
	"strings"
	"sync"
	"time"

	ethcommon "github.com/ethereum/go-ethereum/common"
	log "github.com/sirupsen/logrus"
	"github.com/useryege/athena/common"
	tokenstore "github.com/useryege/athena/internal/token/store"
)

const (
	pollInterval   = 3 * time.Second
	candidateLimit = int32(100)
)

type Options struct {
	Store             *tokenstore.SQLStore
	EthNodeWSURL      string
	BSCNodeWSURL      string
	EthAthenaContract string
	BSCAthenaContract string
	EthEnabled        bool
	BSCEnabled        bool
	NodeWSUseProxy    bool
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
	chainIDs, nodeWSURLs, athenaContracts, err := w.enabledChainRuntime()
	if err != nil {
		return err
	}

	runCtx, cancel := context.WithCancel(ctx)
	runner := newQualifierRunner(qualifierRunnerOptions{
		store:           w.opts.Store,
		chainIDs:        chainIDs,
		nodeWSURLs:      nodeWSURLs,
		athenaContracts: athenaContracts,
		nodeWSUseProxy:  w.opts.NodeWSUseProxy,
		pollInterval:    pollInterval,
		candidateLimit:  candidateLimit,
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

func (w *Worker) enabledChainRuntime() ([]int64, map[int64]string, map[int64]ethcommon.Address, error) {
	chainIDs := make([]int64, 0, 2)
	nodeWSURLs := make(map[int64]string)
	athenaContracts := make(map[int64]ethcommon.Address)
	for _, cfg := range w.chainConfigs() {
		if !cfg.enabled {
			log.WithFields(log.Fields{
				"chain_id":   cfg.chainID,
				"chain_name": common.ChainName(cfg.chainID),
			}).Info("token project qualifier chain disabled by runtime config")
			continue
		}
		if strings.TrimSpace(cfg.nodeWSURL) == "" {
			return nil, nil, nil, errNodeWSURLRequired(cfg.chainID)
		}
		athenaContract, err := parseAthenaContract(cfg.chainID, cfg.athenaContract)
		if err != nil {
			return nil, nil, nil, err
		}
		chainIDs = append(chainIDs, cfg.chainID)
		nodeWSURLs[cfg.chainID] = cfg.nodeWSURL
		athenaContracts[cfg.chainID] = athenaContract
	}
	if len(chainIDs) == 0 {
		log.Info("token project qualifier has no enabled chains")
	}
	return chainIDs, nodeWSURLs, athenaContracts, nil
}

func parseAthenaContract(chainID int64, value string) (ethcommon.Address, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return ethcommon.Address{}, errAthenaContractRequired(chainID)
	}
	if !ethcommon.IsHexAddress(value) {
		return ethcommon.Address{}, errAthenaContractInvalid(chainID, value)
	}
	address := ethcommon.HexToAddress(value)
	if address == (ethcommon.Address{}) {
		return ethcommon.Address{}, errAthenaContractInvalid(chainID, value)
	}
	return address, nil
}

type chainConfig struct {
	chainID        int64
	nodeWSURL      string
	athenaContract string
	enabled        bool
}

func (w *Worker) chainConfigs() []chainConfig {
	return []chainConfig{
		{
			chainID:        common.ChainIDEthereumMainnet,
			nodeWSURL:      w.opts.EthNodeWSURL,
			athenaContract: w.opts.EthAthenaContract,
			enabled:        w.opts.EthEnabled,
		},
		{
			chainID:        common.ChainIDBSCMainnet,
			nodeWSURL:      w.opts.BSCNodeWSURL,
			athenaContract: w.opts.BSCAthenaContract,
			enabled:        w.opts.BSCEnabled,
		},
	}
}
