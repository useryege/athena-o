package worm

import (
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	code := m.Run()
	cleanupSharedAuthReadIntegrationClient()
	os.Exit(code)
}

var (
	sharedAuthReadOnce   sync.Once
	sharedAuthReadClient Client
	sharedAuthReadCreds  *APIKeySecret
	sharedAuthReadErr    error
)

func TestIntegrationPublicSearch(t *testing.T) {
	ctx, cancel, client := newPublicIntegrationClient(t)
	defer cancel()

	resp, err := client.Search(ctx, SearchOptions{
		PageOptions: PageOptions{Limit: 1},
		State:       "open",
	})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(resp.Results) == 0 {
		t.Fatal("Search returned no results")
	}
	result := resp.Results[0]
	if result.ResultType == "" {
		t.Fatal("Search returned empty result_type")
	}
	validateSearchSummary(t, result.Summary)

	t.Logf("Search returned result_type=%q condition_id=%q title=%q", result.ResultType, result.Summary.ConditionID, result.Summary.Title)
}

func TestIntegrationPublicListMarkets(t *testing.T) {
	ctx, cancel, client := newPublicIntegrationClient(t)
	defer cancel()

	markets, err := client.ListMarkets(ctx, ListMarketsOptions{
		PageOptions: PageOptions{Limit: 1},
		State:       "open",
	})
	if err != nil {
		t.Fatalf("ListMarkets: %v", err)
	}
	if len(markets.Markets) == 0 {
		t.Fatal("ListMarkets returned no markets")
	}
	validateMarketSummary(t, markets.Markets[0])

	t.Logf("ListMarkets returned condition_id=%q title=%q", markets.Markets[0].ConditionID, markets.Markets[0].Title)
}

func TestIntegrationPublicGetMarket(t *testing.T) {
	ctx, cancel, client := newPublicIntegrationClient(t)
	defer cancel()

	conditionID := findOpenIntegrationMarket(t, ctx, client).ConditionID
	market, err := client.GetMarket(ctx, conditionID)
	if err != nil {
		t.Fatalf("GetMarket(%s): %v", conditionID, err)
	}
	validateMarketSummary(t, market.MarketSummary)

	t.Logf("GetMarket returned condition_id=%q title=%q state=%q rules=%d", market.ConditionID, market.Title, market.State, len(market.Rules))
}

func TestIntegrationPublicGetMarketStats(t *testing.T) {
	ctx, cancel, client := newPublicIntegrationClient(t)
	defer cancel()

	conditionID := findOpenIntegrationMarket(t, ctx, client).ConditionID
	stats, err := client.GetMarketStats(ctx, conditionID)
	if err != nil {
		t.Fatalf("GetMarketStats(%s): %v", conditionID, err)
	}
	if stats.TotalVolume == "" {
		t.Fatalf("GetMarketStats(%s) returned empty total_volume", conditionID)
	}
	if stats.TotalVolume24H == "" {
		t.Fatalf("GetMarketStats(%s) returned empty total_volume_24h", conditionID)
	}
	if stats.MarketCap == "" {
		t.Fatalf("GetMarketStats(%s) returned empty market_cap", conditionID)
	}
	if stats.TradeCount < 0 {
		t.Fatalf("GetMarketStats(%s) returned negative trade_count=%d", conditionID, stats.TradeCount)
	}

	t.Logf("GetMarketStats returned volume=%q volume_24h=%q market_cap=%q trades=%d", stats.TotalVolume, stats.TotalVolume24H, stats.MarketCap, stats.TradeCount)
}

func TestIntegrationPublicGetMarketPrice(t *testing.T) {
	ctx, cancel, client := newPublicIntegrationClient(t)
	defer cancel()

	conditionID := findOpenIntegrationMarket(t, ctx, client).ConditionID
	isYes := false
	price, err := client.GetMarketPrice(ctx, conditionID, GetMarketPriceOptions{IsYes: &isYes})
	if err != nil {
		t.Fatalf("GetMarketPrice(%s): %v", conditionID, err)
	}
	if price.ConditionID == "" {
		t.Fatalf("GetMarketPrice(%s) returned empty condition_id", conditionID)
	}
	if price.ConditionID != conditionID {
		t.Fatalf("GetMarketPrice(%s) returned condition_id=%q", conditionID, price.ConditionID)
	}
	if price.IsYes != isYes {
		t.Fatalf("GetMarketPrice(%s) returned is_yes=%t, want %t", conditionID, price.IsYes, isYes)
	}
	if price.Price != nil && *price.Price == "" {
		t.Fatalf("GetMarketPrice(%s) returned empty non-nil price", conditionID)
	}

	t.Logf("GetMarketPrice returned condition_id=%q is_yes=%t price_present=%t kind=%q", price.ConditionID, price.IsYes, price.Price != nil, price.PriceKind)
}

func TestIntegrationPublicGetMarketOrderBook(t *testing.T) {
	ctx, cancel, client := newPublicIntegrationClient(t)
	defer cancel()

	conditionID := findOpenIntegrationMarket(t, ctx, client).ConditionID
	isYes := false
	depth := 5
	book, err := client.GetMarketOrderBook(ctx, conditionID, GetMarketOrderBookOptions{Depth: depth, IsYes: &isYes})
	if err != nil {
		t.Fatalf("GetMarketOrderBook(%s): %v", conditionID, err)
	}
	if book.Market == "" {
		t.Fatalf("GetMarketOrderBook(%s) returned empty market", conditionID)
	}
	if book.Market != conditionID {
		t.Fatalf("GetMarketOrderBook(%s) returned market=%q", conditionID, book.Market)
	}
	if book.IsYes != isYes {
		t.Fatalf("GetMarketOrderBook(%s) returned is_yes=%t, want %t", conditionID, book.IsYes, isYes)
	}
	if len(book.Bid) > depth {
		t.Fatalf("GetMarketOrderBook(%s) returned %d bid levels, want <= %d", conditionID, len(book.Bid), depth)
	}
	if len(book.Ask) > depth {
		t.Fatalf("GetMarketOrderBook(%s) returned %d ask levels, want <= %d", conditionID, len(book.Ask), depth)
	}
	validateOrderBookLevels(t, "bid", book.Bid)
	validateOrderBookLevels(t, "ask", book.Ask)

	t.Logf("GetMarketOrderBook returned market=%q is_yes=%t bid_levels=%d ask_levels=%d", book.Market, book.IsYes, len(book.Bid), len(book.Ask))
}

func TestIntegrationPublicGetMarketCandles(t *testing.T) {
	ctx, cancel, client := newPublicIntegrationClient(t)
	defer cancel()

	conditionID := findOpenIntegrationMarket(t, ctx, client).ConditionID
	isYes := true
	endTime := time.Now().Unix()
	startTime := endTime - int64(24*time.Hour/time.Second)
	candles, err := client.GetMarketCandles(ctx, conditionID, GetMarketCandlesOptions{
		StartTime: startTime,
		EndTime:   endTime,
		Interval:  "30m",
		IsYes:     &isYes,
	})
	if err != nil {
		t.Fatalf("GetMarketCandles(%s): %v", conditionID, err)
	}
	if candles.Meta.IsYes != nil && *candles.Meta.IsYes != isYes {
		t.Fatalf("GetMarketCandles(%s) returned meta is_yes=%t, want %t", conditionID, *candles.Meta.IsYes, isYes)
	}
	for i, candle := range candles.Candles {
		if candle.Timestamp <= 0 {
			t.Fatalf("GetMarketCandles(%s) candle[%d] has empty timestamp", conditionID, i)
		}
		if candle.Open == "" || candle.High == "" || candle.Low == "" || candle.Close == "" || candle.Volume == "" {
			t.Fatalf("GetMarketCandles(%s) candle[%d] has empty OHLCV fields: %#v", conditionID, i, candle)
		}
		if candle.IsYes != isYes {
			t.Fatalf("GetMarketCandles(%s) candle[%d] is_yes=%t, want %t", conditionID, i, candle.IsYes, isYes)
		}
	}

	t.Logf("GetMarketCandles returned condition_id=%q candles=%d", conditionID, len(candles.Candles))
}

