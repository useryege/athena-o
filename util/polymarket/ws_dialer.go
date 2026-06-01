package polymarket

import (
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

func newWSDialer(handshakeTimeout time.Duration, useProxy bool) websocket.Dialer {
	dialer := websocket.Dialer{
		HandshakeTimeout: handshakeTimeout,
	}
	if useProxy {
		dialer.Proxy = http.ProxyFromEnvironment
	}
	return dialer
}
