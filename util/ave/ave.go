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
	Status   int             `json:"status"`
	Msg      string          `json:"msg"`
	DataType int             `json:"data_type"`
	Data     TokenDetailData `json:"data"`
}

type TokenDetailData struct {
	Token     Token  `json:"token"`
	Pairs     []Pair `json:"pairs"`
	IsAudited bool   `json:"is_audited"`
}

type Token struct {
	Total               string `json:"total"`
	LaunchPrice         string `json:"launch_price"`
	CurrentPriceETH     string `json:"current_price_eth"`
	CurrentPriceUSD     string `json:"current_price_usd"`
	PriceChange1D       string `json:"price_change_1d"`
	PriceChange24H      string `json:"price_change_24h"`
	PriceChange1H       string `json:"price_change_1h"`
	LockAmount          string `json:"lock_amount"`
	BurnAmount          string `json:"burn_amount"`
	OtherAmount         string `json:"other_amount"`
	TxAmount24H         string `json:"tx_amount_24h"`
	TxVolumeU24H        string `json:"tx_volume_u_24h"`
	LockedPercent       string `json:"locked_percent"`
	MarketCap           string `json:"market_cap"`
	FDV                 string `json:"fdv"`
	TVL                 string `json:"tvl"`
	MainPairTVL         string `json:"main_pair_tvl"`
	TokenPriceChange5M  string `json:"token_price_change_5m"`
	TokenPriceChange1H  string `json:"token_price_change_1h"`
	TokenPriceChange4H  string `json:"token_price_change_4h"`
	TokenPriceChange24H string `json:"token_price_change_24h"`
	TokenTxVolumeUSD5M  string `json:"token_tx_volume_usd_5m"`
	TokenTxVolumeUSD1H  string `json:"token_tx_volume_usd_1h"`
	TokenTxVolumeUSD4H  string `json:"token_tx_volume_usd_4h"`
	TokenTxVolumeUSD24H string `json:"token_tx_volume_usd_24h"`
	TokenBuyVolumeU5M   string `json:"token_buy_volume_u_5m"`
	TokenSellVolumeU5M  string `json:"token_sell_volume_u_5m"`
	Token               string `json:"token"`
	Chain               string `json:"chain"`
	Decimal             int    `json:"decimal"`
	Name                string `json:"name"`
	Symbol              string `json:"symbol"`
	Holders             int    `json:"holders"`
	Appendix            string `json:"appendix"`
	RiskLevel           int    `json:"risk_level"`
	LogoURL             string `json:"logo_url"`
	RiskInfo            string `json:"risk_info"`
	RiskScore           string `json:"risk_score"`
	LaunchAt            int64  `json:"launch_at"`
	CreatedAt           int64  `json:"created_at"`
	TxCount24H          int    `json:"tx_count_24h"`
	LockPlatform        string `json:"lock_platform"`
	IsMintable          string `json:"is_mintable"`
	UpdatedAt           int64  `json:"updated_at"`
	MainPair            string `json:"main_pair"`
	HasMintMethod       bool   `json:"has_mint_method"`
	IsLPNotLocked       bool   `json:"is_lp_not_locked"`
	HasNotRenounced     bool   `json:"has_not_renounced"`
	HasNotAudited       bool   `json:"has_not_audited"`
	HasNotOpenSource    bool   `json:"has_not_open_source"`
	IsInBlacklist       bool   `json:"is_in_blacklist"`
	IsHoneypot          bool   `json:"is_honeypot"`
	AveRiskLevel        int    `json:"ave_risk_level"`
}

type Pair struct {
	Reserve0       string `json:"reserve0"`
	Reserve1       string `json:"reserve1"`
	Token0PriceETH string `json:"token0_price_eth"`
	Token0PriceUSD string `json:"token0_price_usd"`
	Token1PriceETH string `json:"token1_price_eth"`
	Token1PriceUSD string `json:"token1_price_usd"`
	PriceChange    string `json:"price_change"`
	PriceChange24H string `json:"price_change_24h"`
	PriceChange1H  string `json:"price_change_1h"`
	VolumeU        string `json:"volume_u"`
	LowU           string `json:"low_u"`
	HighU          string `json:"high_u"`
	Fee            string `json:"fee"`
	TotalSupply    string `json:"total_supply"`
	TxAmount       string `json:"tx_amount"`
	Pair           string `json:"pair"`
	Chain          string `json:"chain"`
	AMM            string `json:"amm"`
	Token0Address  string `json:"token0_address"`
	Token0Symbol   string `json:"token0_symbol"`
	Token0Decimal  int    `json:"token0_decimal"`
	Token1Address  string `json:"token1_address"`
	Token1Symbol   string `json:"token1_symbol"`
	Token1Decimal  int    `json:"token1_decimal"`
	TargetToken    string `json:"target_token"`
	PriceChange1D  string `json:"price_change_1d"`
	CreatedAt      int64  `json:"created_at"`
	TxCount        int    `json:"tx_count"`
	UpdatedAt      int64  `json:"updated_at"`
	MarketCap      string `json:"market_cap"`
	FDV            string `json:"fdv"`
	IsFake         bool   `json:"is_fake"`
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
