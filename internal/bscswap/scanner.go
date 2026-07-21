package bscswap

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"sort"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/jackc/pgx/v5"
	log "github.com/sirupsen/logrus"
	"github.com/useryege/athena/internal/bscswap/store"
	"golang.org/x/sync/errgroup"
)

const (
	defaultInitialLookback  = 30 * 24 * time.Hour
	ScanBatchSize           = 100
	DefaultFetchConcurrency = 16
	DefaultPollInterval     = time.Second
)

type ScannerConfig struct {
	FetchConcurrency int
	PollInterval     time.Duration
}

func DefaultScannerConfig() ScannerConfig {
	return ScannerConfig{FetchConcurrency: DefaultFetchConcurrency, PollInterval: DefaultPollInterval}
}

type Scanner struct {
	store   *store.Store
	node    *Node
	metrics *Metrics
	config  ScannerConfig
}

func NewScanner(repository *store.Store, node *Node, metrics *Metrics, config ScannerConfig) (*Scanner, error) {
	if repository == nil {
		return nil, fmt.Errorf("BSC swap store is required")
	}
	if node == nil {
		return nil, fmt.Errorf("BSC node is required")
	}
	if metrics == nil {
		return nil, fmt.Errorf("BSC swap metrics are required")
	}
	if config.FetchConcurrency <= 0 {
		return nil, fmt.Errorf("BSC swap fetch concurrency must be positive")
	}
	if config.PollInterval <= 0 {
		return nil, fmt.Errorf("BSC swap poll interval must be positive")
	}
	return &Scanner{store: repository, node: node, metrics: metrics, config: config}, nil
}

