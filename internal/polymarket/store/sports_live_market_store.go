package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	polymarketsqlc "github.com/useryege/athena/internal/polymarket/store/sqlc"
)

const sportsLiveSyncName = "sports_live_markets"

var (
	jsonObject = json.RawMessage(`{}`)
	jsonArray  = json.RawMessage(`[]`)
)

func (s *SQLStore) SyncSportsLiveEvents(ctx context.Context, events []SportsLiveEvent, markets []SportsLiveMarket, syncStartedAt, fetchedAt time.Time) error {
	if s == nil || s.pool == nil {
		return fmt.Errorf("polymarket postgres database is not configured")
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin sports live sync: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	queries := s.queries.WithTx(tx)
	if len(events) > 0 {
		if err := queries.BatchUpsertSportsLiveEvents(ctx, batchUpsertSportsLiveEventsParams(events)); err != nil {
			return fmt.Errorf("batch upsert sports live events: %w", err)
		}
	}
	if err := queries.SeedSportsLiveScoreAlertStates(ctx); err != nil {
		return fmt.Errorf("seed sports live score alert states: %w", err)
	}
	if len(markets) > 0 {
		if err := queries.BatchUpsertSportsLiveMarkets(ctx, batchUpsertSportsLiveMarketsParams(markets)); err != nil {
			return fmt.Errorf("batch upsert sports live markets: %w", err)
		}
	}
	if _, err := queries.DeleteSportsLiveMarketsNotSeenSince(ctx, nullableTime(syncStartedAt)); err != nil {
		return fmt.Errorf("delete stale sports live markets: %w", err)
	}
	if _, err := queries.DeleteSportsLiveEventsNotSeenSince(ctx, nullableTime(syncStartedAt)); err != nil {
		return fmt.Errorf("delete stale sports live events: %w", err)
	}
	if err := queries.UpsertPolymarketSyncState(ctx, polymarketsqlc.UpsertPolymarketSyncStateParams{
		SyncName:      sportsLiveSyncName,
		LastSuccessAt: nullableTime(fetchedAt),
	}); err != nil {
		return fmt.Errorf("update sports live sync state: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit sports live sync: %w", err)
	}
	return nil
}

func (s *SQLStore) ListSportsLiveEventCards(ctx context.Context, limit int32) ([]SportsLiveEventCard, error) {
	if s == nil || s.queries == nil {
		return nil, fmt.Errorf("polymarket postgres database is not configured")
	}
	rows, err := s.queries.ListSportsLiveEvents(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("list sports live events: %w", err)
	}
	items := make([]SportsLiveEventCard, 0, len(rows))
	eventKeys := make([]string, 0, len(rows))
	for _, row := range rows {
		items = append(items, SportsLiveEventCard{
			EventKey:       row.EventKey,
			EventID:        row.EventID,
			Slug:           row.Slug,
			Title:          row.Title,
			Image:          row.Image,
			Score:          row.Score,
			Period:         row.Period,
			Elapsed:        row.Elapsed,
			GameStatus:     row.GameStatus,
			StartTime:      timeValue(row.StartTime),
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

	marketRows, err := s.queries.ListSportsLiveMarketsByEventKeys(ctx, eventKeys)
	if err != nil {
		return nil, fmt.Errorf("list sports live markets by event keys: %w", err)
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
		items[idx].Markets = append(items[idx].Markets, SportsLiveMarketCard{
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

func (s *SQLStore) ListSportsLiveMoneylineMarketsForPriceHistory(ctx context.Context) ([]SportsLivePriceHistoryMarket, error) {
	if s == nil || s.queries == nil {
		return nil, fmt.Errorf("polymarket postgres database is not configured")
	}
	rows, err := s.queries.ListSportsLiveMoneylineMarketsForPriceHistory(ctx)
	if err != nil {
		return nil, fmt.Errorf("list sports live moneyline markets for price history: %w", err)
	}
	items := make([]SportsLivePriceHistoryMarket, 0, len(rows))
	for _, row := range rows {
		items = append(items, SportsLivePriceHistoryMarket{
			EventKey:     row.EventKey,
			MarketKey:    row.MarketKey,
			ConditionID:  row.ConditionID,
			Outcomes:     row.Outcomes,
			ClobTokenIDs: row.ClobTokenIds,
		})
	}
	return items, nil
}

func (s *SQLStore) ListSportsLiveLatestPricePointTimes(ctx context.Context, tokenIDs []string) (map[string]time.Time, error) {
	if s == nil || s.queries == nil {
		return nil, fmt.Errorf("polymarket postgres database is not configured")
	}
	rows, err := s.queries.ListSportsLiveLatestPricePointTimes(ctx, tokenIDs)
	if err != nil {
		return nil, fmt.Errorf("list sports live latest price point times: %w", err)
	}
	out := make(map[string]time.Time, len(rows))
	for _, row := range rows {
		out[row.TokenID] = timeValue(row.PriceTs)
	}
	return out, nil
}

func (s *SQLStore) BatchUpsertSportsLivePricePoints(ctx context.Context, points []SportsLivePricePoint) error {
	if s == nil || s.queries == nil {
		return fmt.Errorf("polymarket postgres database is not configured")
	}
	if len(points) == 0 {
		return nil
	}
	if err := s.queries.BatchUpsertSportsLivePricePoints(ctx, batchUpsertSportsLivePricePointsParams(points)); err != nil {
		return fmt.Errorf("batch upsert sports live price points: %w", err)
	}
	return nil
}

func (s *SQLStore) ListSportsLiveLatestPriceAlertTokens(ctx context.Context) ([]SportsLivePriceAlertToken, error) {
	if s == nil || s.queries == nil {
		return nil, fmt.Errorf("polymarket postgres database is not configured")
	}
	rows, err := s.queries.ListSportsLiveLatestPriceAlertTokens(ctx)
	if err != nil {
		return nil, fmt.Errorf("list sports live latest price alert tokens: %w", err)
	}
	items := make([]SportsLivePriceAlertToken, 0, len(rows))
	for _, row := range rows {
		priceTs := timeValue(row.PriceTs)
		if priceTs.IsZero() {
			continue
		}
		items = append(items, SportsLivePriceAlertToken{
			TokenID:       row.TokenID,
			MarketKey:     row.MarketKey,
			EventKey:      row.EventKey,
			ConditionID:   row.ConditionID,
			Outcome:       row.Outcome,
			PriceTs:       priceTs,
			Price:         row.Price,
			EventSlug:     row.EventSlug,
			EventTitle:    row.EventTitle,
			MarketTitle:   row.MarketTitle,
			Score:         row.Score,
			Period:        row.Period,
			Elapsed:       row.Elapsed,
			GameStatus:    row.GameStatus,
			Volume:        row.Volume,
			Liquidity:     row.Liquidity,
			LastAlertBand: row.LastAlertBand,
			LastAlertedAt: timeValue(row.LastAlertedAt),
		})
	}
	return items, nil
}

func (s *SQLStore) UpsertSportsLivePriceAlertState(ctx context.Context, state SportsLivePriceAlertState) error {
	if s == nil || s.queries == nil {
		return fmt.Errorf("polymarket postgres database is not configured")
	}
	if err := s.queries.UpsertSportsLivePriceAlertState(ctx, polymarketsqlc.UpsertSportsLivePriceAlertStateParams{
		TokenID:       state.TokenID,
		MarketKey:     state.MarketKey,
		EventKey:      state.EventKey,
		ConditionID:   state.ConditionID,
		Outcome:       state.Outcome,
		AlertBand:     state.AlertBand,
		LastAlertedAt: nullableTime(state.LastAlertedAt),
		LastPriceTs:   nullableTime(state.LastPriceTs),
		LastPrice:     state.LastPrice,
	}); err != nil {
		return fmt.Errorf("upsert sports live price alert state: %w", err)
	}
	return nil
}

func (s *SQLStore) DeleteSportsLivePriceAlertState(ctx context.Context, tokenID string) error {
	if s == nil || s.queries == nil {
		return fmt.Errorf("polymarket postgres database is not configured")
	}
	if err := s.queries.DeleteSportsLivePriceAlertState(ctx, tokenID); err != nil {
		return fmt.Errorf("delete sports live price alert state: %w", err)
	}
	return nil
}

func (s *SQLStore) ListSportsLiveScoreAlertCandidates(ctx context.Context) ([]SportsLiveScoreAlertCandidate, error) {
	if s == nil || s.queries == nil {
		return nil, fmt.Errorf("polymarket postgres database is not configured")
	}
	rows, err := s.queries.ListSportsLiveScoreAlertCandidates(ctx)
	if err != nil {
		return nil, fmt.Errorf("list sports live score alert candidates: %w", err)
	}
	items := make([]SportsLiveScoreAlertCandidate, 0, len(rows))
	for _, row := range rows {
		items = append(items, SportsLiveScoreAlertCandidate{
			EventKey:      row.EventKey,
			Slug:          row.Slug,
			Title:         row.Title,
			PreviousScore: row.PreviousScore,
			Score:         row.Score,
			Period:        row.Period,
			Elapsed:       row.Elapsed,
			GameStatus:    row.GameStatus,
			FetchedAt:     timeValue(row.FetchedAt),
		})
	}
	return items, nil
}

func (s *SQLStore) UpdateSportsLiveScoreAlertState(ctx context.Context, state SportsLiveScoreAlertState) error {
	if s == nil || s.queries == nil {
		return fmt.Errorf("polymarket postgres database is not configured")
	}
	if err := s.queries.UpdateSportsLiveScoreAlertState(ctx, polymarketsqlc.UpdateSportsLiveScoreAlertStateParams{
		EventKey:       state.EventKey,
		Score:          state.Score,
		NotificationID: state.NotificationID,
		LastNotifiedAt: nullableTime(state.LastNotifiedAt),
	}); err != nil {
		return fmt.Errorf("update sports live score alert state: %w", err)
	}
	return nil
}

func (s *SQLStore) ListSportsLivePriceHistorySeries(ctx context.Context, marketKeys []string, limitPerToken int32) ([]SportsLivePriceHistorySeries, error) {
	if s == nil || s.queries == nil {
		return nil, fmt.Errorf("polymarket postgres database is not configured")
	}
	if len(marketKeys) == 0 || limitPerToken <= 0 {
		return nil, nil
	}
	rows, err := s.queries.ListSportsLivePriceHistoryByMarketKeys(ctx, polymarketsqlc.ListSportsLivePriceHistoryByMarketKeysParams{
		LimitPerToken: limitPerToken,
		MarketKeys:    marketKeys,
	})
	if err != nil {
		return nil, fmt.Errorf("list sports live price history by market keys: %w", err)
	}

	items := make([]SportsLivePriceHistorySeries, 0)
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
			items = append(items, SportsLivePriceHistorySeries{
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

func batchUpsertSportsLiveEventsParams(items []SportsLiveEvent) polymarketsqlc.BatchUpsertSportsLiveEventsParams {
	params := polymarketsqlc.BatchUpsertSportsLiveEventsParams{
		EventKeys:            make([]string, 0, len(items)),
		EventIds:             make([]string, 0, len(items)),
		Tickers:              make([]string, 0, len(items)),
		Slugs:                make([]string, 0, len(items)),
		Titles:               make([]string, 0, len(items)),
		Descriptions:         make([]string, 0, len(items)),
		ResolutionSources:    make([]string, 0, len(items)),
		StartDateValues:      make([]pgtype.Timestamptz, 0, len(items)),
		CreationDateValues:   make([]pgtype.Timestamptz, 0, len(items)),
		EndDateValues:        make([]pgtype.Timestamptz, 0, len(items)),
		StartTimeValues:      make([]pgtype.Timestamptz, 0, len(items)),
		CreatedAtGammaValues: make([]pgtype.Timestamptz, 0, len(items)),
		UpdatedAtGammaValues: make([]pgtype.Timestamptz, 0, len(items)),
		Images:               make([]string, 0, len(items)),
		Icons:                make([]string, 0, len(items)),
		ActiveValues:         make([]bool, 0, len(items)),
		ClosedValues:         make([]bool, 0, len(items)),
		ArchivedValues:       make([]bool, 0, len(items)),
		FeaturedValues:       make([]bool, 0, len(items)),
		RestrictedValues:     make([]bool, 0, len(items)),
		LiveValues:           make([]bool, 0, len(items)),
		EndedValues:          make([]bool, 0, len(items)),
		LiquidityValues:      make([]float64, 0, len(items)),
		VolumeValues:         make([]float64, 0, len(items)),
		OpenInterestValues:   make([]float64, 0, len(items)),
		Categories:           make([]string, 0, len(items)),
		Scores:               make([]string, 0, len(items)),
		Periods:              make([]string, 0, len(items)),
		ElapsedValues:        make([]string, 0, len(items)),
		FinishedTimestamps:   make([]string, 0, len(items)),
		GameIDValues:         make([]int64, 0, len(items)),
		EventDates:           make([]string, 0, len(items)),
		GameStatuses:         make([]string, 0, len(items)),
		CommentCountValues:   make([]int64, 0, len(items)),
		SportValues:          make([][]byte, 0, len(items)),
		TeamsValues:          make([][]byte, 0, len(items)),
		TagsValues:           make([][]byte, 0, len(items)),
		RawValues:            make([][]byte, 0, len(items)),
		FetchedAtValues:      make([]pgtype.Timestamptz, 0, len(items)),
		LastSeenAtValues:     make([]pgtype.Timestamptz, 0, len(items)),
	}
	for _, item := range items {
		params.EventKeys = append(params.EventKeys, item.EventKey)
		params.EventIds = append(params.EventIds, item.EventID)
		params.Tickers = append(params.Tickers, item.Ticker)
		params.Slugs = append(params.Slugs, item.Slug)
		params.Titles = append(params.Titles, item.Title)
		params.Descriptions = append(params.Descriptions, item.Description)
		params.ResolutionSources = append(params.ResolutionSources, item.ResolutionSource)
		params.StartDateValues = append(params.StartDateValues, nullableTime(item.StartDate))
		params.CreationDateValues = append(params.CreationDateValues, nullableTime(item.CreationDate))
		params.EndDateValues = append(params.EndDateValues, nullableTime(item.EndDate))
		params.StartTimeValues = append(params.StartTimeValues, nullableTime(item.StartTime))
		params.CreatedAtGammaValues = append(params.CreatedAtGammaValues, nullableTime(item.CreatedAtGamma))
		params.UpdatedAtGammaValues = append(params.UpdatedAtGammaValues, nullableTime(item.UpdatedAtGamma))
		params.Images = append(params.Images, item.Image)
		params.Icons = append(params.Icons, item.Icon)
		params.ActiveValues = append(params.ActiveValues, item.Active)
		params.ClosedValues = append(params.ClosedValues, item.Closed)
		params.ArchivedValues = append(params.ArchivedValues, item.Archived)
		params.FeaturedValues = append(params.FeaturedValues, item.Featured)
		params.RestrictedValues = append(params.RestrictedValues, item.Restricted)
		params.LiveValues = append(params.LiveValues, item.Live)
		params.EndedValues = append(params.EndedValues, item.Ended)
		params.LiquidityValues = append(params.LiquidityValues, item.Liquidity)
		params.VolumeValues = append(params.VolumeValues, item.Volume)
		params.OpenInterestValues = append(params.OpenInterestValues, item.OpenInterest)
		params.Categories = append(params.Categories, item.Category)
		params.Scores = append(params.Scores, item.Score)
		params.Periods = append(params.Periods, item.Period)
		params.ElapsedValues = append(params.ElapsedValues, item.Elapsed)
		params.FinishedTimestamps = append(params.FinishedTimestamps, item.FinishedTimestamp)
		params.GameIDValues = append(params.GameIDValues, item.GameID)
		params.EventDates = append(params.EventDates, item.EventDate)
		params.GameStatuses = append(params.GameStatuses, item.GameStatus)
		params.CommentCountValues = append(params.CommentCountValues, item.CommentCount)
		params.SportValues = append(params.SportValues, jsonBytes(item.Sport, jsonObject))
		params.TeamsValues = append(params.TeamsValues, jsonBytes(item.Teams, jsonArray))
		params.TagsValues = append(params.TagsValues, jsonBytes(item.Tags, jsonArray))
		params.RawValues = append(params.RawValues, jsonBytes(item.Raw, jsonObject))
		params.FetchedAtValues = append(params.FetchedAtValues, nullableTime(item.FetchedAt))
		params.LastSeenAtValues = append(params.LastSeenAtValues, nullableTime(item.LastSeenAt))
	}
	return params
}

func batchUpsertSportsLiveMarketsParams(items []SportsLiveMarket) polymarketsqlc.BatchUpsertSportsLiveMarketsParams {
	params := polymarketsqlc.BatchUpsertSportsLiveMarketsParams{
		MarketKeys:            make([]string, 0, len(items)),
		EventKeys:             make([]string, 0, len(items)),
		EventIds:              make([]string, 0, len(items)),
		EventSlugs:            make([]string, 0, len(items)),
		MarketIds:             make([]string, 0, len(items)),
		ConditionIds:          make([]string, 0, len(items)),
		Slugs:                 make([]string, 0, len(items)),
		Questions:             make([]string, 0, len(items)),
		Titles:                make([]string, 0, len(items)),
		Descriptions:          make([]string, 0, len(items)),
		ResolutionSources:     make([]string, 0, len(items)),
		SportsMarketTypes:     make([]string, 0, len(items)),
		GroupItemTitles:       make([]string, 0, len(items)),
		Images:                make([]string, 0, len(items)),
		Icons:                 make([]string, 0, len(items)),
		OutcomesValues:        make([]string, 0, len(items)),
		OutcomePricesValues:   make([]string, 0, len(items)),
		ClobTokenIdsValues:    make([]string, 0, len(items)),
		ActiveValues:          make([]bool, 0, len(items)),
		ClosedValues:          make([]bool, 0, len(items)),
		ArchivedValues:        make([]bool, 0, len(items)),
		RestrictedValues:      make([]bool, 0, len(items)),
		EnableOrderBookValues: make([]bool, 0, len(items)),
		VolumeValues:          make([]string, 0, len(items)),
		VolumeNumValues:       make([]float64, 0, len(items)),
		LiquidityNumValues:    make([]float64, 0, len(items)),
		Volume24hrValues:      make([]float64, 0, len(items)),
		Volume1wkValues:       make([]float64, 0, len(items)),
		Volume1moValues:       make([]float64, 0, len(items)),
		Volume1yrValues:       make([]float64, 0, len(items)),
		SpreadValues:          make([]float64, 0, len(items)),
		BestBidValues:         make([]float64, 0, len(items)),
		BestAskValues:         make([]float64, 0, len(items)),
		LastTradePriceValues:  make([]float64, 0, len(items)),
		StartDateValues:       make([]pgtype.Timestamptz, 0, len(items)),
		EndDateValues:         make([]pgtype.Timestamptz, 0, len(items)),
		CreatedAtGammaValues:  make([]pgtype.Timestamptz, 0, len(items)),
		UpdatedAtGammaValues:  make([]pgtype.Timestamptz, 0, len(items)),
		TagsValues:            make([][]byte, 0, len(items)),
		RawValues:             make([][]byte, 0, len(items)),
		FetchedAtValues:       make([]pgtype.Timestamptz, 0, len(items)),
		LastSeenAtValues:      make([]pgtype.Timestamptz, 0, len(items)),
	}
	for _, item := range items {
		params.MarketKeys = append(params.MarketKeys, item.MarketKey)
		params.EventKeys = append(params.EventKeys, item.EventKey)
		params.EventIds = append(params.EventIds, item.EventID)
		params.EventSlugs = append(params.EventSlugs, item.EventSlug)
		params.MarketIds = append(params.MarketIds, item.MarketID)
		params.ConditionIds = append(params.ConditionIds, item.ConditionID)
		params.Slugs = append(params.Slugs, item.Slug)
		params.Questions = append(params.Questions, item.Question)
		params.Titles = append(params.Titles, item.Title)
		params.Descriptions = append(params.Descriptions, item.Description)
		params.ResolutionSources = append(params.ResolutionSources, item.ResolutionSource)
		params.SportsMarketTypes = append(params.SportsMarketTypes, item.SportsMarketType)
		params.GroupItemTitles = append(params.GroupItemTitles, item.GroupItemTitle)
		params.Images = append(params.Images, item.Image)
		params.Icons = append(params.Icons, item.Icon)
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
		params.StartDateValues = append(params.StartDateValues, nullableTime(item.StartDate))
		params.EndDateValues = append(params.EndDateValues, nullableTime(item.EndDate))
		params.CreatedAtGammaValues = append(params.CreatedAtGammaValues, nullableTime(item.CreatedAtGamma))
		params.UpdatedAtGammaValues = append(params.UpdatedAtGammaValues, nullableTime(item.UpdatedAtGamma))
		params.TagsValues = append(params.TagsValues, jsonBytes(item.Tags, jsonArray))
		params.RawValues = append(params.RawValues, jsonBytes(item.Raw, jsonObject))
		params.FetchedAtValues = append(params.FetchedAtValues, nullableTime(item.FetchedAt))
		params.LastSeenAtValues = append(params.LastSeenAtValues, nullableTime(item.LastSeenAt))
	}
	return params
}

func batchUpsertSportsLivePricePointsParams(items []SportsLivePricePoint) polymarketsqlc.BatchUpsertSportsLivePricePointsParams {
	params := polymarketsqlc.BatchUpsertSportsLivePricePointsParams{
		TokenIds:        make([]string, 0, len(items)),
		MarketKeys:      make([]string, 0, len(items)),
		EventKeys:       make([]string, 0, len(items)),
		ConditionIds:    make([]string, 0, len(items)),
		Outcomes:        make([]string, 0, len(items)),
		PriceTsValues:   make([]pgtype.Timestamptz, 0, len(items)),
		PriceValues:     make([]float64, 0, len(items)),
		FetchedAtValues: make([]pgtype.Timestamptz, 0, len(items)),
	}
	for _, item := range items {
		params.TokenIds = append(params.TokenIds, item.TokenID)
		params.MarketKeys = append(params.MarketKeys, item.MarketKey)
		params.EventKeys = append(params.EventKeys, item.EventKey)
		params.ConditionIds = append(params.ConditionIds, item.ConditionID)
		params.Outcomes = append(params.Outcomes, item.Outcome)
		params.PriceTsValues = append(params.PriceTsValues, nullableTime(item.PriceTs))
		params.PriceValues = append(params.PriceValues, item.Price)
		params.FetchedAtValues = append(params.FetchedAtValues, nullableTime(item.FetchedAt))
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

func jsonBytes(value json.RawMessage, fallback json.RawMessage) []byte {
	if json.Valid(value) {
		return []byte(value)
	}
	return []byte(fallback)
}
