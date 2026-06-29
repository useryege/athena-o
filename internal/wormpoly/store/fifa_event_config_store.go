package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	wormpolysqlc "github.com/useryege/athena/internal/wormpoly/store/sqlc"
)

func (s *SQLStore) GetWormPolyFIFAEventConfig(ctx context.Context) (*FIFAEventConfig, error) {
	queries, err := s.querier()
	if err != nil {
		return nil, err
	}
	row, err := queries.GetWormPolyFIFAEventConfig(ctx)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get worm-poly FIFA event config: %w", err)
	}
	return &FIFAEventConfig{
		WormEventID: row.WormEventID,
		EventRef:    row.EventRef,
		UpdatedAt:   row.UpdatedAt.Time,
	}, nil
}

func (s *SQLStore) UpdateWormPolyFIFAEventConfig(ctx context.Context, config FIFAEventConfig) (*FIFAEventConfig, error) {
	queries, err := s.querier()
	if err != nil {
		return nil, err
	}
	row, err := queries.UpdateWormPolyFIFAEventConfig(ctx, wormpolysqlc.UpdateWormPolyFIFAEventConfigParams{
		WormEventID: config.WormEventID,
		EventRef:    config.EventRef,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("update worm-poly FIFA event config: %w", err)
	}
	return &FIFAEventConfig{
		WormEventID: row.WormEventID,
		EventRef:    row.EventRef,
		UpdatedAt:   row.UpdatedAt.Time,
	}, nil
}
