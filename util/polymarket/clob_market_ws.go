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

const DefaultCLOBMarketWSURL = "wss://ws-subscriptions-clob.polymarket.com/ws/market"

type CLOBMarketWSClient interface {
	Run(ctx context.Context, sub CLOBMarketWSSubscription, handler CLOBMarketWSHandler) error
}

type CLOBMarketWSConfig struct {
	WSURL            string
	HandshakeTimeout time.Duration
	ReconnectInitial time.Duration
	ReconnectMax     time.Duration
	ReadLimit        int64
	dial             func(ctx context.Context, url string, header http.Header) (*websocket.Conn, *http.Response, error)
}

func (c CLOBMarketWSConfig) WithDefaults() CLOBMarketWSConfig {
	return c.withDefaults()
}

func (c CLOBMarketWSConfig) withDefaults() CLOBMarketWSConfig {
	c.WSURL = strings.TrimSpace(c.WSURL)
	if c.WSURL == "" {
		c.WSURL = DefaultCLOBMarketWSURL
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
		dialer := websocket.Dialer{HandshakeTimeout: c.HandshakeTimeout}
		c.dial = dialer.DialContext
	}
	return c
}

type clobMarketWSClientImpl struct {
	config CLOBMarketWSConfig
}

var _ CLOBMarketWSClient = (*clobMarketWSClientImpl)(nil)

func NewCLOBMarketWSClient(config CLOBMarketWSConfig) (CLOBMarketWSClient, error) {
	config = config.withDefaults()
	if _, err := url.ParseRequestURI(config.WSURL); err != nil {
		return nil, fmt.Errorf("invalid polymarket clob market ws url: %w", err)
	}
	return &clobMarketWSClientImpl{config: config}, nil
}

type CLOBMarketWSSubscription struct {
	AssetIDs             []string `json:"assets_ids"`
	Type                 string   `json:"type,omitempty"`
	InitialDump          *bool    `json:"initial_dump,omitempty"`
	Level                *int     `json:"level,omitempty"`
	CustomFeatureEnabled *bool    `json:"custom_feature_enabled,omitempty"`
}

type CLOBMarketWSHandler struct {
	OnBook          func(CLOBMarketBookEvent)
	OnPriceChange   func(CLOBMarketPriceChangeEvent)
	OnLastTrade     func(CLOBMarketLastTradePriceEvent)
	OnTickSize      func(CLOBMarketTickSizeChangeEvent)
	OnBestBidAsk    func(CLOBMarketBestBidAskEvent)
	OnNewMarket     func(CLOBMarketNewMarketEvent)
	OnMarketResolve func(CLOBMarketResolvedEvent)
	OnHeartbeat     func(string)
	OnUnknown       func(json.RawMessage)
	OnError         func(error)
}

type CLOBMarketEventEnvelope struct {
	EventType string `json:"event_type"`
}

type CLOBMarketBookEvent struct {
	EventType string             `json:"event_type"`
	AssetID   string             `json:"asset_id"`
	Market    string             `json:"market"`
	Bids      []CLOBOrderSummary `json:"bids"`
	Asks      []CLOBOrderSummary `json:"asks"`
	Timestamp string             `json:"timestamp"`
	Hash      string             `json:"hash"`
}

type CLOBMarketPriceChangeEvent struct {
	EventType    string                  `json:"event_type"`
	Market       string                  `json:"market"`
	PriceChanges []CLOBMarketPriceChange `json:"price_changes"`
	Timestamp    string                  `json:"timestamp"`
}

type CLOBMarketPriceChange struct {
	AssetID string  `json:"asset_id"`
	Price   string  `json:"price"`
	Size    string  `json:"size"`
	Side    string  `json:"side"`
	Hash    string  `json:"hash"`
	BestBid *string `json:"best_bid,omitempty"`
	BestAsk *string `json:"best_ask,omitempty"`
}

type CLOBMarketLastTradePriceEvent struct {
	EventType       string  `json:"event_type"`
	AssetID         string  `json:"asset_id"`
	Market          string  `json:"market"`
	Price           string  `json:"price"`
	Size            string  `json:"size"`
	FeeRateBps      *string `json:"fee_rate_bps,omitempty"`
	Side            string  `json:"side"`
	Timestamp       string  `json:"timestamp"`
	TransactionHash *string `json:"transaction_hash,omitempty"`
}

