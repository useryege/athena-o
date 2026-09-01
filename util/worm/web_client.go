package worm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"math/big"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	WebAPIBaseURL              = "https://api.worm.wtf/api"
	WebOrigin                  = "https://www.worm.wtf"
	WebHost                    = "www.worm.wtf"
	WebNetworkType             = "2"
	DefaultWebRequestTimeout   = 15 * time.Second
	WebErrorCodeEdgeBlocked    = "WORM_EDGE_BLOCKED"
	webMaximumResponseBodySize = int64(64 << 10)
	webMaximumErrorMessageSize = 512
)

var (
	webPositiveDecimalPattern = regexp.MustCompile(`^[0-9]+(?:\.[0-9]+)?$`)
	webStableSlugPattern      = regexp.MustCompile(`^[A-Za-z0-9_][A-Za-z0-9_.-]{0,99}$`)
)

// WebMarketPositionOpenClient is the single-mutation surface required to
// dispatch one prepared Worm Web market-position Open.
type WebMarketPositionOpenClient interface {
	OpenMarketPosition(ctx context.Context, accessToken string, request WebMarketPositionOpenRequest) (*WebPositionRequest, error)
}

// WebPositionFinalizeClient is the single-mutation surface required to
// dispatch one prepared Worm Web Finalize.
type WebPositionFinalizeClient interface {
	FinalizePosition(ctx context.Context, accessToken string, request WebPositionFinalizeRequest) (*WebPositionRequest, error)
}

// WebPositionRequestClient is the read-only surface used to observe one Worm
// Web request after a durable caller has recorded its numeric identity.
type WebPositionRequestClient interface {
	GetPositionRequest(ctx context.Context, accessToken string, requestID int64) (*WebPositionRequest, error)
}

// WebMarketPositionSubmitClient is the complete narrow protocol required by
// the convenience SubmitWebMarketPosition orchestration. Durable production
// execution may instead depend on the individual stage interfaces above.
type WebMarketPositionSubmitClient interface {
	WebSignInClient
	WebMarketPositionOpenClient
	WebPositionFinalizeClient
	WebPositionRequestClient
}

// WebMarginPositionCashOutClient is the narrow Worm Web protocol required by
// CashOutWebMarginPosition.
type WebMarginPositionCashOutClient interface {
	WebSignInClient
	ListMarginPositions(ctx context.Context, accessToken string, options WebMarginPositionListOptions) ([]WebMarginPosition, error)
	CloseMarginPosition(ctx context.Context, accessToken string, request WebMarginPositionCloseRequest) error
}

// WebClient implements the fixed official Worm Web JWT flows. Access tokens
// are supplied per call so one client can safely serve multiple managed
// wallets. It deliberately contains no retry behavior; callers must never
// replay Open, Finalize, or Close after a request may have reached Worm.
type WebClient interface {
	WebMarketPositionSubmitClient
	WebMarginPositionCashOutClient
}

type WebClientConfig struct {
	Timeout    time.Duration
	HTTPClient *http.Client
}

type webClient struct {
	httpClient *http.Client
}

var _ WebClient = (*webClient)(nil)

func NewWebClient(config WebClientConfig) WebClient {
	timeout := config.Timeout
	if timeout <= 0 {
		timeout = DefaultWebRequestTimeout
	}

	client := http.Client{Timeout: timeout}
	if config.HTTPClient != nil {
		client = *config.HTTPClient
		if client.Timeout <= 0 {
			client.Timeout = timeout
		}
	}
	previousRedirectPolicy := client.CheckRedirect
	client.CheckRedirect = func(request *http.Request, via []*http.Request) error {
		if len(via) == 0 {
			return errors.New("worm web redirect has no originating request")
		}
		if !strings.EqualFold(request.URL.Hostname(), via[0].URL.Hostname()) {
			return errors.New("worm web cross-host redirect refused")
		}
		if request.URL.Scheme != "https" || request.URL.Scheme != via[0].URL.Scheme {
			return errors.New("worm web insecure redirect refused")
		}
		if via[0].Method != http.MethodGet && via[0].Method != http.MethodHead {
			return errors.New("worm web mutation redirect refused")
		}
		if len(via) >= 10 {
			return errors.New("worm web redirect limit exceeded")
		}
		if previousRedirectPolicy != nil {
			return previousRedirectPolicy(request, via)
		}
		return nil
	}
	return &webClient{httpClient: &client}
}

