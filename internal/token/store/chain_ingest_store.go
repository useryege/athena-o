package store

import (
	"context"
	"fmt"

	tokensqlc "github.com/useryege/athena/internal/token/store/sqlc"
)

func (s *SQLStore) IngestProjectCandidateBatch(ctx context.Context, checkpoint ChainIngestCheckpoint, candidates []ProjectCandidate) (*ChainIngestCheckpoint, error) {
	if s == nil || s.pool == nil {
		return nil, fmt.Errorf("token postgres database is not configured")
	}
	cursorBlockNumber, err := uint64ToInt64("cursor_block_number", checkpoint.CursorBlockNumber)
	if err != nil {
		return nil, err
	}
	status := checkpoint.Status
	if status == "" {
		status = ChainIngestStatusRunning
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin ingest project candidate batch transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	q := tokensqlc.New(tx)
	if len(candidates) > 0 {
		params, err := batchUpsertProjectCandidatesParams(candidates)
		if err != nil {
			return nil, err
		}
		if err := q.BatchUpsertProjectCandidates(ctx, params); err != nil {
			return nil, fmt.Errorf("batch upsert project candidates: %w", err)
		}
	}
	row, err := q.UpsertChainIngestCheckpoint(ctx, tokensqlc.UpsertChainIngestCheckpointParams{
		ChainID:           checkpoint.ChainID,
		CursorBlockNumber: cursorBlockNumber,
		Status:            status,
	})
	if err != nil {
		return nil, fmt.Errorf("upsert chain ingest checkpoint: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit ingest project candidate batch transaction: %w", err)
	}
	mapped, err := mapChainCheckpoint(row)
	if err != nil {
		return nil, fmt.Errorf("map chain ingest checkpoint: %w", err)
	}
	mapped.ChainName = checkpoint.ChainName
	mapped.Enabled = checkpoint.Enabled
	return mapped, nil
}
