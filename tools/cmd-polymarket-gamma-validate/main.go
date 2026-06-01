package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/useryege/athena/util/polymarket"
)

type runResult struct {
	name string
	url  string
	err  error
}

type discovery struct {
	marketID   int64
	marketSlug string
	eventID    int64
	eventSlug  string
	tagID      string
	tagSlug    string
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	baseURL := strings.TrimSpace(os.Getenv("POLYMARKET_GAMMA_BASE_URL"))
	cfg := polymarket.GammaConfig{GammaBaseURL: baseURL, Timeout: 20 * time.Second}
	client, err := polymarket.NewGammaClient(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "create client: %v\n", err)
		os.Exit(1)
	}

	d, err := discover(ctx, client)
	if err != nil {
		fmt.Fprintf(os.Stderr, "discovery failed: %v\n", err)
		os.Exit(1)
	}

	resolvedBase := cfg.WithDefaults().GammaBaseURL
	httpClient := &http.Client{Timeout: 20 * time.Second}

	checks := []runResult{}
	appendCheck := func(name, rawURL string, out any) {
		err := strictDecodeFromURL(ctx, httpClient, rawURL, out)
		checks = append(checks, runResult{name: name, url: rawURL, err: err})
	}

	appendCheck("markets.list", buildURL(resolvedBase, "/markets", url.Values{"limit": {"1"}}), &[]polymarket.Market{})
	appendCheck("markets.getByID", buildURL(resolvedBase, "/markets/"+strconv.FormatInt(d.marketID, 10), nil), &polymarket.Market{})
	appendCheck("markets.getBySlug", buildURL(resolvedBase, "/markets/slug/"+url.PathEscape(d.marketSlug), nil), &polymarket.Market{})
	appendCheck("markets.getTags", buildURL(resolvedBase, "/markets/"+strconv.FormatInt(d.marketID, 10)+"/tags", nil), &[]polymarket.Tag{})

	appendCheck("events.list", buildURL(resolvedBase, "/events", url.Values{"limit": {"1"}}), &[]polymarket.Event{})
	appendCheck("events.getByID", buildURL(resolvedBase, "/events/"+strconv.FormatInt(d.eventID, 10), nil), &polymarket.Event{})
	appendCheck("events.getBySlug", buildURL(resolvedBase, "/events/slug/"+url.PathEscape(d.eventSlug), nil), &polymarket.Event{})
	appendCheck("events.getTags", buildURL(resolvedBase, "/events/"+strconv.FormatInt(d.eventID, 10)+"/tags", nil), &[]polymarket.Tag{})

	appendCheck("tags.list", buildURL(resolvedBase, "/tags", url.Values{"limit": {"1"}}), &[]polymarket.Tag{})
	appendCheck("tags.getByID", buildURL(resolvedBase, "/tags/"+url.PathEscape(d.tagID), nil), &polymarket.Tag{})
	appendCheck("tags.getBySlug", buildURL(resolvedBase, "/tags/slug/"+url.PathEscape(d.tagSlug), nil), &polymarket.Tag{})
	appendCheck("tags.relatedByID", buildURL(resolvedBase, "/tags/"+url.PathEscape(d.tagID)+"/related-tags", nil), &[]polymarket.RelatedTagRelationship{})
	appendCheck("tags.relatedBySlug", buildURL(resolvedBase, "/tags/slug/"+url.PathEscape(d.tagSlug)+"/related-tags", nil), &[]polymarket.RelatedTagRelationship{})
	appendCheck("tags.relatedTagsByID", buildURL(resolvedBase, "/tags/"+url.PathEscape(d.tagID)+"/related-tags/tags", nil), &[]polymarket.Tag{})
	appendCheck("tags.relatedTagsBySlug", buildURL(resolvedBase, "/tags/slug/"+url.PathEscape(d.tagSlug)+"/related-tags/tags", nil), &[]polymarket.Tag{})

	appendCheck("search.public", buildURL(resolvedBase, "/public-search", url.Values{"q": {"btc"}, "limit_per_type": {"1"}}), &polymarket.PublicSearchResponse{})

	appendCheck("sports.metadata", buildURL(resolvedBase, "/sports", nil), &[]polymarket.SportsMetadata{})
	appendCheck("sports.marketTypes", buildURL(resolvedBase, "/sports/market-types", nil), &polymarket.SportsMarketTypesResponse{})
	appendCheck("sports.teams", buildURL(resolvedBase, "/teams", url.Values{"league": {"nfl"}, "limit": {"1"}}), &[]polymarket.Team{})

	failed := 0
	for _, result := range checks {
		if result.err != nil {
			failed++
			fmt.Fprintf(os.Stderr, "FAIL %s\n  URL: %s\n  ERR: %v\n", result.name, result.url, result.err)
			continue
		}
		fmt.Printf("PASS %s\n", result.name)
	}

	if failed > 0 {
		fmt.Fprintf(os.Stderr, "\nvalidation finished with %d failure(s)\n", failed)
		os.Exit(1)
	}
	fmt.Println("\nvalidation finished with no failures")
}

