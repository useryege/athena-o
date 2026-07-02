package pred

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

func (c *clientImpl) GetLeverageAuthData(
	ctx context.Context,
	options GetLeverageAuthDataOptions,
) (json.RawMessage, error) {
	query := url.Values{
		"token_id":    []string{options.TokenID},
		"usdc_amount": []string{fmt.Sprintf("%g", options.USDCAmount)},
	}
	setOptionalFloat(query, "target_leverage", options.TargetLeverage)
	return doRaw(ctx, c, http.MethodGet, "/leverage/auth-data", query, nil, true)
}

func (c *clientImpl) GetLeverageStatus(
	ctx context.Context,
	leverageID string,
) (*LeverageStatusResponse, error) {
	endpoint := fmt.Sprintf("/leverage/status/%s", escapePath(leverageID))
	return doTyped[LeverageStatusResponse](ctx, c, http.MethodGet, endpoint, nil, nil, false)
}

func (c *clientImpl) OpenLeverage(
	ctx context.Context,
	request LeverageOpenRequest,
) (*LeverageOpenResponse, error) {
	body, err := jsonRequestBody(request)
	if err != nil {
		return nil, err
	}
	return doTyped[LeverageOpenResponse](ctx, c, http.MethodPost, "/leverage/open", nil, body, true)
}

func (c *clientImpl) GetFlashCloseAuthData(
	ctx context.Context,
	options GetFlashCloseAuthDataOptions,
) (json.RawMessage, error) {
	query := url.Values{
		"wallet":   []string{options.Wallet},
		"token_id": []string{options.TokenID},
	}
	return doRaw(ctx, c, http.MethodGet, "/leverage/flash-close-auth-data", query, nil, false)
}

func (c *clientImpl) OpenFlashClose(
	ctx context.Context,
	request FlashCloseOpenRequest,
) (json.RawMessage, error) {
	body, err := jsonRequestBody(request)
	if err != nil {
		return nil, err
	}
	return doRaw(ctx, c, http.MethodPost, "/leverage/flash-close", nil, body, true)
}

func (c *clientImpl) GetFlashCloseStatus(ctx context.Context, closeID string) (json.RawMessage, error) {
	endpoint := fmt.Sprintf("/leverage/flash-close-status/%s", escapePath(closeID))
	return doRaw(ctx, c, http.MethodGet, endpoint, nil, nil, false)
}

func (c *clientImpl) UpsertTPSL(ctx context.Context, request TPSLCreateRequest) (*TPSLResponse, error) {
	body, err := jsonRequestBody(request)
	if err != nil {
		return nil, err
	}
	return doTyped[TPSLResponse](ctx, c, http.MethodPost, "/tp-sl", nil, body, true)
}

func (c *clientImpl) GetTPSL(ctx context.Context, tokenID string) (*TPSLResponse, error) {
	query := url.Values{"token_id": []string{tokenID}}
	return doNullable[TPSLResponse](ctx, c, http.MethodGet, "/tp-sl", query, nil, true)
}

func (c *clientImpl) CancelTPSL(ctx context.Context, tokenID string) (json.RawMessage, error) {
	query := url.Values{"token_id": []string{tokenID}}
	return doRaw(ctx, c, http.MethodDelete, "/tp-sl", query, nil, true)
}

func (c *clientImpl) GetTPSLAuthData(ctx context.Context, tokenID string) (json.RawMessage, error) {
	query := url.Values{"token_id": []string{tokenID}}
	return doRaw(ctx, c, http.MethodGet, "/tp-sl/auth-data", query, nil, true)
}

func (c *clientImpl) TriggerTPSLTest(
	ctx context.Context,
	options TriggerTPSLTestOptions,
) (json.RawMessage, error) {
	query := url.Values{"token_id": []string{options.TokenID}}
	setString(query, "trigger_type", options.TriggerType)
	return doRaw(ctx, c, http.MethodPost, "/tp-sl/test-trigger", query, nil, true)
}
