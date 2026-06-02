package polymarket

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewClientDefaults(t *testing.T) {
	c, err := NewGammaClient(GammaConfig{})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	impl, ok := c.(*gammaClientImpl)
	if !ok {
		t.Fatalf("client type = %T, want *gammaClientImpl", c)
	}
	if impl.config.GammaBaseURL != DefaultGammaBaseURL {
		t.Fatalf("GammaBaseURL = %q, want %q", impl.config.GammaBaseURL, DefaultGammaBaseURL)
	}
	if impl.config.Timeout != DefaultTimeout {
		t.Fatalf("Timeout = %s, want %s", impl.config.Timeout, DefaultTimeout)
	}
}

func TestNewClientInvalidBaseURL(t *testing.T) {
	_, err := NewGammaClient(GammaConfig{GammaBaseURL: "://bad"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestListTeamsQueryEncoding(t *testing.T) {
	var gotPath string
	var gotQuery string

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"id":1,"name":"Dallas Cowboys"}]`))
	}))
	defer ts.Close()

	client, err := NewGammaClient(GammaConfig{GammaBaseURL: ts.URL, Timeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	asc := false
	limit := 2
	teams, err := client.ListTeams(context.Background(), ListTeamsOptions{
		ListOptions: ListOptions{Limit: &limit, Ascending: &asc, Order: "createdAt"},
		League:      []string{"nfl", "nba"},
		Name:        []string{"Dallas Cowboys"},
	})
	if err != nil {
		t.Fatalf("ListTeams: %v", err)
	}
	if len(teams) != 1 {
		t.Fatalf("len(teams) = %d, want 1", len(teams))
	}
	if gotPath != "/teams" {
		t.Fatalf("path = %q, want /teams", gotPath)
	}
	for _, expected := range []string{"limit=2", "ascending=false", "order=createdAt", "league=nfl", "league=nba", "name=Dallas+Cowboys"} {
		if !strings.Contains(gotQuery, expected) {
			t.Fatalf("query %q missing %q", gotQuery, expected)
		}
	}
}

func TestPathEscaping(t *testing.T) {
	var gotPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.EscapedPath()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"1"}`))
	}))
	defer ts.Close()

	client, err := NewGammaClient(GammaConfig{GammaBaseURL: ts.URL})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	_, err = client.GetTagBySlug(context.Background(), "a/b c", GetTagOptions{})
	if err != nil {
		t.Fatalf("GetTagBySlug: %v", err)
	}
	if gotPath != "/tags/slug/a%2Fb%20c" {
		t.Fatalf("path = %q, want %q", gotPath, "/tags/slug/a%2Fb%20c")
	}
}

