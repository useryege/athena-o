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

const umaResolutionMarketsSyncName = "uma_resolution_markets"

func (s *SQLStore) SyncUMAResolutionMarkets(ctx context.Context, markets []UMAResolutionMarket, syncStartedAt, fetchedAt time.Time) ([]UMAResolutionNotificationCandidate, error) {
	if s == nil || s.pool == nil {
		return nil, fmt.Errorf("polymarket postgres database is not configured")
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin uma resolution markets sync: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	queries := s.queries.WithTx(tx)
	lastSuccessAt, err := queries.GetPolymarketSyncState(ctx, umaResolutionMarketsSyncName)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("get uma resolution markets sync state: %w", err)
	}
	firstSync := errors.Is(err, pgx.ErrNoRows) || timeValue(lastSuccessAt).IsZero()
	if len(markets) > 0 {
		if err := queries.BatchUpsertUMAResolutionMarkets(ctx, batchUpsertUMAResolutionMarketsParams(markets)); err != nil {
			return nil, fmt.Errorf("batch upsert uma resolution markets: %w", err)
		}
	}
	if _, err := queries.DeleteUMAResolutionMarketsNotSeenSince(ctx, nullableTime(syncStartedAt)); err != nil {
		return nil, fmt.Errorf("delete stale uma resolution markets: %w", err)
	}

	var candidates []UMAResolutionNotificationCandidate
	if firstSync {
		if err := queries.UpsertUMAResolutionNotificationBaselines(ctx, nullableTime(fetchedAt)); err != nil {
			return nil, fmt.Errorf("upsert uma resolution notification baselines: %w", err)
		}
	} else {
		rows, err := queries.ListPendingUMAResolutionNotificationCandidates(ctx)
		if err != nil {
			return nil, fmt.Errorf("list pending uma resolution notification candidates: %w", err)
		}
		candidates = make([]UMAResolutionNotificationCandidate, 0, len(rows))
		for _, row := range rows {
			candidates = append(candidates, UMAResolutionNotificationCandidate{
				MarketKey:             row.MarketKey,
				ConditionID:           row.ConditionID,
				Slug:                  row.Slug,
				EventSlug:             row.EventSlug,
				Question:              row.Question,
				UMAResolutionStatus:   row.UmaResolutionStatus,
				UMAResolutionStatuses: row.UmaResolutionStatuses,
				Volume24hr:            row.Volume24hr,
				LiquidityNum:          row.LiquidityNum,
				FetchedAt:             timeValue(row.FetchedAt),
				LastSeenAt:            timeValue(row.LastSeenAt),
			})
		}
	}
	if err := queries.UpsertPolymarketSyncState(ctx, polymarketsqlc.UpsertPolymarketSyncStateParams{
		SyncName:      umaResolutionMarketsSyncName,
		LastSuccessAt: nullableTime(fetchedAt),
	}); err != nil {
		return nil, fmt.Errorf("update uma resolution markets sync state: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit uma resolution markets sync: %w", err)
	}
	return candidates, nil
}

func (s *SQLStore) SyncDisputedMarkets(ctx context.Context, markets []DisputedMarket, syncStartedAt, fetchedAt time.Time) error {
	_, err := s.SyncUMAResolutionMarkets(ctx, markets, syncStartedAt, fetchedAt)
	return err
}

func (s *SQLStore) ListDisputedMarkets(ctx context.Context, limit int32) ([]DisputedMarket, error) {
	if s == nil || s.queries == nil {
		return nil, fmt.Errorf("polymarket postgres database is not configured")
	}
	rows, err := s.queries.ListDisputedMarkets(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("list disputed markets: %w", err)
	}
	items := make([]DisputedMarket, 0, len(rows))
	for _, row := range rows {
		items = append(items, DisputedMarket{
			MarketKey:             row.MarketKey,
			MarketID:              row.MarketID,
			ConditionID:           row.ConditionID,
			Slug:                  row.Slug,
			EventID:               row.EventID,
			EventSlug:             row.EventSlug,
			Question:              row.Question,
			Image:                 firstNonEmptyText(row.Image, row.Icon),
			Icon:                  row.Icon,
			UMAResolutionStatus:   row.UmaResolutionStatus,
			UMAResolutionStatuses: row.UmaResolutionStatuses,
			Active:                row.Active,
			Closed:                row.Closed,
			EnableOrderBook:       row.EnableOrderBook,
			VolumeNum:             row.VolumeNum,
			LiquidityNum:          row.LiquidityNum,
			Volume24hr:            row.Volume24hr,
			Spread:                row.Spread,
			BestBid:               row.BestBid,
			BestAsk:               row.BestAsk,
			LastTradePrice:        row.LastTradePrice,
			FetchedAt:             timeValue(row.FetchedAt),
			LastSeenAt:            timeValue(row.LastSeenAt),
		})
	}
	return items, nil
}

