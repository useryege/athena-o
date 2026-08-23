package worm

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"math/big"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	solana "github.com/gagliardetto/solana-go"
)

const (
	wormWebPositionOpenGateEnv            = "ATHENA_WORM_LIVE_WEB_POSITION_OPEN"
	wormWebPrivateKeyEnv                  = "ATHENA_WORM_PRIVATE_KEY"
	wormWebExpectedWalletAddressEnv       = "ATHENA_WORM_WEB_EXPECTED_WALLET_ADDRESS"
	wormWebMarketConditionIDEnv           = "ATHENA_WORM_WEB_MARKET_CONDITION_ID"
	wormWebIsYesEnv                       = "ATHENA_WORM_WEB_IS_YES"
	wormWebFundsEnv                       = "ATHENA_WORM_WEB_FUNDS"
	wormWebLeverageEnv                    = "ATHENA_WORM_WEB_LEVERAGE"
	wormWebAPIBaseURL                     = "https://api.worm.wtf/api"
	wormWebHost                           = "www.worm.wtf"
	wormWebOrigin                         = "https://www.worm.wtf"
	wormWebNetworkType                    = "2"
	wormWebRequestTimeout                 = 15 * time.Second
	wormWebPositionOpenTestTimeout        = 90 * time.Second
	wormWebPositionOpenPollInterval       = time.Second
	wormWebPositionOpenPollTimeout        = 30 * time.Second
	wormWebPositionOpenPollAttempts       = 30
	wormWebMaximumResponseBodyBytes int64 = 64 << 10
	wormWebMaximumErrorMessageBytes       = 512
)

var wormWebPositiveDecimalPattern = regexp.MustCompile(`^[0-9]+(?:\.[0-9]+)?$`)

type liveWormWebPositionOpenConfig struct {
	privateKey      ed25519.PrivateKey
	walletAddress   string
	marketCondition string
	isYes           bool
	funds           string
	leverage        float64
	leverageText    string
}

type liveWormWebClient struct {
	httpClient  *http.Client
	accessToken string
}

type liveWormWebSignInChallenge struct {
	Nonce string `json:"nonce"`
}

type liveWormWebSignInRequest struct {
	Message   string `json:"message"`
	Signature string `json:"signature"`
	Address   string `json:"address"`
	Nonce     string `json:"nonce"`
}

type liveWormWebSignInResponse struct {
	AccessToken string `json:"access_token"`
}

type liveWormWebPositionOpenRequest struct {
	MarketConditionID string  `json:"market_condition_id"`
	Funds             string  `json:"funds"`
	IsYes             bool    `json:"is_yes"`
	Leverage          float64 `json:"leverage"`
}

type liveWormWebPositionFinalizeSignedTransactionRequest struct {
	PositionRequestID int64  `json:"position_request_id"`
	SignedTransaction string `json:"signed_transaction"`
}

type liveWormWebPositionFinalizeSignatureRequest struct {
	PositionRequestID int64  `json:"position_request_id"`
	Signature         string `json:"signature"`
}

type liveWormWebFinalizeMode string

const (
	liveWormWebFinalizeModeSignedTransaction liveWormWebFinalizeMode = "signed_transaction"
	liveWormWebFinalizeModeSignature         liveWormWebFinalizeMode = "signature"
)

type liveWormWebSignedPositionRequest struct {
	signatureHex         string
	signedTransactionHex string
	transactionVersion   string
	requiredSignatures   int
	signerIndex          int
	finalizeMode         liveWormWebFinalizeMode
}

type liveWormWebRequestID int64

func (id *liveWormWebRequestID) UnmarshalJSON(data []byte) error {
	encodedID := strings.TrimSpace(string(data))
	if strings.HasPrefix(encodedID, `"`) {
		var stringID string
		if err := json.Unmarshal(data, &stringID); err != nil {
			return fmt.Errorf("decode quoted Worm Web request id: %w", err)
		}
		encodedID = strings.TrimSpace(stringID)
	}
	parsedID, err := strconv.ParseInt(encodedID, 10, 64)
	if err != nil {
		return fmt.Errorf("decode Worm Web request id %q: %w", encodedID, err)
	}
	*id = liveWormWebRequestID(parsedID)
	return nil
}

type liveWormWebPositionRequest struct {
	ID          liveWormWebRequestID `json:"id"`
	Message     string               `json:"message"`
	State       string               `json:"state"`
	OrderState  string               `json:"order_state"`
	FundingTxID *string              `json:"funding_txid"`
	RefundTxID  *string              `json:"refund_txid"`
}

