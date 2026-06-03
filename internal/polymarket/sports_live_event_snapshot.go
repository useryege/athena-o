package polymarket

import (
	"context"
	"encoding/json"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/useryege/athena/internal/polymarket/apiclient"
	"github.com/useryege/athena/pkg/apis/application/v1alpha1"
	utilpolymarket "github.com/useryege/athena/util/polymarket"
)

const sportsLiveMoneylineMarketType = "moneyline"

type sportsLiveTeamRaw struct {
	Name     *string `json:"name,omitempty"`
	Logo     *string `json:"logo,omitempty"`
	Ordering *string `json:"ordering,omitempty"`
}

type sportsLiveEventFetchResult struct {
	Events   []*v1alpha1.PolymarketSportsLiveEventItem
	Snapshot *utilpolymarket.SportsLiveSnapshot
}

func (s *Service) refreshSportsLiveEventSnapshot(ctx context.Context) error {
	_, err, _ := s.syncGroup.Do("sports-live-event-snapshot", func() (any, error) {
		result, fetchErr := s.fetchSportsLiveEvents(ctx)
		if fetchErr != nil {
			s.cacheMu.Lock()
			if len(s.snapshotEvents) > 0 || s.eventFetched > 0 {
				s.eventStale = true
			}
			s.cacheMu.Unlock()
			return nil, fetchErr
		}
		events := result.Events
		sortSportsLiveEvents(events)
		nowUnix := s.nowUnix()
		s.cacheMu.Lock()
		s.snapshotEvents = events
		s.eventFetched = nowUnix
		s.eventStale = false
		alerts := s.collectSportsKickoffAlertsLocked(result.Snapshot, nowUnix)
		s.cacheMu.Unlock()
		s.sendSportsKickoffAlerts(ctx, alerts)
		return nil, nil
	})
	return err
}

func (s *Service) fetchSportsLiveEvents(ctx context.Context) (sportsLiveEventFetchResult, error) {
	if s.gammaClient == nil {
		return sportsLiveEventFetchResult{}, errNoSportsLiveEventSnapshot()
	}

	snapshot, err := utilpolymarket.BuildSportsLiveSnapshot(ctx, s.gammaClient, utilpolymarket.SportsLiveSnapshotOptions{
		LiveLimit: maxSportsLiveSnapshotLimit,
		SoonLimit: maxSportsLiveSnapshotLimit,
		MarketTypes: []string{
			sportsLiveMoneylineMarketType,
		},
	})
	if err != nil {
		return sportsLiveEventFetchResult{}, err
	}

	items := make([]*v1alpha1.PolymarketSportsLiveEventItem, 0, len(snapshot.Live))
	for i := range snapshot.Live {
		event := snapshot.Live[i]
		if !boolValue(event.Live) || boolValue(event.Ended) {
			continue
		}

		eventSlug := strings.TrimSpace(event.Slug)
		if eventSlug == "" {
			continue
		}

		groups := mapSportsLiveMarketGroups(event.Markets)
		if len(groups) == 0 {
			continue
		}

		items = append(items, &v1alpha1.PolymarketSportsLiveEventItem{
			EventSlug:  eventSlug,
			Title:      strings.TrimSpace(stringValue(event.Title)),
			Image:      pickSnapshotEventImage(event),
			Score:      strings.TrimSpace(stringValue(event.Score)),
			Period:     strings.TrimSpace(stringValue(event.Period)),
			Elapsed:    strings.TrimSpace(stringValue(event.Elapsed)),
			LastUpdate: formatTimeRFC3339(event.UpdatedAt),
			Live:       boolValue(event.Live),
			Ended:      boolValue(event.Ended),
			GameStatus: strings.TrimSpace(stringValue(event.GameStatus)),
			StartTime:  formatTimeRFC3339(event.StartTime),
			Markets:    groups,
			Teams:      mapSportsLiveTeams(event.Teams),
		})
	}

	return sportsLiveEventFetchResult{Events: items, Snapshot: snapshot}, nil
}

func mapSportsLiveTeams(rawTeams []json.RawMessage) []*v1alpha1.PolymarketSportsLiveTeamItem {
	out := make([]*v1alpha1.PolymarketSportsLiveTeamItem, 0, len(rawTeams))
	for i := range rawTeams {
		if len(rawTeams[i]) == 0 {
			continue
		}
		var team sportsLiveTeamRaw
		if err := json.Unmarshal(rawTeams[i], &team); err != nil {
			continue
		}
		name := strings.TrimSpace(stringValue(team.Name))
		logo := strings.TrimSpace(stringValue(team.Logo))
		ordering := strings.TrimSpace(stringValue(team.Ordering))
		if name == "" && logo == "" && ordering == "" {
			continue
		}
		out = append(out, &v1alpha1.PolymarketSportsLiveTeamItem{
			Name:     name,
			Logo:     logo,
			Ordering: ordering,
		})
	}
	return out
}

