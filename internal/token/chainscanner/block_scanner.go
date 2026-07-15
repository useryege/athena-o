package chainscanner

import (
	"context"
	"fmt"

	log "github.com/sirupsen/logrus"
	tokenstore "github.com/useryege/athena/internal/token/store"
)

func (r *chainRunner) processAvailableBlocks(ctx context.Context) error {
	checkpoint, err := r.opts.store.GetChainIngestCheckpoint(ctx, r.opts.chainID)
	if err != nil {
		return err
	}
	if checkpoint == nil {
		return fmt.Errorf("token chain scanner checkpoint missing for chain %d", r.opts.chainID)
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
		}).Debug("token chain scanner is caught up")
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
		}).Debug("token chain scanner processed block batch")
		next = batchEnd + 1
	}
	return nil
}