func (s *Scanner) Run(ctx context.Context) {
	for {
		processed, err := s.runOnce(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			s.metrics.RecordScanFailure()
			log.WithError(err).Error("BSC swap scanner iteration failed")
		}
		if processed && err == nil {
			continue
		}
		timer := time.NewTimer(s.config.PollInterval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}
}

func (s *Scanner) runOnce(ctx context.Context) (bool, error) {
	checkpoint, err := s.store.GetCheckpoint(ctx)
	if errors.Is(err, pgx.ErrNoRows) {
		checkpoint, err = s.initializeCheckpoint(ctx)
	}
	if err != nil {
		return false, err
	}
	finalized, err := s.node.FinalizedHeader(ctx)
	if err != nil {
		return false, err
	}
	finalizedNumber := finalized.Number.Uint64()
	s.metrics.RecordProgress(finalizedNumber, checkpoint.CursorBlockNumber)
	if checkpoint.CursorBlockNumber >= finalizedNumber {
		s.metrics.RecordScanSuccess(0, 0, 0)
		return false, nil
	}

	start := checkpoint.CursorBlockNumber + 1
	end := finalizedNumber
	if remaining := end - start + 1; remaining > ScanBatchSize {
		end = start + ScanBatchSize - 1
	}
	startedAt := time.Now()
	anchor, err := s.node.HeaderByNumber(ctx, start-1)
	if err != nil {
		return false, err
	}
	if anchor.Hash() != checkpoint.CursorBlockHash {
		return false, fmt.Errorf("BSC block %d hash does not match committed swap checkpoint", start-1)
	}
	logs, err := s.node.FilterSwapLogs(ctx, start, end)
	if err != nil {
		return false, err
	}
	candidates, blockNumbers, err := collectCandidates(logs, start, end)
	if err != nil {
		return false, err
	}
	blocks, err := s.fetchBlocks(ctx, blockNumbers)
	if err != nil {
		return false, err
	}
	transactions, err := swapTransactionsFromCandidates(candidates, blocks)
	if err != nil {
		return false, err
	}
	endHeader, err := s.node.HeaderByNumber(ctx, end)
	if err != nil {
		return false, err
	}
	nextCheckpoint := store.Checkpoint{
		StartBlockNumber: checkpoint.StartBlockNumber, CursorBlockNumber: end,
		CursorBlockHash: endHeader.Hash(), CursorBlockTimestamp: endHeader.Time,
	}
	if err := s.store.CommitBatch(ctx, checkpoint.CursorBlockNumber, transactions, nextCheckpoint); err != nil {
		return false, err
	}
	blockCount := end - start + 1
	s.metrics.RecordProgress(finalizedNumber, nextCheckpoint.CursorBlockNumber)
	s.metrics.RecordScanSuccess(blockCount, uint64(len(logs)), uint64(len(transactions)))
	log.WithFields(log.Fields{
		"start_block": start, "end_block": end, "block_count": blockCount,
		"matching_log_count": len(logs), "transaction_count": len(transactions),
		"duration_ms": time.Since(startedAt).Milliseconds(),
	}).Info("BSC swap scan batch completed")
	return true, nil
}

func (s *Scanner) initializeCheckpoint(ctx context.Context) (store.Checkpoint, error) {
	finalized, err := s.node.FinalizedHeader(ctx)
	if err != nil {
		return store.Checkpoint{}, err
	}
	finalizedNumber := finalized.Number.Uint64()
	if finalizedNumber == 0 {
		return store.Checkpoint{}, fmt.Errorf("finalized BSC height must be positive")
	}
	lookbackSeconds := uint64(defaultInitialLookback / time.Second)
	targetTimestamp := uint64(0)
	if finalized.Time > lookbackSeconds {
		targetTimestamp = finalized.Time - lookbackSeconds
	}
	startBlock, err := s.findFirstBlockAtOrAfter(ctx, finalizedNumber, targetTimestamp)
	if err != nil {
		return store.Checkpoint{}, err
	}
	if startBlock == 0 {
		startBlock = 1
	}
	previous, err := s.node.HeaderByNumber(ctx, startBlock-1)
	if err != nil {
		return store.Checkpoint{}, err
	}
	checkpoint, err := s.store.InitializeCheckpoint(ctx, store.Checkpoint{
		StartBlockNumber: startBlock, CursorBlockNumber: startBlock - 1,
		CursorBlockHash: previous.Hash(), CursorBlockTimestamp: previous.Time,
	})
	if err != nil {
		return store.Checkpoint{}, err
	}
	s.metrics.RecordProgress(finalizedNumber, checkpoint.CursorBlockNumber)
	log.WithFields(log.Fields{
		"finalized_block": finalizedNumber, "start_block": checkpoint.StartBlockNumber,
		"target_timestamp": targetTimestamp,
	}).Info("initialized BSC swap scan checkpoint")
	return checkpoint, nil
}

func (s *Scanner) findFirstBlockAtOrAfter(ctx context.Context, high, targetTimestamp uint64) (uint64, error) {
	low := uint64(0)
	for low < high {
		mid := low + (high-low)/2
		header, err := s.node.HeaderByNumber(ctx, mid)
		if err != nil {
			return 0, err
		}
		if header.Time < targetTimestamp {
			low = mid + 1
		} else {
			high = mid
		}
	}
	return low, nil
}

type swapCandidate struct {
	blockNumber      uint64
	blockHash        common.Hash
	transactionHash  common.Hash
	transactionIndex uint64
}

func collectCandidates(logs []types.Log, start, end uint64) ([]swapCandidate, []uint64, error) {
	candidatesByHash := make(map[common.Hash]swapCandidate)
	blockSet := make(map[uint64]struct{})
	for _, item := range logs {
		if item.Removed {
			return nil, nil, fmt.Errorf("BSC swap log %s:%d is marked removed", item.TxHash.Hex(), item.Index)
		}
		if item.BlockNumber < start || item.BlockNumber > end {
			return nil, nil, fmt.Errorf("BSC swap log block %d is outside requested range %d-%d", item.BlockNumber, start, end)
		}
		if len(item.Topics) == 0 || item.Topics[0] != V2SwapTopic {
			return nil, nil, fmt.Errorf("BSC swap log %s:%d has an unexpected topic", item.TxHash.Hex(), item.Index)
		}
		if item.BlockHash == (common.Hash{}) || item.TxHash == (common.Hash{}) {
			return nil, nil, fmt.Errorf("BSC swap log at block %d has an empty hash", item.BlockNumber)
		}
		candidate := swapCandidate{
			blockNumber: item.BlockNumber, blockHash: item.BlockHash,
			transactionHash: item.TxHash, transactionIndex: uint64(item.TxIndex),
		}
		if existing, ok := candidatesByHash[item.TxHash]; ok {
			if existing != candidate {
				return nil, nil, fmt.Errorf("BSC swap transaction %s has inconsistent log positions", item.TxHash.Hex())
			}
			continue
		}
		candidatesByHash[item.TxHash] = candidate
		blockSet[item.BlockNumber] = struct{}{}
	}

	candidates := make([]swapCandidate, 0, len(candidatesByHash))
	for _, candidate := range candidatesByHash {
		candidates = append(candidates, candidate)
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].blockNumber == candidates[j].blockNumber {
			return candidates[i].transactionIndex < candidates[j].transactionIndex
		}
		return candidates[i].blockNumber < candidates[j].blockNumber
	})
	blockNumbers := make([]uint64, 0, len(blockSet))
	for number := range blockSet {
		blockNumbers = append(blockNumbers, number)
	}
	sort.Slice(blockNumbers, func(i, j int) bool { return blockNumbers[i] < blockNumbers[j] })
	return candidates, blockNumbers, nil
}