type CLOBMarketTickSizeChangeEvent struct {
	EventType   string `json:"event_type"`
	AssetID     string `json:"asset_id"`
	Market      string `json:"market"`
	OldTickSize string `json:"old_tick_size"`
	NewTickSize string `json:"new_tick_size"`
	Timestamp   string `json:"timestamp"`
}

type CLOBMarketBestBidAskEvent struct {
	EventType string `json:"event_type"`
	AssetID   string `json:"asset_id"`
	Market    string `json:"market"`
	BestBid   string `json:"best_bid"`
	BestAsk   string `json:"best_ask"`
	Spread    string `json:"spread"`
	Timestamp string `json:"timestamp"`
}

type CLOBMarketEventMessage struct {
	ID          *string `json:"id,omitempty"`
	Ticker      *string `json:"ticker,omitempty"`
	Slug        *string `json:"slug,omitempty"`
	Title       *string `json:"title,omitempty"`
	Description *string `json:"description,omitempty"`
}

type CLOBMarketNewMarketEvent struct {
	EventType             string                  `json:"event_type"`
	ID                    string                  `json:"id"`
	Question              string                  `json:"question"`
	Market                string                  `json:"market"`
	Slug                  string                  `json:"slug"`
	Description           *string                 `json:"description,omitempty"`
	AssetsIDs             []string                `json:"assets_ids"`
	Outcomes              []string                `json:"outcomes"`
	EventMessage          *CLOBMarketEventMessage `json:"event_message,omitempty"`
	Timestamp             string                  `json:"timestamp"`
	Tags                  []string                `json:"tags,omitempty"`
	ConditionID           *string                 `json:"condition_id,omitempty"`
	Active                *bool                   `json:"active,omitempty"`
	ClobTokenIDs          []string                `json:"clob_token_ids,omitempty"`
	SportsMarketType      *string                 `json:"sports_market_type,omitempty"`
	Line                  *string                 `json:"line,omitempty"`
	GameStartTime         *string                 `json:"game_start_time,omitempty"`
	OrderPriceMinTickSize *string                 `json:"order_price_min_tick_size,omitempty"`
	GroupItemTitle        *string                 `json:"group_item_title,omitempty"`
}

type CLOBMarketResolvedEvent struct {
	EventType      string                  `json:"event_type"`
	ID             string                  `json:"id"`
	Market         string                  `json:"market"`
	AssetsIDs      []string                `json:"assets_ids"`
	WinningAssetID string                  `json:"winning_asset_id"`
	WinningOutcome string                  `json:"winning_outcome"`
	EventMessage   *CLOBMarketEventMessage `json:"event_message,omitempty"`
	Timestamp      string                  `json:"timestamp"`
	Tags           []string                `json:"tags,omitempty"`
}

