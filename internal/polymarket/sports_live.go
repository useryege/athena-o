package polymarket

import (
	"context"
	"sort"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/useryege/athena/internal/polymarket/apiclient"
	"github.com/useryege/athena/pkg/apis/application/v1alpha1"
	utilpolymarket "github.com/useryege/athena/util/polymarket"
)

type sportsLiveMarket struct {
	ConditionID  string
	MarketSlug   string
	EventSlug    string
	Title        string
	Image        string
	LiquidityNum float64
	VolumeNum    float64
}

type sportsLiveWSState struct {
	HasLive    bool
	Live       bool
	HasEnded   bool
	Ended      bool
	Score      string
	Period     string
	Elapsed    string
	LastUpdate string
}

func (s *Service) runFullSyncLoop(ctx context.Context) {
	defer s.runWG.Done()
	if err := s.refreshSportsLiveSnapshot(ctx); err != nil {
		log.WithError(err).Warn("initial polymarket sports live sync failed")
	}
	if err := s.refreshSportsLiveEventSnapshot(ctx); err != nil {
		log.WithError(err).Warn("initial polymarket sports live event sync failed")
	}
	ticker := time.NewTicker(s.syncInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.refreshSportsLiveSnapshot(ctx); err != nil {
				log.WithError(err).Warn("periodic polymarket sports live sync failed")
			}
			if err := s.refreshSportsLiveEventSnapshot(ctx); err != nil {
				log.WithError(err).Warn("periodic polymarket sports live event sync failed")
			}
		}
	}
}

func (s *Service) runSportsWSLoop(ctx context.Context) {
	defer s.runWG.Done()
	if s.sportsWSClient == nil {
		return
	}
	err := s.sportsWSClient.Run(ctx, utilpolymarket.SportsWSHandler{
		OnUpdate: s.applySportsWSUpdate,
		OnError: func(err error) {
			if err != nil {
				log.WithError(err).Debug("polymarket sports ws update error")
			}
		},
	})
	if err != nil && ctx.Err() == nil {
		log.WithError(err).Warn("polymarket sports ws loop exited unexpectedly")
	}
}

func (s *Service) refreshSportsLiveSnapshot(ctx context.Context) error {
	var alerts []sportsKickoffAlertCandidate
	_, err, _ := s.syncGroup.Do("sports-live-snapshot", func() (any, error) {
		markets, fetchErr := s.fetchSportsLiveMarkets(ctx)
		if fetchErr != nil {
			s.cacheMu.Lock()
			if len(s.baseMarkets) > 0 {
				s.snapshotStale = true
			}
			s.cacheMu.Unlock()
			return nil, fetchErr
		}
		s.cacheMu.Lock()
		s.baseMarkets = markets
		s.snapshotFetched = s.nowUnix()
		s.snapshotStale = false
		s.rebuildSnapshotLocked()
		alerts = s.collectSportsKickoffAlertsLocked(s.snapshotItems, s.snapshotFetched)
		s.cacheMu.Unlock()
		return nil, nil
	})
	if err == nil {
		s.sendSportsKickoffAlerts(ctx, alerts)
	}
	return err
}

func (s *Service) fetchSportsLiveMarkets(ctx context.Context) ([]sportsLiveMarket, error) {
	if s.gammaClient == nil {
		return nil, errNoSportsLiveSnapshot()
	}

	limit := s.eventPageLimit
	if limit <= 0 {
		limit = defaultSportsLiveEventPageLimit
	}

	items := make([]sportsLiveMarket, 0, limit)
	cursor := ""
	live := true
	closed := false
	for {
		opts := utilpolymarket.ListEventsKeysetOptions{
			Limit:        &limit,
			Live:         ptrBool(live),
			Closed:       ptrBool(closed),
			TagSlug:      "sports",
			ExcludeTagID: []int64{utilpolymarket.PolymarketEsportsTagID},
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
			if boolValue(event.Ended) || (event.Live != nil && !boolValue(event.Live)) {
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
				title := firstNonEmpty(
					strings.TrimSpace(stringValue(market.Question)),
					eventTitle,
				)
				image := firstNonEmpty(
					strings.TrimSpace(stringValue(market.Image)),
					strings.TrimSpace(stringValue(market.Icon)),
					eventImage,
				)
				items = append(items, sportsLiveMarket{
					ConditionID:  conditionID,
					MarketSlug:   marketSlug,
					EventSlug:    eventSlug,
					Title:        title,
					Image:        image,
					LiquidityNum: float64Value(market.LiquidityNum),
					VolumeNum:    float64Value(market.VolumeNum),
				})
			}
		}
		if resp.NextCursor == nil || strings.TrimSpace(*resp.NextCursor) == "" {
			break
		}
		cursor = strings.TrimSpace(*resp.NextCursor)
	}
	return items, nil
}

