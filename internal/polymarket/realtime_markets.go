package polymarket

import (
	"context"
	"strings"
	"time"

	"github.com/useryege/athena/internal/polymarket/apiclient"
	"github.com/useryege/athena/pkg/apis/application/v1alpha1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	realtimeWindowRetention     = 16 * time.Minute
	realtimeConnectedStaleAfter = 120 * time.Second
	realtimeWindow1m            = time.Minute
	realtimeWindow5m            = 5 * time.Minute
	realtimeWindow15m           = 15 * time.Minute
)

type realtimeTokenState struct {
	tokenID        string
	outcome        string
	price          float64
	bestBid        float64
	bestAsk        float64
	spread         float64
	lastTradePrice float64
	lastEventAt    int64
}

type realtimeSample struct {
	at    int64
	price float64
}

func (s *Service) ListPolymarketRealtimeMarkets(ctx context.Context, req *apiclient.ListPolymarketRealtimeMarketsRequest) (*apiclient.ListPolymarketRealtimeMarketsResponse, error) {
	s.startStopMu.Lock()
	started := s.started
	s.startStopMu.Unlock()
	if !started {
		return nil, status.Error(codes.FailedPrecondition, "polymarket service is not running")
	}

	limit := defaultRealtimeListLimit
	if req != nil && req.GetLimit() > 0 {
		limit = int(req.GetLimit())
	}
	if limit < 1 || limit > maxRealtimeListLimit {
		return nil, status.Errorf(codes.InvalidArgument, "limit must be between 1 and %d", maxRealtimeListLimit)
	}

	if !s.hasHotMarkets() {
		syncCtx, cancel := context.WithTimeout(ctx, defaultRealtimeInitialSyncWait)
		err := s.refreshHotMarkets(syncCtx)
		cancel()
		if err != nil {
			return nil, status.Errorf(codes.Unavailable, "polymarket realtime markets are unavailable: %v", err)
		}
	}

	resp := s.currentRealtimeMarketResponse(limit)
	if resp == nil {
		return nil, status.Error(codes.Unavailable, "polymarket realtime markets are unavailable")
	}
	return resp, nil
}

func (s *Service) currentRealtimeMarketResponse(limit int) *apiclient.ListPolymarketRealtimeMarketsResponse {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	markets := realtimeTopHotMarketsLocked(s.hotMarketItems)
	if len(markets) == 0 && s.hotMarketFetched == 0 {
		return nil
	}
	s.ensureRealtimeStatesForMarketsLocked(markets)
	if limit > len(markets) {
		limit = len(markets)
	}

	now := s.now()
	nowUnix := now.Unix()
	items := make([]*v1alpha1.PolymarketRealtimeMarketItem, 0, limit)
	for i := 0; i < limit; i++ {
		market := markets[i]
		item := &v1alpha1.PolymarketRealtimeMarketItem{
			ConditionID:  market.ConditionID,
			MarketSlug:   market.MarketSlug,
			EventSlug:    market.EventSlug,
			Question:     market.Question,
			Image:        market.Image,
			Volume24hr:   market.Volume24hr,
			VolumeNum:    market.VolumeNum,
			LiquidityNum: market.LiquidityNum,
			UpdatedAt:    market.UpdatedAt,
			Tokens:       make([]*v1alpha1.PolymarketRealtimeTokenItem, 0, len(market.Tokens)),
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
			item.Tokens = append(item.Tokens, s.realtimeTokenItemLocked(token, state, nowUnix))
		}
		items = append(items, item)
	}

	fetchedAt := s.realtimeFetched
	if fetchedAt == 0 {
		fetchedAt = s.hotMarketFetched
	}
	connected := s.realtimeConnected && (s.realtimeLastEventAt == 0 || nowUnix-s.realtimeLastEventAt <= int64(realtimeConnectedStaleAfter.Seconds()))
	return &apiclient.ListPolymarketRealtimeMarketsResponse{
		Items:             items,
		FetchedAt:         fetchedAt,
		Stale:             s.realtimeStale || s.hotMarketStale,
		SubscribedMarkets: int32(len(markets)),
		SubscribedTokens:  int32(countHotMarketTokens(markets)),
		Connected:         connected,
		LastEventAt:       s.realtimeLastEventAt,
		CandidateCount:    s.hotMarketCandidateCount,
	}
}

