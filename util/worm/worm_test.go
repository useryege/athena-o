package worm

import (
	"context"
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/mr-tron/base58/base58"
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
		gotAPIKey = r.Header.Get(headerAPIKey)
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
		t.Fatalf("%s = %q, want empty for public endpoint", headerAPIKey, gotAPIKey)
	}
	if len(resp.Markets) != 1 || resp.Markets[0].ConditionID != "market-1" {
		t.Fatalf("markets = %#v", resp.Markets)
	}
	if resp.Meta.NextCursor == nil || *resp.Meta.NextCursor != "next" {
		t.Fatalf("next cursor = %#v, want next", resp.Meta.NextCursor)
	}
}

func TestGetMarketDecodesRulesList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/markets/market-1/" {
			t.Fatalf("path = %q, want /markets/market-1/", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"data": {
				"condition_id": "market-1",
				"title": "Example market",
				"state": "open",
				"category": "crypto",
				"rules": [
					"The market resolves YES if the source resolves YES.",
					"The market resolves NO if the source resolves NO."
				]
			},
			"meta": {},
			"error": null
		}`))
	}))
	defer server.Close()

	client, err := NewClient(Config{BaseURL: server.URL})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	resp, err := client.GetMarket(context.Background(), "market-1")
	if err != nil {
		t.Fatalf("GetMarket: %v", err)
	}
	if len(resp.Rules) != 2 || resp.Rules[0] == "" {
		t.Fatalf("rules = %#v, want decoded rules list", resp.Rules)
	}
	if resp.RawRules == nil || len(*resp.RawRules) == 0 {
		t.Fatal("RawRules is nil, want original rules payload")
	}
}

func TestGetMarketDecodesRulesObject(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/markets/market-1/" {
			t.Fatalf("path = %q, want /markets/market-1/", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"data": {
				"condition_id": "market-1",
				"title": "Example market",
				"state": "open",
				"category": "crypto",
				"rules": {"resolution_source": "official source"},
				"config": {"kind": "orderbook"}
			},
			"meta": {},
			"error": null
		}`))
	}))
	defer server.Close()

	client, err := NewClient(Config{BaseURL: server.URL})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	resp, err := client.GetMarket(context.Background(), "market-1")
	if err != nil {
		t.Fatalf("GetMarket: %v", err)
	}
	if len(resp.Rules) != 0 {
		t.Fatalf("rules = %#v, want empty legacy list for object-shaped rules", resp.Rules)
	}
	if resp.RawRules == nil || !strings.Contains(string(*resp.RawRules), "resolution_source") {
		t.Fatalf("RawRules = %v, want object payload", resp.RawRules)
	}
	if resp.RawConfig == nil || !strings.Contains(string(*resp.RawConfig), "orderbook") {
		t.Fatalf("RawConfig = %v, want config payload", resp.RawConfig)
	}
}

