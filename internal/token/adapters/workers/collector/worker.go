package collector

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
	"github.com/useryege/athena/internal/token/research"
	"github.com/useryege/athena/internal/token/telemetry"
	"github.com/useryege/athena/util/ethws"
)

const (
	pollInterval            = 1 * time.Second
	aveTaskLimit            = int32(20)
	aveFetchConcurrency     = 5
	contractCodeSourceLimit = int32(20)
	chainStateTaskLimit     = int32(20)
	walletAssetTaskLimit    = int32(20)
	simulationTaskLimit     = int32(20)
)

type Options struct {
	Store      research.CollectorRepository
	Chains     []chainregistry.Chain
	Clients    *evm.ChainClientRegistry
	DataType   research.DataCollectionType
	MarketData research.MarketDataProvider
	SourceCode research.SourceCodeProvider
	Telemetry  telemetry.Reporter
}

type Worker struct {
	opts Options

	startStopMu sync.Mutex
	cancel      context.CancelFunc
	done        chan struct{}
	runner      *dataCollectorRunner
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
	if _, ok := research.ParseDataCollectionType(string(w.opts.DataType)); !ok {
		return fmt.Errorf("token data collector data type %q is invalid", w.opts.DataType)
	}
	requireEVM := w.opts.DataType == research.DataCollectionTypeChainState || w.opts.DataType == research.DataCollectionTypeWalletAssetState || w.opts.DataType == research.DataCollectionTypeSimulationResult
	chainIDs, _, athenaContracts, _, err := w.enabledChainRuntime(requireEVM)
	if err != nil {
		return err
	}
	if requireEVM && w.opts.Clients == nil {
		return fmt.Errorf("token data collector EVM client registry is required for %s", w.opts.DataType)
	}
	if len(chainIDs) > 0 && w.opts.DataType == research.DataCollectionTypeAve && w.opts.MarketData == nil {
		return errAveAPIKeyRequired()
	}
	if len(chainIDs) > 0 && w.opts.DataType == research.DataCollectionTypeContractCodeSource && w.opts.SourceCode == nil {
		return errEthereumAPIServerAddressRequired()
	}
	runCtx, cancel := context.WithCancel(ctx)
	runner := newDataCollectorRunner(dataCollectorRunnerOptions{
		store:           w.opts.Store,
		clients:         w.opts.Clients,
		dataType:        w.opts.DataType,
		chainIDs:        chainIDs,
		athenaContracts: athenaContracts,
		marketData:      w.opts.MarketData,
		sourceCode:      w.opts.SourceCode,
		pollInterval:    pollInterval,
		telemetry:       w.opts.Telemetry,
	})
	if len(chainIDs) > 0 {
		telemetry.Register(w.opts.Telemetry, telemetry.Scope{Component: "data_collector", DataType: string(w.opts.DataType)})
	}
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
	w.cancel = nil
	w.done = nil
	w.runner = nil
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

func (w *Worker) run(ctx context.Context, runner *dataCollectorRunner) {
	defer close(w.done)
	runner.run(ctx)
}

func (w *Worker) enabledChainRuntime(requireEVM bool) ([]int64, map[int64][]string, map[int64]ethcommon.Address, map[int64]bool, error) {
	chainIDs := make([]int64, 0, len(w.opts.Chains))
	nodeWSURLs := make(map[int64][]string)
	athenaContracts := make(map[int64]ethcommon.Address)
	useProxy := make(map[int64]bool)
	for _, cfg := range w.opts.Chains {
		if !cfg.Enabled {
			log.WithFields(log.Fields{
				"chain_id":   cfg.ID,
				"chain_name": cfg.Name,
			}).Info("token project data collector chain disabled by runtime config")
			continue
		}
		var normalizedNodeWSURLs []string
		var athenaContract ethcommon.Address
		if requireEVM {
			normalizedNodeWSURLs = ethws.NormalizeEndpoints(cfg.NodeWSURLs)
			if len(normalizedNodeWSURLs) == 0 {
				return nil, nil, nil, nil, errNodeWSURLRequired(cfg.ID)
			}
			var err error
			athenaContract, err = parseAthenaContract(cfg.ID, cfg.AthenaContract)
			if err != nil {
				return nil, nil, nil, nil, err
			}
		}
		chainIDs = append(chainIDs, cfg.ID)
		nodeWSURLs[cfg.ID] = normalizedNodeWSURLs
		athenaContracts[cfg.ID] = athenaContract
		useProxy[cfg.ID] = cfg.UseProxy
	}
	if len(chainIDs) == 0 {
		log.Info("token project data collector has no enabled chains")
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

func parseAddresses(name string, values []string) ([]ethcommon.Address, error) {
	addresses := make([]ethcommon.Address, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if !ethcommon.IsHexAddress(value) {
			return nil, fmt.Errorf("token project data collector %s address %q is invalid", name, value)
		}
		addresses = append(addresses, ethcommon.HexToAddress(value))
	}
	return addresses, nil
}