func (s *Service) realtimeTokenItemLocked(token *v1alpha1.PolymarketHotMarketTokenItem, state *realtimeTokenState, nowUnix int64) *v1alpha1.PolymarketRealtimeTokenItem {
	price := token.Price
	outcome := token.Outcome
	item := &v1alpha1.PolymarketRealtimeTokenItem{
		TokenID: token.TokenID,
		Outcome: outcome,
		Price:   price,
		Warmup:  true,
		Windows: []*v1alpha1.PolymarketRealtimeWindowItem{
			s.realtimeWindowItemLocked(token.TokenID, nowUnix, realtimeWindow1m, price),
			s.realtimeWindowItemLocked(token.TokenID, nowUnix, realtimeWindow5m, price),
			s.realtimeWindowItemLocked(token.TokenID, nowUnix, realtimeWindow15m, price),
		},
	}
	if state != nil {
		item.Outcome = firstNonEmpty(state.outcome, outcome)
		if state.price > 0 {
			item.Price = state.price
		}
		item.BestBid = state.bestBid
		item.BestAsk = state.bestAsk
		item.Spread = state.spread
		item.LastTradePrice = state.lastTradePrice
		item.LastEventAt = state.lastEventAt
		item.Windows = []*v1alpha1.PolymarketRealtimeWindowItem{
			s.realtimeWindowItemLocked(token.TokenID, nowUnix, realtimeWindow1m, item.Price),
			s.realtimeWindowItemLocked(token.TokenID, nowUnix, realtimeWindow5m, item.Price),
			s.realtimeWindowItemLocked(token.TokenID, nowUnix, realtimeWindow15m, item.Price),
		}
	}
	item.Warmup = true
	for _, window := range item.Windows {
		if window != nil && !window.Warmup {
			item.Warmup = false
			break
		}
	}
	return item
}

func (s *Service) realtimeWindowItemLocked(tokenID string, nowUnix int64, window time.Duration, currentPrice float64) *v1alpha1.PolymarketRealtimeWindowItem {
	samples := s.realtimeSamples[tokenID]
	windowSeconds := int64(window.Seconds())
	label := realtimeWindowLabel(window)
	out := &v1alpha1.PolymarketRealtimeWindowItem{
		Window:      label,
		Warmup:      true,
		SampleCount: int32(len(samples)),
	}
	if currentPrice <= 0 || len(samples) == 0 {
		return out
	}
	target := nowUnix - windowSeconds
	var start *realtimeSample
	for i := range samples {
		if samples[i].at <= target {
			start = &samples[i]
			continue
		}
		if start == nil {
			start = &samples[i]
		}
		break
	}
	if start == nil || start.price <= 0 || nowUnix-start.at < windowSeconds {
		return out
	}
	out.PriceChangePp = (currentPrice - start.price) * 100
	out.Warmup = false
	return out
}

func (s *Service) ensureRealtimeStatesForMarketsLocked(markets []*v1alpha1.PolymarketHotMarketItem) {
	for _, market := range markets {
		if market == nil {
			continue
		}
		for _, token := range market.Tokens {
			if token == nil {
				continue
			}
			tokenID := strings.TrimSpace(token.TokenID)
			if tokenID == "" {
				continue
			}
			state := s.ensureRealtimeStateLocked(tokenID)
			state.outcome = strings.TrimSpace(token.Outcome)
			if token.Price > 0 && token.Price <= 1 {
				state.price = token.Price
			}
			state.bestBid = market.BestBid
			state.bestAsk = market.BestAsk
			state.spread = market.Spread
			state.lastTradePrice = market.LastTradePrice
		}
	}
}

