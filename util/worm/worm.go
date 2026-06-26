package worm

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/mr-tron/base58/base58"
	"github.com/useryege/athena/util/ratelimit"
)

const (
	DefaultBaseURL           = "https://api.worm.wtf"
	DefaultTimeout           = 30 * time.Second
	DefaultRateLimitRequests = 120
	DefaultRateLimitPeriod   = time.Minute

	errorBodyLimit            = 4096
	rateLimitMaxRetries       = 3
	rateLimitBackoffBase      = time.Second
	rateLimitBackoffJitterMax = 250 * time.Millisecond

	headerAPIKey    = "WORM-API-KEY"
	headerTimestamp = "WORM-TIMESTAMP"
	headerSignature = "WORM-SIGNATURE"
)

type Client interface {
	Search(ctx context.Context, options SearchOptions) (*SearchResponse, error)
	// SPORTS
	ListSports(ctx context.Context) (*ListSportsResponse, error)
	// MARKETS
	ListMarkets(ctx context.Context, options ListMarketsOptions) (*ListMarketsResponse, error)
	GetMarket(ctx context.Context, conditionID string) (*Market, error)
	GetMarketStats(ctx context.Context, conditionID string) (*MarketStats, error)
	GetMarketPrice(ctx context.Context, conditionID string, options GetMarketPriceOptions) (*MarketPrice, error)
	GetMarketOrderBook(ctx context.Context, conditionID string, options GetMarketOrderBookOptions) (*MarketOrderBook, error)
	GetMarketCandles(ctx context.Context, conditionID string, options GetMarketCandlesOptions) (*ListMarketCandlesResponse, error)
	ListMarketTrades(ctx context.Context, conditionID string, options ListMarketTradesOptions) (*ListMarketTradesResponse, error)
	ListMarketMarginActivity(ctx context.Context, conditionID string, options ListMarketMarginActivityOptions) (*ListMarketMarginActivityResponse, error)
	// EVENTS
	ListEvents(ctx context.Context, options ListEventsOptions) (*ListEventsResponse, error)
	GetEvent(ctx context.Context, conditionID string) (*Event, error)
	// AUTH KEYS
	CreateAuthChallenge(ctx context.Context, request CreateAuthChallengeRequest) (*AuthChallenge, error)
	CreateAPIKey(ctx context.Context, request CreateAPIKeyRequest) (*APIKeySecret, error)
	CreateAPIKeyFromPrivateKey(ctx context.Context, privateKey string) (*APIKeySecret, error)
	ListAPIKeys(ctx context.Context) (*ListAPIKeysResponse, error)
	RevokeAPIKey(ctx context.Context, keyID string) (*APIKey, error)
	// ORDER
	CreateOrderDraft(ctx context.Context, request CreateOrderDraftRequest) (*DraftMessage, error)
	SubmitOrder(ctx context.Context, pubkey string, request SubmitSignatureRequest) (*Order, error)
	GetOrder(ctx context.Context, pubkey string) (*Order, error)
	ListOrders(ctx context.Context, options ListOrdersOptions) (*ListOrdersResponse, error)
	CreateCancelDraft(ctx context.Context, pubkey string) (*DraftMessage, error)
	SubmitOrderCancel(ctx context.Context, pubkey string, request SubmitSignatureRequest) (*Order, error)
	// TRADE
	ListTrades(ctx context.Context, options ListTradesOptions) (*ListTradesResponse, error)
	// ACCOUNT
	GetAccountSummary(ctx context.Context) (*AccountSummary, error)
	GetAccountPnL(ctx context.Context, options GetAccountPnLOptions) (*AccountPnL, error)
	ListAccountAssets(ctx context.Context, options ListAccountAssetsOptions) (*ListAccountAssetsResponse, error)
	// MARGIN
	EstimateMarginPosition(ctx context.Context, options EstimateMarginPositionOptions) (*MarginPositionEstimate, error)
	CreatePositionRequest(ctx context.Context, request CreatePositionRequestRequest) (*PositionRequest, error)
	SubmitPositionRequest(ctx context.Context, pubkey string, request SubmitSignatureRequest) (*PositionRequest, error)
	GetPositionRequest(ctx context.Context, pubkey string) (*PositionRequest, error)
	CancelPositionRequest(ctx context.Context, pubkey string) (*PositionRequest, error)
	ListPositionRequests(ctx context.Context, options ListPositionRequestsOptions) (*ListPositionRequestsResponse, error)
	ListMarginPositions(ctx context.Context, options ListMarginPositionsOptions) (*ListMarginPositionsResponse, error)
	GetMarginPosition(ctx context.Context, pubkey string) (*MarginPosition, error)
	SetTPSL(ctx context.Context, pubkey string, request SetTPSLRequest) (*TPSL, error)
	DeleteTPSL(ctx context.Context, pubkey string) (*TPSL, error)
	CloseMarginPosition(ctx context.Context, pubkey string, options CloseMarginPositionOptions) (*CloseMarginPositionResult, error)
	ListMarginSettlements(ctx context.Context, options ListMarginSettlementsOptions) (*ListMarginSettlementsResponse, error)
	ClaimPositionSettlement(ctx context.Context, pubkey string) (*ClaimPositionSettlementResult, error)
	// REDEEMS
	ListRedeems(ctx context.Context, options ListRedeemsOptions) (*ListRedeemsResponse, error)
	StartRedeem(ctx context.Context, request StartRedeemRequest) (*DraftMessage, error)
	GetRedeem(ctx context.Context, pubkey string) (*Redeem, error)
	SubmitRedeem(ctx context.Context, pubkey string, request SubmitSignatureRequest) (*Redeem, error)
}

type Config struct {
	BaseURL     string
	APIKey      string
	APISecret   string
	Timeout     time.Duration
	Now         func() time.Time
	RateLimiter ratelimit.Limiter
}

type clientImpl struct {
	config      Config
	client      *http.Client
	rateLimiter ratelimit.Limiter
}

var _ Client = (*clientImpl)(nil)

func NewClient(config Config) (Client, error) {
	config = config.withDefaults()
	if _, err := url.ParseRequestURI(config.BaseURL); err != nil {
		return nil, fmt.Errorf("invalid worm base url: %w", err)
	}
	rateLimiter := config.RateLimiter
	if rateLimiter == nil {
		var err error
		rateLimiter, err = ratelimit.New(ratelimit.Config{
			Requests: DefaultRateLimitRequests,
			Per:      DefaultRateLimitPeriod,
			Burst:    1,
		})
		if err != nil {
			return nil, fmt.Errorf("create worm rate limiter: %w", err)
		}
	}
	return &clientImpl{
		config:      config,
		client:      &http.Client{Timeout: config.Timeout},
		rateLimiter: rateLimiter,
	}, nil
}

func (c Config) WithDefaults() Config {
	return c.withDefaults()
}

func (c Config) withDefaults() Config {
	c.BaseURL = strings.TrimSpace(c.BaseURL)
	if c.BaseURL == "" {
		c.BaseURL = DefaultBaseURL
	}
	c.APIKey = strings.TrimSpace(c.APIKey)
	c.APISecret = strings.TrimSpace(c.APISecret)
	if c.Timeout <= 0 {
		c.Timeout = DefaultTimeout
	}
	if c.Now == nil {
		c.Now = time.Now
	}
	return c
}

