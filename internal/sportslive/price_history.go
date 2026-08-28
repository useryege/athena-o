package sportslive

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	sportslivestore "github.com/useryege/athena/internal/sportslive/store"
	utilpolymarket "github.com/useryege/athena/util/polymarket"
)

const (
	sportsLivePriceHistoryBackfillWindow = 6 * time.Hour
	sportsLivePriceHistoryOverlap        = 2 * time.Minute
	sportsLivePriceHistoryFidelity       = 1
	sportsLivePriceHistoryBatchLimit     = 20
)

type sportsLivePriceHistoryToken struct {
	tokenID   string
	marketKey string
	outcome   string
}

type sportsLivePriceHistoryPointKey struct {
	tokenID string
	priceTs int64
}

func (s *Service) runSportsLivePriceHistorySyncLoop(ctx context.Context) {
	defer s.runWG.Done()
	if err := s.syncSportsLivePriceHistory(ctx); err != nil {
		log.WithError(err).Warn("initial polymarket sports live price history sync failed")
	}
	ticker := time.NewTicker(defaultSportsLivePriceInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.syncSportsLivePriceHistory(ctx); err != nil {
				log.WithError(err).Warn("periodic polymarket sports live price history sync failed")
			}
		}
	}
}

func (s *Service) syncSportsLivePriceHistory(ctx context.Context) error {
	if s.clobClient == nil {
		return fmt.Errorf("polymarket clob client is required")
	}
	if s.store == nil {
		return fmt.Errorf("sports live store is required")
	}

	markets, err := s.store.ListSportsLiveMoneylineMarketsForPriceHistory(ctx)
	if err != nil {
		return err
	}
	tokens := sportsLivePriceHistoryTokens(markets)
	if len(tokens) == 0 {
		return nil
	}

	tokenIDs := make([]string, 0, len(tokens))
	for tokenID := range tokens {
		tokenIDs = append(tokenIDs, tokenID)
	}
	sort.Strings(tokenIDs)

	latestByToken, err := s.store.ListSportsLiveLatestPricePointTimes(ctx, tokenIDs)
	if err != nil {
		return err
	}

	now := s.now().UTC()
	groupedTokenIDs := make(map[int64][]string)
	for _, tokenID := range tokenIDs {
		start := now.Add(-sportsLivePriceHistoryBackfillWindow)
		if latest := latestByToken[tokenID]; !latest.IsZero() {
			start = latest.UTC().Add(-sportsLivePriceHistoryOverlap)
		}
		if start.After(now) {
			start = now.Add(-sportsLivePriceHistoryOverlap)
		}
		groupedTokenIDs[start.Unix()] = append(groupedTokenIDs[start.Unix()], tokenID)
	}

	starts := make([]int64, 0, len(groupedTokenIDs))
	for start := range groupedTokenIDs {
		starts = append(starts, start)
	}
	sort.Slice(starts, func(i, j int) bool { return starts[i] < starts[j] })
	for _, start := range starts {
		group := groupedTokenIDs[start]
		for offset := 0; offset < len(group); offset += sportsLivePriceHistoryBatchLimit {
			end := offset + sportsLivePriceHistoryBatchLimit
			if end > len(group) {
				end = len(group)
			}
			s.syncSportsLivePriceHistoryBatch(ctx, group[offset:end], tokens, start, now)
		}
	}
	s.updateSportsLivePriceAlerts(ctx)
	return nil
}

func (s *Service) syncSportsLivePriceHistoryBatch(ctx context.Context, tokenIDs []string, tokenByID map[string]sportsLivePriceHistoryToken, start int64, now time.Time) {
	if len(tokenIDs) == 0 {
		return
	}
	startTs := float64(start)
	endTs := float64(now.Unix())
	fidelity := sportsLivePriceHistoryFidelity
	resp, err := s.clobClient.GetBatchPricesHistory(ctx, utilpolymarket.CLOBBatchPricesHistoryRequest{
		Markets:  tokenIDs,
		StartTs:  &startTs,
		EndTs:    &endTs,
		Fidelity: &fidelity,
	})
	if err != nil {
		log.WithError(err).WithField("token_count", len(tokenIDs)).Warn("failed to fetch polymarket sports live price history batch")
		return
	}
	if resp == nil || len(resp.History) == 0 {
		return
	}

	fetchedAt := s.now().UTC()
	pointsByKey := make(map[sportsLivePriceHistoryPointKey]sportslivestore.SportsLivePricePoint)
	rawPointCount := 0
	for _, tokenID := range tokenIDs {
		token := tokenByID[tokenID]
		history := resp.History[tokenID]
		for _, point := range history {
			if point.T <= 0 || point.P < 0 || point.P > 1 {
				continue
			}
			rawPointCount++
			key := sportsLivePriceHistoryPointKey{tokenID: token.tokenID, priceTs: point.T}
			pointsByKey[key] = sportslivestore.SportsLivePricePoint{
				TokenID:   token.tokenID,
				MarketKey: token.marketKey,
				Outcome:   token.outcome,
				PriceTs:   time.Unix(point.T, 0).UTC(),
				Price:     point.P,
				FetchedAt: fetchedAt,
			}
		}
	}
	points := make([]sportslivestore.SportsLivePricePoint, 0, len(pointsByKey))
	for _, point := range pointsByKey {
		points = append(points, point)
	}
	if len(points) == 0 {
		return
	}
	if duplicateCount := rawPointCount - len(points); duplicateCount > 0 {
		log.WithFields(log.Fields{
			"raw_point_count":       rawPointCount,
			"deduped_point_count":   len(points),
			"duplicate_point_count": duplicateCount,
		}).Debug("deduped polymarket sports live price history batch")
	}
	upsertedCount, err := s.store.BatchUpsertSportsLivePricePoints(ctx, points)
	if err != nil {
		log.WithError(err).WithField("point_count", len(points)).Warn("failed to upsert polymarket sports live price history batch")
		return
	}
	if skippedCount := int64(len(points)) - upsertedCount; skippedCount > 0 {
		log.WithFields(log.Fields{
			"point_count":          len(points),
			"upserted_point_count": upsertedCount,
			"skipped_point_count":  skippedCount,
		}).Debug("skipped polymarket sports live price points for stale or deleting markets")
	}
}

func sportsLivePriceHistoryTokens(markets []sportslivestore.SportsLivePriceHistoryMarket) map[string]sportsLivePriceHistoryToken {
	out := make(map[string]sportsLivePriceHistoryToken)
	for _, market := range markets {
		tokenIDs := parseSportsLiveStringList(market.ClobTokenIDs)
		outcomes := parseSportsLiveStringList(market.Outcomes)
		for i := range tokenIDs {
			tokenID := strings.TrimSpace(tokenIDs[i])
			if tokenID == "" {
				continue
			}
			if _, exists := out[tokenID]; exists {
				continue
			}
			outcome := ""
			if i < len(outcomes) {
				outcome = strings.TrimSpace(outcomes[i])
			}
			out[tokenID] = sportsLivePriceHistoryToken{
				tokenID:   tokenID,
				marketKey: strings.TrimSpace(market.MarketKey),
				outcome:   outcome,
			}
		}
	}
	return out
}

func parseSportsLiveStringList(raw string) []string {
	value := strings.TrimSpace(raw)
	if value == "" {
		return nil
	}
	var items []string
	if err := json.Unmarshal([]byte(value), &items); err == nil {
		return items
	}
	return strings.Split(value, ",")
}
