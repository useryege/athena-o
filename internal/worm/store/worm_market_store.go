package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	wormsqlc "github.com/useryege/athena/internal/worm/store/sqlc"
)

func (s *SQLStore) UpsertWormMarket(ctx context.Context, item WormMarket) (*WormMarket, error) {
	q, err := s.querier()
	if err != nil {
		return nil, err
	}
	params := upsertWormMarketParams(item)
	row, err := q.UpsertWormMarket(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("upsert worm market: %w", err)
	}
	return mapWormMarket(row), nil
}

func (s *SQLStore) BatchUpsertWormMarkets(ctx context.Context, items []WormMarket) error {
	if len(items) == 0 {
		return nil
	}
	q, err := s.querier()
	if err != nil {
		return err
	}
	if err := q.BatchUpsertWormMarkets(ctx, batchUpsertWormMarketsParams(items)); err != nil {
		return fmt.Errorf("batch upsert worm markets: %w", err)
	}
	return nil
}

func (s *SQLStore) BatchInsertWormMarketPriceHistory(ctx context.Context, samples []WormMarketPriceSample) error {
	if len(samples) == 0 {
		return nil
	}
	q, err := s.querier()
	if err != nil {
		return err
	}
	params := wormsqlc.BatchInsertWormMarketPriceHistoryParams{
		ConditionIds:    make([]string, 0, len(samples)),
		Prices:          make([]string, 0, len(samples)),
		SampledAtValues: make([]pgtype.Timestamptz, 0, len(samples)),
	}
	for _, sample := range samples {
		params.ConditionIds = append(params.ConditionIds, sample.ConditionID)
		params.Prices = append(params.Prices, sample.Price)
		params.SampledAtValues = append(params.SampledAtValues, nullableTime(sample.SampledAt))
	}
	if err := q.BatchInsertWormMarketPriceHistory(ctx, params); err != nil {
		return fmt.Errorf("batch insert worm market price history: %w", err)
	}
	return nil
}

func (s *SQLStore) DeleteWormMarketPriceHistoryBefore(ctx context.Context, sampledAt time.Time) (int64, error) {
	q, err := s.querier()
	if err != nil {
		return 0, err
	}
	rowsAffected, err := q.DeleteWormMarketPriceHistoryBefore(ctx, nullableTime(sampledAt))
	if err != nil {
		return 0, fmt.Errorf("delete worm market price history before %s: %w", sampledAt.Format(time.RFC3339), err)
	}
	return rowsAffected, nil
}

func (s *SQLStore) ListWormMarketLivePriceChanges(ctx context.Context, sampledAt time.Time) ([]WormMarketLivePriceChange, error) {
	q, err := s.querier()
	if err != nil {
		return nil, err
	}
	rows, err := q.ListWormMarketLivePriceChanges(ctx, nullableTime(sampledAt))
	if err != nil {
		return nil, fmt.Errorf("list worm market live price changes: %w", err)
	}
	items := make([]WormMarketLivePriceChange, 0, len(rows))
	for _, row := range rows {
		items = append(items, WormMarketLivePriceChange{
			ConditionID: row.ConditionID,
			SampleCount: row.SampleCount,
			PriceChange: row.PriceChange,
			IsLive:      row.IsLive,
		})
	}
	return items, nil
}

func (s *SQLStore) GetWormMarket(ctx context.Context, conditionID string) (*WormMarket, error) {
	q, err := s.querier()
	if err != nil {
		return nil, err
	}
	row, err := q.GetWormMarket(ctx, conditionID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get worm market: %w", err)
	}
	return mapWormMarket(row), nil
}

func (s *SQLStore) ListWormMarkets(ctx context.Context) ([]WormMarket, error) {
	q, err := s.querier()
	if err != nil {
		return nil, err
	}
	rows, err := q.ListWormMarkets(ctx)
	if err != nil {
		return nil, fmt.Errorf("list worm markets: %w", err)
	}
	return mapWormMarkets(rows), nil
}

