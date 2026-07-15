package scanner

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/core/types"
	log "github.com/sirupsen/logrus"
)

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
	}).Info("token chain scanner block fetch workers started")
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

type blockFetchJob struct {
	number uint64
}

type blockFetchResult struct {
	number uint64
	block  *types.Block
	err    error
}
