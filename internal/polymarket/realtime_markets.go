package polymarket

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"math"
	"strconv"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/useryege/athena/internal/polymarket/apiclient"
	"github.com/useryege/athena/pkg/apis/application/v1alpha1"
	utilpolymarket "github.com/useryege/athena/util/polymarket"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	realtimeSubscriptionBatchSize       = 250
	realtimeSubscriptionRefreshInterval = 5 * time.Second
	realtimeWindowRetention             = 16 * time.Minute
	realtimeConnectedStaleAfter         = 45 * time.Second
	realtimeWindow1m                    = time.Minute
	realtimeWindow5m                    = 5 * time.Minute
	realtimeWindow15m                   = 15 * time.Minute
)

type realtimeTokenState struct {
	tokenID        string
	outcome        string
	price          float64
	bestBid        float64
	bestAsk        float64
	spread         float64
	lastTradePrice float64
	lastTradeSize  float64
	lastTradeSide  string
	lastEventAt    int64
}

type realtimeSample struct {
	at    int64
	price float64
}

type realtimeSubscriptionSnapshot struct {
	markets        []*v1alpha1.PolymarketHotMarketItem
	tokenIDs       []string
	hash           string
	candidateCount int32
}

func (s *Service) runCLOBMarketWSLoop(ctx context.Context) {
	defer s.runWG.Done()

	var currentCancel context.CancelFunc
	var currentWG sync.WaitGroup
	currentHash := ""
	stopCurrent := func() {
		if currentCancel != nil {
			currentCancel()
			currentWG.Wait()
			currentCancel = nil
		}
		s.cacheMu.Lock()
		s.realtimeConnected = false
		s.cacheMu.Unlock()
	}
	defer stopCurrent()

	refreshTicker := time.NewTicker(realtimeSubscriptionRefreshInterval)
	defer refreshTicker.Stop()
	sampleTicker := time.NewTicker(s.realtimeSampleInterval)
	defer sampleTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-sampleTicker.C:
			s.sampleRealtime(s.now())
		case <-refreshTicker.C:
			snapshot := s.currentRealtimeSubscriptionSnapshot()
			if len(snapshot.tokenIDs) == 0 {
				if currentHash != "" {
					stopCurrent()
					currentHash = ""
				}
				continue
			}
			if snapshot.hash == currentHash {
				s.markRealtimeDisconnectedIfIdle(s.now())
				continue
			}
			stopCurrent()
			currentHash = snapshot.hash
			s.startRealtimeSubscriptions(ctx, snapshot.tokenIDs, &currentWG, &currentCancel)
			s.cacheMu.Lock()
			s.realtimeSubscribedHash = snapshot.hash
			s.realtimeSubscribedMarkets = int32(len(snapshot.markets))
			s.realtimeSubscribedTokens = int32(len(snapshot.tokenIDs))
			s.realtimeStale = false
			s.cacheMu.Unlock()
		}
	}
}

func (s *Service) startRealtimeSubscriptions(parent context.Context, tokenIDs []string, currentWG *sync.WaitGroup, currentCancel *context.CancelFunc) {
	ctx, cancel := context.WithCancel(parent)
	*currentCancel = cancel
	for _, batch := range chunkStrings(tokenIDs, realtimeSubscriptionBatchSize) {
		assetIDs := append([]string(nil), batch...)
		currentWG.Add(1)
		go func() {
			defer currentWG.Done()
			err := s.clobMarketWSClient.Run(ctx, utilpolymarket.CLOBMarketWSSubscription{
				AssetIDs:             assetIDs,
				InitialDump:          ptrBool(true),
				CustomFeatureEnabled: ptrBool(true),
			}, s.realtimeWSHandler())
			if err != nil && ctx.Err() == nil {
				log.WithError(err).Warn("polymarket clob market ws subscription stopped")
				s.markRealtimeStale()
			}
		}()
	}
}