func TestNullableResponseFieldsDecodeAsNil(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/markets/market-1/price/":
			_, _ = w.Write([]byte(`{"data":{"condition_id":"market-1","price":null,"price_kind":"mid","is_yes":true},"meta":{},"error":null}`))
		case "/orders/":
			_, _ = w.Write([]byte(`{"data":[{"pubkey":null,"status":null,"side":"buy","price":null,"amount":null,"remaining_amount":null,"filled_amount":null,"funds":null,"remaining_funds":null,"filled_funds":null,"market":{"condition_id":"market-1","title":"Market"},"outcome":{"is_yes":true,"text":"YES"}}],"meta":{"limit":1},"error":null}`))
		case "/trades/":
			_, _ = w.Write([]byte(`{"data":[{"pubkey":"trade-1","market_condition_id":null,"amount":"1","price":"0.5","timestamp":1714300000,"fee":null,"is_maker":false,"state":"done","order":{"pubkey":null,"status":null,"side":"buy","market":{"condition_id":"market-1","title":"Market"},"outcome":{"is_yes":true,"text":"YES"}}}],"meta":{"limit":1},"error":null}`))
		case "/account/":
			_, _ = w.Write([]byte(`{"data":{"username":"trader","twitter_username":null,"joined_at":null},"meta":{},"error":null}`))
		case "/account/assets/":
			_, _ = w.Write([]byte(`{"data":[{"asset_kind":"share","amounts":{"total":"1","locked":"0","available":"1"},"token":{"symbol":"share","address":null},"value":{"usdt":"0.5","basis":"mark_to_market"},"position":{"market":{"condition_id":"market-1","title":"Market"},"is_yes":true,"outcome_text":"YES","is_final":false,"avg_trade_price":"0.5"},"created":null}],"meta":{"limit":1},"error":null}`))
		case "/redeems/":
			_, _ = w.Write([]byte(`{"data":[{"pubkey":"redeem-1","market_condition_id":null,"state":"created","funds":"1","onchain_funds":"1","yes_shares":"0","no_shares":"0","message":null,"created":null}],"meta":{"limit":1},"error":null}`))
		case "/margin/positions/":
			_, _ = w.Write([]byte(`{"data":[{"pubkey":"position-1","position_request_pubkey":null,"market":{"condition_id":"market-1","title":"Market"},"is_yes":true,"leverage":"2","total_shares":"1","avg_entry_price":"0.5","closing_price":null,"unrealized_pnl":null,"realized_pnl":"0","user_liquidity":"1","total_liquidity":"2","liquidation_price":"0.1","is_closed":false,"is_liquidated":false,"is_claimed":false,"tp_sl":null,"created":null}],"meta":{"limit":1},"error":null}`))
		case "/margin/positions/settlements/":
			_, _ = w.Write([]byte(`{"data":[{"position_pubkey":"position-1","position":{"pubkey":"position-1","position_request_pubkey":null,"market":{"condition_id":"market-1","title":"Market"},"is_yes":true,"leverage":"2","total_shares":"1","avg_entry_price":"0.5","closing_price":null,"unrealized_pnl":null,"realized_pnl":"0","user_liquidity":"1","total_liquidity":"2","liquidation_price":"0.1","is_closed":false,"is_liquidated":false,"is_claimed":false,"tp_sl":null,"created":null},"total_pnl":"0","user_liquidity":"1","state":"created","created":null}],"meta":{"limit":1},"error":null}`))
		case "/margin/positions/requests/":
			_, _ = w.Write([]byte(`{"data":[{"pubkey":"request-1","type":"MARKET","state":"created","message":null,"funding_txid":null,"refund_txid":null,"market":null,"is_yes":true,"leverage":"2","funds":"1","price":null,"shares":null,"take_profit_price":null,"stop_loss_price":null,"created":null}],"meta":{"limit":1},"error":null}`))
		case "/markets/market-1/margin-activity/":
			_, _ = w.Write([]byte(`{"data":[{"activity_type":"opened","user":{"username":"trader"},"market":null,"shares":null,"price":null,"realized_pnl":null,"leverage":null,"is_yes":null,"created":1714300000}],"meta":{"limit":1},"error":null}`))
		case "/markets/market-1/trades/":
			_, _ = w.Write([]byte(`{"data":[{"market_condition_id":null,"amount":"1","price":"0.5","maker_fee":"0","taker_fee":"0","timestamp":1714300000,"state":"done","is_yes":true}],"meta":{"limit":1},"error":null}`))
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

	price, err := client.GetMarketPrice(context.Background(), "market-1", GetMarketPriceOptions{})
	if err != nil {
		t.Fatalf("GetMarketPrice: %v", err)
	}
	if price.Price != nil {
		t.Fatalf("price = %#v, want nil", price.Price)
	}

	orders, err := client.ListOrders(context.Background(), ListOrdersOptions{PageOptions: PageOptions{Limit: 1}})
	if err != nil {
		t.Fatalf("ListOrders: %v", err)
	}
	if orders.Orders[0].Pubkey != nil || orders.Orders[0].Status != nil {
		t.Fatalf("order = %#v, want nil pubkey/status", orders.Orders[0])
	}

	trades, err := client.ListTrades(context.Background(), ListTradesOptions{PageOptions: PageOptions{Limit: 1}})
	if err != nil {
		t.Fatalf("ListTrades: %v", err)
	}
	if trades.Trades[0].MarketConditionID != nil || trades.Trades[0].Fee != nil || trades.Trades[0].Order.Pubkey != nil {
		t.Fatalf("trade = %#v, want nil nullable fields", trades.Trades[0])
	}

	account, err := client.GetAccountSummary(context.Background())
	if err != nil {
		t.Fatalf("GetAccountSummary: %v", err)
	}
	if account.JoinedAt != nil {
		t.Fatalf("joined_at = %#v, want nil", account.JoinedAt)
	}

	assets, err := client.ListAccountAssets(context.Background(), ListAccountAssetsOptions{PageOptions: PageOptions{Limit: 1}})
	if err != nil {
		t.Fatalf("ListAccountAssets: %v", err)
	}
	if assets.Assets[0].Created != nil {
		t.Fatalf("asset created = %#v, want nil", assets.Assets[0].Created)
	}

	redeems, err := client.ListRedeems(context.Background(), ListRedeemsOptions{PageOptions: PageOptions{Limit: 1}})
	if err != nil {
		t.Fatalf("ListRedeems: %v", err)
	}
	if redeems.Redeems[0].MarketConditionID != nil || redeems.Redeems[0].Created != nil {
		t.Fatalf("redeem = %#v, want nil nullable fields", redeems.Redeems[0])
	}

	positions, err := client.ListMarginPositions(context.Background(), ListMarginPositionsOptions{PageOptions: PageOptions{Limit: 1}})
	if err != nil {
		t.Fatalf("ListMarginPositions: %v", err)
	}
	if positions.Positions[0].PositionRequestPubkey != nil || positions.Positions[0].Created != nil {
		t.Fatalf("position = %#v, want nil nullable fields", positions.Positions[0])
	}

	settlements, err := client.ListMarginSettlements(context.Background(), ListMarginSettlementsOptions{PageOptions: PageOptions{Limit: 1}})
	if err != nil {
		t.Fatalf("ListMarginSettlements: %v", err)
	}
	if settlements.Settlements[0].Created != nil || settlements.Settlements[0].Position.Created != nil {
		t.Fatalf("settlement = %#v, want nil nullable fields", settlements.Settlements[0])
	}

	requests, err := client.ListPositionRequests(context.Background(), ListPositionRequestsOptions{PageOptions: PageOptions{Limit: 1}})
	if err != nil {
		t.Fatalf("ListPositionRequests: %v", err)
	}
	if requests.Requests[0].Market != nil || requests.Requests[0].Created != nil {
		t.Fatalf("position request = %#v, want nil nullable fields", requests.Requests[0])
	}

	activities, err := client.ListMarketMarginActivity(context.Background(), "market-1", ListMarketMarginActivityOptions{PageOptions: PageOptions{Limit: 1}})
	if err != nil {
		t.Fatalf("ListMarketMarginActivity: %v", err)
	}
	if activities.Activities[0].Market != nil || activities.Activities[0].Shares != nil || activities.Activities[0].Price != nil || activities.Activities[0].Leverage != nil {
		t.Fatalf("activity = %#v, want nil nullable fields", activities.Activities[0])
	}

	marketTrades, err := client.ListMarketTrades(context.Background(), "market-1", ListMarketTradesOptions{PageOptions: PageOptions{Limit: 1}})
	if err != nil {
		t.Fatalf("ListMarketTrades: %v", err)
	}
	if marketTrades.Trades[0].MarketConditionID != nil {
		t.Fatalf("market trade = %#v, want nil market_condition_id", marketTrades.Trades[0])
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
		gotAPIKey = r.Header.Get(headerAPIKey)
		gotTimestamp = r.Header.Get(headerTimestamp)
		gotSignature = r.Header.Get(headerSignature)
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

func TestOrderReadEndpoints(t *testing.T) {
	const nextCursor = "next-orders"
	requests := make([]string, 0, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.URL.RequestURI())
		assertAuthenticatedRequest(t, r)
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s, want GET", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.EscapedPath() {
		case "/orders/":
			assertQueryValue(t, r, "limit", "2")
			assertQueryValue(t, r, "cursor", "cursor")
			assertQueryValue(t, r, "market_condition_id", "market-1")
			assertQueryValue(t, r, "status", "OPEN,FILLED")
			assertQueryValue(t, r, "is_yes", "false")
			assertQueryValue(t, r, "side", "buy")
			_, _ = w.Write([]byte(`{"data":[{"pubkey":"order-1","status":"open","side":"buy","price":"0.68","amount":"120","remaining_amount":"100","filled_amount":"20","funds":null,"remaining_funds":null,"filled_funds":null,"market":{"condition_id":"market-1","title":"Market"},"outcome":{"is_yes":false,"text":"NO"}}],"meta":{"limit":2,"next_cursor":"` + nextCursor + `"},"error":null}`))
		case "/orders/order%2F1/":
			_, _ = w.Write([]byte(`{"data":{"pubkey":"order/1","status":"filled","side":"sell","price":"0.42","amount":"10","remaining_amount":"0","filled_amount":"10","funds":null,"remaining_funds":null,"filled_funds":null,"market":{"condition_id":"market-2","title":"Escaped Market"},"outcome":{"is_yes":true,"text":"YES"}},"meta":{},"error":null}`))
		default:
			t.Fatalf("unexpected path %s", r.URL.RequestURI())
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

	isYes := false
	orders, err := client.ListOrders(context.Background(), ListOrdersOptions{
		PageOptions:       PageOptions{Limit: 2, Cursor: "cursor"},
		MarketConditionID: "market-1",
		Status:            "OPEN,FILLED",
		IsYes:             &isYes,
		Side:              "buy",
	})
	if err != nil {
		t.Fatalf("ListOrders: %v", err)
	}
	if len(orders.Orders) != 1 || orders.Orders[0].Pubkey == nil || *orders.Orders[0].Pubkey != "order-1" {
		t.Fatalf("orders = %#v, want decoded order", orders.Orders)
	}
	if orders.Meta.Limit != 2 || orders.Meta.NextCursor == nil || *orders.Meta.NextCursor != nextCursor {
		t.Fatalf("meta = %#v, want limit=2 next_cursor=%q", orders.Meta, nextCursor)
	}

	order, err := client.GetOrder(context.Background(), "order/1")
	if err != nil {
		t.Fatalf("GetOrder: %v", err)
	}
	if order.Pubkey == nil || *order.Pubkey != "order/1" || order.Market.ConditionID != "market-2" {
		t.Fatalf("order = %#v, want escaped detail order", order)
	}
	if len(requests) != 2 || requests[1] != "/orders/order%2F1/" {
		t.Fatalf("requests = %#v, want escaped GetOrder path", requests)
	}
}

func TestMarginReadEndpoints(t *testing.T) {
	const (
		requestsCursor   = "next-requests"
		positionsCursor  = "next-positions"
		settlementCursor = "next-settlements"
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s, want GET", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.EscapedPath() {
		case "/margin/positions/estimate/":
			if r.Header.Get(headerAPIKey) != "" {
				t.Fatalf("%s = %q, want empty for public estimate endpoint", headerAPIKey, r.Header.Get(headerAPIKey))
			}
			assertQueryValue(t, r, "market_condition_id", "market-1")
			assertQueryValue(t, r, "funds", "10.5")
			assertQueryValue(t, r, "is_yes", "true")
			assertQueryValue(t, r, "leverage", "2.5")
			_, _ = w.Write([]byte(`{"data":{"average_price":"0.66","total_shares":"10","total_cost":"6.6","best_ask":"0.65","worst_fill_price":"0.67","is_fully_filled":true,"fee_amount":"0.1","user_funds_needed":"10.6","liquidation_price":"0.5"},"meta":{},"error":null}`))
		case "/margin/positions/requests/":
			assertAuthenticatedRequest(t, r)
			assertQueryValue(t, r, "limit", "2")
			assertQueryValue(t, r, "cursor", "request-cursor")
			assertQueryValue(t, r, "market_condition_id", "market-1")
			assertQueryValue(t, r, "states", "CREATED,COMPLETED")
			assertQueryValue(t, r, "is_yes", "false")
			assertQueryValue(t, r, "leverage", "3")
			assertQueryValue(t, r, "sort", "-created")
			_, _ = w.Write([]byte(`{"data":[{"pubkey":"request-1","type":"MARKET","state":"created","message":"sign request","funding_txid":"funding-1","refund_txid":null,"market":{"condition_id":"market-1","title":"Market"},"is_yes":false,"leverage":"3","funds":"10","price":null,"shares":null,"take_profit_price":"0.9","stop_loss_price":"0.2","created":1714300000}],"meta":{"limit":2,"next_cursor":"` + requestsCursor + `"},"error":null}`))
		case "/margin/positions/requests/request%2F1/":
			assertAuthenticatedRequest(t, r)
			_, _ = w.Write([]byte(`{"data":{"pubkey":"request/1","type":"LIMIT","state":"completed","message":null,"funding_txid":"funding-2","refund_txid":null,"market":{"condition_id":"market-2","title":"Request Market"},"is_yes":true,"leverage":"2","funds":"5","price":"0.55","shares":"9","take_profit_price":null,"stop_loss_price":null,"created":1714300001},"meta":{},"error":null}`))
		case "/margin/positions/":
			assertAuthenticatedRequest(t, r)
			assertQueryValue(t, r, "limit", "3")
			assertQueryValue(t, r, "cursor", "position-cursor")
			assertQueryValue(t, r, "market_condition_id", "market-1")
			assertQueryValue(t, r, "is_closed", "false")
			assertQueryValue(t, r, "sort", "created")
			_, _ = w.Write([]byte(`{"data":[{"pubkey":"position-1","position_request_pubkey":"request-1","market":{"condition_id":"market-1","title":"Market"},"is_yes":true,"leverage":"2","total_shares":"10","avg_entry_price":"0.5","closing_price":null,"unrealized_pnl":"1.2","realized_pnl":"0","user_liquidity":"5","total_liquidity":"10","liquidation_price":"0.2","is_closed":false,"is_liquidated":false,"is_claimed":false,"tp_sl":{"take_profit_price":"0.9","stop_loss_price":"0.2","state":"active","trigger_type":null,"triggered_price":null,"triggered_at":null},"created":1714300000}],"meta":{"limit":3,"next_cursor":"` + positionsCursor + `"},"error":null}`))
		case "/margin/positions/position%2F1/":
			assertAuthenticatedRequest(t, r)
			_, _ = w.Write([]byte(`{"data":{"pubkey":"position/1","position_request_pubkey":"request-2","market":{"condition_id":"market-2","title":"Position Market"},"is_yes":false,"leverage":"4","total_shares":"20","avg_entry_price":"0.4","closing_price":"0.6","unrealized_pnl":null,"realized_pnl":"2","user_liquidity":"8","total_liquidity":"32","liquidation_price":"0.1","is_closed":true,"is_liquidated":false,"is_claimed":true,"tp_sl":null,"created":1714300002},"meta":{},"error":null}`))
		case "/margin/positions/settlements/":
			assertAuthenticatedRequest(t, r)
			assertQueryValue(t, r, "limit", "4")
			assertQueryValue(t, r, "cursor", "settlement-cursor")
			assertQueryValue(t, r, "position_pubkey", "position-1")
			assertQueryValue(t, r, "market_condition_id", "market-1")
			assertQueryValue(t, r, "states", "CREATED,CLAIMED")
			assertQueryValue(t, r, "market_state", "resolved")
			assertQueryValue(t, r, "position_leverage", "2.5")
			assertQueryValue(t, r, "sort", "-created")
			_, _ = w.Write([]byte(`{"data":[{"position_pubkey":"position-1","position":{"pubkey":"position-1","position_request_pubkey":"request-1","market":{"condition_id":"market-1","title":"Market"},"is_yes":true,"leverage":"2.5","total_shares":"10","avg_entry_price":"0.5","closing_price":"0.8","unrealized_pnl":null,"realized_pnl":"3","user_liquidity":"5","total_liquidity":"10","liquidation_price":"0.2","is_closed":true,"is_liquidated":false,"is_claimed":true,"tp_sl":null,"created":1714300000},"total_pnl":"3","user_liquidity":"5","state":"claimed","created":1714300010}],"meta":{"limit":4,"next_cursor":"` + settlementCursor + `"},"error":null}`))
		default:
			t.Fatalf("unexpected path %s", r.URL.RequestURI())
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

	isYes := true
	leverage := 2.5
	estimate, err := client.EstimateMarginPosition(context.Background(), EstimateMarginPositionOptions{
		MarketConditionID: "market-1",
		Funds:             "10.5",
		IsYes:             &isYes,
		Leverage:          &leverage,
	})
	if err != nil {
		t.Fatalf("EstimateMarginPosition: %v", err)
	}
	if estimate.AveragePrice != "0.66" || estimate.LiquidationPrice == nil || *estimate.LiquidationPrice != "0.5" {
		t.Fatalf("estimate = %#v, want decoded estimate", estimate)
	}

	requestIsYes := false
	requestLeverage := 3.0
	positionRequests, err := client.ListPositionRequests(context.Background(), ListPositionRequestsOptions{
		PageOptions:       PageOptions{Limit: 2, Cursor: "request-cursor"},
		MarketConditionID: "market-1",
		States:            "CREATED,COMPLETED",
		IsYes:             &requestIsYes,
		Leverage:          &requestLeverage,
		Sort:              "-created",
	})
	if err != nil {
		t.Fatalf("ListPositionRequests: %v", err)
	}
	if len(positionRequests.Requests) != 1 || positionRequests.Requests[0].Pubkey != "request-1" {
		t.Fatalf("position requests = %#v, want decoded request", positionRequests.Requests)
	}
	if positionRequests.Meta.NextCursor == nil || *positionRequests.Meta.NextCursor != requestsCursor {
		t.Fatalf("position request meta = %#v, want next cursor", positionRequests.Meta)
	}

	positionRequest, err := client.GetPositionRequest(context.Background(), "request/1")
	if err != nil {
		t.Fatalf("GetPositionRequest: %v", err)
	}
	if positionRequest.Pubkey != "request/1" || positionRequest.Market == nil || positionRequest.Market.ConditionID != "market-2" {
		t.Fatalf("position request = %#v, want escaped detail request", positionRequest)
	}

	isClosed := false
	positions, err := client.ListMarginPositions(context.Background(), ListMarginPositionsOptions{
		PageOptions:       PageOptions{Limit: 3, Cursor: "position-cursor"},
		MarketConditionID: "market-1",
		IsClosed:          &isClosed,
		Sort:              "created",
	})
	if err != nil {
		t.Fatalf("ListMarginPositions: %v", err)
	}
	if len(positions.Positions) != 1 || positions.Positions[0].Pubkey != "position-1" || positions.Positions[0].TPSL == nil {
		t.Fatalf("positions = %#v, want decoded position with tp_sl", positions.Positions)
	}
	if positions.Meta.NextCursor == nil || *positions.Meta.NextCursor != positionsCursor {
		t.Fatalf("positions meta = %#v, want next cursor", positions.Meta)
	}

	position, err := client.GetMarginPosition(context.Background(), "position/1")
	if err != nil {
		t.Fatalf("GetMarginPosition: %v", err)
	}
	if position.Pubkey != "position/1" || !position.IsClosed || position.Market.ConditionID != "market-2" {
		t.Fatalf("position = %#v, want escaped detail position", position)
	}

	settlementLeverage := 2.5
	settlements, err := client.ListMarginSettlements(context.Background(), ListMarginSettlementsOptions{
		PageOptions:       PageOptions{Limit: 4, Cursor: "settlement-cursor"},
		PositionPubkey:    "position-1",
		MarketConditionID: "market-1",
		States:            "CREATED,CLAIMED",
		MarketState:       "resolved",
		PositionLeverage:  &settlementLeverage,
		Sort:              "-created",
	})
	if err != nil {
		t.Fatalf("ListMarginSettlements: %v", err)
	}
	if len(settlements.Settlements) != 1 || settlements.Settlements[0].PositionPubkey != "position-1" || settlements.Settlements[0].Position.Pubkey != "position-1" {
		t.Fatalf("settlements = %#v, want decoded settlement", settlements.Settlements)
	}
	if settlements.Meta.NextCursor == nil || *settlements.Meta.NextCursor != settlementCursor {
		t.Fatalf("settlement meta = %#v, want next cursor", settlements.Meta)
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

func TestParseSolanaPrivateKeyFormats(t *testing.T) {
	seed := testSeed()
	key := ed25519.NewKeyFromSeed(seed)
	tests := []struct {
		name  string
		input string
	}{
		{name: "base58 keypair", input: base58.Encode(key)},
		{name: "hex keypair", input: hex.EncodeToString(key)},
		{name: "json keypair", input: mustJSONBytes(t, key)},
		{name: "base58 seed", input: base58.Encode(seed)},
		{name: "hex seed", input: hex.EncodeToString(seed)},
		{name: "json seed", input: mustJSONBytes(t, seed)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseSolanaPrivateKey(tt.input)
			if err != nil {
				t.Fatalf("parseSolanaPrivateKey: %v", err)
			}
			if !ed25519.PrivateKey(got).Equal(key) {
				t.Fatalf("private key mismatch")
			}
		})
	}
}

func TestParseSolanaPrivateKeyInvalid(t *testing.T) {
	key := ed25519.NewKeyFromSeed(testSeed())
	mismatched := append([]byte(nil), key...)
	mismatched[len(mismatched)-1] ^= 0xff
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "empty", in: "  ", want: "required"},
		{name: "malformed json", in: "[1,", want: "json"},
		{name: "wrong length json", in: "[1,2,3]", want: "length"},
		{name: "wrong length base58", in: base58.Encode([]byte{1, 2, 3}), want: "length"},
		{name: "invalid base58", in: "0OIl", want: "base58"},
		{name: "mismatched keypair", in: base58.Encode(mismatched), want: "public key does not match seed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseSolanaPrivateKey(tt.in)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want contains %q", err, tt.want)
			}
		})
	}
}

