package devruntime

import (
	"slices"
	"strings"
	"testing"
)

func TestResolveSelectedServices(t *testing.T) {
	for _, name := range []string{"notification", "api-server", "trader-sync", "operation-log", "ui"} {
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
	for _, names := range [][]string{nil, {"token"}, {"notification", "notification"}, {"api-server; touch /tmp/pwn"}} {
		if _, err := ResolveServices(names); err == nil {
			t.Fatalf("accepted %v", names)
		}
	}
	if _, err := resolveInfrastructure([]string{"a"}, map[string][]string{"a": {"b"}, "b": {"a"}}); err == nil {
		t.Fatal("accepted cycle")
	}
}

func TestOperationLogRegistryIsIndependentAndConfiguresItsAPIConsumer(t *testing.T) {
	specs, err := ResolveServices([]string{"operation-log"})
	if err != nil || len(specs) != 1 {
		t.Fatalf("operation-log: specs=%v err=%v", specs, err)
	}
	spec := specs[0]
	if spec.BuildPackage != "./cmd/athena-operation-log" || spec.Binary != "athena-operation-log" || spec.DefaultPort != "8124" || spec.Readiness != "grpc" || len(spec.Infrastructure) != 1 || spec.Infrastructure[0] != "postgres" {
		t.Fatalf("unexpected operation-log spec: %+v", spec)
	}
	if got := spec.Address(map[string]string{"ATHENA_OPERATION_LOG_LISTEN_ADDRESS": "127.0.0.1:9124"}); got != "127.0.0.1:9124" {
		t.Fatalf("operation-log address=%s", got)
	}
	for _, key := range []string{"ATHENA_OPERATION_LOG_INTERNAL_AUTH_TOKEN", "ATHENA_OPERATION_LOG_CURSOR_HMAC_KEY", "ATHENA_OPERATION_LOG_POSTGRES_DSN", "ATHENA_ACCOUNT_STATE_POSTGRES_DSN", "ATHENA_OPERATION_LOG_TLS_CA_FILE", "ATHENA_OPERATION_LOG_TLS_SERVER_NAME"} {
		if !slices.Contains(spec.EnvironmentKeys, key) {
			t.Fatalf("operation-log missing %s", key)
		}
	}
	api := serviceRegistry()["api-server"]
	for _, key := range []string{"ATHENA_OPERATION_LOG_SERVER_ADDRESS", "ATHENA_OPERATION_LOG_INTERNAL_AUTH_TOKEN", "ATHENA_OPERATION_LOG_POSTGRES_DSN"} {
		if !slices.Contains(api.EnvironmentKeys, key) {
			t.Fatalf("api missing operation-log consumer key %s", key)
		}
	}
	if slices.Contains(api.EnvironmentKeys, "ATHENA_OPERATION_LOG_CURSOR_HMAC_KEY") {
		t.Fatal("API received operation-log cursor key")
	}
}

func TestRegistryDoesNotExposeWormMarketsConfiguration(t *testing.T) {
	if _, err := ResolveServices([]string{"worm-markets"}); err == nil {
		t.Fatal("retired Worm Markets service remains selectable")
	}
	api := serviceRegistry()["api-server"]
	for _, key := range api.EnvironmentKeys {
		if strings.HasPrefix(key, "ATHENA_WORM_MARKETS_") {
			t.Fatalf("retired Worm Markets configuration remains: %s", key)
		}
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
		if spec.Name == "solana-discovery" {
			continue
		}
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