func (s *Service) realtimeWSHandler() utilpolymarket.CLOBMarketWSHandler {
	return utilpolymarket.CLOBMarketWSHandler{
		OnBook: func(event utilpolymarket.CLOBMarketBookEvent) {
			s.applyRealtimeBook(event)
		},
		OnPriceChange: func(event utilpolymarket.CLOBMarketPriceChangeEvent) {
			s.applyRealtimePriceChange(event)
		},
		OnLastTrade: func(event utilpolymarket.CLOBMarketLastTradePriceEvent) {
			s.applyRealtimeLastTrade(event)
		},
		OnBestBidAsk: func(event utilpolymarket.CLOBMarketBestBidAskEvent) {
			s.applyRealtimeBestBidAsk(event)
		},
		OnHeartbeat: func(string) {
			s.markRealtimeConnected(s.nowUnix())
		},
		OnError: func(err error) {
			if err == nil {
				return
			}
			var decodeErr *utilpolymarket.CLOBMarketWSDecodeError
			if errors.As(err, &decodeErr) {
				log.WithError(err).Debug("polymarket clob market ws event decode error")
				return
			}
			log.WithError(err).Debug("polymarket clob market ws event error")
			s.markRealtimeStale()
		},
	}
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

func (s *Service) currentRealtimeSubscriptionSnapshot() realtimeSubscriptionSnapshot {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	markets := realtimeTopHotMarketsLocked(s.hotMarketItems)
	s.ensureRealtimeStatesForMarketsLocked(markets)

	seen := make(map[string]struct{}, len(markets)*2)
	tokenIDs := make([]string, 0, len(markets)*2)
	for _, market := range markets {
		for _, token := range market.Tokens {
			if token == nil {
				continue
			}
			tokenID := strings.TrimSpace(token.TokenID)
			if tokenID == "" {
				continue
			}
			if _, exists := seen[tokenID]; exists {
				continue
			}
			seen[tokenID] = struct{}{}
			tokenIDs = append(tokenIDs, tokenID)
		}
	}
	for tokenID := range s.realtimeStates {
		if _, exists := seen[tokenID]; !exists {
			delete(s.realtimeStates, tokenID)
			delete(s.realtimeSamples, tokenID)
		}
	}
	return realtimeSubscriptionSnapshot{
		markets:        markets,
		tokenIDs:       tokenIDs,
		hash:           hashStrings(tokenIDs),
		candidateCount: s.hotMarketCandidateCount,
	}
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
		item.Price = firstPositive(state.price, price)
		item.BestBid = state.bestBid
		item.BestAsk = state.bestAsk
		item.Spread = state.spread
		item.LastTradePrice = state.lastTradePrice
		item.LastTradeSize = state.lastTradeSize
		item.LastTradeSide = state.lastTradeSide
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
		for _, token := range market.Tokens {
			if token == nil {
				continue
			}
			tokenID := strings.TrimSpace(token.TokenID)
			if tokenID == "" {
				continue
			}
			state := s.realtimeStates[tokenID]
			if state == nil {
				state = &realtimeTokenState{tokenID: tokenID}
				s.realtimeStates[tokenID] = state
			}
			state.outcome = strings.TrimSpace(token.Outcome)
			if state.price <= 0 && token.Price > 0 {
				state.price = token.Price
			}
		}
	}
}

func (s *Service) sampleRealtime(now time.Time) {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	s.sampleRealtimeLocked(now)
}

func (s *Service) sampleRealtimeLocked(now time.Time) {
	nowUnix := now.Unix()
	cutoff := now.Add(-realtimeWindowRetention).Unix()
	for tokenID, state := range s.realtimeStates {
		if state == nil || state.price <= 0 {
			continue
		}
		samples := s.realtimeSamples[tokenID]
		if len(samples) > 0 && samples[len(samples)-1].at == nowUnix {
			samples[len(samples)-1].price = state.price
		} else {
			samples = append(samples, realtimeSample{at: nowUnix, price: state.price})
		}
		first := 0
		for first < len(samples) && samples[first].at < cutoff {
			first++
		}
		if first > 0 {
			samples = append(samples[:0], samples[first:]...)
		}
		s.realtimeSamples[tokenID] = samples
	}
	if len(s.realtimeStates) > 0 {
		s.realtimeFetched = nowUnix
	}
}

func (s *Service) applyRealtimeBook(event utilpolymarket.CLOBMarketBookEvent) {
	tokenID := strings.TrimSpace(event.AssetID)
	if tokenID == "" {
		return
	}
	bestBid := bestBidFromOrders(event.Bids)
	bestAsk := bestAskFromOrders(event.Asks)
	eventAt := s.marketEventUnix(event.Timestamp)
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	state := s.ensureRealtimeStateLocked(tokenID)
	if bestBid > 0 {
		state.bestBid = bestBid
	}
	if bestAsk > 0 {
		state.bestAsk = bestAsk
	}
	state.spread = spreadFromBidAsk(state.bestBid, state.bestAsk)
	if state.price <= 0 {
		state.price = midpointFromBidAsk(state.bestBid, state.bestAsk)
	}
	s.markRealtimeEventLocked(eventAt)
}

func (s *Service) applyRealtimePriceChange(event utilpolymarket.CLOBMarketPriceChangeEvent) {
	eventAt := s.marketEventUnix(event.Timestamp)
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	for _, change := range event.PriceChanges {
		tokenID := strings.TrimSpace(change.AssetID)
		if tokenID == "" {
			continue
		}
		state := s.ensureRealtimeStateLocked(tokenID)
		if price := parseFloatOrZero(change.Price); price > 0 {
			state.price = price
		}
		if change.BestBid != nil {
			if bestBid := parseFloatOrZero(*change.BestBid); bestBid > 0 {
				state.bestBid = bestBid
			}
		}
		if change.BestAsk != nil {
			if bestAsk := parseFloatOrZero(*change.BestAsk); bestAsk > 0 {
				state.bestAsk = bestAsk
			}
		}
		state.spread = spreadFromBidAsk(state.bestBid, state.bestAsk)
		state.lastEventAt = eventAt
	}
	s.markRealtimeEventLocked(eventAt)
}