func TestIntegrationPublicListMarketTrades(t *testing.T) {
	ctx, cancel, client := newPublicIntegrationClient(t)
	defer cancel()

	conditionID := findOpenIntegrationMarket(t, ctx, client).ConditionID
	limit := 5
	trades, err := client.ListMarketTrades(ctx, conditionID, ListMarketTradesOptions{
		PageOptions: PageOptions{Limit: limit},
	})
	if err != nil {
		t.Fatalf("ListMarketTrades(%s): %v", conditionID, err)
	}
	if len(trades.Trades) > limit {
		t.Fatalf("ListMarketTrades(%s) returned %d trades, want <= %d", conditionID, len(trades.Trades), limit)
	}
	for i, trade := range trades.Trades {
		if trade.MarketConditionID != nil && *trade.MarketConditionID == "" {
			t.Fatalf("ListMarketTrades(%s) trade[%d] has empty non-nil market_condition_id", conditionID, i)
		}
		if trade.Amount == "" || trade.Price == "" || trade.MakerFee == "" || trade.TakerFee == "" || trade.State == "" {
			t.Fatalf("ListMarketTrades(%s) trade[%d] has empty required fields: %#v", conditionID, i, trade)
		}
		if trade.Timestamp <= 0 {
			t.Fatalf("ListMarketTrades(%s) trade[%d] has empty timestamp", conditionID, i)
		}
	}

	t.Logf("ListMarketTrades returned condition_id=%q trades=%d", conditionID, len(trades.Trades))
}

func TestIntegrationPublicListMarketMarginActivity(t *testing.T) {
	ctx, cancel, client := newPublicIntegrationClient(t)
	defer cancel()

	conditionID := findOpenIntegrationMarket(t, ctx, client).ConditionID
	limit := 5
	activities, err := client.ListMarketMarginActivity(ctx, conditionID, ListMarketMarginActivityOptions{
		PageOptions: PageOptions{Limit: limit},
	})
	if err != nil {
		t.Fatalf("ListMarketMarginActivity(%s): %v", conditionID, err)
	}
	if len(activities.Activities) > limit {
		t.Fatalf("ListMarketMarginActivity(%s) returned %d activities, want <= %d", conditionID, len(activities.Activities), limit)
	}
	for i, activity := range activities.Activities {
		if activity.ActivityType == "" {
			t.Fatalf("ListMarketMarginActivity(%s) activity[%d] has empty activity_type", conditionID, i)
		}
		if activity.Created <= 0 {
			t.Fatalf("ListMarketMarginActivity(%s) activity[%d] has empty created timestamp", conditionID, i)
		}
		if activity.Market != nil {
			validateMarketSummary(t, *activity.Market)
		}
	}

	t.Logf("ListMarketMarginActivity returned condition_id=%q activities=%d", conditionID, len(activities.Activities))
}

func TestIntegrationPublicListEvents(t *testing.T) {
	ctx, cancel, client := newPublicIntegrationClient(t)
	defer cancel()

	events, err := client.ListEvents(ctx, ListEventsOptions{
		PageOptions: PageOptions{Limit: 1},
		State:       "open",
	})
	if err != nil {
		t.Fatalf("ListEvents: %v", err)
	}
	if len(events.Events) == 0 {
		t.Fatal("ListEvents returned no events")
	}
	validateEvent(t, events.Events[0])

	t.Logf("ListEvents returned condition_id=%q title=%q markets=%d", events.Events[0].ConditionID, events.Events[0].Title, len(events.Events[0].Markets))
}

func TestIntegrationPublicGetEvent(t *testing.T) {
	ctx, cancel, client := newPublicIntegrationClient(t)
	defer cancel()

	conditionID := findIntegrationEvent(t, ctx, client).ConditionID
	event, err := client.GetEvent(ctx, conditionID)
	if err != nil {
		t.Fatalf("GetEvent(%s): %v", conditionID, err)
	}
	validateEvent(t, *event)

	t.Logf("GetEvent returned condition_id=%q title=%q markets=%d", event.ConditionID, event.Title, len(event.Markets))
}

func TestIntegrationAuthKeysCreateAuthChallenge(t *testing.T) {
	fixture := newAuthKeysIntegrationFixture(t)
	defer fixture.cancel()

	challenge, err := fixture.client.CreateAuthChallenge(fixture.ctx, CreateAuthChallengeRequest{WalletAddress: fixture.walletAddress})
	if err != nil {
		t.Fatalf("CreateAuthChallenge: %v", err)
	}
	validateAuthChallenge(t, challenge)

	t.Logf("CreateAuthChallenge returned nonce=%t message=%t expires_in_seconds=%d", challenge.Nonce != "", challenge.Message != "", challenge.ExpiresInSeconds)
}

func TestIntegrationAuthKeysCreateAPIKey(t *testing.T) {
	fixture := newAuthKeysIntegrationFixture(t)
	defer fixture.cancel()

	challenge, err := fixture.client.CreateAuthChallenge(fixture.ctx, CreateAuthChallengeRequest{WalletAddress: fixture.walletAddress})
	if err != nil {
		t.Fatalf("CreateAuthChallenge: %v", err)
	}
	validateAuthChallenge(t, challenge)

	signature := ed25519.Sign(fixture.privateKey, []byte(challenge.Message))
	creds, err := fixture.client.CreateAPIKey(fixture.ctx, CreateAPIKeyRequest{
		WalletAddress: fixture.walletAddress,
		Message:       challenge.Message,
		Signature:     hex.EncodeToString(signature),
		Nonce:         challenge.Nonce,
	})
	if err != nil {
		t.Fatalf("CreateAPIKey: %v", err)
	}
	validateAPIKeySecret(t, creds)

	revoke := registerTemporaryAPIKeyCleanup(t, creds)

	revoked := revoke(fixture.ctx)
	validateRevokedAPIKey(t, revoked, creds.APIKey)

	t.Logf("CreateAPIKey returned api_key=%t secret=%t and revoked temporary key=%t", creds.APIKey != "", creds.Secret != "", revoked.RevokedAt != nil)
}

func TestIntegrationAuthKeysCreateAPIKeyFromPrivateKey(t *testing.T) {
	fixture := newAuthKeysIntegrationFixture(t)
	defer fixture.cancel()

	creds, err := fixture.client.CreateAPIKeyFromPrivateKey(fixture.ctx, fixture.privateKeyInput)
	if err != nil {
		t.Fatalf("CreateAPIKeyFromPrivateKey: %v", err)
	}
	validateAPIKeySecret(t, creds)
	revoke := registerTemporaryAPIKeyCleanup(t, creds)

	revoked := revoke(fixture.ctx)
	validateRevokedAPIKey(t, revoked, creds.APIKey)

	t.Logf("CreateAPIKeyFromPrivateKey returned api_key=%t secret=%t and revoked temporary key=%t", creds.APIKey != "", creds.Secret != "", revoked.RevokedAt != nil)
}

func TestIntegrationAuthKeysListAndRevokeAPIKeys(t *testing.T) {
	fixture := newAuthKeysIntegrationFixture(t)
	defer fixture.cancel()

	creds, err := fixture.client.CreateAPIKeyFromPrivateKey(fixture.ctx, fixture.privateKeyInput)
	if err != nil {
		t.Fatalf("CreateAPIKeyFromPrivateKey: %v", err)
	}
	validateAPIKeySecret(t, creds)
	revoke := registerTemporaryAPIKeyCleanup(t, creds)

	authClient := newAuthenticatedIntegrationClient(t, creds)
	keys, err := authClient.ListAPIKeys(fixture.ctx)
	if err != nil {
		t.Fatalf("ListAPIKeys: %v", err)
	}
	if !hasAPIKey(keys.Keys, creds.APIKey) {
		t.Fatalf("ListAPIKeys did not include newly created temporary key")
	}

	revoked := revoke(fixture.ctx)
	validateRevokedAPIKey(t, revoked, creds.APIKey)

	t.Logf("ListAPIKeys returned keys=%d and RevokeAPIKey revoked temporary key=%t", len(keys.Keys), revoked.RevokedAt != nil)
}