func (s *Service) sampleHotMarketCandidatesLocked(candidates []hotMarketCandidate, nowUnix int64) {
	cutoff := nowUnix - int64(realtimeWindowRetention.Seconds())
	for i := range candidates {
		market := candidates[i].item
		if market == nil {
			continue
		}
		for _, token := range market.Tokens {
			if token == nil {
				continue
			}
			tokenID := strings.TrimSpace(token.TokenID)
			if tokenID == "" || token.Price <= 0 || token.Price > 1 {
				continue
			}
			state := s.ensureRealtimeStateLocked(tokenID)
			state.outcome = strings.TrimSpace(token.Outcome)
			state.price = token.Price
			state.bestBid = market.BestBid
			state.bestAsk = market.BestAsk
			state.spread = market.Spread
			state.lastTradePrice = market.LastTradePrice
			state.lastEventAt = nowUnix
			s.realtimeSamples[tokenID] = appendOrReplaceRealtimeSample(s.realtimeSamples[tokenID], realtimeSample{at: nowUnix, price: token.Price})
		}
	}

	for tokenID, samples := range s.realtimeSamples {
		s.realtimeSamples[tokenID] = pruneRealtimeSamples(samples, cutoff)
	}
	s.cleanupRealtimeStatesForMarketsLocked(s.hotMarketItems)
	s.realtimeFetched = nowUnix
	s.realtimeStale = false
	s.realtimeConnected = len(candidates) > 0
	if len(candidates) > 0 {
		s.realtimeLastEventAt = nowUnix
	}
	s.realtimeSubscribedMarkets = int32(len(s.hotMarketItems))
	s.realtimeSubscribedTokens = int32(countHotMarketTokens(s.hotMarketItems))
}

func (s *Service) cleanupRealtimeStatesForMarketsLocked(markets []*v1alpha1.PolymarketHotMarketItem) {
	seen := make(map[string]struct{}, countHotMarketTokens(markets))
	for _, market := range markets {
		if market == nil {
			continue
		}
		for _, token := range market.Tokens {
			if token == nil {
				continue
			}
			tokenID := strings.TrimSpace(token.TokenID)
			if tokenID != "" {
				seen[tokenID] = struct{}{}
			}
		}
	}
	for tokenID := range s.realtimeStates {
		if _, exists := seen[tokenID]; !exists {
			delete(s.realtimeStates, tokenID)
			delete(s.realtimeSamples, tokenID)
		}
	}
}

func appendOrReplaceRealtimeSample(samples []realtimeSample, sample realtimeSample) []realtimeSample {
	if sample.at <= 0 || sample.price <= 0 || sample.price > 1 {
		return samples
	}
	insert := 0
	for insert < len(samples) && samples[insert].at < sample.at {
		insert++
	}
	if insert < len(samples) && samples[insert].at == sample.at {
		samples[insert] = sample
		return samples
	}
	samples = append(samples, realtimeSample{})
	copy(samples[insert+1:], samples[insert:])
	samples[insert] = sample
	return samples
}

func pruneRealtimeSamples(samples []realtimeSample, cutoff int64) []realtimeSample {
	first := 0
	for first < len(samples) && samples[first].at < cutoff {
		first++
	}
	if first > 0 {
		samples = append(samples[:0], samples[first:]...)
	}
	return samples
}

func (s *Service) ensureRealtimeStateLocked(tokenID string) *realtimeTokenState {
	state := s.realtimeStates[tokenID]
	if state == nil {
		state = &realtimeTokenState{tokenID: tokenID}
		s.realtimeStates[tokenID] = state
	}
	return state
}

func (s *Service) markRealtimeStale() {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	if len(s.realtimeStates) > 0 || s.realtimeFetched > 0 {
		s.realtimeStale = true
	}
	s.realtimeConnected = false
}

func (s *Service) now() time.Time {
	if s.nowFn == nil {
		return time.Now()
	}
	return s.nowFn()
}

func realtimeTopHotMarketsLocked(items []*v1alpha1.PolymarketHotMarketItem) []*v1alpha1.PolymarketHotMarketItem {
	limit := hotMarketTargetLimit
	if len(items) < limit {
		limit = len(items)
	}
	out := make([]*v1alpha1.PolymarketHotMarketItem, 0, limit)
	for i := 0; i < limit; i++ {
		out = append(out, cloneHotMarketItem(items[i]))
	}
	return out
}

func realtimeWindowLabel(window time.Duration) string {
	switch window {
	case realtimeWindow1m:
		return "1m"
	case realtimeWindow5m:
		return "5m"
	case realtimeWindow15m:
		return "15m"
	default:
		return window.String()
	}
}
