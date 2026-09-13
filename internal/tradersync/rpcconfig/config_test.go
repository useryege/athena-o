package rpcconfig

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const testToken = "0123456789abcdef0123456789abcdef"

func lookup(values map[string]string) func(string) (string, bool) {
	return func(k string) (string, bool) { v, ok := values[k]; return v, ok }
}
func base() map[string]string {
	return map[string]string{"ATHENA_TRADER_SYNC_INTERNAL_AUTH_TOKEN": testToken, "ATHENA_TRADER_SYNC_GRPC_TRANSPORT": "loopback-insecure", "ATHENA_TRADER_SYNC_SERVER_ADDRESS": "127.0.0.1:8122"}
}

func TestSecretSourcesAndExactWhitespace(t *testing.T) {
	path := filepath.Join(t.TempDir(), "token")
	require.NoError(t, os.WriteFile(path, []byte(testToken+"\n"), 0600))
	got, err := ResolveSecret(lookup(map[string]string{"SECRET_FILE": path}), "SECRET")
	require.NoError(t, err)
	require.Equal(t, testToken, got)
	_, err = ResolveSecret(lookup(map[string]string{"SECRET": "", "SECRET_FILE": path}), "SECRET")
	require.Error(t, err)
	require.NoError(t, os.WriteFile(path, []byte(testToken+"\n\n"), 0600))
	got, err = ResolveSecret(lookup(map[string]string{"SECRET_FILE": path}), "SECRET")
	require.NoError(t, err)
	require.Equal(t, testToken+"\n", got)
	_, err = ResolveSecret(lookup(map[string]string{"SECRET_FILE": path + "missing"}), "SECRET")
	require.Error(t, err)
	_, err = ResolveSecret(lookup(map[string]string{"SECRET_FILE": ""}), "SECRET")
	require.Error(t, err)
}
func TestLoadConfigurationDefaultsAndIsolation(t *testing.T) {
	values := base()
	cfg, err := LoadServer(lookup(values))
	require.NoError(t, err)
	require.Equal(t, "127.0.0.1:8122", cfg.ListenAddress)
	require.Equal(t, 200*1024*1024, cfg.MaxMessageBytes)
	values["ATHENA_GRPC_MAX_SIZE_MB"] = "1"
	client, err := LoadClient(func(k string) (string, bool) {
		if strings.Contains(k, "HTTP") || strings.Contains(k, "WSS") || strings.Contains(k, "HMAC") || strings.Contains(k, "CERT") || strings.Contains(k, "KEY") {
			t.Fatalf("client read unrelated %s", k)
		}
		return lookup(values)(k)
	})
	require.NoError(t, err)
	require.Equal(t, 1024*1024, client.MaxMessageBytes)
	delete(values, "ATHENA_TRADER_SYNC_GRPC_TRANSPORT")
	client, err = LoadClient(lookup(values))
	require.NoError(t, err)
	require.Equal(t, "tls", client.Transport)
	_, err = LoadServer(lookup(values))
	require.Error(t, err, "TLS server must require a certificate")
}
func TestLoadConfigurationRejectsInvalidInput(t *testing.T) {
	for _, tc := range []struct{ name, key, value string }{
		{"short token", "ATHENA_TRADER_SYNC_INTERNAL_AUTH_TOKEN", "short"},
		{"space token", "ATHENA_TRADER_SYNC_INTERNAL_AUTH_TOKEN", testToken + " "},
		{"newline token", "ATHENA_TRADER_SYNC_INTERNAL_AUTH_TOKEN", testToken + "\n"},
		{"control token", "ATHENA_TRADER_SYNC_INTERNAL_AUTH_TOKEN", testToken + "\x00"},
		{"unicode whitespace", "ATHENA_TRADER_SYNC_INTERNAL_AUTH_TOKEN", testToken + "\u00a0"},
		{"unknown transport", "ATHENA_TRADER_SYNC_GRPC_TRANSPORT", "insecure"},
		{"empty transport", "ATHENA_TRADER_SYNC_GRPC_TRANSPORT", ""},
		{"zero size", "ATHENA_GRPC_MAX_SIZE_MB", "0"},
		{"negative size", "ATHENA_GRPC_MAX_SIZE_MB", "-1"},
		{"overflow size", "ATHENA_GRPC_MAX_SIZE_MB", "9223372036854775807"},
		{"fraction size", "ATHENA_GRPC_MAX_SIZE_MB", "1.5"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			values := base()
			values[tc.key] = tc.value
			_, err := LoadClient(lookup(values))
			require.Error(t, err)
			_, err = LoadServer(lookup(values))
			require.Error(t, err)
		})
	}
	for _, address := range []string{"", "127.0.0.1", "127.0.0.1:0", "127.0.0.1:65536", "127.0.0.1:http", ":8122", "0.0.0.0:8122", "[::]:8122", "192.0.2.1:8122", "example.com:8122", "http://localhost:8122", "localhost:8122/path"} {
		t.Run(address, func(t *testing.T) {
			values := base()
			values["ATHENA_TRADER_SYNC_SERVER_ADDRESS"] = address
			_, err := LoadClient(lookup(values))
			require.Error(t, err)
			values["ATHENA_TRADER_SYNC_LISTEN_ADDRESS"] = address
			_, err = LoadServer(lookup(values))
			require.Error(t, err)
		})
	}
	for _, address := range []string{"127.0.0.2:8122", "[::1]:8122", "localhost:8122"} {
		values := base()
		values["ATHENA_TRADER_SYNC_SERVER_ADDRESS"] = address
		_, err := LoadClient(lookup(values))
		require.NoError(t, err)
	}
}
func TestTokenFileUsesCredentialValidation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "token")
	values := base()
	delete(values, "ATHENA_TRADER_SYNC_INTERNAL_AUTH_TOKEN")
	values["ATHENA_TRADER_SYNC_INTERNAL_AUTH_TOKEN_FILE"] = path
	for _, content := range []string{testToken, testToken + "\n"} {
		require.NoError(t, os.WriteFile(path, []byte(content), 0600))
		cfg, err := LoadClient(lookup(values))
		require.NoError(t, err)
		require.Equal(t, testToken, cfg.Token)
	}
	for _, content := range []string{"", "short\n", testToken + "\n\n", testToken + " \n"} {
		require.NoError(t, os.WriteFile(path, []byte(content), 0600))
		_, err := LoadClient(lookup(values))
		require.Error(t, err)
	}
	values["ATHENA_TRADER_SYNC_INTERNAL_AUTH_TOKEN"] = ""
	_, err := LoadClient(lookup(values))
	require.Error(t, err)
}
func TestMethodBudgetsAreExplicit(t *testing.T) {
	prefix := "/tradersync.internal.v1.TraderSyncService/"
	for _, name := range []string{"GetActivity", "GetSubscription", "ListSubscriptions", "ListActivities", "ListSubscriptionHistory", "GetSummaryBatch", "ListSummaryParts", "GetSubscriptionSummary", "ListSubscriptionSummaries", "GetTraderSyncRuntimeStatus"} {
		got, err := MethodBudget(prefix + name)
		require.NoError(t, err)
		require.Equal(t, 5*time.Second, got)
	}
	for _, name := range []string{"ResolveTarget", "CreateSubscription", "PauseSubscription", "ResumeSubscription", "CancelSubscription", "UpdateTargetNote"} {
		got, err := MethodBudget(prefix + name)
		require.NoError(t, err)
		require.Equal(t, 15*time.Second, got)
	}
	for _, method := range []string{"", "GetActivity", prefix + "Unknown", "/other/GetActivity"} {
		_, err := MethodBudget(method)
		require.Error(t, err)
	}
}

func TestServerPinsExplicitLocalhostBeforeListening(t *testing.T) {
	values := base()
	values["ATHENA_TRADER_SYNC_LISTEN_ADDRESS"] = "localhost:8122"
	cfg, err := LoadServer(lookup(values))
	require.NoError(t, err)
	require.Equal(t, "127.0.0.1:8122", cfg.ListenAddress)
}
func TestTLSConfigurationRejectsUnreadableAndInvalidMaterial(t *testing.T) {
	values := base()
	values["ATHENA_TRADER_SYNC_GRPC_TRANSPORT"] = "tls"
	path := filepath.Join(t.TempDir(), "invalid.pem")
	require.NoError(t, os.WriteFile(path, []byte("not a certificate"), 0600))
	for _, file := range []string{path, path + "missing"} {
		values["ATHENA_TRADER_SYNC_TLS_CA_FILE"] = file
		_, err := LoadClient(lookup(values))
		require.Error(t, err)
	}
	values["ATHENA_TRADER_SYNC_TLS_CERT_FILE"] = path
	values["ATHENA_TRADER_SYNC_TLS_KEY_FILE"] = path
	_, err := LoadServer(lookup(values))
	require.Error(t, err)
}
