package ethereumapi

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/useryege/athena/util/ratelimit"
)

const (
	DefaultBaseURL = "https://api.etherscan.io/v2/api"
	DefaultTimeout = 30 * time.Second

	maxAPIErrorBodyLength = 4096
)

const (
	APIErrorTypeAuthentication = "authentication"
	APIErrorTypeInvalidRequest = "invalid_request"
	APIErrorTypeMalformed      = "malformed_response"
	APIErrorTypePlan           = "plan"
	APIErrorTypeRateLimit      = "rate_limit"
	APIErrorTypeUpstream       = "upstream"
)

type EthereumAPI interface {
	GetSourceCode(ctx context.Context, chainID int64, contractAddress string) (*SourceCodeResponse, error)
	GetABI(ctx context.Context, chainID int64, contractAddress string) (*ABIResponse, error)
	ListNormalTransactions(ctx context.Context, opts ListNormalTransactionsOptions) (*NormalTransactionsResponse, error)
}

type ethereumAPIImpl struct {
	baseURL     string
	apiKey      string
	client      *http.Client
	rateLimiter ratelimit.Limiter
}

type Config struct {
	BaseURL     string
	APIKey      string
	RateLimiter ratelimit.Limiter
	Timeout     time.Duration
	HTTPClient  *http.Client
}

func NewEthereumAPI(baseURL string, apiKey string) EthereumAPI {
	return NewEthereumAPIWithConfig(Config{
		BaseURL: baseURL,
		APIKey:  apiKey,
	})
}

func NewEthereumAPIWithConfig(config Config) EthereumAPI {
	baseURL := strings.TrimSpace(config.BaseURL)
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	rateLimiter := config.RateLimiter
	if rateLimiter == nil {
		rateLimiter = ratelimit.Noop()
	}
	client := config.HTTPClient
	if client == nil {
		timeout := config.Timeout
		if timeout <= 0 {
			timeout = DefaultTimeout
		}
		client = &http.Client{Timeout: timeout}
	}
	return &ethereumAPIImpl{
		baseURL:     baseURL,
		apiKey:      config.APIKey,
		client:      client,
		rateLimiter: rateLimiter,
	}
}

type APIError struct {
	StatusCode int
	Type       string
	Message    string
	RawBody    string
}

func (e *APIError) Error() string {
	if e == nil {
		return ""
	}
	message := strings.TrimSpace(e.Message)
	if message == "" {
		message = "etherscan API request failed"
	}
	if e.StatusCode > 0 {
		return fmt.Sprintf("%s (status %d)", message, e.StatusCode)
	}
	return message
}

type NormalTransactionSort string

const (
	NormalTransactionSortASC  NormalTransactionSort = "asc"
	NormalTransactionSortDESC NormalTransactionSort = "desc"
)

type ListNormalTransactionsOptions struct {
	ChainID    int64
	Address    string
	StartBlock uint64
	EndBlock   uint64
	Page       int32
	PageSize   int32
	Sort       NormalTransactionSort
}

type NormalTransactionsResponse struct {
	Status  string                    `json:"status"`
	Message string                    `json:"message"`
	Result  []NormalTransactionResult `json:"result"`
}

type NormalTransactionResult struct {
	BlockNumber       string `json:"blockNumber"`
	BlockHash         string `json:"blockHash"`
	TimeStamp         string `json:"timeStamp"`
	Hash              string `json:"hash"`
	Nonce             string `json:"nonce"`
	TransactionIndex  string `json:"transactionIndex"`
	From              string `json:"from"`
	To                string `json:"to"`
	Value             string `json:"value"`
	Gas               string `json:"gas"`
	GasPrice          string `json:"gasPrice"`
	Input             string `json:"input"`
	MethodID          string `json:"methodId"`
	FunctionName      string `json:"functionName"`
	ContractAddress   string `json:"contractAddress"`
	CumulativeGasUsed string `json:"cumulativeGasUsed"`
	TxReceiptStatus   string `json:"txreceipt_status"`
	GasUsed           string `json:"gasUsed"`
	Confirmations     string `json:"confirmations"`
	IsError           string `json:"isError"`
}

