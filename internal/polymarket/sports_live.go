package polymarket

import (
	"context"
	"encoding/json"
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

const polymarketEsportsTagID int64 = 64
const sportsLiveMinEventLiquidity = 10000.0

var (
	sportsLiveJSONObject = json.RawMessage(`{}`)
	sportsLiveJSONArray  = json.RawMessage(`[]`)
)

func (s *Service) runSportsLiveSyncLoop(ctx context.Context) {
	defer s.runWG.Done()
	if err := s.syncSportsLiveMarkets(ctx); err != nil {
		log.WithError(err).Warn("initial polymarket sports live sync failed")
	}
	ticker := time.NewTicker(s.syncInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.syncSportsLiveMarkets(ctx); err != nil {
				log.WithError(err).Warn("periodic polymarket sports live sync failed")
			}
		}
	}
}

func (s *Service) syncSportsLiveMarkets(ctx context.Context) error {
	syncStartedAt := s.now().UTC()
	events, markets, err := s.fetchSportsLiveEvents(ctx, syncStartedAt)
	if err != nil {
		return err
	}
	return s.store.SyncSportsLiveEvents(ctx, events, markets, syncStartedAt, s.now().UTC())
}

func (s *Service) fetchSportsLiveEvents(ctx context.Context, fetchedAt time.Time) ([]polymarketstore.SportsLiveEvent, []polymarketstore.SportsLiveMarket, error) {
	if s.gammaClient == nil {
		return nil, nil, status.Error(codes.FailedPrecondition, "polymarket gamma client is required")
	}

	limit := s.sportsLivePageLimit
	if limit <= 0 {
		limit = defaultSportsLiveEventPageLimit
	}

	eventsByKey := make(map[string]polymarketstore.SportsLiveEvent)
	marketsByKey := make(map[string]polymarketstore.SportsLiveMarket)
	minLiquidity := sportsLiveMinEventLiquidity
	cursor := ""
	live := true
	closed := false
	for {
		opts := utilpolymarket.ListEventsKeysetOptions{
			Limit:        &limit,
			Order:        "id",
			Ascending:    ptrBool(true),
			Live:         ptrBool(live),
			Closed:       ptrBool(closed),
			LiquidityMin: &minLiquidity,
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
			if !boolValue(event.Live) || boolValue(event.Ended) {
				continue
			}
			eventKey := firstNonEmpty(event.ID, stringValue(event.Slug))
			if eventKey == "" {
				continue
			}
			eventSlug := strings.TrimSpace(stringValue(event.Slug))
			eventsByKey[eventKey] = sportsLiveEventFromGamma(eventKey, event, fetchedAt)
			for j := range event.Markets {
				market := event.Markets[j]
				if boolValue(market.Closed) {
					continue
				}
				marketKey := firstNonEmpty(stringValue(market.ConditionID), market.ID, stringValue(market.Slug))
				if marketKey == "" {
					continue
				}
				marketsByKey[marketKey] = sportsLiveMarketFromGamma(eventKey, event.ID, eventSlug, marketKey, market, fetchedAt)
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

	events := make([]polymarketstore.SportsLiveEvent, 0, len(eventsByKey))
	for _, item := range eventsByKey {
		events = append(events, item)
	}
	markets := make([]polymarketstore.SportsLiveMarket, 0, len(marketsByKey))
	for _, item := range marketsByKey {
		markets = append(markets, item)
	}
	return events, markets, nil
}

func sportsLiveEventFromGamma(eventKey string, event utilpolymarket.Event, fetchedAt time.Time) polymarketstore.SportsLiveEvent {
	return polymarketstore.SportsLiveEvent{
		EventKey:          eventKey,
		EventID:           strings.TrimSpace(event.ID),
		Ticker:            strings.TrimSpace(stringValue(event.Ticker)),
		Slug:              strings.TrimSpace(stringValue(event.Slug)),
		Title:             strings.TrimSpace(stringValue(event.Title)),
		Description:       strings.TrimSpace(stringValue(event.Description)),
		ResolutionSource:  strings.TrimSpace(stringValue(event.ResolutionSource)),
		StartDate:         timePtrValue(event.StartDate),
		CreationDate:      timePtrValue(event.CreationDate),
		EndDate:           timePtrValue(event.EndDate),
		StartTime:         timePtrValue(event.StartTime),
		CreatedAtGamma:    timePtrValue(event.CreatedAt),
		UpdatedAtGamma:    timePtrValue(event.UpdatedAt),
		Image:             strings.TrimSpace(stringValue(event.Image)),
		Icon:              strings.TrimSpace(stringValue(event.Icon)),
		Active:            boolValue(event.Active),
		Closed:            boolValue(event.Closed),
		Archived:          boolValue(event.Archived),
		Featured:          boolValue(event.Featured),
		Restricted:        boolValue(event.Restricted),
		Live:              boolValue(event.Live),
		Ended:             boolValue(event.Ended),
		Liquidity:         float64Value(event.Liquidity),
		Volume:            float64Value(event.Volume),
		OpenInterest:      float64Value(event.OpenInterest),
		Category:          strings.TrimSpace(stringValue(event.Category)),
		Score:             strings.TrimSpace(stringValue(event.Score)),
		Period:            strings.TrimSpace(stringValue(event.Period)),
		Elapsed:           strings.TrimSpace(stringValue(event.Elapsed)),
		FinishedTimestamp: strings.TrimSpace(stringValue(event.FinishedTimestamp)),
		GameID:            int64Value(event.GameID),
		EventDate:         strings.TrimSpace(stringValue(event.EventDate)),
		GameStatus:        strings.TrimSpace(stringValue(event.GameStatus)),
		CommentCount:      int64Value(event.CommentCount),
		Sport:             jsonObjectOrFallback(event.Sport),
		Teams:             marshalArrayOrFallback(event.Teams),
		Tags:              marshalArrayOrFallback(event.Tags),
		Raw:               rawObjectOrMarshal(event.Raw, event),
		FetchedAt:         fetchedAt,
		LastSeenAt:        fetchedAt,
	}
}

func sportsLiveMarketFromGamma(eventKey, eventID, eventSlug, marketKey string, market utilpolymarket.Market, fetchedAt time.Time) polymarketstore.SportsLiveMarket {
	return polymarketstore.SportsLiveMarket{
		MarketKey:        marketKey,
		EventKey:         eventKey,
		EventID:          strings.TrimSpace(eventID),
		EventSlug:        strings.TrimSpace(eventSlug),
		MarketID:         strings.TrimSpace(market.ID),
		ConditionID:      strings.TrimSpace(stringValue(market.ConditionID)),
		Slug:             strings.TrimSpace(stringValue(market.Slug)),
		Question:         strings.TrimSpace(stringValue(market.Question)),
		Title:            strings.TrimSpace(stringValue(market.GroupItemTitle)),
		Description:      strings.TrimSpace(stringValue(market.Description)),
		ResolutionSource: strings.TrimSpace(stringValue(market.ResolutionSource)),
		SportsMarketType: strings.TrimSpace(stringValue(market.SportsMarketType)),
		GroupItemTitle:   strings.TrimSpace(stringValue(market.GroupItemTitle)),
		Image:            strings.TrimSpace(stringValue(market.Image)),
		Icon:             strings.TrimSpace(stringValue(market.Icon)),
		Outcomes:         strings.TrimSpace(stringValue(market.Outcomes)),
		OutcomePrices:    strings.TrimSpace(stringValue(market.OutcomePrices)),
		ClobTokenIDs:     strings.TrimSpace(stringValue(market.ClobTokenIDs)),
		Active:           boolValue(market.Active),
		Closed:           boolValue(market.Closed),
		Archived:         boolValue(market.Archived),
		Restricted:       boolValue(market.Restricted),
		EnableOrderBook:  boolValue(market.EnableOrderBook),
		Volume:           strings.TrimSpace(stringValue(market.Volume)),
		VolumeNum:        float64Value(market.VolumeNum),
		LiquidityNum:     float64Value(market.LiquidityNum),
		Volume24hr:       float64Value(market.Volume24hr),
		Volume1wk:        float64Value(market.Volume1wk),
		Volume1mo:        float64Value(market.Volume1mo),
		Volume1yr:        float64Value(market.Volume1yr),
		Spread:           float64Value(market.Spread),
		BestBid:          float64Value(market.BestBid),
		BestAsk:          float64Value(market.BestAsk),
		LastTradePrice:   float64Value(market.LastTradePrice),
		StartDate:        timePtrValue(market.StartDate),
		EndDate:          timePtrValue(market.EndDate),
		CreatedAtGamma:   timePtrValue(market.CreatedAt),
		UpdatedAtGamma:   timePtrValue(market.UpdatedAt),
		Tags:             marshalArrayOrFallback(market.Tags),
		Raw:              rawObjectOrMarshal(market.Raw, market),
		FetchedAt:        fetchedAt,
		LastSeenAt:       fetchedAt,
	}
}

func (s *Service) ListPolymarketSportsLiveEvents(ctx context.Context, req *apiclient.ListPolymarketSportsLiveEventsRequest) (*apiclient.ListPolymarketSportsLiveEventsResponse, error) {
	limit := defaultSportsLiveListLimit
	if req != nil && req.GetLimit() > 0 {
		limit = int(req.GetLimit())
	}
	if limit < 1 || limit > maxSportsLiveListLimit {
		return nil, status.Errorf(codes.InvalidArgument, "limit must be between 1 and %d", maxSportsLiveListLimit)
	}
	if s.store == nil {
		return nil, status.Error(codes.FailedPrecondition, "polymarket store is required")
	}

	events, err := s.store.ListSportsLiveEventCards(ctx, int32(limit))
	if err != nil {
		return nil, status.Errorf(codes.Unavailable, "failed to list sports live events: %v", err)
	}
	lastSuccessAt, err := s.store.GetSportsLiveLastSuccessAt(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unavailable, "failed to read sports live sync state: %v", err)
	}

	resp := &apiclient.ListPolymarketSportsLiveEventsResponse{
		Items: make([]*v1alpha1.PolymarketSportsLiveEventCardItem, 0, len(events)),
		Stale: lastSuccessAt.IsZero() || s.now().Sub(lastSuccessAt) > 2*s.syncInterval,
	}
	if !lastSuccessAt.IsZero() {
		resp.FetchedAt = lastSuccessAt.Unix()
	}
	for _, event := range events {
		item := &v1alpha1.PolymarketSportsLiveEventCardItem{
			EventKey:    event.EventKey,
			EventID:     event.EventID,
			Slug:        event.Slug,
			Title:       event.Title,
			Image:       event.Image,
			Score:       event.Score,
			Period:      event.Period,
			Elapsed:     event.Elapsed,
			GameStatus:  event.GameStatus,
			StartTime:   formatTime(event.StartTime),
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

func sportsLiveTeamsFromRaw(raw json.RawMessage) []*v1alpha1.PolymarketSportsLiveTeamItem {
	type gammaTeam struct {
		Name         *string `json:"name,omitempty"`
		Logo         *string `json:"logo,omitempty"`
		Abbreviation *string `json:"abbreviation,omitempty"`
		Alias        *string `json:"alias,omitempty"`
	}
	var teams []gammaTeam
	if err := json.Unmarshal(raw, &teams); err != nil {
		return nil
	}
	items := make([]*v1alpha1.PolymarketSportsLiveTeamItem, 0, len(teams))
	for _, team := range teams {
		item := &v1alpha1.PolymarketSportsLiveTeamItem{
			Name:         strings.TrimSpace(stringValue(team.Name)),
			Logo:         strings.TrimSpace(stringValue(team.Logo)),
			Abbreviation: strings.TrimSpace(stringValue(team.Abbreviation)),
			Alias:        strings.TrimSpace(stringValue(team.Alias)),
		}
		if firstNonEmpty(item.Name, item.Abbreviation, item.Alias) == "" && item.Logo == "" {
			continue
		}
		items = append(items, item)
	}
	return items
}

func firstNonEmpty(values ...string) string {
	for i := range values {
		if strings.TrimSpace(values[i]) != "" {
			return strings.TrimSpace(values[i])
		}
	}
	return ""
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func boolValue(value *bool) bool {
	if value == nil {
		return false
	}
	return *value
}

func float64Value(value *float64) float64 {
	if value == nil {
		return 0
	}
	return *value
}

func int64Value(value *int64) int64 {
	if value == nil {
		return 0
	}
	return *value
}

func timePtrValue(value *time.Time) time.Time {
	if value == nil {
		return time.Time{}
	}
	return value.UTC()
}

func formatTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}

func jsonObjectOrFallback(value json.RawMessage) json.RawMessage {
	if jsonType(value) == "object" {
		return value
	}
	return sportsLiveJSONObject
}

func marshalOrFallback(value any, fallback json.RawMessage) json.RawMessage {
	data, err := json.Marshal(value)
	if err != nil || !json.Valid(data) {
		return fallback
	}
	return data
}

func marshalArrayOrFallback(value any) json.RawMessage {
	data := marshalOrFallback(value, sportsLiveJSONArray)
	if jsonType(data) != "array" {
		return sportsLiveJSONArray
	}
	return data
}

func rawObjectOrMarshal(raw json.RawMessage, value any) json.RawMessage {
	if jsonType(raw) == "object" {
		return raw
	}
	data := marshalOrFallback(value, sportsLiveJSONObject)
	if jsonType(data) != "object" {
		return sportsLiveJSONObject
	}
	return data
}

func jsonType(value json.RawMessage) string {
	var decoded any
	if err := json.Unmarshal(value, &decoded); err != nil {
		return ""
	}
	switch decoded.(type) {
	case map[string]any:
		return "object"
	case []any:
		return "array"
	default:
		return ""
	}
}
