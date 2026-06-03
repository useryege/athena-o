package polymarket

import (
	"context"
	"encoding/json"
	"sort"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/useryege/athena/internal/polymarket/apiclient"
	"github.com/useryege/athena/pkg/apis/application/v1alpha1"
	utilpolymarket "github.com/useryege/athena/util/polymarket"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	hotMarketTargetLimit      = 500
	hotMarketCandidateLimit   = 650
	hotMarketPageLimit        = 100
	hotMarketMissingThreshold = 3
)

type hotMarketCandidate struct {
	item *v1alpha1.PolymarketHotMarketItem
	rank int
}

func (s *Service) runHotMarketDiscoveryLoop(ctx context.Context) {
	defer s.runWG.Done()
	if err := s.refreshHotMarkets(ctx); err != nil {
		log.WithError(err).Warn("initial polymarket hot market discovery failed")
	}
	ticker := time.NewTicker(s.hotMarketRefreshInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.refreshHotMarkets(ctx); err != nil {
				log.WithError(err).Warn("periodic polymarket hot market discovery failed")
			}
		}
	}
}

func (s *Service) ListPolymarketHotMarkets(ctx context.Context, req *apiclient.ListPolymarketHotMarketsRequest) (*apiclient.ListPolymarketHotMarketsResponse, error) {
	s.startStopMu.Lock()
	started := s.started
	s.startStopMu.Unlock()
	if !started {
		return nil, status.Error(codes.FailedPrecondition, "polymarket service is not running")
	}

	limit := defaultHotMarketListLimit
	if req != nil && req.GetLimit() > 0 {
		limit = int(req.GetLimit())
	}
	if limit < 1 || limit > maxHotMarketListLimit {
		return nil, status.Errorf(codes.InvalidArgument, "limit must be between 1 and %d", maxHotMarketListLimit)
	}

	if !s.hasHotMarkets() {
		syncCtx, cancel := context.WithTimeout(ctx, defaultHotMarketInitialSyncWait)
		err := s.refreshHotMarkets(syncCtx)
		cancel()
		if err != nil {
			return nil, status.Errorf(codes.Unavailable, "polymarket hot markets are unavailable: %v", err)
		}
	}

	resp := s.currentHotMarketResponse(limit)
	if resp == nil {
		return nil, status.Error(codes.Unavailable, "polymarket hot markets are unavailable")
	}
	return resp, nil
}

func (s *Service) refreshHotMarkets(ctx context.Context) error {
	_, err, _ := s.syncGroup.Do("hot-markets", func() (any, error) {
		candidates, fetchErr := s.fetchHotMarketCandidates(ctx)
		if fetchErr != nil {
			s.cacheMu.Lock()
			if len(s.hotMarketItems) > 0 || s.hotMarketFetched > 0 {
				s.hotMarketStale = true
				s.realtimeStale = true
				s.realtimeConnected = false
			}
			s.cacheMu.Unlock()
			return nil, fetchErr
		}

		s.cacheMu.Lock()
		fetchedAt := s.nowUnix()
		s.applyHotMarketCandidatesLocked(candidates)
		s.sampleHotMarketCandidatesLocked(candidates, fetchedAt)
		s.hotMarketFetched = fetchedAt
		s.hotMarketStale = false
		s.hotMarketCandidateCount = int32(len(candidates))
		s.cacheMu.Unlock()
		return nil, nil
	})
	return err
}

func (s *Service) fetchHotMarketCandidates(ctx context.Context) ([]hotMarketCandidate, error) {
	if s.gammaClient == nil {
		return nil, status.Error(codes.Unavailable, "polymarket gamma client is required")
	}

	items := make([]*v1alpha1.PolymarketHotMarketItem, 0, hotMarketCandidateLimit)
	cursor := ""
	closed := false
	ascending := false
	includeTag := true
	for len(items) < hotMarketCandidateLimit {
		limit := hotMarketPageLimit
		resp, err := s.gammaClient.ListMarketsKeyset(ctx, utilpolymarket.ListMarketsKeysetOptions{
			Limit:       &limit,
			Order:       "volume24hr",
			Ascending:   &ascending,
			AfterCursor: cursor,
			Closed:      &closed,
			IncludeTag:  &includeTag,
		})
		if err != nil {
			return nil, err
		}
		if resp == nil || len(resp.Markets) == 0 {
			break
		}
		for i := range resp.Markets {
			item, ok := mapHotMarket(resp.Markets[i])
			if !ok {
				continue
			}
			items = append(items, item)
			if len(items) >= hotMarketCandidateLimit {
				break
			}
		}
		if resp.NextCursor == nil || strings.TrimSpace(*resp.NextCursor) == "" {
			break
		}
		cursor = strings.TrimSpace(*resp.NextCursor)
	}

	sortHotMarkets(items)
	candidates := make([]hotMarketCandidate, 0, len(items))
	for i := range items {
		candidates = append(candidates, hotMarketCandidate{
			item: items[i],
			rank: i,
		})
	}
	return candidates, nil
}