func (s *SQLStore) GetDisputedMarketsLastSuccessAt(ctx context.Context) (time.Time, error) {
	if s == nil || s.queries == nil {
		return time.Time{}, fmt.Errorf("polymarket postgres database is not configured")
	}
	value, err := s.queries.GetPolymarketSyncState(ctx, umaResolutionMarketsSyncName)
	if errors.Is(err, pgx.ErrNoRows) {
		return time.Time{}, nil
	}
	if err != nil {
		return time.Time{}, fmt.Errorf("get disputed markets sync state: %w", err)
	}
	return timeValue(value), nil
}

func (s *SQLStore) MarkUMAResolutionNotificationSent(ctx context.Context, candidate UMAResolutionNotificationCandidate, notificationID int64, notifiedAt time.Time) error {
	if s == nil || s.queries == nil {
		return fmt.Errorf("polymarket postgres database is not configured")
	}
	if err := s.queries.UpsertUMAResolutionNotificationSent(ctx, polymarketsqlc.UpsertUMAResolutionNotificationSentParams{
		MarketKey:           candidate.MarketKey,
		UmaResolutionStatus: candidate.UMAResolutionStatus,
		ConditionID:         candidate.ConditionID,
		Slug:                candidate.Slug,
		EventSlug:           candidate.EventSlug,
		Question:            candidate.Question,
		FirstSeenAt:         nullableTime(candidate.LastSeenAt),
		LastSeenAt:          nullableTime(candidate.LastSeenAt),
		NotifiedAt:          nullableTime(notifiedAt),
		NotificationID:      notificationID,
	}); err != nil {
		return fmt.Errorf("upsert uma resolution notification state: %w", err)
	}
	return nil
}

