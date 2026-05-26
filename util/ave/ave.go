package ave

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	DefaultBaseURL = "https://prod.ave-api.com"
	DefaultTimeout = 60 * time.Second

	errorBodyLimit = 4096
)

type Client interface {
	GetTokenDetail(ctx context.Context, tokenID string) (*TokenDetailResponse, error)
}

type Config struct {
	BaseURL string
	APIKey  string
	Timeout time.Duration
}

type TokenDetailResponse struct {
	Status   int             `json:"status"`    // API result status; confirms whether the token detail request succeeded.
	Msg      string          `json:"msg"`       // API status message; useful for diagnostics when a request fails.
	DataType int             `json:"data_type"` // Ave payload type marker; useful for interpreting vendor response variants.
	Data     TokenDetailData `json:"data"`      // Token detail payload; contains the token, liquidity pairs, and audit summary.
}

type TokenDetailData struct {
	Token     Token  `json:"token"`      // Primary token attributes; useful for display, market context, and risk review.
	Pairs     []Pair `json:"pairs"`      // Liquidity pair list; useful for pool, volume, and fake-pair analysis.
	IsAudited bool   `json:"is_audited"` // Audit summary flag; useful as a high-level trust and risk signal.
}

type Token struct {
	Total               string `json:"total"`                   // Reported token supply; useful for valuation and supply distribution context.
	LaunchPrice         string `json:"launch_price"`            // Initial token price; useful for comparing launch valuation with current price.
	CurrentPriceETH     string `json:"current_price_eth"`       // Current token price in ETH; useful for native market valuation.
	CurrentPriceUSD     string `json:"current_price_usd"`       // Current token price in USD; useful for display, valuation, and market monitoring.
	PriceChange1D       string `json:"price_change_1d"`         // One-day price movement; useful for short-term trend checks.
	PriceChange24H      string `json:"price_change_24h"`        // Twenty-four-hour price movement; useful for daily volatility review.
	PriceChange1H       string `json:"price_change_1h"`         // One-hour price movement; useful for detecting recent momentum.
	LockAmount          string `json:"lock_amount"`             // Locked token amount; useful for liquidity and supply-lock analysis.
	BurnAmount          string `json:"burn_amount"`             // Burned token amount; useful for circulating supply assessment.
	OtherAmount         string `json:"other_amount"`            // Token amount outside standard lock or burn buckets; useful for supply accounting.
	TxAmount24H         string `json:"tx_amount_24h"`           // Token quantity traded in 24 hours; useful for activity analysis.
	TxVolumeU24H        string `json:"tx_volume_u_24h"`         // USD trading volume in 24 hours; useful for liquidity and activity review.
	LockedPercent       string `json:"locked_percent"`          // Locked supply ratio; useful for liquidity-lock and holder-risk assessment.
	MarketCap           string `json:"market_cap"`              // Market capitalization; useful for valuation and ranking.
	FDV                 string `json:"fdv"`                     // Fully diluted valuation; useful for comparing valuation against total supply.
	TVL                 string `json:"tvl"`                     // Total value locked across pools; useful for liquidity depth assessment.
	MainPairTVL         string `json:"main_pair_tvl"`           // Total value locked in the main pair; useful for primary liquidity checks.
	TokenPriceChange5M  string `json:"token_price_change_5m"`   // Five-minute token price movement; useful for detecting sudden moves.
	TokenPriceChange1H  string `json:"token_price_change_1h"`   // One-hour token price movement; useful for recent trend analysis.
	TokenPriceChange4H  string `json:"token_price_change_4h"`   // Four-hour token price movement; useful for intraday trend analysis.
	TokenPriceChange24H string `json:"token_price_change_24h"`  // Twenty-four-hour token price movement; useful for daily trend analysis.
	TokenTxVolumeUSD5M  string `json:"token_tx_volume_usd_5m"`  // Five-minute USD volume; useful for spotting immediate activity spikes.
	TokenTxVolumeUSD1H  string `json:"token_tx_volume_usd_1h"`  // One-hour USD volume; useful for recent liquidity and demand checks.
	TokenTxVolumeUSD4H  string `json:"token_tx_volume_usd_4h"`  // Four-hour USD volume; useful for intraday activity review.
	TokenTxVolumeUSD24H string `json:"token_tx_volume_usd_24h"` // Twenty-four-hour USD volume; useful for daily market activity analysis.
	TokenBuyVolumeU5M   string `json:"token_buy_volume_u_5m"`   // Five-minute buy volume in USD; useful for detecting near-term buying pressure.
	TokenSellVolumeU5M  string `json:"token_sell_volume_u_5m"`  // Five-minute sell volume in USD; useful for detecting near-term selling pressure.
	Token               string `json:"token"`                   // Token contract or asset identifier; useful as the canonical external token key.
	Chain               string `json:"chain"`                   // Chain identifier; useful for routing, display, and cross-chain disambiguation.
	Decimal             int    `json:"decimal"`                 // Token decimal precision; useful for amount normalization and display.
	Name                string `json:"name"`                    // Token display name; useful for UI identity.
	Symbol              string `json:"symbol"`                  // Token ticker symbol; useful for compact UI identity.
	Holders             int    `json:"holders"`                 // Holder count; useful for adoption and concentration risk context.
	Appendix            string `json:"appendix"`                // Vendor-provided extra metadata; useful when Ave includes auxiliary token details.
	RiskLevel           int    `json:"risk_level"`              // Vendor risk level; useful as a compact risk summary.
	LogoURL             string `json:"logo_url"`                // Token logo URL; useful for recognizable project display.
	RiskInfo            string `json:"risk_info"`               // Vendor risk explanation payload; useful for presenting risk details.
	RiskScore           string `json:"risk_score"`              // Vendor risk score; useful for ranking and comparing risk severity.
	LaunchAt            int64  `json:"launch_at"`               // Token launch timestamp; useful for project-age and lifecycle analysis.
	CreatedAt           int64  `json:"created_at"`              // Ave record creation timestamp; useful for source data freshness context.
	TxCount24H          int    `json:"tx_count_24h"`            // Twenty-four-hour transaction count; useful for activity and liquidity checks.
	LockPlatform        string `json:"lock_platform"`           // Liquidity or token lock platform; useful for validating lock provenance.
	IsMintable          string `json:"is_mintable"`             // Mintability flag from Ave; useful for token supply risk review.
	UpdatedAt           int64  `json:"updated_at"`              // Vendor update timestamp; useful for judging market data freshness.
	MainPair            string `json:"main_pair"`               // Primary liquidity pair identifier; useful for linking token data to its main pool.
	HasMintMethod       bool   `json:"has_mint_method"`         // Indicates whether mint logic is detected; useful as a supply inflation risk signal.
	IsLPNotLocked       bool   `json:"is_lp_not_locked"`        // Indicates unlocked liquidity; useful as a liquidity withdrawal risk signal.
	HasNotRenounced     bool   `json:"has_not_renounced"`       // Indicates ownership is not renounced; useful for admin-control risk review.
	HasNotAudited       bool   `json:"has_not_audited"`         // Indicates missing audit coverage; useful as a trust and risk signal.
	HasNotOpenSource    bool   `json:"has_not_open_source"`     // Indicates missing open-source contract code; useful for transparency risk review.
	IsInBlacklist       bool   `json:"is_in_blacklist"`         // Indicates vendor blacklist presence; useful as a high-priority risk signal.
	IsHoneypot          bool   `json:"is_honeypot"`             // Indicates whether the token behaves like a honeypot; high-value risk signal.
	AveRiskLevel        int    `json:"ave_risk_level"`          // Ave-specific risk level; useful for vendor risk classification.
}

