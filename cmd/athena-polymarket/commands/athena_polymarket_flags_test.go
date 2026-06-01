package commands

import "testing"

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
