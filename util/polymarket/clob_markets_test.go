package polymarket

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestCLOBMarketsPathAndQuery(t *testing.T) {
	type call struct {
		Path  string
		Query string
		Body  string
	}

	calls := make([]call, 0, 7)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		calls = append(calls, call{Path: r.URL.Path, Query: r.URL.RawQuery, Body: string(body)})
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/markets-by-token/t%2F1":
			_, _ = w.Write([]byte(`{"condition_id":"c1","primary_token_id":"p1","secondary_token_id":"s1"}`))
		case "/clob-markets/c%2F1":
			_, _ = w.Write([]byte(`{"t":[{"t":"p1","o":"Yes"}],"mos":5,"mts":0.01}`))
		case "/prices-history":
			_, _ = w.Write([]byte(`{"history":[{"t":1,"p":0.5}]}`))
		case "/batch-prices-history":
			_, _ = w.Write([]byte(`{"history":{"m1":[{"t":1,"p":0.5}]}}`))
		case "/simplified-markets":
			_, _ = w.Write([]byte(`{"limit":2,"next_cursor":"abc","count":1,"data":[{"condition_id":"c1"}]}`))
		case "/sampling-markets":
			_, _ = w.Write([]byte(`{"limit":2,"next_cursor":"def","count":1,"data":[{"condition_id":"c1"}]}`))
		case "/sampling-simplified-markets":
			_, _ = w.Write([]byte(`{"limit":2,"next_cursor":"ghi","count":1,"data":[{"condition_id":"c1"}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()

	client, err := NewCLOBClient(CLOBConfig{CLOBBaseURL: ts.URL, Timeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("NewCLOBClient: %v", err)
	}

	start := 1.0
	end := 2.0
	fidelity := 5
	_, _ = client.GetMarketByToken(context.Background(), "t/1")
	_, _ = client.GetCLOBMarketInfo(context.Background(), "c/1")
	_, _ = client.GetPricesHistory(context.Background(), GetCLOBPricesHistoryOptions{
		Market: "m1", StartTs: &start, EndTs: &end, Interval: "1h", Fidelity: &fidelity,
	})
	_, _ = client.GetBatchPricesHistory(context.Background(), CLOBBatchPricesHistoryRequest{
		Markets: []string{"m1"}, Interval: "1h", Fidelity: &fidelity,
	})
	_, _ = client.ListSimplifiedMarkets(context.Background(), "abc")
	_, _ = client.ListSamplingMarkets(context.Background(), "def")
	_, _ = client.ListSamplingSimplifiedMarkets(context.Background(), "ghi")

	if len(calls) != 7 {
		t.Fatalf("calls = %d, want 7", len(calls))
	}

	if calls[0].Path != "/markets-by-token/t/1" {
		t.Fatalf("GetMarketByToken path = %q", calls[0].Path)
	}
	if calls[1].Path != "/clob-markets/c/1" {
		t.Fatalf("GetCLOBMarketInfo path = %q", calls[1].Path)
	}
	if calls[2].Path != "/prices-history" {
		t.Fatalf("GetPricesHistory path = %q", calls[2].Path)
	}
	for _, want := range []string{"market=m1", "startTs=1", "endTs=2", "interval=1h", "fidelity=5"} {
		if !strings.Contains(calls[2].Query, want) {
			t.Fatalf("GetPricesHistory query %q missing %q", calls[2].Query, want)
		}
	}
	if calls[3].Path != "/batch-prices-history" {
		t.Fatalf("GetBatchPricesHistory path = %q", calls[3].Path)
	}
	if !strings.Contains(calls[3].Body, `"markets":["m1"]`) {
		t.Fatalf("GetBatchPricesHistory body = %q", calls[3].Body)
	}
	if calls[4].Query != "next_cursor=abc" {
		t.Fatalf("ListSimplifiedMarkets query = %q", calls[4].Query)
	}
	if calls[5].Query != "next_cursor=def" {
		t.Fatalf("ListSamplingMarkets query = %q", calls[5].Query)
	}
	if calls[6].Query != "next_cursor=ghi" {
		t.Fatalf("ListSamplingSimplifiedMarkets query = %q", calls[6].Query)
	}
}

func TestCLOBMarketsDecode(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/markets-by-token/1":
			_, _ = w.Write([]byte(`{"condition_id":"cond","primary_token_id":"p","secondary_token_id":"s"}`))
		case "/clob-markets/cond":
			_, _ = w.Write([]byte(`{"gst":"2026-01-01T00:00:00Z","r":{"foo":"bar"},"t":[{"t":"p","o":"Yes"}],"mbf":1}`))
		case "/prices-history":
			_, _ = w.Write([]byte(`{"history":[{"t":10,"p":0.75}]}`))
		case "/batch-prices-history":
			_, _ = w.Write([]byte(`{"history":{"m":[{"t":11,"p":0.55}]}}`))
		case "/simplified-markets":
			_, _ = w.Write([]byte(`{"limit":1,"next_cursor":"n","count":1,"data":[{"condition_id":"c"}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()

	client, err := NewCLOBClient(CLOBConfig{CLOBBaseURL: ts.URL, Timeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("NewCLOBClient: %v", err)
	}

	marketByToken, err := client.GetMarketByToken(context.Background(), "1")
	if err != nil {
		t.Fatalf("GetMarketByToken: %v", err)
	}
	if marketByToken.ConditionID != "cond" {
		t.Fatalf("condition_id = %q", marketByToken.ConditionID)
	}

	marketInfo, err := client.GetCLOBMarketInfo(context.Background(), "cond")
	if err != nil {
		t.Fatalf("GetCLOBMarketInfo: %v", err)
	}
	if marketInfo == nil || len(marketInfo.Tokens) != 1 {
		t.Fatalf("tokens = %#v", marketInfo)
	}

	history, err := client.GetPricesHistory(context.Background(), GetCLOBPricesHistoryOptions{Market: "m"})
	if err != nil {
		t.Fatalf("GetPricesHistory: %v", err)
	}
	if len(history.History) != 1 || history.History[0].P != 0.75 {
		t.Fatalf("history = %#v", history.History)
	}

	batch, err := client.GetBatchPricesHistory(context.Background(), CLOBBatchPricesHistoryRequest{Markets: []string{"m"}})
	if err != nil {
		t.Fatalf("GetBatchPricesHistory: %v", err)
	}
	if len(batch.History["m"]) != 1 {
		t.Fatalf("batch history = %#v", batch.History)
	}

	page, err := client.ListSimplifiedMarkets(context.Background(), "")
	if err != nil {
		t.Fatalf("ListSimplifiedMarkets: %v", err)
	}
	if page.NextCursor != "n" || len(page.Items) != 1 {
		t.Fatalf("page = %#v", page)
	}
}