type WebSignInChallenge struct {
	Nonce string `json:"nonce"`
}

type WebSignInRequest struct {
	Message   string `json:"message"`
	Signature string `json:"signature"`
	Address   string `json:"address"`
	Nonce     string `json:"nonce"`
}

type WebSignInResponse struct {
	AccessToken string `json:"access_token"`
}

type WebMarginPositionID int64

func (id *WebMarginPositionID) UnmarshalJSON(data []byte) error {
	parsedID, err := decodeWebPositiveID(
		data,
		"Worm Web margin position id",
		"margin position id must be positive",
	)
	if err != nil {
		return err
	}
	*id = WebMarginPositionID(parsedID)
	return nil
}

type WebMarginPositionState string

const (
	WebMarginPositionStateOpen       WebMarginPositionState = "open"
	WebMarginPositionStateClosed     WebMarginPositionState = "closed"
	WebMarginPositionStateLiquidated WebMarginPositionState = "liquidated"
	WebMarginPositionStateClosing    WebMarginPositionState = "closing"
)

func (state *WebMarginPositionState) UnmarshalJSON(data []byte) error {
	encodedState := strings.TrimSpace(string(data))
	if encodedState == "null" {
		*state = ""
		return nil
	}
	if strings.HasPrefix(encodedState, `"`) {
		var stringState string
		if err := json.Unmarshal(data, &stringState); err != nil {
			return fmt.Errorf("decode Worm Web margin position state: %w", err)
		}
		encodedState = stringState
	} else {
		if _, err := strconv.ParseInt(encodedState, 10, 64); err != nil {
			return fmt.Errorf("decode Worm Web margin position state %q: %w", encodedState, err)
		}
	}

	normalizedState := strings.ToLower(strings.TrimSpace(encodedState))
	switch normalizedState {
	case "1", string(WebMarginPositionStateOpen):
		normalizedState = string(WebMarginPositionStateOpen)
	case "2", string(WebMarginPositionStateClosed):
		normalizedState = string(WebMarginPositionStateClosed)
	case "3", string(WebMarginPositionStateLiquidated):
		normalizedState = string(WebMarginPositionStateLiquidated)
	case "4", string(WebMarginPositionStateClosing):
		normalizedState = string(WebMarginPositionStateClosing)
	}
	*state = WebMarginPositionState(normalizedState)
	return nil
}

type WebMarginPositionMarket struct {
	ConditionID string `json:"condition_id"`
}

// WebMarginPosition models only the identity and lifecycle fields required to
// safely select and observe a Web cash-out target. Pointer booleans preserve
// missing-field drift so a malformed open position cannot be closed.
type WebMarginPosition struct {
	PositionID   WebMarginPositionID      `json:"position_id"`
	Market       *WebMarginPositionMarket `json:"market"`
	IsYes        *bool                    `json:"is_yes"`
	IsClosed     *bool                    `json:"is_closed"`
	IsLiquidated *bool                    `json:"is_liquidated"`
	State        WebMarginPositionState   `json:"state"`
}

type WebMarginPositionListOptions struct {
	MarketConditionID string
}

// WebMarginPositionCloseRequest is the complete Worm Web cash-out payload.
// The endpoint closes the entire position and exposes no price or shares.
type WebMarginPositionCloseRequest struct {
	MarketConditionID string `json:"market_condition_id"`
	IsYes             bool   `json:"is_yes"`
	PositionID        int64  `json:"position_id"`
}

// WebMarketPositionOpenRequest is the only Worm Execution order shape. The
// Web endpoint has no order-type, limit-price, or share fields, so callers
// cannot turn this request into a limit order.
type WebMarketPositionOpenRequest struct {
	MarketConditionID string  `json:"market_condition_id"`
	Funds             string  `json:"funds"`
	IsYes             bool    `json:"is_yes"`
	Leverage          float64 `json:"leverage"`
}

