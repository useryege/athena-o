package bscinbound

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/jackc/pgx/v5"
	log "github.com/sirupsen/logrus"
	"github.com/useryege/athena/internal/bscinbound/store"
	"golang.org/x/sync/errgroup"
)

const (
	defaultInitialLookback  = 30 * 24 * time.Hour
	DefaultScanBatchSize    = 1000
	DefaultFetchConcurrency = 64
	DefaultPollInterval     = time.Second
)

var minimumInboundValueWei = big.NewInt(10_000_000_000_000_000)

type ScannerConfig struct {
	BatchSize        int
	FetchConcurrency int
	PollInterval     time.Duration
}

func DefaultScannerConfig() ScannerConfig {
	return ScannerConfig{
		BatchSize: DefaultScanBatchSize, FetchConcurrency: DefaultFetchConcurrency,
		PollInterval: DefaultPollInterval,
	}
}

type Scanner struct {
	store   *store.Store
	node    *Node
	metrics *Metrics
	config  ScannerConfig
}

func NewScanner(repository *store.Store, node *Node, metrics *Metrics, config ScannerConfig) (*Scanner, error) {
	if repository == nil {
		return nil, fmt.Errorf("BSC inbound store is required")
	}
	if node == nil {
		return nil, fmt.Errorf("BSC node is required")
	}
	if metrics == nil {
		return nil, fmt.Errorf("BSC inbound metrics are required")
	}
	if config.BatchSize <= 0 {
		return nil, fmt.Errorf("BSC inbound scan batch size must be positive")
	}
	if config.FetchConcurrency <= 0 {
		return nil, fmt.Errorf("BSC inbound fetch concurrency must be positive")
	}
	if config.PollInterval <= 0 {
		return nil, fmt.Errorf("BSC inbound poll interval must be positive")
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
			log.WithError(err).Error("BSC inbound scanner iteration failed")
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
		s.metrics.RecordScanSuccess(0, 0)
		return false, nil
	}

	start := checkpoint.CursorBlockNumber + 1
	end := finalizedNumber
	if remaining := end - start + 1; remaining > uint64(s.config.BatchSize) {
		end = start + uint64(s.config.BatchSize) - 1
	}
	startedAt := time.Now()
	blocks, err := s.fetchBlocks(ctx, start, end)
	if err != nil {
		return false, err
	}
	if err := verifyBlockContinuity(checkpoint.CursorBlockHash, blocks); err != nil {
		return false, err
	}
	transactions, err := s.collectTransactions(ctx, blocks)
	if err != nil {
		return false, err
	}
	lastBlock := blocks[len(blocks)-1]
	nextCheckpoint := store.Checkpoint{
		StartBlockNumber: checkpoint.StartBlockNumber, CursorBlockNumber: lastBlock.NumberU64(),
		CursorBlockHash: lastBlock.Hash(), CursorBlockTimestamp: lastBlock.Time(),
	}
	if err := s.store.CommitBatch(ctx, checkpoint.CursorBlockNumber, transactions, nextCheckpoint); err != nil {
		return false, err
	}
	blockCount := uint64(len(blocks))
	s.metrics.RecordProgress(finalizedNumber, nextCheckpoint.CursorBlockNumber)
	s.metrics.RecordScanSuccess(blockCount, uint64(len(transactions)))
	log.WithFields(log.Fields{
		"start_block": start, "end_block": end, "block_count": blockCount,
		"transaction_count": len(transactions), "duration_ms": time.Since(startedAt).Milliseconds(),
	}).Info("BSC inbound scan batch completed")
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
	}).Info("initialized BSC inbound scan checkpoint")
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

func (s *Scanner) fetchBlocks(ctx context.Context, start, end uint64) ([]*types.Block, error) {
	blocks := make([]*types.Block, int(end-start+1))
	group, groupCtx := errgroup.WithContext(ctx)
	group.SetLimit(s.config.FetchConcurrency)
	for number := start; number <= end; number++ {
		number := number
		group.Go(func() error {
			block, err := s.node.BlockByNumber(groupCtx, number)
			if err != nil {
				return err
			}
			blocks[number-start] = block
			return nil
		})
	}
	if err := group.Wait(); err != nil {
		return nil, err
	}
	return blocks, nil
}

func verifyBlockContinuity(previousHash [32]byte, blocks []*types.Block) error {
	expectedParent := previousHash
	for _, block := range blocks {
		if block == nil {
			return fmt.Errorf("BSC scan batch contains an empty block")
		}
		if block.ParentHash() != expectedParent {
			return fmt.Errorf("BSC block %d parent hash does not match committed chain", block.NumberU64())
		}
		expectedParent = block.Hash()
	}
	return nil
}

type receiptCandidate struct {
	block            *types.Block
	transaction      *types.Transaction
	transactionIndex uint64
	from             [20]byte
	receipt          *types.Receipt
}

func (s *Scanner) collectTransactions(ctx context.Context, blocks []*types.Block) ([]store.InboundNormalTransaction, error) {
	signer := types.LatestSignerForChainID(big.NewInt(bscMainnetChainID))
	candidates := make([]receiptCandidate, 0)
	for _, block := range blocks {
		for transactionIndex, transaction := range block.Transactions() {
			if transaction == nil || transaction.To() == nil || transaction.Value().Cmp(minimumInboundValueWei) <= 0 || len(transaction.Data()) != 0 {
				continue
			}
			from, err := types.Sender(signer, transaction)
			if err != nil {
				return nil, fmt.Errorf("derive sender for BSC transaction %s: %w", transaction.Hash().Hex(), err)
			}
			candidates = append(candidates, receiptCandidate{
				block: block, transaction: transaction, transactionIndex: uint64(transactionIndex), from: from,
			})
		}
	}

	group, groupCtx := errgroup.WithContext(ctx)
	group.SetLimit(s.config.FetchConcurrency)
	for index := range candidates {
		index := index
		group.Go(func() error {
			receipt, err := s.node.TransactionReceipt(groupCtx, candidates[index].transaction.Hash())
			if err != nil {
				return err
			}
			candidates[index].receipt = receipt
			return nil
		})
	}
	if err := group.Wait(); err != nil {
		return nil, err
	}

	transactions := make([]store.InboundNormalTransaction, 0, len(candidates))
	for _, candidate := range candidates {
		if candidate.receipt.Status != types.ReceiptStatusSuccessful {
			continue
		}
		if candidate.receipt.BlockHash != candidate.block.Hash() || uint64(candidate.receipt.TransactionIndex) != candidate.transactionIndex {
			return nil, fmt.Errorf("receipt position mismatch for BSC transaction %s", candidate.transaction.Hash().Hex())
		}
		transactions = append(transactions, store.InboundNormalTransaction{
			TransactionHash: candidate.transaction.Hash(), BlockNumber: candidate.block.NumberU64(),
			BlockHash: candidate.block.Hash(), BlockTimestamp: candidate.block.Time(),
			TransactionIndex: candidate.transactionIndex, FromAddress: candidate.from,
			ToAddress: *candidate.transaction.To(), ValueWei: new(big.Int).Set(candidate.transaction.Value()),
		})
	}
	return transactions, nil
}
