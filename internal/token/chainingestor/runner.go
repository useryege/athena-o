package chainingestor

import (
	"context"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	log "github.com/sirupsen/logrus"
	tokenstore "github.com/useryege/athena/internal/token/store"
	"github.com/useryege/athena/util/ethws"
)

const maxBlocksPerBatch = uint64(100)

type chainRunnerOptions struct {
	store                 *tokenstore.SQLStore
	chainID               int64
	chainName             string
	nodeWSURL             string
	nodeWSUseProxy        bool
	pollInterval          time.Duration
	blockFetchConcurrency int
}

type chainRunner struct {
	opts   chainRunnerOptions
	signer types.Signer

	clientMu sync.Mutex
	client   *ethclient.Client

	blockFetchJobs    chan blockFetchJob
	blockFetchResults chan blockFetchResult
	blockFetchCancel  context.CancelFunc
	blockFetchWG      sync.WaitGroup
	closeOnce         sync.Once
}

func newChainRunner(opts chainRunnerOptions) *chainRunner {
	if opts.blockFetchConcurrency <= 0 {
		opts.blockFetchConcurrency = 1
	}
	return &chainRunner{
		opts:              opts,
		signer:            types.LatestSignerForChainID(big.NewInt(opts.chainID)),
		blockFetchJobs:    make(chan blockFetchJob, maxBlocksPerBatch),
		blockFetchResults: make(chan blockFetchResult, maxBlocksPerBatch),
	}
}

func (r *chainRunner) run(ctx context.Context) {
	defer r.close()
	r.startBlockFetchWorkers(ctx)
	for {
		if err := ctx.Err(); err != nil {
			return
		}
		if err := r.processAvailableBlocks(ctx); err != nil {
			log.WithError(err).WithFields(log.Fields{
				"chain_id":   r.opts.chainID,
				"chain_name": r.opts.chainName,
			}).Error("token chain ingestor scan failed")
			if !sleepContext(ctx, r.opts.pollInterval) {
				return
			}
			continue
		}
		if !sleepContext(ctx, r.opts.pollInterval) {
			return
		}
	}
}

func (r *chainRunner) processAvailableBlocks(ctx context.Context) error {
	checkpoint, err := r.opts.store.GetChainIngestCheckpoint(ctx, r.opts.chainID)
	if err != nil {
		return err
	}
	if checkpoint == nil {
		return fmt.Errorf("token chain ingestor checkpoint missing for chain %d", r.opts.chainID)
	}
	if !checkpoint.Enabled || checkpoint.Status != tokenstore.ChainIngestStatusRunning {
		return nil
	}
	client, err := r.ensureClient(ctx)
	if err != nil {
		return err
	}
	latest, err := client.BlockNumber(ctx)
	if err != nil {
		r.resetClient()
		return err
	}
	next := uint64(1)
	if checkpoint.CursorBlockNumber > 0 {
		next = checkpoint.CursorBlockNumber + 1
	}
	if next > latest {
		log.WithFields(log.Fields{
			"chain_id":            r.opts.chainID,
			"latest_block_number": latest,
			"cursor_block_number": checkpoint.CursorBlockNumber,
		}).Debug("token chain ingestor is caught up")
		return nil
	}
	for next <= latest {
		if err := ctx.Err(); err != nil {
			return err
		}
		checkpoint, err = r.opts.store.GetChainIngestCheckpoint(ctx, r.opts.chainID)
		if err != nil {
			return err
		}
		if checkpoint == nil || !checkpoint.Enabled || checkpoint.Status != tokenstore.ChainIngestStatusRunning {
			return nil
		}
		batchStart := next
		batchEnd := batchStart + maxBlocksPerBatch - 1
		if batchEnd > latest {
			batchEnd = latest
		}
		blocks, err := r.fetchBlockBatch(ctx, batchStart, batchEnd)
		if err != nil {
			r.resetClient()
			return err
		}
		candidates := make([]tokenstore.ProjectCandidate, 0)
		for _, block := range blocks {
			blockCandidates, err := r.projectCandidatesFromBlock(block)
			if err != nil {
				return err
			}
			candidates = append(candidates, blockCandidates...)
		}
		if _, err := r.opts.store.IngestProjectCandidateBatch(ctx, tokenstore.ChainIngestCheckpoint{
			ChainID:           r.opts.chainID,
			CursorBlockNumber: batchEnd,
			Status:            tokenstore.ChainIngestStatusRunning,
		}, candidates); err != nil {
			return err
		}
		log.WithFields(log.Fields{
			"chain_id":         r.opts.chainID,
			"batch_start":      batchStart,
			"batch_end":        batchEnd,
			"candidate_count":  len(candidates),
			"latest_block_num": latest,
		}).Debug("token chain ingestor processed block batch")
		next = batchEnd + 1
	}
	return nil
}