type Error struct {
	Code               int              `json:"code"`
	Slug               string           `json:"slug"`
	Message            string           `json:"message"`
	Details            []map[string]any `json:"details"`
	Status             string           `json:"-"`
	StatusCode         int              `json:"-"`
	RetryAfter         time.Duration    `json:"-"`
	RateLimitLimit     int              `json:"-"`
	RateLimitRemaining int              `json:"-"`
	RateLimitReset     int64            `json:"-"`
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	if e.Status != "" {
		return fmt.Sprintf("worm request failed with status %s: %s (%d %s)", e.Status, e.Message, e.Code, e.Slug)
	}
	return fmt.Sprintf("worm request failed: %s (%d %s)", e.Message, e.Code, e.Slug)
}

type EnvelopeMeta struct {
	Limit      int     `json:"limit,omitempty"`
	NextCursor *string `json:"next_cursor,omitempty"`
	IsYes      *bool   `json:"is_yes,omitempty"`
}

type PageOptions struct {
	Limit  int
	Cursor string
}

type SearchOptions struct {
	PageOptions
	Q                 string
	Category          string
	State             string
	Sport             string
	League            string
	ResolutionTimeGT  int64
	ResolutionTimeLT  int64
	PriceGTE          string
	PriceLTE          string
	Leveraged         *bool
	MaxLeverageYesGTE string
	MaxLeverageYesLTE string
	MaxLeverageNoGTE  string
	MaxLeverageNoLTE  string
	ResultType        string
	Sort              string
}

type ListMarketsOptions struct {
	PageOptions
	State    string
	Category string
	Sort     string
	Sport    string
	League   string
}

type ListEventsOptions = ListMarketsOptions

type GetMarketPriceOptions struct {
	IsYes *bool
}

type GetMarketOrderBookOptions struct {
	Depth int
	IsYes *bool
}

type GetMarketCandlesOptions struct {
	StartTime int64
	EndTime   int64
	Interval  string
	IsYes     *bool
}

type ListMarketTradesOptions struct {
	PageOptions
}

type ListMarketMarginActivityOptions struct {
	PageOptions
	ActivityType string
}

type ListOrdersOptions struct {
	PageOptions
	MarketConditionID string
	Status            string
	IsYes             *bool
	Side              string
}

type ListTradesOptions struct {
	PageOptions
	MarketConditionID string
}

type GetAccountPnLOptions struct {
	MarketConditionID string
	EventConditionID  string
}

type ListAccountAssetsOptions struct {
	PageOptions
	MarketConditionID string
}

type EstimateMarginPositionOptions struct {
	MarketConditionID string
	Funds             string
	IsYes             *bool
	Leverage          *float64
}

type ListPositionRequestsOptions struct {
	PageOptions
	MarketConditionID string
	States            string
	IsYes             *bool
	Leverage          *float64
	Sort              string
}

type ListMarginPositionsOptions struct {
	PageOptions
	MarketConditionID string
	EventConditionID  string
	IsClosed          *bool
	Sort              string
}

type ListMarginSettlementsOptions struct {
	PageOptions
	PositionPubkey    string
	MarketConditionID string
	States            string
	MarketState       string
	PositionLeverage  *float64
	Sort              string
}

type CloseMarginPositionOptions struct {
	Price *float64
}

type ListRedeemsOptions struct {
	PageOptions
	MarketConditionID string
	State             string
}

type SearchResponse struct {
	Results []SearchResult
	Meta    EnvelopeMeta
}

type ListSportsResponse struct {
	Sports []Sport
	Meta   EnvelopeMeta
}

type ListMarketsResponse struct {
	Markets []MarketSummary
	Meta    EnvelopeMeta
}

type ListMarketCandlesResponse struct {
	Candles []MarketCandle
	Meta    EnvelopeMeta
}

type ListMarketTradesResponse struct {
	Trades []MarketTrade
	Meta   EnvelopeMeta
}

type ListMarketMarginActivityResponse struct {
	Activities []MarketMarginActivity
	Meta       EnvelopeMeta
}

type ListEventsResponse struct {
	Events []Event
	Meta   EnvelopeMeta
}

type ListAPIKeysResponse struct {
	Keys []APIKey
	Meta EnvelopeMeta
}

type ListOrdersResponse struct {
	Orders []Order
	Meta   EnvelopeMeta
}

type ListTradesResponse struct {
	Trades []Trade
	Meta   EnvelopeMeta
}

type ListAccountAssetsResponse struct {
	Assets []AccountAsset
	Meta   EnvelopeMeta
}

type ListPositionRequestsResponse struct {
	Requests []PositionRequest
	Meta     EnvelopeMeta
}

type ListMarginPositionsResponse struct {
	Positions []MarginPosition
	Meta      EnvelopeMeta
}

type ListMarginSettlementsResponse struct {
	Settlements []MarginSettlement
	Meta        EnvelopeMeta
}

type ListRedeemsResponse struct {
	Redeems []Redeem
	Meta    EnvelopeMeta
}

type UserSummary struct {
	Username        string  `json:"username,omitempty"`
	Image           *string `json:"image,omitempty"`
	ProfileImage    *string `json:"profile_image,omitempty"`
	TwitterUsername *string `json:"twitter_username,omitempty"`
}

type EventMini struct {
	Title       string  `json:"title,omitempty"`
	ConditionID string  `json:"condition_id,omitempty"`
	Logo        *string `json:"logo,omitempty"`
}

type MarketMini struct {
	ConditionID string `json:"condition_id,omitempty"`
	Title       string `json:"title,omitempty"`
}

type Sport struct {
	Name    string   `json:"name,omitempty"`
	Slug    string   `json:"slug,omitempty"`
	Leagues []League `json:"leagues,omitempty"`
}

type League struct {
	Name string `json:"name,omitempty"`
	Slug string `json:"slug,omitempty"`
}

type MarketSummary struct {
	ConditionID    string       `json:"condition_id,omitempty"`
	Title          string       `json:"title,omitempty"`
	Description    *string      `json:"description,omitempty"`
	Logo           *string      `json:"logo,omitempty"`
	LastTradePrice *string      `json:"last_trade_price,omitempty"`
	State          string       `json:"state,omitempty"`
	Category       string       `json:"category,omitempty"`
	Created        *int64       `json:"created,omitempty"`
	Creator        *UserSummary `json:"creator,omitempty"`
	Event          *EventMini   `json:"event,omitempty"`
	MarginEnabled  bool         `json:"margin_enabled,omitempty"`
	Outcomes       []Outcome    `json:"outcomes,omitempty"`
}

type Market struct {
	MarketSummary
	YesOutcomeLabel *string          `json:"yes_outcome_label,omitempty"`
	NoOutcomeLabel  *string          `json:"no_outcome_label,omitempty"`
	Outcomes        []Outcome        `json:"outcomes,omitempty"`
	Rules           []string         `json:"rules,omitempty"`
	ResolutionDate  *int64           `json:"resolution_date,omitempty"`
	MakerFee        *string          `json:"maker_fee,omitempty"`
	TakerFee        *string          `json:"taker_fee,omitempty"`
	Config          *MarketConfig    `json:"config,omitempty"`
	RawConfig       *json.RawMessage `json:"-"`
}