func discover(ctx context.Context, client polymarket.GammaClient) (*discovery, error) {
	limit1 := 1

	markets, err := client.ListMarkets(ctx, polymarket.ListMarketsOptions{
		ListOptions: polymarket.ListOptions{Limit: &limit1},
	})
	if err != nil {
		return nil, fmt.Errorf("discover markets: %w", err)
	}
	if len(markets) == 0 {
		return nil, fmt.Errorf("discover markets: empty result")
	}
	marketID, err := strconv.ParseInt(markets[0].ID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("discover market id parse: %w", err)
	}
	if markets[0].Slug == nil || strings.TrimSpace(*markets[0].Slug) == "" {
		return nil, fmt.Errorf("discover market slug: missing")
	}

	events, err := client.ListEvents(ctx, polymarket.ListEventsOptions{
		ListOptions: polymarket.ListOptions{Limit: &limit1},
	})
	if err != nil {
		return nil, fmt.Errorf("discover events: %w", err)
	}
	if len(events) == 0 {
		return nil, fmt.Errorf("discover events: empty result")
	}
	eventID, err := strconv.ParseInt(events[0].ID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("discover event id parse: %w", err)
	}
	if events[0].Slug == nil || strings.TrimSpace(*events[0].Slug) == "" {
		return nil, fmt.Errorf("discover event slug: missing")
	}

	tags, err := client.ListTags(ctx, polymarket.ListTagsOptions{
		ListOptions: polymarket.ListOptions{Limit: &limit1},
	})
	if err != nil {
		return nil, fmt.Errorf("discover tags: %w", err)
	}
	if len(tags) == 0 {
		return nil, fmt.Errorf("discover tags: empty result")
	}
	if strings.TrimSpace(tags[0].ID) == "" || tags[0].Slug == nil || strings.TrimSpace(*tags[0].Slug) == "" {
		return nil, fmt.Errorf("discover tags: missing id or slug")
	}

	return &discovery{
		marketID:   marketID,
		marketSlug: *markets[0].Slug,
		eventID:    eventID,
		eventSlug:  *events[0].Slug,
		tagID:      tags[0].ID,
		tagSlug:    *tags[0].Slug,
	}, nil
}

func buildURL(baseURL, path string, query url.Values) string {
	u, _ := url.Parse(strings.TrimRight(baseURL, "/") + path)
	if len(query) > 0 {
		u.RawQuery = query.Encode()
	}
	return u.String()
}

func strictDecodeFromURL(ctx context.Context, httpClient *http.Client, endpoint string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, http.NoBody)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("http %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	dec := json.NewDecoder(resp.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(out); err != nil {
		return fmt.Errorf("strict decode failed: %w", err)
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return fmt.Errorf("strict decode trailing data: %v", err)
	}
	return nil
}