func (r *chainRunner) projectCandidatesFromBlock(block *types.Block) ([]tokenstore.ProjectCandidate, error) {
	if block == nil {
		return nil, nil
	}
	candidates := make([]tokenstore.ProjectCandidate, 0)
	for txIndex, tx := range block.Transactions() {
		if tx == nil || tx.To() != nil {
			continue
		}
		creator, err := types.Sender(r.signer, tx)
		if err != nil {
			return nil, fmt.Errorf("derive contract creator for tx %s: %w", tx.Hash().Hex(), err)
		}
		contract := crypto.CreateAddress(creator, tx.Nonce())
		if contract == (common.Address{}) {
			continue
		}
		candidates = append(candidates, tokenstore.ProjectCandidate{
			ChainID:     r.opts.chainID,
			Contract:    contract,
			Creator:     creator,
			TxHash:      tx.Hash(),
			TxIndex:     uint64(txIndex),
			BlockNumber: block.NumberU64(),
			BlockTime:   block.Time(),
			Status:      tokenstore.ProjectCandidateStatusPending,
		})
	}
	return candidates, nil
}

func (r *chainRunner) startBlockFetchWorkers(ctx context.Context) {
	workerCtx, cancel := context.WithCancel(ctx)
	r.blockFetchCancel = cancel
	for i := 0; i < r.opts.blockFetchConcurrency; i++ {
		r.blockFetchWG.Add(1)
		go func() {
			defer r.blockFetchWG.Done()
			r.blockFetchWorker(workerCtx)
		}()
	}
	log.WithFields(log.Fields{
		"chain_id":          r.opts.chainID,
		"chain_name":        r.opts.chainName,
		"fetch_concurrency": r.opts.blockFetchConcurrency,
	}).Info("token chain ingestor block fetch workers started")
}

func (r *chainRunner) blockFetchWorker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case job := <-r.blockFetchJobs:
			block, err := r.fetchBlock(ctx, job.number)
			result := blockFetchResult{
				number: job.number,
				block:  block,
				err:    err,
			}
			select {
			case <-ctx.Done():
				return
			case r.blockFetchResults <- result:
			}
		}
	}
}

func (r *chainRunner) fetchBlock(ctx context.Context, number uint64) (*types.Block, error) {
	client, err := r.ensureClient(ctx)
	if err != nil {
		return nil, err
	}
	return client.BlockByNumber(ctx, new(big.Int).SetUint64(number))
}

func (r *chainRunner) fetchBlockBatch(ctx context.Context, batchStart, batchEnd uint64) ([]*types.Block, error) {
	total := int(batchEnd - batchStart + 1)
	for number := batchStart; number <= batchEnd; number++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case r.blockFetchJobs <- blockFetchJob{number: number}:
		}
	}

	blocks := make([]*types.Block, total)
	var firstErr error
	for i := 0; i < total; i++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case result := <-r.blockFetchResults:
			if result.err != nil && firstErr == nil {
				firstErr = fmt.Errorf("fetch block %d: %w", result.number, result.err)
				continue
			}
			if result.err == nil && result.block == nil && firstErr == nil {
				firstErr = fmt.Errorf("fetch block %d returned nil block", result.number)
				continue
			}
			if result.err == nil {
				blocks[result.number-batchStart] = result.block
			}
		}
	}
	if firstErr != nil {
		return nil, firstErr
	}
	return blocks, nil
}

func (r *chainRunner) ensureClient(ctx context.Context) (*ethclient.Client, error) {
	r.clientMu.Lock()
	defer r.clientMu.Unlock()
	if r.client != nil {
		return r.client, nil
	}
	client, err := ethws.DialContext(ctx, r.opts.nodeWSURL, r.opts.nodeWSUseProxy)
	if err != nil {
		return nil, err
	}
	chainID, err := client.ChainID(ctx)
	if err != nil {
		client.Close()
		return nil, err
	}
	if chainID == nil || chainID.Int64() != r.opts.chainID {
		client.Close()
		return nil, fmt.Errorf("token chain ingestor node returned chain_id %v for configured chain_id %d", chainID, r.opts.chainID)
	}
	r.client = client
	log.WithFields(log.Fields{
		"chain_id":   r.opts.chainID,
		"chain_name": r.opts.chainName,
	}).Info("token chain ingestor connected to node websocket")
	return client, nil
}

func (r *chainRunner) resetClient() {
	r.clientMu.Lock()
	defer r.clientMu.Unlock()
	if r.client != nil {
		r.client.Close()
		r.client = nil
	}
}

func (r *chainRunner) close() {
	r.closeOnce.Do(func() {
		if r.blockFetchCancel != nil {
			r.blockFetchCancel()
		}
		r.blockFetchWG.Wait()
		r.resetClient()
	})
}

type blockFetchJob struct {
	number uint64
}

type blockFetchResult struct {
	number uint64
	block  *types.Block
	err    error
}

func sleepContext(ctx context.Context, interval time.Duration) bool {
	timer := time.NewTimer(interval)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