type WebFinalizeMode string

const (
	WebFinalizeModeSignature         WebFinalizeMode = "signature"
	WebFinalizeModeSignedTransaction WebFinalizeMode = "signed_transaction"
)

// WebPositionFinalizePayload is a tagged union. Exactly one payload value is
// serialized according to Mode, and callers must keep that choice durable once
// a Finalize request is dispatched.
type WebPositionFinalizePayload struct {
	Mode  WebFinalizeMode
	Value string
}

type WebPositionFinalizeRequest struct {
	PositionRequestID int64
	Payload           WebPositionFinalizePayload
}

type webPositionFinalizeSignatureRequest struct {
	PositionRequestID int64  `json:"position_request_id"`
	Signature         string `json:"signature"`
}

type webPositionFinalizeSignedTransactionRequest struct {
	PositionRequestID int64  `json:"position_request_id"`
	SignedTransaction string `json:"signed_transaction"`
}

type WebRequestID int64

func (id *WebRequestID) UnmarshalJSON(data []byte) error {
	parsedID, err := decodeWebPositiveID(data, "Worm Web request id", "request id must be positive")
	if err != nil {
		return err
	}
	*id = WebRequestID(parsedID)
	return nil
}

func decodeWebPositiveID(data []byte, fieldName, positiveError string) (int64, error) {
	encodedID := strings.TrimSpace(string(data))
	if strings.HasPrefix(encodedID, `"`) {
		var stringID string
		if err := json.Unmarshal(data, &stringID); err != nil {
			return 0, fmt.Errorf("decode quoted %s: %w", fieldName, err)
		}
		encodedID = strings.TrimSpace(stringID)
	}
	parsedID, err := strconv.ParseInt(encodedID, 10, 64)
	if err != nil || parsedID <= 0 {
		if err == nil {
			err = errors.New(positiveError)
		}
		return 0, fmt.Errorf("decode %s %q: %w", fieldName, encodedID, err)
	}
	return parsedID, nil
}

type WebPositionRequest struct {
	ID          WebRequestID `json:"id"`
	Message     string       `json:"message"`
	State       string       `json:"state"`
	OrderState  string       `json:"order_state"`
	FundingTxID *string      `json:"funding_txid"`
	RefundTxID  *string      `json:"refund_txid"`
}

type webEnvelope struct {
	Success *bool              `json:"success"`
	Message string             `json:"message"`
	Result  *webEnvelopeResult `json:"result"`
	Error   *webEnvelopeError  `json:"error"`
}

type webEnvelopeResult struct {
	Data json.RawMessage `json:"data"`
}

type webEnvelopeError struct {
	Code    int              `json:"code"`
	Slug    string           `json:"slug"`
	Details []map[string]any `json:"details"`
}

// WebAPIError is a structured JSON rejection from Worm. Error intentionally
// omits Message because upstream may echo submitted values; callers may inspect
// the bounded field but must not log it.
type WebAPIError struct {
	StatusCode     int
	Code           int
	Slug           string
	Message        string
	StructuredJSON bool
}

func (e *WebAPIError) Error() string {
	if e == nil {
		return ""
	}
	return fmt.Sprintf("worm web request failed: status_code=%d code=%d", e.StatusCode, e.Code)
}

type WebEdgeBlockedError struct {
	StatusCode int
	Code       string
}

func (e *WebEdgeBlockedError) Error() string {
	if e == nil {
		return ""
	}
	return fmt.Sprintf("worm web request blocked at edge: status_code=%d code=%s", e.StatusCode, e.Code)
}

type WebTransportError struct {
	err error
}

func (e *WebTransportError) Error() string {
	return fmt.Sprintf("worm web transport failed: %v", e.err)
}
func (e *WebTransportError) Unwrap() error { return e.err }

type WebResponseError struct {
	err error
}

func (e *WebResponseError) Error() string { return fmt.Sprintf("worm web response failed: %v", e.err) }
func (e *WebResponseError) Unwrap() error { return e.err }