func TestCreateAPIKeyFromPrivateKey(t *testing.T) {
	privateKey := ed25519.NewKeyFromSeed(testSeed())
	walletAddress := solanaWalletAddress(privateKey)
	const message = "Sign this message to create your Worm API key. Nonce: nonce_test"
	const nonce = "nonce_test"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/auth/keys/challenge/":
			if r.Method != http.MethodPost {
				t.Fatalf("challenge method = %s, want POST", r.Method)
			}
			var req CreateAuthChallengeRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode challenge request: %v", err)
			}
			if req.WalletAddress != walletAddress {
				t.Fatalf("wallet_address = %q, want %q", req.WalletAddress, walletAddress)
			}
			_, _ = w.Write([]byte(`{"data":{"nonce":"nonce_test","message":"Sign this message to create your Worm API key. Nonce: nonce_test","expires_in_seconds":600},"meta":{},"error":null}`))
		case "/auth/keys/create/":
			if r.Method != http.MethodPost {
				t.Fatalf("create method = %s, want POST", r.Method)
			}
			var req CreateAPIKeyRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode create request: %v", err)
			}
			if req.WalletAddress != walletAddress || req.Message != message || req.Nonce != nonce {
				t.Fatalf("create request = %#v", req)
			}
			signature, err := hex.DecodeString(req.Signature)
			if err != nil {
				t.Fatalf("signature is not hex: %v", err)
			}
			publicKey := privateKey.Public().(ed25519.PublicKey)
			if !ed25519.Verify(publicKey, []byte(message), signature) {
				t.Fatal("signature did not verify")
			}
			_, _ = w.Write([]byte(`{"data":{"api_key":"wk_test","secret":"ws_test"},"meta":{},"error":null}`))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client, err := NewClient(Config{BaseURL: server.URL})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	creds, err := client.CreateAPIKeyFromPrivateKey(context.Background(), base58.Encode(privateKey))
	if err != nil {
		t.Fatalf("CreateAPIKeyFromPrivateKey: %v", err)
	}
	if creds.APIKey != "wk_test" || creds.Secret != "ws_test" {
		t.Fatalf("creds = %#v", creds)
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

func testSeed() []byte {
	seed := make([]byte, ed25519.SeedSize)
	for i := range seed {
		seed[i] = byte(i + 1)
	}
	return seed
}

func mustJSONBytes(t *testing.T, value []byte) string {
	t.Helper()
	items := make([]int, len(value))
	for i, b := range value {
		items[i] = int(b)
	}
	body, err := json.Marshal(items)
	if err != nil {
		t.Fatalf("json marshal: %v", err)
	}
	return string(body)
}

func testSignature(secret string, payload string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(payload))
	return hex.EncodeToString(mac.Sum(nil))
}

func assertAuthenticatedRequest(t *testing.T, r *http.Request) {
	t.Helper()
	if r.Header.Get(headerAPIKey) != "wk_test" {
		t.Fatalf("%s = %q, want wk_test", headerAPIKey, r.Header.Get(headerAPIKey))
	}
	if r.Header.Get(headerTimestamp) == "" {
		t.Fatalf("%s is empty, want signed request", headerTimestamp)
	}
	if r.Header.Get(headerSignature) == "" {
		t.Fatalf("%s is empty, want signed request", headerSignature)
	}
}

func assertQueryValue(t *testing.T, r *http.Request, key string, want string) {
	t.Helper()
	if got := r.URL.Query().Get(key); got != want {
		t.Fatalf("query %s = %q, want %q in %q", key, got, want, r.URL.RawQuery)
	}
}
