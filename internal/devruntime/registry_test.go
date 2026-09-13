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
