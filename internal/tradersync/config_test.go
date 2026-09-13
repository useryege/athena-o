package tradersync

import (
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestConfigLoadsOnlyDedicatedProxyAndStableResources(t *testing.T) {
	unsetConfigTestEnv(t, "ATHENA_TRADER_SYNC_CURSOR_HMAC_KEY_FILE")
	t.Setenv("ATHENA_ACCOUNT_STATE_POSTGRES_DSN", "postgres://fixture@localhost/test")
	t.Setenv("ATHENA_TRADER_SYNC_HTTP_URL", "https://rpc.test")
	t.Setenv("ATHENA_TRADER_SYNC_WSS_URL", "wss://rpc.test")
	t.Setenv("ATHENA_TRADER_SYNC_CURSOR_HMAC_KEY", "deployment-stable-key")
	t.Setenv("ATHENA_URL", "https://athena.test")
	t.Setenv("HTTP_PROXY", "http://environment.invalid:1")
	t.Setenv("ATHENA_TOKEN_POLYGON_PROXY_URL", "http://token.invalid:2")
	t.Setenv("ATHENA_TRADER_SYNC_PROXY_URL", "")
	c, e := LoadConfigFromEnv()
	if e != nil {
		t.Fatal(e)
	}
	if c.HTTPURL != "https://rpc.test" || c.WebSocketURL != "wss://rpc.test" || c.ProxyURL != "" || c.CursorHMACKey != "deployment-stable-key" || c.Projector.MaxInFlightSources != 100 || c.Projector.MetadataTimeout != 30*time.Second {
		t.Fatal("dedicated config/defaults not consumed", c)
	}
	t.Setenv("ATHENA_TRADER_SYNC_PROXY_URL", "http://explicit.test:10809")
	c, e = LoadConfigFromEnv()
	if e != nil || c.ProxyURL != "http://explicit.test:10809" {
		t.Fatal(c, e)
	}
	t.Setenv("ATHENA_TRADER_SYNC_MAX_IN_FLIGHT_SOURCES", "17")
	c, e = LoadConfigFromEnv()
	if e != nil || c.Projector.MaxInFlightSources != 17 {
		t.Fatal(c, e)
	}
	t.Setenv("ATHENA_TRADER_SYNC_MAX_IN_FLIGHT_SOURCES", "0")
	if _, e = LoadConfigFromEnv(); e == nil {
		t.Fatal("zero resource limit accepted")
	}
}

func unsetConfigTestEnv(t *testing.T, name string) {
	t.Helper()
	t.Setenv(name, "")
	require.NoError(t, os.Unsetenv(name))
}
func TestConfigCursorSecretFileSource(t *testing.T) {
	const name = "ATHENA_TRADER_SYNC_CURSOR_HMAC_KEY"
	t.Setenv("ATHENA_ACCOUNT_STATE_POSTGRES_DSN", "postgres://fixture@localhost/test")
	t.Setenv("ATHENA_TRADER_SYNC_HTTP_URL", "https://rpc.test")
	t.Setenv("ATHENA_TRADER_SYNC_WSS_URL", "wss://rpc.test")
	t.Setenv("ATHENA_TRADER_SYNC_PROXY_URL", "")
	t.Setenv("ATHENA_URL", "https://athena.test")
	unsetConfigTestEnv(t, name)
	unsetConfigTestEnv(t, name+"_FILE")
	for _, tc := range []struct{ name, contents, want string }{
		{"one trailing LF", "stable-fixture-key\n", "stable-fixture-key"},
		{"only one LF removed", "stable-fixture-key\n\n", "stable-fixture-key\n"},
		{"other whitespace preserved", " stable-fixture-key \n", " stable-fixture-key "},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "cursor-key")
			require.NoError(t, os.WriteFile(path, []byte(tc.contents), 0600))
			t.Setenv(name+"_FILE", path)
			cfg, err := LoadConfigFromEnv()
			require.NoError(t, err)
			require.Equal(t, tc.want, cfg.CursorHMACKey)
		})
	}
	t.Run("direct and file mutually exclusive including empty direct", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "private-cursor-path")
		require.NoError(t, os.WriteFile(path, []byte("file-secret-fixture"), 0600))
		t.Setenv(name+"_FILE", path)
		for _, direct := range []string{"direct-secret-fixture", ""} {
			t.Setenv(name, direct)
			_, err := LoadConfigFromEnv()
			require.ErrorContains(t, err, "mutually exclusive")
			require.NotContains(t, err.Error(), path)
			require.NotContains(t, err.Error(), "secret-fixture")
		}
	})
	t.Run("unreadable file does not disclose path", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "private-missing-key")
		t.Setenv(name+"_FILE", path)
		_, err := LoadConfigFromEnv()
		require.ErrorContains(t, err, "cannot read "+name+"_FILE")
		require.NotContains(t, err.Error(), path)
	})
	t.Run("empty file path rejected", func(t *testing.T) {
		t.Setenv(name+"_FILE", "")
		_, err := LoadConfigFromEnv()
		require.ErrorContains(t, err, "requires a path")
	})
	for _, contents := range []string{"", "\n"} {
		t.Run("empty resolved key "+contents, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "empty-key")
			require.NoError(t, os.WriteFile(path, []byte(contents), 0600))
			t.Setenv(name+"_FILE", path)
			_, err := LoadConfigFromEnv()
			require.ErrorContains(t, err, "stable cursor HMAC key required")
		})
	}
}
