package polymarket

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
)

type CLOBMarketByTokenResponse struct {
	ConditionID      string `json:"condition_id"`
	PrimaryTokenID   string `json:"primary_token_id"`
	SecondaryTokenID string `json:"secondary_token_id"`
}

type CLOBMarketInfo struct {
	GameStartTime      *string               `json:"gst,omitempty"`
	Rewards            json.RawMessage       `json:"r,omitempty"`
	Tokens             []CLOBMarketInfoToken `json:"t,omitempty"`
	MinimumOrderSize   *float64              `json:"mos,omitempty"`
	MinimumTickSize    *float64              `json:"mts,omitempty"`
	MakerBaseFee       *int64                `json:"mbf,omitempty"`
	TakerBaseFee       *int64                `json:"tbf,omitempty"`
	RFQEnabled         *bool                 `json:"rfqe,omitempty"`
	IsTakerDelay       *bool                 `json:"itode,omitempty"`
	IsBlockaidCheck    *bool                 `json:"ibce,omitempty"`
	FeeDetails         *CLOBFeeDetails       `json:"fd,omitempty"`
	MinOrderAgeSeconds *int64                `json:"oas,omitempty"`
}

type CLOBMarketInfoToken struct {
	TokenID string `json:"t"`
	Outcome string `json:"o"`
}

type CLOBFeeDetails struct {
	Rate      *float64 `json:"r,omitempty"`
	Exponent  *float64 `json:"e,omitempty"`
	TakerOnly *bool    `json:"to,omitempty"`
}

type GetCLOBPricesHistoryOptions struct {
	Market   string
	StartTs  *float64
	EndTs    *float64
	Interval string
	Fidelity *int
}

type CLOBMarketPricePoint struct {
	T int64   `json:"t"`
	P float64 `json:"p"`
}

type CLOBPricesHistoryResponse struct {
	History []CLOBMarketPricePoint `json:"history"`
}

type CLOBBatchPricesHistoryRequest struct {
	Markets  []string `json:"markets"`
	StartTs  *float64 `json:"start_ts,omitempty"`
	EndTs    *float64 `json:"end_ts,omitempty"`
	Interval string   `json:"interval,omitempty"`
	Fidelity *int     `json:"fidelity,omitempty"`
}

type CLOBBatchPricesHistoryResponse struct {
	History map[string][]CLOBMarketPricePoint `json:"history"`
}

type CLOBMarketsPage struct {
	Limit      *int              `json:"limit,omitempty"`
	NextCursor string            `json:"next_cursor,omitempty"`
	Count      *int              `json:"count,omitempty"`
	Items      []json.RawMessage `json:"data"`
}

func (c *clobClientImpl) GetMarketByToken(ctx context.Context, tokenID string) (*CLOBMarketByTokenResponse, error) {
	var out CLOBMarketByTokenResponse
	if err := c.doJSON(ctx, http.MethodGet, "/markets-by-token/"+url.PathEscape(strings.TrimSpace(tokenID)), nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *clobClientImpl) GetCLOBMarketInfo(ctx context.Context, conditionID string) (*CLOBMarketInfo, error) {
	var out CLOBMarketInfo
	if err := c.doJSON(ctx, http.MethodGet, "/clob-markets/"+url.PathEscape(strings.TrimSpace(conditionID)), nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *clobClientImpl) GetPricesHistory(ctx context.Context, options GetCLOBPricesHistoryOptions) (*CLOBPricesHistoryResponse, error) {
	q := make(url.Values)
	setString(q, "market", options.Market)
	setFloat64Ptr(q, "startTs", options.StartTs)
	setFloat64Ptr(q, "endTs", options.EndTs)
	setString(q, "interval", options.Interval)
	setIntPtr(q, "fidelity", options.Fidelity)

	var out CLOBPricesHistoryResponse
	if err := c.doJSON(ctx, http.MethodGet, "/prices-history", q, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *clobClientImpl) GetBatchPricesHistory(ctx context.Context, request CLOBBatchPricesHistoryRequest) (*CLOBBatchPricesHistoryResponse, error) {
	var out CLOBBatchPricesHistoryResponse
	if err := c.doJSON(ctx, http.MethodPost, "/batch-prices-history", nil, request, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *clobClientImpl) ListSimplifiedMarkets(ctx context.Context, nextCursor string) (*CLOBMarketsPage, error) {
	var out CLOBMarketsPage
	if err := c.doJSON(ctx, http.MethodGet, "/simplified-markets", cursorQuery(nextCursor), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *clobClientImpl) ListSamplingMarkets(ctx context.Context, nextCursor string) (*CLOBMarketsPage, error) {
	var out CLOBMarketsPage
	if err := c.doJSON(ctx, http.MethodGet, "/sampling-markets", cursorQuery(nextCursor), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *clobClientImpl) ListSamplingSimplifiedMarkets(ctx context.Context, nextCursor string) (*CLOBMarketsPage, error) {
	var out CLOBMarketsPage
	if err := c.doJSON(ctx, http.MethodGet, "/sampling-simplified-markets", cursorQuery(nextCursor), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func cursorQuery(nextCursor string) url.Values {
	q := make(url.Values)
	setString(q, "next_cursor", nextCursor)
	return q
}
