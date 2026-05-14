package application

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/useryege/athena/internal/application/evm"
)

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
		err := w.run(ctx, 97782632, 0)
		if err != nil && !errors.Is(err, context.Canceled) {
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
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		block, err := w.nodeClient.BlockByNumber(ctx, big.NewInt(int64(blockNumber)))
		if err != nil {
			return fmt.Errorf("failed to get block %d: %w", blockNumber, err)
		}

		projects := make([]*Project, 0)
		contracts := make([]common.Address, 0)
		for txIndex, tx := range block.Transactions() {
			if tx.To() == nil {
				// query sender from transaction
				from, err := types.Sender(types.LatestSignerForChainID(w.chainID), tx)
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
						TxIndex:     uint64(txIndex),
						Tx:          tx,
						Contract:    contractAddress,
						Creator:     from,
					},
				}
				projects = append(projects, event)
				contracts = append(contracts, contractAddress)
			}
		}
		if err := w.syncProjects(ctx, projects, contracts); err != nil {
			return fmt.Errorf("failed to sync projects for block %d: %w", blockNumber, err)
		}
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
			log.WithFields(log.Fields{
				"projectID": project.Meta.ProjectID,
				"error":     err,
			}).Error("failed to store project")
			continue
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