type Pair struct {
	Reserve0       string `json:"reserve0"`         // Reserve amount for token0; useful for pool depth and price impact analysis.
	Reserve1       string `json:"reserve1"`         // Reserve amount for token1; useful for pool depth and price impact analysis.
	Token0PriceETH string `json:"token0_price_eth"` // Token0 price in ETH; useful for native valuation inside this pair.
	Token0PriceUSD string `json:"token0_price_usd"` // Token0 price in USD; useful for display and pair-level valuation.
	Token1PriceETH string `json:"token1_price_eth"` // Token1 price in ETH; useful for native valuation inside this pair.
	Token1PriceUSD string `json:"token1_price_usd"` // Token1 price in USD; useful for display and quote-token valuation.
	PriceChange    string `json:"price_change"`     // Pair price movement; useful for pool-level trend checks.
	PriceChange24H string `json:"price_change_24h"` // Twenty-four-hour pair price movement; useful for daily pair volatility review.
	PriceChange1H  string `json:"price_change_1h"`  // One-hour pair price movement; useful for recent pool momentum.
	VolumeU        string `json:"volume_u"`         // Pair trading volume in USD; useful for liquidity and activity analysis.
	LowU           string `json:"low_u"`            // Low price in USD for the period; useful for range and volatility checks.
	HighU          string `json:"high_u"`           // High price in USD for the period; useful for range and volatility checks.
	Fee            string `json:"fee"`              // Pair fee value when provided; useful for exchange and pool cost context.
	TotalSupply    string `json:"total_supply"`     // Pair token total supply when provided; useful for LP token analysis.
	TxAmount       string `json:"tx_amount"`        // Token amount traded through this pair; useful for activity analysis.
	Pair           string `json:"pair"`             // Pair contract or pool identifier; useful for linking liquidity data to the pool.
	Chain          string `json:"chain"`            // Chain identifier for the pair; useful for routing and cross-chain disambiguation.
	AMM            string `json:"amm"`              // AMM or exchange name; useful for identifying the liquidity venue.
	Token0Address  string `json:"token0_address"`   // Token0 address; useful for identifying pair composition.
	Token0Symbol   string `json:"token0_symbol"`    // Token0 symbol; useful for compact pair display.
	Token0Decimal  int    `json:"token0_decimal"`   // Token0 decimal precision; useful for amount normalization.
	Token1Address  string `json:"token1_address"`   // Token1 address; useful for identifying pair composition.
	Token1Symbol   string `json:"token1_symbol"`    // Token1 symbol; useful for compact pair display.
	Token1Decimal  int    `json:"token1_decimal"`   // Token1 decimal precision; useful for amount normalization.
	TargetToken    string `json:"target_token"`     // Token being analyzed in this pair; useful for mapping pair data back to the project token.
	PriceChange1D  string `json:"price_change_1d"`  // One-day pair price movement; useful for daily trend checks.
	CreatedAt      int64  `json:"created_at"`       // Pair creation timestamp; useful for liquidity-age analysis.
	TxCount        int    `json:"tx_count"`         // Pair transaction count; useful for activity and wash-trading context.
	UpdatedAt      int64  `json:"updated_at"`       // Vendor pair update timestamp; useful for judging data freshness.
	MarketCap      string `json:"market_cap"`       // Pair-implied market capitalization; useful for pair-level valuation checks.
	FDV            string `json:"fdv"`              // Pair-implied fully diluted valuation; useful for valuation comparison.
	IsFake         bool   `json:"is_fake"`          // Indicates whether Ave flags the pair as fake; high-value liquidity risk signal.
}