func batchUpsertUMAResolutionMarketsParams(items []UMAResolutionMarket) polymarketsqlc.BatchUpsertUMAResolutionMarketsParams {
	params := polymarketsqlc.BatchUpsertUMAResolutionMarketsParams{
		MarketKeys:                  make([]string, 0, len(items)),
		MarketIds:                   make([]string, 0, len(items)),
		ConditionIds:                make([]string, 0, len(items)),
		Slugs:                       make([]string, 0, len(items)),
		EventIds:                    make([]string, 0, len(items)),
		EventSlugs:                  make([]string, 0, len(items)),
		Questions:                   make([]string, 0, len(items)),
		Descriptions:                make([]string, 0, len(items)),
		ResolutionSources:           make([]string, 0, len(items)),
		Images:                      make([]string, 0, len(items)),
		Icons:                       make([]string, 0, len(items)),
		UmaResolutionStatusValues:   make([]string, 0, len(items)),
		UmaResolutionStatusesValues: make([]string, 0, len(items)),
		OutcomesValues:              make([]string, 0, len(items)),
		OutcomePricesValues:         make([]string, 0, len(items)),
		ClobTokenIdsValues:          make([]string, 0, len(items)),
		ActiveValues:                make([]bool, 0, len(items)),
		ClosedValues:                make([]bool, 0, len(items)),
		ArchivedValues:              make([]bool, 0, len(items)),
		RestrictedValues:            make([]bool, 0, len(items)),
		EnableOrderBookValues:       make([]bool, 0, len(items)),
		VolumeValues:                make([]string, 0, len(items)),
		VolumeNumValues:             make([]float64, 0, len(items)),
		LiquidityNumValues:          make([]float64, 0, len(items)),
		Volume24hrValues:            make([]float64, 0, len(items)),
		Volume1wkValues:             make([]float64, 0, len(items)),
		Volume1moValues:             make([]float64, 0, len(items)),
		Volume1yrValues:             make([]float64, 0, len(items)),
		SpreadValues:                make([]float64, 0, len(items)),
		BestBidValues:               make([]float64, 0, len(items)),
		BestAskValues:               make([]float64, 0, len(items)),
		LastTradePriceValues:        make([]float64, 0, len(items)),
		TagsValues:                  make([][]byte, 0, len(items)),
		RawValues:                   make([][]byte, 0, len(items)),
		FetchedAtValues:             make([]pgtype.Timestamptz, 0, len(items)),
		LastSeenAtValues:            make([]pgtype.Timestamptz, 0, len(items)),
	}
	for _, item := range items {
		params.MarketKeys = append(params.MarketKeys, item.MarketKey)
		params.MarketIds = append(params.MarketIds, item.MarketID)
		params.ConditionIds = append(params.ConditionIds, item.ConditionID)
		params.Slugs = append(params.Slugs, item.Slug)
		params.EventIds = append(params.EventIds, item.EventID)
		params.EventSlugs = append(params.EventSlugs, item.EventSlug)
		params.Questions = append(params.Questions, item.Question)
		params.Descriptions = append(params.Descriptions, item.Description)
		params.ResolutionSources = append(params.ResolutionSources, item.ResolutionSource)
		params.Images = append(params.Images, item.Image)
		params.Icons = append(params.Icons, item.Icon)
		params.UmaResolutionStatusValues = append(params.UmaResolutionStatusValues, item.UMAResolutionStatus)
		params.UmaResolutionStatusesValues = append(params.UmaResolutionStatusesValues, item.UMAResolutionStatuses)
		params.OutcomesValues = append(params.OutcomesValues, item.Outcomes)
		params.OutcomePricesValues = append(params.OutcomePricesValues, item.OutcomePrices)
		params.ClobTokenIdsValues = append(params.ClobTokenIdsValues, item.ClobTokenIDs)
		params.ActiveValues = append(params.ActiveValues, item.Active)
		params.ClosedValues = append(params.ClosedValues, item.Closed)
		params.ArchivedValues = append(params.ArchivedValues, item.Archived)
		params.RestrictedValues = append(params.RestrictedValues, item.Restricted)
		params.EnableOrderBookValues = append(params.EnableOrderBookValues, item.EnableOrderBook)
		params.VolumeValues = append(params.VolumeValues, item.Volume)
		params.VolumeNumValues = append(params.VolumeNumValues, item.VolumeNum)
		params.LiquidityNumValues = append(params.LiquidityNumValues, item.LiquidityNum)
		params.Volume24hrValues = append(params.Volume24hrValues, item.Volume24hr)
		params.Volume1wkValues = append(params.Volume1wkValues, item.Volume1wk)
		params.Volume1moValues = append(params.Volume1moValues, item.Volume1mo)
		params.Volume1yrValues = append(params.Volume1yrValues, item.Volume1yr)
		params.SpreadValues = append(params.SpreadValues, item.Spread)
		params.BestBidValues = append(params.BestBidValues, item.BestBid)
		params.BestAskValues = append(params.BestAskValues, item.BestAsk)
		params.LastTradePriceValues = append(params.LastTradePriceValues, item.LastTradePrice)
		params.TagsValues = append(params.TagsValues, jsonBytes(item.Tags, jsonArray))
		params.RawValues = append(params.RawValues, jsonBytes(item.Raw, jsonObject))
		params.FetchedAtValues = append(params.FetchedAtValues, nullableTime(item.FetchedAt))
		params.LastSeenAtValues = append(params.LastSeenAtValues, nullableTime(item.LastSeenAt))
	}
	return params
}

func batchUpsertDisputedMarketsParams(items []DisputedMarket) polymarketsqlc.BatchUpsertUMAResolutionMarketsParams {
	return batchUpsertUMAResolutionMarketsParams(items)
}

func firstNonEmptyText(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