func (s *Service) applyHotMarketCandidatesLocked(candidates []hotMarketCandidate) {
	byID := make(map[string]hotMarketCandidate, len(candidates))
	for i := range candidates {
		if candidates[i].item == nil || strings.TrimSpace(candidates[i].item.ConditionID) == "" {
			continue
		}
		byID[candidates[i].item.ConditionID] = candidates[i]
	}

	selected := make(map[string]struct{}, hotMarketCandidateLimit)
	nextMissing := make(map[string]int, hotMarketCandidateLimit)
	out := make([]*v1alpha1.PolymarketHotMarketItem, 0, hotMarketTargetLimit)
	addCandidate := func(candidate hotMarketCandidate) {
		if len(out) >= hotMarketCandidateLimit {
			return
		}
		if candidate.item == nil {
			return
		}
		id := strings.TrimSpace(candidate.item.ConditionID)
		if id == "" {
			return
		}
		if _, exists := selected[id]; exists {
			return
		}
		selected[id] = struct{}{}
		nextMissing[id] = 0
		out = append(out, cloneHotMarketItem(candidate.item))
	}

	for i := 0; i < len(candidates) && i < hotMarketTargetLimit; i++ {
		addCandidate(candidates[i])
	}

	for i := range s.hotMarketItems {
		previous := s.hotMarketItems[i]
		if previous == nil {
			continue
		}
		id := strings.TrimSpace(previous.ConditionID)
		if id == "" {
			continue
		}
		if _, exists := selected[id]; exists {
			continue
		}
		if candidate, exists := byID[id]; exists {
			addCandidate(candidate)
			continue
		}
		missing := s.hotMarketMissing[id] + 1
		if missing < hotMarketMissingThreshold {
			if len(out) >= hotMarketCandidateLimit {
				continue
			}
			selected[id] = struct{}{}
			nextMissing[id] = missing
			out = append(out, cloneHotMarketItem(previous))
		}
	}

	s.hotMarketItems = out
	s.hotMarketMissing = nextMissing
}

func mapHotMarket(market utilpolymarket.Market) (*v1alpha1.PolymarketHotMarketItem, bool) {
	if isExcludedHotMarketCategory(market) {
		return nil, false
	}
	if !boolValue(market.Active) || boolValue(market.Closed) || !boolValue(market.EnableOrderBook) {
		return nil, false
	}
	volume24hr := float64Value(market.Volume24hr)
	if volume24hr <= 0 {
		return nil, false
	}

	conditionID := strings.TrimSpace(stringValue(market.ConditionID))
	marketSlug := strings.TrimSpace(stringValue(market.Slug))
	question := strings.TrimSpace(stringValue(market.Question))
	if conditionID == "" || marketSlug == "" || question == "" {
		return nil, false
	}

	tokenIDs := parseHotMarketStringList(market.ClobTokenIDs)
	if len(tokenIDs) == 0 {
		return nil, false
	}
	outcomes := parseHotMarketStringList(market.Outcomes)
	prices := parseHotMarketStringList(market.OutcomePrices)

	tokens := make([]*v1alpha1.PolymarketHotMarketTokenItem, 0, len(tokenIDs))
	seenTokens := make(map[string]struct{}, len(tokenIDs))
	for i := range tokenIDs {
		tokenID := strings.TrimSpace(tokenIDs[i])
		if tokenID == "" {
			continue
		}
		if _, exists := seenTokens[tokenID]; exists {
			continue
		}
		seenTokens[tokenID] = struct{}{}

		outcome := "Outcome " + strconv.Itoa(i+1)
		if len(outcomes) == len(tokenIDs) && strings.TrimSpace(outcomes[i]) != "" {
			outcome = strings.TrimSpace(outcomes[i])
		}
		price := 0.0
		if len(prices) == len(tokenIDs) {
			price = parseFloatOrZero(prices[i])
		}
		tokens = append(tokens, &v1alpha1.PolymarketHotMarketTokenItem{
			TokenID: tokenID,
			Outcome: outcome,
			Price:   price,
		})
	}
	if len(tokens) == 0 {
		return nil, false
	}

	return &v1alpha1.PolymarketHotMarketItem{
		ConditionID:    conditionID,
		MarketSlug:     marketSlug,
		Question:       question,
		Image:          firstNonEmpty(strings.TrimSpace(stringValue(market.Image)), strings.TrimSpace(stringValue(market.Icon))),
		Volume24hr:     volume24hr,
		VolumeNum:      float64Value(market.VolumeNum),
		LiquidityNum:   float64Value(market.LiquidityNum),
		Spread:         float64Value(market.Spread),
		BestBid:        float64Value(market.BestBid),
		BestAsk:        float64Value(market.BestAsk),
		LastTradePrice: float64Value(market.LastTradePrice),
		UpdatedAt:      formatTimeRFC3339(market.UpdatedAt),
		Tokens:         tokens,
		EventSlug:      hotMarketEventSlug(market),
	}, true
}

