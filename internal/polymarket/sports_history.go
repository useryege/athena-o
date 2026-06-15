package polymarket

import (
	"context"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/useryege/athena/internal/polymarket/apiclient"
	polymarketstore "github.com/useryege/athena/internal/polymarket/store"
	"github.com/useryege/athena/pkg/apis/application/v1alpha1"
	utilpolymarket "github.com/useryege/athena/util/polymarket"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	sportsHistoryWindow                   = 72 * time.Hour
	defaultSportsHistoryListLimit         = 200
	maxSportsHistoryListLimit             = 1000
	defaultSportsHistoryPriceHistoryLimit = 360
	maxSportsHistoryPriceHistoryLimit     = 720
)

func (s *Service) runSportsHistorySync(ctx context.Context) {
	defer s.runWG.Done()
	if err := s.syncSportsHistory(ctx); err != nil {
		log.WithError(err).Warn("initial polymarket sports history sync failed")
		return
	}
	s.cacheMu.Lock()
	s.sportsHistoryStale = false
	s.cacheMu.Unlock()
	if err := s.syncSportsHistoryPriceHistory(ctx); err != nil {
		log.WithError(err).Warn("initial polymarket sports history price sync failed")
	}
}

func (s *Service) syncSportsHistory(ctx context.Context) error {
	syncStartedAt := s.now().UTC()
	events, markets, err := s.fetchSportsHistoryEvents(ctx, syncStartedAt)
	if err != nil {
		return err
	}
	return s.store.SyncSportsHistory(ctx, events, markets, syncStartedAt, s.now().UTC())
}

func (s *Service) fetchSportsHistoryEvents(ctx context.Context, fetchedAt time.Time) ([]polymarketstore.SportsHistoryEvent, []polymarketstore.SportsHistoryMarket, error) {
	if s.gammaClient == nil {
		return nil, nil, status.Error(codes.FailedPrecondition, "polymarket gamma client is required")
	}

	limit := s.sportsLivePageLimit
	if limit <= 0 {
		limit = defaultSportsLiveEventPageLimit
	}
	windowStart := fetchedAt.Add(-sportsHistoryWindow)
	windowEnd := fetchedAt
	eventsByKey := make(map[string]polymarketstore.SportsHistoryEvent)
	marketsByKey := make(map[string]polymarketstore.SportsHistoryMarket)
	for _, closed := range []bool{true, false} {
		cursor := ""
		for {
			opts := utilpolymarket.ListEventsKeysetOptions{
				Limit:        &limit,
				Order:        "startTime",
				Ascending:    ptrBool(false),
				Closed:       ptrBool(closed),
				StartTimeMin: &windowStart,
				StartTimeMax: &windowEnd,
				TagSlug:      "sports",
				ExcludeTagID: []int64{polymarketEsportsTagID},
			}
			if cursor != "" {
				opts.AfterCursor = cursor
			}
			resp, err := s.gammaClient.ListEventsKeyset(ctx, opts)
			if err != nil {
				return nil, nil, err
			}
			if resp == nil || len(resp.Events) == 0 {
				break
			}
			for i := range resp.Events {
				event := resp.Events[i]
				league := sportsHistoryLeague(event)
				finishedAt, ok := sportsHistoryFinishedAt(event, fetchedAt)
				if league == "" || !ok || !sportsHistoryEventInWindow(event, windowStart, windowEnd) || isSportsLiveDerivativeEvent(event) {
					continue
				}
				eventKey := firstNonEmpty(event.ID, stringValue(event.Slug))
				if eventKey == "" {
					continue
				}
				eventMarkets := sportsHistoryMarketsFromGamma(eventKey, event, fetchedAt)
				if len(eventMarkets) == 0 {
					continue
				}
				eventsByKey[eventKey] = sportsHistoryEventFromGamma(eventKey, league, event, finishedAt, fetchedAt)
				for _, market := range eventMarkets {
					marketsByKey[market.MarketKey] = market
				}
			}
			if resp.NextCursor == nil {
				break
			}
			nextCursor := strings.TrimSpace(*resp.NextCursor)
			if nextCursor == "" || nextCursor == cursor {
				break
			}
			cursor = nextCursor
		}
	}

	events := make([]polymarketstore.SportsHistoryEvent, 0, len(eventsByKey))
	for _, event := range eventsByKey {
		events = append(events, event)
	}
	markets := make([]polymarketstore.SportsHistoryMarket, 0, len(marketsByKey))
	for _, market := range marketsByKey {
		markets = append(markets, market)
	}
	return events, markets, nil
}

func sportsHistoryLeague(event utilpolymarket.Event) string {
	slug := strings.ToLower(strings.TrimSpace(stringValue(event.Slug)))
	switch {
	case strings.HasPrefix(slug, "atp-") && !strings.HasPrefix(slug, "atp-doubles-"):
		return "ATP"
	case strings.HasPrefix(slug, "wta-") && !strings.HasPrefix(slug, "wta-doubles-"):
		return "WTA"
	default:
		return ""
	}
}