type SourceCodeResponse struct {
	Status  string             `json:"status"`
	Message string             `json:"message"`
	Result  []SourceCodeResult `json:"result"`
}

type SourceCodeResult struct {
	SourceCode           string `json:"SourceCode"`
	ABI                  string `json:"ABI"`
	ContractName         string `json:"ContractName"`
	CompilerVersion      string `json:"CompilerVersion"`
	OptimizationUsed     string `json:"OptimizationUsed"`
	Runs                 string `json:"Runs"`
	ConstructorArguments string `json:"ConstructorArguments"`
	EVMVersion           string `json:"EVMVersion"`
	Library              string `json:"Library"`
	LicenseType          string `json:"LicenseType"`
	Proxy                string `json:"Proxy"`
	Implementation       string `json:"Implementation"`
	SwarmSource          string `json:"SwarmSource"`
}

type ABIResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Result  string `json:"result"`
}

func (e *ethereumAPIImpl) GetSourceCode(ctx context.Context, chainID int64, contractAddress string) (*SourceCodeResponse, error) {
	resp, err := e.do(ctx, e.apiKey, chainID, "contract", "getsourcecode", map[string]interface{}{
		"address": contractAddress,
	})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := ensureHTTPSuccess(resp); err != nil {
		return nil, err
	}

	var rawResp struct {
		Status  string          `json:"status"`
		Message string          `json:"message"`
		Result  json.RawMessage `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&rawResp); err != nil {
		return nil, fmt.Errorf("failed to decode etherscan getsourcecode response: %w", err)
	}
	if rawResp.Status != "1" {
		var result string
		if err := json.Unmarshal(rawResp.Result, &result); err != nil {
			result = summarizeRawJSON(rawResp.Result)
		}
		if result == "" {
			return nil, fmt.Errorf("etherscan getsourcecode failed: %s", rawResp.Message)
		}
		return nil, fmt.Errorf("etherscan getsourcecode failed: %s: %s", rawResp.Message, result)
	}

	sourceCodeResp := SourceCodeResponse{
		Status:  rawResp.Status,
		Message: rawResp.Message,
	}
	if err := json.Unmarshal(rawResp.Result, &sourceCodeResp.Result); err != nil {
		return nil, fmt.Errorf("malformed etherscan getsourcecode response: message=%s result=%s: %w", rawResp.Message, summarizeRawJSON(rawResp.Result), err)
	}
	return &sourceCodeResp, nil
}

func (e *ethereumAPIImpl) GetABI(ctx context.Context, chainID int64, contractAddress string) (*ABIResponse, error) {
	resp, err := e.do(ctx, e.apiKey, chainID, "contract", "getabi", map[string]interface{}{
		"address": contractAddress,
	})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := ensureHTTPSuccess(resp); err != nil {
		return nil, err
	}

	var abiResp ABIResponse
	if err := json.NewDecoder(resp.Body).Decode(&abiResp); err != nil {
		return nil, fmt.Errorf("failed to decode etherscan getabi response: %w", err)
	}
	if abiResp.Status != "1" {
		return nil, fmt.Errorf("etherscan getabi failed: %s: %s", abiResp.Message, abiResp.Result)
	}
	return &abiResp, nil
}

func (e *ethereumAPIImpl) ListNormalTransactions(ctx context.Context, opts ListNormalTransactionsOptions) (*NormalTransactionsResponse, error) {
	resp, err := e.do(ctx, e.apiKey, opts.ChainID, "account", "txlist", map[string]interface{}{
		"address":    opts.Address,
		"startblock": opts.StartBlock,
		"endblock":   opts.EndBlock,
		"page":       opts.Page,
		"offset":     opts.PageSize,
		"sort":       opts.Sort,
	})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := ensureHTTPSuccess(resp); err != nil {
		return nil, err
	}

	var rawResp struct {
		Status  string          `json:"status"`
		Message string          `json:"message"`
		Result  json.RawMessage `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&rawResp); err != nil {
		return nil, &APIError{
			StatusCode: http.StatusOK,
			Type:       APIErrorTypeMalformed,
			Message:    fmt.Sprintf("failed to decode etherscan txlist response: %v", err),
		}
	}

	result := make([]NormalTransactionResult, 0)
	resultErr := json.Unmarshal(rawResp.Result, &result)
	if resultErr == nil {
		if rawResp.Status == "1" || isNoTransactionsResponse(rawResp.Message, result) {
			return &NormalTransactionsResponse{
				Status:  rawResp.Status,
				Message: rawResp.Message,
				Result:  result,
			}, nil
		}
	}
	if rawResp.Status == "1" {
		return nil, &APIError{
			StatusCode: http.StatusOK,
			Type:       APIErrorTypeMalformed,
			Message:    fmt.Sprintf("malformed etherscan txlist result: %v", resultErr),
			RawBody:    limitAPIErrorBody(summarizeRawJSON(rawResp.Result)),
		}
	}

	var resultMessage string
	if err := json.Unmarshal(rawResp.Result, &resultMessage); err != nil {
		resultMessage = summarizeRawJSON(rawResp.Result)
	}
	return nil, newEnvelopeAPIError(rawResp.Message, resultMessage)
}