func (m *Market) UnmarshalJSON(data []byte) error {
	var raw struct {
		MarketSummary
		YesOutcomeLabel *string       `json:"yes_outcome_label,omitempty"`
		NoOutcomeLabel  *string       `json:"no_outcome_label,omitempty"`
		Outcomes        []Outcome     `json:"outcomes,omitempty"`
		Rules           []string      `json:"rules"`
		ResolutionDate  *int64        `json:"resolution_date,omitempty"`
		MakerFee        *string       `json:"maker_fee,omitempty"`
		TakerFee        *string       `json:"taker_fee,omitempty"`
		Config          *MarketConfig `json:"config,omitempty"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	m.MarketSummary = raw.MarketSummary
	m.YesOutcomeLabel = raw.YesOutcomeLabel
	m.NoOutcomeLabel = raw.NoOutcomeLabel
	m.Outcomes = raw.Outcomes
	m.ResolutionDate = raw.ResolutionDate
	m.MakerFee = raw.MakerFee
	m.TakerFee = raw.TakerFee
	m.Config = raw.Config
	m.Rules = nil
	if raw.Rules != nil {
		m.Rules = append([]string{}, raw.Rules...)
	}
	if raw.Config != nil {
		var rawFields map[string]json.RawMessage
		if err := json.Unmarshal(data, &rawFields); err == nil && len(rawFields["config"]) > 0 && string(rawFields["config"]) != "null" {
			configCopy := append(json.RawMessage(nil), rawFields["config"]...)
			m.RawConfig = &configCopy
		}
	} else {
		m.RawConfig = nil
	}
	return nil
}

type Outcome struct {
	IsYes bool   `json:"is_yes"`
	Text  string `json:"text"`
}

type MarketConfig struct {
	Kind                string  `json:"kind,omitempty"`
	MaxLeverageYes      *string `json:"max_leverage_yes,omitempty"`
	MaxLeverageNo       *string `json:"max_leverage_no,omitempty"`
	OpeningFee          *string `json:"opening_fee,omitempty"`
	ClosingFee          *string `json:"closing_fee,omitempty"`
	AnnualFeeRate       *string `json:"annual_fee_rate,omitempty"`
	OrderMinSize        *string `json:"order_min_size,omitempty"`
	PriceDecimals       *int    `json:"price_decimals,omitempty"`
	SharesDecimals      *int    `json:"shares_decimals,omitempty"`
	MinPrice            *string `json:"min_price,omitempty"`
	MaxPrice            *string `json:"max_price,omitempty"`
	MinAmount           *string `json:"min_amount,omitempty"`
	MaxAmount           *string `json:"max_amount,omitempty"`
	MinFunds            *string `json:"min_funds,omitempty"`
	MaxFunds            *string `json:"max_funds,omitempty"`
	PricePrecision      *int    `json:"price_precision,omitempty"`
	AmountPrecision     *int    `json:"amount_precision,omitempty"`
	FundsPrecision      *int    `json:"funds_precision,omitempty"`
	MakerFeeRate        *string `json:"maker_fee_rate,omitempty"`
	TakerFeeRate        *string `json:"taker_fee_rate,omitempty"`
	DefaultSlippageRate *string `json:"default_slippage_rate,omitempty"`
}

type MarketStats struct {
	TotalVolume    string `json:"total_volume"`
	TotalVolume24H string `json:"total_volume_24h"`
	MarketCap      string `json:"market_cap"`
	TradeCount     int64  `json:"trade_count"`
}

type MarketPrice struct {
	ConditionID string  `json:"condition_id"`
	Price       *string `json:"price"`
	PriceKind   string  `json:"price_kind"`
	IsYes       bool    `json:"is_yes"`
}

type MarketOrderBook struct {
	Market string           `json:"market"`
	IsYes  bool             `json:"is_yes"`
	Bid    []OrderBookLevel `json:"bid"`
	Ask    []OrderBookLevel `json:"ask"`
}

type OrderBookLevel struct {
	Price       string `json:"price"`
	TotalAmount string `json:"total_amount"`
}

type MarketCandle struct {
	Timestamp int64  `json:"timestamp"`
	Open      string `json:"open"`
	High      string `json:"high"`
	Low       string `json:"low"`
	Close     string `json:"close"`
	Volume    string `json:"volume"`
	IsYes     bool   `json:"is_yes"`
}

type MarketTrade struct {
	MarketConditionID *string `json:"market_condition_id"`
	Amount            string  `json:"amount"`
	Price             string  `json:"price"`
	MakerFee          string  `json:"maker_fee"`
	TakerFee          string  `json:"taker_fee"`
	Timestamp         int64   `json:"timestamp"`
	State             string  `json:"state"`
	IsYes             bool    `json:"is_yes"`
}

type MarketMarginActivity struct {
	ActivityType string         `json:"activity_type"`
	User         UserSummary    `json:"user"`
	Market       *MarketSummary `json:"market"`
	Shares       *string        `json:"shares"`
	Price        *string        `json:"price"`
	RealizedPnL  *string        `json:"realized_pnl"`
	Leverage     *string        `json:"leverage"`
	IsYes        *bool          `json:"is_yes"`
	Created      int64          `json:"created"`
}

type Event struct {
	ConditionID string          `json:"condition_id"`
	Title       string          `json:"title"`
	Description *string         `json:"description,omitempty"`
	Logo        *string         `json:"logo,omitempty"`
	VideoURL    *string         `json:"video_url,omitempty"`
	Category    string          `json:"category"`
	Created     *int64          `json:"created"`
	Markets     []MarketSummary `json:"markets,omitempty"`
}

type SearchResult struct {
	ResultType string        `json:"result_type"`
	Summary    SearchSummary `json:"summary"`
}

type SearchSummary struct {
	MarketSummary
	VideoURL        *string          `json:"video_url,omitempty"`
	Markets         []MarketSummary  `json:"markets,omitempty"`
	YesOutcomeLabel *string          `json:"yes_outcome_label,omitempty"`
	NoOutcomeLabel  *string          `json:"no_outcome_label,omitempty"`
	Rules           []string         `json:"rules,omitempty"`
	ResolutionDate  *int64           `json:"resolution_date,omitempty"`
	MakerFee        *string          `json:"maker_fee,omitempty"`
	TakerFee        *string          `json:"taker_fee,omitempty"`
	Config          *MarketConfig    `json:"config,omitempty"`
	RawConfig       *json.RawMessage `json:"-"`
}

func (s *SearchSummary) UnmarshalJSON(data []byte) error {
	var raw struct {
		MarketSummary
		VideoURL        *string         `json:"video_url,omitempty"`
		Markets         []MarketSummary `json:"markets,omitempty"`
		YesOutcomeLabel *string         `json:"yes_outcome_label,omitempty"`
		NoOutcomeLabel  *string         `json:"no_outcome_label,omitempty"`
		Rules           []string        `json:"rules"`
		ResolutionDate  *int64          `json:"resolution_date,omitempty"`
		MakerFee        *string         `json:"maker_fee,omitempty"`
		TakerFee        *string         `json:"taker_fee,omitempty"`
		Config          *MarketConfig   `json:"config,omitempty"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	s.MarketSummary = raw.MarketSummary
	s.VideoURL = raw.VideoURL
	s.Markets = raw.Markets
	s.YesOutcomeLabel = raw.YesOutcomeLabel
	s.NoOutcomeLabel = raw.NoOutcomeLabel
	s.ResolutionDate = raw.ResolutionDate
	s.MakerFee = raw.MakerFee
	s.TakerFee = raw.TakerFee
	s.Config = raw.Config
	s.Rules = nil
	if raw.Rules != nil {
		s.Rules = append([]string{}, raw.Rules...)
	}
	if raw.Config != nil {
		var rawFields map[string]json.RawMessage
		if err := json.Unmarshal(data, &rawFields); err == nil && len(rawFields["config"]) > 0 && string(rawFields["config"]) != "null" {
			configCopy := append(json.RawMessage(nil), rawFields["config"]...)
			s.RawConfig = &configCopy
		}
	} else {
		s.RawConfig = nil
	}
	return nil
}

type CreateAuthChallengeRequest struct {
	WalletAddress string `json:"wallet_address"`
}

type CreateAPIKeyRequest struct {
	WalletAddress string `json:"wallet_address"`
	Message       string `json:"message"`
	Signature     string `json:"signature"`
	Nonce         string `json:"nonce"`
}

type AuthChallenge struct {
	Nonce            string `json:"nonce"`
	Message          string `json:"message"`
	ExpiresInSeconds int64  `json:"expires_in_seconds"`
}

type APIKeySecret struct {
	APIKey string `json:"api_key"`
	Secret string `json:"secret"`
}

type APIKey struct {
	KeyID     string `json:"key_id"`
	Created   int64  `json:"created"`
	RevokedAt *int64 `json:"revoked_at"`
}

type CreateOrderDraftRequest struct {
	MarketConditionID string `json:"market_condition_id"`
	IsYes             bool   `json:"is_yes"`
	Side              string `json:"side"`
	OrderType         string `json:"order_type"`
	Price             string `json:"price,omitempty"`
	Amount            string `json:"amount,omitempty"`
	Funds             string `json:"funds,omitempty"`
}

type SubmitSignatureRequest struct {
	Signature string `json:"signature"`
}

type DraftMessage struct {
	Pubkey  *string `json:"pubkey"`
	Message *string `json:"message"`
}

type Order struct {
	Pubkey          *string    `json:"pubkey"`
	Status          *string    `json:"status"`
	Side            string     `json:"side"`
	Price           *string    `json:"price"`
	Amount          *string    `json:"amount"`
	RemainingAmount *string    `json:"remaining_amount"`
	FilledAmount    *string    `json:"filled_amount"`
	Funds           *string    `json:"funds"`
	RemainingFunds  *string    `json:"remaining_funds"`
	FilledFunds     *string    `json:"filled_funds"`
	Market          MarketMini `json:"market"`
	Outcome         Outcome    `json:"outcome"`
}

type Trade struct {
	Pubkey            string  `json:"pubkey"`
	MarketConditionID *string `json:"market_condition_id"`
	Amount            string  `json:"amount"`
	Price             string  `json:"price"`
	Timestamp         int64   `json:"timestamp"`
	Fee               *string `json:"fee"`
	IsMaker           bool    `json:"is_maker"`
	State             string  `json:"state"`
	Order             Order   `json:"order"`
}

type AccountSummary struct {
	Username        string  `json:"username"`
	TwitterUsername *string `json:"twitter_username"`
	JoinedAt        *int64  `json:"joined_at"`
}

type AccountPnL struct {
	MarginPositionSettlementsPnL string `json:"margin_position_settlements_pnl"`
	MarginPositionsUnrealizedPnL string `json:"margin_positions_unrealized_pnl"`
	RedeemsFunds                 string `json:"redeems_funds"`
	ActiveAssetsValue            string `json:"active_assets_value"`
	CreatorFees                  string `json:"creator_fees"`
	TotalPnL                     string `json:"total_pnl"`
}

type AccountAsset struct {
	AssetKind string               `json:"asset_kind"`
	Amounts   AssetAmounts         `json:"amounts"`
	Token     TokenReference       `json:"token"`
	Value     AccountAssetValue    `json:"value"`
	Position  AccountAssetPosition `json:"position"`
	Created   *int64               `json:"created"`
}

type AssetAmounts struct {
	Total     string `json:"total"`
	Locked    string `json:"locked"`
	Available string `json:"available"`
}

type TokenReference struct {
	Symbol  string  `json:"symbol"`
	Address *string `json:"address"`
}

type AccountAssetValue struct {
	USDT  string `json:"usdt"`
	Basis string `json:"basis"`
}

type AccountAssetPosition struct {
	Market        MarketSummary `json:"market"`
	IsYes         bool          `json:"is_yes"`
	OutcomeText   string        `json:"outcome_text"`
	IsFinal       bool          `json:"is_final"`
	AvgTradePrice string        `json:"avg_trade_price"`
}

type MarginPositionEstimate struct {
	AveragePrice     string  `json:"average_price"`
	TotalShares      string  `json:"total_shares"`
	TotalCost        string  `json:"total_cost"`
	BestAsk          string  `json:"best_ask"`
	WorstFillPrice   string  `json:"worst_fill_price"`
	IsFullyFilled    bool    `json:"is_fully_filled"`
	FeeAmount        string  `json:"fee_amount"`
	UserFundsNeeded  string  `json:"user_funds_needed"`
	LiquidationPrice *string `json:"liquidation_price"`
}

type CreatePositionRequestRequest struct {
	Type              string   `json:"type"`
	MarketConditionID string   `json:"market_condition_id"`
	IsYes             *bool    `json:"is_yes,omitempty"`
	Leverage          *float64 `json:"leverage,omitempty"`
	TakeProfitPrice   string   `json:"take_profit_price,omitempty"`
	StopLossPrice     string   `json:"stop_loss_price,omitempty"`
	Funds             string   `json:"funds,omitempty"`
	Price             string   `json:"price,omitempty"`
	Shares            string   `json:"shares,omitempty"`
}

type PositionRequest struct {
	Pubkey          string         `json:"pubkey"`
	Type            string         `json:"type"`
	State           string         `json:"state"`
	OrderState      *string        `json:"order_state"`
	Message         *string        `json:"message"`
	FundingTxID     *string        `json:"funding_txid"`
	RefundTxID      *string        `json:"refund_txid"`
	Market          *MarketSummary `json:"market"`
	IsYes           bool           `json:"is_yes"`
	Leverage        string         `json:"leverage"`
	Funds           string         `json:"funds"`
	Price           *string        `json:"price"`
	Shares          *string        `json:"shares"`
	TakeProfitPrice *string        `json:"take_profit_price"`
	StopLossPrice   *string        `json:"stop_loss_price"`
	Created         *int64         `json:"created"`
}

type MarginPosition struct {
	Pubkey                string        `json:"pubkey"`
	PositionRequestPubkey *string       `json:"position_request_pubkey"`
	Market                MarketSummary `json:"market"`
	IsYes                 bool          `json:"is_yes"`
	Leverage              string        `json:"leverage"`
	TotalShares           string        `json:"total_shares"`
	AvgEntryPrice         string        `json:"avg_entry_price"`
	ClosingPrice          *string       `json:"closing_price"`
	UnrealizedPnL         *string       `json:"unrealized_pnl"`
	RealizedPnL           string        `json:"realized_pnl"`
	UserLiquidity         string        `json:"user_liquidity"`
	TotalLiquidity        string        `json:"total_liquidity"`
	LiquidationPrice      string        `json:"liquidation_price"`
	IsClosed              bool          `json:"is_closed"`
	IsLiquidated          bool          `json:"is_liquidated"`
	IsClaimed             bool          `json:"is_claimed"`
	TPSL                  *TPSL         `json:"tp_sl"`
	Created               *int64        `json:"created"`
}

type SetTPSLRequest struct {
	TakeProfitPrice string `json:"take_profit_price,omitempty"`
	StopLossPrice   string `json:"stop_loss_price,omitempty"`
}

type TPSL struct {
	TakeProfitPrice *string `json:"take_profit_price"`
	StopLossPrice   *string `json:"stop_loss_price"`
	State           string  `json:"state"`
	TriggerType     *string `json:"trigger_type"`
	TriggeredPrice  *string `json:"triggered_price"`
	TriggeredAt     *int64  `json:"triggered_at"`
}

type CloseMarginPositionResult struct {
	CloseType      string `json:"close_type"`
	PositionPubkey string `json:"position_pubkey"`
	IsClosed       bool   `json:"is_closed"`
}

type MarginSettlement struct {
	PositionPubkey string         `json:"position_pubkey"`
	Position       MarginPosition `json:"position"`
	TotalPnL       string         `json:"total_pnl"`
	UserLiquidity  string         `json:"user_liquidity"`
	State          string         `json:"state"`
	Created        *int64         `json:"created"`
}

type ClaimPositionSettlementResult struct {
	PositionPubkey string `json:"position_pubkey"`
	State          string `json:"state"`
}

type StartRedeemRequest struct {
	MarketConditionID string `json:"market_condition_id"`
}

type Redeem struct {
	Pubkey            string  `json:"pubkey"`
	MarketConditionID *string `json:"market_condition_id"`
	State             string  `json:"state"`
	Funds             string  `json:"funds"`
	OnchainFunds      string  `json:"onchain_funds"`
	YesShares         string  `json:"yes_shares"`
	NoShares          string  `json:"no_shares"`
	Message           *string `json:"message"`
	Created           *int64  `json:"created"`
}

func (c *clientImpl) Search(ctx context.Context, options SearchOptions) (*SearchResponse, error) {
	query := options.pageValues()
	setString(query, "q", options.Q)
	setString(query, "category", options.Category)
	setString(query, "state", options.State)
	setString(query, "sport", options.Sport)
	setString(query, "league", options.League)
	setInt64(query, "resolution_time_gt", options.ResolutionTimeGT)
	setInt64(query, "resolution_time_lt", options.ResolutionTimeLT)
	setString(query, "price_gte", options.PriceGTE)
	setString(query, "price_lte", options.PriceLTE)
	setBoolPtr(query, "leveraged", options.Leveraged)
	setString(query, "max_leverage_yes_gte", options.MaxLeverageYesGTE)
	setString(query, "max_leverage_yes_lte", options.MaxLeverageYesLTE)
	setString(query, "max_leverage_no_gte", options.MaxLeverageNoGTE)
	setString(query, "max_leverage_no_lte", options.MaxLeverageNoLTE)
	setString(query, "result_type", options.ResultType)
	setString(query, "sort", options.Sort)
	var data []SearchResult
	meta, err := c.do(ctx, http.MethodGet, "/search/", query, nil, false, &data)
	if err != nil {
		return nil, err
	}
	return &SearchResponse{Results: data, Meta: meta}, nil
}

func (c *clientImpl) ListSports(ctx context.Context) (*ListSportsResponse, error) {
	var data []Sport
	meta, err := c.do(ctx, http.MethodGet, "/sports/", nil, nil, false, &data)
	if err != nil {
		return nil, err
	}
	return &ListSportsResponse{Sports: data, Meta: meta}, nil
}

func (c *clientImpl) ListMarkets(ctx context.Context, options ListMarketsOptions) (*ListMarketsResponse, error) {
	query := options.pageValues()
	setString(query, "state", options.State)
	setString(query, "category", options.Category)
	setString(query, "sort", options.Sort)
	setString(query, "sport", options.Sport)
	setString(query, "league", options.League)
	var data []MarketSummary
	meta, err := c.do(ctx, http.MethodGet, "/markets/", query, nil, false, &data)
	if err != nil {
		return nil, err
	}
	return &ListMarketsResponse{Markets: data, Meta: meta}, nil
}

func (c *clientImpl) GetMarket(ctx context.Context, conditionID string) (*Market, error) {
	var data Market
	_, err := c.do(ctx, http.MethodGet, "/markets/"+pathEscape(conditionID)+"/", nil, nil, false, &data)
	return &data, err
}

func (c *clientImpl) GetMarketStats(ctx context.Context, conditionID string) (*MarketStats, error) {
	var data MarketStats
	_, err := c.do(ctx, http.MethodGet, "/markets/"+pathEscape(conditionID)+"/stats/", nil, nil, false, &data)
	return &data, err
}

func (c *clientImpl) GetMarketPrice(ctx context.Context, conditionID string, options GetMarketPriceOptions) (*MarketPrice, error) {
	query := make(url.Values)
	setBoolPtr(query, "is_yes", options.IsYes)
	var data MarketPrice
	_, err := c.do(ctx, http.MethodGet, "/markets/"+pathEscape(conditionID)+"/price/", query, nil, false, &data)
	return &data, err
}

func (c *clientImpl) GetMarketOrderBook(ctx context.Context, conditionID string, options GetMarketOrderBookOptions) (*MarketOrderBook, error) {
	query := make(url.Values)
	setInt(query, "depth", options.Depth)
	setBoolPtr(query, "is_yes", options.IsYes)
	var data MarketOrderBook
	_, err := c.do(ctx, http.MethodGet, "/markets/"+pathEscape(conditionID)+"/book/", query, nil, false, &data)
	return &data, err
}

func (c *clientImpl) GetMarketCandles(ctx context.Context, conditionID string, options GetMarketCandlesOptions) (*ListMarketCandlesResponse, error) {
	query := make(url.Values)
	setInt64(query, "start_time", options.StartTime)
	setInt64(query, "end_time", options.EndTime)
	setString(query, "interval", options.Interval)
	setBoolPtr(query, "is_yes", options.IsYes)
	var data []MarketCandle
	meta, err := c.do(ctx, http.MethodGet, "/markets/"+pathEscape(conditionID)+"/candles/", query, nil, false, &data)
	if err != nil {
		return nil, err
	}
	return &ListMarketCandlesResponse{Candles: data, Meta: meta}, nil
}

func (c *clientImpl) ListMarketTrades(ctx context.Context, conditionID string, options ListMarketTradesOptions) (*ListMarketTradesResponse, error) {
	var data []MarketTrade
	meta, err := c.do(ctx, http.MethodGet, "/markets/"+pathEscape(conditionID)+"/trades/", options.pageValues(), nil, false, &data)
	if err != nil {
		return nil, err
	}
	return &ListMarketTradesResponse{Trades: data, Meta: meta}, nil
}

func (c *clientImpl) ListMarketMarginActivity(ctx context.Context, conditionID string, options ListMarketMarginActivityOptions) (*ListMarketMarginActivityResponse, error) {
	query := options.pageValues()
	setString(query, "activity_type", options.ActivityType)
	var data []MarketMarginActivity
	meta, err := c.do(ctx, http.MethodGet, "/markets/"+pathEscape(conditionID)+"/margin-activity/", query, nil, false, &data)
	if err != nil {
		return nil, err
	}
	return &ListMarketMarginActivityResponse{Activities: data, Meta: meta}, nil
}

func (c *clientImpl) ListEvents(ctx context.Context, options ListEventsOptions) (*ListEventsResponse, error) {
	query := options.pageValues()
	setString(query, "state", options.State)
	setString(query, "category", options.Category)
	setString(query, "sort", options.Sort)
	setString(query, "sport", options.Sport)
	setString(query, "league", options.League)
	var data []Event
	meta, err := c.do(ctx, http.MethodGet, "/events/", query, nil, false, &data)
	if err != nil {
		return nil, err
	}
	return &ListEventsResponse{Events: data, Meta: meta}, nil
}

func (c *clientImpl) GetEvent(ctx context.Context, conditionID string) (*Event, error) {
	var data Event
	_, err := c.do(ctx, http.MethodGet, "/events/"+pathEscape(conditionID)+"/", nil, nil, false, &data)
	return &data, err
}

func (c *clientImpl) CreateAuthChallenge(ctx context.Context, request CreateAuthChallengeRequest) (*AuthChallenge, error) {
	var data AuthChallenge
	_, err := c.do(ctx, http.MethodPost, "/auth/keys/challenge/", nil, request, false, &data)
	return &data, err
}

func (c *clientImpl) CreateAPIKey(ctx context.Context, request CreateAPIKeyRequest) (*APIKeySecret, error) {
	var data APIKeySecret
	_, err := c.do(ctx, http.MethodPost, "/auth/keys/create/", nil, request, false, &data)
	return &data, err
}

func (c *clientImpl) CreateAPIKeyFromPrivateKey(ctx context.Context, privateKey string) (*APIKeySecret, error) {
	key, err := parseSolanaPrivateKey(privateKey)
	if err != nil {
		return nil, err
	}
	walletAddress := solanaWalletAddress(key)

	challenge, err := c.CreateAuthChallenge(ctx, CreateAuthChallengeRequest{WalletAddress: walletAddress})
	if err != nil {
		return nil, err
	}

	signature := ed25519.Sign(key, []byte(challenge.Message))
	return c.CreateAPIKey(ctx, CreateAPIKeyRequest{
		WalletAddress: walletAddress,
		Message:       challenge.Message,
		Signature:     hex.EncodeToString(signature),
		Nonce:         challenge.Nonce,
	})
}

func (c *clientImpl) ListAPIKeys(ctx context.Context) (*ListAPIKeysResponse, error) {
	var data []APIKey
	meta, err := c.do(ctx, http.MethodGet, "/auth/keys/", nil, nil, true, &data)
	if err != nil {
		return nil, err
	}
	return &ListAPIKeysResponse{Keys: data, Meta: meta}, nil
}

func (c *clientImpl) RevokeAPIKey(ctx context.Context, keyID string) (*APIKey, error) {
	var data APIKey
	_, err := c.do(ctx, http.MethodDelete, "/auth/keys/"+pathEscape(keyID)+"/", nil, nil, true, &data)
	return &data, err
}

func (c *clientImpl) CreateOrderDraft(ctx context.Context, request CreateOrderDraftRequest) (*DraftMessage, error) {
	var data DraftMessage
	_, err := c.do(ctx, http.MethodPost, "/orders/", nil, request, true, &data)
	return &data, err
}

func (c *clientImpl) SubmitOrder(ctx context.Context, pubkey string, request SubmitSignatureRequest) (*Order, error) {
	var data Order
	_, err := c.do(ctx, http.MethodPost, "/orders/"+pathEscape(pubkey)+"/submit/", nil, request, true, &data)
	return &data, err
}

func (c *clientImpl) GetOrder(ctx context.Context, pubkey string) (*Order, error) {
	var data Order
	_, err := c.do(ctx, http.MethodGet, "/orders/"+pathEscape(pubkey)+"/", nil, nil, true, &data)
	return &data, err
}

func (c *clientImpl) ListOrders(ctx context.Context, options ListOrdersOptions) (*ListOrdersResponse, error) {
	query := options.pageValues()
	setString(query, "market_condition_id", options.MarketConditionID)
	setString(query, "status", options.Status)
	setBoolPtr(query, "is_yes", options.IsYes)
	setString(query, "side", options.Side)
	var data []Order
	meta, err := c.do(ctx, http.MethodGet, "/orders/", query, nil, true, &data)
	if err != nil {
		return nil, err
	}
	return &ListOrdersResponse{Orders: data, Meta: meta}, nil
}

func (c *clientImpl) CreateCancelDraft(ctx context.Context, pubkey string) (*DraftMessage, error) {
	var data DraftMessage
	_, err := c.do(ctx, http.MethodPost, "/orders/"+pathEscape(pubkey)+"/cancel/", nil, nil, true, &data)
	return &data, err
}

func (c *clientImpl) SubmitOrderCancel(ctx context.Context, pubkey string, request SubmitSignatureRequest) (*Order, error) {
	var data Order
	_, err := c.do(ctx, http.MethodPost, "/orders/"+pathEscape(pubkey)+"/cancel/submit/", nil, request, true, &data)
	return &data, err
}

func (c *clientImpl) ListTrades(ctx context.Context, options ListTradesOptions) (*ListTradesResponse, error) {
	query := options.pageValues()
	setString(query, "market_condition_id", options.MarketConditionID)
	var data []Trade
	meta, err := c.do(ctx, http.MethodGet, "/trades/", query, nil, true, &data)
	if err != nil {
		return nil, err
	}
	return &ListTradesResponse{Trades: data, Meta: meta}, nil
}

func (c *clientImpl) GetAccountSummary(ctx context.Context) (*AccountSummary, error) {
	var data AccountSummary
	_, err := c.do(ctx, http.MethodGet, "/account/", nil, nil, true, &data)
	return &data, err
}

func (c *clientImpl) GetAccountPnL(ctx context.Context, options GetAccountPnLOptions) (*AccountPnL, error) {
	query := make(url.Values)
	setString(query, "market_condition_id", options.MarketConditionID)
	setString(query, "event_condition_id", options.EventConditionID)
	var data AccountPnL
	_, err := c.do(ctx, http.MethodGet, "/account/pnl/", query, nil, true, &data)
	return &data, err
}

func (c *clientImpl) ListAccountAssets(ctx context.Context, options ListAccountAssetsOptions) (*ListAccountAssetsResponse, error) {
	query := options.pageValues()
	setString(query, "market_condition_id", options.MarketConditionID)
	var data []AccountAsset
	meta, err := c.do(ctx, http.MethodGet, "/account/assets/", query, nil, true, &data)
	if err != nil {
		return nil, err
	}
	return &ListAccountAssetsResponse{Assets: data, Meta: meta}, nil
}

func (c *clientImpl) EstimateMarginPosition(ctx context.Context, options EstimateMarginPositionOptions) (*MarginPositionEstimate, error) {
	query := make(url.Values)
	setString(query, "market_condition_id", options.MarketConditionID)
	setString(query, "funds", options.Funds)
	setBoolPtr(query, "is_yes", options.IsYes)
	setFloatPtr(query, "leverage", options.Leverage)
	var data MarginPositionEstimate
	_, err := c.do(ctx, http.MethodGet, "/margin/positions/estimate/", query, nil, false, &data)
	return &data, err
}

func (c *clientImpl) CreatePositionRequest(ctx context.Context, request CreatePositionRequestRequest) (*PositionRequest, error) {
	var data PositionRequest
	_, err := c.do(ctx, http.MethodPost, "/margin/positions/requests/", nil, request, true, &data)
	return &data, err
}

func (c *clientImpl) SubmitPositionRequest(ctx context.Context, pubkey string, request SubmitSignatureRequest) (*PositionRequest, error) {
	var data PositionRequest
	_, err := c.do(ctx, http.MethodPost, "/margin/positions/requests/"+pathEscape(pubkey)+"/submit/", nil, request, true, &data)
	return &data, err
}

func (c *clientImpl) GetPositionRequest(ctx context.Context, pubkey string) (*PositionRequest, error) {
	var data PositionRequest
	_, err := c.do(ctx, http.MethodGet, "/margin/positions/requests/"+pathEscape(pubkey)+"/", nil, nil, true, &data)
	return &data, err
}

func (c *clientImpl) CancelPositionRequest(ctx context.Context, pubkey string) (*PositionRequest, error) {
	var data PositionRequest
	_, err := c.do(ctx, http.MethodDelete, "/margin/positions/requests/"+pathEscape(pubkey)+"/", nil, nil, true, &data)
	return &data, err
}

func (c *clientImpl) ListPositionRequests(ctx context.Context, options ListPositionRequestsOptions) (*ListPositionRequestsResponse, error) {
	query := options.pageValues()
	setString(query, "market_condition_id", options.MarketConditionID)
	setString(query, "states", options.States)
	setBoolPtr(query, "is_yes", options.IsYes)
	setFloatPtr(query, "leverage", options.Leverage)
	setString(query, "sort", options.Sort)
	var data []PositionRequest
	meta, err := c.do(ctx, http.MethodGet, "/margin/positions/requests/", query, nil, true, &data)
	if err != nil {
		return nil, err
	}
	return &ListPositionRequestsResponse{Requests: data, Meta: meta}, nil
}

func (c *clientImpl) ListMarginPositions(ctx context.Context, options ListMarginPositionsOptions) (*ListMarginPositionsResponse, error) {
	query := options.pageValues()
	setString(query, "market_condition_id", options.MarketConditionID)
	setString(query, "event_condition_id", options.EventConditionID)
	setBoolPtr(query, "is_closed", options.IsClosed)
	setString(query, "sort", options.Sort)
	var data []MarginPosition
	meta, err := c.do(ctx, http.MethodGet, "/margin/positions/", query, nil, true, &data)
	if err != nil {
		return nil, err
	}
	return &ListMarginPositionsResponse{Positions: data, Meta: meta}, nil
}

func (c *clientImpl) GetMarginPosition(ctx context.Context, pubkey string) (*MarginPosition, error) {
	var data MarginPosition
	_, err := c.do(ctx, http.MethodGet, "/margin/positions/"+pathEscape(pubkey)+"/", nil, nil, true, &data)
	return &data, err
}

func (c *clientImpl) SetTPSL(ctx context.Context, pubkey string, request SetTPSLRequest) (*TPSL, error) {
	var data TPSL
	_, err := c.do(ctx, http.MethodPost, "/margin/positions/"+pathEscape(pubkey)+"/tp-sl/", nil, request, true, &data)
	return &data, err
}

func (c *clientImpl) DeleteTPSL(ctx context.Context, pubkey string) (*TPSL, error) {
	var data TPSL
	_, err := c.do(ctx, http.MethodDelete, "/margin/positions/"+pathEscape(pubkey)+"/tp-sl/", nil, nil, true, &data)
	return &data, err
}

func (c *clientImpl) CloseMarginPosition(ctx context.Context, pubkey string, options CloseMarginPositionOptions) (*CloseMarginPositionResult, error) {
	query := make(url.Values)
	setFloatPtr(query, "price", options.Price)
	var data CloseMarginPositionResult
	_, err := c.do(ctx, http.MethodDelete, "/margin/positions/"+pathEscape(pubkey)+"/", query, nil, true, &data)
	return &data, err
}

func (c *clientImpl) ListMarginSettlements(ctx context.Context, options ListMarginSettlementsOptions) (*ListMarginSettlementsResponse, error) {
	query := options.pageValues()
	setString(query, "position_pubkey", options.PositionPubkey)
	setString(query, "market_condition_id", options.MarketConditionID)
	setString(query, "states", options.States)
	setString(query, "market_state", options.MarketState)
	setFloatPtr(query, "position_leverage", options.PositionLeverage)
	setString(query, "sort", options.Sort)
	var data []MarginSettlement
	meta, err := c.do(ctx, http.MethodGet, "/margin/positions/settlements/", query, nil, true, &data)
	if err != nil {
		return nil, err
	}
	return &ListMarginSettlementsResponse{Settlements: data, Meta: meta}, nil
}

func (c *clientImpl) ClaimPositionSettlement(ctx context.Context, pubkey string) (*ClaimPositionSettlementResult, error) {
	var data ClaimPositionSettlementResult
	_, err := c.do(ctx, http.MethodPost, "/margin/positions/"+pathEscape(pubkey)+"/settlements/", nil, nil, true, &data)
	return &data, err
}

func (c *clientImpl) ListRedeems(ctx context.Context, options ListRedeemsOptions) (*ListRedeemsResponse, error) {
	query := options.pageValues()
	setString(query, "market_condition_id", options.MarketConditionID)
	setString(query, "state", options.State)
	var data []Redeem
	meta, err := c.do(ctx, http.MethodGet, "/redeems/", query, nil, true, &data)
	if err != nil {
		return nil, err
	}
	return &ListRedeemsResponse{Redeems: data, Meta: meta}, nil
}

func (c *clientImpl) StartRedeem(ctx context.Context, request StartRedeemRequest) (*DraftMessage, error) {
	var data DraftMessage
	_, err := c.do(ctx, http.MethodPost, "/redeems/", nil, request, true, &data)
	return &data, err
}

func (c *clientImpl) GetRedeem(ctx context.Context, pubkey string) (*Redeem, error) {
	var data Redeem
	_, err := c.do(ctx, http.MethodGet, "/redeems/"+pathEscape(pubkey)+"/", nil, nil, true, &data)
	return &data, err
}

func (c *clientImpl) SubmitRedeem(ctx context.Context, pubkey string, request SubmitSignatureRequest) (*Redeem, error) {
	var data Redeem
	_, err := c.do(ctx, http.MethodPost, "/redeems/"+pathEscape(pubkey)+"/submit/", nil, request, true, &data)
	return &data, err
}

func (c *clientImpl) do(ctx context.Context, method string, path string, query url.Values, body any, authRequired bool, out any) (EnvelopeMeta, error) {
	if authRequired {
		if strings.TrimSpace(c.config.APIKey) == "" || strings.TrimSpace(c.config.APISecret) == "" {
			return EnvelopeMeta{}, errors.New("worm api key and secret are required for authenticated requests")
		}
	}

	endpoint, err := c.buildURL(path, query)
	if err != nil {
		return EnvelopeMeta{}, err
	}

	var rawBody []byte
	if body != nil {
		rawBody, err = json.Marshal(body)
		if err != nil {
			return EnvelopeMeta{}, fmt.Errorf("failed to encode worm request body: %w", err)
		}
	}

	for attempt := 0; ; attempt++ {
		var requestBody io.Reader = http.NoBody
		if body != nil {
			requestBody = bytes.NewReader(rawBody)
		}
		req, err := http.NewRequestWithContext(ctx, method, endpoint.String(), requestBody)
		if err != nil {
			return EnvelopeMeta{}, fmt.Errorf("failed to create worm request: %w", err)
		}
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		if authRequired {
			c.sign(req, method, rawBody)
		}

		if err := c.rateLimiter.Wait(ctx); err != nil {
			return EnvelopeMeta{}, fmt.Errorf("wait worm rate limit: %w", err)
		}

		resp, err := c.client.Do(req)
		if err != nil {
			return EnvelopeMeta{}, fmt.Errorf("failed to send worm request: %w", err)
		}

		if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
			requestErr := decodeHTTPError(resp)
			_ = resp.Body.Close()
			if method != http.MethodGet || resp.StatusCode != http.StatusTooManyRequests || attempt >= rateLimitMaxRetries {
				return EnvelopeMeta{}, requestErr
			}
			if err := waitWithContext(ctx, c.rateLimitRetryDelay(requestErr, resp.Header, attempt)); err != nil {
				return EnvelopeMeta{}, err
			}
			continue
		}

		var envelope responseEnvelope
		decodeErr := json.NewDecoder(resp.Body).Decode(&envelope)
		_ = resp.Body.Close()
		if decodeErr != nil {
			return EnvelopeMeta{}, fmt.Errorf("failed to decode worm response: %w", decodeErr)
		}
		if envelope.Error != nil {
			return EnvelopeMeta{}, envelope.Error
		}
		if out != nil && len(envelope.Data) > 0 && string(envelope.Data) != "null" {
			if err := json.Unmarshal(envelope.Data, out); err != nil {
				return EnvelopeMeta{}, fmt.Errorf("failed to decode worm response data: %w", err)
			}
		}
		return envelope.Meta, nil
	}
}

