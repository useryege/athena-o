package chainwatcher

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/ethclient"
	log "github.com/sirupsen/logrus"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/klog/v2"
)

type Server struct {
	nodeClient *ethclient.Client

	started bool
	stopped bool
	cancel  context.CancelFunc
	doneCh  chan struct{}
	mu      sync.Mutex
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
				log.WithFields(log.Fields{
					"blockNumber": blockNumber,
					"transaction": tx.Hash(),
				}).Info("transaction is a contract creation")
			}
		}
	}
	return nil
}

func (s *Server) RunWithContext(ctx context.Context) {
	// recover from panic and log the error using the configured logger instead of the default.
	defer utilruntime.HandleCrashWithContext(ctx)
	logger := klog.FromContext(ctx)

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
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.started
}

func (s *Server) Start() error {
	s.mu.Lock()
	if s.started {
		s.mu.Unlock()
		return nil
	}

	runCtx, cancel := context.WithCancel(context.Background())
	doneCh := make(chan struct{})
	s.cancel = cancel
	s.doneCh = doneCh
	s.started = true
	s.stopped = false
	s.mu.Unlock()

	go func() {
		defer close(doneCh)
		defer func() {
			s.mu.Lock()
			defer s.mu.Unlock()
			s.started = false
			s.stopped = true
			s.cancel = nil
		}()
		s.RunWithContext(runCtx)
	}()

	return nil
}

func (s *Server) Stop() error {
	s.mu.Lock()
	if !s.started {
		s.mu.Unlock()
		return nil
	}

	cancel := s.cancel
	doneCh := s.doneCh
	s.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	if doneCh != nil {
		<-doneCh
	}

	return nil
}
