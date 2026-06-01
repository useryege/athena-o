package polymarket

import (
	"context"
	"encoding/json"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

const (
	integrationMainGate = "POLYMARKET_GAMMA_INTEGRATION"
	integrationLogGate  = "POLYMARKET_GAMMA_INTEGRATION_LOG_RESPONSE"

	integrationMarketsGate   = "POLYMARKET_GAMMA_INTEGRATION_MARKETS"
	integrationEventsGate    = "POLYMARKET_GAMMA_INTEGRATION_EVENTS"
	integrationTagsGate      = "POLYMARKET_GAMMA_INTEGRATION_TAGS"
	integrationSearchGate    = "POLYMARKET_GAMMA_INTEGRATION_SEARCH"
	integrationSportsGate    = "POLYMARKET_GAMMA_INTEGRATION_SPORTS"
	integrationCommunityGate = "POLYMARKET_GAMMA_INTEGRATION_COMMUNITY"
)

type integrationSamples struct {
	marketID    int64
	marketSlug  string
	eventID     int64
	eventSlug   string
	tagID       string
	tagSlug     string
	commentID   string
	userAddress string
	seriesID    int64
}

func TestIntegrationGamma(t *testing.T) {
	requireMainGate(t)

	client, err := NewGammaClient(GammaConfig{})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	var (
		samples      integrationSamples
		samplesReady bool
	)
	getSamples := func(t *testing.T) integrationSamples {
		t.Helper()
		if !samplesReady {
			samples = discoverIntegrationSamples(t, ctx, client)
			samplesReady = true
		}
		return samples
	}

	t.Run("Markets/ListMarkets", func(t *testing.T) {
		requireGroupGate(t, integrationMarketsGate)
		limit := 1
		got, err := client.ListMarkets(ctx, ListMarketsOptions{ListOptions: ListOptions{Limit: &limit}})
		if err != nil {
			t.Fatalf("ListMarkets: %v", err)
		}
		if len(got) == 0 {
			t.Fatal("ListMarkets returned no rows")
		}
		logIntegrationResponse(t, "Markets/ListMarkets", got)
	})

	t.Run("Markets/ListMarketsKeyset", func(t *testing.T) {
		requireGroupGate(t, integrationMarketsGate)
		limit := 1
		got, err := client.ListMarketsKeyset(ctx, ListMarketsKeysetOptions{Limit: &limit})
		if err != nil {
			t.Fatalf("ListMarketsKeyset: %v", err)
		}
		if len(got.Markets) == 0 {
			t.Log("ListMarketsKeyset returned empty data (allowed)")
		}
		logIntegrationResponse(t, "Markets/ListMarketsKeyset", got)
	})

	t.Run("Markets/GetMarketByID", func(t *testing.T) {
		requireGroupGate(t, integrationMarketsGate)
		samples := getSamples(t)
		got, err := client.GetMarketByID(ctx, samples.marketID, GetMarketOptions{})
		if err != nil {
			t.Fatalf("GetMarketByID: %v", err)
		}
		if got.ID == "" {
			t.Fatal("GetMarketByID returned empty id")
		}
		if got.ID != strconv.FormatInt(samples.marketID, 10) {
			t.Fatalf("GetMarketByID returned id=%q, want %d", got.ID, samples.marketID)
		}
		logIntegrationResponse(t, "Markets/GetMarketByID", got)
	})

	t.Run("Markets/GetMarketBySlug", func(t *testing.T) {
		requireGroupGate(t, integrationMarketsGate)
		samples := getSamples(t)
		got, err := client.GetMarketBySlug(ctx, samples.marketSlug, GetMarketOptions{})
		if err != nil {
			t.Fatalf("GetMarketBySlug: %v", err)
		}
		if got.ID == "" {
			t.Fatal("GetMarketBySlug returned empty id")
		}
		if got.ID != strconv.FormatInt(samples.marketID, 10) {
			t.Fatalf("GetMarketBySlug returned id=%q, want %d", got.ID, samples.marketID)
		}
		logIntegrationResponse(t, "Markets/GetMarketBySlug", got)
	})

	t.Run("Markets/GetMarketTagsByID", func(t *testing.T) {
		requireGroupGate(t, integrationMarketsGate)
		samples := getSamples(t)
		got, err := client.GetMarketTagsByID(ctx, samples.marketID)
		if err != nil {
			t.Fatalf("GetMarketTagsByID: %v", err)
		}
		for i, tag := range got {
			if strings.TrimSpace(tag.ID) == "" {
				t.Fatalf("GetMarketTagsByID returned empty tag id at index %d", i)
			}
		}
		logIntegrationResponse(t, "Markets/GetMarketTagsByID", got)
	})

	t.Run("Events/ListEvents", func(t *testing.T) {
		requireGroupGate(t, integrationEventsGate)
		limit := 1
		got, err := client.ListEvents(ctx, ListEventsOptions{ListOptions: ListOptions{Limit: &limit}})
		if err != nil {
			t.Fatalf("ListEvents: %v", err)
		}
		if len(got) == 0 {
			t.Fatal("ListEvents returned no rows")
		}
		logIntegrationResponse(t, "Events/ListEvents", got)
	})

	t.Run("Events/ListEventsKeyset", func(t *testing.T) {
		requireGroupGate(t, integrationEventsGate)
		limit := 1
		got, err := client.ListEventsKeyset(ctx, ListEventsKeysetOptions{Limit: &limit})
		if err != nil {
			t.Fatalf("ListEventsKeyset: %v", err)
		}
		if len(got.Events) == 0 {
			t.Log("ListEventsKeyset returned empty data (allowed)")
		}
		logIntegrationResponse(t, "Events/ListEventsKeyset", got)
	})

	t.Run("Events/GetEventByID", func(t *testing.T) {
		requireGroupGate(t, integrationEventsGate)
		samples := getSamples(t)
		got, err := client.GetEventByID(ctx, samples.eventID, GetEventOptions{})
		if err != nil {
			t.Fatalf("GetEventByID: %v", err)
		}
		if got.ID == "" {
			t.Fatal("GetEventByID returned empty id")
		}
		if got.ID != strconv.FormatInt(samples.eventID, 10) {
			t.Fatalf("GetEventByID returned id=%q, want %d", got.ID, samples.eventID)
		}
		logIntegrationResponse(t, "Events/GetEventByID", got)
	})

	t.Run("Events/GetEventBySlug", func(t *testing.T) {
		requireGroupGate(t, integrationEventsGate)
		samples := getSamples(t)
		got, err := client.GetEventBySlug(ctx, samples.eventSlug, GetEventOptions{})
		if err != nil {
			t.Fatalf("GetEventBySlug: %v", err)
		}
		if got.ID == "" {
			t.Fatal("GetEventBySlug returned empty id")
		}
		if got.ID != strconv.FormatInt(samples.eventID, 10) {
			t.Fatalf("GetEventBySlug returned id=%q, want %d", got.ID, samples.eventID)
		}
		logIntegrationResponse(t, "Events/GetEventBySlug", got)
	})

	t.Run("Events/GetEventTags", func(t *testing.T) {
		requireGroupGate(t, integrationEventsGate)
		samples := getSamples(t)
		got, err := client.GetEventTags(ctx, samples.eventID)
		if err != nil {
			t.Fatalf("GetEventTags: %v", err)
		}
		for i, tag := range got {
			if strings.TrimSpace(tag.ID) == "" {
				t.Fatalf("GetEventTags returned empty tag id at index %d", i)
			}
		}
		logIntegrationResponse(t, "Events/GetEventTags", got)
	})

	t.Run("Tags/ListTags", func(t *testing.T) {
		requireGroupGate(t, integrationTagsGate)
		limit := 1
		got, err := client.ListTags(ctx, ListTagsOptions{ListOptions: ListOptions{Limit: &limit}})
		if err != nil {
			t.Fatalf("ListTags: %v", err)
		}
		if len(got) == 0 {
			t.Fatal("ListTags returned no rows")
		}
		logIntegrationResponse(t, "Tags/ListTags", got)
	})

	t.Run("Tags/GetTagByID", func(t *testing.T) {
		requireGroupGate(t, integrationTagsGate)
		samples := getSamples(t)
		got, err := client.GetTagByID(ctx, samples.tagID, GetTagOptions{})
		if err != nil {
			t.Fatalf("GetTagByID: %v", err)
		}
		if got.ID == "" {
			t.Fatal("GetTagByID returned empty id")
		}
		if got.ID != samples.tagID {
			t.Fatalf("GetTagByID returned id=%q, want %q", got.ID, samples.tagID)
		}
		logIntegrationResponse(t, "Tags/GetTagByID", got)
	})

	t.Run("Tags/GetTagBySlug", func(t *testing.T) {
		requireGroupGate(t, integrationTagsGate)
		samples := getSamples(t)
		got, err := client.GetTagBySlug(ctx, samples.tagSlug, GetTagOptions{})
		if err != nil {
			t.Fatalf("GetTagBySlug: %v", err)
		}
		if got.ID == "" {
			t.Fatal("GetTagBySlug returned empty id")
		}
		if got.ID != samples.tagID {
			t.Fatalf("GetTagBySlug returned id=%q, want %q", got.ID, samples.tagID)
		}
		logIntegrationResponse(t, "Tags/GetTagBySlug", got)
	})

	t.Run("Tags/GetRelatedTagsByTagID", func(t *testing.T) {
		requireGroupGate(t, integrationTagsGate)
		samples := getSamples(t)
		got, err := client.GetRelatedTagsByTagID(ctx, samples.tagID, RelatedTagsOptions{})
		if err != nil {
			t.Fatalf("GetRelatedTagsByTagID: %v", err)
		}
		logIntegrationResponse(t, "Tags/GetRelatedTagsByTagID", got)
	})

	t.Run("Tags/GetRelatedTagsByTagSlug", func(t *testing.T) {
		requireGroupGate(t, integrationTagsGate)
		samples := getSamples(t)
		got, err := client.GetRelatedTagsByTagSlug(ctx, samples.tagSlug, RelatedTagsOptions{})
		if err != nil {
			t.Fatalf("GetRelatedTagsByTagSlug: %v", err)
		}
		logIntegrationResponse(t, "Tags/GetRelatedTagsByTagSlug", got)
	})

	t.Run("Tags/GetTagsRelatedToTagID", func(t *testing.T) {
		requireGroupGate(t, integrationTagsGate)
		samples := getSamples(t)
		got, err := client.GetTagsRelatedToTagID(ctx, samples.tagID, RelatedTagsOptions{})
		if err != nil {
			t.Fatalf("GetTagsRelatedToTagID: %v", err)
		}
		logIntegrationResponse(t, "Tags/GetTagsRelatedToTagID", got)
	})

	t.Run("Tags/GetTagsRelatedToTagSlug", func(t *testing.T) {
		requireGroupGate(t, integrationTagsGate)
		samples := getSamples(t)
		got, err := client.GetTagsRelatedToTagSlug(ctx, samples.tagSlug, RelatedTagsOptions{})
		if err != nil {
			t.Fatalf("GetTagsRelatedToTagSlug: %v", err)
		}
		logIntegrationResponse(t, "Tags/GetTagsRelatedToTagSlug", got)
	})

	t.Run("Comments/ListComments", func(t *testing.T) {
		requireGroupGate(t, integrationCommunityGate)
		samples := getSamples(t)
		limit := 1
		parentID := samples.eventID
		got, err := client.ListComments(ctx, ListCommentsOptions{
			ListOptions:      ListOptions{Limit: &limit},
			ParentEntityType: "Event",
			ParentEntityID:   &parentID,
		})
		if err != nil {
			t.Fatalf("ListComments: %v", err)
		}
		if len(got) == 0 {
			t.Log("ListComments returned empty data (allowed)")
		}
		logIntegrationResponse(t, "Comments/ListComments", got)
	})

	t.Run("Comments/GetCommentByID", func(t *testing.T) {
		requireGroupGate(t, integrationCommunityGate)
		samples := getSamples(t)
		if strings.TrimSpace(samples.commentID) == "" {
			t.Skip("no discovered comment id sample")
		}
		got, err := client.GetCommentByID(ctx, samples.commentID, GetCommentOptions{})
		if err != nil {
			t.Fatalf("GetCommentByID: %v", err)
		}
		logIntegrationResponse(t, "Comments/GetCommentByID", got)
	})

	t.Run("Comments/GetCommentsByUserAddress", func(t *testing.T) {
		requireGroupGate(t, integrationCommunityGate)
		samples := getSamples(t)
		if strings.TrimSpace(samples.userAddress) == "" {
			t.Skip("no discovered comment user address sample")
		}
		limit := 1
		got, err := client.GetCommentsByUserAddress(ctx, samples.userAddress, ListOptions{Limit: &limit})
		if err != nil {
			t.Fatalf("GetCommentsByUserAddress: %v", err)
		}
		logIntegrationResponse(t, "Comments/GetCommentsByUserAddress", got)
	})

	t.Run("Profiles/GetPublicProfile", func(t *testing.T) {
		requireGroupGate(t, integrationCommunityGate)
		samples := getSamples(t)
		if strings.TrimSpace(samples.userAddress) == "" {
			t.Skip("no discovered profile address sample")
		}
		got, err := client.GetPublicProfile(ctx, samples.userAddress)
		if err != nil {
			t.Fatalf("GetPublicProfile: %v", err)
		}
		logIntegrationResponse(t, "Profiles/GetPublicProfile", got)
	})

	t.Run("Series/ListSeries", func(t *testing.T) {
		requireGroupGate(t, integrationCommunityGate)
		limit := 1
		got, err := client.ListSeries(ctx, ListSeriesOptions{
			ListOptions: ListOptions{Limit: &limit},
		})
		if err != nil {
			t.Fatalf("ListSeries: %v", err)
		}
		if len(got) == 0 {
			t.Log("ListSeries returned empty data (allowed)")
		}
		logIntegrationResponse(t, "Series/ListSeries", got)
	})

	t.Run("Series/GetSeriesByID", func(t *testing.T) {
		requireGroupGate(t, integrationCommunityGate)
		samples := getSamples(t)
		if samples.seriesID == 0 {
			t.Skip("no discovered series id sample")
		}
		got, err := client.GetSeriesByID(ctx, samples.seriesID, GetSeriesOptions{})
		if err != nil {
			t.Fatalf("GetSeriesByID: %v", err)
		}
		if got.ID == "" {
			t.Fatal("GetSeriesByID returned empty id")
		}
		logIntegrationResponse(t, "Series/GetSeriesByID", got)
	})

	t.Run("Search/PublicSearch", func(t *testing.T) {
		requireGroupGate(t, integrationSearchGate)
		limit := 1
		got, err := client.PublicSearch(ctx, PublicSearchOptions{Q: "btc", LimitPerType: &limit})
		if err != nil {
			t.Fatalf("PublicSearch: %v", err)
		}
		if got.Pagination == nil {
			t.Log("PublicSearch returned nil pagination (allowed)")
		}
		logIntegrationResponse(t, "Search/PublicSearch", got)
	})

	t.Run("Sports/GetSportsMetadata", func(t *testing.T) {
		requireGroupGate(t, integrationSportsGate)
		got, err := client.GetSportsMetadata(ctx)
		if err != nil {
			t.Fatalf("GetSportsMetadata: %v", err)
		}
		logIntegrationResponse(t, "Sports/GetSportsMetadata", got)
	})

	t.Run("Sports/GetSportsMarketTypes", func(t *testing.T) {
		requireGroupGate(t, integrationSportsGate)
		got, err := client.GetSportsMarketTypes(ctx)
		if err != nil {
			t.Fatalf("GetSportsMarketTypes: %v", err)
		}
		if len(got.MarketTypes) == 0 {
			t.Fatal("GetSportsMarketTypes returned empty marketTypes")
		}
		logIntegrationResponse(t, "Sports/GetSportsMarketTypes", got)
	})

	t.Run("Sports/ListTeams", func(t *testing.T) {
		requireGroupGate(t, integrationSportsGate)
		limit := 1
		got, err := client.ListTeams(ctx, ListTeamsOptions{
			ListOptions: ListOptions{Limit: &limit},
			League:      []string{"nfl"},
		})
		if err != nil {
			t.Fatalf("ListTeams: %v", err)
		}
		if len(got) == 0 {
			t.Fatal("ListTeams returned no rows")
		}
		logIntegrationResponse(t, "Sports/ListTeams", got)
	})
}

func requireMainGate(t *testing.T) {
	t.Helper()
	if os.Getenv(integrationMainGate) != "1" {
		t.Skip("set POLYMARKET_GAMMA_INTEGRATION=1 to run")
	}
}

func requireGroupGate(t *testing.T, envKey string) {
	t.Helper()
	if os.Getenv(envKey) != "1" {
		t.Skipf("set %s=1 to run this group", envKey)
	}
}

func discoverIntegrationSamples(t *testing.T, ctx context.Context, client GammaClient) integrationSamples {
	t.Helper()

	limit := 1

	markets, err := client.ListMarkets(ctx, ListMarketsOptions{ListOptions: ListOptions{Limit: &limit}})
	if err != nil {
		t.Fatalf("discover ListMarkets: %v", err)
	}
	if len(markets) == 0 {
		t.Skip("discover ListMarkets returned no rows")
	}
	marketID, err := strconv.ParseInt(markets[0].ID, 10, 64)
	if err != nil {
		t.Fatalf("discover market id parse: %v", err)
	}
	if markets[0].Slug == nil || strings.TrimSpace(*markets[0].Slug) == "" {
		t.Fatal("discover market slug missing")
	}

	events, err := client.ListEvents(ctx, ListEventsOptions{ListOptions: ListOptions{Limit: &limit}})
	if err != nil {
		t.Fatalf("discover ListEvents: %v", err)
	}
	if len(events) == 0 {
		t.Skip("discover ListEvents returned no rows")
	}
	eventID, err := strconv.ParseInt(events[0].ID, 10, 64)
	if err != nil {
		t.Fatalf("discover event id parse: %v", err)
	}
	if events[0].Slug == nil || strings.TrimSpace(*events[0].Slug) == "" {
		t.Fatal("discover event slug missing")
	}

	tags, err := client.ListTags(ctx, ListTagsOptions{ListOptions: ListOptions{Limit: &limit}})
	if err != nil {
		t.Fatalf("discover ListTags: %v", err)
	}
	if len(tags) == 0 {
		t.Skip("discover ListTags returned no rows")
	}
	if strings.TrimSpace(tags[0].ID) == "" || tags[0].Slug == nil || strings.TrimSpace(*tags[0].Slug) == "" {
		t.Fatal("discover tag id or slug missing")
	}

	samples := integrationSamples{
		marketID:   marketID,
		marketSlug: *markets[0].Slug,
		eventID:    eventID,
		eventSlug:  *events[0].Slug,
		tagID:      tags[0].ID,
		tagSlug:    *tags[0].Slug,
	}

	commentsLimit := 1
	parentEntityID := samples.eventID
	comments, err := client.ListComments(ctx, ListCommentsOptions{
		ListOptions:      ListOptions{Limit: &commentsLimit},
		ParentEntityType: "Event",
		ParentEntityID:   &parentEntityID,
	})
	if err == nil && len(comments) > 0 {
		samples.commentID = strings.TrimSpace(comments[0].ID)
		if comments[0].UserAddress != nil {
			samples.userAddress = strings.TrimSpace(*comments[0].UserAddress)
		}
	}

	seriesLimit := 1
	seriesRows, err := client.ListSeries(ctx, ListSeriesOptions{
		ListOptions: ListOptions{Limit: &seriesLimit},
	})
	if err == nil && len(seriesRows) > 0 {
		if parsedSeriesID, parseErr := strconv.ParseInt(strings.TrimSpace(seriesRows[0].ID), 10, 64); parseErr == nil {
			samples.seriesID = parsedSeriesID
		}
	}

	logIntegrationResponse(t, "Discovery/Samples", map[string]any{
		"market_id":    samples.marketID,
		"market_slug":  samples.marketSlug,
		"event_id":     samples.eventID,
		"event_slug":   samples.eventSlug,
		"tag_id":       samples.tagID,
		"tag_slug":     samples.tagSlug,
		"comment_id":   samples.commentID,
		"user_address": samples.userAddress,
		"series_id":    samples.seriesID,
	})
	return samples
}

func shouldLogIntegrationResponse() bool {
	return os.Getenv(integrationLogGate) == "1"
}

func logIntegrationResponse(t *testing.T, endpoint string, payload any) {
	t.Helper()
	if !shouldLogIntegrationResponse() {
		return
	}
	b, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		t.Logf("%s response (marshal failed: %v): %+v", endpoint, err, payload)
		return
	}
	t.Logf("%s response:\n%s", endpoint, string(b))
}
