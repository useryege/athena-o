package polymarket

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

const (
	defaultSportsLiveSnapshotLimit      = 200
	maxSportsLiveSnapshotLimit          = 500
	defaultSportsLiveSnapshotSoonWindow = 24 * time.Hour

	// PolymarketEsportsTagID is the Gamma tag ID for Esports.
	PolymarketEsportsTagID int64 = 64
)

type SportsLiveSnapshotGammaClient interface {
	ListEventsKeyset(ctx context.Context, options ListEventsKeysetOptions) (*EventKeysetResponse, error)
}

type SportsLiveSnapshotOptions struct {
	LiveLimit   int
	SoonLimit   int
	SoonWindow  time.Duration
	MarketTypes []string
}

type SportsMarketGroup struct {
	Type    string
	Title   string
	Markets []Market
}

type SportsEventSnapshot struct {
	Slug              string
	Title             *string
	GameID            *int64
	EventDate         *string
	StartTime         *time.Time
	Live              *bool
	Ended             *bool
	Score             *string
	Period            *string
	Elapsed           *string
	GameStatus        *string
	FinishedTimestamp *string
	Sport             json.RawMessage
	Teams             []json.RawMessage
	UpdatedAt         *time.Time
	Markets           []SportsMarketGroup
}

type SportsLiveSnapshot struct {
	Live      []SportsEventSnapshot
	Soon      []SportsEventSnapshot
	FetchedAt time.Time
}

func BuildSportsLiveSnapshot(ctx context.Context, gammaClient SportsLiveSnapshotGammaClient, options SportsLiveSnapshotOptions) (*SportsLiveSnapshot, error) {
	return buildSportsLiveSnapshotWithNow(ctx, gammaClient, options.withDefaults(), time.Now().UTC())
}

func buildSportsLiveSnapshotWithNow(ctx context.Context, gammaClient SportsLiveSnapshotGammaClient, options SportsLiveSnapshotOptions, now time.Time) (*SportsLiveSnapshot, error) {
	if gammaClient == nil {
		return nil, fmt.Errorf("sports live snapshot gamma client is required")
	}

	liveEvents, err := listEventsKeysetPaged(ctx, gammaClient, ListEventsKeysetOptions{
		Live:         boolPtr(true),
		Closed:       boolPtr(false),
		TagSlug:      "sports",
		ExcludeTagID: []int64{PolymarketEsportsTagID},
	}, options.LiveLimit)
	if err != nil {
		return nil, fmt.Errorf("list sports live events: %w", err)
	}

	start := now
	end := now.Add(options.SoonWindow)
	soonEvents, err := listEventsKeysetPaged(ctx, gammaClient, ListEventsKeysetOptions{
		Closed:       boolPtr(false),
		TagSlug:      "sports",
		ExcludeTagID: []int64{PolymarketEsportsTagID},
		StartTimeMin: &start,
		StartTimeMax: &end,
	}, options.SoonLimit)
	if err != nil {
		return nil, fmt.Errorf("list sports soon events: %w", err)
	}

	marketTypes := normalizeMarketTypeSet(options.MarketTypes)
	live := buildSportsEventSnapshots(liveEvents, marketTypes)
	liveBySlug := make(map[string]struct{}, len(liveEvents))
	for i := range liveEvents {
		slug := strings.TrimSpace(stringPtrValue(liveEvents[i].Slug))
		if slug == "" {
			continue
		}
		liveBySlug[slug] = struct{}{}
	}

	soon := buildSportsEventSnapshots(soonEvents, marketTypes)
	soon = dedupeSoonSnapshots(soon, liveBySlug)

	sort.SliceStable(live, func(i, j int) bool {
		left := timePtrUnix(live[i].UpdatedAt)
		right := timePtrUnix(live[j].UpdatedAt)
		if left != right {
			return left > right
		}
		return live[i].Slug < live[j].Slug
	})
	sort.SliceStable(soon, func(i, j int) bool {
		left := timePtrUnix(soon[i].StartTime)
		right := timePtrUnix(soon[j].StartTime)
		if left != right {
			return left < right
		}
		return soon[i].Slug < soon[j].Slug
	})

	return &SportsLiveSnapshot{
		Live:      live,
		Soon:      soon,
		FetchedAt: now,
	}, nil
}

func (o SportsLiveSnapshotOptions) withDefaults() SportsLiveSnapshotOptions {
	if o.LiveLimit <= 0 {
		o.LiveLimit = defaultSportsLiveSnapshotLimit
	}
	if o.LiveLimit > maxSportsLiveSnapshotLimit {
		o.LiveLimit = maxSportsLiveSnapshotLimit
	}
	if o.SoonLimit <= 0 {
		o.SoonLimit = defaultSportsLiveSnapshotLimit
	}
	if o.SoonLimit > maxSportsLiveSnapshotLimit {
		o.SoonLimit = maxSportsLiveSnapshotLimit
	}
	if o.SoonWindow <= 0 {
		o.SoonWindow = defaultSportsLiveSnapshotSoonWindow
	}
	return o
}

