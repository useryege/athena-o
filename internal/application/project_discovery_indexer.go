package application

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"time"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	log "github.com/sirupsen/logrus"
	appstore "github.com/useryege/athena/internal/application/store"
)

const initialProjectSyncLookback = 30 * 24 * time.Hour
const defaultBlockHeaderQueueCapacity = 16

var pancakeV2SwapTopicHash = common.HexToHash("0xd78ad95fa46c994b6551d0da85fc275fe613ce37657fb8d5e3d130840159d822")

type projectDiscoveryNodeClient interface {
	SubscribeNewHead(ctx context.Context, ch chan<- *types.Header) (ethereum.Subscription, error)
	BlockNumber(ctx context.Context) (uint64, error)
	BlockByNumber(ctx context.Context, number *big.Int) (*types.Block, error)
	TransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error)
	FilterLogs(ctx context.Context, q ethereum.FilterQuery) ([]types.Log, error)
	ChainID(ctx context.Context) (*big.Int, error)
}

type projectDiscoveryIndexerImpl struct {
	nodeClient   projectDiscoveryNodeClient
	projectCache ProjectSnapshotCache
	projectStore appstore.ProjectStore
	intake       DiscoveryIntake
	wg           sync.WaitGroup

	chainID *big.Int
}

type discoveryIntakeImpl struct {
	reconciler ProjectStateReconciler
}

func NewProjectDiscoveryIndexer(
	nodeClient *ethclient.Client,
	projectCache ProjectSnapshotCache,
	projectStore appstore.ProjectStore,
	intake DiscoveryIntake,
) (ProjectDiscoveryIndexer, error) {
	chainID, err := nodeClient.ChainID(context.Background())
	if err != nil {
		return nil, err
	}

	return &projectDiscoveryIndexerImpl{
		nodeClient:   nodeClient,
		projectCache: projectCache,
		projectStore: projectStore,
		intake:       intake,
		chainID:      chainID,
	}, nil
}

func NewDiscoveryIntake(
	reconciler ProjectStateReconciler,
) DiscoveryIntake {
	return &discoveryIntakeImpl{
		reconciler: reconciler,
	}
}

func (w *projectDiscoveryIndexerImpl) Start(ctx context.Context) error {
	log.Info("starting project discovery indexer")
	w.wg.Add(1)
	go func() {
		defer w.wg.Done()
		err := w.run(ctx)
		if err != nil && !errors.Is(err, context.Canceled) {
			log.Errorf("failed to run project discovery indexer: %v", err)
		}
	}()
	return nil
}

func (w *projectDiscoveryIndexerImpl) Stop() error {
	w.wg.Wait()
	return nil
}

func (w *projectDiscoveryIndexerImpl) getLatestBlock(ctx context.Context) (uint64, error) {
	latestBlock, err := w.nodeClient.BlockNumber(ctx)
	if err != nil {
		return 0, err
	}
	return latestBlock, nil
}

func (w *projectDiscoveryIndexerImpl) run(ctx context.Context) error {
	cursor, err := w.loadCursor(ctx)
	if err != nil {
		return err
	}
	log.WithField("cursor", cursor).Info("initialized project discovery cursor")
	if err := w.catchUpToLatest(ctx, &cursor); err != nil {
		return err
	}
	return w.followHeads(ctx, &cursor)
}

func (w *projectDiscoveryIndexerImpl) loadCursor(ctx context.Context) (uint64, error) {
	if w.projectStore != nil {
		maxBlock, ok, err := w.projectStore.GetMaxProjectBlockNumber(ctx)
		if err != nil {
			return 0, err
		}
		if ok {
			cursor := resumeCursorFromProjectBlock(maxBlock)
			log.WithFields(log.Fields{
				"projectBlockNumber": maxBlock,
				"cursor":             cursor,
			}).Info("initialized project discovery cursor from persisted projects")
			return cursor, nil
		}
	}

	latestBlock, err := w.nodeClient.BlockByNumber(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to initialize project discovery cursor: %w", err)
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
	}).Info("initialized project discovery cursor from lookback")
	return cursor, nil
}

func resumeCursorFromProjectBlock(blockNumber uint64) uint64 {
	if blockNumber == 0 {
		return 0
	}
	return blockNumber - 1
}

