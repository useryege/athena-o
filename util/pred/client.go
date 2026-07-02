package pred

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	DefaultBaseURL = "https://api.predmart.com"
	DefaultTimeout = 30 * time.Second

	errorBodyLimit = 16 * 1024
)

var ErrBearerTokenRequired = errors.New("pred bearer token is required")

// Client is a typed client for the core PredMart Swagger API.
type Client interface {
	// Authentication
	GetNonce(ctx context.Context, address string) (*NonceResponse, error)
	VerifySignature(ctx context.Context, request SignatureVerifyRequest) (*AuthResponse, error)

	// Oracle
	GetBorrowSigningData(ctx context.Context, borrower, recipient, tokenID, amount string) (json.RawMessage, error)
	SubmitBorrowRelay(ctx context.Context, request BorrowIntentRequest) (*RelayResponse, error)
	GetWithdrawSigningData(ctx context.Context, borrower, to, tokenID, amount string) (json.RawMessage, error)
	SubmitWithdrawRelay(ctx context.Context, request WithdrawIntentRequest) (*RelayResponse, error)

	// Lending
	GetPoolStats(ctx context.Context) (*PoolStats, error)
	GetInterestCorrection(ctx context.Context, address string) (json.RawMessage, error)
	GetLendingPosition(ctx context.Context, address, tokenID string, options GetLendingPositionOptions) (*PositionInfo, error)
	ListLendingPositions(ctx context.Context, address string) (*UserPositionsResponse, error)
	ListLendingEvents(ctx context.Context, address string, options ListLendingEventsOptions) (*EventHistoryResponse, error)
	ListLiquidations(ctx context.Context, address string, options ListLiquidationsOptions) (*LiquidationHistoryResponse, error)
	GetRateHistory(ctx context.Context, options GetRateHistoryOptions) (*RateHistoryResponse, error)
	GetRealAPY(ctx context.Context) (json.RawMessage, error)
	GetDepthStatus(ctx context.Context, tokenIDs string) (*DepthStatusResponse, error)
	GetOraclePrice(ctx context.Context, tokenID string) (json.RawMessage, error)
	GetOraclePrices(ctx context.Context, tokenIDs string) (json.RawMessage, error)
	GetOrderbookAsks(ctx context.Context, tokenID string) (json.RawMessage, error)
	GetContractInfo(ctx context.Context) (*ContractInfo, error)

	// Leverage
	GetLeverageAuthData(ctx context.Context, options GetLeverageAuthDataOptions) (json.RawMessage, error)
	GetLeverageStatus(ctx context.Context, leverageID string) (*LeverageStatusResponse, error)
	OpenLeverage(ctx context.Context, request LeverageOpenRequest) (*LeverageOpenResponse, error)
	GetFlashCloseAuthData(ctx context.Context, options GetFlashCloseAuthDataOptions) (json.RawMessage, error)
	OpenFlashClose(ctx context.Context, request FlashCloseOpenRequest) (json.RawMessage, error)
	GetFlashCloseStatus(ctx context.Context, closeID string) (json.RawMessage, error)

	// Take-Profit / Stop-Loss
	UpsertTPSL(ctx context.Context, request TPSLCreateRequest) (*TPSLResponse, error)
	GetTPSL(ctx context.Context, tokenID string) (*TPSLResponse, error)
	CancelTPSL(ctx context.Context, tokenID string) (json.RawMessage, error)
	GetTPSLAuthData(ctx context.Context, tokenID string) (json.RawMessage, error)
	TriggerTPSLTest(ctx context.Context, options TriggerTPSLTestOptions) (json.RawMessage, error)

	// Users
	GetUser(ctx context.Context, identifier string) (*UserResponse, error)
	EnsureUser(ctx context.Context, request EnsureUserRequest) (*UserResponse, error)
	UpdateUserProfile(ctx context.Context, targetWallet string, request UpdateUserProfileRequest) (*UserResponse, error)
	ListUserActivity(ctx context.Context, identifier string, options ListUserActivityOptions) (*ActivityResponse, error)
	DownloadLiquidationJustification(ctx context.Context, liquidationID string) ([]byte, error)

	// Portfolio
	GetPortfolioSummary(ctx context.Context, wallet string) (*PortfolioSummary, error)

	// Markets
	ListMarkets(ctx context.Context, options ListMarketsOptions) (json.RawMessage, error)
	ListGroups(ctx context.Context, options ListGroupsOptions) (json.RawMessage, error)
	ListCategories(ctx context.Context) (json.RawMessage, error)
	ListTrendingTags(ctx context.Context) (json.RawMessage, error)
	SearchMarkets(ctx context.Context, options SearchMarketsOptions) (json.RawMessage, error)
	LookupSlug(ctx context.Context, slug string) (json.RawMessage, error)
	GetMarket(ctx context.Context, slug string) (json.RawMessage, error)
	ListSeriesSiblings(ctx context.Context, seriesSlug string) (json.RawMessage, error)
	GetGroup(ctx context.Context, slug string) (json.RawMessage, error)
	GetPriceHistory(ctx context.Context, marketID string, options GetPriceHistoryOptions) (json.RawMessage, error)
	GetAggregatedOrderbook(ctx context.Context, marketID, side string, options GetAggregatedOrderbookOptions) (json.RawMessage, error)
	GetMarketPosition(ctx context.Context, marketID, wallet string) (json.RawMessage, error)

	// Orders
	GetCLOBAuthData(ctx context.Context, wallet string) (json.RawMessage, error)
	GetSafeDeployData(ctx context.Context, wallet string) (json.RawMessage, error)
	DeploySafe(ctx context.Context, request SafeDeployRequest) (json.RawMessage, error)
	ConfirmSafeDeployment(ctx context.Context, request ConfirmDeployRequest) (json.RawMessage, error)
	GetRelayStatus(ctx context.Context, options GetRelayStatusOptions) (json.RawMessage, error)
	GetTradingSetupStatus(ctx context.Context, wallet string) (json.RawMessage, error)
	GetApproveTokensData(ctx context.Context, wallet string) (json.RawMessage, error)
	ApproveTokens(ctx context.Context, request ApproveTokensRequest) (json.RawMessage, error)
	GetLendingApproveData(ctx context.Context, wallet string) (json.RawMessage, error)
	GetLendingApproveCalldata(ctx context.Context, request LendingApproveRequest) (json.RawMessage, error)
	GetWithdrawRelayerData(ctx context.Context, wallet, amount string) (json.RawMessage, error)
	WithdrawViaRelayer(ctx context.Context, request WithdrawRequest) (json.RawMessage, error)
	GetRepayRelayerData(ctx context.Context, wallet, tokenID, amount string) (json.RawMessage, error)
	RepayViaRelayer(ctx context.Context, request RepayRelayerRequest) (json.RawMessage, error)
	GetDepositCollateralRelayerData(ctx context.Context, wallet, tokenID, amount string) (json.RawMessage, error)
	DepositCollateralViaRelayer(ctx context.Context, request DepositCollateralRelayerRequest) (json.RawMessage, error)
	GetCLOBInfo(ctx context.Context, marketID string) (json.RawMessage, error)

	// Leagues
	GetLeagueCatalog(ctx context.Context) (json.RawMessage, error)
	RefreshLeagueCatalog(ctx context.Context) (json.RawMessage, error)
}