func listEventsKeysetPaged(ctx context.Context, gammaClient SportsLiveSnapshotGammaClient, baseOptions ListEventsKeysetOptions, limit int) ([]Event, error) {
	if limit < 1 {
		return []Event{}, nil
	}

	items := make([]Event, 0, limit)
	afterCursor := ""
	for len(items) < limit {
		pageLimit := limit - len(items)
		if pageLimit > maxSportsLiveSnapshotLimit {
			pageLimit = maxSportsLiveSnapshotLimit
		}

		opts := baseOptions
		opts.Limit = &pageLimit
		opts.AfterCursor = afterCursor

		resp, err := gammaClient.ListEventsKeyset(ctx, opts)
		if err != nil {
			return nil, err
		}
		if resp == nil || len(resp.Events) == 0 {
			break
		}
		for i := range resp.Events {
			if len(items) >= limit {
				break
			}
			items = append(items, resp.Events[i])
		}
		if resp.NextCursor == nil {
			break
		}
		afterCursor = strings.TrimSpace(*resp.NextCursor)
		if afterCursor == "" {
			break
		}
	}
	return items, nil
}

func buildSportsEventSnapshots(events []Event, marketTypeSet map[string]struct{}) []SportsEventSnapshot {
	out := make([]SportsEventSnapshot, 0, len(events))
	for i := range events {
		snapshot, ok := buildSportsEventSnapshot(events[i], marketTypeSet)
		if !ok {
			continue
		}
		out = append(out, snapshot)
	}
	return out
}

func buildSportsEventSnapshot(event Event, marketTypeSet map[string]struct{}) (SportsEventSnapshot, bool) {
	slug := strings.TrimSpace(stringPtrValue(event.Slug))
	if slug == "" {
		return SportsEventSnapshot{}, false
	}

	groups := groupSportsMarkets(event.Markets, marketTypeSet)
	if len(groups) == 0 {
		return SportsEventSnapshot{}, false
	}

	return SportsEventSnapshot{
		Slug:              slug,
		Title:             event.Title,
		GameID:            event.GameID,
		EventDate:         event.EventDate,
		StartTime:         event.StartTime,
		Live:              event.Live,
		Ended:             event.Ended,
		Score:             event.Score,
		Period:            event.Period,
		Elapsed:           event.Elapsed,
		GameStatus:        event.GameStatus,
		FinishedTimestamp: event.FinishedTimestamp,
		Sport:             event.Sport,
		Teams:             event.Teams,
		UpdatedAt:         event.UpdatedAt,
		Markets:           groups,
	}, true
}

func groupSportsMarkets(markets []Market, marketTypeSet map[string]struct{}) []SportsMarketGroup {
	ordered := make([]SportsMarketGroup, 0)
	byKey := make(map[string]int)
	for i := range markets {
		marketType := normalizeMarketType(markets[i].SportsMarketType)
		if len(marketTypeSet) > 0 {
			if _, ok := marketTypeSet[marketType]; !ok {
				continue
			}
		}

		title := strings.TrimSpace(stringPtrValue(markets[i].GroupItemTitle))
		groupKey := marketType + "\x00" + title
		groupIndex, found := byKey[groupKey]
		if !found {
			ordered = append(ordered, SportsMarketGroup{
				Type:    marketType,
				Title:   title,
				Markets: make([]Market, 0, 4),
			})
			groupIndex = len(ordered) - 1
			byKey[groupKey] = groupIndex
		}
		ordered[groupIndex].Markets = append(ordered[groupIndex].Markets, markets[i])
	}

	return ordered
}

func dedupeSoonSnapshots(soon []SportsEventSnapshot, liveBySlug map[string]struct{}) []SportsEventSnapshot {
	if len(soon) == 0 {
		return soon
	}
	filtered := make([]SportsEventSnapshot, 0, len(soon))
	for i := range soon {
		if _, exists := liveBySlug[soon[i].Slug]; exists {
			continue
		}
		filtered = append(filtered, soon[i])
	}
	return filtered
}

func normalizeMarketTypeSet(marketTypes []string) map[string]struct{} {
	if len(marketTypes) == 0 {
		return nil
	}
	out := make(map[string]struct{}, len(marketTypes))
	for i := range marketTypes {
		out[normalizeMarketType(&marketTypes[i])] = struct{}{}
	}
	return out
}

func normalizeMarketType(value *string) string {
	return strings.ToLower(strings.TrimSpace(stringPtrValue(value)))
}

func stringPtrValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func timePtrUnix(value *time.Time) int64 {
	if value == nil {
		return 0
	}
	return value.Unix()
}

func boolPtr(value bool) *bool {
	return &value
}
