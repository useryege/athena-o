package polymarket

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

type wsIntegrationDialerInfo struct {
	Description string
}

func newIntegrationWSDialer(handshakeTimeout time.Duration) (*websocket.Dialer, wsIntegrationDialerInfo, error) {
	dialer := &websocket.Dialer{
		HandshakeTimeout: handshakeTimeout,
	}

	proxyURL, source, err := resolveWSIntegrationProxyURL()
	if err != nil {
		return nil, wsIntegrationDialerInfo{}, err
	}

	if proxyURL != nil {
		dialer.Proxy = http.ProxyURL(proxyURL)
		return dialer, wsIntegrationDialerInfo{
			Description: fmt.Sprintf("%s=%s", source, proxyURL.String()),
		}, nil
	}

	dialer.Proxy = http.ProxyFromEnvironment
	return dialer, wsIntegrationDialerInfo{
		Description: "http.ProxyFromEnvironment (no ALL_PROXY/all_proxy override)",
	}, nil
}

func resolveWSIntegrationProxyURL() (*url.URL, string, error) {
	for _, key := range []string{"ALL_PROXY", "all_proxy"} {
		raw := strings.TrimSpace(os.Getenv(key))
		if raw == "" {
			continue
		}

		proxyURL, err := url.Parse(raw)
		if err != nil {
			return nil, key, fmt.Errorf("invalid %s: %w", key, err)
		}
		if strings.TrimSpace(proxyURL.Scheme) == "" || strings.TrimSpace(proxyURL.Host) == "" {
			return nil, key, fmt.Errorf("invalid %s: expected proxy URL with scheme and host, got %q", key, raw)
		}

		return proxyURL, key, nil
	}

	return nil, "", nil
}
