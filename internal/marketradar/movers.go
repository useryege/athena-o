package marketradar

import (
	"context"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/useryege/athena/internal/marketradar/apiclient"
	"github.com/useryege/athena/pkg/apis/application/v1alpha1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	moverFreshAfter = 120 * time.Second
)

var moverWindowWeights = map[string]float64{
	"1m":  1.0,
	"5m":  0.6,
	"15m": 0.3,
}

func (s *Service) ListMarketMovers(ctx context.Context, req *apiclient.ListMarketMoversRequest) (*apiclient.ListMarketMoversResponse, error) {
	s.startStopMu.Lock()
	started := s.started
	s.startStopMu.Unlock()
	if !started {
		return nil, status.Error(codes.FailedPrecondition, "market radar service is not running")
	}

	limit := defaultMoverListLimit
	if req != nil && req.GetLimit() > 0 {
		limit = int(req.GetLimit())
	}
	if limit < 1 || limit > maxMoverListLimit {
		return nil, status.Errorf(codes.InvalidArgument, "limit must be between 1 and %d", maxMoverListLimit)
	}

	if !s.hasHotMarkets() {
		syncCtx, cancel := context.WithTimeout(ctx, defaultRealtimeInitialSyncWait)
		err := s.refreshHotMarkets(syncCtx)
		cancel()
		if err != nil {
			return nil, status.Errorf(codes.Unavailable, "polymarket movers are unavailable: %v", err)
		}
	}

	resp := s.currentMoverMarketResponse(limit)
	if resp == nil {
		return nil, status.Error(codes.Unavailable, "polymarket movers are unavailable")
	}
	return resp, nil
}

func (s *Service) currentMoverMarketResponse(limit int) *apiclient.ListMarketMoversResponse {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()

	markets := realtimeTopHotMarketsLocked(s.hotMarketItems)
	if len(markets) == 0 && s.hotMarketFetched == 0 {
		return nil
	}
	s.ensureRealtimeStatesForMarketsLocked(markets)

	now := s.now()
	nowUnix := now.Unix()
	items := make([]*v1alpha1.MarketRadarMoverMarketItem, 0, len(markets))
	for _, market := range markets {
		item := s.moverMarketItemLocked(market, nowUnix)
		if item == nil {
			continue
		}
		items = append(items, item)
	}

	sort.SliceStable(items, func(i, j int) bool {
		left, right := items[i], items[j]
		if left.Score != right.Score {
			return left.Score > right.Score
		}
		left1m := moverWindowAbsChange(left.Leader, "1m")
		right1m := moverWindowAbsChange(right.Leader, "1m")
		if left1m != right1m {
			return left1m > right1m
		}
		if left.Volume24hr != right.Volume24hr {
			return left.Volume24hr > right.Volume24hr
		}
		return left.ConditionID < right.ConditionID
	})
	if limit > len(items) {
		limit = len(items)
	}
	items = items[:limit]

	fetchedAt := s.realtimeFetched
	if fetchedAt == 0 {
		fetchedAt = s.hotMarketFetched
	}
	connected := s.realtimeConnected && (s.realtimeLastEventAt == 0 || nowUnix-s.realtimeLastEventAt <= int64(realtimeConnectedStaleAfter.Seconds()))
	return &apiclient.ListMarketMoversResponse{
		Items:            items,
		FetchedAt:        fetchedAt,
		Stale:            s.realtimeStale || s.hotMarketStale,
		Connected:        connected,
		LastEventAt:      s.realtimeLastEventAt,
		MonitoredMarkets: int32(len(markets)),
		MonitoredTokens:  int32(countHotMarketTokens(markets)),
		CandidateCount:   s.hotMarketCandidateCount,
	}
}

