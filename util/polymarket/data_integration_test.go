package polymarket

import (
	"context"
	"encoding/json"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

const (
	dataIntegrationMainGate = "POLYMARKET_DATA_INTEGRATION"
	dataIntegrationLogGate  = "POLYMARKET_DATA_INTEGRATION_LOG_RESPONSE"
)

type dataIntegrationSamples struct {
	user    string
	market  string
	eventID int64
}

func TestIntegrationData(t *testing.T) {
	requireDataMainGate(t)

	clientRaw, err := NewDataClient(DataConfig{})
	if err != nil {
		t.Fatalf("NewDataClient: %v", err)
	}
	client := clientRaw

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	samples := discoverDataIntegrationSamples(t, ctx)
	one := 1

	t.Run("Core/ListUserActivity", func(t *testing.T) {
		got, err := client.ListUserActivity(ctx, ListUserActivityOptions{
			DataListOptions: DataListOptions{Limit: &one},
			User:            samples.user,
		})
		if shouldSkipDataIntegrationError(t, err, "ListUserActivity") {
			return
		}
		if err != nil {
			t.Fatalf("ListUserActivity: %v", err)
		}
		logDataIntegrationResponse(t, "Core/ListUserActivity", got)
	})

	t.Run("Core/ListTrades", func(t *testing.T) {
		opts := ListDataTradesOptions{DataListOptions: DataListOptions{Limit: &one}}
		if samples.market != "" {
			opts.Market = []string{samples.market}
		}
		got, err := client.ListTrades(ctx, opts)
		if shouldSkipDataIntegrationError(t, err, "ListTrades") {
			return
		}
		if err != nil {
			t.Fatalf("ListTrades: %v", err)
		}
		logDataIntegrationResponse(t, "Core/ListTrades", got)
	})

	t.Run("Core/ListCurrentPositions", func(t *testing.T) {
		got, err := client.ListCurrentPositions(ctx, ListCurrentPositionsOptions{
			DataListOptions: DataListOptions{Limit: &one},
			User:            samples.user,
		})
		if shouldSkipDataIntegrationError(t, err, "ListCurrentPositions") {
			return
		}
		if err != nil {
			t.Fatalf("ListCurrentPositions: %v", err)
		}
		logDataIntegrationResponse(t, "Core/ListCurrentPositions", got)
	})

	t.Run("Core/ListClosedPositions", func(t *testing.T) {
		got, err := client.ListClosedPositions(ctx, ListClosedPositionsOptions{
			DataListOptions: DataListOptions{Limit: &one},
			User:            samples.user,
		})
		if shouldSkipDataIntegrationError(t, err, "ListClosedPositions") {
			return
		}
		if err != nil {
			t.Fatalf("ListClosedPositions: %v", err)
		}
		logDataIntegrationResponse(t, "Core/ListClosedPositions", got)
	})

	t.Run("Core/ListMarketPositions", func(t *testing.T) {
		if samples.market == "" {
			t.Skip("no discovered market sample")
		}
		got, err := client.ListMarketPositions(ctx, ListMarketPositionsOptions{
			DataListOptions: DataListOptions{Limit: &one},
			Market:          samples.market,
		})
		if shouldSkipDataIntegrationError(t, err, "ListMarketPositions") {
			return
		}
		if err != nil {
			t.Fatalf("ListMarketPositions: %v", err)
		}
		logDataIntegrationResponse(t, "Core/ListMarketPositions", got)
	})

	t.Run("Core/ListTopHolders", func(t *testing.T) {
		if samples.market == "" {
			t.Skip("no discovered market sample")
		}
		got, err := client.ListTopHolders(ctx, ListTopHoldersOptions{
			Limit:  &one,
			Market: []string{samples.market},
		})
		if shouldSkipDataIntegrationError(t, err, "ListTopHolders") {
			return
		}
		if err != nil {
			t.Fatalf("ListTopHolders: %v", err)
		}
		logDataIntegrationResponse(t, "Core/ListTopHolders", got)
	})

	t.Run("Core/GetTotalValueForUser", func(t *testing.T) {
		opts := GetTotalValueOptions{}
		if samples.market != "" {
			opts.Market = []string{samples.market}
		}
		got, err := client.GetTotalValueForUser(ctx, samples.user, opts)
		if shouldSkipDataIntegrationError(t, err, "GetTotalValueForUser") {
			return
		}
		if err != nil {
			t.Fatalf("GetTotalValueForUser: %v", err)
		}
		logDataIntegrationResponse(t, "Core/GetTotalValueForUser", got)
	})

	t.Run("Core/ListTraderLeaderboard", func(t *testing.T) {
		got, err := client.ListTraderLeaderboard(ctx, ListTraderLeaderboardOptions{
			Category:   "OVERALL",
			TimePeriod: "DAY",
			OrderBy:    "PNL",
			Limit:      &one,
		})
		if shouldSkipDataIntegrationError(t, err, "ListTraderLeaderboard") {
			return
		}
		if err != nil {
			t.Fatalf("ListTraderLeaderboard: %v", err)
		}
		logDataIntegrationResponse(t, "Core/ListTraderLeaderboard", got)
	})

	t.Run("Misc/GetOpenInterest", func(t *testing.T) {
		opts := GetOpenInterestOptions{}
		if samples.market != "" {
			opts.Market = []string{samples.market}
		}
		got, err := client.GetOpenInterest(ctx, opts)
		if shouldSkipDataIntegrationError(t, err, "GetOpenInterest") {
			return
		}
		if err != nil {
			t.Fatalf("GetOpenInterest: %v", err)
		}
		logDataIntegrationResponse(t, "Misc/GetOpenInterest", got)
	})

	t.Run("Misc/GetLiveVolumeByEventID", func(t *testing.T) {
		if samples.eventID == 0 {
			t.Skip("no discovered event id sample")
		}
		got, err := client.GetLiveVolumeByEventID(ctx, samples.eventID)
		if shouldSkipDataIntegrationError(t, err, "GetLiveVolumeByEventID") {
			return
		}
		if err != nil {
			t.Fatalf("GetLiveVolumeByEventID: %v", err)
		}
		logDataIntegrationResponse(t, "Misc/GetLiveVolumeByEventID", got)
	})

	t.Run("Misc/GetTotalMarketsTraded", func(t *testing.T) {
		got, err := client.GetTotalMarketsTraded(ctx, samples.user)
		if shouldSkipDataIntegrationError(t, err, "GetTotalMarketsTraded") {
			return
		}
		if err != nil {
			t.Fatalf("GetTotalMarketsTraded: %v", err)
		}
		logDataIntegrationResponse(t, "Misc/GetTotalMarketsTraded", got)
	})

	t.Run("Misc/DownloadAccountingSnapshot", func(t *testing.T) {
		got, err := client.DownloadAccountingSnapshot(ctx, samples.user)
		if shouldSkipDataIntegrationError(t, err, "DownloadAccountingSnapshot") {
			return
		}
		if err != nil {
			t.Fatalf("DownloadAccountingSnapshot: %v", err)
		}
		if len(got) == 0 {
			t.Log("DownloadAccountingSnapshot returned empty payload (allowed)")
		}
		logDataIntegrationResponse(t, "Misc/DownloadAccountingSnapshot", map[string]any{"bytes": len(got)})
	})

	t.Run("Builders/ListBuilderLeaderboard", func(t *testing.T) {
		got, err := client.ListBuilderLeaderboard(ctx, ListBuilderLeaderboardOptions{
			TimePeriod: "DAY",
			Limit:      &one,
		})
		if shouldSkipDataIntegrationError(t, err, "ListBuilderLeaderboard") {
			return
		}
		if err != nil {
			t.Fatalf("ListBuilderLeaderboard: %v", err)
		}
		logDataIntegrationResponse(t, "Builders/ListBuilderLeaderboard", got)
	})

	t.Run("Builders/ListBuilderDailyVolume", func(t *testing.T) {
		got, err := client.ListBuilderDailyVolume(ctx, ListBuilderDailyVolumeOptions{TimePeriod: "DAY"})
		if shouldSkipDataIntegrationError(t, err, "ListBuilderDailyVolume") {
			return
		}
		if err != nil {
			t.Fatalf("ListBuilderDailyVolume: %v", err)
		}
		logDataIntegrationResponse(t, "Builders/ListBuilderDailyVolume", got)
	})
}

func discoverDataIntegrationSamples(t *testing.T, ctx context.Context) dataIntegrationSamples {
	t.Helper()
	one := 1
	samples := dataIntegrationSamples{
		user: "0x0000000000000000000000000000000000000000",
	}

	gamma, err := NewGammaClient(GammaConfig{})
	if err != nil {
		t.Fatalf("NewGammaClient for data samples: %v", err)
	}

	markets, err := gamma.ListMarkets(ctx, ListMarketsOptions{ListOptions: ListOptions{Limit: &one}})
	if err == nil && len(markets) > 0 && markets[0].ConditionID != nil && strings.TrimSpace(*markets[0].ConditionID) != "" {
		samples.market = strings.TrimSpace(*markets[0].ConditionID)
	}

	events, err := gamma.ListEvents(ctx, ListEventsOptions{ListOptions: ListOptions{Limit: &one}})
	if err == nil && len(events) > 0 {
		eventID, parseErr := strconv.ParseInt(events[0].ID, 10, 64)
		if parseErr == nil {
			samples.eventID = eventID
		}
	}

	if samples.market != "" {
		dataClient, err := NewDataClient(DataConfig{})
		if err == nil {
			holders, err := dataClient.ListTopHolders(ctx, ListTopHoldersOptions{Limit: &one, Market: []string{samples.market}})
			if err == nil && len(holders) > 0 {
				for _, key := range []string{"proxyWallet", "user", "address", "owner"} {
					if val, ok := holders[0][key].(string); ok && strings.TrimSpace(val) != "" {
						samples.user = strings.TrimSpace(val)
						break
					}
				}
			}
		}
	}

	logDataIntegrationResponse(t, "Discovery/Samples", samples)
	return samples
}

func requireDataMainGate(t *testing.T) {
	t.Helper()
	if os.Getenv(dataIntegrationMainGate) != "1" {
		t.Skip("set POLYMARKET_DATA_INTEGRATION=1 to run")
	}
}

func shouldSkipDataIntegrationError(t *testing.T, err error, endpoint string) bool {
	t.Helper()
	if err == nil {
		return false
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		return false
	}
	if apiErr.StatusCode == 404 || apiErr.StatusCode == 422 || apiErr.StatusCode == 503 {
		t.Skipf("%s returned status %d (allowed skip for public data variability)", endpoint, apiErr.StatusCode)
		return true
	}
	return false
}

func logDataIntegrationResponse(t *testing.T, endpoint string, payload any) {
	t.Helper()
	if os.Getenv(dataIntegrationLogGate) != "1" {
		return
	}
	b, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		t.Logf("%s response (marshal failed: %v): %+v", endpoint, err, payload)
		return
	}
	t.Logf("%s response:\n%s", endpoint, string(b))
}
