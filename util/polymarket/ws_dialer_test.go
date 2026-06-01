package polymarket

import (
	"testing"
	"time"
)

func TestNewWSDialer_NoProxy(t *testing.T) {
	dialer := newWSDialer(3*time.Second, false)
	if dialer.Proxy != nil {
		t.Fatal("proxy resolver should be nil when proxy is disabled")
	}
}

func TestNewWSDialer_UseProxyResolverConfigured(t *testing.T) {
	dialer := newWSDialer(3*time.Second, true)
	if dialer.Proxy == nil {
		t.Fatal("proxy resolver should not be nil when proxy is enabled")
	}
}

func TestSportsWSConfigUseProxyPropagation(t *testing.T) {
	raw, err := NewSportsWSClient(SportsWSConfig{UseProxy: true})
	if err != nil {
		t.Fatalf("NewSportsWSClient: %v", err)
	}
	impl, ok := raw.(*sportsWSClientImpl)
	if !ok {
		t.Fatalf("client type = %T, want *sportsWSClientImpl", raw)
	}
	if !impl.config.UseProxy {
		t.Fatal("sports config UseProxy not propagated")
	}
	if impl.config.dial == nil {
		t.Fatal("sports config dial should not be nil after defaults")
	}
}

func TestSportsWSConfigUseProxyDefaultFalse(t *testing.T) {
	raw, err := NewSportsWSClient(SportsWSConfig{})
	if err != nil {
		t.Fatalf("NewSportsWSClient: %v", err)
	}
	impl, ok := raw.(*sportsWSClientImpl)
	if !ok {
		t.Fatalf("client type = %T, want *sportsWSClientImpl", raw)
	}
	if impl.config.UseProxy {
		t.Fatal("sports config UseProxy default should be false")
	}
}

func TestCLOBMarketWSConfigUseProxyPropagation(t *testing.T) {
	raw, err := NewCLOBMarketWSClient(CLOBMarketWSConfig{UseProxy: true})
	if err != nil {
		t.Fatalf("NewCLOBMarketWSClient: %v", err)
	}
	impl, ok := raw.(*clobMarketWSClientImpl)
	if !ok {
		t.Fatalf("client type = %T, want *clobMarketWSClientImpl", raw)
	}
	if !impl.config.UseProxy {
		t.Fatal("clob config UseProxy not propagated")
	}
	if impl.config.dial == nil {
		t.Fatal("clob config dial should not be nil after defaults")
	}
}

func TestCLOBMarketWSConfigUseProxyDefaultFalse(t *testing.T) {
	raw, err := NewCLOBMarketWSClient(CLOBMarketWSConfig{})
	if err != nil {
		t.Fatalf("NewCLOBMarketWSClient: %v", err)
	}
	impl, ok := raw.(*clobMarketWSClientImpl)
	if !ok {
		t.Fatalf("client type = %T, want *clobMarketWSClientImpl", raw)
	}
	if impl.config.UseProxy {
		t.Fatal("clob config UseProxy default should be false")
	}
}
