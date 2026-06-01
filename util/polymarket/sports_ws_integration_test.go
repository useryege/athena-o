package polymarket

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"
)

const (
	sportsWSIntegrationMainGate = "POLYMARKET_SPORTS_WSS_INTEGRATION"
	sportsWSIntegrationLogGate  = "POLYMARKET_SPORTS_WSS_INTEGRATION_LOG_RESPONSE"
	sportsWSIntegrationWindow   = 60 * time.Second
)

type sportsWSIntegrationRecord struct {
	Type      string         `json:"type"`
	Heartbeat string         `json:"heartbeat,omitempty"`
	Update    SportsWSUpdate `json:"update,omitempty"`
}

func TestIntegrationSportsWSS(t *testing.T) {
	if os.Getenv(sportsWSIntegrationMainGate) != "1" {
		t.Skip("set POLYMARKET_SPORTS_WSS_INTEGRATION=1 to run")
	}

	config := (SportsWSConfig{}).WithDefaults()
	dialer, dialerInfo, err := newIntegrationWSDialer(config.HandshakeTimeout)
	if err != nil {
		t.Fatalf("newIntegrationWSDialer: %v", err)
	}
	config.dial = dialer.DialContext

	if os.Getenv(sportsWSIntegrationLogGate) == "1" {
		t.Logf("SportsWSS integration proxy: %s", dialerInfo.Description)
	}

	client, err := NewSportsWSClient(config)
	if err != nil {
		t.Fatalf("NewSportsWSClient: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), sportsWSIntegrationWindow+20*time.Second)
	defer cancel()

	recordsCh := make(chan sportsWSIntegrationRecord, 128)
	errCh := make(chan error, 1)

	go func() {
		errCh <- client.Run(ctx, SportsWSHandler{
			OnUpdate: func(update SportsWSUpdate) {
				select {
				case recordsCh <- sportsWSIntegrationRecord{
					Type:   "update",
					Update: update,
				}:
				default:
				}
			},
			OnHeartbeat: func(msg string) {
				select {
				case recordsCh <- sportsWSIntegrationRecord{
					Type:      "heartbeat",
					Heartbeat: msg,
				}:
				default:
				}
			},
		})
	}()

	t.Logf("waiting for first SportsWSS message/heartbeat")

	records := make([]sportsWSIntegrationRecord, 0, 16)
	select {
	case rec := <-recordsCh:
		records = append(records, rec)
	case err := <-errCh:
		t.Fatalf("Run exited before receiving first message: %v", err)
	case <-ctx.Done():
		t.Fatal("timed out waiting for first sports ws update/heartbeat")
	}

	t.Logf("collecting SportsWSS messages for %s", sportsWSIntegrationWindow)
	windowTimer := time.NewTimer(sportsWSIntegrationWindow)
	defer windowTimer.Stop()

collectWindow:
	for {
		select {
		case rec := <-recordsCh:
			records = append(records, rec)
		case err := <-errCh:
			t.Fatalf("Run exited before collection window completed: %v", err)
		case <-windowTimer.C:
			break collectWindow
		case <-ctx.Done():
			t.Fatal("timed out while collecting sports ws update/heartbeat")
		}
	}

drainRecords:
	for {
		select {
		case rec := <-recordsCh:
			records = append(records, rec)
		default:
			break drainRecords
		}
	}

	for i, rec := range records {
		switch rec.Type {
		case "update":
			logSportsWSIntegrationResponse(t, fmt.Sprintf("SportsWSS/Update[%d]", i), rec.Update)
		case "heartbeat":
			logSportsWSIntegrationResponse(t, fmt.Sprintf("SportsWSS/Heartbeat[%d]", i), map[string]string{"message": rec.Heartbeat})
		default:
			logSportsWSIntegrationResponse(t, fmt.Sprintf("SportsWSS/Unknown[%d]", i), rec)
		}
	}

	cancel()
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
