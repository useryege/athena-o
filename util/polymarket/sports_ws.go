package polymarket

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

const DefaultSportsWSURL = "wss://sports-api.polymarket.com/ws"

type SportsWSClient interface {
	Run(ctx context.Context, handler SportsWSHandler) error
}

type SportsWSConfig struct {
	WSURL            string
	UseProxy         bool
	HandshakeTimeout time.Duration
	ReconnectInitial time.Duration
	ReconnectMax     time.Duration
	ReadLimit        int64
	dial             func(ctx context.Context, url string, header http.Header) (*websocket.Conn, *http.Response, error)
}

func (c SportsWSConfig) WithDefaults() SportsWSConfig {
	return c.withDefaults()
}

func (c SportsWSConfig) withDefaults() SportsWSConfig {
	c.WSURL = strings.TrimSpace(c.WSURL)
	if c.WSURL == "" {
		c.WSURL = DefaultSportsWSURL
	}
	if c.HandshakeTimeout <= 0 {
		c.HandshakeTimeout = 10 * time.Second
	}
	if c.ReconnectInitial <= 0 {
		c.ReconnectInitial = time.Second
	}
	if c.ReconnectMax <= 0 {
		c.ReconnectMax = 30 * time.Second
	}
	if c.ReconnectInitial > c.ReconnectMax {
		c.ReconnectInitial = c.ReconnectMax
	}
	if c.ReadLimit <= 0 {
		c.ReadLimit = 4 * 1024 * 1024
	}
	if c.dial == nil {
		dialer := newWSDialer(c.HandshakeTimeout, c.UseProxy)
		c.dial = dialer.DialContext
	}
	return c
}

type sportsWSClientImpl struct {
	config SportsWSConfig
}

var _ SportsWSClient = (*sportsWSClientImpl)(nil)

func NewSportsWSClient(config SportsWSConfig) (SportsWSClient, error) {
	config = config.withDefaults()
	if _, err := url.ParseRequestURI(config.WSURL); err != nil {
		return nil, fmt.Errorf("invalid polymarket sports ws url: %w", err)
	}
	return &sportsWSClientImpl{config: config}, nil
}

type SportsWSHandler struct {
	OnUpdate    func(SportsWSUpdate)
	OnHeartbeat func(string)
	OnUnknown   func(json.RawMessage)
	OnError     func(error)
}

type SportsWSUpdate struct {
	Slug              string  `json:"slug"`
	Live              *bool   `json:"live,omitempty"`
	Ended             *bool   `json:"ended,omitempty"`
	Score             *string `json:"score,omitempty"`
	Period            *string `json:"period,omitempty"`
	Elapsed           *string `json:"elapsed,omitempty"`
	LastUpdate        *string `json:"last_update,omitempty"`
	FinishedTimestamp *string `json:"finished_timestamp,omitempty"`
	Turn              *string `json:"turn,omitempty"`
}

func (c *sportsWSClientImpl) Run(ctx context.Context, handler SportsWSHandler) error {
	backoff := c.config.ReconnectInitial
	for {
		err := c.runOnce(ctx, handler)
		if ctx.Err() != nil {
			return nil
		}
		if err != nil && handler.OnError != nil {
			handler.OnError(err)
		}
		if !sleepWithContext(ctx, withJitter(backoff)) {
			return nil
		}
		backoff *= 2
		if backoff > c.config.ReconnectMax {
			backoff = c.config.ReconnectMax
		}
	}
}

func (c *sportsWSClientImpl) runOnce(ctx context.Context, handler SportsWSHandler) error {
	conn, _, err := c.config.dial(ctx, c.config.WSURL, nil)
	if err != nil {
		return fmt.Errorf("failed to connect polymarket sports ws: %w", err)
	}
	done := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			_ = conn.Close()
		case <-done:
		}
	}()
	defer close(done)
	defer conn.Close()

	conn.SetReadLimit(c.config.ReadLimit)
	conn.SetPingHandler(func(appData string) error {
		if handler.OnHeartbeat != nil {
			handler.OnHeartbeat("PING_FRAME")
		}
		return conn.WriteControl(websocket.PongMessage, []byte(appData), time.Now().Add(time.Second))
	})

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		msgType, payload, err := conn.ReadMessage()
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return fmt.Errorf("failed reading polymarket sports ws: %w", err)
		}
		if msgType != websocket.TextMessage && msgType != websocket.BinaryMessage {
			continue
		}

		text := strings.TrimSpace(string(payload))
		if text == "ping" {
			if handler.OnHeartbeat != nil {
				handler.OnHeartbeat("ping")
			}
			if err := conn.WriteMessage(websocket.TextMessage, []byte("pong")); err != nil {
				return fmt.Errorf("failed to reply sports pong: %w", err)
			}
			continue
		}

		var out SportsWSUpdate
		if err := json.Unmarshal(payload, &out); err != nil {
			if handler.OnUnknown != nil {
				handler.OnUnknown(json.RawMessage(append([]byte(nil), payload...)))
			}
			if handler.OnError != nil {
				handler.OnError(fmt.Errorf("failed to decode sports ws message: %w", err))
			}
			continue
		}
		if strings.TrimSpace(out.Slug) == "" {
			if handler.OnUnknown != nil {
				handler.OnUnknown(json.RawMessage(append([]byte(nil), payload...)))
			}
			continue
		}
		if handler.OnUpdate != nil {
			handler.OnUpdate(out)
		}
	}
}
