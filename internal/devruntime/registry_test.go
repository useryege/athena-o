package devruntime

import (
	"slices"
	"testing"
)

func TestResolveSelectedServices(t *testing.T) {
	for _, name := range []string{"notification", "api-server", "trader-sync", "ui"} {
		specs, err := ResolveServices([]string{name})
		if err != nil || len(specs) != 1 || specs[0].Name != name {
			t.Fatalf("%s: %v %v", name, specs, err)
		}
		if name == "notification" || name == "ui" {
			for _, key := range []string{"ATHENA_TRADER_SYNC_CURSOR_HMAC_KEY", "ATHENA_TRADER_SYNC_INTERNAL_AUTH_TOKEN"} {
				if slices.Contains(specs[0].EnvironmentKeys, key) {
					t.Fatal(name, key)
				}
			}
		}
		if name == "ui" && len(specs[0].Infrastructure) != 0 {
			t.Fatal("UI acquired infrastructure")
		}
	}
	for _, names := range [][]string{nil, {"wallet"}, {"notification", "notification"}, {"api-server; touch /tmp/pwn"}} {
		if _, err := ResolveServices(names); err == nil {
			t.Fatalf("accepted %v", names)
		}
	}
	if _, err := resolveInfrastructure([]string{"a"}, map[string][]string{"a": {"b"}, "b": {"a"}}); err == nil {
		t.Fatal("accepted cycle")
	}
}

func TestAPIReceivesSolanaFacadeConfigurationWithoutCollectorCredentials(t *testing.T) {
	input := map[string]string{
		"ATHENA_SOLANA_DISCOVERY_SERVER_ADDRESS":      "127.0.0.1:38112",
		"ATHENA_SOLANA_DISCOVERY_INTERNAL_AUTH_TOKEN": "literal-solana-facade-token-123456789",
		"ATHENA_SOLANA_DISCOVERY_RPC_URL":             "https://private-node.example",
		"ATHENA_SOLANA_DISCOVERY_POSTGRES_DSN":        "postgres://collector/private",
	}
	for _, spec := range fullStackSpecs() {
		got := environmentMap(EnvironmentFor(input, spec.EnvironmentKeys))
		if spec.Name != "api-server" {
			if len(got) != 0 {
				t.Fatalf("%s received Solana credentials: %v", spec.Name, got)
			}
			continue
		}
		if got["ATHENA_SOLANA_DISCOVERY_SERVER_ADDRESS"] != "127.0.0.1:38112" ||
			got["ATHENA_SOLANA_DISCOVERY_INTERNAL_AUTH_TOKEN"] != "literal-solana-facade-token-123456789" || len(got) != 2 {
			t.Fatalf("API Solana environment: %v", got)
		}
	}
}