type Config struct {
	BaseURL     string
	BearerToken string
	Timeout     time.Duration
}

func (c Config) WithDefaults() Config {
	return c.withDefaults()
}

func (c Config) withDefaults() Config {
	c.BaseURL = strings.TrimSpace(c.BaseURL)
	if c.BaseURL == "" {
		c.BaseURL = DefaultBaseURL
	}
	c.BaseURL = strings.TrimRight(c.BaseURL, "/")
	c.BearerToken = strings.TrimSpace(c.BearerToken)
	if c.Timeout <= 0 {
		c.Timeout = DefaultTimeout
	}
	return c
}

type clientImpl struct {
	config  Config
	baseURL *url.URL
	http    *http.Client
}

var _ Client = (*clientImpl)(nil)

func NewClient(config Config) (Client, error) {
	config = config.withDefaults()
	baseURL, err := url.ParseRequestURI(config.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid pred base url: %w", err)
	}
	if baseURL.Scheme == "" || baseURL.Host == "" {
		return nil, errors.New("invalid pred base url: absolute URL is required")
	}
	return &clientImpl{
		config:  config,
		baseURL: baseURL,
		http:    &http.Client{Timeout: config.Timeout},
	}, nil
}

type APIError struct {
	StatusCode int    `json:"-"`
	Type       string `json:"type,omitempty"`
	Message    string `json:"message,omitempty"`
	RawBody    string `json:"-"`
}

func (e *APIError) Error() string {
	if e == nil {
		return ""
	}
	if e.Type != "" && e.Message != "" {
		return fmt.Sprintf("pred request failed (%d): %s: %s", e.StatusCode, e.Type, e.Message)
	}
	if e.Message != "" {
		return fmt.Sprintf("pred request failed (%d): %s", e.StatusCode, e.Message)
	}
	if e.RawBody != "" {
		return fmt.Sprintf("pred request failed (%d): %s", e.StatusCode, e.RawBody)
	}
	return fmt.Sprintf("pred request failed with status %d", e.StatusCode)
}

type encodedBody struct {
	contentType string
	data        []byte
}

func jsonRequestBody(value any) (*encodedBody, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("marshal pred request body: %w", err)
	}
	return &encodedBody{contentType: "application/json", data: data}, nil
}

func formRequestBody(values url.Values) *encodedBody {
	return &encodedBody{
		contentType: "application/x-www-form-urlencoded",
		data:        []byte(values.Encode()),
	}
}

func (c *clientImpl) doJSON(
	ctx context.Context,
	method string,
	endpoint string,
	query url.Values,
	body *encodedBody,
	authRequired bool,
	out any,
) error {
	data, err := c.do(ctx, method, endpoint, query, body, authRequired)
	if err != nil {
		return err
	}
	if out == nil || len(bytes.TrimSpace(data)) == 0 {
		return nil
	}
	if err := json.Unmarshal(data, out); err != nil {
		return fmt.Errorf("decode pred response: %w", err)
	}
	return nil
}