func (s *SQLStore) ListWormEventsPage(ctx context.Context, page, pageSize int32) (*WormEventPage, error) {
	q, err := s.querier()
	if err != nil {
		return nil, err
	}
	page, pageSize, offset := normalizePage(page, pageSize)
	total, err := q.CountWormEvents(ctx)
	if err != nil {
		return nil, fmt.Errorf("count worm events: %w", err)
	}
	eventRows, err := q.ListWormEventsPage(ctx, wormsqlc.ListWormEventsPageParams{
		Offset: offset,
		Limit:  pageSize,
	})
	if err != nil {
		return nil, fmt.Errorf("list worm events page: %w", err)
	}
	eventIDs := make([]string, 0, len(eventRows))
	for _, row := range eventRows {
		eventIDs = append(eventIDs, row.EventConditionID)
	}
	marketsByEvent := make(map[string][]WormMarket, len(eventIDs))
	if len(eventIDs) > 0 {
		marketRows, err := q.ListWormMarketsByEventConditionIDs(ctx, eventIDs)
		if err != nil {
			return nil, fmt.Errorf("list worm event markets: %w", err)
		}
		for _, market := range mapWormMarkets(marketRows) {
			marketsByEvent[market.EventConditionID] = append(marketsByEvent[market.EventConditionID], market)
		}
	}
	items := make([]WormEvent, 0, len(eventRows))
	for _, row := range eventRows {
		items = append(items, WormEvent{
			ConditionID: row.EventConditionID,
			Title:       row.EventTitle,
			Logo:        row.EventLogo,
			Live:        row.Live,
			MarketCount: row.MarketCount,
			Markets:     marketsByEvent[row.EventConditionID],
			NewestAt:    row.NewestCreated,
			FetchedAt:   timeValue(row.FetchedAt),
		})
	}
	return &WormEventPage{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func (s *SQLStore) ListWormMarketsPendingGetMarket(ctx context.Context) ([]WormMarket, error) {
	q, err := s.querier()
	if err != nil {
		return nil, err
	}
	rows, err := q.ListWormMarketsPendingGetMarket(ctx)
	if err != nil {
		return nil, fmt.Errorf("list worm markets pending get market: %w", err)
	}
	return mapWormMarkets(rows), nil
}

func (s *SQLStore) UpdateWormMarketGetMarketData(ctx context.Context, conditionID string, data json.RawMessage) (int64, error) {
	q, err := s.querier()
	if err != nil {
		return 0, err
	}
	rowsAffected, err := q.UpdateWormMarketGetMarketData(ctx, wormsqlc.UpdateWormMarketGetMarketDataParams{
		ConditionID:   conditionID,
		GetMarketData: []byte(data),
	})
	if err != nil {
		return 0, fmt.Errorf("update worm market get market data: %w", err)
	}
	return rowsAffected, nil
}

func (s *SQLStore) UpdateWormMarket(ctx context.Context, item WormMarket) (*WormMarket, error) {
	q, err := s.querier()
	if err != nil {
		return nil, err
	}
	params := updateWormMarketParams(item)
	row, err := q.UpdateWormMarket(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("update worm market: %w", err)
	}
	return mapWormMarket(row), nil
}

func (s *SQLStore) UpdateWormMarketLiveState(ctx context.Context, item WormMarket) (*WormMarket, error) {
	q, err := s.querier()
	if err != nil {
		return nil, err
	}
	if item.LiveState == "" {
		item.LiveState = "unknown"
	}
	row, err := q.UpdateWormMarketLiveState(ctx, wormsqlc.UpdateWormMarketLiveStateParams{
		ConditionID:     item.ConditionID,
		LiveState:       item.LiveState,
		LiveCheckedAt:   nullableTime(item.LiveCheckedAt),
		LivePriceChange: item.LivePriceChange,
	})
	if err != nil {
		return nil, fmt.Errorf("update worm market live state: %w", err)
	}
	return mapWormMarket(row), nil
}

func (s *SQLStore) DeleteWormMarket(ctx context.Context, conditionID string) (int64, error) {
	q, err := s.querier()
	if err != nil {
		return 0, err
	}
	rowsAffected, err := q.DeleteWormMarket(ctx, conditionID)
	if err != nil {
		return 0, fmt.Errorf("delete worm market: %w", err)
	}
	return rowsAffected, nil
}

func (s *SQLStore) DeleteWormMarketsNotSeenSince(ctx context.Context, lastSeenAt time.Time) (int64, error) {
	q, err := s.querier()
	if err != nil {
		return 0, err
	}
	rowsAffected, err := q.DeleteWormMarketsNotSeenSince(ctx, nullableTime(lastSeenAt))
	if err != nil {
		return 0, fmt.Errorf("delete stale worm markets: %w", err)
	}
	return rowsAffected, nil
}

func upsertWormMarketParams(item WormMarket) wormsqlc.UpsertWormMarketParams {
	if item.FetchedAt.IsZero() {
		item.FetchedAt = time.Now().UTC()
	}
	if item.LastSeenAt.IsZero() {
		item.LastSeenAt = item.FetchedAt
	}
	if item.LiveState == "" {
		item.LiveState = "unknown"
	}
	if len(item.Raw) == 0 {
		item.Raw = []byte("{}")
	}
	return wormsqlc.UpsertWormMarketParams{
		ConditionID:      item.ConditionID,
		Title:            item.Title,
		Description:      item.Description,
		Logo:             item.Logo,
		LastTradePrice:   item.LastTradePrice,
		State:            item.State,
		Category:         item.Category,
		SortOption:       item.SortOption,
		Created:          item.Created,
		EventTitle:       item.EventTitle,
		EventConditionID: item.EventConditionID,
		EventLogo:        item.EventLogo,
		MarginEnabled:    item.MarginEnabled,
		LiveState:        item.LiveState,
		LiveCheckedAt:    nullableTime(item.LiveCheckedAt),
		LivePriceChange:  item.LivePriceChange,
		Raw:              []byte(item.Raw),
		FetchedAt:        nullableTime(item.FetchedAt),
		LastSeenAt:       nullableTime(item.LastSeenAt),
	}
}

func updateWormMarketParams(item WormMarket) wormsqlc.UpdateWormMarketParams {
	params := upsertWormMarketParams(item)
	return wormsqlc.UpdateWormMarketParams{
		ConditionID:      params.ConditionID,
		Title:            params.Title,
		Description:      params.Description,
		Logo:             params.Logo,
		LastTradePrice:   params.LastTradePrice,
		State:            params.State,
		Category:         params.Category,
		SortOption:       params.SortOption,
		Created:          params.Created,
		EventTitle:       params.EventTitle,
		EventConditionID: params.EventConditionID,
		EventLogo:        params.EventLogo,
		MarginEnabled:    params.MarginEnabled,
		LiveState:        params.LiveState,
		LiveCheckedAt:    params.LiveCheckedAt,
		LivePriceChange:  params.LivePriceChange,
		Raw:              params.Raw,
		FetchedAt:        params.FetchedAt,
		LastSeenAt:       params.LastSeenAt,
	}
}

func batchUpsertWormMarketsParams(items []WormMarket) wormsqlc.BatchUpsertWormMarketsParams {
	now := time.Now().UTC()
	params := wormsqlc.BatchUpsertWormMarketsParams{
		ConditionIds:        make([]string, 0, len(items)),
		Titles:              make([]string, 0, len(items)),
		Descriptions:        make([]string, 0, len(items)),
		Logos:               make([]string, 0, len(items)),
		LastTradePrices:     make([]string, 0, len(items)),
		States:              make([]string, 0, len(items)),
		Categories:          make([]string, 0, len(items)),
		SortOptions:         make([]string, 0, len(items)),
		CreatedValues:       make([]int64, 0, len(items)),
		EventTitles:         make([]string, 0, len(items)),
		EventConditionIds:   make([]string, 0, len(items)),
		EventLogos:          make([]string, 0, len(items)),
		MarginEnabledValues: make([]bool, 0, len(items)),
		LiveStates:          make([]string, 0, len(items)),
		LiveCheckedAtValues: make([]pgtype.Timestamptz, 0, len(items)),
		LivePriceChanges:    make([]string, 0, len(items)),
		RawValues:           make([][]byte, 0, len(items)),
		FetchedAtValues:     make([]pgtype.Timestamptz, 0, len(items)),
		LastSeenAtValues:    make([]pgtype.Timestamptz, 0, len(items)),
	}
	for _, item := range items {
		if item.FetchedAt.IsZero() {
			item.FetchedAt = now
		}
		if item.LastSeenAt.IsZero() {
			item.LastSeenAt = item.FetchedAt
		}
		if item.LiveState == "" {
			item.LiveState = "unknown"
		}
		if len(item.Raw) == 0 {
			item.Raw = []byte("{}")
		}
		params.ConditionIds = append(params.ConditionIds, item.ConditionID)
		params.Titles = append(params.Titles, item.Title)
		params.Descriptions = append(params.Descriptions, item.Description)
		params.Logos = append(params.Logos, item.Logo)
		params.LastTradePrices = append(params.LastTradePrices, item.LastTradePrice)
		params.States = append(params.States, item.State)
		params.Categories = append(params.Categories, item.Category)
		params.SortOptions = append(params.SortOptions, item.SortOption)
		params.CreatedValues = append(params.CreatedValues, item.Created)
		params.EventTitles = append(params.EventTitles, item.EventTitle)
		params.EventConditionIds = append(params.EventConditionIds, item.EventConditionID)
		params.EventLogos = append(params.EventLogos, item.EventLogo)
		params.MarginEnabledValues = append(params.MarginEnabledValues, item.MarginEnabled)
		params.LiveStates = append(params.LiveStates, item.LiveState)
		params.LiveCheckedAtValues = append(params.LiveCheckedAtValues, nullableTime(item.LiveCheckedAt))
		params.LivePriceChanges = append(params.LivePriceChanges, item.LivePriceChange)
		params.RawValues = append(params.RawValues, []byte(item.Raw))
		params.FetchedAtValues = append(params.FetchedAtValues, nullableTime(item.FetchedAt))
		params.LastSeenAtValues = append(params.LastSeenAtValues, nullableTime(item.LastSeenAt))
	}
	return params
}