func sportsHistoryEventInWindow(event utilpolymarket.Event, start, end time.Time) bool {
	if event.StartTime == nil {
		return false
	}
	startTime := event.StartTime.UTC()
	return !startTime.Before(start) && !startTime.After(end)
}

func sportsHistoryFinishedAt(event utilpolymarket.Event, fetchedAt time.Time) (time.Time, bool) {
	finishedAt := parseSportsHistoryTime(stringValue(event.FinishedTimestamp))
	statusValue := strings.ToLower(strings.TrimSpace(stringValue(event.GameStatus)))
	finished := boolValue(event.Ended) || !finishedAt.IsZero() ||
		strings.Contains(statusValue, "final") ||
		strings.Contains(statusValue, "finished") ||
		strings.Contains(statusValue, "complete") ||
		strings.Contains(statusValue, "ended")
	if !finished {
		return time.Time{}, false
	}
	if !finishedAt.IsZero() {
		if event.StartTime != nil && finishedAt.Before(event.StartTime.UTC()) {
			return time.Time{}, false
		}
		return finishedAt, true
	}
	if event.EndDate != nil {
		endDate := event.EndDate.UTC()
		if !endDate.After(fetchedAt) && (event.StartTime == nil || endDate.After(event.StartTime.UTC())) {
			return endDate, true
		}
	}
	return fetchedAt, true
}

func parseSportsHistoryTime(value string) time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339} {
		parsed, err := time.Parse(layout, value)
		if err == nil {
			return parsed.UTC()
		}
	}
	return time.Time{}
}

func sportsHistoryEventFromGamma(eventKey, league string, event utilpolymarket.Event, finishedAt, fetchedAt time.Time) polymarketstore.SportsHistoryEvent {
	return polymarketstore.SportsHistoryEvent{
		EventKey:       eventKey,
		EventID:        strings.TrimSpace(event.ID),
		League:         league,
		Slug:           strings.TrimSpace(stringValue(event.Slug)),
		Title:          strings.TrimSpace(stringValue(event.Title)),
		Image:          strings.TrimSpace(stringValue(event.Image)),
		Icon:           strings.TrimSpace(stringValue(event.Icon)),
		Score:          strings.TrimSpace(stringValue(event.Score)),
		Period:         strings.TrimSpace(stringValue(event.Period)),
		Elapsed:        strings.TrimSpace(stringValue(event.Elapsed)),
		GameStatus:     strings.TrimSpace(stringValue(event.GameStatus)),
		StartTime:      timePtrValue(event.StartTime),
		FinishedAt:     finishedAt,
		UpdatedAtGamma: timePtrValue(event.UpdatedAt),
		Liquidity:      float64Value(event.Liquidity),
		Volume:         float64Value(event.Volume),
		Teams:          marshalArrayOrFallback(event.Teams),
		Raw:            rawObjectOrMarshal(event.Raw, event),
		FetchedAt:      fetchedAt,
		LastSeenAt:     fetchedAt,
	}
}

func sportsHistoryMarketsFromGamma(eventKey string, event utilpolymarket.Event, fetchedAt time.Time) []polymarketstore.SportsHistoryMarket {
	items := make([]polymarketstore.SportsHistoryMarket, 0, len(event.Markets))
	for i := range event.Markets {
		market := event.Markets[i]
		if !strings.EqualFold(strings.TrimSpace(stringValue(market.SportsMarketType)), "moneyline") {
			continue
		}
		marketKey := firstNonEmpty(stringValue(market.ConditionID), market.ID, stringValue(market.Slug))
		if marketKey == "" {
			continue
		}
		items = append(items, polymarketstore.SportsHistoryMarket{
			MarketKey:        marketKey,
			EventKey:         eventKey,
			ConditionID:      strings.TrimSpace(stringValue(market.ConditionID)),
			Slug:             strings.TrimSpace(stringValue(market.Slug)),
			Question:         firstNonEmpty(stringValue(market.Question), stringValue(market.GroupItemTitle)),
			SportsMarketType: strings.TrimSpace(stringValue(market.SportsMarketType)),
			Outcomes:         strings.TrimSpace(stringValue(market.Outcomes)),
			OutcomePrices:    strings.TrimSpace(stringValue(market.OutcomePrices)),
			ClobTokenIDs:     strings.TrimSpace(stringValue(market.ClobTokenIDs)),
			BestBid:          float64Value(market.BestBid),
			BestAsk:          float64Value(market.BestAsk),
			LastTradePrice:   float64Value(market.LastTradePrice),
			Spread:           float64Value(market.Spread),
			LiquidityNum:     float64Value(market.LiquidityNum),
			VolumeNum:        float64Value(market.VolumeNum),
			UpdatedAtGamma:   timePtrValue(market.UpdatedAt),
			Raw:              rawObjectOrMarshal(market.Raw, market),
			FetchedAt:        fetchedAt,
			LastSeenAt:       fetchedAt,
		})
	}
	return items
}