func doTyped[T any](
	ctx context.Context,
	client *clientImpl,
	method string,
	endpoint string,
	query url.Values,
	body *encodedBody,
	authRequired bool,
) (*T, error) {
	var out T
	if err := client.doJSON(ctx, method, endpoint, query, body, authRequired, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func doNullable[T any](
	ctx context.Context,
	client *clientImpl,
	method string,
	endpoint string,
	query url.Values,
	body *encodedBody,
	authRequired bool,
) (*T, error) {
	var out *T
	if err := client.doJSON(ctx, method, endpoint, query, body, authRequired, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func doRaw(
	ctx context.Context,
	client *clientImpl,
	method string,
	endpoint string,
	query url.Values,
	body *encodedBody,
	authRequired bool,
) (json.RawMessage, error) {
	var out json.RawMessage
	if err := client.doJSON(ctx, method, endpoint, query, body, authRequired, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *clientImpl) doBytes(
	ctx context.Context,
	method string,
	endpoint string,
	query url.Values,
	body *encodedBody,
	authRequired bool,
) ([]byte, error) {
	return c.doWithAccept(ctx, method, endpoint, query, body, authRequired, "*/*")
}

func (c *clientImpl) do(
	ctx context.Context,
	method string,
	endpoint string,
	query url.Values,
	body *encodedBody,
	authRequired bool,
) ([]byte, error) {
	return c.doWithAccept(ctx, method, endpoint, query, body, authRequired, "application/json")
}

func (c *clientImpl) doWithAccept(
	ctx context.Context,
	method string,
	endpoint string,
	query url.Values,
	body *encodedBody,
	authRequired bool,
	accept string,
) ([]byte, error) {
	if authRequired && c.config.BearerToken == "" {
		return nil, ErrBearerTokenRequired
	}

	requestURL := *c.baseURL
	rawPath := strings.TrimRight(requestURL.EscapedPath(), "/") + "/" + strings.TrimLeft(endpoint, "/")
	decodedPath, err := url.PathUnescape(rawPath)
	if err != nil {
		return nil, fmt.Errorf("build pred request path: %w", err)
	}
	requestURL.Path = decodedPath
	requestURL.RawPath = rawPath
	requestURL.RawQuery = query.Encode()

	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body.data)
	}
	request, err := http.NewRequestWithContext(ctx, method, requestURL.String(), reader)
	if err != nil {
		return nil, fmt.Errorf("create pred request: %w", err)
	}
	request.Header.Set("Accept", accept)
	if body != nil {
		request.Header.Set("Content-Type", body.contentType)
	}
	if authRequired {
		request.Header.Set("Authorization", "Bearer "+c.config.BearerToken)
	}

	response, err := c.http.Do(request)
	if err != nil {
		return nil, fmt.Errorf("execute pred request: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, decodeAPIError(response)
	}
	data, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("read pred response: %w", err)
	}
	return data, nil
}

func decodeAPIError(response *http.Response) error {
	data, err := io.ReadAll(io.LimitReader(response.Body, errorBodyLimit+1))
	if err != nil {
		return fmt.Errorf("read pred error response: %w", err)
	}
	truncated := len(data) > errorBodyLimit
	if truncated {
		data = data[:errorBodyLimit]
	}
	rawBody := strings.TrimSpace(string(data))
	if truncated {
		rawBody += "…"
	}

	apiError := &APIError{
		StatusCode: response.StatusCode,
		RawBody:    rawBody,
	}
	var payload struct {
		Type    string          `json:"type"`
		Message string          `json:"message"`
		Error   string          `json:"error"`
		Detail  json.RawMessage `json:"detail"`
	}
	if json.Unmarshal(data, &payload) == nil {
		apiError.Type = payload.Type
		apiError.Message = firstNonEmpty(payload.Message, payload.Error, detailMessage(payload.Detail))
	}
	return apiError
}

func detailMessage(detail json.RawMessage) string {
	if len(detail) == 0 || string(detail) == "null" {
		return ""
	}
	var message string
	if json.Unmarshal(detail, &message) == nil {
		return message
	}
	return strings.TrimSpace(string(detail))
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}

func escapePath(value string) string {
	return url.PathEscape(value)
}

func setString(query url.Values, key, value string) {
	if value != "" {
		query.Set(key, value)
	}
}

func setOptionalString(query url.Values, key string, value *string) {
	if value != nil {
		query.Set(key, *value)
	}
}

func setOptionalInt(query url.Values, key string, value *int) {
	if value != nil {
		query.Set(key, fmt.Sprintf("%d", *value))
	}
}

func setOptionalFloat(query url.Values, key string, value *float64) {
	if value != nil {
		query.Set(key, fmt.Sprintf("%g", *value))
	}
}

func setOptionalBool(query url.Values, key string, value *bool) {
	if value != nil {
		query.Set(key, fmt.Sprintf("%t", *value))
	}
}
