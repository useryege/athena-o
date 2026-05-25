package worm

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestIntegrationPublicReadFlow(t *testing.T) {
	if os.Getenv("WORM_INTEGRATION") != "1" {
		t.Skip("set WORM_INTEGRATION=1 to run real Worm API integration test")
	}

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
		t.Fatal("ListMarkets returned no markets")
	}

	conditionID := markets.Markets[0].ConditionID
	if conditionID == "" {
		t.Fatal("ListMarkets returned empty condition_id")
	}

	market, err := client.GetMarket(ctx, conditionID)
	if err != nil {
		t.Fatalf("GetMarket(%s): %v", conditionID, err)
	}
	if market.ConditionID == "" {
		t.Fatalf("GetMarket(%s) returned empty condition_id", conditionID)
	}
	if market.Title == "" {
		t.Fatalf("GetMarket(%s) returned empty title", conditionID)
	}

	price, err := client.GetMarketPrice(ctx, conditionID, GetMarketPriceOptions{})
	if err != nil {
		t.Fatalf("GetMarketPrice(%s): %v", conditionID, err)
	}
	if price.Price == nil || *price.Price == "" {
		t.Fatalf("GetMarketPrice(%s) returned empty price", conditionID)
	}

	book, err := client.GetMarketOrderBook(ctx, conditionID, GetMarketOrderBookOptions{Depth: 5})
	if err != nil {
		t.Fatalf("GetMarketOrderBook(%s): %v", conditionID, err)
	}

	t.Logf(
		"market=%q state=%s price=%s rules=%d bid_levels=%d ask_levels=%d",
		market.Title,
		market.State,
		*price.Price,
		len(market.Rules),
		len(book.Bid),
		len(book.Ask),
	)
}

func TestIntegrationCreateAPIKeyFromPrivateKey(t *testing.T) {
	if os.Getenv("WORM_INTEGRATION") != "1" {
		t.Skip("set WORM_INTEGRATION=1 to run real Worm API integration tests")
	}
	if os.Getenv("WORM_API_KEY_BOOTSTRAP_INTEGRATION") != "1" {
		t.Skip("set WORM_API_KEY_BOOTSTRAP_INTEGRATION=1 to run real CreateAPIKeyFromPrivateKey integration test")
	}

	privateKey := requiredIntegrationEnv(t, "WORM_PRIVATE_KEY")

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	client, err := NewClient(Config{})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

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

	t.Logf("CreateAPIKeyFromPrivateKey returned api_key=%t secret=%t", creds.APIKey != "", creds.Secret != "")
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

func requiredIntegrationEnv(t *testing.T, name string) string {
	t.Helper()
	value := os.Getenv(name)
	if value == "" {
		t.Fatalf("%s is required for this integration test", name)
	}
	return value
}
