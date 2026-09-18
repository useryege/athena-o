package devruntime

import (
	"slices"
	"strings"
	"testing"
)

func TestElevenApplicationsResolveIndependently(t *testing.T) {
	names := []string{"api-server", "ui", "notification", "wallet", "etherscan-manager", "trader-sync", "solana-discovery", "market-radar", "managed-oo", "profit-sharing", "worm-trading", "operation-log"}
	if len(FullStackServices()) != 12 {
		t.Errorf("full stack has %d applications", len(FullStackServices()))
	}
	for _, name := range names {
		specs, err := ResolveServices([]string{name})
		if err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		if len(specs) != 1 {
			t.Fatalf("implicit application dependencies: %v", specs)
		}
		if name != "ui" && specs[0].BuildPackage != "./cmd/athena-"+map[string]string{"api-server": "server", "notification": "notification", "wallet": "wallet", "etherscan-manager": "etherscan-manager", "trader-sync": "trader-sync", "solana-discovery": "solana-discovery", "market-radar": "market-radar", "managed-oo": "managed-oo", "profit-sharing": "profit-sharing", "worm-trading": "worm-trading", "operation-log": "operation-log"}[name] {
			t.Errorf("%s not independently built: %s", name, specs[0].BuildPackage)
		}
	}
}
func TestIndependentStorageEnvironmentNeedsOnlySelectedDatabase(t *testing.T) {
	for _, tc := range []struct{ name, key string }{{"wallet", "ATHENA_WALLET_POSTGRES_DSN"}, {"managed-oo", "ATHENA_MANAGED_OO_POSTGRES_DSN"}, {"profit-sharing", "ATHENA_PROFIT_SHARING_POSTGRES_DSN"}} {
		t.Run(tc.name, func(t *testing.T) {
			specs, err := ResolveServices([]string{tc.name})
			if err != nil {
				t.Fatal(err)
			}
			env, err := NewManager(testKey(t)).PrepareEnvironment(map[string]string{tc.key: "postgres://localhost/borrowed"}, specs, "external")
			if err != nil {
				t.Fatal(err)
			}
			if env["ATHENA_ACCOUNT_STATE_POSTGRES_DSN"] != "" || slices.Contains(specs[0].EnvironmentKeys, "ATHENA_ACCOUNT_STATE_POSTGRES_DSN") {
				t.Fatal("unrelated account database required")
			}
			if _, err := NewManager(testKey(t)).PrepareEnvironment(nil, specs, "external"); err == nil {
				t.Fatal("missing selected DSN accepted")
			}
		})
	}
}

func TestOperationLogExternalUsesAccountDatabaseAndKeepsCursorPrivate(t *testing.T) {
	specs, err := ResolveServices([]string{"operation-log"})
	if err != nil {
		t.Fatal(err)
	}
	dsn := "postgres://localhost/athena"
	env, err := NewManager(testKey(t)).PrepareEnvironment(map[string]string{"ATHENA_ACCOUNT_STATE_POSTGRES_DSN": dsn}, specs, "external")
	if err != nil {
		t.Fatal(err)
	}
	if env["ATHENA_OPERATION_LOG_POSTGRES_DSN"] != dsn {
		t.Fatalf("operation-log DSN=%q, want account DSN", env["ATHENA_OPERATION_LOG_POSTGRES_DSN"])
	}
	if env["ATHENA_OPERATION_LOG_CURSOR_HMAC_KEY"] == "" || env["ATHENA_OPERATION_LOG_INTERNAL_AUTH_TOKEN"] == "" || env["ATHENA_OPERATION_LOG_CURSOR_HMAC_KEY"] == env["ATHENA_OPERATION_LOG_INTERNAL_AUTH_TOKEN"] {
		t.Fatal("operation-log credentials are not independent")
	}
	api, err := ResolveServices([]string{"api-server"})
	if err != nil {
		t.Fatal(err)
	}
	apiEnv, err := NewManager(testKey(t)).PrepareEnvironment(map[string]string{"ATHENA_ACCOUNT_STATE_POSTGRES_DSN": dsn, tokenKey: strings.Repeat("t", 32)}, api, "external")
	if err != nil {
		t.Fatal(err)
	}
	if apiEnv["ATHENA_OPERATION_LOG_POSTGRES_DSN"] != dsn || apiEnv["ATHENA_OPERATION_LOG_CURSOR_HMAC_KEY"] != "" {
		t.Fatalf("API operation-log environment leaked or omitted DSN: %+v", apiEnv)
	}
	if _, err := NewManager(testKey(t)).PrepareEnvironment(nil, specs, "external"); err == nil {
		t.Fatal("operation-log accepted missing account DSN")
	}
}
func TestProfitSharingUsesOnlyOfficialPort(t *testing.T) {
	env := map[string]string{"ATHENA_PROFIT_SHARING_PORT": "39001", "ATHENA_PROFIT_SHARING_LISTEN_PORT": "39002"}
	if got := serviceAddress("profit-sharing", env); got != "127.0.0.1:39002" {
		t.Fatal(got)
	}
}

func TestSelectedSchemasExcludeUnrelatedDatabases(t *testing.T) {
	for _, tc := range []struct {
		names []string
		want  []string
	}{{[]string{"wallet"}, []string{"wallet"}}, {[]string{"managed-oo"}, []string{"managed-oo"}}, {[]string{"profit-sharing"}, []string{"profit-sharing"}}, {[]string{"solana-discovery"}, []string{"account", "solana-discovery"}}, {[]string{"worm-trading"}, []string{"account", "worm-trading"}}, {[]string{"operation-log"}, []string{"account", "operation-log"}}, {[]string{"api-server"}, []string{"account", "operation-log"}}, {[]string{"market-radar", "ui", "etherscan-manager"}, nil}, {FullStackServices(), []string{"account", "solana-discovery", "wallet", "managed-oo", "profit-sharing", "worm-trading", "operation-log"}}} {
		specs, err := ResolveServices(tc.names)
		if err != nil {
			t.Fatal(err)
		}
		var got []string
		for _, s := range selectedSchemas(specs) {
			got = append(got, s.Name)
		}
		if !slices.Equal(got, tc.want) {
			t.Errorf("%v: %v", tc.names, got)
		}
	}
}