type liveWormWebEnvelope struct {
	Success *bool                      `json:"success"`
	Message string                     `json:"message"`
	Result  *liveWormWebEnvelopeResult `json:"result"`
	Error   *liveWormWebEnvelopeError  `json:"error"`
}

type liveWormWebEnvelopeResult struct {
	Data json.RawMessage `json:"data"`
}

type liveWormWebEnvelopeError struct {
	Code    int                 `json:"code"`
	Slug    string              `json:"slug"`
	Details []map[string]string `json:"details"`
}

type liveWormWebAPIError struct {
	StatusCode int
	Code       int
	Slug       string
	Message    string
}

func (e *liveWormWebAPIError) Error() string {
	if e == nil {
		return ""
	}
	// Worm may echo submitted values in its message. Keep the parsed message on
	// the typed error for inspection, but never include it in live test output.
	return fmt.Sprintf(
		"worm web request failed: status_code=%d code=%d slug=%s",
		e.StatusCode,
		e.Code,
		e.Slug,
	)
}

type liveWormWebTransportError struct {
	err error
}

func (e *liveWormWebTransportError) Error() string {
	return fmt.Sprintf("worm web transport failed: %v", e.err)
}

func (e *liveWormWebTransportError) Unwrap() error {
	return e.err
}

type liveWormWebResponseError struct {
	err error
}

func (e *liveWormWebResponseError) Error() string {
	return fmt.Sprintf("worm web response failed: %v", e.err)
}

func (e *liveWormWebResponseError) Unwrap() error {
	return e.err
}

