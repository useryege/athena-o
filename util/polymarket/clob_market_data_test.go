package polymarket

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewCLOBClientDefaults(t *testing.T) {
	c, err := NewCLOBClient(CLOBConfig{})
	if err != nil {
		t.Fatalf("NewCLOBClient: %v", err)
	}
	impl, ok := c.(*clobClientImpl)
	if !ok {
		t.Fatalf("client type = %T, want *clobClientImpl", c)
	}
	if impl.config.CLOBBaseURL != DefaultCLOBBaseURL {
		t.Fatalf("CLOBBaseURL = %q, want %q", impl.config.CLOBBaseURL, DefaultCLOBBaseURL)
	}
	if impl.config.Timeout != DefaultTimeout {
		t.Fatalf("Timeout = %s, want %s", impl.config.Timeout, DefaultTimeout)
	}
}

func TestNewCLOBClientInvalidBaseURL(t *testing.T) {
	_, err := NewCLOBClient(CLOBConfig{CLOBBaseURL: "://bad"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCLOBPostBodyEncoding(t *testing.T) {
	var gotPath string
	var gotBody string

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		body, _ := io.ReadAll(r.Body)
		gotBody = string(body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[]`))
	}))
	defer ts.Close()

	client, err := NewCLOBClient(CLOBConfig{CLOBBaseURL: ts.URL, Timeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("NewCLOBClient: %v", err)
	}

	_, err = client.GetOrderBooks(context.Background(), []CLOBBookRequest{{TokenID: "0x1"}})
	if err != nil {
		t.Fatalf("GetOrderBooks: %v", err)
	}
	if gotPath != "/books" {
		t.Fatalf("path = %q, want /books", gotPath)
	}
	if !strings.Contains(gotBody, `"token_id":"0x1"`) {
		t.Fatalf("body = %q, expected token_id", gotBody)
	}
}

func TestCLOBBulkMethodsUsePOSTBody(t *testing.T) {
	type call struct {
		Method string              `json:"method"`
		Path   string              `json:"path"`
		Body   []map[string]string `json:"body"`
		Query  map[string][]string `json:"query"`
		Header map[string][]string `json:"header"`
	}

	calls := make([]call, 0, 3)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		reqBody := make([]map[string]string, 0)
		_ = json.Unmarshal(body, &reqBody)
		calls = append(calls, call{
			Method: r.Method,
			Path:   r.URL.Path,
			Body:   reqBody,
			Query:  map[string][]string(r.URL.Query()),
			Header: map[string][]string(r.Header),
		})
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/midpoints":
			_, _ = w.Write([]byte(`{"0x1":"0.5"}`))
		case "/prices":
			_, _ = w.Write([]byte(`{"0x1":{"BUY":"0.5"}}`))
		case "/last-trades-prices":
			_, _ = w.Write([]byte(`[{"token_id":"0x1","price":"0.5","side":"BUY"}]`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()

	client, err := NewCLOBClient(CLOBConfig{CLOBBaseURL: ts.URL, Timeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("NewCLOBClient: %v", err)
	}

	if _, err := client.GetMidpointPrices(context.Background(), []string{"0x1"}); err != nil {
		t.Fatalf("GetMidpointPrices: %v", err)
	}
	if _, err := client.GetMarketPrices(context.Background(), []string{"0x1"}, []string{"BUY"}); err != nil {
		t.Fatalf("GetMarketPrices: %v", err)
	}
	if _, err := client.GetLastTradePrices(context.Background(), []string{"0x1"}); err != nil {
		t.Fatalf("GetLastTradePrices: %v", err)
	}

	if len(calls) != 3 {
		t.Fatalf("calls = %d, want 3", len(calls))
	}
	for i, c := range calls {
		if c.Method != http.MethodPost {
			t.Fatalf("call %d method = %s, want POST", i, c.Method)
		}
		if len(c.Query) != 0 {
			t.Fatalf("call %d query = %#v, want empty", i, c.Query)
		}
	}
	if calls[0].Path != "/midpoints" || calls[1].Path != "/prices" || calls[2].Path != "/last-trades-prices" {
		t.Fatalf("paths = [%s, %s, %s]", calls[0].Path, calls[1].Path, calls[2].Path)
	}
}

func TestGetMarketPricesLengthMismatch(t *testing.T) {
	client, err := NewCLOBClient(CLOBConfig{CLOBBaseURL: "http://example.com"})
	if err != nil {
		t.Fatalf("NewCLOBClient: %v", err)
	}
	_, err = client.GetMarketPrices(context.Background(), []string{"0x1", "0x2"}, []string{"BUY"})
	if err == nil {
		t.Fatal("expected length mismatch error")
	}
	if !strings.Contains(err.Error(), "length mismatch") {
		t.Fatalf("error = %v", err)
	}
}

func TestGetMarketPriceStringDecode(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"price":"0.53"}`))
	}))
	defer ts.Close()

	client, err := NewCLOBClient(CLOBConfig{CLOBBaseURL: ts.URL, Timeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("NewCLOBClient: %v", err)
	}

	got, err := client.GetMarketPrice(context.Background(), "0x1", "BUY")
	if err != nil {
		t.Fatalf("GetMarketPrice: %v", err)
	}
	if got.Price != "0.53" {
		t.Fatalf("price = %q, want %q", got.Price, "0.53")
	}
}

func TestGetMidpointPriceQueryAndDecode(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{name: "mid field", body: `{"mid":"0.45"}`, want: "0.45"},
		{name: "mid_price field", body: `{"mid_price":"0.46"}`, want: "0.46"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotMethod string
			var gotPath string
			var gotTokenID string
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotMethod = r.Method
				gotPath = r.URL.Path
				gotTokenID = r.URL.Query().Get("token_id")
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(tt.body))
			}))
			defer ts.Close()

			client, err := NewCLOBClient(CLOBConfig{CLOBBaseURL: ts.URL, Timeout: 5 * time.Second})
			if err != nil {
				t.Fatalf("NewCLOBClient: %v", err)
			}

			got, err := client.GetMidpointPrice(context.Background(), "0xabc")
			if err != nil {
				t.Fatalf("GetMidpointPrice: %v", err)
			}
			if gotMethod != http.MethodGet {
				t.Fatalf("method = %s, want GET", gotMethod)
			}
			if gotPath != "/midpoint" {
				t.Fatalf("path = %q, want /midpoint", gotPath)
			}
			if gotTokenID != "0xabc" {
				t.Fatalf("token_id = %q, want 0xabc", gotTokenID)
			}
			if got.MidPrice != tt.want {
				t.Fatalf("mid price = %q, want %q", got.MidPrice, tt.want)
			}
		})
	}
}

func TestGetServerTimeNumericDecode(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/time" {
			t.Fatalf("path = %s, want /time", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`1234567890`))
	}))
	defer ts.Close()

	client, err := NewCLOBClient(CLOBConfig{CLOBBaseURL: ts.URL, Timeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("NewCLOBClient: %v", err)
	}

	got, err := client.GetServerTime(context.Background())
	if err != nil {
		t.Fatalf("GetServerTime: %v", err)
	}
	if got.Unix != 1234567890 {
		t.Fatalf("unix = %d, want 1234567890", got.Unix)
	}
}