func mapSportsLiveMarketGroups(groups []utilpolymarket.SportsMarketGroup) []*v1alpha1.PolymarketSportsLiveMarketGroupItem {
	out := make([]*v1alpha1.PolymarketSportsLiveMarketGroupItem, 0, len(groups))
	for i := range groups {
		if strings.TrimSpace(strings.ToLower(groups[i].Type)) != sportsLiveMoneylineMarketType {
			continue
		}
		options := make([]*v1alpha1.PolymarketSportsLiveMarketOptionItem, 0, len(groups[i].Markets))
		for j := range groups[i].Markets {
			market := groups[i].Markets[j]
			options = append(options, &v1alpha1.PolymarketSportsLiveMarketOptionItem{
				ConditionID:    strings.TrimSpace(stringValue(market.ConditionID)),
				MarketSlug:     strings.TrimSpace(stringValue(market.Slug)),
				Question:       strings.TrimSpace(stringValue(market.Question)),
				Outcomes:       parseJSONStringList(market.Outcomes),
				OutcomePrices:  parseJSONStringList(market.OutcomePrices),
				BestBid:        float64Value(market.BestBid),
				BestAsk:        float64Value(market.BestAsk),
				LastTradePrice: float64Value(market.LastTradePrice),
				VolumeNum:      float64Value(market.VolumeNum),
				LiquidityNum:   float64Value(market.LiquidityNum),
			})
		}
		if len(options) == 0 {
			continue
		}
		out = append(out, &v1alpha1.PolymarketSportsLiveMarketGroupItem{
			Type:    strings.TrimSpace(groups[i].Type),
			Title:   strings.TrimSpace(groups[i].Title),
			Markets: options,
		})
	}
	return out
}

func pickSnapshotEventImage(event utilpolymarket.SportsEventSnapshot) string {
	for i := range event.Markets {
		for j := range event.Markets[i].Markets {
			image := strings.TrimSpace(stringValue(event.Markets[i].Markets[j].Image))
			if image != "" {
				return image
			}
			icon := strings.TrimSpace(stringValue(event.Markets[i].Markets[j].Icon))
			if icon != "" {
				return icon
			}
		}
	}
	return ""
}

func parseJSONStringList(raw *string) []string {
	value := strings.TrimSpace(stringValue(raw))
	if value == "" {
		return []string{}
	}

	var stringValues []string
	if err := json.Unmarshal([]byte(value), &stringValues); err == nil {
		return stringValues
	}

	var mixedValues []any
	if err := json.Unmarshal([]byte(value), &mixedValues); err != nil {
		return []string{}
	}

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
	return out
}

func formatTimeRFC3339(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}

func (s *Service) hasEventSnapshot() bool {
	s.cacheMu.RLock()
	defer s.cacheMu.RUnlock()
	return len(s.snapshotEvents) > 0 || s.eventFetched > 0
}

func (s *Service) currentSportsLiveEventResponse(limit int) *apiclient.GetPolymarketSportsLiveSnapshotResponse {
	s.cacheMu.RLock()
	defer s.cacheMu.RUnlock()
	if len(s.snapshotEvents) == 0 && s.eventFetched == 0 {
		return nil
	}
	if limit > len(s.snapshotEvents) {
		limit = len(s.snapshotEvents)
	}

	events := make([]*v1alpha1.PolymarketSportsLiveEventItem, 0, limit)
	for i := 0; i < limit; i++ {
		events = append(events, cloneSportsLiveEventItem(s.snapshotEvents[i]))
	}
	return &apiclient.GetPolymarketSportsLiveSnapshotResponse{
		Events:    events,
		FetchedAt: s.eventFetched,
		Stale:     s.eventStale,
	}
}

func cloneSportsLiveEventItem(in *v1alpha1.PolymarketSportsLiveEventItem) *v1alpha1.PolymarketSportsLiveEventItem {
	if in == nil {
		return nil
	}
	out := *in
	out.Teams = make([]*v1alpha1.PolymarketSportsLiveTeamItem, 0, len(in.Teams))
	for i := range in.Teams {
		if in.Teams[i] == nil {
			continue
		}
		team := *in.Teams[i]
		out.Teams = append(out.Teams, &team)
	}
	out.Markets = make([]*v1alpha1.PolymarketSportsLiveMarketGroupItem, 0, len(in.Markets))
	for i := range in.Markets {
		if in.Markets[i] == nil {
			continue
		}
		group := *in.Markets[i]
		group.Markets = make([]*v1alpha1.PolymarketSportsLiveMarketOptionItem, 0, len(in.Markets[i].Markets))
		for j := range in.Markets[i].Markets {
			if in.Markets[i].Markets[j] == nil {
				continue
			}
			option := *in.Markets[i].Markets[j]
			option.Outcomes = append([]string(nil), option.Outcomes...)
			option.OutcomePrices = append([]string(nil), option.OutcomePrices...)
			group.Markets = append(group.Markets, &option)
		}
		out.Markets = append(out.Markets, &group)
	}
	return &out
}

func sortSportsLiveEvents(events []*v1alpha1.PolymarketSportsLiveEventItem) {
	sort.SliceStable(events, func(i, j int) bool {
		leftVolume := sportsLiveEventTotalVolume(events[i])
		rightVolume := sportsLiveEventTotalVolume(events[j])
		if leftVolume != rightVolume {
			return leftVolume > rightVolume
		}

		leftUpdate := parseRFC3339Unix(strings.TrimSpace(events[i].LastUpdate))
		rightUpdate := parseRFC3339Unix(strings.TrimSpace(events[j].LastUpdate))
		if leftUpdate != rightUpdate {
			return leftUpdate > rightUpdate
		}

		return strings.TrimSpace(events[i].EventSlug) < strings.TrimSpace(events[j].EventSlug)
	})
}

func sportsLiveEventTotalVolume(event *v1alpha1.PolymarketSportsLiveEventItem) float64 {
	if event == nil {
		return 0
	}
	total := 0.0
	for i := range event.Markets {
		if event.Markets[i] == nil {
			continue
		}
		for j := range event.Markets[i].Markets {
			if event.Markets[i].Markets[j] == nil {
				continue
			}
			total += event.Markets[i].Markets[j].VolumeNum
		}
	}
	return total
}
