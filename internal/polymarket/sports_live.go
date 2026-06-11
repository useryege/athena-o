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

const polymarketEsportsTagID int64 = 64

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
	markets, err := s.fetchSportsLiveMarkets(ctx, syncStartedAt)
	if err != nil {
		return err
	}
	return s.store.SyncSportsLiveMarkets(ctx, markets, syncStartedAt, s.now().UTC())
}

func (s *Service) fetchSportsLiveMarkets(ctx context.Context, fetchedAt time.Time) ([]polymarketstore.SportsLiveMarket, error) {
	if s.gammaClient == nil {
		return nil, status.Error(codes.FailedPrecondition, "polymarket gamma client is required")
	}

	limit := s.sportsLivePageLimit
	if limit <= 0 {
		limit = defaultSportsLiveEventPageLimit
	}

	itemsByConditionID := make(map[string]polymarketstore.SportsLiveMarket)
	cursor := ""
	live := true
	closed := false
	for {
		opts := utilpolymarket.ListEventsKeysetOptions{
			Limit:        &limit,
			Live:         ptrBool(live),
			Closed:       ptrBool(closed),
			TagSlug:      "sports",
			ExcludeTagID: []int64{polymarketEsportsTagID},
		}
		if cursor != "" {
			opts.AfterCursor = cursor
		}
		resp, err := s.gammaClient.ListEventsKeyset(ctx, opts)
		if err != nil {
			return nil, err
		}
		if resp == nil || len(resp.Events) == 0 {
			break
		}
		for i := range resp.Events {
			event := resp.Events[i]
			if !boolValue(event.Live) || boolValue(event.Ended) {
				continue
			}
			eventSlug := strings.TrimSpace(stringValue(event.Slug))
			if eventSlug == "" {
				continue
			}
			eventTitle := strings.TrimSpace(stringValue(event.Title))
			eventImage := firstNonEmpty(
				strings.TrimSpace(stringValue(event.Image)),
				strings.TrimSpace(stringValue(event.Icon)),
			)
			eventUpdatedAt := time.Time{}
			if event.UpdatedAt != nil {
				eventUpdatedAt = event.UpdatedAt.UTC()
			}
			for j := range event.Markets {
				market := event.Markets[j]
				if boolValue(market.Closed) {
					continue
				}
				conditionID := strings.TrimSpace(stringValue(market.ConditionID))
				marketSlug := strings.TrimSpace(stringValue(market.Slug))
				if conditionID == "" || marketSlug == "" {
					continue
				}
				itemsByConditionID[conditionID] = polymarketstore.SportsLiveMarket{
					ConditionID: conditionID,
					MarketSlug:  marketSlug,
					EventSlug:   eventSlug,
					Title: firstNonEmpty(
						strings.TrimSpace(stringValue(market.Question)),
						eventTitle,
					),
					Image: firstNonEmpty(
						strings.TrimSpace(stringValue(market.Image)),
						strings.TrimSpace(stringValue(market.Icon)),
						eventImage,
					),
					Score:          strings.TrimSpace(stringValue(event.Score)),
					Period:         strings.TrimSpace(stringValue(event.Period)),
					Elapsed:        strings.TrimSpace(stringValue(event.Elapsed)),
					GammaUpdatedAt: eventUpdatedAt,
					LiquidityNum:   float64Value(market.LiquidityNum),
					VolumeNum:      float64Value(market.VolumeNum),
					FetchedAt:      fetchedAt,
					LastSeenAt:     fetchedAt,
				}
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

	items := make([]polymarketstore.SportsLiveMarket, 0, len(itemsByConditionID))
	for _, item := range itemsByConditionID {
		items = append(items, item)
	}
	return items, nil
}

func (s *Service) ListPolymarketSportsLiveMarkets(ctx context.Context, req *apiclient.ListPolymarketSportsLiveMarketsRequest) (*apiclient.ListPolymarketSportsLiveMarketsResponse, error) {
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

	markets, err := s.store.ListSportsLiveMarkets(ctx, int32(limit))
	if err != nil {
		return nil, status.Errorf(codes.Unavailable, "failed to list sports live markets: %v", err)
	}
	lastSuccessAt, err := s.store.GetSportsLiveLastSuccessAt(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unavailable, "failed to read sports live sync state: %v", err)
	}

	resp := &apiclient.ListPolymarketSportsLiveMarketsResponse{
		Items: make([]*v1alpha1.PolymarketSportsLiveMarketItem, 0, len(markets)),
		Stale: lastSuccessAt.IsZero() || s.now().Sub(lastSuccessAt) > 2*s.syncInterval,
	}
	if !lastSuccessAt.IsZero() {
		resp.FetchedAt = lastSuccessAt.Unix()
	}
	for _, market := range markets {
		lastUpdate := ""
		if !market.GammaUpdatedAt.IsZero() {
			lastUpdate = market.GammaUpdatedAt.UTC().Format(time.RFC3339)
		}
		resp.Items = append(resp.Items, &v1alpha1.PolymarketSportsLiveMarketItem{
			ConditionID:  market.ConditionID,
			MarketSlug:   market.MarketSlug,
			EventSlug:    market.EventSlug,
			Title:        market.Title,
			Image:        market.Image,
			Score:        market.Score,
			Period:       market.Period,
			Elapsed:      market.Elapsed,
			LastUpdate:   lastUpdate,
			LiquidityNum: market.LiquidityNum,
			VolumeNum:    market.VolumeNum,
		})
	}
	return resp, nil
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