func (w *projectDiscoveryIndexerImpl) initialCursorFromLookback(ctx context.Context, latestBlock *types.Block) (uint64, uint64, error) {
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

func (w *projectDiscoveryIndexerImpl) findBlockByTimestamp(ctx context.Context, latestBlockNumber uint64, targetTimestamp uint64) (uint64, error) {
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

func (w *projectDiscoveryIndexerImpl) catchUpToLatest(ctx context.Context, cursor *uint64) error {
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

func (w *projectDiscoveryIndexerImpl) followHeads(ctx context.Context, cursor *uint64) error {
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
			if err := w.processBlock(ctx, cursor, header.Number.Uint64(), ProjectDiscoverySourceFollowHeads); err != nil {
				return err
			}
		}
	}
}

func (w *projectDiscoveryIndexerImpl) processBlock(ctx context.Context, cursor *uint64, blockNumber uint64, source ProjectDiscoverySource) error {
	if blockNumber <= *cursor {
		return nil
	}
	if err := w.scanBlock(ctx, blockNumber, source); err != nil {
		return err
	}
	*cursor = blockNumber
	return nil
}

func (w *projectDiscoveryIndexerImpl) processNextBlock(ctx context.Context, cursor *uint64) error {
	return w.processBlock(ctx, cursor, *cursor+1, ProjectDiscoverySourceCatchUp)
}

func (w *projectDiscoveryIndexerImpl) scanBlock(ctx context.Context, blockNumber uint64, source ProjectDiscoverySource) error {
	errCh := make(chan error, 2)
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		if err := w.scanBlockProjectCreations(ctx, blockNumber, source); err != nil {
			errCh <- err
		}
	}()

	go func() {
		defer wg.Done()
		if err := w.scanBlockPairSwaps(ctx, blockNumber); err != nil {
			errCh <- err
		}
	}()

	wg.Wait()
	close(errCh)
	for err := range errCh {
		if err != nil {
			return err
		}
	}
	return nil
}

func (w *projectDiscoveryIndexerImpl) scanBlockProjectCreations(ctx context.Context, blockNumber uint64, source ProjectDiscoverySource) error {
	block, err := w.nodeClient.BlockByNumber(ctx, new(big.Int).SetUint64(blockNumber))
	if err != nil {
		return fmt.Errorf("failed to get block %d: %w", blockNumber, err)
	}

	candidates := w.discoverBlockCandidates(block, blockNumber, source)
	if len(candidates) == 0 {
		return nil
	}
	if w.intake == nil {
		return errors.New("discovery intake is not configured")
	}
	return w.intake.IntakeCandidates(ctx, candidates)
}

func (w *projectDiscoveryIndexerImpl) scanBlockPairSwaps(ctx context.Context, blockNumber uint64) error {
	swapPairAddresses, err := w.fetchPancakeV2SwapPairAddresses(ctx, blockNumber)
	if err != nil {
		return fmt.Errorf("failed to fetch pancake v2 swap logs for block %d: %w", blockNumber, err)
	}
	for _, pairAddress := range swapPairAddresses {
		log.WithFields(log.Fields{
			"blockNumber": blockNumber,
			"pairAddress": pairAddress.Hex(),
		}).Info("pancake v2 swap pair detected")
	}
	if err := w.scheduleProjectsBySwapPairs(ctx, blockNumber, swapPairAddresses); err != nil {
		return err
	}
	return nil
}

func (w *projectDiscoveryIndexerImpl) fetchPancakeV2SwapPairAddresses(ctx context.Context, blockNumber uint64) ([]common.Address, error) {
	block := new(big.Int).SetUint64(blockNumber)
	query := ethereum.FilterQuery{
		FromBlock: block,
		ToBlock:   block,
		Topics: [][]common.Hash{
			{pancakeV2SwapTopicHash},
		},
	}
	logs, err := w.nodeClient.FilterLogs(ctx, query)
	if err != nil {
		return nil, err
	}

	seen := make(map[common.Address]struct{}, len(logs))
	pairAddresses := make([]common.Address, 0, len(logs))
	for _, entry := range logs {
		if _, ok := seen[entry.Address]; ok {
			continue
		}
		seen[entry.Address] = struct{}{}
		pairAddresses = append(pairAddresses, entry.Address)
	}
	return pairAddresses, nil
}

func (w *projectDiscoveryIndexerImpl) scheduleProjectsBySwapPairs(ctx context.Context, blockNumber uint64, pairAddresses []common.Address) error {
	pairAddresses = uniqueProjectPairAddresses(pairAddresses)
	if len(pairAddresses) == 0 {
		return nil
	}
	projects, matchedPairs, err := w.projectsByCachedSwapPairs(ctx, pairAddresses)
	if err != nil {
		return fmt.Errorf("match swap pairs from cache: %w", err)
	}
	unmatchedPairs := unmatchedProjectPairAddresses(pairAddresses, matchedPairs)
	if len(unmatchedPairs) > 0 {
		dbProjects, err := w.projectsByStoredSwapPairs(ctx, unmatchedPairs)
		if err != nil {
			return fmt.Errorf("match swap pairs from store: %w", err)
		}
		projects = append(projects, dbProjects...)
	}

	candidates := discoveredCandidatesFromSwapProjects(blockNumber, projects)
	if len(candidates) == 0 {
		return nil
	}
	if w.intake == nil {
		return errors.New("discovery intake is not configured")
	}
	log.WithFields(log.Fields{
		"blockNumber":  blockNumber,
		"projectCount": len(candidates),
	}).Info("scheduling projects from pancake v2 swap pairs")
	return w.intake.ScheduleProjects(ctx, candidates)
}

