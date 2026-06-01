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
	clobMarketsIntegrationMainGate = "POLYMARKET_CLOB_MARKETS_INTEGRATION"
	clobMarketsIntegrationLogGate  = "POLYMARKET_CLOB_MARKETS_INTEGRATION_LOG_RESPONSE"

	clobMarketsRebatesFallbackMaker = "0xFeA4cB3dD4ca7CefD3368653B7D6FF9BcDFca604"
)

type clobMarketsIntegrationSamples struct {
	conditionID string
	tokenID     string
}

func TestIntegrationCLOBMarkets(t *testing.T) {
	requireCLOBMarketsMainGate(t)

	client, err := NewCLOBClient(CLOBConfig{})
	if err != nil {
		t.Fatalf("NewCLOBClient: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	samples := discoverCLOBMarketsIntegrationSamples(t, ctx)
	if samples.conditionID == "" || samples.tokenID == "" {
		t.Skip("failed to discover CLOB market samples")
	}

	t.Run("GetMarketByToken", func(t *testing.T) {
		got, err := client.GetMarketByToken(ctx, samples.tokenID)
		if shouldSkipCLOBMarketsIntegrationError(t, err, "GetMarketByToken") {
			return
		}
		if err != nil {
			t.Fatalf("GetMarketByToken: %v", err)
		}
		logCLOBMarketsIntegrationResponse(t, "GetMarketByToken", got)
	})

	t.Run("GetCLOBMarketInfo", func(t *testing.T) {
		got, err := client.GetCLOBMarketInfo(ctx, samples.conditionID)
		if shouldSkipCLOBMarketsIntegrationError(t, err, "GetCLOBMarketInfo") {
			return
		}
		if err != nil {
			t.Fatalf("GetCLOBMarketInfo: %v", err)
		}
		logCLOBMarketsIntegrationResponse(t, "GetCLOBMarketInfo", got)
	})

	t.Run("GetPricesHistory", func(t *testing.T) {
		got, err := client.GetPricesHistory(ctx, GetCLOBPricesHistoryOptions{
			Market:   samples.tokenID,
			Interval: "1d",
		})
		if shouldSkipCLOBMarketsIntegrationError(t, err, "GetPricesHistory") {
			return
		}
		if err != nil {
			t.Fatalf("GetPricesHistory: %v", err)
		}
		logCLOBMarketsIntegrationResponse(t, "GetPricesHistory", got)
	})

	t.Run("GetBatchPricesHistory", func(t *testing.T) {
		got, err := client.GetBatchPricesHistory(ctx, CLOBBatchPricesHistoryRequest{
			Markets:  []string{samples.tokenID},
			Interval: "1d",
		})
		if shouldSkipCLOBMarketsIntegrationError(t, err, "GetBatchPricesHistory") {
			return
		}
		if err != nil {
			t.Fatalf("GetBatchPricesHistory: %v", err)
		}
		logCLOBMarketsIntegrationResponse(t, "GetBatchPricesHistory", got)
	})

	t.Run("ListSimplifiedMarkets", func(t *testing.T) {
		got, err := client.ListSimplifiedMarkets(ctx, "")
		if shouldSkipCLOBMarketsIntegrationError(t, err, "ListSimplifiedMarkets") {
			return
		}
		if err != nil {
			t.Fatalf("ListSimplifiedMarkets: %v", err)
		}
		logCLOBMarketsIntegrationResponse(t, "ListSimplifiedMarkets", got)
	})

	t.Run("ListSamplingMarkets", func(t *testing.T) {
		got, err := client.ListSamplingMarkets(ctx, "")
		if shouldSkipCLOBMarketsIntegrationError(t, err, "ListSamplingMarkets") {
			return
		}
		if err != nil {
			t.Fatalf("ListSamplingMarkets: %v", err)
		}
		logCLOBMarketsIntegrationResponse(t, "ListSamplingMarkets", got)
	})

	t.Run("ListSamplingSimplifiedMarkets", func(t *testing.T) {
		got, err := client.ListSamplingSimplifiedMarkets(ctx, "")
		if shouldSkipCLOBMarketsIntegrationError(t, err, "ListSamplingSimplifiedMarkets") {
			return
		}
		if err != nil {
			t.Fatalf("ListSamplingSimplifiedMarkets: %v", err)
		}
		logCLOBMarketsIntegrationResponse(t, "ListSamplingSimplifiedMarkets", got)
	})

	t.Run("Rebates/GetCurrentRebatedFees", func(t *testing.T) {
		maker := clobMarketsRebatesFallbackMaker
		date := time.Now().UTC().Format("2006-01-02")
		got, err := client.GetCurrentRebatedFees(ctx, GetCurrentRebatedFeesOptions{
			Date:         date,
			MakerAddress: maker,
		})
		if shouldSkipCLOBMarketsIntegrationError(t, err, "GetCurrentRebatedFees") {
			return
		}
		if err != nil {
			t.Fatalf("GetCurrentRebatedFees: %v", err)
		}
		logCLOBMarketsIntegrationResponse(t, "Rebates/GetCurrentRebatedFees", got)
	})
}

func discoverCLOBMarketsIntegrationSamples(t *testing.T, ctx context.Context) clobMarketsIntegrationSamples {
	t.Helper()
	one := 1
	samples := clobMarketsIntegrationSamples{}

	gamma, err := NewGammaClient(GammaConfig{})
	if err != nil {
		t.Fatalf("NewGammaClient for CLOB markets samples: %v", err)
	}
	markets, err := gamma.ListMarkets(ctx, ListMarketsOptions{ListOptions: ListOptions{Limit: &one}})
	if err != nil || len(markets) == 0 {
		t.Skipf("failed to discover gamma market sample: %v", err)
	}
	if markets[0].ConditionID != nil {
		samples.conditionID = strings.TrimSpace(*markets[0].ConditionID)
	}
	if markets[0].ClobTokenIDs != nil {
		var tokenIDs []string
		if err := json.Unmarshal([]byte(*markets[0].ClobTokenIDs), &tokenIDs); err == nil && len(tokenIDs) > 0 {
			samples.tokenID = strings.TrimSpace(tokenIDs[0])
		}
	}

	logCLOBMarketsIntegrationResponse(t, "Discovery/Samples", samples)
	return samples
}

func requireCLOBMarketsMainGate(t *testing.T) {
	t.Helper()
	if os.Getenv(clobMarketsIntegrationMainGate) != "1" {
		t.Skip("set POLYMARKET_CLOB_MARKETS_INTEGRATION=1 to run")
	}
}

func shouldSkipCLOBMarketsIntegrationError(t *testing.T, err error, endpoint string) bool {
	t.Helper()
	if err == nil {
		return false
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		return false
	}
	if apiErr.StatusCode == 400 || apiErr.StatusCode == 404 || apiErr.StatusCode == 422 || apiErr.StatusCode == 503 {
		t.Skipf("%s returned status %d (allowed skip for public variability)", endpoint, apiErr.StatusCode)
		return true
	}
	return false
}

func logCLOBMarketsIntegrationResponse(t *testing.T, endpoint string, payload any) {
	t.Helper()
	if os.Getenv(clobMarketsIntegrationLogGate) != "1" {
		return
	}
	b, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		t.Logf("%s response (marshal failed: %v): %+v", endpoint, err, payload)
		return
	}
	t.Logf("%s response:\n%s", endpoint, string(b))
}
