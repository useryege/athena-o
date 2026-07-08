package projectdatacollector

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	ethcommon "github.com/ethereum/go-ethereum/common"
	log "github.com/sirupsen/logrus"
	"github.com/useryege/athena/common"
	ethereumapiapiclient "github.com/useryege/athena/internal/ethereumapi/apiclient"
	tokenstore "github.com/useryege/athena/internal/token/store"
	"github.com/useryege/athena/util/ave"
	"github.com/useryege/athena/util/ethws"
	utilio "github.com/useryege/athena/util/io"
)

const (
	pollInterval            = 1 * time.Second
	aveTaskLimit            = int32(20)
	aveFetchConcurrency     = 5
	contractCodeSourceLimit = int32(20)
	chainStateTaskLimit     = int32(20)
	walletAssetTaskLimit    = int32(20)
	simulationTaskLimit     = int32(20)
	aveHTTPClientTimeout    = 60 * time.Second
)

type Options struct {
	Store              *tokenstore.SQLStore
	EthNodeWSURLs      []string
	BSCNodeWSURLs      []string
	EthAthenaContract  string
	BSCAthenaContract  string
	EthEnabled         bool
	BSCEnabled         bool
	NodeWSUseProxy     bool
	AveAPIKey          string
	AveAPIBaseURL      string
	EthereumAPIAddress string
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
	chainIDs, nodeWSURLs, athenaContracts, err := w.enabledChainRuntime()
	if err != nil {
		return err
	}
	var aveClient ave.Client
	var ethereumAPI ethereumapiapiclient.EthereumAPIServiceClient
	var ethereumAPIConn utilio.Closer
	if len(chainIDs) > 0 {
		if strings.TrimSpace(w.opts.AveAPIKey) == "" {
			return errAveAPIKeyRequired()
		}
		ethereumAPIAddress := strings.TrimSpace(w.opts.EthereumAPIAddress)
		if ethereumAPIAddress == "" {
			return errEthereumAPIServerAddressRequired()
		}
		aveClient, err = ave.NewClient(ave.Config{
			BaseURL: w.opts.AveAPIBaseURL,
			APIKey:  w.opts.AveAPIKey,
			Timeout: aveHTTPClientTimeout,
		})
		if err != nil {
			return err
		}
		ethereumAPIConn, ethereumAPI, err = ethereumapiapiclient.NewEthereumAPIClientset(ethereumAPIAddress).NewEthereumAPIServiceClient()
		if err != nil {
			return err
		}
	}
	runCtx, cancel := context.WithCancel(ctx)
	runner := newDataCollectorRunner(dataCollectorRunnerOptions{
		store:           w.opts.Store,
		chainIDs:        chainIDs,
		nodeWSURLs:      nodeWSURLs,
		athenaContracts: athenaContracts,
		nodeWSUseProxy:  w.opts.NodeWSUseProxy,
		aveClient:       aveClient,
		ethereumAPI:     ethereumAPI,
		ethereumAPIConn: ethereumAPIConn,
		pollInterval:    pollInterval,
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

func (w *Worker) run(ctx context.Context, runner *dataCollectorRunner) {
	defer close(w.done)
	runner.run(ctx)
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
			}).Info("token project data collector chain disabled by runtime config")
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
		log.Info("token project data collector has no enabled chains")
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