func TestGammaKeysetEndpointsQueryAndDecode(t *testing.T) {
	type call struct {
		Path  string
		Query string
	}
	calls := make([]call, 0, 2)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, call{Path: r.URL.Path, Query: r.URL.RawQuery})
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/markets/keyset":
			_, _ = w.Write([]byte(`{"markets":[{"id":"1","sportsMarketType":"moneyline","groupItemTitle":"Main"}],"next_cursor":"mk1"}`))
		case "/events/keyset":
			_, _ = w.Write([]byte(`{"events":[{"id":"2","slug":"nfl-lac-buf-2025-01-26","live":true,"ended":false,"score":"3-16","period":"Q4","elapsed":"5:18","finishedTimestamp":"2025-01-26T23:00:00Z","gameId":19439,"eventDate":"2025-01-26","startTime":"2025-01-26T20:30:00Z","gameStatus":"InProgress","sport":{"id":1,"sport":"nfl"},"teams":[{"id":1,"abbreviation":"LAC"},{"id":2,"abbreviation":"BUF"}]}],"next_cursor":"ev1"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()

	client, err := NewGammaClient(GammaConfig{GammaBaseURL: ts.URL, Timeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("NewGammaClient: %v", err)
	}

	limit := 2
	ascending := false
	includeTag := true
	closed := false
	relatedTags := true
	markets, err := client.ListMarketsKeyset(context.Background(), ListMarketsKeysetOptions{
		Limit:       &limit,
		Order:       "volume_num",
		Ascending:   &ascending,
		AfterCursor: "cursor-1",
		ID:          []int64{11, 22},
		Slug:        []string{"slug-a"},
		TagID:       []int64{3, 4},
		RelatedTags: &relatedTags,
		Closed:      &closed,
		IncludeTag:  &includeTag,
	})
	if err != nil {
		t.Fatalf("ListMarketsKeyset: %v", err)
	}
	if markets.NextCursor == nil || *markets.NextCursor != "mk1" {
		t.Fatalf("markets next_cursor = %v, want mk1", markets.NextCursor)
	}
	if len(markets.Markets) != 1 || markets.Markets[0].ID != "1" {
		t.Fatalf("markets payload = %+v", markets.Markets)
	}
	if markets.Markets[0].SportsMarketType == nil || *markets.Markets[0].SportsMarketType != "moneyline" {
		t.Fatalf("sportsMarketType = %v, want moneyline", markets.Markets[0].SportsMarketType)
	}
	if markets.Markets[0].GroupItemTitle == nil || *markets.Markets[0].GroupItemTitle != "Main" {
		t.Fatalf("groupItemTitle = %v, want Main", markets.Markets[0].GroupItemTitle)
	}

	featured := true
	includeChildren := true
	events, err := client.ListEventsKeyset(context.Background(), ListEventsKeysetOptions{
		Limit:           &limit,
		Order:           "volume",
		Ascending:       &ascending,
		AfterCursor:     "cursor-2",
		ID:              []int64{33},
		TagID:           []int64{7},
		ExcludeTagID:    []int64{8},
		Featured:        &featured,
		IncludeChildren: &includeChildren,
		TitleSearch:     "btc",
	})
	if err != nil {
		t.Fatalf("ListEventsKeyset: %v", err)
	}
	if events.NextCursor == nil || *events.NextCursor != "ev1" {
		t.Fatalf("events next_cursor = %v, want ev1", events.NextCursor)
	}
	if len(events.Events) != 1 || events.Events[0].ID != "2" {
		t.Fatalf("events payload = %+v", events.Events)
	}
	event := events.Events[0]
	if event.Live == nil || !*event.Live || event.Ended == nil || *event.Ended {
		t.Fatalf("event live/ended decode = (%v,%v), want (true,false)", event.Live, event.Ended)
	}
	if event.Score == nil || *event.Score != "3-16" || event.Period == nil || *event.Period != "Q4" || event.Elapsed == nil || *event.Elapsed != "5:18" {
		t.Fatalf("event score fields decode = (%v,%v,%v)", event.Score, event.Period, event.Elapsed)
	}
	if event.GameID == nil || *event.GameID != 19439 {
		t.Fatalf("event gameId = %v, want 19439", event.GameID)
	}
	if event.EventDate == nil || *event.EventDate != "2025-01-26" {
		t.Fatalf("event eventDate = %v, want 2025-01-26", event.EventDate)
	}
	if event.StartTime == nil || event.StartTime.UTC().Format(time.RFC3339) != "2025-01-26T20:30:00Z" {
		t.Fatalf("event startTime = %v, want 2025-01-26T20:30:00Z", event.StartTime)
	}
	if event.GameStatus == nil || *event.GameStatus != "InProgress" {
		t.Fatalf("event gameStatus = %v, want InProgress", event.GameStatus)
	}
	if len(event.Sport) == 0 {
		t.Fatal("event sport raw payload is empty")
	}
	if len(event.Teams) != 2 {
		t.Fatalf("event teams len = %d, want 2", len(event.Teams))
	}

	if len(calls) != 2 {
		t.Fatalf("calls = %d, want 2", len(calls))
	}
	if calls[0].Path != "/markets/keyset" {
		t.Fatalf("markets path = %q, want /markets/keyset", calls[0].Path)
	}
	for _, expected := range []string{
		"limit=2",
		"order=volume_num",
		"ascending=false",
		"after_cursor=cursor-1",
		"id=11",
		"id=22",
		"slug=slug-a",
		"tag_id=3",
		"tag_id=4",
		"related_tags=true",
		"closed=false",
		"include_tag=true",
	} {
		if !strings.Contains(calls[0].Query, expected) {
			t.Fatalf("markets keyset query %q missing %q", calls[0].Query, expected)
		}
	}
	if calls[1].Path != "/events/keyset" {
		t.Fatalf("events path = %q, want /events/keyset", calls[1].Path)
	}
	for _, expected := range []string{
		"limit=2",
		"order=volume",
		"ascending=false",
		"after_cursor=cursor-2",
		"id=33",
		"tag_id=7",
		"exclude_tag_id=8",
		"featured=true",
		"include_children=true",
		"title_search=btc",
	} {
		if !strings.Contains(calls[1].Query, expected) {
			t.Fatalf("events keyset query %q missing %q", calls[1].Query, expected)
		}
	}
}

func TestDecodeHTTPErrorJSON(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"type":"validation error","error":"invalid address"}`))
	}))
	defer ts.Close()

	client, err := NewGammaClient(GammaConfig{GammaBaseURL: ts.URL})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	_, err = client.ListTags(context.Background(), ListTagsOptions{})
	if err == nil {
		t.Fatal("expected error")
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error type = %T, want *APIError", err)
	}
	if apiErr.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d", apiErr.StatusCode, http.StatusUnprocessableEntity)
	}
	if apiErr.Type != "validation error" || apiErr.Message != "invalid address" {
		t.Fatalf("api err = %#v", apiErr)
	}
}