type clientImpl struct {
	config Config
	client *http.Client
}

func NewClient(config Config) (Client, error) {
	config = config.withDefaults()
	if strings.TrimSpace(config.APIKey) == "" {
		return nil, errors.New("ave api key is required")
	}
	return &clientImpl{
		config: config,
		client: &http.Client{Timeout: config.Timeout},
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
	if c.Timeout <= 0 {
		c.Timeout = DefaultTimeout
	}
	return c
}

func (c *clientImpl) GetTokenDetail(ctx context.Context, tokenID string) (*TokenDetailResponse, error) {
	tokenID = strings.TrimSpace(tokenID)
	if tokenID == "" {
		return nil, errors.New("ave token id is required")
	}

	endpoint := strings.TrimRight(c.config.BaseURL, "/") + "/v2/tokens/" + url.PathEscape(tokenID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create ave token detail request: %w", err)
	}
	req.Header.Set("X-API-KEY", c.config.APIKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send ave token detail request: %w", err)
	}
	defer resp.Body.Close()

	if err := ensureHTTPSuccess(resp, "token detail request"); err != nil {
		return nil, err
	}

	var detail TokenDetailResponse
	if err := json.NewDecoder(resp.Body).Decode(&detail); err != nil {
		return nil, fmt.Errorf("failed to decode ave token detail response: %w", err)
	}
	if detail.Status != 1 {
		return nil, fmt.Errorf("ave token detail request failed: %s", detail.Msg)
	}
	return &detail, nil
}

func ensureHTTPSuccess(resp *http.Response, operation string) error {
	if resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusMultipleChoices {
		return nil
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, errorBodyLimit))
	if err != nil {
		return fmt.Errorf("ave %s failed with status %s and unreadable body: %w", operation, resp.Status, err)
	}
	return fmt.Errorf("ave %s failed with status %s: %s", operation, resp.Status, string(body))
}
