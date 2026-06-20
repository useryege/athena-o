package polymarket

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	polymarketstore "github.com/useryege/athena/internal/polymarket/store"
	utilpolymarket "github.com/useryege/athena/util/polymarket"
)

type sportsHistoryPriceHistoryToken struct {
	tokenID     string
	marketKey   string
	eventKey    string
	conditionID string
	outcome     string
	start       time.Time
	end         time.Time
}

func (s *Service) syncSportsHistoryPriceHistory(ctx context.Context) error {
	if s.clobClient == nil {
		return fmt.Errorf("polymarket clob client is required")
	}
	if s.store == nil {
		return fmt.Errorf("polymarket store is required")
	}
	markets, err := s.store.ListSportsHistoryMoneylineMarketsForPriceHistory(ctx)
	if err != nil {
		return err
	}
	tokens := sportsHistoryPriceHistoryTokens(markets)
	if len(tokens) == 0 {
		return nil
	}

	type intervalKey struct {
		start int64
		end   int64
	}
	grouped := make(map[intervalKey][]string)
	for tokenID, token := range tokens {
		key := intervalKey{start: token.start.Unix(), end: token.end.Unix()}
		grouped[key] = append(grouped[key], tokenID)
	}
	keys := make([]intervalKey, 0, len(grouped))
	for key := range grouped {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].start == keys[j].start {
			return keys[i].end < keys[j].end
		}
		return keys[i].start < keys[j].start
	})
	var batchErrors []error
	for _, key := range keys {
		tokenIDs := grouped[key]
		sort.Strings(tokenIDs)
		for offset := 0; offset < len(tokenIDs); offset += sportsLivePriceHistoryBatchLimit {
			end := offset + sportsLivePriceHistoryBatchLimit
			if end > len(tokenIDs) {
				end = len(tokenIDs)
			}
			if err := s.syncSportsHistoryPriceHistoryBatch(ctx, tokenIDs[offset:end], tokens, key.start, key.end); err != nil {
				batchErrors = append(batchErrors, err)
			}
		}
	}
	return errors.Join(batchErrors...)
}

func sportsHistoryPriceHistoryTokens(markets []polymarketstore.SportsHistoryPriceHistoryMarket) map[string]sportsHistoryPriceHistoryToken {
	out := make(map[string]sportsHistoryPriceHistoryToken)
	for _, market := range markets {
		tokenIDs := parseSportsLiveStringList(market.ClobTokenIDs)
		outcomes := parseSportsLiveStringList(market.Outcomes)
		end := market.FinishedAt
		if end.IsZero() || end.Before(market.StartTime) {
			end = market.StartTime
		}
		for i, value := range tokenIDs {
			tokenID := strings.TrimSpace(value)
			if tokenID == "" {
				continue
			}
			outcome := ""
			if i < len(outcomes) {
				outcome = strings.TrimSpace(outcomes[i])
			}
			out[tokenID] = sportsHistoryPriceHistoryToken{
				tokenID:     tokenID,
				marketKey:   market.MarketKey,
				eventKey:    market.EventKey,
				conditionID: market.ConditionID,
				outcome:     outcome,
				start:       market.StartTime,
				end:         end,
			}
		}
	}
	return out
}

func (s *Service) syncSportsHistoryPriceHistoryBatch(ctx context.Context, tokenIDs []string, tokenByID map[string]sportsHistoryPriceHistoryToken, start, end int64) error {
	if len(tokenIDs) == 0 || end < start {
		return nil
	}
	startTs := float64(start)
	endTs := float64(end)
	fidelity := sportsLivePriceHistoryFidelity
	resp, err := s.clobClient.GetBatchPricesHistory(ctx, utilpolymarket.CLOBBatchPricesHistoryRequest{
		Markets:  tokenIDs,
		StartTs:  &startTs,
		EndTs:    &endTs,
		Fidelity: &fidelity,
	})
	if err != nil {
		return fmt.Errorf("fetch sports history price batch for %d tokens: %w", len(tokenIDs), err)
	}
	if resp == nil || len(resp.History) == 0 {
		return nil
	}
	fetchedAt := s.now().UTC()
	pointsByKey := make(map[sportsLivePriceHistoryPointKey]polymarketstore.SportsLivePricePoint)
	for _, tokenID := range tokenIDs {
		token := tokenByID[tokenID]
		for _, point := range resp.History[tokenID] {
			if point.T <= 0 || point.P < 0 || point.P > 1 {
				continue
			}
			key := sportsLivePriceHistoryPointKey{tokenID: tokenID, priceTs: point.T}
			pointsByKey[key] = polymarketstore.SportsLivePricePoint{
				TokenID:     tokenID,
				MarketKey:   token.marketKey,
				EventKey:    token.eventKey,
				ConditionID: token.conditionID,
				Outcome:     token.outcome,
				PriceTs:     time.Unix(point.T, 0).UTC(),
				Price:       point.P,
				FetchedAt:   fetchedAt,
			}
		}
	}
	points := make([]polymarketstore.SportsLivePricePoint, 0, len(pointsByKey))
	for _, point := range pointsByKey {
		points = append(points, point)
	}
	if err := s.store.BatchUpsertSportsHistoryPricePoints(ctx, points); err != nil {
		return fmt.Errorf("upsert sports history price batch with %d points: %w", len(points), err)
	}
	return nil
}
