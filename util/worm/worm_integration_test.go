package worm

import (
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"os"
	"testing"
	"time"
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

func requiredIntegrationEnv(t *testing.T, name string) string {
	t.Helper()
	value := os.Getenv(name)
	if value == "" {
		t.Fatalf("%s is required for this integration test", name)
	}
	return value
}
