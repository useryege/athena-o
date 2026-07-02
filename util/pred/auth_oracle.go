package pred

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

func (c *clientImpl) GetNonce(ctx context.Context, address string) (*NonceResponse, error) {
	return doTyped[NonceResponse](
		ctx,
		c,
		http.MethodGet,
		"/auth/nonce",
		url.Values{"address": []string{address}},
		nil,
		false,
	)
}

func (c *clientImpl) VerifySignature(ctx context.Context, request SignatureVerifyRequest) (*AuthResponse, error) {
	body, err := jsonRequestBody(request)
	if err != nil {
		return nil, err
	}
	return doTyped[AuthResponse](ctx, c, http.MethodPost, "/auth/verify", nil, body, false)
}

func (c *clientImpl) GetBorrowSigningData(
	ctx context.Context,
	borrower string,
	recipient string,
	tokenID string,
	amount string,
) (json.RawMessage, error) {
	endpoint := fmt.Sprintf(
		"/oracle/borrow-intent/%s/%s/%s/%s",
		escapePath(borrower),
		escapePath(recipient),
		escapePath(tokenID),
		escapePath(amount),
	)
	return doRaw(ctx, c, http.MethodGet, endpoint, nil, nil, false)
}

func (c *clientImpl) SubmitBorrowRelay(ctx context.Context, request BorrowIntentRequest) (*RelayResponse, error) {
	body, err := jsonRequestBody(request)
	if err != nil {
		return nil, err
	}
	return doTyped[RelayResponse](ctx, c, http.MethodPost, "/oracle/borrow-relay", nil, body, false)
}

func (c *clientImpl) GetWithdrawSigningData(
	ctx context.Context,
	borrower string,
	to string,
	tokenID string,
	amount string,
) (json.RawMessage, error) {
	endpoint := fmt.Sprintf(
		"/oracle/withdraw-intent/%s/%s/%s/%s",
		escapePath(borrower),
		escapePath(to),
		escapePath(tokenID),
		escapePath(amount),
	)
	return doRaw(ctx, c, http.MethodGet, endpoint, nil, nil, false)
}

func (c *clientImpl) SubmitWithdrawRelay(ctx context.Context, request WithdrawIntentRequest) (*RelayResponse, error) {
	body, err := jsonRequestBody(request)
	if err != nil {
		return nil, err
	}
	return doTyped[RelayResponse](ctx, c, http.MethodPost, "/oracle/withdraw-relay", nil, body, false)
}
