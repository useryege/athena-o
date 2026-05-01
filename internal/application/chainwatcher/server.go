package chainwatcher

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/ethclient"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"
)

type Server struct {
	nodeClient  *ethclient.Client
	started     bool
	stopped     bool
	startedLock sync.Mutex
}

func NewServer(nodeClient *ethclient.Client) *Server {
	return &Server{nodeClient: nodeClient}
}

func (s *Server) TestChainWatcher(ctx context.Context, startBlock uint64, endBlock uint64) error {
	// if endBlock is less than startBlock, return error
	if endBlock < startBlock {
		return errors.New("endBlock must be greater than startBlock")
	}

	// get latest block number
	getlatestblock := func() (uint64, error) {
		latestBlock, err := s.nodeClient.BlockNumber(ctx)
		if err != nil {
			return 0, err
		}
		return latestBlock, nil
	}

	// if startBlock is 0, get the latest block number and set it to latest - 10000 as default
	if startBlock == 0 {
		latestBlock, err := getlatestblock()
		if err != nil {
			return fmt.Errorf("failed to get latest block number: %w", err)
		}
		startBlock = latestBlock - 10000
	}

	// if endBlock is 0, get the latest block number and set it to endBlock
	if endBlock == 0 {
		latestBlock, err := getlatestblock()
		if err != nil {
			return err
		}
		endBlock = latestBlock
	}

	// scan blocks from startBlock to endBlock
	for blockNumber := startBlock; blockNumber <= endBlock; blockNumber++ {
		block, err := s.nodeClient.BlockByNumber(ctx, big.NewInt(int64(blockNumber)))
		if err != nil {
			return fmt.Errorf("failed to get block %d: %w", blockNumber, err)
		}
		for _, tx := range block.Transactions() {
			if tx.To() == nil {
				fmt.Println("Transaction:", tx.Hash())
			}
		}
	}
	return nil
}

func (s *Server) Run(stopCh <-chan struct{}) {
	s.RunWithContext(wait.ContextForChannel(stopCh))
}

func (s *Server) RunWithContext(ctx context.Context) {
	// recover from panic and log the error using the configured logger instead of the default.
	defer utilruntime.HandleCrashWithContext(ctx)
	logger := klog.FromContext(ctx)

	// check if the chainwatcher has started
	if s.HasStarted() {
		logger.Info("Warning: the chainwatcher has started, run more than once is not allowed")
		return
	}

	// maintain the started and stopped status
	func() {
		s.startedLock.Lock()
		defer s.startedLock.Unlock()
		s.started = true
	}()
	defer func() {
		s.startedLock.Lock()
		defer s.startedLock.Unlock()
		s.stopped = true
	}()

	// main logic
	if s.nodeClient == nil {
		logger.Info("chainwatcher node client is nil, skip TestChainWatcher")
		return
	}

	if err := s.TestChainWatcher(ctx, 0, 0); err != nil {
		logger.Error(err, "chainwatcher TestChainWatcher failed")
	}
}

func (s *Server) HasStarted() bool {
	s.startedLock.Lock()
	defer s.startedLock.Unlock()
	return s.started
}

func (s *Server) Start() error {
	return nil
}
func (s *Server) Stop() error {
	return nil
}
