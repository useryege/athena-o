package devruntime

import (
	"strings"
	"testing"
)

func graphEnvironment() map[string]string {
	return map[string]string{
		"ATHENA_URL": "http://localhost:4000", "ATHENA_TRADER_SYNC_PROXY_URL": "", "ATHENA_TRADER_SYNC_HTTP_URL": "http://127.0.0.1:1234", "ATHENA_TRADER_SYNC_WSS_URL": "ws://127.0.0.1:1234", "ATHENA_TRADER_SYNC_CURSOR_HMAC_KEY": "stable-cursor",
		"ATHENA_ETHERSCAN_MANAGER_GATEWAY_ADDRS": "192.0.2.5:6776,192.0.2.4:6776,192.0.2.3:6776,192.0.2.2:6776,192.0.2.1:6776",
		"ETHERSCAN_GATEWAY_IPS":                  "192.0.2.1,192.0.2.2,192.0.2.3,192.0.2.4,192.0.2.5",
	}
}
func TestSelectedGraphOverridesEveryConsumerAndScopesCredentials(t *testing.T) {
	input := graphEnvironment()
	expected := map[string]string{"ATHENA_NOTIFICATION_SERVER_ADDRESS": "127.0.0.1:8086", "ATHENA_WALLET_SERVER_ADDRESS": "127.0.0.1:8088", "ATHENA_MARKET_RADAR_SERVER_ADDRESS": "127.0.0.1:8092", "ATHENA_MANAGED_OO_SERVER_ADDRESS": "127.0.0.1:8106", "ATHENA_PROFIT_SHARING_SERVER_ADDRESS": "127.0.0.1:8108", "ATHENA_WORM_TRADING_SERVER_ADDRESS": "127.0.0.1:8090", "ATHENA_SOLANA_DISCOVERY_SERVER_ADDRESS": "127.0.0.1:8112", "ATHENA_TRADER_SYNC_SERVER_ADDRESS": "127.0.0.1:8122", "ATHENA_MARKET_RADAR_NOTIFICATION_SERVER_ADDRESS": "127.0.0.1:8086", "ATHENA_MANAGED_OO_NOTIFICATION_SERVER_ADDRESS": "127.0.0.1:8086"}
	for key := range expected {
		input[key] = "remote.invalid:1"
	}
	input["ATHENA_SERVER_BASEHREF"] = "/athena"
	input["ATHENA_URL"] = "http://localhost:4000/athena"
	env, err := NewManager(testKey(t)).PrepareEnvironment(input, fullStackSpecs(), "managed")
	if err != nil {
		t.Fatal(err)
	}
	for key, want := range expected {
		if env[key] != want {
			t.Errorf("%s=%s, want %s", key, env[key], want)
		}
	}
	api := environmentMap(EnvironmentFor(env, serviceRegistry()["api-server"].EnvironmentKeys))
	if _, ok := api["ATHENA_WALLET_WORM_EXECUTION_SIGNER_TOKEN"]; ok {
		t.Fatal("API receives signer")
	}
	ui := environmentMap(EnvironmentFor(env, serviceRegistry()["ui"].EnvironmentKeys))
	if ui["ATHENA_SERVER_BASEHREF"] != "/athena/" || ui["ATHENA_API_URL"] != "http://127.0.0.1:8080" {
		t.Fatal("UI lost prefix or endpoint", ui)
	}
	if env["ATHENA_GOOGLE_OIDC_REDIRECT_URI"] != "http://localhost:4000/athena/auth/google/callback" {
		t.Fatal("callback mismatch")
	}
	if env["ATHENA_OPERATION_LOG_SERVER_ADDRESS"] != "127.0.0.1:8124" {
		t.Fatalf("operation-log consumer was not wired: %+v", env)
	}
	if env["ATHENA_OPERATION_LOG_INTERNAL_AUTH_TOKEN"] == "" || env["ATHENA_OPERATION_LOG_CURSOR_HMAC_KEY"] == "" || env["ATHENA_OPERATION_LOG_INTERNAL_AUTH_TOKEN"] == env["ATHENA_OPERATION_LOG_CURSOR_HMAC_KEY"] {
		t.Fatal("operation-log credentials missing or reused")
	}
	apiEnv := environmentMap(EnvironmentFor(env, serviceRegistry()["api-server"].EnvironmentKeys))
	if _, ok := apiEnv["ATHENA_OPERATION_LOG_CURSOR_HMAC_KEY"]; ok {
		t.Fatal("API received operation-log cursor key")
	}
	specs, _ := ResolveServices([]string{"wallet"})
	env, err = NewManager(testKey(t)).PrepareEnvironment(map[string]string{"ATHENA_MANAGED_OO_SERVER_ADDRESS": "external:8106"}, specs, "managed")
	if err != nil || env["ATHENA_MANAGED_OO_SERVER_ADDRESS"] != "external:8106" {
		t.Fatal("unselected endpoint changed", err)
	}
}
func TestPersistentCredentialsRejectConflictAndKeepTradingKey(t *testing.T) {
	m := NewManager(testKey(t))
	specs, _ := ResolveServices([]string{"wallet", "worm-trading"})
	input := map[string]string{"ATHENA_WORM_TRADING_CREDENTIAL_ENCRYPTION_KEY": strings.Repeat("a", 64)}
	a, err := m.PrepareEnvironment(input, specs, "managed")
	if err != nil {
		t.Fatal(err)
	}
	b, err := m.PrepareEnvironment(nil, specs, "managed")
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"ATHENA_WALLET_INTERNAL_AUTH_TOKEN", "ATHENA_WALLET_WORM_EXECUTION_SIGNER_TOKEN", "ATHENA_WALLET_ENCRYPTION_KEY", "ATHENA_WORM_TRADING_INTERNAL_AUTH_TOKEN", "ATHENA_WORM_TRADING_CREDENTIAL_ENCRYPTION_KEY"} {
		if a[key] == "" || a[key] != b[key] {
			t.Errorf("%s missing/rotated", key)
		}
	}
	if a["ATHENA_WALLET_INTERNAL_AUTH_TOKEN"] == a["ATHENA_WALLET_WORM_EXECUTION_SIGNER_TOKEN"] {
		t.Fatal("signer token not independent")
	}
	input["ATHENA_WORM_TRADING_CREDENTIAL_ENCRYPTION_KEY"] = strings.Repeat("b", 64)
	if _, err := m.PrepareEnvironment(input, specs, "managed"); err == nil {
		t.Fatal("persisted encryption identity silently changed")
	}
}
func TestGatewayConfigurationRejectsConflictingSets(t *testing.T) {
	specs, _ := ResolveServices([]string{"etherscan-manager"})
	input := graphEnvironment()
	input["ETHERSCAN_GATEWAY_IPS"] = "192.0.2.6,192.0.2.2,192.0.2.3,192.0.2.4,192.0.2.5"
	if _, err := NewManager(testKey(t)).PrepareEnvironment(input, specs, "managed"); err == nil {
		t.Fatal("conflicting gateway lists accepted")
	}
	input = graphEnvironment()
	input["ATHENA_ETHERSCAN_MANAGER_GATEWAY_ADDRS"] = "example.com:6776"
	if _, err := NewManager(testKey(t)).PrepareEnvironment(input, specs, "managed"); err == nil {
		t.Fatal("hostname accepted")
	}
	input = graphEnvironment()
	env, err := NewManager(testKey(t)).PrepareEnvironment(input, specs, "managed")
	if err != nil {
		t.Fatal(err)
	}
	if env["ATHENA_ETHERSCAN_MANAGER_GATEWAY_ADDRS"] != "192.0.2.1:6776,192.0.2.2:6776,192.0.2.3:6776,192.0.2.4:6776,192.0.2.5:6776" {
		t.Fatal("not normalized", env)
	}
}
func TestUIRejectsConflictingOriginAndCallback(t *testing.T) {
	specs, _ := ResolveServices([]string{"ui"})
	for _, input := range []map[string]string{{"ATHENA_UI_PORT": "4200", "ATHENA_URL": "http://localhost:4000"}, {"ATHENA_URL": "http://localhost:4000", "ATHENA_GOOGLE_OIDC_REDIRECT_URI": "http://other:4000/auth/google/callback"}} {
		if _, err := NewManager(testKey(t)).PrepareEnvironment(input, specs, "managed"); err == nil {
			t.Fatal("inconsistent UI origin accepted")
		}
	}
}

func TestRestartReusesRecordedEncryptionIdentity(t *testing.T) {
	m := NewManager(testKey(t))
	specs, _ := ResolveServices([]string{"worm-trading"})
	if err := m.SaveSecret("environment.json", []byte(`{"ATHENA_WORM_TRADING_CREDENTIAL_ENCRYPTION_KEY":"recorded-trading-encryption-key"}`)); err != nil {
		t.Fatal(err)
	}
	env, err := m.PrepareEnvironment(nil, specs, "managed")
	if err != nil {
		t.Fatal(err)
	}
	if env["ATHENA_WORM_TRADING_CREDENTIAL_ENCRYPTION_KEY"] != "recorded-trading-encryption-key" {
		t.Fatal("existing encrypted data key rotated")
	}
}