func (s *Service) applyRealtimeLastTrade(event utilpolymarket.CLOBMarketLastTradePriceEvent) {
	tokenID := strings.TrimSpace(event.AssetID)
	if tokenID == "" {
		return
	}
	eventAt := s.marketEventUnix(event.Timestamp)
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	state := s.ensureRealtimeStateLocked(tokenID)
	if price := parseFloatOrZero(event.Price); price > 0 {
		state.lastTradePrice = price
		if state.price <= 0 {
			state.price = price
		}
	}
	if size := parseFloatOrZero(event.Size); size > 0 {
		state.lastTradeSize = size
	}
	state.lastTradeSide = strings.TrimSpace(event.Side)
	state.lastEventAt = eventAt
	s.markRealtimeEventLocked(eventAt)
}

func (s *Service) applyRealtimeBestBidAsk(event utilpolymarket.CLOBMarketBestBidAskEvent) {
	tokenID := strings.TrimSpace(event.AssetID)
	if tokenID == "" {
		return
	}
	eventAt := s.marketEventUnix(event.Timestamp)
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	state := s.ensureRealtimeStateLocked(tokenID)
	if bestBid := parseFloatOrZero(event.BestBid); bestBid > 0 {
		state.bestBid = bestBid
	}
	if bestAsk := parseFloatOrZero(event.BestAsk); bestAsk > 0 {
		state.bestAsk = bestAsk
	}
	if spread := parseFloatOrZero(event.Spread); spread > 0 {
		state.spread = spread
	} else {
		state.spread = spreadFromBidAsk(state.bestBid, state.bestAsk)
	}
	if state.price <= 0 {
		state.price = midpointFromBidAsk(state.bestBid, state.bestAsk)
	}
	state.lastEventAt = eventAt
	s.markRealtimeEventLocked(eventAt)
}

func (s *Service) ensureRealtimeStateLocked(tokenID string) *realtimeTokenState {
	state := s.realtimeStates[tokenID]
	if state == nil {
		state = &realtimeTokenState{tokenID: tokenID}
		s.realtimeStates[tokenID] = state
	}
	return state
}

func (s *Service) markRealtimeEventLocked(eventAt int64) {
	if eventAt <= 0 {
		eventAt = s.nowUnix()
	}
	s.realtimeConnected = true
	s.realtimeStale = false
	s.realtimeLastEventAt = eventAt
	s.realtimeFetched = eventAt
}

func (s *Service) markRealtimeConnected(eventAt int64) {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	s.markRealtimeEventLocked(eventAt)
}

func (s *Service) markRealtimeStale() {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	if len(s.realtimeStates) > 0 || s.realtimeFetched > 0 {
		s.realtimeStale = true
	}
	s.realtimeConnected = false
}

func (s *Service) markRealtimeDisconnectedIfIdle(now time.Time) {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	if s.realtimeLastEventAt > 0 && now.Unix()-s.realtimeLastEventAt > int64(realtimeConnectedStaleAfter.Seconds()) {
		s.realtimeConnected = false
	}
}

func (s *Service) marketEventUnix(raw string) int64 {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return s.nowUnix()
	}
	if parsed, err := strconv.ParseInt(raw, 10, 64); err == nil {
		if parsed > 1_000_000_000_000 {
			return parsed / 1000
		}
		return parsed
	}
	if parsed, err := time.Parse(time.RFC3339Nano, raw); err == nil {
		return parsed.Unix()
	}
	return s.nowUnix()
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

func chunkStrings(values []string, size int) [][]string {
	if size <= 0 || len(values) == 0 {
		return nil
	}
	chunks := make([][]string, 0, (len(values)+size-1)/size)
	for start := 0; start < len(values); start += size {
		end := start + size
		if end > len(values) {
			end = len(values)
		}
		chunks = append(chunks, values[start:end])
	}
	return chunks
}

func hashStrings(values []string) string {
	h := sha1.New()
	for _, value := range values {
		h.Write([]byte(value))
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))
}

func bestBidFromOrders(orders []utilpolymarket.CLOBOrderSummary) float64 {
	best := 0.0
	for _, order := range orders {
		price := parseFloatOrZero(order.Price)
		if price > best {
			best = price
		}
	}
	return best
}

func bestAskFromOrders(orders []utilpolymarket.CLOBOrderSummary) float64 {
	best := math.MaxFloat64
	for _, order := range orders {
		price := parseFloatOrZero(order.Price)
		if price > 0 && price < best {
			best = price
		}
	}
	if best == math.MaxFloat64 {
		return 0
	}
	return best
}

func spreadFromBidAsk(bestBid, bestAsk float64) float64 {
	if bestBid <= 0 || bestAsk <= 0 || bestAsk < bestBid {
		return 0
	}
	return bestAsk - bestBid
}

func midpointFromBidAsk(bestBid, bestAsk float64) float64 {
	if bestBid <= 0 || bestAsk <= 0 || bestAsk < bestBid {
		return 0
	}
	return (bestBid + bestAsk) / 2
}

func firstPositive(values ...float64) float64 {
	for _, value := range values {
		if value > 0 {
			return value
		}
	}
	return 0
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
