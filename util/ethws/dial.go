package ethws

import (
	"context"
	"net/http"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/gorilla/websocket"
)

// DialContext dials an Ethereum JSON-RPC endpoint.
//
// When useProxy is true, it keeps default go-ethereum behavior and respects
// proxy-related environment variables (e.g. HTTP_PROXY/HTTPS_PROXY/ALL_PROXY).
// When useProxy is false, WebSocket dialing bypasses proxy environment variables.
func DialContext(ctx context.Context, endpoint string, useProxy bool) (*ethclient.Client, error) {
	if useProxy {
		return ethclient.DialContext(ctx, endpoint)
	}

	rpcClient, err := rpc.DialOptions(ctx, endpoint, rpc.WithWebsocketDialer(newWebsocketDialer(false)))
	if err != nil {
		return nil, err
	}
	return ethclient.NewClient(rpcClient), nil
}

func newWebsocketDialer(useProxy bool) websocket.Dialer {
	dialer := websocket.Dialer{}
	if useProxy {
		dialer.Proxy = http.ProxyFromEnvironment
	}
	return dialer
}
