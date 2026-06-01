package ethws

import (
	"net/http"
	"net/url"
	"testing"
)

func TestNewWebsocketDialer_NoProxy(t *testing.T) {
	dialer := newWebsocketDialer(false)
	if dialer.Proxy != nil {
		t.Fatalf("proxy func should be nil when proxy is disabled")
	}
}

func TestNewWebsocketDialer_UseProxyFromEnvironment(t *testing.T) {
	t.Setenv("HTTP_PROXY", "http://127.0.0.1:18080")
	t.Setenv("HTTPS_PROXY", "")
	t.Setenv("ALL_PROXY", "")
	t.Setenv("NO_PROXY", "")
	t.Setenv("http_proxy", "")
	t.Setenv("https_proxy", "")
	t.Setenv("all_proxy", "")
	t.Setenv("no_proxy", "")

	dialer := newWebsocketDialer(true)
	if dialer.Proxy == nil {
		t.Fatalf("proxy func should not be nil when proxy is enabled")
	}

	req := &http.Request{URL: &url.URL{Scheme: "http", Host: "example.com"}}
	proxyURL, err := dialer.Proxy(req)
	if err != nil {
		t.Fatalf("proxy resolution returned error: %v", err)
	}
	if proxyURL == nil {
		t.Fatalf("expected proxy URL from environment, got nil")
	}
	if got := proxyURL.String(); got != "http://127.0.0.1:18080" {
		t.Fatalf("unexpected proxy URL: got %q", got)
	}
}
