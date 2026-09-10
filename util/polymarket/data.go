package polymarket

import (
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

const DefaultDataBaseURL = "https://data-api.polymarket.com"

// DataClient is a typed Polymarket Data API client.
type DataClient interface {
	ListUserActivity(ctx context.Context, options ListUserActivityOptions) ([]DataActivity, error)
	ListTrades(ctx context.Context, options ListDataTradesOptions) ([]DataTrade, error)
	ListCurrentPositions(ctx context.Context, options ListCurrentPositionsOptions) ([]DataPosition, error)
	ListClosedPositions(ctx context.Context, options ListClosedPositionsOptions) ([]DataPosition, error)
	ListMarketPositions(ctx context.Context, options ListMarketPositionsOptions) ([]DataMarketPosition, error)
	ListTopHolders(ctx context.Context, options ListTopHoldersOptions) ([]DataHolder, error)
	GetTotalValueForUser(ctx context.Context, user string, options GetTotalValueOptions) ([]DataUserValue, error)
	ListTraderLeaderboard(ctx context.Context, options ListTraderLeaderboardOptions) ([]DataLeaderboardEntry, error)

	GetOpenInterest(ctx context.Context, options GetOpenInterestOptions) ([]DataOpenInterest, error)
	GetLiveVolumeByEventID(ctx context.Context, id int64) ([]DataLiveVolume, error)
	GetTotalMarketsTraded(ctx context.Context, user string) (MarketsTraded, error)
	DownloadAccountingSnapshot(ctx context.Context, user string) ([]byte, error)

	ListBuilderLeaderboard(ctx context.Context, options ListBuilderLeaderboardOptions) ([]BuilderLeaderboardEntry, error)
	ListBuilderDailyVolume(ctx context.Context, options ListBuilderDailyVolumeOptions) ([]BuilderDailyVolumeEntry, error)
}

type DataConfig struct {
	DataBaseURL string
	Timeout     time.Duration
}

func (c DataConfig) WithDefaults() DataConfig {
	return c.withDefaults()
}

func (c DataConfig) withDefaults() DataConfig {
	c.DataBaseURL = strings.TrimSpace(c.DataBaseURL)
	if c.DataBaseURL == "" {
		c.DataBaseURL = DefaultDataBaseURL
	}
	if c.Timeout <= 0 {
		c.Timeout = DefaultTimeout
	}
	return c
}

type dataClientImpl struct {
	config DataConfig
	http   *http.Client
}

var _ DataClient = (*dataClientImpl)(nil)

func NewDataClient(config DataConfig) (DataClient, error) {
	config = config.withDefaults()
	if _, err := url.ParseRequestURI(config.DataBaseURL); err != nil {
		return nil, fmt.Errorf("invalid polymarket data base url: %w", err)
	}
	return &dataClientImpl{
		config: config,
		http:   &http.Client{Timeout: config.Timeout},
	}, nil
}

type DataListOptions struct {
	Limit  *int
	Offset *int
}

type ListUserActivityOptions struct {
	DataListOptions
	User          string
	Market        []string
	EventID       []int64
	Type          []string
	Start         *int64
	End           *int64
	SortBy        string
	SortDirection string
	Side          string
}

type ListDataTradesOptions struct {
	DataListOptions
	TakerOnly    *bool
	FilterType   string
	FilterAmount *float64
	Market       []string
	EventID      []int64
	User         string
	Side         string
}

type ListCurrentPositionsOptions struct {
	DataListOptions
	User          string
	Market        []string
	EventID       []int64
	SizeThreshold *float64
	Redeemable    *bool
	Mergeable     *bool
	SortBy        string
	SortDirection string
	Title         string
}

type ListClosedPositionsOptions struct {
	DataListOptions
	User          string
	Market        []string
	Title         string
	EventID       []int64
	SortBy        string
	SortDirection string
}

type ListMarketPositionsOptions struct {
	DataListOptions
	Market        string
	User          string
	Status        string
	SortBy        string
	SortDirection string
}

type ListTopHoldersOptions struct {
	Limit      *int
	Market     []string
	MinBalance *float64
}

type GetTotalValueOptions struct {
	Market []string
}

type ListTraderLeaderboardOptions struct {
	Category   string
	TimePeriod string
	OrderBy    string
	Limit      *int
	Offset     *int
	User       string
	UserName   string
}

type GetOpenInterestOptions struct {
	Market []string
}

type ListBuilderLeaderboardOptions struct {
	TimePeriod string
	Limit      *int
	Offset     *int
}

type ListBuilderDailyVolumeOptions struct {
	TimePeriod string
}

type DataActivity map[string]any

type DataTrade map[string]any

type DataPosition map[string]any

type DataMarketPosition map[string]any

type DataHolder map[string]any

type DataLeaderboardEntry map[string]any

type BuilderLeaderboardEntry map[string]any

type BuilderDailyVolumeEntry map[string]any

type DataUserValue struct {
	User  string  `json:"user"`
	Value Decimal `json:"value"`
}

type DataOpenInterest struct {
	Market string  `json:"market"`
	Value  float64 `json:"value"`
}

type DataMarketVolume struct {
	Market string  `json:"market"`
	Value  float64 `json:"value"`
}

type DataLiveVolume struct {
	Total   float64            `json:"total"`
	Markets []DataMarketVolume `json:"markets"`
}

func (c *dataClientImpl) ListUserActivity(ctx context.Context, options ListUserActivityOptions) ([]DataActivity, error) {
	var out []DataActivity
	if err := c.doJSON(ctx, http.MethodGet, "/activity", options.values(), nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *dataClientImpl) ListTrades(ctx context.Context, options ListDataTradesOptions) ([]DataTrade, error) {
	var out []DataTrade
	if err := c.doJSON(ctx, http.MethodGet, "/trades", options.values(), nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *dataClientImpl) ListCurrentPositions(ctx context.Context, options ListCurrentPositionsOptions) ([]DataPosition, error) {
	var out []DataPosition
	if err := c.doJSON(ctx, http.MethodGet, "/positions", options.values(), nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *dataClientImpl) ListClosedPositions(ctx context.Context, options ListClosedPositionsOptions) ([]DataPosition, error) {
	var out []DataPosition
	if err := c.doJSON(ctx, http.MethodGet, "/closed-positions", options.values(), nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *dataClientImpl) ListMarketPositions(ctx context.Context, options ListMarketPositionsOptions) ([]DataMarketPosition, error) {
	var out []DataMarketPosition
	if err := c.doJSON(ctx, http.MethodGet, "/v1/market-positions", options.values(), nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *dataClientImpl) ListTopHolders(ctx context.Context, options ListTopHoldersOptions) ([]DataHolder, error) {
	var out []DataHolder
	if err := c.doJSON(ctx, http.MethodGet, "/holders", options.values(), nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *dataClientImpl) GetTotalValueForUser(ctx context.Context, user string, options GetTotalValueOptions) ([]DataUserValue, error) {
	q := options.values()
	setString(q, "user", user)
	var out []DataUserValue
	if err := c.doJSON(ctx, http.MethodGet, "/value", q, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *dataClientImpl) ListTraderLeaderboard(ctx context.Context, options ListTraderLeaderboardOptions) ([]DataLeaderboardEntry, error) {
	var out []DataLeaderboardEntry
	if err := c.doJSON(ctx, http.MethodGet, "/v1/leaderboard", options.values(), nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *dataClientImpl) GetOpenInterest(ctx context.Context, options GetOpenInterestOptions) ([]DataOpenInterest, error) {
	var out []DataOpenInterest
	if err := c.doJSON(ctx, http.MethodGet, "/oi", options.values(), nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *dataClientImpl) GetLiveVolumeByEventID(ctx context.Context, id int64) ([]DataLiveVolume, error) {
	q := make(url.Values)
	q.Set("id", strconv.FormatInt(id, 10))
	var out []DataLiveVolume
	if err := c.doJSON(ctx, http.MethodGet, "/live-volume", q, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *dataClientImpl) GetTotalMarketsTraded(ctx context.Context, user string) (MarketsTraded, error) {
	q := make(url.Values)
	setString(q, "user", user)
	out := MarketsTraded{}
	if err := c.doJSON(ctx, http.MethodGet, "/traded", q, nil, &out); err != nil {
		return MarketsTraded{}, err
	}
	return out, nil
}

func (c *dataClientImpl) DownloadAccountingSnapshot(ctx context.Context, user string) ([]byte, error) {
	q := make(url.Values)
	setString(q, "user", user)
	return c.doBytes(ctx, http.MethodGet, "/v1/accounting/snapshot", q)
}

func (c *dataClientImpl) ListBuilderLeaderboard(ctx context.Context, options ListBuilderLeaderboardOptions) ([]BuilderLeaderboardEntry, error) {
	var out []BuilderLeaderboardEntry
	if err := c.doJSON(ctx, http.MethodGet, "/v1/builders/leaderboard", options.values(), nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *dataClientImpl) ListBuilderDailyVolume(ctx context.Context, options ListBuilderDailyVolumeOptions) ([]BuilderDailyVolumeEntry, error) {
	var out []BuilderDailyVolumeEntry
	if err := c.doJSON(ctx, http.MethodGet, "/v1/builders/volume", options.values(), nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *dataClientImpl) doJSON(ctx context.Context, method, path string, query url.Values, body io.Reader, out any) error {
	endpoint, err := c.buildURL(path, query)
	if err != nil {
		return err
	}

	if body == nil {
		body = http.NoBody
	}

	req, err := http.NewRequestWithContext(ctx, method, endpoint.String(), body)
	if err != nil {
		return fmt.Errorf("failed to create polymarket data request: %w", err)
	}
	if body != http.NoBody {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send polymarket data request: %w", err)
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
		return fmt.Errorf("failed to decode polymarket data response: %w", err)
	}
	return nil
}

func (c *dataClientImpl) doBytes(ctx context.Context, method, path string, query url.Values) ([]byte, error) {
	endpoint, err := c.buildURL(path, query)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, method, endpoint.String(), http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create polymarket data request: %w", err)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send polymarket data request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, decodeHTTPError(resp)
	}

	payload, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read polymarket data response: %w", err)
	}
	return payload, nil
}

func (c *dataClientImpl) buildURL(path string, query url.Values) (*url.URL, error) {
	endpoint, err := url.Parse(strings.TrimRight(c.config.DataBaseURL, "/") + path)
	if err != nil {
		return nil, fmt.Errorf("failed to parse polymarket data request url: %w", err)
	}
	if len(query) > 0 {
		endpoint.RawQuery = query.Encode()
	}
	return endpoint, nil
}

func (o DataListOptions) values() url.Values {
	q := make(url.Values)
	setIntPtr(q, "limit", o.Limit)
	setIntPtr(q, "offset", o.Offset)
	return q
}

func (o ListUserActivityOptions) values() url.Values {
	q := o.DataListOptions.values()
	setString(q, "user", o.User)
	setCSVStrings(q, "market", o.Market)
	setCSVInt64(q, "eventId", o.EventID)
	setCSVStrings(q, "type", o.Type)
	setInt64Ptr(q, "start", o.Start)
	setInt64Ptr(q, "end", o.End)
	setString(q, "sortBy", o.SortBy)
	setString(q, "sortDirection", o.SortDirection)
	setString(q, "side", o.Side)
	return q
}

func (o ListDataTradesOptions) values() url.Values {
	q := o.DataListOptions.values()
	setBoolPtr(q, "takerOnly", o.TakerOnly)
	setString(q, "filterType", o.FilterType)
	setFloat64Ptr(q, "filterAmount", o.FilterAmount)
	setCSVStrings(q, "market", o.Market)
	setCSVInt64(q, "eventId", o.EventID)
	setString(q, "user", o.User)
	setString(q, "side", o.Side)
	return q
}

func (o ListCurrentPositionsOptions) values() url.Values {
	q := o.DataListOptions.values()
	setString(q, "user", o.User)
	setCSVStrings(q, "market", o.Market)
	setCSVInt64(q, "eventId", o.EventID)
	setFloat64Ptr(q, "sizeThreshold", o.SizeThreshold)
	setBoolPtr(q, "redeemable", o.Redeemable)
	setBoolPtr(q, "mergeable", o.Mergeable)
	setString(q, "sortBy", o.SortBy)
	setString(q, "sortDirection", o.SortDirection)
	setString(q, "title", o.Title)
	return q
}

func (o ListClosedPositionsOptions) values() url.Values {
	q := o.DataListOptions.values()
	setString(q, "user", o.User)
	setCSVStrings(q, "market", o.Market)
	setString(q, "title", o.Title)
	setCSVInt64(q, "eventId", o.EventID)
	setString(q, "sortBy", o.SortBy)
	setString(q, "sortDirection", o.SortDirection)
	return q
}

func (o ListMarketPositionsOptions) values() url.Values {
	q := o.DataListOptions.values()
	setString(q, "market", o.Market)
	setString(q, "user", o.User)
	setString(q, "status", o.Status)
	setString(q, "sortBy", o.SortBy)
	setString(q, "sortDirection", o.SortDirection)
	return q
}

func (o ListTopHoldersOptions) values() url.Values {
	q := make(url.Values)
	setIntPtr(q, "limit", o.Limit)
	setCSVStrings(q, "market", o.Market)
	setFloat64Ptr(q, "minBalance", o.MinBalance)
	return q
}

func (o GetTotalValueOptions) values() url.Values {
	q := make(url.Values)
	setCSVStrings(q, "market", o.Market)
	return q
}

func (o ListTraderLeaderboardOptions) values() url.Values {
	q := make(url.Values)
	setString(q, "category", o.Category)
	setString(q, "timePeriod", o.TimePeriod)
	setString(q, "orderBy", o.OrderBy)
	setIntPtr(q, "limit", o.Limit)
	setIntPtr(q, "offset", o.Offset)
	setString(q, "user", o.User)
	setString(q, "userName", o.UserName)
	return q
}

func (o GetOpenInterestOptions) values() url.Values {
	q := make(url.Values)
	setCSVStrings(q, "market", o.Market)
	return q
}

func (o ListBuilderLeaderboardOptions) values() url.Values {
	q := make(url.Values)
	setString(q, "timePeriod", o.TimePeriod)
	setIntPtr(q, "limit", o.Limit)
	setIntPtr(q, "offset", o.Offset)
	return q
}

func (o ListBuilderDailyVolumeOptions) values() url.Values {
	q := make(url.Values)
	setString(q, "timePeriod", o.TimePeriod)
	return q
}

func setCSVStrings(query url.Values, key string, values []string) {
	if len(values) == 0 {
		return
	}
	clean := make([]string, 0, len(values))
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v != "" {
			clean = append(clean, v)
		}
	}
	if len(clean) > 0 {
		query.Set(key, strings.Join(clean, ","))
	}
}

func setCSVInt64(query url.Values, key string, values []int64) {
	if len(values) == 0 {
		return
	}
	parts := make([]string, 0, len(values))
	for _, v := range values {
		parts = append(parts, strconv.FormatInt(v, 10))
	}
	query.Set(key, strings.Join(parts, ","))
}
