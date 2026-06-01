package polymarket

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const DefaultCLOBBaseURL = "https://clob.polymarket.com"

// CLOBClient is a typed Polymarket CLOB Market Data API client.
type CLOBClient interface {
	GetOrderBook(ctx context.Context, tokenID string) (*OrderBookSummary, error)
	GetOrderBooks(ctx context.Context, requests []CLOBBookRequest) ([]OrderBookSummary, error)

	// CLOB markets (read-only)
	GetMarketByToken(ctx context.Context, tokenID string) (*CLOBMarketByTokenResponse, error)
	GetCLOBMarketInfo(ctx context.Context, conditionID string) (*CLOBMarketInfo, error)
	GetPricesHistory(ctx context.Context, options GetCLOBPricesHistoryOptions) (*CLOBPricesHistoryResponse, error)
	GetBatchPricesHistory(ctx context.Context, request CLOBBatchPricesHistoryRequest) (*CLOBBatchPricesHistoryResponse, error)
	ListSimplifiedMarkets(ctx context.Context, nextCursor string) (*CLOBMarketsPage, error)
	ListSamplingMarkets(ctx context.Context, nextCursor string) (*CLOBMarketsPage, error)
	ListSamplingSimplifiedMarkets(ctx context.Context, nextCursor string) (*CLOBMarketsPage, error)
	GetCurrentRebatedFees(ctx context.Context, options GetCurrentRebatedFeesOptions) ([]CLOBRebatedFee, error)

	GetMidpointPrice(ctx context.Context, tokenID string) (*CLOBMidpointPriceResponse, error)
	GetMidpointPrices(ctx context.Context, tokenIDs []string) (map[string]string, error)
	GetMidpointPricesByBody(ctx context.Context, requests []CLOBBookRequest) (map[string]string, error)

	GetMarketPrice(ctx context.Context, tokenID, side string) (*CLOBPriceResponse, error)
	GetMarketPrices(ctx context.Context, tokenIDs []string, sides []string) (map[string]map[string]string, error)
	GetMarketPricesByBody(ctx context.Context, requests []CLOBBookRequest) (map[string]map[string]string, error)

	GetLastTradePrice(ctx context.Context, tokenID string) (*CLOBLastTradePrice, error)
	GetLastTradePrices(ctx context.Context, tokenIDs []string) ([]CLOBLastTradePrice, error)
	GetLastTradePricesByBody(ctx context.Context, requests []CLOBBookRequest) ([]CLOBLastTradePrice, error)

	GetSpread(ctx context.Context, tokenID string) (*CLOBSpreadResponse, error)
	GetSpreads(ctx context.Context, requests []CLOBBookRequest) (map[string]string, error)

	GetTickSize(ctx context.Context, tokenID string) (*CLOBTickSize, error)
	GetTickSizeByTokenID(ctx context.Context, tokenID string) (*CLOBTickSize, error)

	GetFeeRate(ctx context.Context, tokenID string) (*CLOBFeeRate, error)
	GetFeeRateByTokenID(ctx context.Context, tokenID string) (*CLOBFeeRate, error)

	GetServerTime(ctx context.Context) (*CLOBServerTimeResponse, error)
}

type CLOBConfig struct {
	CLOBBaseURL string
	Timeout     time.Duration
}

func (c CLOBConfig) WithDefaults() CLOBConfig {
	return c.withDefaults()
}

func (c CLOBConfig) withDefaults() CLOBConfig {
	c.CLOBBaseURL = strings.TrimSpace(c.CLOBBaseURL)
	if c.CLOBBaseURL == "" {
		c.CLOBBaseURL = DefaultCLOBBaseURL
	}
	if c.Timeout <= 0 {
		c.Timeout = DefaultTimeout
	}
	return c
}

type clobClientImpl struct {
	config CLOBConfig
	http   *http.Client
}

var _ CLOBClient = (*clobClientImpl)(nil)

func NewCLOBClient(config CLOBConfig) (CLOBClient, error) {
	config = config.withDefaults()
	if _, err := url.ParseRequestURI(config.CLOBBaseURL); err != nil {
		return nil, fmt.Errorf("invalid polymarket clob base url: %w", err)
	}
	return &clobClientImpl{
		config: config,
		http:   &http.Client{Timeout: config.Timeout},
	}, nil
}

type CLOBBookRequest struct {
	TokenID string `json:"token_id"`
	Side    string `json:"side,omitempty"`
}

type CLOBOrderSummary struct {
	Price string `json:"price"`
	Size  string `json:"size"`
}

type OrderBookSummary struct {
	Market         string             `json:"market"`
	AssetID        string             `json:"asset_id"`
	Timestamp      string             `json:"timestamp"`
	Hash           string             `json:"hash"`
	Bids           []CLOBOrderSummary `json:"bids"`
	Asks           []CLOBOrderSummary `json:"asks"`
	MinOrderSize   string             `json:"min_order_size"`
	TickSize       string             `json:"tick_size"`
	NegRisk        bool               `json:"neg_risk"`
	LastTradePrice string             `json:"last_trade_price"`
}

