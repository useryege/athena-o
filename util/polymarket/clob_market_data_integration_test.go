package polymarket

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"
)

const (
	clobIntegrationMainGate = "POLYMARKET_CLOB_MARKET_DATA_INTEGRATION"
	clobIntegrationLogGate  = "POLYMARKET_CLOB_MARKET_DATA_INTEGRATION_LOG_RESPONSE"
)

type clobIntegrationSamples struct {
	tokenID string
}

func TestIntegrationCLOBMarketData(t *testing.T) {
	requireCLOBMainGate(t)

	client, err := NewCLOBClient(CLOBConfig{})
	if err != nil {
		t.Fatalf("NewCLOBClient: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	samples := discoverCLOBIntegrationSamples(t, ctx)
	if strings.TrimSpace(samples.tokenID) == "" {
		t.Skip("no token sample discovered")
	}

	request := []CLOBBookRequest{{TokenID: samples.tokenID}}
	requestWithSide := []CLOBBookRequest{{TokenID: samples.tokenID, Side: "BUY"}}

	t.Run("GetOrderBook", func(t *testing.T) {
		got, err := client.GetOrderBook(ctx, samples.tokenID)
		if shouldSkipCLOBIntegrationError(t, err, "GetOrderBook") {
			return
		}
		if err != nil {
			t.Fatalf("GetOrderBook: %v", err)
		}
		logCLOBIntegrationResponse(t, "GetOrderBook", got)
	})

	t.Run("GetOrderBooks", func(t *testing.T) {
		got, err := client.GetOrderBooks(ctx, request)
		if shouldSkipCLOBIntegrationError(t, err, "GetOrderBooks") {
			return
		}
		if err != nil {
			t.Fatalf("GetOrderBooks: %v", err)
		}
		logCLOBIntegrationResponse(t, "GetOrderBooks", got)
	})

	t.Run("GetMidpointPrice", func(t *testing.T) {
		got, err := client.GetMidpointPrice(ctx, samples.tokenID)
		if shouldSkipCLOBIntegrationError(t, err, "GetMidpointPrice") {
			return
		}
		if err != nil {
			t.Fatalf("GetMidpointPrice: %v", err)
		}
		if strings.TrimSpace(got.MidPrice) == "" {
			t.Fatalf("GetMidpointPrice returned empty mid price")
		}
		logCLOBIntegrationResponse(t, "GetMidpointPrice", got)
	})

	t.Run("GetMidpointPrices", func(t *testing.T) {
		got, err := client.GetMidpointPrices(ctx, []string{samples.tokenID})
		if shouldSkipCLOBIntegrationError(t, err, "GetMidpointPrices") {
			return
		}
		if err != nil {
			t.Fatalf("GetMidpointPrices: %v", err)
		}
		logCLOBIntegrationResponse(t, "GetMidpointPrices", got)
	})

	t.Run("GetMidpointPricesByBody", func(t *testing.T) {
		got, err := client.GetMidpointPricesByBody(ctx, request)
		if shouldSkipCLOBIntegrationError(t, err, "GetMidpointPricesByBody") {
			return
		}
		if err != nil {
			t.Fatalf("GetMidpointPricesByBody: %v", err)
		}
		logCLOBIntegrationResponse(t, "GetMidpointPricesByBody", got)
	})

	t.Run("GetMarketPrice", func(t *testing.T) {
		got, err := client.GetMarketPrice(ctx, samples.tokenID, "BUY")
		if shouldSkipCLOBIntegrationError(t, err, "GetMarketPrice") {
			return
		}
		if err != nil {
			t.Fatalf("GetMarketPrice: %v", err)
		}
		logCLOBIntegrationResponse(t, "GetMarketPrice", got)
	})

	t.Run("GetMarketPrices", func(t *testing.T) {
		got, err := client.GetMarketPrices(ctx, []string{samples.tokenID}, []string{"BUY"})
		if shouldSkipCLOBIntegrationError(t, err, "GetMarketPrices") {
			return
		}
		if err != nil {
			t.Fatalf("GetMarketPrices: %v", err)
		}
		logCLOBIntegrationResponse(t, "GetMarketPrices", got)
	})

	t.Run("GetMarketPricesByBody", func(t *testing.T) {
		got, err := client.GetMarketPricesByBody(ctx, requestWithSide)
		if shouldSkipCLOBIntegrationError(t, err, "GetMarketPricesByBody") {
			return
		}
		if err != nil {
			t.Fatalf("GetMarketPricesByBody: %v", err)
		}
		logCLOBIntegrationResponse(t, "GetMarketPricesByBody", got)
	})

	t.Run("GetLastTradePrice", func(t *testing.T) {
		got, err := client.GetLastTradePrice(ctx, samples.tokenID)
		if shouldSkipCLOBIntegrationError(t, err, "GetLastTradePrice") {
			return
		}
		if err != nil {
			t.Fatalf("GetLastTradePrice: %v", err)
		}
		logCLOBIntegrationResponse(t, "GetLastTradePrice", got)
	})

	t.Run("GetLastTradePrices", func(t *testing.T) {
		got, err := client.GetLastTradePrices(ctx, []string{samples.tokenID})
		if shouldSkipCLOBIntegrationError(t, err, "GetLastTradePrices") {
			return
		}
		if err != nil {
			t.Fatalf("GetLastTradePrices: %v", err)
		}
		logCLOBIntegrationResponse(t, "GetLastTradePrices", got)
	})

	t.Run("GetLastTradePricesByBody", func(t *testing.T) {
		got, err := client.GetLastTradePricesByBody(ctx, request)
		if shouldSkipCLOBIntegrationError(t, err, "GetLastTradePricesByBody") {
			return
		}
		if err != nil {
			t.Fatalf("GetLastTradePricesByBody: %v", err)
		}
		logCLOBIntegrationResponse(t, "GetLastTradePricesByBody", got)
	})

	t.Run("GetSpread", func(t *testing.T) {
		got, err := client.GetSpread(ctx, samples.tokenID)
		if shouldSkipCLOBIntegrationError(t, err, "GetSpread") {
			return
		}
		if err != nil {
			t.Fatalf("GetSpread: %v", err)
		}
		logCLOBIntegrationResponse(t, "GetSpread", got)
	})

	t.Run("GetSpreads", func(t *testing.T) {
		got, err := client.GetSpreads(ctx, request)
		if shouldSkipCLOBIntegrationError(t, err, "GetSpreads") {
			return
		}
		if err != nil {
			t.Fatalf("GetSpreads: %v", err)
		}
		logCLOBIntegrationResponse(t, "GetSpreads", got)
	})

	t.Run("GetTickSize", func(t *testing.T) {
		got, err := client.GetTickSize(ctx, samples.tokenID)
		if shouldSkipCLOBIntegrationError(t, err, "GetTickSize") {
			return
		}
		if err != nil {
			t.Fatalf("GetTickSize: %v", err)
		}
		logCLOBIntegrationResponse(t, "GetTickSize", got)
	})

	t.Run("GetTickSizeByTokenID", func(t *testing.T) {
		got, err := client.GetTickSizeByTokenID(ctx, samples.tokenID)
		if shouldSkipCLOBIntegrationError(t, err, "GetTickSizeByTokenID") {
			return
		}
		if err != nil {
			t.Fatalf("GetTickSizeByTokenID: %v", err)
		}
		logCLOBIntegrationResponse(t, "GetTickSizeByTokenID", got)
	})

	t.Run("GetFeeRate", func(t *testing.T) {
		got, err := client.GetFeeRate(ctx, samples.tokenID)
		if shouldSkipCLOBIntegrationError(t, err, "GetFeeRate") {
			return
		}
		if err != nil {
			t.Fatalf("GetFeeRate: %v", err)
		}
		logCLOBIntegrationResponse(t, "GetFeeRate", got)
	})

	t.Run("GetFeeRateByTokenID", func(t *testing.T) {
		got, err := client.GetFeeRateByTokenID(ctx, samples.tokenID)
		if shouldSkipCLOBIntegrationError(t, err, "GetFeeRateByTokenID") {
			return
		}
		if err != nil {
			t.Fatalf("GetFeeRateByTokenID: %v", err)
		}
		logCLOBIntegrationResponse(t, "GetFeeRateByTokenID", got)
	})

	t.Run("GetServerTime", func(t *testing.T) {
		got, err := client.GetServerTime(ctx)
		if shouldSkipCLOBIntegrationError(t, err, "GetServerTime") {
			return
		}
		if err != nil {
			t.Fatalf("GetServerTime: %v", err)
		}
		if got.Unix <= 0 {
			t.Fatalf("GetServerTime returned non-positive unix time: %d", got.Unix)
		}
		logCLOBIntegrationResponse(t, "GetServerTime", got)
	})
}

func discoverCLOBIntegrationSamples(t *testing.T, ctx context.Context) clobIntegrationSamples {
	t.Helper()
	one := 1
	samples := clobIntegrationSamples{}

	gamma, err := NewGammaClient(GammaConfig{})
	if err != nil {
		t.Fatalf("NewGammaClient for clob samples: %v", err)
	}

	markets, err := gamma.ListMarkets(ctx, ListMarketsOptions{ListOptions: ListOptions{Limit: &one}})
	if err != nil {
		t.Skipf("failed to discover token from gamma markets: %v", err)
	}
	if len(markets) == 0 || markets[0].ClobTokenIDs == nil {
		t.Skip("gamma market sample missing clob token ids")
	}

	var tokenIDs []string
	if err := json.Unmarshal([]byte(*markets[0].ClobTokenIDs), &tokenIDs); err != nil {
		t.Skipf("failed to parse clob token ids: %v", err)
	}
	if len(tokenIDs) == 0 || strings.TrimSpace(tokenIDs[0]) == "" {
		t.Skip("no token id found in discovered market")
	}
	samples.tokenID = strings.TrimSpace(tokenIDs[0])
	logCLOBIntegrationResponse(t, "Discovery/Samples", samples)
	return samples
}

func requireCLOBMainGate(t *testing.T) {
	t.Helper()
	if os.Getenv(clobIntegrationMainGate) != "1" {
		t.Skip("set POLYMARKET_CLOB_MARKET_DATA_INTEGRATION=1 to run")
	}
}

func shouldSkipCLOBIntegrationError(t *testing.T, err error, endpoint string) bool {
	t.Helper()
	if err == nil {
		return false
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		return false
	}
	// Bulk query semantic methods are now backed by POST body requests and should
	// not silently skip on 400 payload errors.
	if endpoint == "GetMidpointPrices" || endpoint == "GetMarketPrices" || endpoint == "GetLastTradePrices" {
		if apiErr.StatusCode == 404 || apiErr.StatusCode == 422 || apiErr.StatusCode == 503 {
			t.Skipf("%s returned status %d (allowed skip for market-data variability)", endpoint, apiErr.StatusCode)
			return true
		}
		return false
	}
	if apiErr.StatusCode == 400 || apiErr.StatusCode == 404 || apiErr.StatusCode == 422 || apiErr.StatusCode == 503 {
		t.Skipf("%s returned status %d (allowed skip for market-data variability)", endpoint, apiErr.StatusCode)
		return true
	}
	return false
}

func logCLOBIntegrationResponse(t *testing.T, endpoint string, payload any) {
	t.Helper()
	if os.Getenv(clobIntegrationLogGate) != "1" {
		return
	}
	b, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		t.Logf("%s response (marshal failed: %v): %+v", endpoint, err, payload)
		return
	}
	t.Logf("%s response:\n%s", endpoint, string(b))
}
