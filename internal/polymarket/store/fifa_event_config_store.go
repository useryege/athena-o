package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	polymarketsqlc "github.com/useryege/athena/internal/polymarket/store/sqlc"
)

func (s *SQLStore) GetPolymarketFIFAEventConfig(ctx context.Context) (*FIFAEventConfig, error) {
	if s == nil || s.queries == nil {
		return nil, fmt.Errorf("polymarket postgres database is not configured")
	}
	row, err := s.queries.GetPolymarketFIFAEventConfig(ctx)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get polymarket FIFA event config: %w", err)
	}
	return &FIFAEventConfig{
		WormEventID: row.WormEventID,
		EventRef:    row.EventRef,
		UpdatedAt:   timeValue(row.UpdatedAt),
	}, nil
}

func (s *SQLStore) UpdatePolymarketFIFAEventConfig(ctx context.Context, config FIFAEventConfig) (*FIFAEventConfig, error) {
	if s == nil || s.queries == nil {
		return nil, fmt.Errorf("polymarket postgres database is not configured")
	}
	row, err := s.queries.UpdatePolymarketFIFAEventConfig(ctx, polymarketsqlc.UpdatePolymarketFIFAEventConfigParams{
		WormEventID: config.WormEventID,
		EventRef:    config.EventRef,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("update polymarket FIFA event config: %w", err)
	}
	return &FIFAEventConfig{
		WormEventID: row.WormEventID,
		EventRef:    row.EventRef,
		UpdatedAt:   timeValue(row.UpdatedAt),
	}, nil
}
