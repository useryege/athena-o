package tradersync

import (
	"testing"
	"time"
)

func TestConfigLoadsOnlyDedicatedProxyAndStableResources(t *testing.T) {
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