func (e *ethereumAPIImpl) do(ctx context.Context, apiKey string, chainID int64, module string, action string, params map[string]interface{}) (*http.Response, error) {
	endpoint, err := url.Parse(e.baseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse etherscan base url: %w", err)
	}

	query := endpoint.Query()
	query.Set("apikey", apiKey)
	query.Set("chainid", strconv.FormatInt(chainID, 10))
	query.Set("module", module)
	query.Set("action", action)
	for key, value := range params {
		if value == nil {
			continue
		}
		query.Set(key, fmt.Sprint(value))
	}
	endpoint.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create etherscan request: %w", err)
	}

	if err := e.rateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("wait etherscan rate limit: %w", err)
	}

	client := e.client
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		var urlErr *url.Error
		if stderrors.As(err, &urlErr) {
			err = urlErr.Err
		}
		return nil, fmt.Errorf("failed to send etherscan request: %w", err)
	}
	return resp, nil
}

func ensureHTTPSuccess(resp *http.Response) error {
	if resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusMultipleChoices {
		return nil
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil {
		return fmt.Errorf("etherscan http request failed with status %s and unreadable body: %w", resp.Status, err)
	}
	return &APIError{
		StatusCode: resp.StatusCode,
		Type:       APIErrorTypeUpstream,
		Message:    fmt.Sprintf("etherscan HTTP request failed with status %s", resp.Status),
		RawBody:    limitAPIErrorBody(string(body)),
	}
}

func summarizeRawJSON(raw json.RawMessage) string {
	const maxRawJSONSummaryLength = 256

	if len(raw) == 0 {
		return ""
	}
	summary := string(raw)
	if len(summary) <= maxRawJSONSummaryLength {
		return summary
	}
	return summary[:maxRawJSONSummaryLength] + "..."
}

func isNoTransactionsResponse(message string, result []NormalTransactionResult) bool {
	return len(result) == 0 && strings.Contains(strings.ToLower(message), "no transactions")
}

func newEnvelopeAPIError(message string, result string) error {
	combined := strings.TrimSpace(strings.Join([]string{message, result}, ": "))
	lower := strings.ToLower(combined)
	errorType := APIErrorTypeUpstream
	switch {
	case strings.Contains(lower, "rate limit"):
		errorType = APIErrorTypeRateLimit
	case strings.Contains(lower, "invalid api key") || strings.Contains(lower, "invalid api-key"):
		errorType = APIErrorTypeAuthentication
	case strings.Contains(lower, "free api access") || strings.Contains(lower, "upgrade your api plan"):
		errorType = APIErrorTypePlan
	case strings.Contains(lower, "invalid address") ||
		strings.Contains(lower, "unsupported chain") ||
		strings.Contains(lower, "invalid action") ||
		strings.Contains(lower, "missing"):
		errorType = APIErrorTypeInvalidRequest
	}
	if combined == "" {
		combined = "etherscan API request failed"
	}
	return &APIError{
		StatusCode: http.StatusOK,
		Type:       errorType,
		Message:    combined,
		RawBody:    limitAPIErrorBody(result),
	}
}

func limitAPIErrorBody(body string) string {
	if len(body) <= maxAPIErrorBodyLength {
		return body
	}
	return body[:maxAPIErrorBodyLength] + "..."
}