func TestDecodeHTTPErrorPlainText(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("404 page not found"))
	}))
	defer ts.Close()

	client, err := NewGammaClient(GammaConfig{GammaBaseURL: ts.URL})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	_, err = client.ListTags(context.Background(), ListTagsOptions{})
	if err == nil {
		t.Fatal("expected error")
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error type = %T, want *APIError", err)
	}
	if apiErr.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", apiErr.StatusCode, http.StatusNotFound)
	}
	if !strings.Contains(apiErr.Message, "404 page not found") {
		t.Fatalf("message = %q", apiErr.Message)
	}
}

func TestContextCancel(t *testing.T) {
	started := make(chan struct{})
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		time.Sleep(500 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]Tag{})
	}))
	defer ts.Close()

	client, err := NewGammaClient(GammaConfig{GammaBaseURL: ts.URL, Timeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-started
		cancel()
	}()

	_, err = client.ListTags(ctx, ListTagsOptions{})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "failed to send polymarket gamma request") {
		t.Fatalf("error = %v", err)
	}
}

func TestGammaCommunityEndpointsQueryAndPath(t *testing.T) {
	type call struct {
		Path  string
		Query string
	}
	calls := make([]call, 0, 6)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, call{Path: r.URL.Path, Query: r.URL.RawQuery})
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/comments":
			_, _ = w.Write([]byte(`[{"id":"c1"}]`))
		case "/comments/12":
			_, _ = w.Write([]byte(`[{"id":"c1"}]`))
		case "/comments/user_address/0xabc":
			_, _ = w.Write([]byte(`[{"id":"c1"}]`))
		case "/public-profile":
			_, _ = w.Write([]byte(`{"name":"demo"}`))
		case "/series":
			_, _ = w.Write([]byte(`[{"id":"1"}]`))
		case "/series/1":
			_, _ = w.Write([]byte(`{"id":"1"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()

	client, err := NewGammaClient(GammaConfig{GammaBaseURL: ts.URL, Timeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("NewGammaClient: %v", err)
	}

	limit := 1
	parentID := int64(10)
	getPositions := true
	_, _ = client.ListComments(context.Background(), ListCommentsOptions{
		ListOptions:      ListOptions{Limit: &limit},
		ParentEntityType: "Event",
		ParentEntityID:   &parentID,
		GetPositions:     &getPositions,
	})
	_, _ = client.GetCommentByID(context.Background(), "12", GetCommentOptions{GetPositions: &getPositions})
	_, _ = client.GetCommentsByUserAddress(context.Background(), "0xabc", ListOptions{Limit: &limit})
	_, _ = client.GetPublicProfile(context.Background(), "0xabc")
	_, _ = client.ListSeries(context.Background(), ListSeriesOptions{
		ListOptions:   ListOptions{Limit: &limit},
		Slug:          []string{"slug1"},
		CategoriesIDs: []int64{1, 2},
		ExcludeEvents: &getPositions,
	})
	_, _ = client.GetSeriesByID(context.Background(), 1, GetSeriesOptions{IncludeChat: &getPositions})

	if len(calls) != 6 {
		t.Fatalf("calls = %d, want 6", len(calls))
	}
	for _, expected := range []string{"limit=1", "parent_entity_type=Event", "parent_entity_id=10", "get_positions=true"} {
		if !strings.Contains(calls[0].Query, expected) {
			t.Fatalf("comments query %q missing %q", calls[0].Query, expected)
		}
	}
	if calls[1].Path != "/comments/12" || !strings.Contains(calls[1].Query, "get_positions=true") {
		t.Fatalf("GetCommentByID call = %+v", calls[1])
	}
	if calls[2].Path != "/comments/user_address/0xabc" || !strings.Contains(calls[2].Query, "limit=1") {
		t.Fatalf("GetCommentsByUserAddress call = %+v", calls[2])
	}
	if calls[3].Path != "/public-profile" || !strings.Contains(calls[3].Query, "address=0xabc") {
		t.Fatalf("GetPublicProfile call = %+v", calls[3])
	}
	for _, expected := range []string{"limit=1", "slug=slug1", "categories_ids=1", "categories_ids=2", "exclude_events=true"} {
		if !strings.Contains(calls[4].Query, expected) {
			t.Fatalf("ListSeries query %q missing %q", calls[4].Query, expected)
		}
	}
	if calls[5].Path != "/series/1" || !strings.Contains(calls[5].Query, "include_chat=true") {
		t.Fatalf("GetSeriesByID call = %+v", calls[5])
	}
}