func (s *Service) moverMarketItemLocked(market *v1alpha1.MarketRadarHotMarketItem, nowUnix int64) *v1alpha1.MarketRadarMoverMarketItem {
	if market == nil {
		return nil
	}
	item := &v1alpha1.MarketRadarMoverMarketItem{
		ConditionID:  market.ConditionID,
		MarketSlug:   market.MarketSlug,
		EventSlug:    market.EventSlug,
		Question:     market.Question,
		Image:        market.Image,
		Volume24hr:   market.Volume24hr,
		VolumeNum:    market.VolumeNum,
		LiquidityNum: market.LiquidityNum,
		UpdatedAt:    market.UpdatedAt,
		Tokens:       make([]*v1alpha1.MarketRadarMoverTokenItem, 0, len(market.Tokens)),
	}
	for _, token := range market.Tokens {
		if token == nil {
			continue
		}
		tokenID := strings.TrimSpace(token.TokenID)
		if tokenID == "" {
			continue
		}
		state := s.realtimeStates[tokenID]
		tokenItem := s.moverTokenItemLocked(token, state, nowUnix)
		if tokenItem == nil {
			continue
		}
		item.Tokens = append(item.Tokens, tokenItem)
		if tokenItem.Score <= 0 {
			continue
		}
		if item.Leader == nil || tokenItem.Score > item.Leader.Score || (tokenItem.Score == item.Leader.Score && moverWindowAbsChange(tokenItem, "1m") > moverWindowAbsChange(item.Leader, "1m")) {
			item.Leader = tokenItem
			item.Score = tokenItem.Score
			item.Direction = tokenItem.Direction
		}
	}
	if item.Leader == nil || item.Score <= 0 || item.Direction == "" {
		return nil
	}
	return item
}

func (s *Service) moverTokenItemLocked(token *v1alpha1.MarketRadarHotMarketTokenItem, state *realtimeTokenState, nowUnix int64) *v1alpha1.MarketRadarMoverTokenItem {
	if state == nil {
		return nil
	}
	price := state.price
	outcome := token.Outcome
	outcome = firstNonEmpty(state.outcome, outcome)
	if price <= 0 || price > 1 {
		return nil
	}
	if state.lastEventAt <= 0 || nowUnix-state.lastEventAt > int64(moverFreshAfter.Seconds()) {
		return nil
	}

	windows := []*v1alpha1.MarketRadarMoverWindowItem{
		s.moverWindowItemLocked(token.TokenID, nowUnix, realtimeWindow1m, price),
		s.moverWindowItemLocked(token.TokenID, nowUnix, realtimeWindow5m, price),
		s.moverWindowItemLocked(token.TokenID, nowUnix, realtimeWindow15m, price),
	}
	score, direction := scoreMoverWindows(windows)
	item := &v1alpha1.MarketRadarMoverTokenItem{
		TokenID:        token.TokenID,
		Outcome:        outcome,
		Price:          price,
		BestBid:        state.bestBid,
		BestAsk:        state.bestAsk,
		Spread:         state.spread,
		LastTradePrice: state.lastTradePrice,
		LastEventAt:    state.lastEventAt,
		Windows:        windows,
		Score:          score,
		Direction:      direction,
		Warmup:         true,
	}
	for _, window := range windows {
		if window != nil && !window.Warmup {
			item.Warmup = false
			break
		}
	}
	return item
}

func (s *Service) moverWindowItemLocked(tokenID string, nowUnix int64, window time.Duration, currentPrice float64) *v1alpha1.MarketRadarMoverWindowItem {
	realtimeWindow := s.realtimeWindowItemLocked(tokenID, nowUnix, window, currentPrice)
	return &v1alpha1.MarketRadarMoverWindowItem{
		Window:        realtimeWindow.Window,
		PriceChangePp: realtimeWindow.PriceChangePp,
		Warmup:        realtimeWindow.Warmup,
		SampleCount:   realtimeWindow.SampleCount,
	}
}

func scoreMoverWindows(windows []*v1alpha1.MarketRadarMoverWindowItem) (float64, string) {
	score := 0.0
	bestContribution := 0.0
	direction := ""
	for _, window := range windows {
		if window == nil || window.Warmup {
			continue
		}
		weight := moverWindowWeights[window.Window]
		if weight <= 0 {
			continue
		}
		contribution := math.Abs(window.PriceChangePp) * weight
		score += contribution
		if contribution > bestContribution {
			bestContribution = contribution
			if window.PriceChangePp > 0 {
				direction = "up"
			} else if window.PriceChangePp < 0 {
				direction = "down"
			} else {
				direction = ""
			}
		}
	}
	if score <= 0 || direction == "" {
		return 0, ""
	}
	return score, direction
}

func moverWindowAbsChange(token *v1alpha1.MarketRadarMoverTokenItem, label string) float64 {
	if token == nil {
		return 0
	}
	for _, window := range token.Windows {
		if window != nil && window.Window == label && !window.Warmup {
			return math.Abs(window.PriceChangePp)
		}
	}
	return 0
}
