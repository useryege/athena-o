package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	sportshistorysqlc "github.com/useryege/athena/internal/sportshistory/store/sqlc"
)

const sportsHistorySyncName = "sports_history"

func (s *SQLStore) SyncSportsHistory(ctx context.Context, events []SportsHistoryEvent, markets []SportsHistoryMarket, syncStartedAt, fetchedAt time.Time) error {
	if s == nil || s.pool == nil {
		return fmt.Errorf("sports history postgres database is not configured")
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin sports history sync: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	queries := s.queries.WithTx(tx)
	if len(events) > 0 {
		if err := queries.BatchUpsertSportsHistoryEvents(ctx, batchUpsertSportsHistoryEventsParams(events)); err != nil {
			return fmt.Errorf("batch upsert sports history events: %w", err)
		}
	}
	if len(markets) > 0 {
		if err := queries.BatchUpsertSportsHistoryMarkets(ctx, batchUpsertSportsHistoryMarketsParams(markets)); err != nil {
			return fmt.Errorf("batch upsert sports history markets: %w", err)
		}
	}
	if _, err := queries.DeleteSportsHistoryMarketsNotSeenSince(ctx, nullableTime(syncStartedAt)); err != nil {
		return fmt.Errorf("delete stale sports history markets: %w", err)
	}
	if _, err := queries.DeleteSportsHistoryEventsNotSeenSince(ctx, nullableTime(syncStartedAt)); err != nil {
		return fmt.Errorf("delete stale sports history events: %w", err)
	}
	if err := queries.UpsertSportsHistorySyncState(ctx, sportshistorysqlc.UpsertSportsHistorySyncStateParams{
		SyncName:      sportsHistorySyncName,
		LastSuccessAt: nullableTime(fetchedAt),
	}); err != nil {
		return fmt.Errorf("update sports history sync state: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit sports history sync: %w", err)
	}
	return nil
}

func (s *SQLStore) ListSportsHistoryEventCards(ctx context.Context, limit int32) ([]SportsHistoryEventCard, error) {
	if s == nil || s.queries == nil {
		return nil, fmt.Errorf("sports history postgres database is not configured")
	}
	rows, err := s.queries.ListSportsHistoryEvents(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("list sports history events: %w", err)
	}
	items := make([]SportsHistoryEventCard, 0, len(rows))
	eventKeys := make([]string, 0, len(rows))
	for _, row := range rows {
		items = append(items, SportsHistoryEventCard{
			EventKey:       row.EventKey,
			EventID:        row.EventID,
			League:         row.League,
			Slug:           row.Slug,
			Title:          row.Title,
			Image:          row.Image,
			Score:          row.Score,
			Period:         row.Period,
			Elapsed:        row.Elapsed,
			GameStatus:     row.GameStatus,
			StartTime:      timeValue(row.StartTime),
			FinishedAt:     timeValue(row.FinishedAt),
			UpdatedAtGamma: timeValue(row.UpdatedAtGamma),
			Liquidity:      row.Liquidity,
			Volume:         row.Volume,
			MarketCount:    row.MarketCount,
			Teams:          jsonBytes(row.Teams, jsonArray),
			FetchedAt:      timeValue(row.FetchedAt),
			LastSeenAt:     timeValue(row.LastSeenAt),
		})
		eventKeys = append(eventKeys, row.EventKey)
	}
	if len(eventKeys) == 0 {
		return items, nil
	}

	marketRows, err := s.queries.ListSportsHistoryMarketsByEventKeys(ctx, eventKeys)
	if err != nil {
		return nil, fmt.Errorf("list sports history markets by event keys: %w", err)
	}
	indexByEventKey := make(map[string]int, len(items))
	for i := range items {
		indexByEventKey[items[i].EventKey] = i
	}
	for _, row := range marketRows {
		idx, ok := indexByEventKey[row.EventKey]
		if !ok {
			continue
		}
		items[idx].Markets = append(items[idx].Markets, SportsHistoryMarketCard{
			EventKey:         row.EventKey,
			MarketKey:        row.MarketKey,
			ConditionID:      row.ConditionID,
			Slug:             row.Slug,
			SportsMarketType: row.SportsMarketType,
			Question:         row.Question,
			Outcomes:         row.Outcomes,
			OutcomePrices:    row.OutcomePrices,
			BestBid:          row.BestBid,
			BestAsk:          row.BestAsk,
			LastTradePrice:   row.LastTradePrice,
			Spread:           row.Spread,
			LiquidityNum:     row.LiquidityNum,
			VolumeNum:        row.VolumeNum,
			UpdatedAtGamma:   timeValue(row.UpdatedAtGamma),
		})
	}
	return items, nil
}

func (s *SQLStore) GetSportsHistoryLastSuccessAt(ctx context.Context) (time.Time, error) {
	if s == nil || s.queries == nil {
		return time.Time{}, fmt.Errorf("sports history postgres database is not configured")
	}
	value, err := s.queries.GetSportsHistorySyncState(ctx, sportsHistorySyncName)
	if errors.Is(err, pgx.ErrNoRows) {
		return time.Time{}, nil
	}
	if err != nil {
		return time.Time{}, fmt.Errorf("get sports history sync state: %w", err)
	}
	return timeValue(value), nil
}

func (s *SQLStore) ListSportsHistoryMoneylineMarketsForPriceHistory(ctx context.Context) ([]SportsHistoryPriceHistoryMarket, error) {
	if s == nil || s.queries == nil {
		return nil, fmt.Errorf("sports history postgres database is not configured")
	}
	rows, err := s.queries.ListSportsHistoryMoneylineMarketsForPriceHistory(ctx)
	if err != nil {
		return nil, fmt.Errorf("list sports history moneyline markets for price history: %w", err)
	}
	items := make([]SportsHistoryPriceHistoryMarket, 0, len(rows))
	for _, row := range rows {
		items = append(items, SportsHistoryPriceHistoryMarket{
			EventKey:     row.EventKey,
			MarketKey:    row.MarketKey,
			ConditionID:  row.ConditionID,
			Outcomes:     row.Outcomes,
			ClobTokenIDs: row.ClobTokenIds,
			StartTime:    timeValue(row.StartTime),
			FinishedAt:   timeValue(row.FinishedAt),
		})
	}
	return items, nil
}

func (s *SQLStore) BatchUpsertSportsHistoryPricePoints(ctx context.Context, points []SportsHistoryPricePoint) error {
	if s == nil || s.queries == nil {
		return fmt.Errorf("sports history postgres database is not configured")
	}
	if len(points) == 0 {
		return nil
	}
	params := sportshistorysqlc.BatchUpsertSportsHistoryPricePointsParams{
		TokenIds:        make([]string, 0, len(points)),
		MarketKeys:      make([]string, 0, len(points)),
		EventKeys:       make([]string, 0, len(points)),
		ConditionIds:    make([]string, 0, len(points)),
		Outcomes:        make([]string, 0, len(points)),
		PriceTsValues:   make([]pgtype.Timestamptz, 0, len(points)),
		PriceValues:     make([]float64, 0, len(points)),
		FetchedAtValues: make([]pgtype.Timestamptz, 0, len(points)),
	}
	for _, point := range points {
		params.TokenIds = append(params.TokenIds, point.TokenID)
		params.MarketKeys = append(params.MarketKeys, point.MarketKey)
		params.EventKeys = append(params.EventKeys, point.EventKey)
		params.ConditionIds = append(params.ConditionIds, point.ConditionID)
		params.Outcomes = append(params.Outcomes, point.Outcome)
		params.PriceTsValues = append(params.PriceTsValues, nullableTime(point.PriceTs))
		params.PriceValues = append(params.PriceValues, point.Price)
		params.FetchedAtValues = append(params.FetchedAtValues, nullableTime(point.FetchedAt))
	}
	if err := s.queries.BatchUpsertSportsHistoryPricePoints(ctx, params); err != nil {
		return fmt.Errorf("batch upsert sports history price points: %w", err)
	}
	return nil
}

func (s *SQLStore) ListSportsHistoryPriceHistorySeries(ctx context.Context, marketKeys []string, limitPerToken int32) ([]SportsHistoryPriceHistorySeries, error) {
	if s == nil || s.queries == nil {
		return nil, fmt.Errorf("sports history postgres database is not configured")
	}
	if len(marketKeys) == 0 || limitPerToken <= 0 {
		return nil, nil
	}
	rows, err := s.queries.ListSportsHistoryPriceHistoryByMarketKeys(ctx, sportshistorysqlc.ListSportsHistoryPriceHistoryByMarketKeysParams{
		LimitPerToken: limitPerToken,
		MarketKeys:    marketKeys,
	})
	if err != nil {
		return nil, fmt.Errorf("list sports history price history by market keys: %w", err)
	}
	items := make([]SportsHistoryPriceHistorySeries, 0)
	indexByKey := make(map[string]int)
	for _, row := range rows {
		priceTs := timeValue(row.PriceTs)
		if priceTs.IsZero() {
			continue
		}
		key := row.MarketKey + "\x00" + row.TokenID + "\x00" + row.Outcome
		idx, ok := indexByKey[key]
		if !ok {
			idx = len(items)
			indexByKey[key] = idx
			items = append(items, SportsHistoryPriceHistorySeries{
				MarketKey:  row.MarketKey,
				TokenID:    row.TokenID,
				Outcome:    row.Outcome,
				Timestamps: make([]int64, 0, limitPerToken),
				Prices:     make([]float64, 0, limitPerToken),
			})
		}
		items[idx].Timestamps = append(items[idx].Timestamps, priceTs.Unix())
		items[idx].Prices = append(items[idx].Prices, row.Price)
	}
	return items, nil
}

func batchUpsertSportsHistoryEventsParams(items []SportsHistoryEvent) sportshistorysqlc.BatchUpsertSportsHistoryEventsParams {
	params := sportshistorysqlc.BatchUpsertSportsHistoryEventsParams{
		EventKeys:            make([]string, 0, len(items)),
		EventIds:             make([]string, 0, len(items)),
		Leagues:              make([]string, 0, len(items)),
		Slugs:                make([]string, 0, len(items)),
		Titles:               make([]string, 0, len(items)),
		Images:               make([]string, 0, len(items)),
		Icons:                make([]string, 0, len(items)),
		Scores:               make([]string, 0, len(items)),
		Periods:              make([]string, 0, len(items)),
		ElapsedValues:        make([]string, 0, len(items)),
		GameStatuses:         make([]string, 0, len(items)),
		StartTimeValues:      make([]pgtype.Timestamptz, 0, len(items)),
		FinishedAtValues:     make([]pgtype.Timestamptz, 0, len(items)),
		UpdatedAtGammaValues: make([]pgtype.Timestamptz, 0, len(items)),
		LiquidityValues:      make([]float64, 0, len(items)),
		VolumeValues:         make([]float64, 0, len(items)),
		TeamsValues:          make([][]byte, 0, len(items)),
		RawValues:            make([][]byte, 0, len(items)),
		FetchedAtValues:      make([]pgtype.Timestamptz, 0, len(items)),
		LastSeenAtValues:     make([]pgtype.Timestamptz, 0, len(items)),
	}
	for _, item := range items {
		params.EventKeys = append(params.EventKeys, item.EventKey)
		params.EventIds = append(params.EventIds, item.EventID)
		params.Leagues = append(params.Leagues, item.League)
		params.Slugs = append(params.Slugs, item.Slug)
		params.Titles = append(params.Titles, item.Title)
		params.Images = append(params.Images, item.Image)
		params.Icons = append(params.Icons, item.Icon)
		params.Scores = append(params.Scores, item.Score)
		params.Periods = append(params.Periods, item.Period)
		params.ElapsedValues = append(params.ElapsedValues, item.Elapsed)
		params.GameStatuses = append(params.GameStatuses, item.GameStatus)
		params.StartTimeValues = append(params.StartTimeValues, nullableTime(item.StartTime))
		params.FinishedAtValues = append(params.FinishedAtValues, nullableTime(item.FinishedAt))
		params.UpdatedAtGammaValues = append(params.UpdatedAtGammaValues, nullableTime(item.UpdatedAtGamma))
		params.LiquidityValues = append(params.LiquidityValues, item.Liquidity)
		params.VolumeValues = append(params.VolumeValues, item.Volume)
		params.TeamsValues = append(params.TeamsValues, jsonBytes(item.Teams, jsonArray))
		params.RawValues = append(params.RawValues, jsonBytes(item.Raw, jsonObject))
		params.FetchedAtValues = append(params.FetchedAtValues, nullableTime(item.FetchedAt))
		params.LastSeenAtValues = append(params.LastSeenAtValues, nullableTime(item.LastSeenAt))
	}
	return params
}

func batchUpsertSportsHistoryMarketsParams(items []SportsHistoryMarket) sportshistorysqlc.BatchUpsertSportsHistoryMarketsParams {
	params := sportshistorysqlc.BatchUpsertSportsHistoryMarketsParams{
		MarketKeys:           make([]string, 0, len(items)),
		EventKeys:            make([]string, 0, len(items)),
		ConditionIds:         make([]string, 0, len(items)),
		Slugs:                make([]string, 0, len(items)),
		Questions:            make([]string, 0, len(items)),
		SportsMarketTypes:    make([]string, 0, len(items)),
		OutcomesValues:       make([]string, 0, len(items)),
		OutcomePricesValues:  make([]string, 0, len(items)),
		ClobTokenIdsValues:   make([]string, 0, len(items)),
		BestBidValues:        make([]float64, 0, len(items)),
		BestAskValues:        make([]float64, 0, len(items)),
		LastTradePriceValues: make([]float64, 0, len(items)),
		SpreadValues:         make([]float64, 0, len(items)),
		LiquidityNumValues:   make([]float64, 0, len(items)),
		VolumeNumValues:      make([]float64, 0, len(items)),
		UpdatedAtGammaValues: make([]pgtype.Timestamptz, 0, len(items)),
		RawValues:            make([][]byte, 0, len(items)),
		FetchedAtValues:      make([]pgtype.Timestamptz, 0, len(items)),
		LastSeenAtValues:     make([]pgtype.Timestamptz, 0, len(items)),
	}
	for _, item := range items {
		params.MarketKeys = append(params.MarketKeys, item.MarketKey)
		params.EventKeys = append(params.EventKeys, item.EventKey)
		params.ConditionIds = append(params.ConditionIds, item.ConditionID)
		params.Slugs = append(params.Slugs, item.Slug)
		params.Questions = append(params.Questions, item.Question)
		params.SportsMarketTypes = append(params.SportsMarketTypes, item.SportsMarketType)
		params.OutcomesValues = append(params.OutcomesValues, item.Outcomes)
		params.OutcomePricesValues = append(params.OutcomePricesValues, item.OutcomePrices)
		params.ClobTokenIdsValues = append(params.ClobTokenIdsValues, item.ClobTokenIDs)
		params.BestBidValues = append(params.BestBidValues, item.BestBid)
		params.BestAskValues = append(params.BestAskValues, item.BestAsk)
		params.LastTradePriceValues = append(params.LastTradePriceValues, item.LastTradePrice)
		params.SpreadValues = append(params.SpreadValues, item.Spread)
		params.LiquidityNumValues = append(params.LiquidityNumValues, item.LiquidityNum)
		params.VolumeNumValues = append(params.VolumeNumValues, item.VolumeNum)
		params.UpdatedAtGammaValues = append(params.UpdatedAtGammaValues, nullableTime(item.UpdatedAtGamma))
		params.RawValues = append(params.RawValues, jsonBytes(item.Raw, jsonObject))
		params.FetchedAtValues = append(params.FetchedAtValues, nullableTime(item.FetchedAt))
		params.LastSeenAtValues = append(params.LastSeenAtValues, nullableTime(item.LastSeenAt))
	}
	return params
}
