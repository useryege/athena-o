package commands

import (
	"testing"
	"time"
)

func TestPolymarketWSUseProxy_DefaultTrue(t *testing.T) {
	t.Setenv("ATHENA_POLYMARKET_WS_USE_PROXY", "")

	cmd := NewCommand()
	got, err := cmd.Flags().GetBool("ws-use-proxy")
	if err != nil {
		t.Fatalf("get ws-use-proxy flag: %v", err)
	}
	if !got {
		t.Fatalf("ws-use-proxy default = %v, want true", got)
	}
}

func TestPolymarketWSUseProxy_EnvFalse(t *testing.T) {
	t.Setenv("ATHENA_POLYMARKET_WS_USE_PROXY", "false")

	cmd := NewCommand()
	got, err := cmd.Flags().GetBool("ws-use-proxy")
	if err != nil {
		t.Fatalf("get ws-use-proxy flag: %v", err)
	}
	if got {
		t.Fatalf("ws-use-proxy from env = %v, want false", got)
	}
}

func TestPolymarketWSUseProxy_FlagOverridesEnv(t *testing.T) {
	t.Setenv("ATHENA_POLYMARKET_WS_USE_PROXY", "false")

	cmd := NewCommand()
	if err := cmd.Flags().Set("ws-use-proxy", "true"); err != nil {
		t.Fatalf("set ws-use-proxy flag: %v", err)
	}

	got, err := cmd.Flags().GetBool("ws-use-proxy")
	if err != nil {
		t.Fatalf("get ws-use-proxy flag: %v", err)
	}
	if !got {
		t.Fatalf("ws-use-proxy after flag override = %v, want true", got)
	}
}

func TestPolymarketNotification_DefaultDisabled(t *testing.T) {
	t.Setenv("ATHENA_POLYMARKET_NOTIFICATION_ENABLED", "")
	t.Setenv("ATHENA_POLYMARKET_NOTIFICATION_SERVER_ADDRESS", "")

	cmd := NewCommand()
	enabled, err := cmd.Flags().GetBool("notification-enabled")
	if err != nil {
		t.Fatalf("get notification-enabled flag: %v", err)
	}
	if enabled {
		t.Fatalf("notification-enabled default = %v, want false", enabled)
	}
	address, err := cmd.Flags().GetString("notification-server-address")
	if err != nil {
		t.Fatalf("get notification-server-address flag: %v", err)
	}
	if address != "localhost:8086" {
		t.Fatalf("notification-server-address = %q, want localhost:8086", address)
	}
}

func TestPolymarketNotification_EnvAndDefaultMoverAlertConfig(t *testing.T) {
	t.Setenv("ATHENA_POLYMARKET_NOTIFICATION_ENABLED", "true")
	t.Setenv("ATHENA_POLYMARKET_NOTIFICATION_SERVER_ADDRESS", "notification:8086")

	cmd := NewCommand()
	enabled, err := cmd.Flags().GetBool("notification-enabled")
	if err != nil {
		t.Fatalf("get notification-enabled flag: %v", err)
	}
	if !enabled {
		t.Fatalf("notification-enabled from env = %v, want true", enabled)
	}
	address, err := cmd.Flags().GetString("notification-server-address")
	if err != nil {
		t.Fatalf("get notification-server-address flag: %v", err)
	}
	if address != "notification:8086" {
		t.Fatalf("notification-server-address = %q, want env address", address)
	}
	warningScore, err := cmd.Flags().GetFloat64("mover-alert-warning-score")
	if err != nil {
		t.Fatalf("get mover-alert-warning-score flag: %v", err)
	}
	criticalScore, err := cmd.Flags().GetFloat64("mover-alert-critical-score")
	if err != nil {
		t.Fatalf("get mover-alert-critical-score flag: %v", err)
	}
	cooldown, err := cmd.Flags().GetDuration("mover-alert-cooldown")
	if err != nil {
		t.Fatalf("get mover-alert-cooldown flag: %v", err)
	}
	maxPerRefresh, err := cmd.Flags().GetInt("mover-alert-max-per-refresh")
	if err != nil {
		t.Fatalf("get mover-alert-max-per-refresh flag: %v", err)
	}
	if warningScore != 6 || criticalScore != 12 || cooldown != 15*time.Minute || maxPerRefresh != 3 {
		t.Fatalf("mover alert defaults = warning %.2f critical %.2f cooldown %s max %d, want 6/12/15m/3", warningScore, criticalScore, cooldown, maxPerRefresh)
	}
}
