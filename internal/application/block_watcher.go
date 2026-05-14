package application

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/useryege/athena/internal/application/evm"
)

const initialProjectSyncLookback = 30 * 24 * time.Hour

type BlockWatcher struct {
	nodeClient *ethclient.Client
	registry   ProjectRegistry
	fetcher    evm.AthenaFetcher
	scheduler  ProjectScheduler
	wg         sync.WaitGroup

	chainID *big.Int
}

func NewBlockWatcher(
	nodeClient *ethclient.Client,
	registry ProjectRegistry,
	fetcher evm.AthenaFetcher,
	scheduler ProjectScheduler,
) *BlockWatcher {
	chainID, err := nodeClient.ChainID(context.Background())
	if err != nil {
		panic(err)
	}

	return &BlockWatcher{
		nodeClient: nodeClient,
		registry:   registry,
		fetcher:    fetcher,
		scheduler:  scheduler,
		chainID:    chainID,
	}
}

func (w *BlockWatcher) Start(ctx context.Context) error {
	w.wg.Add(1)
	go func() {
		defer w.wg.Done()
		err := w.run(ctx)
		if err != nil && !errors.Is(err, context.Canceled) {
			log.Errorf("failed to run block watcher: %v", err)
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

func (w *BlockWatcher) run(ctx context.Context) error {
	cursor, err := w.loadCursor(ctx)
	if err != nil {
		return err
	}
	log.WithField("cursor", cursor).Info("initialized block watcher cursor")
	if err := w.catchUpToLatest(ctx, &cursor); err != nil {
		return err
	}
	return w.followHeads(ctx, &cursor)
}

func (w *BlockWatcher) loadCursor(ctx context.Context) (uint64, error) {
	maxBlock, ok, err := w.registry.MaxProjectBlockNumber(ctx)
	if err != nil {
		return 0, err
	}
	if ok {
		cursor := resumeCursorFromProjectBlock(maxBlock)
		log.WithFields(log.Fields{
			"projectBlockNumber": maxBlock,
			"cursor":             cursor,
		}).Info("initialized block watcher cursor from persisted projects")
		return cursor, nil
	}

	latestBlock, err := w.nodeClient.BlockByNumber(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to initialize block watcher cursor: %w", err)
	}
	cursor, startBlock, err := w.initialCursorFromLookback(ctx, latestBlock)
	if err != nil {
		return 0, err
	}
	log.WithFields(log.Fields{
		"latestBlockNumber": latestBlock.NumberU64(),
		"startBlockNumber":  startBlock,
		"cursor":            cursor,
		"lookback":          initialProjectSyncLookback.String(),
	}).Info("initialized block watcher cursor from lookback")
	return cursor, nil
}

func resumeCursorFromProjectBlock(blockNumber uint64) uint64 {
	if blockNumber == 0 {
		return 0
	}
	return blockNumber - 1
}

func (w *BlockWatcher) initialCursorFromLookback(ctx context.Context, latestBlock *types.Block) (uint64, uint64, error) {
	if latestBlock == nil {
		return 0, 0, errors.New("latest block is nil")
	}
	lookbackSeconds := uint64(initialProjectSyncLookback / time.Second)
	targetTimestamp := uint64(0)
	if latestBlock.Time() > lookbackSeconds {
		targetTimestamp = latestBlock.Time() - lookbackSeconds
	}

	startBlock, err := w.findBlockByTimestamp(ctx, latestBlock.NumberU64(), targetTimestamp)
	if err != nil {
		return 0, 0, err
	}
	return cursorBeforeBlock(startBlock), startBlock, nil
}

func (w *BlockWatcher) findBlockByTimestamp(ctx context.Context, latestBlockNumber uint64, targetTimestamp uint64) (uint64, error) {
	var result uint64
	low := uint64(0)
	high := latestBlockNumber
	for low <= high {
		if err := ctx.Err(); err != nil {
			return 0, err
		}
		mid := low + (high-low)/2
		block, err := w.nodeClient.BlockByNumber(ctx, new(big.Int).SetUint64(mid))
		if err != nil {
			return 0, fmt.Errorf("failed to get block %d while searching initial sync block: %w", mid, err)
		}
		if block.Time() <= targetTimestamp {
			result = mid
			low = mid + 1
			continue
		}
		if mid == 0 {
			break
		}
		high = mid - 1
	}
	return result, nil
}

func cursorBeforeBlock(blockNumber uint64) uint64 {
	if blockNumber == 0 {
		return 0
	}
	return blockNumber - 1
}

func (w *BlockWatcher) catchUpToLatest(ctx context.Context, cursor *uint64) error {
	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		latestBlock, err := w.getLatestBlock(ctx)
		if err != nil {
			return fmt.Errorf("failed to get latest block number: %w", err)
		}
		if *cursor >= latestBlock {
			return nil
		}
		if err := w.processNextBlock(ctx, cursor); err != nil {
			return err
		}
		log.WithField("cursor", *cursor).Info("caught up to latest block")
	}
}

func (w *BlockWatcher) followHeads(ctx context.Context, cursor *uint64) error {
	headers := make(chan *types.Header, defaultBlockHeaderQueueCapacity)
	subscription, err := w.nodeClient.SubscribeNewHead(ctx, headers)
	if err != nil {
		return err
	}
	defer subscription.Unsubscribe()

	go func() {
		if err := w.catchUpToLatest(ctx, cursor); err != nil {
			log.WithError(err).Error("failed to catch up to latest block")
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-subscription.Err():
			if err != nil {
				return err
			}
			return nil
		case header := <-headers:
			if header == nil || header.Number == nil {
				continue
			}
			if err := w.scanBlock(ctx, header.Number.Uint64()); err != nil {
				return err
			}
			log.WithField("cursor", *cursor).Info("scanned block")
		}
	}
}

func (w *BlockWatcher) processBlock(ctx context.Context, cursor *uint64, blockNumber uint64) error {
	if blockNumber <= *cursor {
		return nil
	}
	if err := w.scanBlock(ctx, blockNumber); err != nil {
		return err
	}
	*cursor = blockNumber
	return nil
}

func (w *BlockWatcher) processNextBlock(ctx context.Context, cursor *uint64) error {
	return w.processBlock(ctx, cursor, *cursor+1)
}

func (w *BlockWatcher) scanBlock(ctx context.Context, blockNumber uint64) error {
	block, err := w.nodeClient.BlockByNumber(ctx, new(big.Int).SetUint64(blockNumber))
	if err != nil {
		return fmt.Errorf("failed to get block %d: %w", blockNumber, err)
	}

	projects := make([]*Project, 0)
	contracts := make([]common.Address, 0)
	for txIndex, tx := range block.Transactions() {
		if tx.To() != nil {
			continue
		}
		from, err := types.Sender(types.LatestSignerForChainID(w.chainID), tx)
		if err != nil {
			continue
		}
		contractAddress := crypto.CreateAddress(from, tx.Nonce())

		project := &Project{
			Meta: ProjectMeta{
				ProjectID:   uuid.New(),
				BlockTime:   block.Time(),
				BlockNumber: blockNumber,
				TxIndex:     uint64(txIndex),
				Tx:          tx,
				TxHash:      tx.Hash(),
				Contract:    contractAddress,
				Creator:     from,
			},
		}
		projects = append(projects, project)
		contracts = append(contracts, contractAddress)
	}
	if err := w.syncProjects(ctx, projects, contracts); err != nil {
		return fmt.Errorf("failed to sync projects for block %d: %w", blockNumber, err)
	}
	return nil
}

func (w *BlockWatcher) syncProjects(ctx context.Context, projects []*Project, contracts []common.Address) error {
	if len(projects) == 0 {
		return nil
	}

	snapshots, err := w.fetcher.FetchProjects(ctx, contracts)
	if err != nil {
		return err
	}
	if len(snapshots) != len(projects) {
		return fmt.Errorf("athena list returned %d projects for %d token contracts", len(snapshots), len(projects))
	}

	for i, project := range projects {
		snapshot := snapshots[i]
		if snapshot.TokenContract != (common.Address{}) && snapshot.TokenContract != project.Meta.Contract {
			return fmt.Errorf("athena list result %d token contract = %s, want %s", i, snapshot.TokenContract, project.Meta.Contract)
		}
		if !snapshot.Token.IsValidERC20 {
			continue
		}

		project.ChainState = snapshot
		if err := w.registry.SetProject(ctx, project.Meta.ProjectID, project); err != nil {
			return fmt.Errorf("failed to store project %s: %w", project.Meta.ProjectID, err)
		}

		if w.scheduler != nil {
			if err := w.scheduler.EnqueueProject(ctx, project); err != nil {
				log.WithFields(log.Fields{
					"projectID": project.Meta.ProjectID,
					"error":     err,
				}).Error("failed to schedule project sync")
			}
		}
	}
	return nil
}

func (w *BlockWatcher) Stop() error {
	w.wg.Wait()
	return nil
}
