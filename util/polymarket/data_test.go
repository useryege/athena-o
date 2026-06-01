package polymarket

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewDataClientDefaults(t *testing.T) {
	c, err := NewDataClient(DataConfig{})
	if err != nil {
		t.Fatalf("NewDataClient: %v", err)
	}
	impl, ok := c.(*dataClientImpl)
	if !ok {
		t.Fatalf("client type = %T, want *dataClientImpl", c)
	}
	if impl.config.DataBaseURL != DefaultDataBaseURL {
		t.Fatalf("DataBaseURL = %q, want %q", impl.config.DataBaseURL, DefaultDataBaseURL)
	}
	if impl.config.Timeout != DefaultTimeout {
		t.Fatalf("Timeout = %s, want %s", impl.config.Timeout, DefaultTimeout)
	}
}

func TestNewDataClientInvalidBaseURL(t *testing.T) {
	_, err := NewDataClient(DataConfig{DataBaseURL: "://bad"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestDataQueryEncoding(t *testing.T) {
	var gotPath, gotQuery string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[]`))
	}))
	defer ts.Close()

	client, err := NewDataClient(DataConfig{DataBaseURL: ts.URL, Timeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("NewDataClient: %v", err)
	}

	limit := 2
	offset := 3
	start := int64(10)
	_, err = client.ListUserActivity(context.Background(), ListUserActivityOptions{
		DataListOptions: DataListOptions{Limit: &limit, Offset: &offset},
		User:            "0xabc",
		Market:          []string{"m1", "m2"},
		EventID:         []int64{1, 2},
		Type:            []string{"TRADE", "REDEEM"},
		Start:           &start,
		SortBy:          "TIMESTAMP",
		SortDirection:   "DESC",
	})
	if err != nil {
		t.Fatalf("ListUserActivity: %v", err)
	}
	if gotPath != "/activity" {
		t.Fatalf("path = %q, want /activity", gotPath)
	}
	for _, expected := range []string{"limit=2", "offset=3", "user=0xabc", "market=m1%2Cm2", "eventId=1%2C2", "type=TRADE%2CREDEEM", "start=10", "sortBy=TIMESTAMP", "sortDirection=DESC"} {
		if !strings.Contains(gotQuery, expected) {
			t.Fatalf("query %q missing %q", gotQuery, expected)
		}
	}
}
