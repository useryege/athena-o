package pred

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

func (c *clientImpl) ListMarkets(
	ctx context.Context,
	options ListMarketsOptions,
) (json.RawMessage, error) {
	query := make(url.Values)
	setOptionalString(query, "status", options.Status)
	setOptionalString(query, "category", options.Category)
	setOptionalString(query, "group_id", options.GroupID)
	setOptionalString(query, "created_by", options.CreatedBy)
	setOptionalBool(query, "exclude_test", options.ExcludeTest)
	setOptionalInt(query, "limit", options.Limit)
	setOptionalInt(query, "offset", options.Offset)
	return doRaw(ctx, c, http.MethodGet, "/markets/", query, nil, false)
}

func (c *clientImpl) ListGroups(
	ctx context.Context,
	options ListGroupsOptions,
) (json.RawMessage, error) {
	query := make(url.Values)
	setOptionalString(query, "category", options.Category)
	setOptionalBool(query, "exclude_test", options.ExcludeTest)
	setOptionalInt(query, "limit", options.Limit)
	setOptionalInt(query, "offset", options.Offset)
	return doRaw(ctx, c, http.MethodGet, "/groups/", query, nil, false)
}

func (c *clientImpl) ListCategories(ctx context.Context) (json.RawMessage, error) {
	return doRaw(ctx, c, http.MethodGet, "/categories/", nil, nil, false)
}

func (c *clientImpl) ListTrendingTags(ctx context.Context) (json.RawMessage, error) {
	return doRaw(ctx, c, http.MethodGet, "/trending-tags", nil, nil, false)
}

func (c *clientImpl) SearchMarkets(
	ctx context.Context,
	options SearchMarketsOptions,
) (json.RawMessage, error) {
	query := make(url.Values)
	setString(query, "q", options.Query)
	setOptionalInt(query, "limit", options.Limit)
	return doRaw(ctx, c, http.MethodGet, "/search", query, nil, false)
}

func (c *clientImpl) LookupSlug(ctx context.Context, slug string) (json.RawMessage, error) {
	endpoint := fmt.Sprintf("/lookup/%s", escapePath(slug))
	return doRaw(ctx, c, http.MethodGet, endpoint, nil, nil, false)
}

func (c *clientImpl) GetMarket(ctx context.Context, slug string) (json.RawMessage, error) {
	endpoint := fmt.Sprintf("/markets/%s", escapePath(slug))
	return doRaw(ctx, c, http.MethodGet, endpoint, nil, nil, false)
}

func (c *clientImpl) ListSeriesSiblings(
	ctx context.Context,
	seriesSlug string,
) (json.RawMessage, error) {
	endpoint := fmt.Sprintf("/markets/series/%s/siblings", escapePath(seriesSlug))
	return doRaw(ctx, c, http.MethodGet, endpoint, nil, nil, false)
}

func (c *clientImpl) GetGroup(ctx context.Context, slug string) (json.RawMessage, error) {
	endpoint := fmt.Sprintf("/groups/%s", escapePath(slug))
	return doRaw(ctx, c, http.MethodGet, endpoint, nil, nil, false)
}

func (c *clientImpl) GetPriceHistory(
	ctx context.Context,
	marketID string,
	options GetPriceHistoryOptions,
) (json.RawMessage, error) {
	query := make(url.Values)
	setString(query, "interval", options.Interval)
	endpoint := fmt.Sprintf("/price-history/%s", escapePath(marketID))
	return doRaw(ctx, c, http.MethodGet, endpoint, query, nil, false)
}

func (c *clientImpl) GetAggregatedOrderbook(
	ctx context.Context,
	marketID string,
	side string,
	options GetAggregatedOrderbookOptions,
) (json.RawMessage, error) {
	query := make(url.Values)
	setOptionalInt(query, "levels", options.Levels)
	endpoint := fmt.Sprintf(
		"/orders/book-aggregated/%s/%s",
		escapePath(marketID),
		escapePath(side),
	)
	return doRaw(ctx, c, http.MethodGet, endpoint, query, nil, false)
}

func (c *clientImpl) GetMarketPosition(
	ctx context.Context,
	marketID string,
	wallet string,
) (json.RawMessage, error) {
	query := url.Values{"wallet": []string{wallet}}
	endpoint := fmt.Sprintf("/portfolio/position/%s", escapePath(marketID))
	return doRaw(ctx, c, http.MethodGet, endpoint, query, nil, false)
}
