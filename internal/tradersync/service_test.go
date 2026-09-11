package tradersync

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"testing"
)

func TestServiceCompositionRequiresExplicitDependencies(t *testing.T) {
	pool := &pgxpool.Pool{}
	collector := &Collector{}
	subs := &SubscriptionService{baseline: collector}
	deps := Dependencies{Pool: pool, Resolver: &TargetResolver{}, Subscriptions: subs, Collector: collector, Projector: &Projector{}, Directory: &DirectoryRefresher{}}
	cfg := Config{HTTPURL: "https://rpc.test", WebSocketURL: "wss://rpc.test", SiteURL: "https://athena.test", CursorHMACKey: "stable-test-cursor-key"}
	if _, e := NewService(cfg, deps); e != nil {
		t.Fatalf("valid explicit composition rejected: %v", e)
	}
	for _, tc := range []struct {
		name   string
		change func(*Config, *Dependencies)
	}{
		{"missing pool", func(_ *Config, d *Dependencies) { d.Pool = nil }},
		{"missing resolver", func(_ *Config, d *Dependencies) { d.Resolver = nil }},
		{"missing subscriptions", func(_ *Config, d *Dependencies) { d.Subscriptions = nil }},
		{"missing collector", func(_ *Config, d *Dependencies) { d.Collector = nil }},
		{"missing projector", func(_ *Config, d *Dependencies) { d.Projector = nil }},
		{"missing directory", func(_ *Config, d *Dependencies) { d.Directory = nil }},
		{"wrong registrar", func(_ *Config, d *Dependencies) { d.Subscriptions = &SubscriptionService{baseline: &Collector{}} }},
		{"missing HTTP", func(c *Config, _ *Dependencies) { c.HTTPURL = "" }},
		{"missing WSS", func(c *Config, _ *Dependencies) { c.WebSocketURL = "" }},
		{"missing key", func(c *Config, _ *Dependencies) { c.CursorHMACKey = "" }},
		{"missing site", func(c *Config, _ *Dependencies) { c.SiteURL = "" }},
		{"bad HTTP", func(c *Config, _ *Dependencies) { c.HTTPURL = "file:///rpc" }},
		{"bad WSS", func(c *Config, _ *Dependencies) { c.WebSocketURL = "https://rpc.test" }},
		{"bad proxy", func(c *Config, _ *Dependencies) { c.ProxyURL = "relative" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, d := cfg, deps
			tc.change(&c, &d)
			if _, e := NewService(c, d); e == nil {
				t.Fatal("invalid composition accepted")
			}
		})
	}
}
