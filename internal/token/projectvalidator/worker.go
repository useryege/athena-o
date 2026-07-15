package projectvalidator

import (
	"context"
	"strings"
	"sync"
	"time"

	ethcommon "github.com/ethereum/go-ethereum/common"
	log "github.com/sirupsen/logrus"
	"github.com/useryege/athena/common"
	"github.com/useryege/athena/util/ethws"
)

const (
	pollInterval         = 3 * time.Second
	candidateLimit       = int32(100)
	candidateConcurrency = 10
)

type Options struct {
	Store             Store
	EthNodeWSURLs     []string
	BSCNodeWSURLs     []string
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
	runners     map[int64]*validatorRunner
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
	runners := make(map[int64]*validatorRunner, len(chainIDs))
	for _, chainID := range chainIDs {
		runners[chainID] = newValidatorRunner(validatorRunnerOptions{
			store:                w.opts.Store,
			chainIDs:             []int64{chainID},
			nodeWSURLs:           map[int64][]string{chainID: nodeWSURLs[chainID]},
			athenaContracts:      map[int64]ethcommon.Address{chainID: athenaContracts[chainID]},
			nodeWSUseProxy:       w.opts.NodeWSUseProxy,
			pollInterval:         pollInterval,
			candidateLimit:       candidateLimit,
			candidateConcurrency: candidateConcurrency,
		})
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
	return nil
}

func (w *Worker) run(ctx context.Context, runners map[int64]*validatorRunner) {
	defer close(w.done)
	var wg sync.WaitGroup
	for _, runner := range runners {
		wg.Add(1)
		go func(runner *validatorRunner) {
			defer wg.Done()
			runner.run(ctx)
		}(runner)
	}
	wg.Wait()
}

func (w *Worker) enabledChainRuntime() ([]int64, map[int64][]string, map[int64]ethcommon.Address, error) {
	chainIDs := make([]int64, 0, 2)
	nodeWSURLs := make(map[int64][]string)
	athenaContracts := make(map[int64]ethcommon.Address)
	for _, cfg := range w.chainConfigs() {
		if !cfg.enabled {
			log.WithFields(log.Fields{
				"chain_id":   cfg.chainID,
				"chain_name": common.ChainName(cfg.chainID),
			}).Info("token project validator chain disabled by runtime config")
			continue
		}
		normalizedNodeWSURLs := ethws.NormalizeEndpoints(cfg.nodeWSURLs)
		if len(normalizedNodeWSURLs) == 0 {
			return nil, nil, nil, errNodeWSURLRequired(cfg.chainID)
		}
		athenaContract, err := parseAthenaContract(cfg.chainID, cfg.athenaContract)
		if err != nil {
			return nil, nil, nil, err
		}
		chainIDs = append(chainIDs, cfg.chainID)
		nodeWSURLs[cfg.chainID] = normalizedNodeWSURLs
		athenaContracts[cfg.chainID] = athenaContract
	}
	if len(chainIDs) == 0 {
		log.Info("token project validator has no enabled chains")
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
	nodeWSURLs     []string
	athenaContract string
	enabled        bool
}

func (w *Worker) chainConfigs() []chainConfig {
	return []chainConfig{
		{
			chainID:        common.ChainIDEthereumMainnet,
			nodeWSURLs:     w.opts.EthNodeWSURLs,
			athenaContract: w.opts.EthAthenaContract,
			enabled:        w.opts.EthEnabled,
		},
		{
			chainID:        common.ChainIDBSCMainnet,
			nodeWSURLs:     w.opts.BSCNodeWSURLs,
			athenaContract: w.opts.BSCAthenaContract,
			enabled:        w.opts.BSCEnabled,
		},
	}
}