func (c *clientImpl) rateLimitRetryDelay(err error, header http.Header, attempt int) time.Duration {
	if delay := parseRetryAfter(header.Get("Retry-After")); delay > 0 {
		return delay
	}
	if reset := int64(parseHeaderInt(header.Get("X-RateLimit-Reset"))); reset > 0 {
		if delay := time.Unix(reset, 0).Sub(c.config.Now()); delay > 0 {
			return delay
		}
	}
	var wormErr *Error
	if errors.As(err, &wormErr) {
		if wormErr.RetryAfter > 0 {
			return wormErr.RetryAfter
		}
		if wormErr.RateLimitReset > 0 {
			delay := time.Unix(wormErr.RateLimitReset, 0).Sub(c.config.Now())
			if delay > 0 {
				return delay
			}
		}
	}
	backoff := rateLimitBackoffBase << attempt
	jitter := time.Duration(rand.Int64N(int64(rateLimitBackoffJitterMax) + 1))
	return backoff + jitter
}

func waitWithContext(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func (c *clientImpl) buildURL(path string, query url.Values) (*url.URL, error) {
	endpoint, err := url.Parse(strings.TrimRight(c.config.BaseURL, "/") + path)
	if err != nil {
		return nil, fmt.Errorf("failed to parse worm request url: %w", err)
	}
	if len(query) > 0 {
		endpoint.RawQuery = query.Encode()
	}
	return endpoint, nil
}

func parseSolanaPrivateKey(input string) (ed25519.PrivateKey, error) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return nil, errors.New("solana private key is required")
	}

	var raw []byte
	var err error
	if strings.HasPrefix(trimmed, "[") {
		if err := json.Unmarshal([]byte(trimmed), &raw); err != nil {
			return nil, fmt.Errorf("invalid solana private key json: %w", err)
		}
	} else if isHexEncodedKey(trimmed) {
		raw, err = hex.DecodeString(trimmed)
		if err != nil {
			return nil, fmt.Errorf("invalid solana private key hex: %w", err)
		}
	} else {
		raw, err = base58.Decode(trimmed)
		if err != nil {
			return nil, fmt.Errorf("invalid solana private key base58: %w", err)
		}
	}

	switch len(raw) {
	case ed25519.SeedSize:
		return ed25519.NewKeyFromSeed(raw), nil
	case ed25519.PrivateKeySize:
		key := ed25519.PrivateKey(raw)
		derived := ed25519.NewKeyFromSeed(key.Seed())
		if !bytes.Equal(derived, key) {
			return nil, errors.New("invalid solana private key: public key does not match seed")
		}
		return key, nil
	default:
		return nil, fmt.Errorf("invalid solana private key length: got %d bytes, want 32-byte seed or 64-byte keypair", len(raw))
	}
}

