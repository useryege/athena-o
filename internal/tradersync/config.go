package tradersync

import "time"

// Config belongs to athena-server's future composition, never the token worker.
type Config struct {
	WebSocketURL, ProxyURL     string
	ReconnectMin, ReconnectMax time.Duration
	OnError                    func(error)
}