func TestLiveWormWebPositionOpen(t *testing.T) {
	if strings.TrimSpace(os.Getenv(wormWebPositionOpenGateEnv)) != "1" {
		t.Skipf("set %s=1 to run the live Worm Web position open test", wormWebPositionOpenGateEnv)
	}

	config, err := loadLiveWormWebPositionOpenConfig()
	if err != nil {
		t.Fatalf("load live Worm Web position open configuration: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), wormWebPositionOpenTestTimeout)
	defer cancel()

	if err := validateLiveWormWebMarket(ctx, config); err != nil {
		t.Fatalf("validate live Worm Web market order: %v", err)
	}

	side := "NO"
	if config.isYes {
		side = "YES"
	}
	t.Logf(
		"validated Worm Web order wallet=%s market=%s side=%s funds=%s leverage=%s",
		config.walletAddress,
		config.marketCondition,
		side,
		config.funds,
		config.leverageText,
	)

	client := &liveWormWebClient{
		httpClient: &http.Client{Timeout: wormWebRequestTimeout},
	}
	accessToken, err := authenticateLiveWormWebClient(ctx, client, config.privateKey, config.walletAddress)
	if err != nil {
		t.Fatalf("authenticate with Worm Web: %v", err)
	}
	client.accessToken = accessToken

	opened, err := client.openPosition(ctx, liveWormWebPositionOpenRequest{
		MarketConditionID: config.marketCondition,
		Funds:             config.funds,
		IsYes:             config.isYes,
		Leverage:          config.leverage,
	})
	if err != nil {
		t.Fatalf("open Worm Web position request: %v", err)
	}
	if opened == nil || opened.ID <= 0 {
		t.Fatalf("open Worm Web position request returned an invalid request id")
	}
	if liveWormWebTerminalFailure(opened) {
		t.Fatalf(
			"Worm Web position request %d failed before signing: state=%s order_state=%s",
			opened.ID,
			liveWormWebState(opened.State),
			liveWormWebState(opened.OrderState),
		)
	}
	if strings.TrimSpace(opened.Message) == "" {
		t.Fatalf(
			"Worm Web position request %d returned no transaction to sign: state=%s order_state=%s",
			opened.ID,
			liveWormWebState(opened.State),
			liveWormWebState(opened.OrderState),
		)
	}
	t.Logf(
		"created Worm Web position request id=%d state=%s order_state=%s",
		opened.ID,
		liveWormWebState(opened.State),
		liveWormWebState(opened.OrderState),
	)

	signedRequest, err := signLiveWormWebTransaction(config.privateKey, opened.Message)
	if err != nil {
		t.Fatalf("sign Worm Web position request %d transaction: %v", opened.ID, err)
	}
	t.Logf(
		"signed Worm Web position request id=%d version=%s required_signatures=%d wallet_signer_index=%d finalize_mode=%s",
		opened.ID,
		signedRequest.transactionVersion,
		signedRequest.requiredSignatures,
		signedRequest.signerIndex,
		signedRequest.finalizeMode,
	)

	var finalized *liveWormWebPositionRequest
	var finalizeErr error
	switch signedRequest.finalizeMode {
	case liveWormWebFinalizeModeSignedTransaction:
		finalized, finalizeErr = client.finalizePositionWithSignedTransaction(
			ctx,
			liveWormWebPositionFinalizeSignedTransactionRequest{
				PositionRequestID: int64(opened.ID),
				SignedTransaction: signedRequest.signedTransactionHex,
			},
		)
	case liveWormWebFinalizeModeSignature:
		finalized, finalizeErr = client.finalizePositionWithSignature(
			ctx,
			liveWormWebPositionFinalizeSignatureRequest{
				PositionRequestID: int64(opened.ID),
				Signature:         signedRequest.signatureHex,
			},
		)
	default:
		t.Fatalf("sign Worm Web position request %d selected unsupported finalize mode %q", opened.ID, signedRequest.finalizeMode)
	}
	if finalizeErr != nil && !liveWormWebFinalizeNeedsStatusCheck(finalizeErr) {
		t.Fatalf("finalize Worm Web position request %d: %v", opened.ID, finalizeErr)
	}
	if finalizeErr != nil {
		t.Logf("finalize returned an error for Worm Web position request %d; checking request state before failing", opened.ID)
	} else {
		if finalized == nil || finalized.ID <= 0 {
			t.Fatalf("finalize Worm Web position request %d returned an invalid request id", opened.ID)
		}
		if finalized.ID != opened.ID {
			t.Fatalf("finalize Worm Web position request id mismatch: got %d, want %d", finalized.ID, opened.ID)
		}
		t.Logf(
			"finalized Worm Web position request id=%d state=%s order_state=%s",
			finalized.ID,
			liveWormWebState(finalized.State),
			liveWormWebState(finalized.OrderState),
		)
	}

	accepted, err := waitForLiveWormWebPositionAccepted(ctx, client, int64(opened.ID))
	if err != nil {
		if finalizeErr != nil {
			t.Fatalf(
				"finalize Worm Web position request %d was ambiguous (%v) and state confirmation failed: %v",
				opened.ID,
				finalizeErr,
				err,
			)
		}
		t.Fatalf("confirm Worm Web position request %d: %v", opened.ID, err)
	}

	t.Logf(
		"accepted Worm Web position request id=%d state=%s order_state=%s",
		accepted.ID,
		liveWormWebState(accepted.State),
		liveWormWebState(accepted.OrderState),
	)
}

func loadLiveWormWebPositionOpenConfig() (*liveWormWebPositionOpenConfig, error) {
	privateKeyText, err := requiredLiveWormWebEnv(wormWebPrivateKeyEnv)
	if err != nil {
		return nil, err
	}
	privateKey, err := parseSolanaPrivateKey(privateKeyText)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", wormWebPrivateKeyEnv, err)
	}

	expectedWallet, err := requiredLiveWormWebEnv(wormWebExpectedWalletAddressEnv)
	if err != nil {
		return nil, err
	}
	if _, err := solana.PublicKeyFromBase58(expectedWallet); err != nil {
		return nil, fmt.Errorf("parse %s: %w", wormWebExpectedWalletAddressEnv, err)
	}
	walletAddress := solanaWalletAddress(privateKey)
	if walletAddress != expectedWallet {
		return nil, fmt.Errorf(
			"%s does not match the wallet derived from %s: got %s, want %s",
			wormWebExpectedWalletAddressEnv,
			wormWebPrivateKeyEnv,
			expectedWallet,
			walletAddress,
		)
	}

	marketCondition, err := requiredLiveWormWebEnv(wormWebMarketConditionIDEnv)
	if err != nil {
		return nil, err
	}
	if _, err := solana.PublicKeyFromBase58(marketCondition); err != nil {
		return nil, fmt.Errorf("parse %s: %w", wormWebMarketConditionIDEnv, err)
	}

	isYesText, err := requiredLiveWormWebEnv(wormWebIsYesEnv)
	if err != nil {
		return nil, err
	}
	var isYes bool
	switch strings.ToLower(isYesText) {
	case "true":
		isYes = true
	case "false":
		isYes = false
	default:
		return nil, fmt.Errorf("%s must be true or false", wormWebIsYesEnv)
	}

	funds, err := requiredLiveWormWebEnv(wormWebFundsEnv)
	if err != nil {
		return nil, err
	}
	if err := validateLiveWormWebPositiveDecimal(wormWebFundsEnv, funds); err != nil {
		return nil, err
	}

	leverageText, err := requiredLiveWormWebEnv(wormWebLeverageEnv)
	if err != nil {
		return nil, err
	}
	if err := validateLiveWormWebPositiveDecimal(wormWebLeverageEnv, leverageText); err != nil {
		return nil, err
	}
	leverage, err := strconv.ParseFloat(leverageText, 64)
	if err != nil || math.IsNaN(leverage) || math.IsInf(leverage, 0) || leverage <= 0 {
		return nil, fmt.Errorf("%s must be a finite positive decimal", wormWebLeverageEnv)
	}

	return &liveWormWebPositionOpenConfig{
		privateKey:      privateKey,
		walletAddress:   walletAddress,
		marketCondition: marketCondition,
		isYes:           isYes,
		funds:           funds,
		leverage:        leverage,
		leverageText:    leverageText,
	}, nil
}

func requiredLiveWormWebEnv(name string) (string, error) {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return "", fmt.Errorf("%s is required", name)
	}
	return value, nil
}