// BuildWebSignInMessage creates the exact SIWS-shaped message used by the Worm
// Web application. The Wallet service separately validates this complete shape
// before signing it.
func BuildWebSignInMessage(walletAddress, nonce string, issuedAt time.Time) string {
	return fmt.Sprintf(
		"%s wants you to sign in with your Solana account:\n%s\n\nSign in with Solana to the app.\n\nURI: %s\nVersion: 1\nChain ID: 1\nNonce: %s\nIssued At: %s",
		WebHost,
		walletAddress,
		WebOrigin,
		nonce,
		issuedAt.UTC().Format("2006-01-02T15:04:05.000Z"),
	)
}

func (c *webClient) GetSignInChallenge(ctx context.Context, walletAddress string) (*WebSignInChallenge, error) {
	walletAddress = strings.TrimSpace(walletAddress)
	if walletAddress == "" {
		return nil, errors.New("Worm Web wallet address is required")
	}
	query := make(url.Values)
	query.Set("address", walletAddress)
	query.Set("network_type", WebNetworkType)
	var response WebSignInChallenge
	if err := c.do(ctx, http.MethodGet, "/sign-in/", query, nil, "", &response); err != nil {
		return nil, err
	}
	response.Nonce = strings.TrimSpace(response.Nonce)
	if response.Nonce == "" {
		return nil, &WebResponseError{err: errors.New("sign-in challenge returned no nonce")}
	}
	return &response, nil
}

func (c *webClient) SignIn(ctx context.Context, request WebSignInRequest) (*WebSignInResponse, error) {
	if strings.TrimSpace(request.Message) == "" || strings.TrimSpace(request.Signature) == "" ||
		strings.TrimSpace(request.Address) == "" || strings.TrimSpace(request.Nonce) == "" {
		return nil, errors.New("Worm Web sign-in message, signature, address, and nonce are required")
	}
	var response WebSignInResponse
	if err := c.do(ctx, http.MethodPost, "/sign-in/", nil, request, "", &response); err != nil {
		return nil, err
	}
	response.AccessToken = strings.TrimSpace(response.AccessToken)
	if response.AccessToken == "" {
		return nil, &WebResponseError{err: errors.New("sign-in returned no access token")}
	}
	return &response, nil
}

func (c *webClient) OpenMarketPosition(ctx context.Context, accessToken string, request WebMarketPositionOpenRequest) (*WebPositionRequest, error) {
	if err := validateWebMarketPositionOpenRequest(request); err != nil {
		return nil, err
	}
	var response WebPositionRequest
	err := c.doWithDecoder(ctx, http.MethodPost, "/margin/positions/open/", nil, request, accessToken,
		func(statusCode int, rawBody []byte) error {
			return decodeWebPositionRequestResponse(statusCode, rawBody, &response)
		})
	if err != nil {
		if response.ID > 0 {
			return &response, err
		}
		return nil, err
	}
	if response.ID <= 0 {
		return nil, &WebResponseError{err: errors.New("position open returned no request id")}
	}
	return &response, nil
}

func validateWebMarketPositionOpenRequest(request WebMarketPositionOpenRequest) error {
	if strings.TrimSpace(request.MarketConditionID) == "" || strings.TrimSpace(request.Funds) == "" {
		return errors.New("Worm Web market condition id and funds are required")
	}
	if request.Funds != strings.TrimSpace(request.Funds) || !webPositiveDecimalPattern.MatchString(request.Funds) {
		return errors.New("Worm Web funds must be a canonical positive decimal")
	}
	funds, ok := new(big.Rat).SetString(request.Funds)
	if !ok || funds.Sign() <= 0 || funds.Cmp(big.NewRat(10, 1)) > 0 {
		return errors.New("Worm Web funds must be positive and at most 10 USDC")
	}
	if request.Leverage != 1 || math.IsNaN(request.Leverage) || math.IsInf(request.Leverage, 0) {
		return errors.New("Worm Web leverage must be exactly one-times")
	}
	return nil
}

