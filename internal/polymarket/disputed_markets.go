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

func (s *Service) runDisputedMarketSyncLoop(ctx context.Context) {
	defer s.runWG.Done()
	if err := s.syncDisputedMarkets(ctx); err != nil {
		log.WithError(err).Warn("initial polymarket disputed markets sync failed")
	}
	ticker := time.NewTicker(s.disputedMarketRefreshInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.syncDisputedMarkets(ctx); err != nil {
				log.WithError(err).Warn("periodic polymarket disputed markets sync failed")
			}
		}
	}
}

func (s *Service) syncDisputedMarkets(ctx context.Context) error {
	syncStartedAt := s.now().UTC()
	markets, err := s.fetchUMAResolutionMarkets(ctx, syncStartedAt)
	if err != nil {
		return err
	}
	candidates, err := s.store.SyncUMAResolutionMarkets(ctx, markets, syncStartedAt, s.now().UTC())
	if err != nil {
		return err
	}
	s.sendUMAResolutionAlerts(ctx, candidates)
	return nil
}

func (s *Service) ListPolymarketDisputedMarkets(ctx context.Context, req *apiclient.ListPolymarketDisputedMarketsRequest) (*apiclient.ListPolymarketDisputedMarketsResponse, error) {
	limit := defaultDisputedMarketListLimit
	if req != nil && req.GetLimit() > 0 {
		limit = int(req.GetLimit())
	}
	if limit < 1 || limit > maxDisputedMarketListLimit {
		return nil, status.Errorf(codes.InvalidArgument, "limit must be between 1 and %d", maxDisputedMarketListLimit)
	}
	if s.store == nil {
		return nil, status.Error(codes.FailedPrecondition, "polymarket store is required")
	}
	markets, err := s.store.ListDisputedMarkets(ctx, int32(limit))
	if err != nil {
		return nil, status.Errorf(codes.Unavailable, "failed to list disputed markets: %v", err)
	}
	lastSuccessAt, err := s.store.GetDisputedMarketsLastSuccessAt(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unavailable, "failed to read disputed markets sync state: %v", err)
	}

	resp := &apiclient.ListPolymarketDisputedMarketsResponse{
		Items: make([]*v1alpha1.PolymarketDisputedMarketItem, 0, len(markets)),
		Stale: lastSuccessAt.IsZero() || s.now().Sub(lastSuccessAt) > 2*s.disputedMarketRefreshInterval,
	}
	if !lastSuccessAt.IsZero() {
		resp.FetchedAt = lastSuccessAt.Unix()
	}
	for _, market := range markets {
		resp.Items = append(resp.Items, &v1alpha1.PolymarketDisputedMarketItem{
			MarketKey:             market.MarketKey,
			MarketID:              market.MarketID,
			ConditionID:           market.ConditionID,
			MarketSlug:            market.Slug,
			EventID:               market.EventID,
			EventSlug:             market.EventSlug,
			Question:              market.Question,
			Image:                 market.Image,
			UMAResolutionStatus:   market.UMAResolutionStatus,
			UMAResolutionStatuses: market.UMAResolutionStatuses,
			Volume24hr:            market.Volume24hr,
			VolumeNum:             market.VolumeNum,
			LiquidityNum:          market.LiquidityNum,
			Spread:                market.Spread,
			BestBid:               market.BestBid,
			BestAsk:               market.BestAsk,
			LastTradePrice:        market.LastTradePrice,
			Active:                market.Active,
			Closed:                market.Closed,
			EnableOrderBook:       market.EnableOrderBook,
			FetchedAt:             formatTime(market.FetchedAt),
			LastSeenAt:            formatTime(market.LastSeenAt),
		})
	}
	return resp, nil
}

func (s *Service) fetchUMAResolutionMarkets(ctx context.Context, fetchedAt time.Time) ([]polymarketstore.UMAResolutionMarket, error) {
	if s.gammaClient == nil {
		return nil, status.Error(codes.FailedPrecondition, "polymarket gamma client is required")
	}

	marketsByKey := make(map[string]polymarketstore.UMAResolutionMarket)
	for _, statusValue := range []string{umaResolutionStatusProposed, umaResolutionStatusDisputed} {
		markets, err := s.fetchUMAResolutionMarketsByStatus(ctx, fetchedAt, statusValue)
		if err != nil {
			return nil, err
		}
		for _, market := range markets {
			marketsByKey[market.MarketKey] = market
		}
	}

	markets := make([]polymarketstore.UMAResolutionMarket, 0, len(marketsByKey))
	for _, market := range marketsByKey {
		markets = append(markets, market)
	}
	return markets, nil
}

