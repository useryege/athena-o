package pred

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

func (c *clientImpl) GetCLOBAuthData(ctx context.Context, wallet string) (json.RawMessage, error) {
	query := url.Values{"wallet": []string{wallet}}
	return doRaw(ctx, c, http.MethodGet, "/orders/clob-auth-data", query, nil, false)
}

func (c *clientImpl) GetSafeDeployData(ctx context.Context, wallet string) (json.RawMessage, error) {
	query := url.Values{"wallet": []string{wallet}}
	return doRaw(ctx, c, http.MethodGet, "/orders/safe-deploy-data", query, nil, false)
}

func (c *clientImpl) DeploySafe(ctx context.Context, request SafeDeployRequest) (json.RawMessage, error) {
	body, err := jsonRequestBody(request)
	if err != nil {
		return nil, err
	}
	return doRaw(ctx, c, http.MethodPost, "/orders/deploy-safe", nil, body, false)
}

func (c *clientImpl) ConfirmSafeDeployment(
	ctx context.Context,
	request ConfirmDeployRequest,
) (json.RawMessage, error) {
	body, err := jsonRequestBody(request)
	if err != nil {
		return nil, err
	}
	return doRaw(ctx, c, http.MethodPost, "/orders/confirm-deploy", nil, body, false)
}

func (c *clientImpl) GetRelayStatus(
	ctx context.Context,
	options GetRelayStatusOptions,
) (json.RawMessage, error) {
	query := url.Values{"txId": []string{options.TxID}}
	setString(query, "wallet", options.Wallet)
	return doRaw(ctx, c, http.MethodGet, "/orders/relay-status", query, nil, false)
}

func (c *clientImpl) GetTradingSetupStatus(
	ctx context.Context,
	wallet string,
) (json.RawMessage, error) {
	query := url.Values{"wallet": []string{wallet}}
	return doRaw(ctx, c, http.MethodGet, "/orders/trading-setup-status", query, nil, false)
}

func (c *clientImpl) GetApproveTokensData(
	ctx context.Context,
	wallet string,
) (json.RawMessage, error) {
	query := url.Values{"wallet": []string{wallet}}
	return doRaw(ctx, c, http.MethodGet, "/orders/approve-tokens-data", query, nil, false)
}

func (c *clientImpl) ApproveTokens(
	ctx context.Context,
	request ApproveTokensRequest,
) (json.RawMessage, error) {
	body, err := jsonRequestBody(request)
	if err != nil {
		return nil, err
	}
	return doRaw(ctx, c, http.MethodPost, "/orders/approve-tokens", nil, body, false)
}

func (c *clientImpl) GetLendingApproveData(
	ctx context.Context,
	wallet string,
) (json.RawMessage, error) {
	query := url.Values{"wallet": []string{wallet}}
	return doRaw(ctx, c, http.MethodGet, "/orders/lending-approve-data", query, nil, false)
}

func (c *clientImpl) GetLendingApproveCalldata(
	ctx context.Context,
	request LendingApproveRequest,
) (json.RawMessage, error) {
	body, err := jsonRequestBody(request)
	if err != nil {
		return nil, err
	}
	return doRaw(ctx, c, http.MethodPost, "/orders/lending-approve-calldata", nil, body, false)
}

func (c *clientImpl) GetWithdrawRelayerData(
	ctx context.Context,
	wallet string,
	amount string,
) (json.RawMessage, error) {
	query := url.Values{
		"wallet": []string{wallet},
		"amount": []string{amount},
	}
	return doRaw(ctx, c, http.MethodGet, "/orders/withdraw-relayer-data", query, nil, false)
}

func (c *clientImpl) WithdrawViaRelayer(
	ctx context.Context,
	request WithdrawRequest,
) (json.RawMessage, error) {
	body, err := jsonRequestBody(request)
	if err != nil {
		return nil, err
	}
	return doRaw(ctx, c, http.MethodPost, "/orders/withdraw-relayer", nil, body, false)
}

func (c *clientImpl) GetRepayRelayerData(
	ctx context.Context,
	wallet string,
	tokenID string,
	amount string,
) (json.RawMessage, error) {
	query := url.Values{
		"wallet":   []string{wallet},
		"token_id": []string{tokenID},
		"amount":   []string{amount},
	}
	return doRaw(ctx, c, http.MethodGet, "/orders/repay-relayer-data", query, nil, false)
}

func (c *clientImpl) RepayViaRelayer(
	ctx context.Context,
	request RepayRelayerRequest,
) (json.RawMessage, error) {
	body, err := jsonRequestBody(request)
	if err != nil {
		return nil, err
	}
	return doRaw(ctx, c, http.MethodPost, "/orders/repay-relayer", nil, body, false)
}

func (c *clientImpl) GetDepositCollateralRelayerData(
	ctx context.Context,
	wallet string,
	tokenID string,
	amount string,
) (json.RawMessage, error) {
	query := url.Values{
		"wallet":   []string{wallet},
		"token_id": []string{tokenID},
		"amount":   []string{amount},
	}
	return doRaw(
		ctx,
		c,
		http.MethodGet,
		"/orders/deposit-collateral-relayer-data",
		query,
		nil,
		false,
	)
}

func (c *clientImpl) DepositCollateralViaRelayer(
	ctx context.Context,
	request DepositCollateralRelayerRequest,
) (json.RawMessage, error) {
	body, err := jsonRequestBody(request)
	if err != nil {
		return nil, err
	}
	return doRaw(
		ctx,
		c,
		http.MethodPost,
		"/orders/deposit-collateral-relayer",
		nil,
		body,
		false,
	)
}

func (c *clientImpl) GetCLOBInfo(ctx context.Context, marketID string) (json.RawMessage, error) {
	endpoint := fmt.Sprintf("/orders/clob-info/%s", escapePath(marketID))
	return doRaw(ctx, c, http.MethodGet, endpoint, nil, nil, false)
}
