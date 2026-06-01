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
