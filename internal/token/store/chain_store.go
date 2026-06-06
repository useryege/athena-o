package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	tokensqlc "github.com/useryege/athena/internal/token/store/sqlc"
)

func (s *SQLStore) GetChainIngestCheckpoint(ctx context.Context, chainID int64) (*ChainIngestCheckpoint, error) {
	q, err := s.querier()
	if err != nil {
		return nil, err
	}
	row, err := q.GetChainIngestCheckpoint(ctx, chainID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get chain ingest checkpoint: %w", err)
	}
	item, err := mapChainCheckpointRow(row)
	if err != nil {
		return nil, fmt.Errorf("map chain ingest checkpoint: %w", err)
	}
	return item, nil
}

func (s *SQLStore) ListChainIngestCheckpoints(ctx context.Context) ([]ChainIngestCheckpoint, error) {
	q, err := s.querier()
	if err != nil {
		return nil, err
	}
	rows, err := q.ListChainIngestCheckpoints(ctx)
	if err != nil {
		return nil, fmt.Errorf("list chain ingest checkpoints: %w", err)
	}
	items := make([]ChainIngestCheckpoint, 0, len(rows))
	for _, row := range rows {
		item, err := mapChainCheckpointListRow(row)
		if err != nil {
			return nil, fmt.Errorf("map chain ingest checkpoint: %w", err)
		}
		items = append(items, *item)
	}
	return items, nil
}

func (s *SQLStore) UpsertChainIngestCheckpoint(ctx context.Context, item ChainIngestCheckpoint) (*ChainIngestCheckpoint, error) {
	q, err := s.querier()
	if err != nil {
		return nil, err
	}
	status := item.Status
	if status == "" {
		status = ChainIngestStatusStopped
	}
	finalizedBlockNumber, err := uint64ToInt64("finalized_block_number", item.FinalizedBlockNumber)
	if err != nil {
		return nil, err
	}
	cursorBlockNumber, err := uint64ToInt64("cursor_block_number", item.CursorBlockNumber)
	if err != nil {
		return nil, err
	}
	row, err := q.UpsertChainIngestCheckpoint(ctx, tokensqlc.UpsertChainIngestCheckpointParams{
		ChainID:              item.ChainID,
		FinalizedBlockNumber: finalizedBlockNumber,
		CursorBlockNumber:    cursorBlockNumber,
		Status:               status,
	})
	if err != nil {
		return nil, fmt.Errorf("upsert chain ingest checkpoint: %w", err)
	}
	mapped, err := mapChainCheckpoint(row)
	if err != nil {
		return nil, fmt.Errorf("map chain ingest checkpoint: %w", err)
	}
	mapped.ChainName = item.ChainName
	mapped.Enabled = item.Enabled
	return mapped, nil
}

func (s *SQLStore) UpdateChainIngestCheckpointStatus(ctx context.Context, chainID int64, status string) (*ChainIngestCheckpoint, error) {
	q, err := s.querier()
	if err != nil {
		return nil, err
	}
	row, err := q.UpdateChainIngestCheckpointStatus(ctx, tokensqlc.UpdateChainIngestCheckpointStatusParams{
		ChainID: chainID,
		Status:  status,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("update chain ingest checkpoint status: %w", err)
	}
	item, err := mapChainCheckpoint(row)
	if err != nil {
		return nil, fmt.Errorf("map chain ingest checkpoint: %w", err)
	}
	return item, nil
}
