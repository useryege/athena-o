package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	polymarketsqlc "github.com/useryege/athena/internal/polymarket/store/sqlc"
)

const sportsLiveSyncName = "sports_live_markets"

func (s *SQLStore) SyncSportsLiveMarkets(ctx context.Context, items []SportsLiveMarket, syncStartedAt, fetchedAt time.Time) error {
	if s == nil || s.pool == nil {
		return fmt.Errorf("polymarket postgres database is not configured")
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin sports live market sync: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	queries := s.queries.WithTx(tx)
	if len(items) > 0 {
		if err := queries.BatchUpsertSportsLiveMarkets(ctx, batchUpsertSportsLiveMarketsParams(items)); err != nil {
			return fmt.Errorf("batch upsert sports live markets: %w", err)
		}
	}
	if _, err := queries.DeleteSportsLiveMarketsNotSeenSince(ctx, nullableTime(syncStartedAt)); err != nil {
		return fmt.Errorf("delete stale sports live markets: %w", err)
	}
	if err := queries.UpsertPolymarketSyncState(ctx, polymarketsqlc.UpsertPolymarketSyncStateParams{
		SyncName:      sportsLiveSyncName,
		LastSuccessAt: nullableTime(fetchedAt),
	}); err != nil {
		return fmt.Errorf("update sports live sync state: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit sports live market sync: %w", err)
	}
	return nil
}

func (s *SQLStore) ListSportsLiveMarkets(ctx context.Context, limit int32) ([]SportsLiveMarket, error) {
	if s == nil || s.queries == nil {
		return nil, fmt.Errorf("polymarket postgres database is not configured")
	}
	rows, err := s.queries.ListSportsLiveMarkets(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("list sports live markets: %w", err)
	}
	items := make([]SportsLiveMarket, 0, len(rows))
	for _, row := range rows {
		items = append(items, SportsLiveMarket{
			ConditionID:    row.ConditionID,
			MarketSlug:     row.MarketSlug,
			EventSlug:      row.EventSlug,
			Title:          row.Title,
			Image:          row.Image,
			Score:          row.Score,
			Period:         row.Period,
			Elapsed:        row.Elapsed,
			GammaUpdatedAt: timeValue(row.GammaUpdatedAt),
			LiquidityNum:   row.LiquidityNum,
			VolumeNum:      row.VolumeNum,
			FetchedAt:      timeValue(row.FetchedAt),
			LastSeenAt:     timeValue(row.LastSeenAt),
		})
	}
	return items, nil
}

func (s *SQLStore) GetSportsLiveLastSuccessAt(ctx context.Context) (time.Time, error) {
	if s == nil || s.queries == nil {
		return time.Time{}, fmt.Errorf("polymarket postgres database is not configured")
	}
	value, err := s.queries.GetPolymarketSyncState(ctx, sportsLiveSyncName)
	if errors.Is(err, pgx.ErrNoRows) {
		return time.Time{}, nil
	}
	if err != nil {
		return time.Time{}, fmt.Errorf("get sports live sync state: %w", err)
	}
	return timeValue(value), nil
}

func batchUpsertSportsLiveMarketsParams(items []SportsLiveMarket) polymarketsqlc.BatchUpsertSportsLiveMarketsParams {
	params := polymarketsqlc.BatchUpsertSportsLiveMarketsParams{
		ConditionIds:         make([]string, 0, len(items)),
		MarketSlugs:          make([]string, 0, len(items)),
		EventSlugs:           make([]string, 0, len(items)),
		Titles:               make([]string, 0, len(items)),
		Images:               make([]string, 0, len(items)),
		Scores:               make([]string, 0, len(items)),
		Periods:              make([]string, 0, len(items)),
		ElapsedValues:        make([]string, 0, len(items)),
		GammaUpdatedAtValues: make([]pgtype.Timestamptz, 0, len(items)),
		LiquidityNumValues:   make([]float64, 0, len(items)),
		VolumeNumValues:      make([]float64, 0, len(items)),
		FetchedAtValues:      make([]pgtype.Timestamptz, 0, len(items)),
		LastSeenAtValues:     make([]pgtype.Timestamptz, 0, len(items)),
	}
	for _, item := range items {
		params.ConditionIds = append(params.ConditionIds, item.ConditionID)
		params.MarketSlugs = append(params.MarketSlugs, item.MarketSlug)
		params.EventSlugs = append(params.EventSlugs, item.EventSlug)
		params.Titles = append(params.Titles, item.Title)
		params.Images = append(params.Images, item.Image)
		params.Scores = append(params.Scores, item.Score)
		params.Periods = append(params.Periods, item.Period)
		params.ElapsedValues = append(params.ElapsedValues, item.Elapsed)
		params.GammaUpdatedAtValues = append(params.GammaUpdatedAtValues, nullableTime(item.GammaUpdatedAt))
		params.LiquidityNumValues = append(params.LiquidityNumValues, item.LiquidityNum)
		params.VolumeNumValues = append(params.VolumeNumValues, item.VolumeNum)
		params.FetchedAtValues = append(params.FetchedAtValues, nullableTime(item.FetchedAt))
		params.LastSeenAtValues = append(params.LastSeenAtValues, nullableTime(item.LastSeenAt))
	}
	return params
}

func nullableTime(value time.Time) pgtype.Timestamptz {
	if value.IsZero() {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: value, Valid: true}
}

func timeValue(value pgtype.Timestamptz) time.Time {
	if !value.Valid {
		return time.Time{}
	}
	return value.Time
}
