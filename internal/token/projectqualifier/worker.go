package projectqualifier

import (
	"context"
	"strings"
	"sync"
	"time"

	ethcommon "github.com/ethereum/go-ethereum/common"
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
	if strings.TrimSpace(w.opts.EthNodeWSURL) == "" {
		return errNodeWSURLRequired(common.ChainIDEthereumMainnet)
	}
	if strings.TrimSpace(w.opts.BSCNodeWSURL) == "" {
		return errNodeWSURLRequired(common.ChainIDBSCMainnet)
	}
	athenaContracts, err := w.athenaContracts()
	if err != nil {
		return err
	}

	runCtx, cancel := context.WithCancel(ctx)
	runner := newQualifierRunner(qualifierRunnerOptions{
		store:           w.opts.Store,
		nodeWSURLs:      w.nodeWSURLs(),
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

func (w *Worker) nodeWSURLs() map[int64]string {
	return map[int64]string{
		common.ChainIDEthereumMainnet: w.opts.EthNodeWSURL,
		common.ChainIDBSCMainnet:      w.opts.BSCNodeWSURL,
	}
}

func (w *Worker) athenaContracts() (map[int64]ethcommon.Address, error) {
	ethContract, err := parseAthenaContract(common.ChainIDEthereumMainnet, w.opts.EthAthenaContract)
	if err != nil {
		return nil, err
	}
	bscContract, err := parseAthenaContract(common.ChainIDBSCMainnet, w.opts.BSCAthenaContract)
	if err != nil {
		return nil, err
	}
	return map[int64]ethcommon.Address{
		common.ChainIDEthereumMainnet: ethContract,
		common.ChainIDBSCMainnet:      bscContract,
	}, nil
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