func validateLiveWormWebPositiveDecimal(name, value string) error {
	if !wormWebPositiveDecimalPattern.MatchString(value) {
		return fmt.Errorf("%s must be a positive decimal", name)
	}
	decimal, ok := new(big.Rat).SetString(value)
	if !ok || decimal.Sign() <= 0 {
		return fmt.Errorf("%s must be greater than zero", name)
	}
	return nil
}

func validateLiveWormWebMarket(ctx context.Context, config *liveWormWebPositionOpenConfig) error {
	client, err := NewClient(Config{
		BaseURL: DefaultBaseURL,
		Timeout: wormWebRequestTimeout,
	})
	if err != nil {
		return fmt.Errorf("create public Worm client: %w", err)
	}

	market, err := client.GetMarket(ctx, config.marketCondition)
	if err != nil {
		return fmt.Errorf("get market %s: %w", config.marketCondition, err)
	}
	if market == nil || market.ConditionID != config.marketCondition {
		return fmt.Errorf("market condition mismatch for %s", config.marketCondition)
	}
	if !strings.EqualFold(strings.TrimSpace(market.State), "open") {
		return fmt.Errorf("market %s is not open: state=%s", config.marketCondition, market.State)
	}
	if !market.MarginEnabled {
		return fmt.Errorf("market %s does not support margin positions", config.marketCondition)
	}
	if market.Config == nil {
		return fmt.Errorf("market %s returned no trading config", config.marketCondition)
	}

	maximumLeverage := market.Config.MaxLeverageNo
	if config.isYes {
		maximumLeverage = market.Config.MaxLeverageYes
	}
	if maximumLeverage == nil || strings.TrimSpace(*maximumLeverage) == "" {
		return fmt.Errorf("market %s returned no maximum leverage for the selected side", config.marketCondition)
	}
	parsedMaximumLeverage, err := strconv.ParseFloat(strings.TrimSpace(*maximumLeverage), 64)
	if err != nil || math.IsNaN(parsedMaximumLeverage) || math.IsInf(parsedMaximumLeverage, 0) {
		return fmt.Errorf("parse market %s maximum leverage %q", config.marketCondition, *maximumLeverage)
	}
	if config.leverage > parsedMaximumLeverage {
		return fmt.Errorf(
			"market %s maximum leverage %s is lower than requested leverage %s",
			config.marketCondition,
			*maximumLeverage,
			config.leverageText,
		)
	}

	estimate, err := client.EstimateMarginPosition(ctx, EstimateMarginPositionOptions{
		MarketConditionID: config.marketCondition,
		Funds:             config.funds,
		IsYes:             &config.isYes,
		Leverage:          &config.leverage,
	})
	if err != nil {
		return fmt.Errorf("estimate market order: %w", err)
	}
	if estimate == nil {
		return errors.New("estimate market order returned no result")
	}
	if !estimate.IsFullyFilled {
		return errors.New("estimate market order is not fully fillable")
	}
	return nil
}