func (c *clobMarketWSClientImpl) Run(ctx context.Context, sub CLOBMarketWSSubscription, handler CLOBMarketWSHandler) error {
	sub, err := sub.normalize()
	if err != nil {
		return err
	}

	backoff := c.config.ReconnectInitial
	for {
		err := c.runOnce(ctx, sub, handler)
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

func (c *clobMarketWSClientImpl) runOnce(ctx context.Context, sub CLOBMarketWSSubscription, handler CLOBMarketWSHandler) error {
	conn, _, err := c.config.dial(ctx, c.config.WSURL, nil)
	if err != nil {
		return fmt.Errorf("failed to connect polymarket market ws: %w", err)
	}
	defer conn.Close()

	conn.SetReadLimit(c.config.ReadLimit)
	conn.SetPingHandler(func(appData string) error {
		if handler.OnHeartbeat != nil {
			handler.OnHeartbeat("PING_FRAME")
		}
		return conn.WriteControl(websocket.PongMessage, []byte(appData), time.Now().Add(time.Second))
	})

	if err := conn.WriteJSON(sub); err != nil {
		return fmt.Errorf("failed to subscribe polymarket market ws: %w", err)
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		msgType, payload, err := conn.ReadMessage()
		if err != nil {
			return fmt.Errorf("failed reading polymarket market ws: %w", err)
		}
		if msgType != websocket.TextMessage && msgType != websocket.BinaryMessage {
			continue
		}
		if handled, err := c.handleHeartbeat(conn, payload, handler); handled {
			if err != nil {
				return err
			}
			continue
		}
		if err := dispatchMarketEvent(payload, handler); err != nil && handler.OnError != nil {
			handler.OnError(err)
		}
	}
}

func (c *clobMarketWSClientImpl) handleHeartbeat(conn *websocket.Conn, payload []byte, handler CLOBMarketWSHandler) (bool, error) {
	text := strings.TrimSpace(string(payload))
	switch text {
	case "PING":
		if handler.OnHeartbeat != nil {
			handler.OnHeartbeat("PING")
		}
		if err := conn.WriteMessage(websocket.TextMessage, []byte("PONG")); err != nil {
			return true, fmt.Errorf("failed to reply PONG: %w", err)
		}
		return true, nil
	case "ping":
		if handler.OnHeartbeat != nil {
			handler.OnHeartbeat("ping")
		}
		if err := conn.WriteMessage(websocket.TextMessage, []byte("pong")); err != nil {
			return true, fmt.Errorf("failed to reply pong: %w", err)
		}
		return true, nil
	default:
		return false, nil
	}
}

func (s CLOBMarketWSSubscription) normalize() (CLOBMarketWSSubscription, error) {
	out := CLOBMarketWSSubscription{
		Type:                 "market",
		InitialDump:          s.InitialDump,
		Level:                s.Level,
		CustomFeatureEnabled: s.CustomFeatureEnabled,
	}
	for _, id := range s.AssetIDs {
		id = strings.TrimSpace(id)
		if id != "" {
			out.AssetIDs = append(out.AssetIDs, id)
		}
	}
	if len(out.AssetIDs) == 0 {
		return out, fmt.Errorf("market ws subscription requires at least one asset id")
	}
	return out, nil
}

func dispatchMarketEvent(payload []byte, handler CLOBMarketWSHandler) error {
	var env CLOBMarketEventEnvelope
	if err := json.Unmarshal(payload, &env); err != nil {
		if handler.OnUnknown != nil {
			handler.OnUnknown(json.RawMessage(append([]byte(nil), payload...)))
		}
		return fmt.Errorf("failed to decode market ws envelope: %w", err)
	}

	switch env.EventType {
	case "book":
		var e CLOBMarketBookEvent
		if err := json.Unmarshal(payload, &e); err != nil {
			return fmt.Errorf("failed to decode market ws book event: %w", err)
		}
		if handler.OnBook != nil {
			handler.OnBook(e)
		}
	case "price_change":
		var e CLOBMarketPriceChangeEvent
		if err := json.Unmarshal(payload, &e); err != nil {
			return fmt.Errorf("failed to decode market ws price_change event: %w", err)
		}
		if handler.OnPriceChange != nil {
			handler.OnPriceChange(e)
		}
	case "last_trade_price":
		var e CLOBMarketLastTradePriceEvent
		if err := json.Unmarshal(payload, &e); err != nil {
			return fmt.Errorf("failed to decode market ws last_trade_price event: %w", err)
		}
		if handler.OnLastTrade != nil {
			handler.OnLastTrade(e)
		}
	case "tick_size_change":
		var e CLOBMarketTickSizeChangeEvent
		if err := json.Unmarshal(payload, &e); err != nil {
			return fmt.Errorf("failed to decode market ws tick_size_change event: %w", err)
		}
		if handler.OnTickSize != nil {
			handler.OnTickSize(e)
		}
	case "best_bid_ask":
		var e CLOBMarketBestBidAskEvent
		if err := json.Unmarshal(payload, &e); err != nil {
			return fmt.Errorf("failed to decode market ws best_bid_ask event: %w", err)
		}
		if handler.OnBestBidAsk != nil {
			handler.OnBestBidAsk(e)
		}
	case "new_market":
		var e CLOBMarketNewMarketEvent
		if err := json.Unmarshal(payload, &e); err != nil {
			return fmt.Errorf("failed to decode market ws new_market event: %w", err)
		}
		if handler.OnNewMarket != nil {
			handler.OnNewMarket(e)
		}
	case "market_resolved":
		var e CLOBMarketResolvedEvent
		if err := json.Unmarshal(payload, &e); err != nil {
			return fmt.Errorf("failed to decode market ws market_resolved event: %w", err)
		}
		if handler.OnMarketResolve != nil {
			handler.OnMarketResolve(e)
		}
	default:
		if handler.OnUnknown != nil {
			handler.OnUnknown(json.RawMessage(append([]byte(nil), payload...)))
		}
	}

	return nil
}

func sleepWithContext(ctx context.Context, d time.Duration) bool {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func withJitter(base time.Duration) time.Duration {
	if base <= 0 {
		return 0
	}
	half := base / 2
	if half <= 0 {
		return base
	}
	jitter := time.Duration(time.Now().UnixNano() % int64(half+1))
	return base + jitter
}
