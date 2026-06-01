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
	clobMarketWSIntegrationMainGate = "POLYMARKET_CLOB_MARKET_WSS_INTEGRATION"
	clobMarketWSIntegrationLogGate  = "POLYMARKET_CLOB_MARKET_WSS_INTEGRATION_LOG_RESPONSE"
)

func TestIntegrationCLOBMarketWSS(t *testing.T) {
	if os.Getenv(clobMarketWSIntegrationMainGate) != "1" {
		t.Skip("set POLYMARKET_CLOB_MARKET_WSS_INTEGRATION=1 to run")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	samples := discoverCLOBIntegrationSamples(t, ctx)
	if strings.TrimSpace(samples.tokenID) == "" {
		t.Skip("no token sample discovered")
	}

	client, err := NewCLOBMarketWSClient(CLOBMarketWSConfig{})
	if err != nil {
		t.Fatalf("NewCLOBMarketWSClient: %v", err)
	}

	received := make(chan any, 1)
	runErr := make(chan error, 1)
	go func() {
		runErr <- client.Run(ctx, CLOBMarketWSSubscription{
			AssetIDs: []string{samples.tokenID},
		}, CLOBMarketWSHandler{
			OnBook: func(event CLOBMarketBookEvent) {
				select {
				case received <- event:
					cancel()
				default:
				}
			},
			OnPriceChange: func(event CLOBMarketPriceChangeEvent) {
				select {
				case received <- event:
					cancel()
				default:
				}
			},
			OnLastTrade: func(event CLOBMarketLastTradePriceEvent) {
				select {
				case received <- event:
					cancel()
				default:
				}
			},
			OnTickSize: func(event CLOBMarketTickSizeChangeEvent) {
				select {
				case received <- event:
					cancel()
				default:
				}
			},
			OnBestBidAsk: func(event CLOBMarketBestBidAskEvent) {
				select {
				case received <- event:
					cancel()
				default:
				}
			},
			OnNewMarket: func(event CLOBMarketNewMarketEvent) {
				select {
				case received <- event:
					cancel()
				default:
				}
			},
			OnMarketResolve: func(event CLOBMarketResolvedEvent) {
				select {
				case received <- event:
					cancel()
				default:
				}
			},
		})
	}()

	select {
	case event := <-received:
		logCLOBMarketWSIntegrationResponse(t, "CLOBMarketWS/Event", event)
	case <-ctx.Done():
		t.Fatal("timed out waiting for CLOB market ws event")
	}

	select {
	case err := <-runErr:
		if err != nil {
			t.Fatalf("Run: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not exit after cancel")
	}
}

func logCLOBMarketWSIntegrationResponse(t *testing.T, endpoint string, payload any) {
	t.Helper()
	if os.Getenv(clobMarketWSIntegrationLogGate) != "1" {
		return
	}
	b, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		t.Logf("%s response (marshal failed: %v): %+v", endpoint, err, payload)
		return
	}
	t.Logf("%s response:\n%s", endpoint, string(b))
}