func (s *Scanner) fetchBlocks(ctx context.Context, numbers []uint64) (map[uint64]*types.Block, error) {
	items := make([]*types.Block, len(numbers))
	group, groupCtx := errgroup.WithContext(ctx)
	group.SetLimit(s.config.FetchConcurrency)
	for index, number := range numbers {
		index, number := index, number
		group.Go(func() error {
			block, err := s.node.BlockByNumber(groupCtx, number)
			if err != nil {
				return err
			}
			items[index] = block
			return nil
		})
	}
	if err := group.Wait(); err != nil {
		return nil, err
	}
	blocks := make(map[uint64]*types.Block, len(numbers))
	for index, number := range numbers {
		blocks[number] = items[index]
	}
	return blocks, nil
}

func swapTransactionsFromCandidates(candidates []swapCandidate, blocks map[uint64]*types.Block) ([]store.SwapTransaction, error) {
	signer := types.LatestSignerForChainID(big.NewInt(bscMainnetChainID))
	transactions := make([]store.SwapTransaction, 0, len(candidates))
	for _, candidate := range candidates {
		block := blocks[candidate.blockNumber]
		if block == nil {
			return nil, fmt.Errorf("BSC swap candidate block %d was not fetched", candidate.blockNumber)
		}
		if block.Hash() != candidate.blockHash {
			return nil, fmt.Errorf("BSC swap candidate block %d hash mismatch", candidate.blockNumber)
		}
		blockTransactions := block.Transactions()
		if candidate.transactionIndex >= uint64(len(blockTransactions)) {
			return nil, fmt.Errorf("BSC swap transaction index %d is outside block %d", candidate.transactionIndex, candidate.blockNumber)
		}
		transaction := blockTransactions[candidate.transactionIndex]
		if transaction == nil || transaction.Hash() != candidate.transactionHash {
			return nil, fmt.Errorf("BSC swap transaction position mismatch for %s", candidate.transactionHash.Hex())
		}
		from, err := types.Sender(signer, transaction)
		if err != nil {
			return nil, fmt.Errorf("derive sender for BSC swap transaction %s: %w", transaction.Hash().Hex(), err)
		}
		transactions = append(transactions, store.SwapTransaction{
			TransactionHash: transaction.Hash(), BlockNumber: block.NumberU64(),
			BlockHash: block.Hash(), BlockTimestamp: block.Time(),
			TransactionIndex: candidate.transactionIndex, FromAddress: from,
		})
	}
	return transactions, nil
}
