package worm

import (
	"context"
	"os"
	"strconv"
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
	if price.Price == "" {
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
		price.Price,
		len(market.Rules),
		len(book.Bid),
		len(book.Ask),
	)
}

func TestIntegrationCreateOrderDraft(t *testing.T) {
	if os.Getenv("WORM_INTEGRATION") != "1" {
		t.Skip("set WORM_INTEGRATION=1 to run real Worm API integration tests")
	}
	if os.Getenv("WORM_ORDER_DRAFT_INTEGRATION") != "1" {
		t.Skip("set WORM_ORDER_DRAFT_INTEGRATION=1 to run real CreateOrderDraft integration test")
	}

	apiKey := requiredIntegrationEnv(t, "WORM_API_KEY")
	apiSecret := requiredIntegrationEnv(t, "WORM_API_SECRET")
	conditionID := requiredIntegrationEnv(t, "WORM_ORDER_DRAFT_MARKET_CONDITION_ID")
	isYesRaw := requiredIntegrationEnv(t, "WORM_ORDER_DRAFT_IS_YES")
	side := requiredIntegrationEnv(t, "WORM_ORDER_DRAFT_SIDE")
	orderType := requiredIntegrationEnv(t, "WORM_ORDER_DRAFT_ORDER_TYPE")
	funds := requiredIntegrationEnv(t, "WORM_ORDER_DRAFT_FUNDS")

	isYes, err := strconv.ParseBool(isYesRaw)
	if err != nil {
		t.Fatalf("WORM_ORDER_DRAFT_IS_YES = %q, want true or false", isYesRaw)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	client, err := NewClient(Config{
		APIKey:    apiKey,
		APISecret: apiSecret,
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	resp, err := client.CreateOrderDraft(ctx, CreateOrderDraftRequest{
		MarketConditionID: conditionID,
		IsYes:             isYes,
		Side:              side,
		OrderType:         orderType,
		Price:             os.Getenv("WORM_ORDER_DRAFT_PRICE"),
		Amount:            os.Getenv("WORM_ORDER_DRAFT_AMOUNT"),
		Funds:             funds,
	})
	if err != nil {
		t.Fatalf("CreateOrderDraft: %v", err)
	}
	if (resp.Pubkey == nil || *resp.Pubkey == "") && (resp.Message == nil || *resp.Message == "") {
		t.Fatalf("CreateOrderDraft returned no pubkey or message: %#v", resp)
	}

	t.Logf(
		"CreateOrderDraft returned pubkey=%t message=%t",
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