func isExcludedHotMarketCategory(market utilpolymarket.Market) bool {
	if strings.TrimSpace(stringValue(market.SportsMarketType)) != "" {
		return true
	}
	for i := range market.Tags {
		if isExcludedHotMarketTag(market.Tags[i]) {
			return true
		}
	}
	for i := range market.Events {
		if isExcludedHotMarketCategoryValue(stringValue(market.Events[i].Category)) {
			return true
		}
		for j := range market.Events[i].Tags {
			if isExcludedHotMarketTag(market.Events[i].Tags[j]) {
				return true
			}
		}
	}
	return false
}

func isExcludedHotMarketTag(tag utilpolymarket.Tag) bool {
	return isExcludedHotMarketCategoryValue(stringValue(tag.Slug)) || isExcludedHotMarketCategoryValue(stringValue(tag.Label))
}

func isExcludedHotMarketCategoryValue(value string) bool {
	switch normalizeHotMarketCategory(value) {
	case "sports", "crypto", "esports":
		return true
	default:
		return false
	}
}

func normalizeHotMarketCategory(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	return strings.Map(func(r rune) rune {
		switch r {
		case ' ', '-', '_':
			return -1
		default:
			return r
		}
	}, value)
}

func hotMarketEventSlug(market utilpolymarket.Market) string {
	for i := range market.Events {
		slug := strings.TrimSpace(stringValue(market.Events[i].Slug))
		if slug != "" {
			return slug
		}
	}
	return ""
}

func parseHotMarketStringList(raw *string) []string {
	value := strings.TrimSpace(stringValue(raw))
	if value == "" {
		return []string{}
	}

	var stringValues []string
	if err := json.Unmarshal([]byte(value), &stringValues); err == nil {
		return cleanStringList(stringValues)
	}

	var mixedValues []any
	if err := json.Unmarshal([]byte(value), &mixedValues); err == nil {
		out := make([]string, 0, len(mixedValues))
		for i := range mixedValues {
			switch typed := mixedValues[i].(type) {
			case string:
				out = append(out, typed)
			case float64:
				out = append(out, strconv.FormatFloat(typed, 'f', -1, 64))
			case bool:
				out = append(out, strconv.FormatBool(typed))
			default:
				encoded, err := json.Marshal(typed)
				if err == nil {
					out = append(out, string(encoded))
				}
			}
		}
		return cleanStringList(out)
	}

	if strings.Contains(value, ",") {
		return cleanStringList(strings.Split(value, ","))
	}
	return cleanStringList([]string{value})
}

func cleanStringList(values []string) []string {
	out := make([]string, 0, len(values))
	for i := range values {
		value := strings.TrimSpace(values[i])
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}

func sortHotMarkets(items []*v1alpha1.PolymarketHotMarketItem) {
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Volume24hr != items[j].Volume24hr {
			return items[i].Volume24hr > items[j].Volume24hr
		}
		if items[i].LiquidityNum != items[j].LiquidityNum {
			return items[i].LiquidityNum > items[j].LiquidityNum
		}
		return strings.TrimSpace(items[i].ConditionID) < strings.TrimSpace(items[j].ConditionID)
	})
}

func parseFloatOrZero(value string) float64 {
	parsed, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	if err != nil {
		return 0
	}
	return parsed
}

func (s *Service) hasHotMarkets() bool {
	s.cacheMu.RLock()
	defer s.cacheMu.RUnlock()
	return len(s.hotMarketItems) > 0 || s.hotMarketFetched > 0
}

func (s *Service) currentHotMarketResponse(limit int) *apiclient.ListPolymarketHotMarketsResponse {
	s.cacheMu.RLock()
	defer s.cacheMu.RUnlock()
	if len(s.hotMarketItems) == 0 && s.hotMarketFetched == 0 {
		return nil
	}
	if limit > len(s.hotMarketItems) {
		limit = len(s.hotMarketItems)
	}

	items := make([]*v1alpha1.PolymarketHotMarketItem, 0, limit)
	for i := 0; i < limit; i++ {
		items = append(items, cloneHotMarketItem(s.hotMarketItems[i]))
	}
	return &apiclient.ListPolymarketHotMarketsResponse{
		Items:            items,
		FetchedAt:        s.hotMarketFetched,
		Stale:            s.hotMarketStale,
		MonitoredMarkets: int32(len(s.hotMarketItems)),
		MonitoredTokens:  int32(countHotMarketTokens(s.hotMarketItems)),
		CandidateCount:   s.hotMarketCandidateCount,
	}
}

func cloneHotMarketItem(in *v1alpha1.PolymarketHotMarketItem) *v1alpha1.PolymarketHotMarketItem {
	if in == nil {
		return nil
	}
	out := *in
	out.Tokens = make([]*v1alpha1.PolymarketHotMarketTokenItem, 0, len(in.Tokens))
	for i := range in.Tokens {
		if in.Tokens[i] == nil {
			continue
		}
		token := *in.Tokens[i]
		out.Tokens = append(out.Tokens, &token)
	}
	return &out
}

func countHotMarketTokens(items []*v1alpha1.PolymarketHotMarketItem) int {
	total := 0
	for i := range items {
		if items[i] == nil {
			continue
		}
		total += len(items[i].Tokens)
	}
	return total
}