func TestIntegrationAuthReadListTrades(t *testing.T) {
	fixture := newAuthReadIntegrationFixture(t)
	defer fixture.cancel()

	limit := 5
	trades, err := fixture.client.ListTrades(fixture.ctx, ListTradesOptions{
		PageOptions: PageOptions{Limit: limit},
	})
	if err != nil {
		t.Fatalf("ListTrades: %v", err)
	}
	if len(trades.Trades) > limit {
		t.Fatalf("ListTrades returned %d trades, want <= %d", len(trades.Trades), limit)
	}
	for i, trade := range trades.Trades {
		validateIntegrationTrade(t, i, trade)
	}

	t.Logf("ListTrades returned trades=%d", len(trades.Trades))
}

func TestIntegrationAuthReadListOrders(t *testing.T) {
	fixture := newAuthReadIntegrationFixture(t)
	defer fixture.cancel()

	limit := 5
	orders, err := fixture.client.ListOrders(fixture.ctx, ListOrdersOptions{
		PageOptions: PageOptions{Limit: limit},
	})
	if err != nil {
		t.Fatalf("ListOrders: %v", err)
	}
	if len(orders.Orders) > limit {
		t.Fatalf("ListOrders returned %d orders, want <= %d", len(orders.Orders), limit)
	}
	for i, order := range orders.Orders {
		validateIntegrationOrder(t, i, order)
	}

	t.Logf("ListOrders returned orders=%d", len(orders.Orders))
}

func TestIntegrationAuthReadGetOrderFromList(t *testing.T) {
	fixture := newAuthReadIntegrationFixture(t)
	defer fixture.cancel()

	orders, err := fixture.client.ListOrders(fixture.ctx, ListOrdersOptions{
		PageOptions: PageOptions{Limit: 1},
	})
	if err != nil {
		t.Fatalf("ListOrders: %v", err)
	}
	if len(orders.Orders) == 0 {
		t.Skip("authenticated account has no orders to fetch by pubkey")
	}
	if orders.Orders[0].Pubkey == nil || *orders.Orders[0].Pubkey == "" {
		t.Skip("authenticated account returned an order without a pubkey to fetch")
	}

	wantPubkey := *orders.Orders[0].Pubkey
	order, err := fixture.client.GetOrder(fixture.ctx, wantPubkey)
	if err != nil {
		t.Fatalf("GetOrder(%s): %v", wantPubkey, err)
	}
	validateIntegrationOrder(t, 0, *order)
	if order.Pubkey == nil || *order.Pubkey != wantPubkey {
		t.Fatalf("GetOrder returned pubkey=%#v, want %q", order.Pubkey, wantPubkey)
	}

	t.Logf("GetOrder returned pubkey=%q status_present=%t", wantPubkey, order.Status != nil)
}

func TestIntegrationAuthReadAccountSummaryAndPnL(t *testing.T) {
	fixture := newAuthReadIntegrationFixture(t)
	defer fixture.cancel()

	account, err := fixture.client.GetAccountSummary(fixture.ctx)
	if err != nil {
		t.Fatalf("GetAccountSummary: %v", err)
	}
	if account.Username == "" {
		t.Fatal("GetAccountSummary returned empty username")
	}
	if account.TwitterUsername != nil && *account.TwitterUsername == "" {
		t.Fatal("GetAccountSummary returned empty non-nil twitter_username")
	}
	if account.JoinedAt != nil && *account.JoinedAt <= 0 {
		t.Fatalf("GetAccountSummary returned joined_at=%d, want positive timestamp", *account.JoinedAt)
	}

	pnl, err := fixture.client.GetAccountPnL(fixture.ctx, GetAccountPnLOptions{})
	if err != nil {
		t.Fatalf("GetAccountPnL: %v", err)
	}
	validateIntegrationAccountPnL(t, pnl)

	t.Logf("GetAccountSummaryAndPnL returned username=%q joined_at=%t total_pnl=%q", account.Username, account.JoinedAt != nil, pnl.TotalPnL)
}

func TestIntegrationAuthReadListAccountAssets(t *testing.T) {
	fixture := newAuthReadIntegrationFixture(t)
	defer fixture.cancel()

	limit := 5
	assets, err := fixture.client.ListAccountAssets(fixture.ctx, ListAccountAssetsOptions{
		PageOptions: PageOptions{Limit: limit},
	})
	if err != nil {
		t.Fatalf("ListAccountAssets: %v", err)
	}
	if len(assets.Assets) > limit {
		t.Fatalf("ListAccountAssets returned %d assets, want <= %d", len(assets.Assets), limit)
	}
	for i, asset := range assets.Assets {
		validateIntegrationAccountAsset(t, i, asset)
	}

	t.Logf("ListAccountAssets returned assets=%d", len(assets.Assets))
}

func TestIntegrationAuthReadListRedeems(t *testing.T) {
	fixture := newAuthReadIntegrationFixture(t)
	defer fixture.cancel()

	limit := 5
	redeems, err := fixture.client.ListRedeems(fixture.ctx, ListRedeemsOptions{
		PageOptions: PageOptions{Limit: limit},
	})
	if err != nil {
		t.Fatalf("ListRedeems: %v", err)
	}
	if len(redeems.Redeems) > limit {
		t.Fatalf("ListRedeems returned %d redeems, want <= %d", len(redeems.Redeems), limit)
	}
	for i, redeem := range redeems.Redeems {
		validateIntegrationRedeem(t, i, redeem)
	}

	t.Logf("ListRedeems returned redeems=%d", len(redeems.Redeems))
}

func TestIntegrationAuthReadGetRedeemFromList(t *testing.T) {
	fixture := newAuthReadIntegrationFixture(t)
	defer fixture.cancel()

	redeems, err := fixture.client.ListRedeems(fixture.ctx, ListRedeemsOptions{
		PageOptions: PageOptions{Limit: 1},
	})
	if err != nil {
		t.Fatalf("ListRedeems: %v", err)
	}
	if len(redeems.Redeems) == 0 {
		t.Skip("authenticated account has no redeems to fetch by pubkey")
	}

	wantPubkey := redeems.Redeems[0].Pubkey
	redeem, err := fixture.client.GetRedeem(fixture.ctx, wantPubkey)
	if err != nil {
		t.Fatalf("GetRedeem(%s): %v", wantPubkey, err)
	}
	validateIntegrationRedeem(t, 0, *redeem)
	if redeem.Pubkey != wantPubkey {
		t.Fatalf("GetRedeem returned pubkey=%q, want %q", redeem.Pubkey, wantPubkey)
	}

	t.Logf("GetRedeem returned pubkey=%q state=%q", redeem.Pubkey, redeem.State)
}

func TestIntegrationAuthReadEstimateMarginPosition(t *testing.T) {
	fixture := newAuthReadIntegrationFixture(t)
	defer fixture.cancel()

	market, isYes, estimate := findIntegrationMarginPositionEstimate(t, fixture.ctx, fixture.client)
	validateIntegrationMarginPositionEstimate(t, estimate)

	t.Logf("EstimateMarginPosition returned condition_id=%q is_yes=%t average_price=%q total_shares=%q", market.ConditionID, isYes, estimate.AveragePrice, estimate.TotalShares)
}

func TestIntegrationEstimateMarginPositionFromEnv(t *testing.T) {
	if os.Getenv("WORM_INTEGRATION") != "1" {
		t.Skip("set WORM_INTEGRATION=1 to run real Worm API integration tests")
	}
	if os.Getenv("WORM_MARGIN_ESTIMATE_INTEGRATION") != "1" {
		t.Skip("set WORM_MARGIN_ESTIMATE_INTEGRATION=1 to run real EstimateMarginPosition integration test")
	}

	marketConditionID := requiredTrimmedIntegrationEnv(t, "WORM_MARGIN_ESTIMATE_MARKET_CONDITION_ID")
	funds := requiredTrimmedIntegrationEnv(t, "WORM_MARGIN_ESTIMATE_FUNDS")
	isYes := requiredBoolIntegrationEnv(t, "WORM_MARGIN_ESTIMATE_IS_YES")
	leverage := requiredFloatIntegrationEnv(t, "WORM_MARGIN_ESTIMATE_LEVERAGE")

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	client, err := NewClient(Config{})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	estimate, err := client.EstimateMarginPosition(ctx, EstimateMarginPositionOptions{
		MarketConditionID: marketConditionID,
		Funds:             funds,
		IsYes:             &isYes,
		Leverage:          &leverage,
	})
	if err != nil {
		t.Fatalf("EstimateMarginPosition: %v", err)
	}
	validateIntegrationMarginPositionEstimate(t, estimate)

	t.Logf(
		"EstimateMarginPosition returned condition_id=%q is_yes=%t funds=%q leverage=%.8g average_price=%q total_shares=%q total_cost=%q is_fully_filled=%t liquidation_price_present=%t user_funds_needed=%q",
		marketConditionID,
		isYes,
		funds,
		leverage,
		estimate.AveragePrice,
		estimate.TotalShares,
		estimate.TotalCost,
		estimate.IsFullyFilled,
		estimate.LiquidationPrice != nil && *estimate.LiquidationPrice != "",
		estimate.UserFundsNeeded,
	)
	if !estimate.IsFullyFilled {
		t.Logf("EstimateMarginPosition returned is_fully_filled=false; this parameter set should not be used for the submit integration test")
	}
}

