package commands

import (
	"strings"
	"testing"
)

func TestConfigRejectsInvalidBeforeOpeningDependencies(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*config)
	}{
		{"short internal identity", func(c *config) { c.InternalToken = "short" }},
		{"missing internal identity", func(c *config) { c.InternalToken = "" }},
		{"non HTTP node", func(c *config) { c.RPCURL = "file:///tmp/node" }},
		{"missing node host", func(c *config) { c.RPCURL = "https:" }},
		{"invalid port", func(c *config) { c.Port = 0 }},
		{"unbounded range", func(c *config) { c.RangeSize = 0 }},
		{"invalid concurrency", func(c *config) { c.Concurrency = 0 }},
		{"invalid rate", func(c *config) { c.RequestsPerSecond = 0 }},
		{"invalid timeout", func(c *config) { c.RequestTimeout = 0 }},
		{"invalid poll", func(c *config) { c.PollInterval = 0 }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := defaultConfig()
			c.InternalToken = strings.Repeat("s", 32)
			tc.change(&c)
			if c.validate() == nil {
				t.Fatal("invalid configuration accepted")
			}
		})
	}
}

func TestConfigDefaultsNeedOnlyOwnCredentials(t *testing.T) {
	c := defaultConfig()
	c.InternalToken = strings.Repeat("s", 32)
	if err := c.validate(); err != nil {
		t.Fatal(err)
	}
	if c.Port != 8112 || c.RangeSize != 4 || c.Concurrency != 1 || c.StartSlot != 0 {
		t.Fatalf("unexpected defaults: %+v", c)
	}
}