func (c *webClient) FinalizePosition(ctx context.Context, accessToken string, request WebPositionFinalizeRequest) (*WebPositionRequest, error) {
	if request.PositionRequestID <= 0 {
		return nil, errors.New("Worm Web position request id must be positive")
	}
	payloadValue := strings.TrimSpace(request.Payload.Value)
	if payloadValue == "" {
		return nil, errors.New("Worm Web finalize payload is required")
	}

	var body any
	switch request.Payload.Mode {
	case WebFinalizeModeSignature:
		body = webPositionFinalizeSignatureRequest{PositionRequestID: request.PositionRequestID, Signature: payloadValue}
	case WebFinalizeModeSignedTransaction:
		body = webPositionFinalizeSignedTransactionRequest{PositionRequestID: request.PositionRequestID, SignedTransaction: payloadValue}
	default:
		return nil, fmt.Errorf("unsupported Worm Web finalize mode %q", request.Payload.Mode)
	}

	var response WebPositionRequest
	err := c.doWithDecoder(ctx, http.MethodPost, "/margin/positions/open/finalize/", nil, body, accessToken,
		func(statusCode int, rawBody []byte) error {
			return decodeWebPositionRequestResponse(statusCode, rawBody, &response)
		})
	if err != nil {
		if response.ID > 0 {
			return &response, err
		}
		return nil, err
	}
	if int64(response.ID) != request.PositionRequestID {
		return nil, &WebResponseError{err: fmt.Errorf(
			"finalize response request id mismatch: got %d, want %d",
			response.ID,
			request.PositionRequestID,
		)}
	}
	return &response, nil
}

func (c *webClient) GetPositionRequest(ctx context.Context, accessToken string, requestID int64) (*WebPositionRequest, error) {
	if requestID <= 0 {
		return nil, errors.New("Worm Web position request id must be positive")
	}
	query := make(url.Values)
	query.Set("position_request_id", strconv.FormatInt(requestID, 10))
	var response WebPositionRequest
	err := c.doWithDecoder(ctx, http.MethodGet, "/margin/positions/open/", query, nil, accessToken,
		func(statusCode int, rawBody []byte) error {
			return decodeWebPositionRequestResponse(statusCode, rawBody, &response)
		})
	if err != nil {
		if response.ID > 0 {
			return &response, err
		}
		return nil, err
	}
	if int64(response.ID) != requestID {
		return nil, &WebResponseError{err: fmt.Errorf("position response request id mismatch: got %d, want %d", response.ID, requestID)}
	}
	return &response, nil
}

func (c *webClient) ListMarginPositions(
	ctx context.Context,
	accessToken string,
	options WebMarginPositionListOptions,
) ([]WebMarginPosition, error) {
	marketConditionID := strings.TrimSpace(options.MarketConditionID)
	if marketConditionID == "" {
		return nil, errors.New("Worm Web market condition id is required")
	}
	query := make(url.Values)
	query.Set("market_condition_id", marketConditionID)
	var response []WebMarginPosition
	if err := c.do(ctx, http.MethodGet, "/margin/positions/", query, nil, accessToken, &response); err != nil {
		return nil, err
	}
	return response, nil
}

func (c *webClient) CloseMarginPosition(
	ctx context.Context,
	accessToken string,
	request WebMarginPositionCloseRequest,
) error {
	if strings.TrimSpace(request.MarketConditionID) == "" {
		return errors.New("Worm Web market condition id is required")
	}
	if request.PositionID <= 0 {
		return errors.New("Worm Web margin position id must be positive")
	}
	return c.doAcknowledgement(
		ctx,
		http.MethodPost,
		"/margin/positions/close/",
		nil,
		request,
		accessToken,
	)
}

func (c *webClient) do(
	ctx context.Context,
	method string,
	path string,
	query url.Values,
	body any,
	accessToken string,
	out any,
) error {
	return c.doWithDecoder(ctx, method, path, query, body, accessToken, func(statusCode int, rawBody []byte) error {
		return decodeWebResponse(statusCode, rawBody, out)
	})
}

func (c *webClient) doAcknowledgement(
	ctx context.Context,
	method string,
	path string,
	query url.Values,
	body any,
	accessToken string,
) error {
	return c.doWithDecoder(ctx, method, path, query, body, accessToken, decodeWebAcknowledgement)
}

