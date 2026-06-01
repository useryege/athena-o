package polymarket

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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

func TestDataTypedResponseDecoding(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/value":
			_, _ = w.Write([]byte(`[{"user":"0xabc","value":1.25}]`))
		case "/oi":
			_, _ = w.Write([]byte(`[{"market":"0xmarket","value":42.5}]`))
		case "/live-volume":
			_, _ = w.Write([]byte(`[{"total":99.9,"markets":[{"market":"0xm1","value":12.3}]}]`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()

	client, err := NewDataClient(DataConfig{DataBaseURL: ts.URL, Timeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("NewDataClient: %v", err)
	}

	gotValue, err := client.GetTotalValueForUser(context.Background(), "0xabc", GetTotalValueOptions{})
	if err != nil {
		t.Fatalf("GetTotalValueForUser: %v", err)
	}
	if len(gotValue) != 1 || gotValue[0].User != "0xabc" || gotValue[0].Value != 1.25 {
		t.Fatalf("GetTotalValueForUser decoded = %#v", gotValue)
	}

	gotOI, err := client.GetOpenInterest(context.Background(), GetOpenInterestOptions{})
	if err != nil {
		t.Fatalf("GetOpenInterest: %v", err)
	}
	if len(gotOI) != 1 || gotOI[0].Market != "0xmarket" || gotOI[0].Value != 42.5 {
		t.Fatalf("GetOpenInterest decoded = %#v", gotOI)
	}

	gotLive, err := client.GetLiveVolumeByEventID(context.Background(), 1)
	if err != nil {
		t.Fatalf("GetLiveVolumeByEventID: %v", err)
	}
	if len(gotLive) != 1 || gotLive[0].Total != 99.9 || len(gotLive[0].Markets) != 1 || gotLive[0].Markets[0].Market != "0xm1" {
		t.Fatalf("GetLiveVolumeByEventID decoded = %#v", gotLive)
	}
}

func TestDataTypedResponseEmptyArrays(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[]`))
	}))
	defer ts.Close()

	client, err := NewDataClient(DataConfig{DataBaseURL: ts.URL, Timeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("NewDataClient: %v", err)
	}

	gotValue, err := client.GetTotalValueForUser(context.Background(), "0xabc", GetTotalValueOptions{})
	if err != nil {
		t.Fatalf("GetTotalValueForUser: %v", err)
	}
	if len(gotValue) != 0 {
		t.Fatalf("GetTotalValueForUser len = %d, want 0", len(gotValue))
	}

	gotOI, err := client.GetOpenInterest(context.Background(), GetOpenInterestOptions{})
	if err != nil {
		t.Fatalf("GetOpenInterest: %v", err)
	}
	if len(gotOI) != 0 {
		t.Fatalf("GetOpenInterest len = %d, want 0", len(gotOI))
	}

	gotLive, err := client.GetLiveVolumeByEventID(context.Background(), 1)
	if err != nil {
		t.Fatalf("GetLiveVolumeByEventID: %v", err)
	}
	if len(gotLive) != 0 {
		t.Fatalf("GetLiveVolumeByEventID len = %d, want 0", len(gotLive))
	}
}

func TestDataTypedResponseLiveVolumeEmptyMarkets(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"total":0,"markets":[]}]`))
	}))
	defer ts.Close()

	client, err := NewDataClient(DataConfig{DataBaseURL: ts.URL, Timeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("NewDataClient: %v", err)
	}

	got, err := client.GetLiveVolumeByEventID(context.Background(), 1)
	if err != nil {
		t.Fatalf("GetLiveVolumeByEventID: %v", err)
	}
	if len(got) != 1 || got[0].Total != 0 || len(got[0].Markets) != 0 {
		t.Fatalf("GetLiveVolumeByEventID decoded = %#v", got)
	}
}

func TestDataSnapshotContracts(t *testing.T) {
	t.Run("GetTotalValueForUser", func(t *testing.T) {
		body := readSnapshotBody(t, "get_data_api.polymarket.com_value")
		var out []DataUserValue
		if err := json.Unmarshal(body, &out); err != nil {
			t.Fatalf("unmarshal snapshot body to []DataUserValue: %v", err)
		}
	})

	t.Run("GetOpenInterest", func(t *testing.T) {
		body := readSnapshotBody(t, "get_data_api.polymarket.com_oi")
		var out []DataOpenInterest
		if err := json.Unmarshal(body, &out); err != nil {
			t.Fatalf("unmarshal snapshot body to []DataOpenInterest: %v", err)
		}
	})

	t.Run("GetLiveVolumeByEventID", func(t *testing.T) {
		body := readSnapshotBody(t, "get_data_api.polymarket.com_live_volume")
		var out []DataLiveVolume
		if err := json.Unmarshal(body, &out); err != nil {
			t.Fatalf("unmarshal snapshot body to []DataLiveVolume: %v", err)
		}
	})
}

func readSnapshotBody(t *testing.T, endpointDir string) json.RawMessage {
	t.Helper()

	path := findSnapshotResponsePath(t, endpointDir)
	payload, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read snapshot response file: %v", err)
	}

	var envelope struct {
		Body json.RawMessage `json:"body"`
	}
	if err := json.Unmarshal(payload, &envelope); err != nil {
		t.Fatalf("unmarshal snapshot response envelope: %v", err)
	}
	if len(envelope.Body) == 0 || string(envelope.Body) == "null" {
		t.Fatalf("snapshot body is empty/null for %s", endpointDir)
	}
	return envelope.Body
}

func findSnapshotResponsePath(t *testing.T, endpointDir string) string {
	t.Helper()
	candidates := []string{
		filepath.Join("request-response", "latest", endpointDir, "response.json"),
		filepath.Join("util", "polymarket", "request-response", "latest", endpointDir, "response.json"),
	}
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	t.Fatalf("snapshot response.json not found for %s, checked: %v", endpointDir, candidates)
	return ""
}