func TestIntegrationCreatePositionRequestFromEnv(t *testing.T) {
	if os.Getenv("WORM_INTEGRATION") != "1" {
		t.Skip("set WORM_INTEGRATION=1 to run real Worm API integration tests")
	}
	if os.Getenv("WORM_POSITION_REQUEST_DRAFT_INTEGRATION") != "1" {
		t.Skip("set WORM_POSITION_REQUEST_DRAFT_INTEGRATION=1 to run real CreatePositionRequest integration test")
	}

	privateKey := requiredTrimmedIntegrationEnv(t, "WORM_PRIVATE_KEY")
	request := positionRequestDraftIntegrationRequest(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	publicClient, err := NewClient(Config{})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if request.Type == "MARKET" {
		estimate, err := publicClient.EstimateMarginPosition(ctx, EstimateMarginPositionOptions{
			MarketConditionID: request.MarketConditionID,
			Funds:             request.Funds,
			IsYes:             request.IsYes,
			Leverage:          request.Leverage,
		})
		if err != nil {
			t.Fatalf("EstimateMarginPosition preflight: %v", err)
		}
		validateIntegrationMarginPositionEstimate(t, estimate)
		t.Logf(
			"EstimateMarginPosition preflight average_price=%q total_shares=%q total_cost=%q is_fully_filled=%t user_funds_needed=%q liquidation_price_present=%t",
			estimate.AveragePrice,
			estimate.TotalShares,
			estimate.TotalCost,
			estimate.IsFullyFilled,
			estimate.UserFundsNeeded,
			estimate.LiquidationPrice != nil && *estimate.LiquidationPrice != "",
		)
	}

	creds, err := publicClient.CreateAPIKeyFromPrivateKey(ctx, privateKey)
	if err != nil {
		t.Fatalf("CreateAPIKeyFromPrivateKey: %v", err)
	}
	validateAPIKeySecret(t, creds)
	registerTemporaryAPIKeyCleanup(t, creds)

	client := newAuthenticatedIntegrationClient(t, creds)
	draft, err := client.CreatePositionRequest(ctx, request)
	if err != nil {
		t.Fatalf("CreatePositionRequest: %v", err)
	}
	validateCreatedPositionRequestDraft(t, draft, request.Type)
	cancelDraft := registerPositionRequestDraftCleanup(t, client, draft.Pubkey)

	fetched, err := client.GetPositionRequest(ctx, draft.Pubkey)
	if err != nil {
		t.Fatalf("GetPositionRequest(%s): %v", draft.Pubkey, err)
	}
	validateCreatedPositionRequestDraft(t, fetched, request.Type)
	if fetched.Pubkey != draft.Pubkey {
		t.Fatalf("GetPositionRequest returned pubkey=%q, want %q", fetched.Pubkey, draft.Pubkey)
	}

	cancelled := cancelDraft(ctx)
	t.Logf(
		"CancelPositionRequest cleanup pubkey_present=%t state=%q funding_txid_present=%t refund_txid_present=%t",
		cancelled.Pubkey != "",
		cancelled.State,
		cancelled.FundingTxID != nil && *cancelled.FundingTxID != "",
		cancelled.RefundTxID != nil && *cancelled.RefundTxID != "",
	)

	t.Logf(
		"CreatePositionRequest returned pubkey_present=%t state=%q type=%q message_present=%t message_len=%d leverage=%q funds=%q market_present=%t",
		draft.Pubkey != "",
		draft.State,
		draft.Type,
		draft.Message != nil && *draft.Message != "",
		stringPtrLen(draft.Message),
		draft.Leverage,
		draft.Funds,
		draft.Market != nil,
	)
}

func TestIntegrationAuthReadListPositionRequests(t *testing.T) {
	fixture := newAuthReadIntegrationFixture(t)
	defer fixture.cancel()

	limit := 5
	requests, err := fixture.client.ListPositionRequests(fixture.ctx, ListPositionRequestsOptions{
		PageOptions: PageOptions{Limit: limit},
	})
	if err != nil {
		t.Fatalf("ListPositionRequests: %v", err)
	}
	if len(requests.Requests) > limit {
		t.Fatalf("ListPositionRequests returned %d requests, want <= %d", len(requests.Requests), limit)
	}
	for i, request := range requests.Requests {
		validateIntegrationPositionRequest(t, i, request)
	}

	t.Logf("ListPositionRequests returned requests=%d", len(requests.Requests))
}

func TestIntegrationAuthReadGetPositionRequestFromList(t *testing.T) {
	fixture := newAuthReadIntegrationFixture(t)
	defer fixture.cancel()

	requests, err := fixture.client.ListPositionRequests(fixture.ctx, ListPositionRequestsOptions{
		PageOptions: PageOptions{Limit: 1},
	})
	if err != nil {
		t.Fatalf("ListPositionRequests: %v", err)
	}
	if len(requests.Requests) == 0 {
		t.Skip("authenticated account has no position requests to fetch by pubkey")
	}
	wantPubkey := requests.Requests[0].Pubkey
	if wantPubkey == "" {
		t.Skip("authenticated account returned a position request without a pubkey to fetch")
	}

	request, err := fixture.client.GetPositionRequest(fixture.ctx, wantPubkey)
	if err != nil {
		t.Fatalf("GetPositionRequest(%s): %v", wantPubkey, err)
	}
	validateIntegrationPositionRequest(t, 0, *request)
	if request.Pubkey != wantPubkey {
		t.Fatalf("GetPositionRequest returned pubkey=%q, want %q", request.Pubkey, wantPubkey)
	}

	t.Logf("GetPositionRequest returned pubkey=%q state=%q", request.Pubkey, request.State)
}

func TestIntegrationAuthReadListMarginPositions(t *testing.T) {
	fixture := newAuthReadIntegrationFixture(t)
	defer fixture.cancel()

	limit := 5
	positions, err := fixture.client.ListMarginPositions(fixture.ctx, ListMarginPositionsOptions{
		PageOptions: PageOptions{Limit: limit},
	})
	if err != nil {
		t.Fatalf("ListMarginPositions: %v", err)
	}
	if len(positions.Positions) > limit {
		t.Fatalf("ListMarginPositions returned %d positions, want <= %d", len(positions.Positions), limit)
	}
	for i, position := range positions.Positions {
		validateIntegrationMarginPosition(t, i, position)
	}

	t.Logf("ListMarginPositions returned positions=%d", len(positions.Positions))
}

func TestIntegrationAuthReadGetMarginPositionFromList(t *testing.T) {
	fixture := newAuthReadIntegrationFixture(t)
	defer fixture.cancel()

	positions, err := fixture.client.ListMarginPositions(fixture.ctx, ListMarginPositionsOptions{
		PageOptions: PageOptions{Limit: 1},
	})
	if err != nil {
		t.Fatalf("ListMarginPositions: %v", err)
	}
	if len(positions.Positions) == 0 {
		t.Skip("authenticated account has no margin positions to fetch by pubkey")
	}
	wantPubkey := positions.Positions[0].Pubkey
	if wantPubkey == "" {
		t.Skip("authenticated account returned a margin position without a pubkey to fetch")
	}

	position, err := fixture.client.GetMarginPosition(fixture.ctx, wantPubkey)
	if err != nil {
		t.Fatalf("GetMarginPosition(%s): %v", wantPubkey, err)
	}
	validateIntegrationMarginPosition(t, 0, *position)
	if position.Pubkey != wantPubkey {
		t.Fatalf("GetMarginPosition returned pubkey=%q, want %q", position.Pubkey, wantPubkey)
	}

	t.Logf("GetMarginPosition returned pubkey=%q closed=%t claimed=%t", position.Pubkey, position.IsClosed, position.IsClaimed)
}

func TestIntegrationAuthReadListMarginSettlements(t *testing.T) {
	fixture := newAuthReadIntegrationFixture(t)
	defer fixture.cancel()

	limit := 5
	settlements, err := fixture.client.ListMarginSettlements(fixture.ctx, ListMarginSettlementsOptions{
		PageOptions: PageOptions{Limit: limit},
	})
	if err != nil {
		t.Fatalf("ListMarginSettlements: %v", err)
	}
	if len(settlements.Settlements) > limit {
		t.Fatalf("ListMarginSettlements returned %d settlements, want <= %d", len(settlements.Settlements), limit)
	}
	for i, settlement := range settlements.Settlements {
		validateIntegrationMarginSettlement(t, i, settlement)
	}

	t.Logf("ListMarginSettlements returned settlements=%d", len(settlements.Settlements))
}

func TestIntegrationCreateOrderDraft(t *testing.T) {
	if os.Getenv("WORM_INTEGRATION") != "1" {
		t.Skip("set WORM_INTEGRATION=1 to run real Worm API integration tests")
	}
	if os.Getenv("WORM_ORDER_DRAFT_INTEGRATION") != "1" {
		t.Skip("set WORM_ORDER_DRAFT_INTEGRATION=1 to run real CreateOrderDraft integration test")
	}

	privateKey := requiredIntegrationEnv(t, "WORM_PRIVATE_KEY")

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	client, err := NewClient(Config{})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	markets, err := client.ListMarkets(ctx, ListMarketsOptions{
		PageOptions: PageOptions{Limit: 1},
		State:       "open",
	})
	if err != nil {
		t.Fatalf("ListMarkets: %v", err)
	}
	if len(markets.Markets) == 0 {
		t.Fatal("ListMarkets returned no open markets")
	}
	t.Logf("ListMarkets returned %d open markets", len(markets.Markets))
	conditionID := markets.Markets[0].ConditionID
	if conditionID == "" {
		t.Fatal("ListMarkets returned empty condition_id")
	}
	t.Logf("ListMarkets returned condition_id=%q", conditionID)

	creds, err := client.CreateAPIKeyFromPrivateKey(ctx, privateKey)
	if err != nil {
		t.Fatalf("CreateAPIKeyFromPrivateKey: %v", err)
	}
	if creds.APIKey == "" {
		t.Fatal("CreateAPIKeyFromPrivateKey returned empty APIKey")
	}
	if creds.Secret == "" {
		t.Fatal("CreateAPIKeyFromPrivateKey returned empty Secret")
	}
	t.Logf("CreateAPIKeyFromPrivateKey returned api_key=%s secret=%s", creds.APIKey, creds.Secret)

	client, err = NewClient(Config{
		APIKey:    creds.APIKey,
		APISecret: creds.Secret,
	})
	if err != nil {
		t.Fatalf("NewClient with bootstrapped credentials: %v", err)
	}

	resp, err := client.CreateOrderDraft(ctx, CreateOrderDraftRequest{
		MarketConditionID: conditionID,
		IsYes:             true,
		Side:              "BUY",
		OrderType:         "MARKET",
		Funds:             "1.00",
	})
	if err != nil {
		t.Fatalf("CreateOrderDraft: %v", err)
	}
	if (resp.Pubkey == nil || *resp.Pubkey == "") && (resp.Message == nil || *resp.Message == "") {
		t.Fatalf("CreateOrderDraft returned no pubkey or message: %#v", resp)
	}

	t.Logf(
		"CreateOrderDraft market_selected=%t bootstrapped credentials=%t returned pubkey=%t message=%t",
		conditionID != "",
		creds.APIKey != "" && creds.Secret != "",
		resp.Pubkey != nil && *resp.Pubkey != "",
		resp.Message != nil && *resp.Message != "",
	)
}

type authKeysIntegrationFixture struct {
	ctx             context.Context
	cancel          context.CancelFunc
	client          Client
	privateKeyInput string
	privateKey      ed25519.PrivateKey
	walletAddress   string
}

func newAuthKeysIntegrationFixture(t *testing.T) authKeysIntegrationFixture {
	t.Helper()
	if os.Getenv("WORM_INTEGRATION") != "1" {
		t.Skip("set WORM_INTEGRATION=1 to run real Worm API integration tests")
	}
	if os.Getenv("WORM_AUTH_KEYS_INTEGRATION") != "1" {
		t.Skip("set WORM_AUTH_KEYS_INTEGRATION=1 to run real auth-key integration tests")
	}

	privateKeyInput := requiredIntegrationEnv(t, "WORM_PRIVATE_KEY")
	privateKey, err := parseSolanaPrivateKey(privateKeyInput)
	if err != nil {
		t.Fatalf("parse WORM_PRIVATE_KEY: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	client, err := NewClient(Config{})
	if err != nil {
		cancel()
		t.Fatalf("NewClient: %v", err)
	}

	return authKeysIntegrationFixture{
		ctx:             ctx,
		cancel:          cancel,
		client:          client,
		privateKeyInput: privateKeyInput,
		privateKey:      privateKey,
		walletAddress:   solanaWalletAddress(privateKey),
	}
}

type authReadIntegrationFixture struct {
	ctx    context.Context
	cancel context.CancelFunc
	client Client
}

func newAuthReadIntegrationFixture(t *testing.T) authReadIntegrationFixture {
	t.Helper()
	if os.Getenv("WORM_INTEGRATION") != "1" {
		t.Skip("set WORM_INTEGRATION=1 to run real Worm API integration tests")
	}
	if os.Getenv("WORM_AUTH_READ_INTEGRATION") != "1" {
		t.Skip("set WORM_AUTH_READ_INTEGRATION=1 to run real authenticated read integration tests")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	client, err := sharedAuthReadIntegrationClient()
	if err != nil {
		cancel()
		if isWormThrottleError(err) {
			t.Skipf("skipping authenticated read integration test because Worm API key bootstrap was throttled: %v", err)
		}
		t.Fatalf("authenticated read integration client: %v", err)
	}

	return authReadIntegrationFixture{ctx: ctx, cancel: cancel, client: client}
}

func sharedAuthReadIntegrationClient() (Client, error) {
	sharedAuthReadOnce.Do(func() {
		apiKey := strings.TrimSpace(os.Getenv("WORM_API_KEY"))
		apiSecret := strings.TrimSpace(os.Getenv("WORM_API_SECRET"))
		if apiKey != "" && apiSecret != "" {
			sharedAuthReadClient, sharedAuthReadErr = NewClient(Config{
				APIKey:    apiKey,
				APISecret: apiSecret,
			})
			return
		}
		if apiKey != "" || apiSecret != "" {
			sharedAuthReadErr = fmt.Errorf("both WORM_API_KEY and WORM_API_SECRET are required when using user-provided Worm API credentials")
			return
		}

		privateKey := strings.TrimSpace(os.Getenv("WORM_PRIVATE_KEY"))
		if privateKey == "" {
			sharedAuthReadErr = fmt.Errorf("WORM_PRIVATE_KEY is required for authenticated read integration tests when WORM_API_KEY/WORM_API_SECRET are not set")
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		publicClient, err := NewClient(Config{})
		if err != nil {
			sharedAuthReadErr = fmt.Errorf("NewClient: %w", err)
			return
		}
		creds, err := publicClient.CreateAPIKeyFromPrivateKey(ctx, privateKey)
		if err != nil {
			sharedAuthReadErr = fmt.Errorf("CreateAPIKeyFromPrivateKey: %w", err)
			return
		}
		if creds.APIKey == "" || creds.Secret == "" {
			sharedAuthReadErr = fmt.Errorf("CreateAPIKeyFromPrivateKey returned incomplete credentials")
			return
		}

		sharedAuthReadCreds = creds
		sharedAuthReadClient, sharedAuthReadErr = NewClient(Config{
			APIKey:    creds.APIKey,
			APISecret: creds.Secret,
		})
		if sharedAuthReadErr != nil {
			sharedAuthReadErr = fmt.Errorf("NewClient with temporary credentials: %w", sharedAuthReadErr)
			return
		}
	})
	return sharedAuthReadClient, sharedAuthReadErr
}

func cleanupSharedAuthReadIntegrationClient() {
	if sharedAuthReadCreds == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	key, err := revokeTemporaryAPIKey(ctx, sharedAuthReadCreds)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to cleanup shared temporary Worm API key: %v\n", err)
		return
	}
	fmt.Fprintf(os.Stderr, "cleanup revoked shared temporary Worm API key=%t revoked_at=%t\n", key.KeyID != "", key.RevokedAt != nil)
}

func isWormThrottleError(err error) bool {
	apiErr := wormAPIError(err)
	return apiErr != nil && (apiErr.Code == -17 || apiErr.Slug == "throttled")
}

func isWormValidationError(err error) bool {
	apiErr := wormAPIError(err)
	return apiErr != nil && (apiErr.Code == -11 || apiErr.Slug == "invalid_request_params")
}

func wormAPIError(err error) *Error {
	for err != nil {
		if apiErr, ok := err.(*Error); ok {
			return apiErr
		}
		unwrapped, ok := err.(interface{ Unwrap() error })
		if !ok {
			return nil
		}
		err = unwrapped.Unwrap()
	}
	return nil
}

func newAuthenticatedIntegrationClient(t *testing.T, creds *APIKeySecret) Client {
	t.Helper()
	client, err := NewClient(Config{
		APIKey:    creds.APIKey,
		APISecret: creds.Secret,
	})
	if err != nil {
		t.Fatalf("NewClient with temporary credentials: %v", err)
	}
	return client
}

func registerTemporaryAPIKeyCleanup(t *testing.T, creds *APIKeySecret) func(context.Context) *APIKey {
	t.Helper()
	revoked := false
	t.Cleanup(func() {
		if revoked {
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		key, err := revokeTemporaryAPIKey(ctx, creds)
		if err != nil {
			t.Logf("failed to cleanup temporary Worm API key: %v", err)
			return
		}
		t.Logf("cleanup revoked temporary Worm API key=%t revoked_at=%t", key.KeyID != "", key.RevokedAt != nil)
	})

	return func(ctx context.Context) *APIKey {
		t.Helper()
		key, err := revokeTemporaryAPIKey(ctx, creds)
		if err != nil {
			t.Fatalf("RevokeAPIKey: %v", err)
		}
		revoked = true
		return key
	}
}

func revokeTemporaryAPIKey(ctx context.Context, creds *APIKeySecret) (*APIKey, error) {
	client, err := NewClient(Config{
		APIKey:    creds.APIKey,
		APISecret: creds.Secret,
	})
	if err != nil {
		return nil, err
	}
	return client.RevokeAPIKey(ctx, creds.APIKey)
}

func newPublicIntegrationClient(t *testing.T) (context.Context, context.CancelFunc, Client) {
	t.Helper()
	if os.Getenv("WORM_INTEGRATION") != "1" {
		t.Skip("set WORM_INTEGRATION=1 to run real Worm API integration tests")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	client, err := NewClient(Config{})
	if err != nil {
		cancel()
		t.Fatalf("NewClient: %v", err)
	}
	return ctx, cancel, client
}

func findOpenIntegrationMarket(t *testing.T, ctx context.Context, client Client) MarketSummary {
	t.Helper()
	markets, err := client.ListMarkets(ctx, ListMarketsOptions{
		PageOptions: PageOptions{Limit: 1},
		State:       "open",
	})
	if err != nil {
		t.Fatalf("ListMarkets: %v", err)
	}
	if len(markets.Markets) == 0 {
		t.Fatal("ListMarkets returned no open markets")
	}
	validateMarketSummary(t, markets.Markets[0])
	return markets.Markets[0]
}

func findIntegrationEvent(t *testing.T, ctx context.Context, client Client) Event {
	t.Helper()
	events, err := client.ListEvents(ctx, ListEventsOptions{
		PageOptions: PageOptions{Limit: 1},
	})
	if err != nil {
		t.Fatalf("ListEvents: %v", err)
	}
	if len(events.Events) == 0 {
		t.Fatal("ListEvents returned no events")
	}
	validateEvent(t, events.Events[0])
	return events.Events[0]
}

func findIntegrationMarginPositionEstimate(t *testing.T, ctx context.Context, client Client) (MarketSummary, bool, *MarginPositionEstimate) {
	t.Helper()
	markets, err := client.ListMarkets(ctx, ListMarketsOptions{
		PageOptions: PageOptions{Limit: 20},
		State:       "open",
	})
	if err != nil {
		t.Fatalf("ListMarkets: %v", err)
	}
	isYesCandidates := []bool{true, false}
	leverage := 2.0
	attempted := make([]string, 0)
	for _, market := range markets.Markets {
		validateMarketSummary(t, market)
		if !market.MarginEnabled {
			continue
		}
		for _, isYes := range isYesCandidates {
			isYes := isYes
			attempted = append(attempted, fmt.Sprintf("%s/is_yes=%t", market.ConditionID, isYes))
			estimate, err := client.EstimateMarginPosition(ctx, EstimateMarginPositionOptions{
				MarketConditionID: market.ConditionID,
				Funds:             "10",
				IsYes:             &isYes,
				Leverage:          &leverage,
			})
			if err == nil {
				return market, isYes, estimate
			}
			if isWormValidationError(err) {
				t.Logf("EstimateMarginPosition candidate condition_id=%q is_yes=%t not suitable: %v", market.ConditionID, isYes, err)
				continue
			}
			t.Fatalf("EstimateMarginPosition(%s, is_yes=%t): %v", market.ConditionID, isYes, err)
		}
	}
	if len(attempted) == 0 {
		t.Skip("ListMarkets returned no open margin-enabled markets")
	}
	t.Skipf("no open margin-enabled market accepted EstimateMarginPosition test parameters; attempted %s", strings.Join(attempted, ", "))
	return MarketSummary{}, false, nil
}

func validateSearchSummary(t *testing.T, summary SearchSummary) {
	t.Helper()
	if summary.ConditionID == "" {
		t.Fatal("search summary has empty condition_id")
	}
	if summary.Title == "" {
		t.Fatal("search summary has empty title")
	}
	for i, market := range summary.Markets {
		if market.ConditionID == "" {
			t.Fatalf("search summary market[%d] has empty condition_id", i)
		}
		if market.Title == "" {
			t.Fatalf("search summary market[%d] has empty title", i)
		}
	}
}

func validateMarketSummary(t *testing.T, market MarketSummary) {
	t.Helper()
	if market.ConditionID == "" {
		t.Fatal("market has empty condition_id")
	}
	if market.Title == "" {
		t.Fatal("market has empty title")
	}
}

func validateEvent(t *testing.T, event Event) {
	t.Helper()
	if event.ConditionID == "" {
		t.Fatal("event has empty condition_id")
	}
	if event.Title == "" {
		t.Fatal("event has empty title")
	}
	for i, market := range event.Markets {
		if market.ConditionID == "" {
			t.Fatalf("event market[%d] has empty condition_id", i)
		}
		if market.Title == "" {
			t.Fatalf("event market[%d] has empty title", i)
		}
	}
}

func validateOrderBookLevels(t *testing.T, side string, levels []OrderBookLevel) {
	t.Helper()
	for i, level := range levels {
		if level.Price == "" {
			t.Fatalf("%s[%d] has empty price", side, i)
		}
		if level.TotalAmount == "" {
			t.Fatalf("%s[%d] has empty total_amount", side, i)
		}
	}
}

func validateIntegrationTrade(t *testing.T, i int, trade Trade) {
	t.Helper()
	if trade.Pubkey == "" {
		t.Fatalf("trade[%d] has empty pubkey", i)
	}
	if trade.MarketConditionID != nil && *trade.MarketConditionID == "" {
		t.Fatalf("trade[%d] has empty non-nil market_condition_id", i)
	}
	if trade.Amount == "" || trade.Price == "" || trade.State == "" {
		t.Fatalf("trade[%d] has empty required fields: %#v", i, trade)
	}
	if trade.Timestamp <= 0 {
		t.Fatalf("trade[%d] has timestamp=%d, want positive", i, trade.Timestamp)
	}
	if trade.Fee != nil && *trade.Fee == "" {
		t.Fatalf("trade[%d] has empty non-nil fee", i)
	}
	if trade.Order.Side == "" {
		t.Fatalf("trade[%d] has empty order side: %#v", i, trade.Order)
	}
	if trade.Order.Pubkey != nil && *trade.Order.Pubkey == "" {
		t.Fatalf("trade[%d] has empty non-nil order pubkey", i)
	}
	if trade.Order.Market.ConditionID == "" {
		t.Fatalf("trade[%d] has empty order market condition_id", i)
	}
	if trade.Order.Market.Title == "" {
		t.Fatalf("trade[%d] has empty order market title", i)
	}
}

func validateIntegrationOrder(t *testing.T, i int, order Order) {
	t.Helper()
	if order.Pubkey != nil && *order.Pubkey == "" {
		t.Fatalf("order[%d] has empty non-nil pubkey", i)
	}
	if order.Status != nil && *order.Status == "" {
		t.Fatalf("order[%d] has empty non-nil status", i)
	}
	if order.Side == "" {
		t.Fatalf("order[%d] has empty side: %#v", i, order)
	}
	if order.Price != nil && *order.Price == "" {
		t.Fatalf("order[%d] has empty non-nil price", i)
	}
	if order.Amount != nil && *order.Amount == "" {
		t.Fatalf("order[%d] has empty non-nil amount", i)
	}
	if order.Funds != nil && *order.Funds == "" {
		t.Fatalf("order[%d] has empty non-nil funds", i)
	}
	if order.Market.ConditionID == "" {
		t.Fatalf("order[%d] has empty market condition_id", i)
	}
	if order.Market.Title == "" {
		t.Fatalf("order[%d] has empty market title", i)
	}
	if order.Outcome.Text == "" {
		t.Fatalf("order[%d] has empty outcome text", i)
	}
}

func validateIntegrationAccountPnL(t *testing.T, pnl *AccountPnL) {
	t.Helper()
	if pnl.MarginPositionSettlementsPnL == "" ||
		pnl.MarginPositionsUnrealizedPnL == "" ||
		pnl.RedeemsFunds == "" ||
		pnl.ActiveAssetsValue == "" ||
		pnl.CreatorFees == "" ||
		pnl.TotalPnL == "" {
		t.Fatalf("GetAccountPnL returned empty required fields: %#v", pnl)
	}
}

func validateIntegrationAccountAsset(t *testing.T, i int, asset AccountAsset) {
	t.Helper()
	if asset.AssetKind == "" {
		t.Fatalf("asset[%d] has empty asset_kind", i)
	}
	if asset.Amounts.Total == "" || asset.Amounts.Locked == "" || asset.Amounts.Available == "" {
		t.Fatalf("asset[%d] has empty amounts fields: %#v", i, asset.Amounts)
	}
	if asset.Token.Symbol == "" {
		t.Fatalf("asset[%d] has empty token symbol", i)
	}
	if asset.Token.Address != nil && *asset.Token.Address == "" {
		t.Fatalf("asset[%d] has empty non-nil token address", i)
	}
	if asset.Value.USDT == "" || asset.Value.Basis == "" {
		t.Fatalf("asset[%d] has empty value fields: %#v", i, asset.Value)
	}
	validateMarketSummary(t, asset.Position.Market)
	if asset.Position.OutcomeText == "" || asset.Position.AvgTradePrice == "" {
		t.Fatalf("asset[%d] has empty position fields: %#v", i, asset.Position)
	}
	if asset.Created != nil && *asset.Created <= 0 {
		t.Fatalf("asset[%d] has created=%d, want positive timestamp", i, *asset.Created)
	}
}

func validateIntegrationMarginPositionEstimate(t *testing.T, estimate *MarginPositionEstimate) {
	t.Helper()
	if estimate.AveragePrice == "" ||
		estimate.TotalShares == "" ||
		estimate.TotalCost == "" ||
		estimate.BestAsk == "" ||
		estimate.WorstFillPrice == "" ||
		estimate.FeeAmount == "" ||
		estimate.UserFundsNeeded == "" {
		t.Fatalf("EstimateMarginPosition returned empty required fields: %#v", estimate)
	}
	if estimate.LiquidationPrice != nil && *estimate.LiquidationPrice == "" {
		t.Fatal("EstimateMarginPosition returned empty non-nil liquidation_price")
	}
}

func validateIntegrationPositionRequest(t *testing.T, i int, request PositionRequest) {
	t.Helper()
	if request.Pubkey == "" {
		t.Fatalf("position request[%d] has empty pubkey", i)
	}
	if request.Type == "" || request.State == "" || request.Leverage == "" || request.Funds == "" {
		t.Fatalf("position request[%d] has empty required fields: %#v", i, request)
	}
	if request.Message != nil && *request.Message == "" {
		t.Fatalf("position request[%d] has empty non-nil message", i)
	}
	if request.FundingTxID != nil && *request.FundingTxID == "" {
		t.Fatalf("position request[%d] has empty non-nil funding_txid", i)
	}
	if request.RefundTxID != nil && *request.RefundTxID == "" {
		t.Fatalf("position request[%d] has empty non-nil refund_txid", i)
	}
	if request.Market != nil {
		validateMarketSummary(t, *request.Market)
	}
	if request.Price != nil && *request.Price == "" {
		t.Fatalf("position request[%d] has empty non-nil price", i)
	}
	if request.Shares != nil && *request.Shares == "" {
		t.Fatalf("position request[%d] has empty non-nil shares", i)
	}
	if request.TakeProfitPrice != nil && *request.TakeProfitPrice == "" {
		t.Fatalf("position request[%d] has empty non-nil take_profit_price", i)
	}
	if request.StopLossPrice != nil && *request.StopLossPrice == "" {
		t.Fatalf("position request[%d] has empty non-nil stop_loss_price", i)
	}
	if request.Created != nil && *request.Created <= 0 {
		t.Fatalf("position request[%d] has created=%d, want positive timestamp", i, *request.Created)
	}
}

func validateIntegrationMarginPosition(t *testing.T, i int, position MarginPosition) {
	t.Helper()
	if position.Pubkey == "" {
		t.Fatalf("margin position[%d] has empty pubkey", i)
	}
	if position.PositionRequestPubkey != nil && *position.PositionRequestPubkey == "" {
		t.Fatalf("margin position[%d] has empty non-nil position_request_pubkey", i)
	}
	validateMarketSummary(t, position.Market)
	if position.Leverage == "" ||
		position.TotalShares == "" ||
		position.AvgEntryPrice == "" ||
		position.RealizedPnL == "" ||
		position.UserLiquidity == "" ||
		position.TotalLiquidity == "" ||
		position.LiquidationPrice == "" {
		t.Fatalf("margin position[%d] has empty required fields: %#v", i, position)
	}
	if position.ClosingPrice != nil && *position.ClosingPrice == "" {
		t.Fatalf("margin position[%d] has empty non-nil closing_price", i)
	}
	if position.UnrealizedPnL != nil && *position.UnrealizedPnL == "" {
		t.Fatalf("margin position[%d] has empty non-nil unrealized_pnl", i)
	}
	if position.TPSL != nil {
		validateIntegrationTPSL(t, i, *position.TPSL)
	}
	if position.Created != nil && *position.Created <= 0 {
		t.Fatalf("margin position[%d] has created=%d, want positive timestamp", i, *position.Created)
	}
}

func validateIntegrationTPSL(t *testing.T, i int, tpsl TPSL) {
	t.Helper()
	if tpsl.TakeProfitPrice != nil && *tpsl.TakeProfitPrice == "" {
		t.Fatalf("tp_sl[%d] has empty non-nil take_profit_price", i)
	}
	if tpsl.StopLossPrice != nil && *tpsl.StopLossPrice == "" {
		t.Fatalf("tp_sl[%d] has empty non-nil stop_loss_price", i)
	}
	if tpsl.State == "" {
		t.Fatalf("tp_sl[%d] has empty state", i)
	}
	if tpsl.TriggerType != nil && *tpsl.TriggerType == "" {
		t.Fatalf("tp_sl[%d] has empty non-nil trigger_type", i)
	}
	if tpsl.TriggeredPrice != nil && *tpsl.TriggeredPrice == "" {
		t.Fatalf("tp_sl[%d] has empty non-nil triggered_price", i)
	}
	if tpsl.TriggeredAt != nil && *tpsl.TriggeredAt <= 0 {
		t.Fatalf("tp_sl[%d] has triggered_at=%d, want positive timestamp", i, *tpsl.TriggeredAt)
	}
}

func validateIntegrationMarginSettlement(t *testing.T, i int, settlement MarginSettlement) {
	t.Helper()
	if settlement.PositionPubkey == "" {
		t.Fatalf("margin settlement[%d] has empty position_pubkey", i)
	}
	validateIntegrationMarginPosition(t, i, settlement.Position)
	if settlement.TotalPnL == "" || settlement.UserLiquidity == "" || settlement.State == "" {
		t.Fatalf("margin settlement[%d] has empty required fields: %#v", i, settlement)
	}
	if settlement.Created != nil && *settlement.Created <= 0 {
		t.Fatalf("margin settlement[%d] has created=%d, want positive timestamp", i, *settlement.Created)
	}
}

func validateIntegrationRedeem(t *testing.T, i int, redeem Redeem) {
	t.Helper()
	if redeem.Pubkey == "" {
		t.Fatalf("redeem[%d] has empty pubkey", i)
	}
	if redeem.MarketConditionID != nil && *redeem.MarketConditionID == "" {
		t.Fatalf("redeem[%d] has empty non-nil market_condition_id", i)
	}
	if redeem.State == "" ||
		redeem.Funds == "" ||
		redeem.OnchainFunds == "" ||
		redeem.YesShares == "" ||
		redeem.NoShares == "" {
		t.Fatalf("redeem[%d] has empty required fields: %#v", i, redeem)
	}
	if redeem.Message != nil && *redeem.Message == "" {
		t.Fatalf("redeem[%d] has empty non-nil message", i)
	}
	if redeem.Created != nil && *redeem.Created <= 0 {
		t.Fatalf("redeem[%d] has created=%d, want positive timestamp", i, *redeem.Created)
	}
}

func validateAuthChallenge(t *testing.T, challenge *AuthChallenge) {
	t.Helper()
	if challenge.Nonce == "" {
		t.Fatal("CreateAuthChallenge returned empty nonce")
	}
	if challenge.Message == "" {
		t.Fatal("CreateAuthChallenge returned empty message")
	}
	if challenge.ExpiresInSeconds <= 0 {
		t.Fatalf("CreateAuthChallenge returned expires_in_seconds=%d, want positive", challenge.ExpiresInSeconds)
	}
}

func validateAPIKeySecret(t *testing.T, creds *APIKeySecret) {
	t.Helper()
	if creds.APIKey == "" {
		t.Fatal("API key response returned empty api_key")
	}
	if creds.Secret == "" {
		t.Fatal("API key response returned empty secret")
	}
}

func validateRevokedAPIKey(t *testing.T, key *APIKey, wantKeyID string) {
	t.Helper()
	if key.KeyID == "" {
		t.Fatal("RevokeAPIKey returned empty key_id")
	}
	if key.KeyID != wantKeyID {
		t.Fatalf("RevokeAPIKey returned key_id=%q, want %q", key.KeyID, wantKeyID)
	}
	if key.RevokedAt == nil {
		t.Fatal("RevokeAPIKey returned nil revoked_at")
	}
}

func hasAPIKey(keys []APIKey, keyID string) bool {
	for _, key := range keys {
		if key.KeyID == keyID {
			return true
		}
	}
	return false
}

func positionRequestDraftIntegrationRequest(t *testing.T) CreatePositionRequestRequest {
	t.Helper()
	requestType := strings.ToUpper(requiredTrimmedIntegrationEnv(t, "WORM_POSITION_REQUEST_TYPE"))
	isYes := requiredBoolIntegrationEnv(t, "WORM_POSITION_REQUEST_IS_YES")
	leverage := requiredFloatIntegrationEnv(t, "WORM_POSITION_REQUEST_LEVERAGE")
	request := CreatePositionRequestRequest{
		Type:              requestType,
		MarketConditionID: requiredTrimmedIntegrationEnv(t, "WORM_POSITION_REQUEST_MARKET_CONDITION_ID"),
		IsYes:             &isYes,
		Leverage:          &leverage,
		TakeProfitPrice:   strings.TrimSpace(os.Getenv("WORM_POSITION_REQUEST_TAKE_PROFIT_PRICE")),
		StopLossPrice:     strings.TrimSpace(os.Getenv("WORM_POSITION_REQUEST_STOP_LOSS_PRICE")),
	}

	switch requestType {
	case "MARKET":
		request.Funds = requiredTrimmedIntegrationEnv(t, "WORM_POSITION_REQUEST_FUNDS")
	case "LIMIT":
		request.Price = requiredTrimmedIntegrationEnv(t, "WORM_POSITION_REQUEST_PRICE")
		request.Shares = requiredTrimmedIntegrationEnv(t, "WORM_POSITION_REQUEST_SHARES")
	default:
		t.Fatalf("WORM_POSITION_REQUEST_TYPE=%q, want MARKET or LIMIT", requestType)
	}

	return request
}

func validateCreatedPositionRequestDraft(t *testing.T, request *PositionRequest, wantType string) {
	t.Helper()
	if request.Pubkey == "" {
		t.Fatal("CreatePositionRequest returned empty pubkey")
	}
	if request.Type == "" {
		t.Fatal("CreatePositionRequest returned empty type")
	}
	if !strings.EqualFold(request.Type, wantType) {
		t.Fatalf("CreatePositionRequest returned type=%q, want %q", request.Type, wantType)
	}
	if request.State == "" {
		t.Fatal("CreatePositionRequest returned empty state")
	}
	if request.Message == nil || *request.Message == "" {
		t.Fatal("CreatePositionRequest returned empty message")
	}
	if request.Leverage == "" {
		t.Fatal("CreatePositionRequest returned empty leverage")
	}
	if request.Funds == "" {
		t.Fatal("CreatePositionRequest returned empty funds")
	}
	if request.Market != nil {
		validateMarketSummary(t, *request.Market)
	}
}

func registerPositionRequestDraftCleanup(t *testing.T, client Client, pubkey string) func(context.Context) *PositionRequest {
	t.Helper()
	cancelled := false
	t.Cleanup(func() {
		if cancelled {
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		request, err := client.CancelPositionRequest(ctx, pubkey)
		if err != nil {
			t.Logf("failed to cleanup temporary Worm position request draft: %v", err)
			return
		}
		t.Logf(
			"cleanup canceled temporary Worm position request draft pubkey_present=%t state=%q funding_txid_present=%t refund_txid_present=%t",
			request.Pubkey != "",
			request.State,
			request.FundingTxID != nil && *request.FundingTxID != "",
			request.RefundTxID != nil && *request.RefundTxID != "",
		)
	})

	return func(ctx context.Context) *PositionRequest {
		t.Helper()
		request, err := client.CancelPositionRequest(ctx, pubkey)
		if err != nil {
			t.Fatalf("CancelPositionRequest(%s): %v", pubkey, err)
		}
		cancelled = true
		return request
	}
}

func stringPtrLen(value *string) int {
	if value == nil {
		return 0
	}
	return len(*value)
}

func requiredIntegrationEnv(t *testing.T, name string) string {
	t.Helper()
	value := os.Getenv(name)
	if value == "" {
		t.Fatalf("%s is required for this integration test", name)
	}
	return value
}

func requiredTrimmedIntegrationEnv(t *testing.T, name string) string {
	t.Helper()
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		t.Fatalf("%s is required for this integration test", name)
	}
	return value
}

func requiredBoolIntegrationEnv(t *testing.T, name string) bool {
	t.Helper()
	value := requiredTrimmedIntegrationEnv(t, name)
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		t.Fatalf("%s=%q must be a boolean: %v", name, value, err)
	}
	return parsed
}

func requiredFloatIntegrationEnv(t *testing.T, name string) float64 {
	t.Helper()
	value := requiredTrimmedIntegrationEnv(t, name)
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		t.Fatalf("%s=%q must be a number: %v", name, value, err)
	}
	if parsed <= 0 {
		t.Fatalf("%s=%q must be greater than zero", name, value)
	}
	return parsed
}
