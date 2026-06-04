package commands

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestApplicationNodeWSUseProxy_DefaultFalse(t *testing.T) {
	t.Setenv("ATHENA_APPLICATION_NODE_WS_USE_PROXY", "")

	cmd := NewCommand()
	got, err := cmd.Flags().GetBool("node-ws-use-proxy")
	if err != nil {
		t.Fatalf("get node-ws-use-proxy flag: %v", err)
	}
	if got {
		t.Fatalf("node-ws-use-proxy default = %v, want false", got)
	}
}

func TestApplicationNodeWSUseProxy_EnvTrue(t *testing.T) {
	t.Setenv("ATHENA_APPLICATION_NODE_WS_USE_PROXY", "true")

	cmd := NewCommand()
	got, err := cmd.Flags().GetBool("node-ws-use-proxy")
	if err != nil {
		t.Fatalf("get node-ws-use-proxy flag: %v", err)
	}
	if !got {
		t.Fatalf("node-ws-use-proxy from env = %v, want true", got)
	}
}

func TestApplicationNodeWSUseProxy_FlagOverridesEnv(t *testing.T) {
	t.Setenv("ATHENA_APPLICATION_NODE_WS_USE_PROXY", "false")

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

func TestApplicationBytecodeFlags(t *testing.T) {
	t.Setenv("ATHENA_APPLICATION_ETHERSCAN_API_BASE_URL", "https://example.invalid/etherscan")
	t.Setenv("ATHENA_APPLICATION_ETHERSCAN_API_KEY", "etherscan-key")
	t.Setenv("ATHENA_APPLICATION_DEEPSEEK_API_KEY", "deepseek-key")
	t.Setenv("ATHENA_APPLICATION_DEEPSEEK_BASE_URL", "https://example.invalid/deepseek")
	t.Setenv("ATHENA_APPLICATION_DEEPSEEK_MODEL", "deepseek-test")

	cmd := NewCommand()
	assertStringFlag(t, cmd, "etherscan-api-base-url", "https://example.invalid/etherscan")
	assertStringFlag(t, cmd, "etherscan-api-key", "etherscan-key")
	assertStringFlag(t, cmd, "deepseek-api-key", "deepseek-key")
	assertStringFlag(t, cmd, "deepseek-api-base-url", "https://example.invalid/deepseek")
	assertStringFlag(t, cmd, "deepseek-model", "deepseek-test")
	deprecatedFlag := "sol" + "idity-server-address"
	if flag := cmd.Flags().Lookup(deprecatedFlag); flag != nil {
		t.Fatalf("%s flag is still registered", deprecatedFlag)
	}
}

func assertStringFlag(t *testing.T, cmd *cobra.Command, name string, want string) {
	t.Helper()
	got, err := cmd.Flags().GetString(name)
	if err != nil {
		t.Fatalf("get %s flag: %v", name, err)
	}
	if got != want {
		t.Fatalf("%s = %q, want %q", name, got, want)
	}
}