func (s *Service) applySportsWSUpdate(update utilpolymarket.SportsWSUpdate) {
	slug := strings.TrimSpace(update.Slug)
	if slug == "" {
		return
	}

	s.cacheMu.Lock()
	state := s.sportsWSState[slug]
	if update.Live != nil {
		state.HasLive = true
		state.Live = *update.Live
	}
	if update.Ended != nil {
		state.HasEnded = true
		state.Ended = *update.Ended
	}
	if update.Score != nil {
		state.Score = strings.TrimSpace(*update.Score)
	}
	if update.Period != nil {
		state.Period = strings.TrimSpace(*update.Period)
	}
	if update.Elapsed != nil {
		state.Elapsed = strings.TrimSpace(*update.Elapsed)
	}
	if update.LastUpdate != nil {
		state.LastUpdate = strings.TrimSpace(*update.LastUpdate)
	}
	s.sportsWSState[slug] = state
	s.rebuildSnapshotLocked()
	s.cacheMu.Unlock()
}

func (s *Service) rebuildSnapshotLocked() {
	items := make([]*v1alpha1.PolymarketSportsLiveMarketItem, 0, len(s.baseMarkets))
	for i := range s.baseMarkets {
		base := s.baseMarkets[i]
		state := s.sportsWSState[base.EventSlug]
		if state.HasEnded && state.Ended {
			continue
		}
		if state.HasLive && !state.Live {
			continue
		}
		items = append(items, &v1alpha1.PolymarketSportsLiveMarketItem{
			ConditionID:  base.ConditionID,
			MarketSlug:   base.MarketSlug,
			EventSlug:    base.EventSlug,
			Title:        base.Title,
			Image:        base.Image,
			Score:        state.Score,
			Period:       state.Period,
			Elapsed:      state.Elapsed,
			LastUpdate:   state.LastUpdate,
			LiquidityNum: base.LiquidityNum,
			VolumeNum:    base.VolumeNum,
		})
	}
	sort.SliceStable(items, func(i, j int) bool {
		leftTs := parseRFC3339Unix(items[i].LastUpdate)
		rightTs := parseRFC3339Unix(items[j].LastUpdate)
		if leftTs != rightTs {
			return leftTs > rightTs
		}
		if items[i].LiquidityNum != items[j].LiquidityNum {
			return items[i].LiquidityNum > items[j].LiquidityNum
		}
		return items[i].ConditionID < items[j].ConditionID
	})
	s.snapshotItems = items
}

func (s *Service) hasSnapshot() bool {
	s.cacheMu.RLock()
	defer s.cacheMu.RUnlock()
	return len(s.baseMarkets) > 0 || s.snapshotFetched > 0
}

func (s *Service) currentSportsLiveResponse(limit int) *apiclient.ListPolymarketSportsLiveMarketsResponse {
	s.cacheMu.RLock()
	defer s.cacheMu.RUnlock()
	if len(s.snapshotItems) == 0 && s.snapshotFetched == 0 {
		return nil
	}
	if limit > len(s.snapshotItems) {
		limit = len(s.snapshotItems)
	}
	result := make([]*v1alpha1.PolymarketSportsLiveMarketItem, 0, limit)
	for i := 0; i < limit; i++ {
		item := *s.snapshotItems[i]
		result = append(result, &item)
	}
	return &apiclient.ListPolymarketSportsLiveMarketsResponse{
		Items:     result,
		FetchedAt: s.snapshotFetched,
		Stale:     s.snapshotStale,
	}
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

func parseRFC3339Unix(value string) int64 {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0
	}
	t, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return 0
	}
	return t.Unix()
}