func (c *webClient) doWithDecoder(
	ctx context.Context,
	method string,
	path string,
	query url.Values,
	body any,
	accessToken string,
	decode func(statusCode int, rawBody []byte) error,
) error {
	endpoint, err := url.Parse(WebAPIBaseURL + path)
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
	request.Header.Set("Origin", WebOrigin)
	request.Header.Set("Referer", WebOrigin+"/")
	request.Header.Set("User-Agent", "ATHENA Worm Web execution")
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	if strings.TrimSpace(accessToken) != "" {
		request.Header.Set("Authorization", "Bearer "+strings.TrimSpace(accessToken))
	} else if path != "/sign-in/" {
		return errors.New("Worm Web access token is required")
	}

	response, err := c.httpClient.Do(request)
	if err != nil {
		return &WebTransportError{err: err}
	}
	defer response.Body.Close()
	rawBody, err := io.ReadAll(io.LimitReader(response.Body, webMaximumResponseBodySize+1))
	if err != nil {
		return &WebResponseError{err: fmt.Errorf("read response body: %w", err)}
	}
	if response.StatusCode == http.StatusForbidden && looksLikeHTML(response.Header.Get("Content-Type"), rawBody) {
		return &WebEdgeBlockedError{StatusCode: response.StatusCode, Code: WebErrorCodeEdgeBlocked}
	}
	if int64(len(rawBody)) > webMaximumResponseBodySize {
		return &WebResponseError{err: errors.New("response body exceeded 64 KiB limit")}
	}
	return decode(response.StatusCode, rawBody)
}

