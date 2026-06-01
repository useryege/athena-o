package commands

import "testing"

func TestSolidityNodeWSUseProxy_DefaultFalse(t *testing.T) {
	t.Setenv("ATHENA_SOLIDITY_NODE_WS_USE_PROXY", "")

	cmd := NewCommand()
	got, err := cmd.Flags().GetBool("node-ws-use-proxy")
	if err != nil {
		t.Fatalf("get node-ws-use-proxy flag: %v", err)
	}
	if got {
		t.Fatalf("node-ws-use-proxy default = %v, want false", got)
	}
}

func TestSolidityNodeWSUseProxy_EnvTrue(t *testing.T) {
	t.Setenv("ATHENA_SOLIDITY_NODE_WS_USE_PROXY", "true")

	cmd := NewCommand()
	got, err := cmd.Flags().GetBool("node-ws-use-proxy")
	if err != nil {
		t.Fatalf("get node-ws-use-proxy flag: %v", err)
	}
	if !got {
		t.Fatalf("node-ws-use-proxy from env = %v, want true", got)
	}
}

func TestSolidityNodeWSUseProxy_FlagOverridesEnv(t *testing.T) {
	t.Setenv("ATHENA_SOLIDITY_NODE_WS_USE_PROXY", "false")

	cmd := NewCommand()
	if err := cmd.Flags().Set("node-ws-use-proxy", "true"); err != nil {
		t.Fatalf("set node-ws-use-proxy flag: %v", err)
	}

	got, err := cmd.Flags().GetBool("node-ws-use-proxy")
	if err != nil {
		t.Fatalf("get node-ws-use-proxy flag: %v", err)
	}
	if !got {
		t.Fatalf("node-ws-use-proxy after flag override = %v, want true", got)
	}
}