func (w *projectDiscoveryIndexerImpl) projectsByCachedSwapPairs(ctx context.Context, pairAddresses []common.Address) ([]*Project, map[common.Address]struct{}, error) {
	matchedPairs := make(map[common.Address]struct{})
	if w.projectCache == nil {
		return nil, matchedPairs, nil
	}
	projects, err := w.projectCache.ListProjectsByPairAddresses(ctx, pairAddresses)
	if err != nil {
		return nil, nil, err
	}
	pairSet := addressSet(pairAddresses)
	for _, project := range projects {
		markProjectMatchedPairs(project, pairSet, matchedPairs)
	}
	return projects, matchedPairs, nil
}

func (w *projectDiscoveryIndexerImpl) projectsByStoredSwapPairs(ctx context.Context, pairAddresses []common.Address) ([]*Project, error) {
	if w.projectStore == nil {
		return nil, nil
	}
	metas, err := w.projectStore.ListProjectMetasByPairAddresses(ctx, pairAddresses)
	if err != nil {
		return nil, err
	}
	projects := make([]*Project, 0, len(metas))
	for _, meta := range metas {
		project := &Project{Meta: projectMetaFromStore(meta)}
		projects = append(projects, project)
		if w.projectCache != nil {
			if err := w.projectCache.SetProject(ctx, project); err != nil {
				return nil, err
			}
		}
	}
	return projects, nil
}

func discoveredCandidatesFromSwapProjects(blockNumber uint64, projects []*Project) []DiscoveredProjectCandidate {
	seen := make(map[common.Address]struct{}, len(projects))
	candidates := make([]DiscoveredProjectCandidate, 0, len(projects))
	for _, project := range projects {
		if project == nil || project.Meta.Contract == (common.Address{}) {
			continue
		}
		if _, ok := seen[project.Meta.Contract]; ok {
			continue
		}
		seen[project.Meta.Contract] = struct{}{}
		candidates = append(candidates, DiscoveredProjectCandidate{
			BlockTime:   project.Meta.BlockTime,
			BlockNumber: blockNumber,
			TxIndex:     project.Meta.TxIndex,
			TxHash:      project.Meta.TxHash,
			Contract:    project.Meta.Contract,
			Creator:     project.Meta.Creator,
			Source:      ProjectDiscoverySourcePairSwap,
		})
	}
	return candidates
}

func unmatchedProjectPairAddresses(pairAddresses []common.Address, matchedPairs map[common.Address]struct{}) []common.Address {
	unmatched := make([]common.Address, 0, len(pairAddresses))
	for _, pair := range pairAddresses {
		if _, ok := matchedPairs[pair]; ok {
			continue
		}
		unmatched = append(unmatched, pair)
	}
	return unmatched
}

func markProjectMatchedPairs(project *Project, pairSet map[common.Address]struct{}, matchedPairs map[common.Address]struct{}) {
	if project == nil {
		return
	}
	for _, pair := range projectPairAddresses(project) {
		if _, ok := pairSet[pair]; ok {
			matchedPairs[pair] = struct{}{}
		}
	}
}

func addressSet(items []common.Address) map[common.Address]struct{} {
	result := make(map[common.Address]struct{}, len(items))
	for _, item := range items {
		result[item] = struct{}{}
	}
	return result
}

func (w *projectDiscoveryIndexerImpl) discoverBlockCandidates(block *types.Block, blockNumber uint64, source ProjectDiscoverySource) []DiscoveredProjectCandidate {
	if block == nil {
		return nil
	}
	candidates := make([]DiscoveredProjectCandidate, 0)
	for txIndex, tx := range block.Transactions() {
		if tx.To() != nil {
			continue
		}
		from, err := types.Sender(types.LatestSignerForChainID(w.chainID), tx)
		if err != nil {
			continue
		}
		contractAddress := crypto.CreateAddress(from, tx.Nonce())
		candidates = append(candidates, DiscoveredProjectCandidate{
			BlockTime:   block.Time(),
			BlockNumber: blockNumber,
			TxIndex:     uint64(txIndex),
			Tx:          tx,
			TxHash:      tx.Hash(),
			Contract:    contractAddress,
			Creator:     from,
			Source:      source,
		})
	}
	return candidates
}

func (d *discoveryIntakeImpl) IntakeCandidates(ctx context.Context, items []DiscoveredProjectCandidate) error {
	if len(items) == 0 {
		return nil
	}
	if d == nil || d.reconciler == nil {
		return nil
	}
	return d.reconciler.InitProject(ctx, items)
}

func (d *discoveryIntakeImpl) ScheduleProjects(ctx context.Context, items []DiscoveredProjectCandidate) error {
	if len(items) == 0 {
		return nil
	}
	if d == nil || d.reconciler == nil {
		return nil
	}
	return d.reconciler.ScheduleProjects(ctx, items)
}