func authenticateLiveWormWebClient(
	ctx context.Context,
	client *liveWormWebClient,
	privateKey ed25519.PrivateKey,
	walletAddress string,
) (string, error) {
	challenge, err := client.getSignInChallenge(ctx, walletAddress)
	if err != nil {
		return "", fmt.Errorf("get sign-in challenge: %w", err)
	}
	if challenge == nil || strings.TrimSpace(challenge.Nonce) == "" {
		return "", errors.New("Worm Web sign-in challenge returned no nonce")
	}

	message := liveWormWebSignInMessage(walletAddress, challenge.Nonce, time.Now())
	signature := ed25519.Sign(privateKey, []byte(message))
	response, err := client.signIn(ctx, liveWormWebSignInRequest{
		Message:   message,
		Signature: hex.EncodeToString(signature),
		Address:   walletAddress,
		Nonce:     challenge.Nonce,
	})
	if err != nil {
		return "", fmt.Errorf("submit signed sign-in challenge: %w", err)
	}
	if response == nil || strings.TrimSpace(response.AccessToken) == "" {
		return "", errors.New("Worm Web sign-in returned no access token")
	}
	return strings.TrimSpace(response.AccessToken), nil
}

func liveWormWebSignInMessage(walletAddress, nonce string, now time.Time) string {
	issuedAt := now.UTC().Format("2006-01-02T15:04:05.000Z")
	return fmt.Sprintf(
		"%s wants you to sign in with your Solana account:\n%s\n\nSign in with Solana to the app.\n\nURI: %s\nVersion: 1\nChain ID: 1\nNonce: %s\nIssued At: %s",
		wormWebHost,
		walletAddress,
		wormWebOrigin,
		nonce,
		issuedAt,
	)
}

