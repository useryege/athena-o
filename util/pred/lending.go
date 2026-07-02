package pred

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

func (c *clientImpl) GetPoolStats(ctx context.Context) (*PoolStats, error) {
	return doTyped[PoolStats](ctx, c, http.MethodGet, "/lending/pool-stats", nil, nil, false)
}

func (c *clientImpl) GetInterestCorrection(ctx context.Context, address string) (json.RawMessage, error) {
	endpoint := fmt.Sprintf("/lending/interest-correction/%s", escapePath(address))
	return doRaw(ctx, c, http.MethodGet, endpoint, nil, nil, false)
}

func (c *clientImpl) GetLendingPosition(
	ctx context.Context,
	address string,
	tokenID string,
	options GetLendingPositionOptions,
) (*PositionInfo, error) {
	query := make(url.Values)
	setOptionalString(query, "price", options.Price)
	endpoint := fmt.Sprintf("/lending/position/%s/%s", escapePath(address), escapePath(tokenID))
	return doTyped[PositionInfo](ctx, c, http.MethodGet, endpoint, query, nil, false)
}

func (c *clientImpl) ListLendingPositions(ctx context.Context, address string) (*UserPositionsResponse, error) {
	endpoint := fmt.Sprintf("/lending/positions/%s", escapePath(address))
	return doTyped[UserPositionsResponse](ctx, c, http.MethodGet, endpoint, nil, nil, false)
}

func (c *clientImpl) ListLendingEvents(
	ctx context.Context,
	address string,
	options ListLendingEventsOptions,
) (*EventHistoryResponse, error) {
	query := make(url.Values)
	setOptionalString(query, "token_id", options.TokenID)
	setOptionalString(query, "event_type", options.EventType)
	setOptionalInt(query, "limit", options.Limit)
	setOptionalInt(query, "offset", options.Offset)
	endpoint := fmt.Sprintf("/lending/events/%s", escapePath(address))
	return doTyped[EventHistoryResponse](ctx, c, http.MethodGet, endpoint, query, nil, false)
}

func (c *clientImpl) ListLiquidations(
	ctx context.Context,
	address string,
	options ListLiquidationsOptions,
) (*LiquidationHistoryResponse, error) {
	query := make(url.Values)
	setOptionalInt(query, "limit", options.Limit)
	setOptionalInt(query, "offset", options.Offset)
	endpoint := fmt.Sprintf("/lending/liquidations/%s", escapePath(address))
	return doTyped[LiquidationHistoryResponse](ctx, c, http.MethodGet, endpoint, query, nil, false)
}

func (c *clientImpl) GetRateHistory(
	ctx context.Context,
	options GetRateHistoryOptions,
) (*RateHistoryResponse, error) {
	query := make(url.Values)
	setString(query, "range", options.Range)
	return doTyped[RateHistoryResponse](ctx, c, http.MethodGet, "/lending/rate-history", query, nil, false)
}

func (c *clientImpl) GetRealAPY(ctx context.Context) (json.RawMessage, error) {
	return doRaw(ctx, c, http.MethodGet, "/lending/real-apy", nil, nil, false)
}

func (c *clientImpl) GetDepthStatus(ctx context.Context, tokenIDs string) (*DepthStatusResponse, error) {
	query := url.Values{"token_ids": []string{tokenIDs}}
	return doTyped[DepthStatusResponse](ctx, c, http.MethodGet, "/lending/depth-status", query, nil, false)
}

func (c *clientImpl) GetOraclePrice(ctx context.Context, tokenID string) (json.RawMessage, error) {
	endpoint := fmt.Sprintf("/lending/oracle-price/%s", escapePath(tokenID))
	return doRaw(ctx, c, http.MethodGet, endpoint, nil, nil, false)
}

func (c *clientImpl) GetOraclePrices(ctx context.Context, tokenIDs string) (json.RawMessage, error) {
	query := url.Values{"token_ids": []string{tokenIDs}}
	return doRaw(ctx, c, http.MethodGet, "/lending/oracle-prices", query, nil, false)
}

func (c *clientImpl) GetOrderbookAsks(ctx context.Context, tokenID string) (json.RawMessage, error) {
	endpoint := fmt.Sprintf("/lending/orderbook-asks/%s", escapePath(tokenID))
	return doRaw(ctx, c, http.MethodGet, endpoint, nil, nil, false)
}

func (c *clientImpl) GetContractInfo(ctx context.Context) (*ContractInfo, error) {
	return doTyped[ContractInfo](ctx, c, http.MethodGet, "/lending/constants", nil, nil, false)
}