func decodeWebResponse(statusCode int, rawBody []byte, out any) error {
	trimmedBody := bytes.TrimSpace(rawBody)
	if len(trimmedBody) == 0 {
		return &WebResponseError{err: errors.New("empty response body")}
	}
	if bytes.Equal(trimmedBody, []byte("null")) {
		return &WebResponseError{err: errors.New("successful response returned no data")}
	}
	if trimmedBody[0] == '[' {
		if statusCode < http.StatusOK || statusCode >= http.StatusMultipleChoices {
			return newWebAPIError(statusCode, nil, "invalid error response")
		}
		if out == nil {
			return nil
		}
		if err := json.Unmarshal(trimmedBody, out); err != nil {
			return &WebResponseError{err: fmt.Errorf("decode direct response: %w", err)}
		}
		return nil
	}

	var envelope webEnvelope
	if err := json.Unmarshal(trimmedBody, &envelope); err != nil {
		if statusCode < http.StatusOK || statusCode >= http.StatusMultipleChoices {
			return newWebAPIError(statusCode, nil, "invalid error response")
		}
		return &WebResponseError{err: fmt.Errorf("decode response JSON: %w", err)}
	}
	if envelope.Error != nil {
		return newWebAPIError(statusCode, &envelope, envelope.Message)
	}
	if envelope.Success != nil {
		if !*envelope.Success || statusCode < http.StatusOK || statusCode >= http.StatusMultipleChoices {
			return newWebAPIError(statusCode, &envelope, envelope.Message)
		}
		if envelope.Result == nil {
			return &WebResponseError{err: errors.New("successful response returned no result")}
		}
		data := bytes.TrimSpace(envelope.Result.Data)
		if len(data) == 0 || bytes.Equal(data, []byte("null")) {
			return &WebResponseError{err: errors.New("successful response returned no data")}
		}
		if out == nil {
			return nil
		}
		if err := json.Unmarshal(data, out); err != nil {
			return &WebResponseError{err: fmt.Errorf("decode response data: %w", err)}
		}
		return nil
	}

	if statusCode < http.StatusOK || statusCode >= http.StatusMultipleChoices {
		return newWebAPIError(statusCode, &envelope, envelope.Message)
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(trimmedBody, out); err != nil {
		return &WebResponseError{err: fmt.Errorf("decode direct response: %w", err)}
	}
	return nil
}

// decodeWebPositionRequestResponse preserves only a valid request identity
// when the remaining DTO is malformed. Durable callers can checkpoint that
// identity and reconcile with safe reads without trusting partially decoded
// provider state or replaying the mutation.
func decodeWebPositionRequestResponse(
	statusCode int,
	rawBody []byte,
	out *WebPositionRequest,
) error {
	if out == nil {
		return &WebResponseError{err: errors.New("position request response target is missing")}
	}
	if err := decodeWebResponse(statusCode, rawBody, out); err != nil {
		requestID, ok := extractWebPositionRequestID(rawBody)
		*out = WebPositionRequest{}
		if ok {
			out.ID = requestID
		}
		return err
	}
	return nil
}

func extractWebPositionRequestID(rawBody []byte) (WebRequestID, bool) {
	payload := bytes.TrimSpace(rawBody)
	if len(payload) == 0 || bytes.Equal(payload, []byte("null")) || payload[0] == '[' {
		return 0, false
	}

	var envelope struct {
		Success *bool              `json:"success"`
		Result  *webEnvelopeResult `json:"result"`
		Error   json.RawMessage    `json:"error"`
	}
	if err := json.Unmarshal(payload, &envelope); err != nil {
		return 0, false
	}
	if envelope.Result != nil {
		payload = bytes.TrimSpace(envelope.Result.Data)
		if len(payload) == 0 || bytes.Equal(payload, []byte("null")) {
			return 0, false
		}
	} else if envelope.Success != nil || len(bytes.TrimSpace(envelope.Error)) != 0 {
		return 0, false
	}

	var identity struct {
		ID WebRequestID `json:"id"`
	}
	if err := json.Unmarshal(payload, &identity); err != nil || identity.ID <= 0 {
		return 0, false
	}
	return identity.ID, true
}

func decodeWebAcknowledgement(statusCode int, rawBody []byte) error {
	trimmedBody := bytes.TrimSpace(rawBody)
	if len(trimmedBody) == 0 {
		if statusCode >= http.StatusOK && statusCode < http.StatusMultipleChoices {
			return nil
		}
		return newWebAPIError(statusCode, nil, "empty error response")
	}

	var envelope webEnvelope
	if err := json.Unmarshal(trimmedBody, &envelope); err != nil {
		if statusCode < http.StatusOK || statusCode >= http.StatusMultipleChoices {
			return newWebAPIError(statusCode, nil, "invalid error response")
		}
		return &WebResponseError{err: fmt.Errorf("decode response JSON: %w", err)}
	}
	if envelope.Error != nil {
		return newWebAPIError(statusCode, &envelope, envelope.Message)
	}
	if envelope.Success != nil {
		if !*envelope.Success || statusCode < http.StatusOK || statusCode >= http.StatusMultipleChoices {
			return newWebAPIError(statusCode, &envelope, envelope.Message)
		}
		return nil
	}
	if statusCode < http.StatusOK || statusCode >= http.StatusMultipleChoices {
		return newWebAPIError(statusCode, &envelope, envelope.Message)
	}
	return nil
}

func newWebAPIError(statusCode int, envelope *webEnvelope, fallbackMessage string) error {
	apiError := &WebAPIError{
		StatusCode:     statusCode,
		Message:        truncateWebErrorMessage(fallbackMessage),
		StructuredJSON: envelope != nil && (envelope.Success != nil || envelope.Error != nil || strings.TrimSpace(envelope.Message) != ""),
	}
	if envelope != nil {
		apiError.Message = truncateWebErrorMessage(envelope.Message)
		if envelope.Error != nil {
			apiError.Code = envelope.Error.Code
			slug := strings.TrimSpace(envelope.Error.Slug)
			if webStableSlugPattern.MatchString(slug) {
				apiError.Slug = slug
			}
		}
	}
	if apiError.Message == "" {
		apiError.Message = "request was rejected"
	}
	return apiError
}

func truncateWebErrorMessage(message string) string {
	message = strings.Join(strings.Fields(message), " ")
	if len(message) > webMaximumErrorMessageSize {
		return message[:webMaximumErrorMessageSize] + "..."
	}
	return message
}

func looksLikeHTML(contentType string, body []byte) bool {
	contentType = strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0]))
	trimmedBody := bytes.TrimSpace(body)
	return contentType == "text/html" || bytes.HasPrefix(bytes.ToLower(trimmedBody), []byte("<!doctype html")) ||
		bytes.HasPrefix(bytes.ToLower(trimmedBody), []byte("<html"))
}