func (s *Service) fetchUMAResolutionMarketsByStatus(ctx context.Context, fetchedAt time.Time, statusValue string) ([]polymarketstore.UMAResolutionMarket, error) {
	limit := defaultDisputedMarketPageLimit
	cursor := ""
	closed := false
	ascending := false
	marketsByKey := make(map[string]polymarketstore.UMAResolutionMarket)
	for {
		opts := utilpolymarket.ListMarketsKeysetOptions{
			Limit:               &limit,
			Order:               "volume24hr",
			Ascending:           &ascending,
			AfterCursor:         cursor,
			Closed:              &closed,
			UMAResolutionStatus: statusValue,
		}
		resp, err := s.gammaClient.ListMarketsKeyset(ctx, opts)
		if err != nil {
			return nil, err
		}
		if resp == nil || len(resp.Markets) == 0 {
			break
		}
		for i := range resp.Markets {
			market, ok := umaResolutionMarketFromGamma(resp.Markets[i], fetchedAt)
			if !ok {
				continue
			}
			marketsByKey[market.MarketKey] = market
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

	markets := make([]polymarketstore.UMAResolutionMarket, 0, len(marketsByKey))
	for _, market := range marketsByKey {
		markets = append(markets, market)
	}
	return markets, nil
}

func umaResolutionMarketFromGamma(market utilpolymarket.Market, fetchedAt time.Time) (polymarketstore.UMAResolutionMarket, bool) {
	umaStatuses := strings.TrimSpace(stringValue(market.UMAResolutionStatuses))
	umaStatus := canonicalUMAResolutionStatus(stringValue(market.UMAResolutionStatus), umaStatuses)
	if !isTrackedUMAResolutionStatus(umaStatus) {
		return polymarketstore.UMAResolutionMarket{}, false
	}

	conditionID := strings.TrimSpace(stringValue(market.ConditionID))
	marketID := strings.TrimSpace(market.ID)
	slug := strings.TrimSpace(stringValue(market.Slug))
	marketKey := firstNonEmpty(conditionID, marketID, slug)
	if marketKey == "" {
		return polymarketstore.UMAResolutionMarket{}, false
	}

	eventID, eventSlug := umaResolutionMarketEventRefs(market)
	return polymarketstore.UMAResolutionMarket{
		MarketKey:             marketKey,
		MarketID:              marketID,
		ConditionID:           conditionID,
		Slug:                  slug,
		EventID:               eventID,
		EventSlug:             eventSlug,
		Question:              strings.TrimSpace(stringValue(market.Question)),
		Description:           strings.TrimSpace(stringValue(market.Description)),
		ResolutionSource:      strings.TrimSpace(stringValue(market.ResolutionSource)),
		Image:                 strings.TrimSpace(stringValue(market.Image)),
		Icon:                  strings.TrimSpace(stringValue(market.Icon)),
		UMAResolutionStatus:   umaStatus,
		UMAResolutionStatuses: umaStatuses,
		Outcomes:              strings.TrimSpace(stringValue(market.Outcomes)),
		OutcomePrices:         strings.TrimSpace(stringValue(market.OutcomePrices)),
		ClobTokenIDs:          strings.TrimSpace(stringValue(market.ClobTokenIDs)),
		Active:                boolValue(market.Active),
		Closed:                boolValue(market.Closed),
		Archived:              boolValue(market.Archived),
		Restricted:            boolValue(market.Restricted),
		EnableOrderBook:       boolValue(market.EnableOrderBook),
		Volume:                strings.TrimSpace(stringValue(market.Volume)),
		VolumeNum:             float64Value(market.VolumeNum),
		LiquidityNum:          float64Value(market.LiquidityNum),
		Volume24hr:            float64Value(market.Volume24hr),
		Volume1wk:             float64Value(market.Volume1wk),
		Volume1mo:             float64Value(market.Volume1mo),
		Volume1yr:             float64Value(market.Volume1yr),
		Spread:                float64Value(market.Spread),
		BestBid:               float64Value(market.BestBid),
		BestAsk:               float64Value(market.BestAsk),
		LastTradePrice:        float64Value(market.LastTradePrice),
		Tags:                  marshalArrayOrFallback(market.Tags),
		Raw:                   rawObjectOrMarshal(market.Raw, market),
		FetchedAt:             fetchedAt,
		LastSeenAt:            fetchedAt,
	}, true
}

func canonicalUMAResolutionStatus(statusValue, statusesValue string) string {
	if status := normalizeUMAResolutionStatus(statusValue); status != "" {
		return status
	}
	var statuses []string
	if err := json.Unmarshal([]byte(statusesValue), &statuses); err == nil {
		for i := len(statuses) - 1; i >= 0; i-- {
			if status := normalizeUMAResolutionStatus(statuses[i]); status != "" {
				return status
			}
		}
		return ""
	}
	trail := parseUMAResolutionStatusTrail(statusesValue)
	for i := len(trail) - 1; i >= 0; i-- {
		if status := normalizeUMAResolutionStatus(trail[i]); status != "" {
			return status
		}
	}
	return ""
}

func normalizeUMAResolutionStatus(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if isTrackedUMAResolutionStatus(value) {
		return value
	}
	return ""
}

func isTrackedUMAResolutionStatus(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case umaResolutionStatusProposed, umaResolutionStatusDisputed:
		return true
	default:
		return false
	}
}

func umaResolutionMarketEventRefs(market utilpolymarket.Market) (string, string) {
	for i := range market.Events {
		eventID := strings.TrimSpace(market.Events[i].ID)
		eventSlug := strings.TrimSpace(stringValue(market.Events[i].Slug))
		if eventID != "" || eventSlug != "" {
			return eventID, eventSlug
		}
	}
	return "", ""
}
