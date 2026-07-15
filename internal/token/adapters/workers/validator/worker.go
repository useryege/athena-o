package validator

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	ethcommon "github.com/ethereum/go-ethereum/common"
	log "github.com/sirupsen/logrus"
	"github.com/useryege/athena/internal/token/adapters/evm"
	"github.com/useryege/athena/internal/token/chainregistry"
	"github.com/useryege/athena/internal/token/discovery"
	"github.com/useryege/athena/internal/token/telemetry"
	"github.com/useryege/athena/util/ethws"
)

const (
	pollInterval         = 3 * time.Second
	candidateLimit       = int32(100)
	candidateConcurrency = 10
	candidateLease       = 5 * time.Minute
	candidateLeaseRenew  = time.Minute
)

type Options struct {
	Store     discovery.ValidatorRepository
	Chains    []chainregistry.Chain
	Clients   *evm.ChainClientRegistry
	Telemetry telemetry.Reporter
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
	if w.opts.Clients == nil {
		return fmt.Errorf("token project validator EVM client registry is required")
	}
	chainIDs, _, athenaContracts, _, err := w.enabledChainRuntime()
	if err != nil {
		return err
	}

	runCtx, cancel := context.WithCancel(ctx)
	runners := make(map[int64]*validatorRunner, len(chainIDs))
	for _, chainID := range chainIDs {
		telemetry.Register(w.opts.Telemetry, telemetry.Scope{Component: "project_validator", ChainID: chainID})
		runners[chainID] = newValidatorRunner(validatorRunnerOptions{
			store:                w.opts.Store,
			clients:              w.opts.Clients,
			chainIDs:             []int64{chainID},
			athenaContracts:      map[int64]ethcommon.Address{chainID: athenaContracts[chainID]},
			pollInterval:         pollInterval,
			candidateLimit:       candidateLimit,
			candidateConcurrency: candidateConcurrency,
			telemetry:            w.opts.Telemetry,
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

func (w *Worker) enabledChainRuntime() ([]int64, map[int64][]string, map[int64]ethcommon.Address, map[int64]bool, error) {
	chainIDs := make([]int64, 0, len(w.opts.Chains))
	nodeWSURLs := make(map[int64][]string)
	athenaContracts := make(map[int64]ethcommon.Address)
	useProxy := make(map[int64]bool)
	for _, cfg := range w.opts.Chains {
		if !cfg.Enabled {
			log.WithFields(log.Fields{
				"chain_id":   cfg.ID,
				"chain_name": cfg.Name,
			}).Info("token project validator chain disabled by runtime config")
			continue
		}
		normalizedNodeWSURLs := ethws.NormalizeEndpoints(cfg.NodeWSURLs)
		if len(normalizedNodeWSURLs) == 0 {
			return nil, nil, nil, nil, errNodeWSURLRequired(cfg.ID)
		}
		athenaContract, err := parseAthenaContract(cfg.ID, cfg.AthenaContract)
		if err != nil {
			return nil, nil, nil, nil, err
		}
		chainIDs = append(chainIDs, cfg.ID)
		nodeWSURLs[cfg.ID] = normalizedNodeWSURLs
		athenaContracts[cfg.ID] = athenaContract
		useProxy[cfg.ID] = cfg.UseProxy
	}
	if len(chainIDs) == 0 {
		log.Info("token project validator has no enabled chains")
	}
	return chainIDs, nodeWSURLs, athenaContracts, useProxy, nil
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
