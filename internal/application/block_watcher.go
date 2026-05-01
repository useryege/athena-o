package application

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
	log "github.com/sirupsen/logrus"
)

type Watcher struct {
	nodeClient *ethclient.Client
	outputCh   chan<- *Project
	wg         sync.WaitGroup
}

func NewWatcher(nodeClient *ethclient.Client, outputCh chan<- *Project) *Watcher {
	return &Watcher{
		nodeClient: nodeClient,
		outputCh:   outputCh,
	}
}

func (w *Watcher) Start(ctx context.Context) error {
	w.wg.Add(1)
	go func() {
		defer w.wg.Done()
		err := w.TestChainWatcher(ctx, 24990983, 0)
		if err != nil {
			log.Errorf("failed to test chain watcher: %v", err)
		}
	}()
	return nil
}

func (s *Watcher) TestChainWatcher(ctx context.Context, startBlock uint64, endBlock uint64) error {

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

	// if endBlock is less than startBlock, return error
	if endBlock < startBlock {
		return errors.New("endBlock must be greater than startBlock")
	}

	// scan blocks from startBlock to endBlock
	for blockNumber := startBlock; blockNumber <= endBlock; blockNumber++ {
		block, err := s.nodeClient.BlockByNumber(ctx, big.NewInt(int64(blockNumber)))
		if err != nil {
			return fmt.Errorf("failed to get block %d: %w", blockNumber, err)
		}
		blockDiscoveredAt := time.Now()

		for _, tx := range block.Transactions() {
			if tx.To() == nil {
				event := &Project{
					PerfTrace: &PerfTrace{
						BlockDiscoveredAt: blockDiscoveredAt,
						TxDiscoveredAt:    time.Now(),
					},
					BlockTime:   block.Time(),
					BlockNumber: blockNumber,
					Tx:          tx,
				}
				s.outputCh <- event
				log.WithFields(log.Fields{
					"component":         "Block Watcher",
					"blockNumber":       blockNumber,
					"blockTime":         block.Time(),
					"transaction":       tx.Hash(),
					"executionDuration": event.PerfTrace.TxDiscoveredAt.Sub(event.PerfTrace.BlockDiscoveredAt).Milliseconds(),
				}).Info("block watcher discovered contract creation transaction")
			}
		}
	}
	return nil
}

func (w *Watcher) Stop() error {
	w.wg.Wait()
	return nil
}
