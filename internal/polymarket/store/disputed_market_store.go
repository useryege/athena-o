package store

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	polymarketsqlc "github.com/useryege/athena/internal/polymarket/store/sqlc"
)

const disputedMarketsSyncName = "disputed_markets"

func (s *SQLStore) SyncDisputedMarkets(ctx context.Context, markets []DisputedMarket, syncStartedAt, fetchedAt time.Time) error {
	if s == nil || s.pool == nil {
		return fmt.Errorf("polymarket postgres database is not configured")
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin disputed markets sync: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	queries := s.queries.WithTx(tx)
	if len(markets) > 0 {
		if err := queries.BatchUpsertDisputedMarkets(ctx, batchUpsertDisputedMarketsParams(markets)); err != nil {
			return fmt.Errorf("batch upsert disputed markets: %w", err)
		}
	}
	if _, err := queries.DeleteDisputedMarketsNotSeenSince(ctx, nullableTime(syncStartedAt)); err != nil {
		return fmt.Errorf("delete stale disputed markets: %w", err)
	}
	if err := queries.UpsertPolymarketSyncState(ctx, polymarketsqlc.UpsertPolymarketSyncStateParams{
		SyncName:      disputedMarketsSyncName,
		LastSuccessAt: nullableTime(fetchedAt),
	}); err != nil {
		return fmt.Errorf("update disputed markets sync state: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit disputed markets sync: %w", err)
	}
	return nil
}

func batchUpsertDisputedMarketsParams(items []DisputedMarket) polymarketsqlc.BatchUpsertDisputedMarketsParams {
	params := polymarketsqlc.BatchUpsertDisputedMarketsParams{
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
