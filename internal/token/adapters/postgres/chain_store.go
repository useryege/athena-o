package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	tokensqlc "github.com/useryege/athena/internal/token/adapters/postgres/sqlc"
	"github.com/useryege/athena/internal/token/discovery"
)

func (s *ChainRepository) SyncChains(ctx context.Context, items []discovery.Chain) error {
	q, err := s.querier()
	if err != nil {
		return err
	}
	for _, item := range items {
		if _, err := q.UpsertChain(ctx, tokensqlc.UpsertChainParams{ID: item.ID, Name: item.Name, Enabled: item.Enabled}); err != nil {
			return fmt.Errorf("sync token chain %d: %w", item.ID, err)
		}
		checkpoint, err := s.GetChainProcessingCheckpoint(ctx, item.ID)
		if err != nil {
			return err
		}
		if checkpoint == nil {
			if _, err := s.UpsertChainProcessingCheckpoint(ctx, discovery.ChainProcessingCheckpoint{ChainID: item.ID, ChainName: item.Name, Enabled: item.Enabled, Status: discovery.ChainProcessingStatusStopped}); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *ChainRepository) GetChainProcessingCheckpoint(ctx context.Context, chainID int64) (*discovery.ChainProcessingCheckpoint, error) {
	q, err := s.querier()
	if err != nil {
		return nil, err
	}
	row, err := q.GetChainProcessingCheckpoint(ctx, chainID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get chain processing checkpoint: %w", err)
	}
	item, err := mapChainCheckpointRow(row)
	if err != nil {
		return nil, fmt.Errorf("map chain processing checkpoint: %w", err)
	}
	return item, nil
}

func (s *ChainRepository) ListChainProcessingCheckpoints(ctx context.Context) ([]discovery.ChainProcessingCheckpoint, error) {
	q, err := s.querier()
	if err != nil {
		return nil, err
	}
	rows, err := q.ListChainProcessingCheckpoints(ctx)
	if err != nil {
		return nil, fmt.Errorf("list chain processing checkpoints: %w", err)
	}
	items := make([]discovery.ChainProcessingCheckpoint, 0, len(rows))
	for _, row := range rows {
		item, err := mapChainCheckpointListRow(row)
		if err != nil {
			return nil, fmt.Errorf("map chain processing checkpoint: %w", err)
		}
		items = append(items, *item)
	}
	return items, nil
}

func (s *ChainRepository) UpsertChainProcessingCheckpoint(ctx context.Context, item discovery.ChainProcessingCheckpoint) (*discovery.ChainProcessingCheckpoint, error) {
	q, err := s.querier()
	if err != nil {
		return nil, err
	}
	status := item.Status
	if status == "" {
		status = discovery.ChainProcessingStatusStopped
	}
	cursorBlockNumber, err := uint64ToInt64("cursor_block_number", item.CursorBlockNumber)
	if err != nil {
		return nil, err
	}
	row, err := q.UpsertChainProcessingCheckpoint(ctx, tokensqlc.UpsertChainProcessingCheckpointParams{
		ChainID:           item.ChainID,
		CursorBlockNumber: cursorBlockNumber,
		Status:            string(status),
	})
	if err != nil {
		return nil, fmt.Errorf("upsert chain processing checkpoint: %w", err)
	}
	mapped, err := mapChainCheckpoint(row)
	if err != nil {
		return nil, fmt.Errorf("map chain processing checkpoint: %w", err)
	}
	mapped.ChainName = item.ChainName
	mapped.Enabled = item.Enabled
	return mapped, nil
}

func (s *ChainRepository) UpdateChainProcessingCheckpointStatus(ctx context.Context, chainID int64, status discovery.ChainProcessingStatus) (*discovery.ChainProcessingCheckpoint, error) {
	q, err := s.querier()
	if err != nil {
		return nil, err
	}
	row, err := q.UpdateChainProcessingCheckpointStatus(ctx, tokensqlc.UpdateChainProcessingCheckpointStatusParams{
		ChainID: chainID,
		Status:  string(status),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("update chain processing checkpoint status: %w", err)
	}
	item, err := mapChainCheckpoint(row)
	if err != nil {
		return nil, fmt.Errorf("map chain processing checkpoint: %w", err)
	}
	return item, nil
}

func (s *ChainRepository) ListChains(ctx context.Context) ([]discovery.Chain, error) {
	q, err := s.querier()
	if err != nil {
		return nil, err
	}
	rows, err := q.ListChains(ctx)
	if err != nil {
		return nil, fmt.Errorf("list chains: %w", err)
	}
	items := make([]discovery.Chain, 0, len(rows))
	for _, row := range rows {
		items = append(items, discovery.Chain{
			ID:        row.ID,
			Name:      row.Name,
			Enabled:   row.Enabled,
			CreatedAt: timeValue(row.CreatedAt),
		})
	}
	return items, nil
}
