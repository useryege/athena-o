package ethereumapi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
)

type EthereumAPI interface {
	GetSourceCode(ctx context.Context, chainID int64, contractAddress string) (*SourceCodeResponse, error)
	GetABI(ctx context.Context, chainID int64, contractAddress string) (*ABIResponse, error)
}

type ethereumAPIImpl struct {
	baseURL string
	apiKey  string
	client  *http.Client
}

func NewEthereumAPI(baseURL string, apiKey string) EthereumAPI {
	return &ethereumAPIImpl{
		baseURL: baseURL,
		apiKey:  apiKey,
		client:  &http.Client{},
	}
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

	client := e.client
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
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
	return fmt.Errorf("etherscan http request failed with status %s: %s", resp.Status, string(body))
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
