package devruntime

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnvironmentScopesAndPersistentToken(t *testing.T) {
	k, _ := NewInstanceKey(t.TempDir(), "env")
	m := NewManager(k)
	base := map[string]string{"ATHENA_URL": "http://localhost:4000", "ATHENA_TRADER_SYNC_PROXY_URL": "", "ATHENA_TRADER_SYNC_CURSOR_HMAC_KEY": "cursor", "UNRELATED_SECRET": "secret"}
	specs, _ := ResolveServices([]string{"api-server"})
	a, err := m.PrepareEnvironment(base, specs, "managed")
	if err != nil {
		t.Fatal(err)
	}
	token := a["ATHENA_TRADER_SYNC_INTERNAL_AUTH_TOKEN"]
	if len(token) < 32 {
		t.Fatal("missing token")
	}
	b, err := m.PrepareEnvironment(base, specs, "managed")
	if err != nil || b["ATHENA_TRADER_SYNC_INTERNAL_AUTH_TOKEN"] != token {
		t.Fatal("token rotated", err)
	}
	info, err := os.Stat(filepath.Join(k.Dir(), "internal-token"))
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatal("secret mode", err)
	}
	for _, s := range []string{"notification", "ui"} {
		spec, _ := ResolveServices([]string{s})
		env, e := m.PrepareEnvironment(base, spec, "external")
		if e != nil && s == "ui" {
			t.Fatal(e)
		}
		for _, entry := range EnvironmentFor(env, spec[0].EnvironmentKeys) {
			if strings.Contains(entry, "TRADER_SYNC") || strings.Contains(entry, "UNRELATED") {
				t.Fatal(entry)
			}
		}
	}
	if _, err = m.PrepareEnvironment(base, specs, "external"); err == nil {
		t.Fatal("external missing credentials accepted")
	}
	base["ATHENA_TRADER_SYNC_INTERNAL_AUTH_TOKEN"] = strings.Repeat("x", 32)
	base["ATHENA_TRADER_SYNC_CURSOR_HMAC_KEY"] = strings.Repeat("x", 32)
	if _, err = m.PrepareEnvironment(base, specs, "managed"); err == nil {
		t.Fatal("same credentials accepted")
	}
}
func TestWSLProxyUndefinedVersusEmpty(t *testing.T) {
	for _, explicit := range []bool{false, true} {
		env := map[string]string{}
		if explicit {
			env["ATHENA_TRADER_SYNC_PROXY_URL"] = ""
		}
		calls := 0
		err := applyLocalProxy(env, true, func() (string, error) { calls++; return "172.22.1.1", nil })
		if err != nil {
			t.Fatal(err)
		}
		if explicit && (calls != 0 || env["ATHENA_TRADER_SYNC_PROXY_URL"] != "") {
			t.Fatal("empty overridden")
		}
		if !explicit && (calls != 1 || env["ATHENA_TRADER_SYNC_PROXY_URL"] != "http://172.22.1.1:10809") {
			t.Fatal(env)
		}
	}
}
func TestInvalidTraderSyncConfigFailsBeforeResourceAllocation(t *testing.T) {
	k, _ := NewInstanceKey(t.TempDir(), "invalid")
	s, _ := ResolveServices([]string{"trader-sync"})
	env := map[string]string{tokenKey: strings.Repeat("t", 32), "ATHENA_TRADER_SYNC_HTTP_URL": "bad", "ATHENA_TRADER_SYNC_WSS_URL": "ws://127.0.0.1:1", "ATHENA_TRADER_SYNC_CURSOR_HMAC_KEY": "cursor", "ATHENA_TRADER_SYNC_PROXY_URL": "", "ATHENA_URL": "http://localhost:4000"}
	if _, e := NewManager(k).PrepareEnvironment(env, s, "managed"); e == nil {
		t.Fatal("invalid source URL accepted")
	}
	if _, e := os.Stat(k.StatePath()); !os.IsNotExist(e) {
		t.Fatal("state allocated for invalid configuration")
	}
}
func TestSelectedEndpointsAreConnectedWithinInstance(t *testing.T) {
	k, _ := NewInstanceKey(t.TempDir(), "endpoints")
	specs, _ := ResolveServices([]string{"api-server", "notification", "ui"})
	env, e := NewManager(k).PrepareEnvironment(map[string]string{tokenKey: strings.Repeat("t", 32), "ATHENA_NOTIFICATION_PORT": "35431", "ATHENA_SERVER_PORT": "35432"}, specs, "managed")
	if e != nil {
		t.Fatal(e)
	}
	if env["ATHENA_NOTIFICATION_SERVER_ADDRESS"] != "127.0.0.1:35431" || env["ATHENA_API_URL"] != "http://127.0.0.1:35432" {
		t.Fatal("selected endpoints not connected")
	}
}
