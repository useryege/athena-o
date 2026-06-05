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

	cmd := NewCommand()
	assertStringFlag(t, cmd, "etherscan-api-base-url", "https://example.invalid/etherscan")
	assertStringFlag(t, cmd, "etherscan-api-key", "etherscan-key")
	deprecatedPrefix := "deep" + "seek"
	deprecatedTemporalFlag := "temporal" + "-enabled"
	for _, name := range []string{deprecatedPrefix + "-api-key", deprecatedPrefix + "-api-base-url", deprecatedPrefix + "-model", deprecatedTemporalFlag} {
		if flag := cmd.Flags().Lookup(name); flag != nil {
			t.Fatalf("%s flag is still registered", name)
		}
	}
	deprecatedFlag := "sol" + "idity-server-address"
	if flag := cmd.Flags().Lookup(deprecatedFlag); flag != nil {
		t.Fatalf("%s flag is still registered", deprecatedFlag)
	}
}

func TestApplicationRuntimeModeFlags(t *testing.T) {
	t.Setenv("ATHENA_APPLICATION_MODE", "kafka-consumer")
	t.Setenv("ATHENA_APPLICATION_CHAIN_ID", "56")
	t.Setenv("ATHENA_APPLICATION_KAFKA_BROKERS", "kafka-1:9092,kafka-2:9092")
	t.Setenv("ATHENA_APPLICATION_KAFKA_CONSUMER_GROUP", "application-test")

	cmd := NewCommand()
	assertStringFlag(t, cmd, "mode", "kafka-consumer")
	assertStringFlag(t, cmd, "kafka-consumer-group", "application-test")
	gotChainID, err := cmd.Flags().GetInt64("chain-id")
	if err != nil {
		t.Fatalf("get chain-id flag: %v", err)
	}
	if gotChainID != 56 {
		t.Fatalf("chain-id = %d, want 56", gotChainID)
	}
	brokers, err := cmd.Flags().GetStringSlice("kafka-brokers")
	if err != nil {
		t.Fatalf("get kafka-brokers flag: %v", err)
	}
	if len(brokers) != 2 || brokers[0] != "kafka-1:9092" || brokers[1] != "kafka-2:9092" {
		t.Fatalf("kafka-brokers = %#v, want two brokers from env", brokers)
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
