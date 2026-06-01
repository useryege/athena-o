package polymarket

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"
)

const (
	sportsWSIntegrationMainGate = "POLYMARKET_SPORTS_WSS_INTEGRATION"
	sportsWSIntegrationLogGate  = "POLYMARKET_SPORTS_WSS_INTEGRATION_LOG_RESPONSE"
)

func TestIntegrationSportsWSS(t *testing.T) {
	if os.Getenv(sportsWSIntegrationMainGate) != "1" {
		t.Skip("set POLYMARKET_SPORTS_WSS_INTEGRATION=1 to run")
	}

	client, err := NewSportsWSClient(SportsWSConfig{})
	if err != nil {
		t.Fatalf("NewSportsWSClient: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	updateSeen := make(chan SportsWSUpdate, 1)
	heartbeatSeen := make(chan string, 1)
	errCh := make(chan error, 1)

	go func() {
		errCh <- client.Run(ctx, SportsWSHandler{
			OnUpdate: func(update SportsWSUpdate) {
				select {
				case updateSeen <- update:
					cancel()
				default:
				}
			},
			OnHeartbeat: func(msg string) {
				select {
				case heartbeatSeen <- msg:
					cancel()
				default:
				}
			},
		})
	}()

	select {
	case update := <-updateSeen:
		logSportsWSIntegrationResponse(t, "SportsWSS/Update", update)
	case heartbeat := <-heartbeatSeen:
		logSportsWSIntegrationResponse(t, "SportsWSS/Heartbeat", map[string]string{"message": heartbeat})
	case <-ctx.Done():
		t.Fatal("timed out waiting for sports ws update/heartbeat")
	}

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("Run: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not exit after cancel")
	}
}

func logSportsWSIntegrationResponse(t *testing.T, endpoint string, payload any) {
	t.Helper()
	if os.Getenv(sportsWSIntegrationLogGate) != "1" {
		return
	}
	b, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		t.Logf("%s response (marshal failed: %v): %+v", endpoint, err, payload)
		return
	}
	t.Logf("%s response:\n%s", endpoint, string(b))
}