func isHexEncodedKey(input string) bool {
	if len(input) != ed25519.SeedSize*2 && len(input) != ed25519.PrivateKeySize*2 {
		return false
	}
	for _, r := range input {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') && (r < 'A' || r > 'F') {
			return false
		}
	}
	return true
}

func solanaWalletAddress(privateKey ed25519.PrivateKey) string {
	publicKey := privateKey.Public().(ed25519.PublicKey)
	return base58.Encode(publicKey)
}

func (c *clientImpl) sign(req *http.Request, method string, rawBody []byte) {
	timestamp := strconv.FormatInt(c.config.Now().Unix(), 10)
	payload := timestamp + method + req.URL.RequestURI() + string(rawBody)
	mac := hmac.New(sha256.New, []byte(c.config.APISecret))
	_, _ = mac.Write([]byte(payload))
	req.Header.Set(headerAPIKey, c.config.APIKey)
	req.Header.Set(headerTimestamp, timestamp)
	req.Header.Set(headerSignature, hex.EncodeToString(mac.Sum(nil)))
}

type responseEnvelope struct {
	Data  json.RawMessage `json:"data"`
	Meta  EnvelopeMeta    `json:"meta"`
	Error *Error          `json:"error"`
}

func decodeHTTPError(resp *http.Response) error {
	body, err := io.ReadAll(io.LimitReader(resp.Body, errorBodyLimit))
	if err != nil {
		return fmt.Errorf("worm http request failed with status %s and unreadable body: %w", resp.Status, err)
	}
	var envelope responseEnvelope
	if err := json.Unmarshal(body, &envelope); err == nil && envelope.Error != nil {
		envelope.Error.Status = resp.Status
		envelope.Error.StatusCode = resp.StatusCode
		envelope.Error.RetryAfter = parseRetryAfter(resp.Header.Get("Retry-After"))
		envelope.Error.RateLimitLimit = parseHeaderInt(resp.Header.Get("X-RateLimit-Limit"))
		envelope.Error.RateLimitRemaining = parseHeaderInt(resp.Header.Get("X-RateLimit-Remaining"))
		envelope.Error.RateLimitReset = int64(parseHeaderInt(resp.Header.Get("X-RateLimit-Reset")))
		return envelope.Error
	}
	return fmt.Errorf("worm http request failed with status %s: %s", resp.Status, string(body))
}

