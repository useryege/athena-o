package application

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)

type BlockWatcher struct {
	nodeClient *ethclient.Client
	outputCh   chan<- *Project
	wg         sync.WaitGroup
}

func NewBlockWatcher(nodeClient *ethclient.Client, outputCh chan<- *Project) *BlockWatcher {
	return &BlockWatcher{
		nodeClient: nodeClient,
		outputCh:   outputCh,
	}
}

func (w *BlockWatcher) Start(ctx context.Context) error {
	w.wg.Add(1)
	go func() {
		defer w.wg.Done()
		err := w.run(ctx, 24990983, 0)
		if err != nil {
			log.Errorf("failed to test chain watcher: %v", err)
		}
	}()
	return nil
}

func (w *BlockWatcher) getLatestBlock(ctx context.Context) (uint64, error) {
	latestBlock, err := w.nodeClient.BlockNumber(ctx)
	if err != nil {
		return 0, err
	}
	return latestBlock, nil
}

func (w *BlockWatcher) run(ctx context.Context, startBlock uint64, endBlock uint64) error {
	// if startBlock is 0, get the latest block number and set it to latest - 10000 as default
	if startBlock == 0 {
		latestBlock, err := w.getLatestBlock(ctx)
		if err != nil {
			return fmt.Errorf("failed to get latest block number: %w", err)
		}
		startBlock = latestBlock - 10000
	}

	// if endBlock is 0, get the latest block number and set it to endBlock
	if endBlock == 0 {
		latestBlock, err := w.getLatestBlock(ctx)
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
		block, err := w.nodeClient.BlockByNumber(ctx, big.NewInt(int64(blockNumber)))
		if err != nil {
			return fmt.Errorf("failed to get block %d: %w", blockNumber, err)
		}
		blockDiscoveredAt := time.Now()

		for _, tx := range block.Transactions() {
			if tx.To() == nil {
				// query sender from transaction
				from, err := types.Sender(types.LatestSignerForChainID(tx.ChainId()), tx)
				if err != nil {
					continue
				}
				// calculate contract address
				contractAddress := crypto.CreateAddress(from, tx.Nonce())

				event := &Project{
					Meta: ProjectMeta{
						ProjectID:   uuid.New(),
						BlockTime:   block.Time(),
						BlockNumber: blockNumber,
						Tx:          tx,
						Contract:    contractAddress,
						Creator:     from,
					},
					PerfTrace: PerfTrace{
						BlockDiscoveredAt: blockDiscoveredAt,
						TxDiscoveredAt:    time.Now(),
					},
				}
				w.outputCh <- event
			}
		}
	}
	return nil
}

func (w *BlockWatcher) Stop() error {
	w.wg.Wait()
	return nil
}