func (s *Service) ListPolymarketSportsHistoryEvents(ctx context.Context, req *apiclient.ListPolymarketSportsHistoryEventsRequest) (*apiclient.ListPolymarketSportsHistoryEventsResponse, error) {
	limit := defaultSportsHistoryListLimit
	if req != nil && req.GetLimit() > 0 {
		limit = int(req.GetLimit())
	}
	if limit < 1 || limit > maxSportsHistoryListLimit {
		return nil, status.Errorf(codes.InvalidArgument, "limit must be between 1 and %d", maxSportsHistoryListLimit)
	}
	if s.store == nil {
		return nil, status.Error(codes.FailedPrecondition, "polymarket store is required")
	}
	events, err := s.store.ListSportsHistoryEventCards(ctx, int32(limit))
	if err != nil {
		return nil, status.Errorf(codes.Unavailable, "failed to list sports history events: %v", err)
	}
	lastSuccessAt, err := s.store.GetSportsHistoryLastSuccessAt(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unavailable, "failed to read sports history sync state: %v", err)
	}
	s.cacheMu.RLock()
	stale := s.sportsHistoryStale
	s.cacheMu.RUnlock()
	resp := &apiclient.ListPolymarketSportsHistoryEventsResponse{
		Items: make([]*v1alpha1.PolymarketSportsHistoryEventCardItem, 0, len(events)),
		Stale: stale || lastSuccessAt.IsZero(),
	}
	if !lastSuccessAt.IsZero() {
		resp.FetchedAt = lastSuccessAt.Unix()
	}
	for _, event := range events {
		item := &v1alpha1.PolymarketSportsHistoryEventCardItem{
			EventKey:    event.EventKey,
			EventID:     event.EventID,
			League:      event.League,
			Slug:        event.Slug,
			Title:       event.Title,
			Image:       event.Image,
			Score:       event.Score,
			Period:      event.Period,
			Elapsed:     event.Elapsed,
			GameStatus:  event.GameStatus,
			StartTime:   formatTime(event.StartTime),
			FinishedAt:  formatTime(event.FinishedAt),
			UpdatedAt:   formatTime(event.UpdatedAtGamma),
			Liquidity:   event.Liquidity,
			Volume:      event.Volume,
			MarketCount: int32(event.MarketCount),
			Markets:     make([]*v1alpha1.PolymarketSportsLiveMarketCardItem, 0, len(event.Markets)),
			Teams:       sportsLiveTeamsFromRaw(event.Teams),
		}
		for _, market := range event.Markets {
			item.Markets = append(item.Markets, &v1alpha1.PolymarketSportsLiveMarketCardItem{
				MarketKey:        market.MarketKey,
				ConditionID:      market.ConditionID,
				Slug:             market.Slug,
				SportsMarketType: market.SportsMarketType,
				Question:         market.Question,
				Outcomes:         market.Outcomes,
				OutcomePrices:    market.OutcomePrices,
				BestBid:          market.BestBid,
				BestAsk:          market.BestAsk,
				LastTradePrice:   market.LastTradePrice,
				Spread:           market.Spread,
				LiquidityNum:     market.LiquidityNum,
				VolumeNum:        market.VolumeNum,
				UpdatedAt:        formatTime(market.UpdatedAtGamma),
			})
		}
		resp.Items = append(resp.Items, item)
	}
	return resp, nil
}

func (s *Service) BatchGetPolymarketSportsHistoryPriceHistory(ctx context.Context, req *apiclient.BatchGetPolymarketSportsHistoryPriceHistoryRequest) (*apiclient.BatchGetPolymarketSportsHistoryPriceHistoryResponse, error) {
	limitPerToken := defaultSportsHistoryPriceHistoryLimit
	if req != nil && req.GetLimitPerToken() > 0 {
		limitPerToken = int(req.GetLimitPerToken())
	}
	if limitPerToken < 1 || limitPerToken > maxSportsHistoryPriceHistoryLimit {
		return nil, status.Errorf(codes.InvalidArgument, "limit_per_token must be between 1 and %d", maxSportsHistoryPriceHistoryLimit)
	}
	if s.store == nil {
		return nil, status.Error(codes.FailedPrecondition, "polymarket store is required")
	}
	marketKeys := uniqueNonEmptyStrings(req.GetMarketKeys())
	if len(marketKeys) == 0 {
		return &apiclient.BatchGetPolymarketSportsHistoryPriceHistoryResponse{}, nil
	}
	series, err := s.store.ListSportsHistoryPriceHistorySeries(ctx, marketKeys, int32(limitPerToken))
	if err != nil {
		return nil, status.Errorf(codes.Unavailable, "failed to list sports history price history: %v", err)
	}
	resp := &apiclient.BatchGetPolymarketSportsHistoryPriceHistoryResponse{
		Items: make([]*v1alpha1.PolymarketSportsLivePriceHistorySeriesItem, 0, len(series)),
	}
	for _, item := range series {
		resp.Items = append(resp.Items, &v1alpha1.PolymarketSportsLivePriceHistorySeriesItem{
			MarketKey:  item.MarketKey,
			TokenID:    item.TokenID,
			Outcome:    item.Outcome,
			Timestamps: item.Timestamps,
			Prices:     item.Prices,
		})
	}
	return resp, nil
}
