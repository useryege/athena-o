package worm

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestConfigWithDefaults(t *testing.T) {
	config := Config{}.WithDefaults()
	if config.BaseURL != DefaultBaseURL {
		t.Fatalf("BaseURL = %q, want %q", config.BaseURL, DefaultBaseURL)
	}
	if config.Timeout != DefaultTimeout {
		t.Fatalf("Timeout = %s, want %s", config.Timeout, DefaultTimeout)
	}
	if config.Now == nil {
		t.Fatal("Now is nil")
	}
}

func TestListMarketsPublicRequest(t *testing.T) {
	var gotPath string
	var gotRawQuery string
	var gotAPIKey string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotRawQuery = r.URL.RawQuery
		gotAPIKey = r.Header.Get("WORM_API_KEY")
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s, want GET", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"data": [
				{
					"condition_id": "market-1",
					"title": "Example market",
					"description": null,
					"state": "open",
					"category": "sports",
					"last_trade_price": "0.68",
					"margin_enabled": true
				}
			],
			"meta": {"limit": 20, "next_cursor": "next"},
			"error": null
		}`))
	}))
	defer server.Close()

	client, err := NewClient(Config{BaseURL: server.URL})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	resp, err := client.ListMarkets(context.Background(), ListMarketsOptions{
		PageOptions: PageOptions{Limit: 20, Cursor: "cursor"},
		State:       "OPEN",
		Category:    "sports",
		Sort:        "trending",
	})
	if err != nil {
		t.Fatalf("ListMarkets: %v", err)
	}
	if gotPath != "/markets/" {
		t.Fatalf("path = %q, want /markets/", gotPath)
	}
	for _, part := range []string{"category=sports", "cursor=cursor", "limit=20", "sort=trending", "state=OPEN"} {
		if !strings.Contains(gotRawQuery, part) {
			t.Fatalf("query = %q, want contains %q", gotRawQuery, part)
		}
	}
	if gotAPIKey != "" {
		t.Fatalf("WORM_API_KEY = %q, want empty for public endpoint", gotAPIKey)
	}
	if len(resp.Markets) != 1 || resp.Markets[0].ConditionID != "market-1" {
		t.Fatalf("markets = %#v", resp.Markets)
	}
	if resp.Meta.NextCursor == nil || *resp.Meta.NextCursor != "next" {
		t.Fatalf("next cursor = %#v, want next", resp.Meta.NextCursor)
	}
}

func TestAuthenticatedRequestSignsExactPayload(t *testing.T) {
	var gotPath string
	var gotBody string
	var gotAPIKey string
	var gotTimestamp string
	var gotSignature string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.RequestURI()
		gotAPIKey = r.Header.Get("WORM_API_KEY")
		gotTimestamp = r.Header.Get("WORM_TIMESTAMP")
		gotSignature = r.Header.Get("WORM_SIGNATURE")
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read request: %v", err)
		}
		gotBody = string(body)
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"data":{"pubkey":"order-1","message":"sign me"},"meta":{},"error":null}`))
	}))
	defer server.Close()

	client, err := NewClient(Config{
		BaseURL:   server.URL,
		APIKey:    "wk_test",
		APISecret: "secret",
		Now:       func() time.Time { return time.Unix(1714300000, 0) },
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	resp, err := client.CreateOrderDraft(context.Background(), CreateOrderDraftRequest{
		MarketConditionID: "market-1",
		IsYes:             false,
		Side:              "BUY",
		OrderType:         "MARKET",
		Funds:             "10.00",
	})
	if err != nil {
		t.Fatalf("CreateOrderDraft: %v", err)
	}
	if resp.Pubkey == nil || *resp.Pubkey != "order-1" {
		t.Fatalf("pubkey = %#v, want order-1", resp.Pubkey)
	}
	if gotPath != "/orders/" {
		t.Fatalf("path = %q, want /orders/", gotPath)
	}
	if gotAPIKey != "wk_test" {
		t.Fatalf("api key = %q, want wk_test", gotAPIKey)
	}
	if gotTimestamp != "1714300000" {
		t.Fatalf("timestamp = %q, want fixed unix seconds", gotTimestamp)
	}
	wantPayload := "1714300000POST/orders/" + gotBody
	if gotSignature != testSignature("secret", wantPayload) {
		t.Fatalf("signature = %q, want payload %q signature", gotSignature, wantPayload)
	}
}

func TestAuthenticatedRequestRequiresCredentials(t *testing.T) {
	client, err := NewClient(Config{BaseURL: "http://example.com"})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	_, err = client.GetAccountSummary(context.Background())
	if err == nil || !strings.Contains(err.Error(), "api key and secret are required") {
		t.Fatalf("error = %v, want credentials error", err)
	}
}

func TestEnvelopeError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"data": null,
			"meta": null,
			"error": {"code": -11, "slug": "invalid_request_params", "message": "Validation Error", "details": [{"limit": "too high"}]}
		}`))
	}))
	defer server.Close()

	client, err := NewClient(Config{BaseURL: server.URL})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	_, err = client.Search(context.Background(), SearchOptions{MarketTitle: "btc"})
	apiErr, ok := err.(*Error)
	if !ok {
		t.Fatalf("error = %T %v, want *Error", err, err)
	}
	if apiErr.Code != -11 || apiErr.Slug != "invalid_request_params" {
		t.Fatalf("api error = %#v", apiErr)
	}
}

func TestHTTPErrorWithEnvelope(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{
			"data": null,
			"meta": null,
			"error": {"code": -12, "slug": "authentication_failed", "message": "Invalid signature", "details": []}
		}`))
	}))
	defer server.Close()

	client, err := NewClient(Config{
		BaseURL:   server.URL,
		APIKey:    "wk_test",
		APISecret: "secret",
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	_, err = client.GetAccountSummary(context.Background())
	apiErr, ok := err.(*Error)
	if !ok {
		t.Fatalf("error = %T %v, want *Error", err, err)
	}
	if apiErr.Status != "401 Unauthorized" || apiErr.Slug != "authentication_failed" {
		t.Fatalf("api error = %#v", apiErr)
	}
}

func TestInvalidJSONResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{`))
	}))
	defer server.Close()

	client, err := NewClient(Config{BaseURL: server.URL})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	_, err = client.CreateAuthChallenge(context.Background(), CreateAuthChallengeRequest{WalletAddress: "wallet"})
	if err == nil || !strings.Contains(err.Error(), "decode worm response") {
		t.Fatalf("error = %v, want decode error", err)
	}
}

func TestContextCanceled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer server.Close()

	client, err := NewClient(Config{BaseURL: server.URL, Timeout: time.Second})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = client.Search(ctx, SearchOptions{MarketTitle: "btc"})
	if err == nil || !strings.Contains(err.Error(), "context canceled") {
		t.Fatalf("error = %v, want context canceled", err)
	}
}

func TestRepresentativeEndpoints(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/search/":
			_, _ = w.Write([]byte(`{"data":[{"result_type":"market","summary":{"condition_id":"market-1","title":"BTC","state":"open","category":"crypto"}}],"meta":{"limit":1},"error":null}`))
		case "/auth/keys/challenge/":
			_, _ = w.Write([]byte(`{"data":{"nonce":"nonce","message":"sign","expires_in_seconds":600},"meta":{},"error":null}`))
		case "/account/":
			_, _ = w.Write([]byte(`{"data":{"username":"trader","twitter_username":null,"joined_at":1714300000},"meta":{},"error":null}`))
		case "/margin/positions/estimate/":
			_, _ = w.Write([]byte(`{"data":{"average_price":"0.66","total_shares":"10","total_cost":"6.6","best_ask":"0.65","worst_fill_price":"0.67","is_fully_filled":true,"fee_amount":"0.1","user_funds_needed":"10.1","liquidation_price":"0.5"},"meta":{},"error":null}`))
		case "/redeems/":
			_, _ = w.Write([]byte(`{"data":{"pubkey":"redeem-1","message":"sign redeem"},"meta":{},"error":null}`))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client, err := NewClient(Config{
		BaseURL:   server.URL,
		APIKey:    "wk_test",
		APISecret: "secret",
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if resp, err := client.Search(context.Background(), SearchOptions{MarketTitle: "BTC", PageOptions: PageOptions{Limit: 1}}); err != nil || len(resp.Results) != 1 {
		t.Fatalf("Search = %#v, %v", resp, err)
	}
	if resp, err := client.CreateAuthChallenge(context.Background(), CreateAuthChallengeRequest{WalletAddress: "wallet"}); err != nil || resp.Nonce != "nonce" {
		t.Fatalf("CreateAuthChallenge = %#v, %v", resp, err)
	}
	if resp, err := client.GetAccountSummary(context.Background()); err != nil || resp.Username != "trader" {
		t.Fatalf("GetAccountSummary = %#v, %v", resp, err)
	}
	if resp, err := client.EstimateMarginPosition(context.Background(), EstimateMarginPositionOptions{MarketConditionID: "market-1", Funds: "10"}); err != nil || resp.AveragePrice != "0.66" {
		t.Fatalf("EstimateMarginPosition = %#v, %v", resp, err)
	}
	if resp, err := client.StartRedeem(context.Background(), StartRedeemRequest{MarketConditionID: "market-1"}); err != nil || resp.Pubkey == nil || *resp.Pubkey != "redeem-1" {
		t.Fatalf("StartRedeem = %#v, %v", resp, err)
	}
}

func testSignature(secret string, payload string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(payload))
	return hex.EncodeToString(mac.Sum(nil))
}
