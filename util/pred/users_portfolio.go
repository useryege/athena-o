package pred

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

func (c *clientImpl) GetUser(ctx context.Context, identifier string) (*UserResponse, error) {
	endpoint := fmt.Sprintf("/users/%s", escapePath(identifier))
	return doTyped[UserResponse](ctx, c, http.MethodGet, endpoint, nil, nil, false)
}

func (c *clientImpl) EnsureUser(ctx context.Context, request EnsureUserRequest) (*UserResponse, error) {
	body, err := jsonRequestBody(request)
	if err != nil {
		return nil, err
	}
	return doTyped[UserResponse](ctx, c, http.MethodPost, "/users/ensure", nil, body, false)
}

func (c *clientImpl) UpdateUserProfile(
	ctx context.Context,
	targetWallet string,
	request UpdateUserProfileRequest,
) (*UserResponse, error) {
	values := make(url.Values)
	if request.Username != nil {
		values.Set("username", *request.Username)
	}
	if request.Bio != nil {
		values.Set("bio", *request.Bio)
	}
	endpoint := fmt.Sprintf("/users/%s/profile", escapePath(targetWallet))
	return doTyped[UserResponse](
		ctx,
		c,
		http.MethodPut,
		endpoint,
		nil,
		formRequestBody(values),
		true,
	)
}

func (c *clientImpl) ListUserActivity(
	ctx context.Context,
	identifier string,
	options ListUserActivityOptions,
) (*ActivityResponse, error) {
	query := make(url.Values)
	setOptionalInt(query, "limit", options.Limit)
	setOptionalInt(query, "offset", options.Offset)
	endpoint := fmt.Sprintf("/users/%s/activity", escapePath(identifier))
	return doTyped[ActivityResponse](ctx, c, http.MethodGet, endpoint, query, nil, false)
}

func (c *clientImpl) DownloadLiquidationJustification(
	ctx context.Context,
	liquidationID string,
) ([]byte, error) {
	endpoint := fmt.Sprintf(
		"/users/liquidation/%s/justification.xlsx",
		escapePath(liquidationID),
	)
	return c.doBytes(ctx, http.MethodGet, endpoint, nil, nil, false)
}

func (c *clientImpl) GetPortfolioSummary(ctx context.Context, wallet string) (*PortfolioSummary, error) {
	query := url.Values{"wallet": []string{wallet}}
	return doTyped[PortfolioSummary](ctx, c, http.MethodGet, "/portfolio/summary", query, nil, false)
}