type CLOBPriceResponse struct {
	Price string `json:"price"`
}

type CLOBMidpointPriceResponse struct {
	MidPrice string `json:"mid_price"`
}

func (r *CLOBMidpointPriceResponse) UnmarshalJSON(data []byte) error {
	var raw struct {
		Mid      *string `json:"mid"`
		MidPrice *string `json:"mid_price"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	switch {
	case raw.Mid != nil:
		r.MidPrice = *raw.Mid
	case raw.MidPrice != nil:
		r.MidPrice = *raw.MidPrice
	default:
		r.MidPrice = ""
	}
	return nil
}

type CLOBLastTradePrice struct {
	TokenID string `json:"token_id,omitempty"`
	Price   string `json:"price"`
	Side    string `json:"side"`
}

type CLOBSpreadResponse struct {
	Spread string `json:"spread"`
}

type CLOBTickSize struct {
	MinimumTickSize float64 `json:"minimum_tick_size"`
}

type CLOBFeeRate struct {
	BaseFee int64 `json:"base_fee"`
}

type CLOBServerTimeResponse struct {
	Unix int64 `json:"unix"`
}

func (r *CLOBServerTimeResponse) UnmarshalJSON(data []byte) error {
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" || trimmed == "null" {
		r.Unix = 0
		return nil
	}

	if trimmed[0] == '{' {
		var raw struct {
			Unix *int64  `json:"unix"`
			Time *int64  `json:"time"`
			TS   *int64  `json:"ts"`
			S    *string `json:"serverTime"`
		}
		if err := json.Unmarshal(data, &raw); err != nil {
			return err
		}
		switch {
		case raw.Unix != nil:
			r.Unix = *raw.Unix
		case raw.Time != nil:
			r.Unix = *raw.Time
		case raw.TS != nil:
			r.Unix = *raw.TS
		case raw.S != nil:
			v, err := strconv.ParseInt(strings.TrimSpace(*raw.S), 10, 64)
			if err != nil {
				return fmt.Errorf("invalid serverTime value %q: %w", *raw.S, err)
			}
			r.Unix = v
		default:
			r.Unix = 0
		}
		return nil
	}

	var unixInt int64
	if err := json.Unmarshal(data, &unixInt); err == nil {
		r.Unix = unixInt
		return nil
	}

	var unixString string
	if err := json.Unmarshal(data, &unixString); err != nil {
		return err
	}
	parsed, err := strconv.ParseInt(strings.TrimSpace(unixString), 10, 64)
	if err != nil {
		return fmt.Errorf("invalid server time string %q: %w", unixString, err)
	}
	r.Unix = parsed
	return nil
}

func (c *clobClientImpl) GetOrderBook(ctx context.Context, tokenID string) (*OrderBookSummary, error) {
	q := make(url.Values)
	setString(q, "token_id", tokenID)
	var out OrderBookSummary
	if err := c.doJSON(ctx, http.MethodGet, "/book", q, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *clobClientImpl) GetOrderBooks(ctx context.Context, requests []CLOBBookRequest) ([]OrderBookSummary, error) {
	var out []OrderBookSummary
	if err := c.doJSON(ctx, http.MethodPost, "/books", nil, requests, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *clobClientImpl) GetMidpointPrice(ctx context.Context, tokenID string) (*CLOBMidpointPriceResponse, error) {
	q := make(url.Values)
	setString(q, "token_id", tokenID)
	var out CLOBMidpointPriceResponse
	if err := c.doJSON(ctx, http.MethodGet, "/midpoint", q, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *clobClientImpl) GetMidpointPrices(ctx context.Context, tokenIDs []string) (map[string]string, error) {
	requests := make([]CLOBBookRequest, 0, len(tokenIDs))
	for _, tokenID := range tokenIDs {
		tokenID = strings.TrimSpace(tokenID)
		if tokenID == "" {
			continue
		}
		requests = append(requests, CLOBBookRequest{TokenID: tokenID})
	}
	return c.GetMidpointPricesByBody(ctx, requests)
}

func (c *clobClientImpl) GetMidpointPricesByBody(ctx context.Context, requests []CLOBBookRequest) (map[string]string, error) {
	out := map[string]string{}
	if err := c.doJSON(ctx, http.MethodPost, "/midpoints", nil, requests, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *clobClientImpl) GetMarketPrice(ctx context.Context, tokenID, side string) (*CLOBPriceResponse, error) {
	q := make(url.Values)
	setString(q, "token_id", tokenID)
	setString(q, "side", side)
	var out CLOBPriceResponse
	if err := c.doJSON(ctx, http.MethodGet, "/price", q, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *clobClientImpl) GetMarketPrices(ctx context.Context, tokenIDs []string, sides []string) (map[string]map[string]string, error) {
	if len(tokenIDs) != len(sides) {
		return nil, fmt.Errorf("tokenIDs and sides length mismatch: %d vs %d", len(tokenIDs), len(sides))
	}
	requests := make([]CLOBBookRequest, 0, len(tokenIDs))
	for i := range tokenIDs {
		tokenID := strings.TrimSpace(tokenIDs[i])
		side := strings.TrimSpace(sides[i])
		if tokenID == "" {
			continue
		}
		requests = append(requests, CLOBBookRequest{TokenID: tokenID, Side: side})
	}
	return c.GetMarketPricesByBody(ctx, requests)
}

func (c *clobClientImpl) GetMarketPricesByBody(ctx context.Context, requests []CLOBBookRequest) (map[string]map[string]string, error) {
	out := map[string]map[string]string{}
	if err := c.doJSON(ctx, http.MethodPost, "/prices", nil, requests, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *clobClientImpl) GetLastTradePrice(ctx context.Context, tokenID string) (*CLOBLastTradePrice, error) {
	q := make(url.Values)
	setString(q, "token_id", tokenID)
	var out CLOBLastTradePrice
	if err := c.doJSON(ctx, http.MethodGet, "/last-trade-price", q, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *clobClientImpl) GetLastTradePrices(ctx context.Context, tokenIDs []string) ([]CLOBLastTradePrice, error) {
	requests := make([]CLOBBookRequest, 0, len(tokenIDs))
	for _, tokenID := range tokenIDs {
		tokenID = strings.TrimSpace(tokenID)
		if tokenID == "" {
			continue
		}
		requests = append(requests, CLOBBookRequest{TokenID: tokenID})
	}
	return c.GetLastTradePricesByBody(ctx, requests)
}

func (c *clobClientImpl) GetLastTradePricesByBody(ctx context.Context, requests []CLOBBookRequest) ([]CLOBLastTradePrice, error) {
	var out []CLOBLastTradePrice
	if err := c.doJSON(ctx, http.MethodPost, "/last-trades-prices", nil, requests, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *clobClientImpl) GetSpread(ctx context.Context, tokenID string) (*CLOBSpreadResponse, error) {
	q := make(url.Values)
	setString(q, "token_id", tokenID)
	var out CLOBSpreadResponse
	if err := c.doJSON(ctx, http.MethodGet, "/spread", q, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *clobClientImpl) GetSpreads(ctx context.Context, requests []CLOBBookRequest) (map[string]string, error) {
	out := map[string]string{}
	if err := c.doJSON(ctx, http.MethodPost, "/spreads", nil, requests, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *clobClientImpl) GetTickSize(ctx context.Context, tokenID string) (*CLOBTickSize, error) {
	q := make(url.Values)
	setString(q, "token_id", tokenID)
	var out CLOBTickSize
	if err := c.doJSON(ctx, http.MethodGet, "/tick-size", q, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *clobClientImpl) GetTickSizeByTokenID(ctx context.Context, tokenID string) (*CLOBTickSize, error) {
	var out CLOBTickSize
	if err := c.doJSON(ctx, http.MethodGet, "/tick-size/"+url.PathEscape(strings.TrimSpace(tokenID)), nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *clobClientImpl) GetFeeRate(ctx context.Context, tokenID string) (*CLOBFeeRate, error) {
	q := make(url.Values)
	setString(q, "token_id", tokenID)
	var out CLOBFeeRate
	if err := c.doJSON(ctx, http.MethodGet, "/fee-rate", q, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *clobClientImpl) GetFeeRateByTokenID(ctx context.Context, tokenID string) (*CLOBFeeRate, error) {
	var out CLOBFeeRate
	if err := c.doJSON(ctx, http.MethodGet, "/fee-rate/"+url.PathEscape(strings.TrimSpace(tokenID)), nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *clobClientImpl) GetServerTime(ctx context.Context) (*CLOBServerTimeResponse, error) {
	var out CLOBServerTimeResponse
	if err := c.doJSON(ctx, http.MethodGet, "/time", nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *clobClientImpl) doJSON(ctx context.Context, method, path string, query url.Values, body any, out any) error {
	endpoint, err := c.buildURL(path, query)
	if err != nil {
		return err
	}

	var payload io.Reader = http.NoBody
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to encode polymarket clob request body: %w", err)
		}
		payload = bytes.NewReader(encoded)
	}

	req, err := http.NewRequestWithContext(ctx, method, endpoint.String(), payload)
	if err != nil {
		return fmt.Errorf("failed to create polymarket clob request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send polymarket clob request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return decodeHTTPError(resp)
	}

	if out == nil {
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("failed to decode polymarket clob response: %w", err)
	}
	return nil
}

func (c *clobClientImpl) buildURL(path string, query url.Values) (*url.URL, error) {
	endpoint, err := url.Parse(strings.TrimRight(c.config.CLOBBaseURL, "/") + path)
	if err != nil {
		return nil, fmt.Errorf("failed to parse polymarket clob request url: %w", err)
	}
	if len(query) > 0 {
		endpoint.RawQuery = query.Encode()
	}
	return endpoint, nil
}