func parseRetryAfter(value string) time.Duration {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0
	}
	if seconds, err := strconv.Atoi(value); err == nil && seconds > 0 {
		return time.Duration(seconds) * time.Second
	}
	if retryAt, err := http.ParseTime(value); err == nil {
		duration := time.Until(retryAt)
		if duration > 0 {
			return duration
		}
	}
	return 0
}

func parseHeaderInt(value string) int {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0
	}
	num, err := strconv.Atoi(value)
	if err != nil {
		return 0
	}
	return num
}

func (p PageOptions) pageValues() url.Values {
	query := make(url.Values)
	setInt(query, "limit", p.Limit)
	setString(query, "cursor", p.Cursor)
	return query
}

func pathEscape(value string) string {
	return url.PathEscape(value)
}

func setString(query url.Values, key string, value string) {
	if strings.TrimSpace(value) != "" {
		query.Set(key, value)
	}
}

func setInt(query url.Values, key string, value int) {
	if value > 0 {
		query.Set(key, strconv.Itoa(value))
	}
}

func setInt64(query url.Values, key string, value int64) {
	if value > 0 {
		query.Set(key, strconv.FormatInt(value, 10))
	}
}

func setBoolPtr(query url.Values, key string, value *bool) {
	if value != nil {
		query.Set(key, strconv.FormatBool(*value))
	}
}

func setFloatPtr(query url.Values, key string, value *float64) {
	if value != nil {
		query.Set(key, strconv.FormatFloat(*value, 'f', -1, 64))
	}
}
