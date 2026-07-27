package ethws

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/gorilla/websocket"
)

const ProxyURLEnvironmentVariable = "ATHENA_TOKEN_NODE_WS_PROXY_URL"

// DialContext dials an Ethereum JSON-RPC endpoint.
// An empty proxyURL always dials directly. A non-empty proxyURL applies only to
// this WebSocket connection and never consults process-wide proxy variables.
func DialContext(ctx context.Context, endpoint string, proxyURL string) (*ethclient.Client, error) {
	dialer, err := newWebsocketDialer(proxyURL)
	if err != nil {
		return nil, err
	}
	rpcClient, err := rpc.DialOptions(ctx, endpoint, rpc.WithWebsocketDialer(dialer))
	if err != nil {
		return nil, err
	}
	return ethclient.NewClient(rpcClient), nil
}

func ValidateProxyURL(raw string) error {
	_, err := parseProxyURL(raw)
	return err
}

func newWebsocketDialer(rawProxyURL string) (websocket.Dialer, error) {
	dialer := websocket.Dialer{}
	proxyURL, err := parseProxyURL(rawProxyURL)
	if err != nil {
		return websocket.Dialer{}, err
	}
	if proxyURL != nil {
		dialer.Proxy = http.ProxyURL(proxyURL)
	}
	return dialer, nil
}

func parseProxyURL(raw string) (*url.URL, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	proxyURL, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("parse token node websocket proxy URL: %w", err)
	}
	proxyURL.Scheme = strings.ToLower(proxyURL.Scheme)
	if proxyURL.Host == "" {
		return nil, fmt.Errorf("token node websocket proxy URL must include a host")
	}
	switch proxyURL.Scheme {
	case "http", "https", "socks5":
	default:
		return nil, fmt.Errorf("token node websocket proxy URL scheme %q is not supported", proxyURL.Scheme)
	}
	return proxyURL, nil
}