func (c *liveWormWebClient) getSignInChallenge(ctx context.Context, walletAddress string) (*liveWormWebSignInChallenge, error) {
	query := make(url.Values)
	query.Set("address", walletAddress)
	query.Set("network_type", wormWebNetworkType)
	var response liveWormWebSignInChallenge
	if err := c.do(ctx, http.MethodGet, "/sign-in/", query, nil, false, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

func (c *liveWormWebClient) signIn(ctx context.Context, request liveWormWebSignInRequest) (*liveWormWebSignInResponse, error) {
	var response liveWormWebSignInResponse
	if err := c.do(ctx, http.MethodPost, "/sign-in/", nil, request, false, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

func (c *liveWormWebClient) openPosition(
	ctx context.Context,
	request liveWormWebPositionOpenRequest,
) (*liveWormWebPositionRequest, error) {
	var response liveWormWebPositionRequest
	if err := c.do(ctx, http.MethodPost, "/margin/positions/open/", nil, request, true, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

func (c *liveWormWebClient) finalizePositionWithSignedTransaction(
	ctx context.Context,
	request liveWormWebPositionFinalizeSignedTransactionRequest,
) (*liveWormWebPositionRequest, error) {
	var response liveWormWebPositionRequest
	if err := c.do(ctx, http.MethodPost, "/margin/positions/open/finalize/", nil, request, true, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

func (c *liveWormWebClient) finalizePositionWithSignature(
	ctx context.Context,
	request liveWormWebPositionFinalizeSignatureRequest,
) (*liveWormWebPositionRequest, error) {
	var response liveWormWebPositionRequest
	if err := c.do(ctx, http.MethodPost, "/margin/positions/open/finalize/", nil, request, true, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

func (c *liveWormWebClient) getPositionRequest(ctx context.Context, requestID int64) (*liveWormWebPositionRequest, error) {
	query := make(url.Values)
	query.Set("position_request_id", strconv.FormatInt(requestID, 10))
	var response liveWormWebPositionRequest
	if err := c.do(ctx, http.MethodGet, "/margin/positions/open/", query, nil, true, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

func (c *liveWormWebClient) do(
	ctx context.Context,
	method string,
	path string,
	query url.Values,
	body any,
	authenticated bool,
	out any,
) error {
	endpoint, err := url.Parse(wormWebAPIBaseURL + path)
	if err != nil {
		return fmt.Errorf("build Worm Web endpoint: %w", err)
	}
	if len(query) > 0 {
		endpoint.RawQuery = query.Encode()
	}

	var requestBody io.Reader
	if body != nil {
		encodedBody, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("encode Worm Web request: %w", err)
		}
		requestBody = bytes.NewReader(encodedBody)
	}

	request, err := http.NewRequestWithContext(ctx, method, endpoint.String(), requestBody)
	if err != nil {
		return fmt.Errorf("create Worm Web request: %w", err)
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Origin", wormWebOrigin)
	request.Header.Set("Referer", wormWebOrigin+"/")
	request.Header.Set("User-Agent", "ATHENA Worm Web live test")
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	if authenticated {
		if strings.TrimSpace(c.accessToken) == "" {
			return errors.New("Worm Web access token is required")
		}
		request.Header.Set("Authorization", "Bearer "+c.accessToken)
	}

	response, err := c.httpClient.Do(request)
	if err != nil {
		return &liveWormWebTransportError{err: err}
	}
	defer response.Body.Close()

	rawBody, err := io.ReadAll(io.LimitReader(response.Body, wormWebMaximumResponseBodyBytes+1))
	if err != nil {
		return &liveWormWebResponseError{err: fmt.Errorf("read response body: %w", err)}
	}
	if int64(len(rawBody)) > wormWebMaximumResponseBodyBytes {
		return &liveWormWebResponseError{err: errors.New("response body exceeded size limit")}
	}
	if err := decodeLiveWormWebResponse(response.StatusCode, rawBody, out); err != nil {
		return err
	}
	return nil
}

func decodeLiveWormWebResponse(statusCode int, rawBody []byte, out any) error {
	trimmedBody := bytes.TrimSpace(rawBody)
	if len(trimmedBody) == 0 {
		return &liveWormWebResponseError{err: errors.New("empty response body")}
	}

	var envelope liveWormWebEnvelope
	if err := json.Unmarshal(trimmedBody, &envelope); err != nil {
		if statusCode < http.StatusOK || statusCode >= http.StatusMultipleChoices {
			return newLiveWormWebAPIError(statusCode, nil, "invalid error response")
		}
		return &liveWormWebResponseError{err: fmt.Errorf("decode response JSON: %w", err)}
	}

	if envelope.Success != nil {
		if !*envelope.Success || statusCode < http.StatusOK || statusCode >= http.StatusMultipleChoices {
			return newLiveWormWebAPIError(statusCode, &envelope, envelope.Message)
		}
		if envelope.Result == nil {
			return &liveWormWebResponseError{err: errors.New("successful response returned no result")}
		}
		data := bytes.TrimSpace(envelope.Result.Data)
		if len(data) == 0 || bytes.Equal(data, []byte("null")) {
			return &liveWormWebResponseError{err: errors.New("successful response returned no data")}
		}
		if out == nil {
			return nil
		}
		if err := json.Unmarshal(data, out); err != nil {
			return &liveWormWebResponseError{err: fmt.Errorf("decode response data: %w", err)}
		}
		return nil
	}

	if statusCode < http.StatusOK || statusCode >= http.StatusMultipleChoices {
		return newLiveWormWebAPIError(statusCode, &envelope, envelope.Message)
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(trimmedBody, out); err != nil {
		return &liveWormWebResponseError{err: fmt.Errorf("decode direct response: %w", err)}
	}
	return nil
}

func newLiveWormWebAPIError(statusCode int, envelope *liveWormWebEnvelope, fallbackMessage string) error {
	apiError := &liveWormWebAPIError{
		StatusCode: statusCode,
		Message:    truncateLiveWormWebErrorMessage(fallbackMessage),
	}
	if envelope != nil {
		apiError.Message = truncateLiveWormWebErrorMessage(envelope.Message)
		if envelope.Error != nil {
			apiError.Code = envelope.Error.Code
			apiError.Slug = strings.TrimSpace(envelope.Error.Slug)
		}
	}
	if apiError.Message == "" {
		apiError.Message = "request was rejected"
	}
	return apiError
}

func truncateLiveWormWebErrorMessage(message string) string {
	message = strings.Join(strings.Fields(message), " ")
	if len(message) > wormWebMaximumErrorMessageBytes {
		return message[:wormWebMaximumErrorMessageBytes] + "..."
	}
	return message
}

func signLiveWormWebTransaction(privateKey ed25519.PrivateKey, message string) (*liveWormWebSignedPositionRequest, error) {
	encodedTransaction := strings.TrimSpace(message)
	if encodedTransaction == "" {
		return nil, errors.New("Worm Web transaction is required")
	}
	if strings.HasPrefix(encodedTransaction, "0x") || strings.HasPrefix(encodedTransaction, "0X") {
		encodedTransaction = encodedTransaction[2:]
	}

	transactionBytes, err := hex.DecodeString(encodedTransaction)
	if err != nil {
		return nil, fmt.Errorf("decode Worm Web transaction: %w", err)
	}
	transaction, err := solana.TransactionFromBytes(transactionBytes)
	if err != nil {
		return nil, fmt.Errorf("parse Worm Web transaction: %w", err)
	}

	solanaPrivateKey := solana.PrivateKey(privateKey)
	signerPublicKey := solanaPrivateKey.PublicKey()
	requiredSignatures := int(transaction.Message.Header.NumRequiredSignatures)
	if requiredSignatures <= 0 || requiredSignatures > len(transaction.Message.AccountKeys) {
		return nil, fmt.Errorf(
			"invalid Worm Web transaction signer set: required=%d account_keys=%d",
			requiredSignatures,
			len(transaction.Message.AccountKeys),
		)
	}

	signerIndex := -1
	for index, accountKey := range transaction.Message.AccountKeys[:requiredSignatures] {
		if accountKey.Equals(signerPublicKey) {
			signerIndex = index
			break
		}
	}
	if signerIndex < 0 {
		return nil, fmt.Errorf("wallet %s is not a required signer for the Worm Web transaction", signerPublicKey)
	}

	signatures, err := transaction.PartialSign(func(candidate solana.PublicKey) *solana.PrivateKey {
		if candidate.Equals(signerPublicKey) {
			return &solanaPrivateKey
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("sign Worm Web transaction: %w", err)
	}
	if signerIndex >= len(signatures) {
		return nil, fmt.Errorf("Worm Web transaction signature slot %d is missing", signerIndex)
	}

	messageBytes, err := transaction.Message.MarshalBinary()
	if err != nil {
		return nil, fmt.Errorf("marshal Worm Web transaction message: %w", err)
	}
	walletSignature := signatures[signerIndex]
	if err := verifyLiveWormWebWalletSignature(signerPublicKey, messageBytes, walletSignature, signerIndex); err != nil {
		return nil, err
	}

	result := &liveWormWebSignedPositionRequest{
		signatureHex:       hex.EncodeToString(walletSignature[:]),
		transactionVersion: positionRequestTransactionVersion(transaction.Message.GetVersion()),
		requiredSignatures: requiredSignatures,
		signerIndex:        signerIndex,
	}

	switch transaction.Message.GetVersion() {
	case solana.MessageVersionLegacy:
		complete, err := liveWormWebLegacySignaturesComplete(transaction, messageBytes, requiredSignatures)
		if err != nil {
			return nil, err
		}
		if !complete {
			result.finalizeMode = liveWormWebFinalizeModeSignature
			return result, nil
		}
	case solana.MessageVersionV0:
		// Worm Web serializes versioned transactions with any other required
		// signer slots left untouched so that Worm can complete them later.
	default:
		return nil, fmt.Errorf("unsupported Worm Web transaction version %s", result.transactionVersion)
	}

	signedTransaction, err := transaction.MarshalBinary()
	if err != nil {
		return nil, fmt.Errorf("marshal signed Worm Web transaction: %w", err)
	}
	result.signedTransactionHex = hex.EncodeToString(signedTransaction)
	result.finalizeMode = liveWormWebFinalizeModeSignedTransaction
	return result, nil
}

func verifyLiveWormWebWalletSignature(
	publicKey solana.PublicKey,
	messageBytes []byte,
	signature solana.Signature,
	signerIndex int,
) error {
	if signature.IsZero() {
		return fmt.Errorf("Worm Web transaction wallet signature slot %d is empty", signerIndex)
	}
	if !publicKey.Verify(messageBytes, signature) {
		return errors.New("Worm Web transaction wallet signature verification failed")
	}
	return nil
}

func liveWormWebLegacySignaturesComplete(
	transaction *solana.Transaction,
	messageBytes []byte,
	requiredSignatures int,
) (bool, error) {
	if len(transaction.Signatures) != requiredSignatures {
		return false, fmt.Errorf(
			"invalid Worm Web legacy signature count: got %d, want %d",
			len(transaction.Signatures),
			requiredSignatures,
		)
	}
	for index, signature := range transaction.Signatures {
		if signature.IsZero() {
			return false, nil
		}
		if !transaction.Message.AccountKeys[index].Verify(messageBytes, signature) {
			return false, fmt.Errorf("Worm Web legacy signature slot %d failed verification", index)
		}
	}
	return true, nil
}

func liveWormWebFinalizeNeedsStatusCheck(err error) bool {
	var apiError *liveWormWebAPIError
	if errors.As(err, &apiError) {
		switch apiError.StatusCode {
		case http.StatusRequestTimeout,
			http.StatusConflict,
			http.StatusTooEarly,
			http.StatusTooManyRequests:
			return true
		default:
			return apiError.StatusCode >= http.StatusInternalServerError
		}
	}
	var transportError *liveWormWebTransportError
	if errors.As(err, &transportError) {
		return true
	}
	var responseError *liveWormWebResponseError
	return errors.As(err, &responseError)
}

func waitForLiveWormWebPositionAccepted(
	ctx context.Context,
	client *liveWormWebClient,
	requestID int64,
) (*liveWormWebPositionRequest, error) {
	pollContext, cancel := context.WithTimeout(ctx, wormWebPositionOpenPollTimeout)
	defer cancel()

	var lastRequest *liveWormWebPositionRequest
	var lastError error
	for attempt := 0; attempt < wormWebPositionOpenPollAttempts; attempt++ {
		if attempt > 0 {
			timer := time.NewTimer(wormWebPositionOpenPollInterval)
			select {
			case <-pollContext.Done():
				timer.Stop()
				return nil, liveWormWebPositionPollStoppedError(requestID, lastRequest, lastError, pollContext.Err())
			case <-timer.C:
			}
		}

		request, err := client.getPositionRequest(pollContext, requestID)
		if err != nil {
			if pollContext.Err() != nil {
				return nil, liveWormWebPositionPollStoppedError(requestID, lastRequest, err, pollContext.Err())
			}
			if !liveWormWebPositionStatusRetryable(err) {
				return nil, fmt.Errorf("get position request status: %w", err)
			}
			lastError = err
			continue
		}
		lastError = nil
		lastRequest = request
		if request == nil || request.ID <= 0 {
			return nil, errors.New("position status returned an invalid request id")
		}
		if request.ID != liveWormWebRequestID(requestID) {
			return nil, fmt.Errorf("position status id mismatch: got %d, want %d", request.ID, requestID)
		}
		if liveWormWebPositionAccepted(request) {
			return request, nil
		}
		if liveWormWebTerminalFailure(request) {
			return nil, fmt.Errorf(
				"position request reached terminal failure: state=%s order_state=%s",
				liveWormWebState(request.State),
				liveWormWebState(request.OrderState),
			)
		}
	}

	if lastRequest != nil {
		return nil, fmt.Errorf(
			"position request was not accepted after %d polls: state=%s order_state=%s",
			wormWebPositionOpenPollAttempts,
			liveWormWebState(lastRequest.State),
			liveWormWebState(lastRequest.OrderState),
		)
	}
	if lastError != nil {
		return nil, fmt.Errorf("position status remained unavailable after %d polls: %w", wormWebPositionOpenPollAttempts, lastError)
	}
	return nil, errors.New("position status was not observed")
}

func liveWormWebPositionStatusRetryable(err error) bool {
	var apiError *liveWormWebAPIError
	if errors.As(err, &apiError) {
		switch apiError.StatusCode {
		case http.StatusNotFound,
			http.StatusRequestTimeout,
			http.StatusConflict,
			http.StatusTooEarly,
			http.StatusTooManyRequests:
			return true
		default:
			return apiError.StatusCode >= http.StatusInternalServerError
		}
	}
	var transportError *liveWormWebTransportError
	if errors.As(err, &transportError) {
		return true
	}
	var responseError *liveWormWebResponseError
	return errors.As(err, &responseError)
}

func liveWormWebPositionPollStoppedError(
	requestID int64,
	lastRequest *liveWormWebPositionRequest,
	lastError error,
	stopError error,
) error {
	if lastRequest != nil {
		return fmt.Errorf(
			"position request %d polling stopped: state=%s order_state=%s: %w",
			requestID,
			liveWormWebState(lastRequest.State),
			liveWormWebState(lastRequest.OrderState),
			stopError,
		)
	}
	if lastError != nil {
		return fmt.Errorf("position request %d polling stopped after status error %v: %w", requestID, lastError, stopError)
	}
	return fmt.Errorf("position request %d polling stopped before status was observed: %w", requestID, stopError)
}

func liveWormWebPositionAccepted(request *liveWormWebPositionRequest) bool {
	if request == nil {
		return false
	}
	state := liveWormWebState(request.State)
	if state == "completed" {
		return true
	}
	if state != "processing" {
		return false
	}
	orderState := liveWormWebState(request.OrderState)
	return orderState == "created" || orderState == "opened"
}

func liveWormWebTerminalFailure(request *liveWormWebPositionRequest) bool {
	if request == nil {
		return false
	}
	switch liveWormWebState(request.State) {
	case "failed", "cancelled":
		return true
	default:
		return false
	}
}

func liveWormWebState(state string) string {
	state = strings.ToLower(strings.TrimSpace(state))
	if state == "" {
		return "-"
	}
	return state
}
