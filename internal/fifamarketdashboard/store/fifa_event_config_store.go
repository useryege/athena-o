package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	fifamarketdashboardsqlc "github.com/useryege/athena/internal/fifamarketdashboard/store/sqlc"
)

func (s *SQLStore) GetFIFAEventConfig(ctx context.Context) (*FIFAEventConfig, error) {
	queries, err := s.querier()
	if err != nil {
		return nil, err
	}
	row, err := queries.GetFIFAEventConfig(ctx)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get FIFA event config: %w", err)
	}
	return &FIFAEventConfig{
		WormEventID: row.WormEventID,
		EventRef:    row.EventRef,
		UpdatedAt:   row.UpdatedAt.Time,
	}, nil
}

func (s *SQLStore) UpdateFIFAEventConfig(ctx context.Context, config FIFAEventConfig) (*FIFAEventConfig, error) {
	queries, err := s.querier()
	if err != nil {
		return nil, err
	}
	row, err := queries.UpdateFIFAEventConfig(ctx, fifamarketdashboardsqlc.UpdateFIFAEventConfigParams{
		WormEventID: config.WormEventID,
		EventRef:    config.EventRef,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("update FIFA event config: %w", err)
	}
	return &FIFAEventConfig{
		WormEventID: row.WormEventID,
		EventRef:    row.EventRef,
		UpdatedAt:   row.UpdatedAt.Time,
	}, nil
}
