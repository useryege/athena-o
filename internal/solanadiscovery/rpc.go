package solanadiscovery

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// HTTPError preserves a provider's retry hint through wrapped scanner errors.
type HTTPError struct {
	Method     string
	StatusCode int
	RetryAfter time.Duration
}

func (e *HTTPError) Error() string             { return fmt.Sprintf("Solana %s: HTTP %d", e.Method, e.StatusCode) }
func (e *HTTPError) RetryDelay() time.Duration { return e.RetryAfter }

func retryAfterDelay(header string) time.Duration {
	header = strings.TrimSpace(header)
	if seconds, err := strconv.ParseUint(header, 10, 64); err == nil {
		const maxSeconds = uint64((1<<63 - 1) / int64(time.Second))
		if seconds > maxSeconds {
			return time.Duration(1<<63 - 1)
		}
		return time.Duration(seconds) * time.Second
	}
	if deadline, err := http.ParseTime(header); err == nil {
		if delay := time.Until(deadline); delay > 0 {
			return delay
		}
	}
	return 0
}

// RPCClient uses Solana's standard HTTP JSON-RPC interface.
type RPCClient struct {
	url    string
	client *http.Client
}

func NewRPCClient(url string, client *http.Client) *RPCClient {
	if client == nil {
		client = http.DefaultClient
	}
	return &RPCClient{url: url, client: client}
}

func (c *RPCClient) call(ctx context.Context, method string, params []any) (json.RawMessage, error) {
	requestBody, err := json.Marshal(struct {
		JSONRPC string `json:"jsonrpc"`
		ID      int    `json:"id"`
		Method  string `json:"method"`
		Params  []any  `json:"params"`
	}{JSONRPC: "2.0", ID: 1, Method: method, Params: params})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url, bytes.NewReader(requestBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.client.Do(req)
	if err != nil {
		var transportErr *url.Error
		if errors.As(err, &transportErr) {
			err = transportErr.Err
		}
		return nil, fmt.Errorf("Solana %s: %w", method, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, &HTTPError{Method: method, StatusCode: resp.StatusCode, RetryAfter: retryAfterDelay(resp.Header.Get("Retry-After"))}
	}
	var response struct {
		Result json.RawMessage `json:"result"`
		Error  *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 64<<20)).Decode(&response); err != nil {
		return nil, fmt.Errorf("Solana %s response: %w", method, err)
	}
	if response.Error != nil {
		return nil, fmt.Errorf("Solana %s: RPC %d: %s", method, response.Error.Code, response.Error.Message)
	}
	if len(response.Result) == 0 || bytes.Equal(bytes.TrimSpace(response.Result), []byte("null")) {
		return nil, fmt.Errorf("Solana %s returned no result", method)
	}
	return response.Result, nil
}

func (c *RPCClient) GenesisHash(ctx context.Context) (string, error) {
	result, err := c.call(ctx, "getGenesisHash", []any{})
	if err != nil {
		return "", err
	}
	var hash string
	if err := json.Unmarshal(result, &hash); err != nil {
		return "", err
	}
	if hash == "" {
		return "", errors.New("Solana genesis hash is empty")
	}
	return hash, nil
}

func (c *RPCClient) FinalizedSlot(ctx context.Context) (uint64, error) {
	result, err := c.call(ctx, "getSlot", []any{map[string]any{"commitment": "finalized"}})
	if err != nil {
		return 0, err
	}
	var slot uint64
	if err := json.Unmarshal(result, &slot); err != nil {
		return 0, err
	}
	return slot, nil
}

func (c *RPCClient) Blocks(ctx context.Context, from, to uint64) ([]uint64, error) {
	result, err := c.call(ctx, "getBlocks", []any{from, to, map[string]any{"commitment": "finalized"}})
	if err != nil {
		return nil, err
	}
	var slots []uint64
	if err := json.Unmarshal(result, &slots); err != nil {
		return nil, err
	}
	return slots, nil
}

func (c *RPCClient) Block(ctx context.Context, slot uint64) (json.RawMessage, error) {
	return c.call(ctx, "getBlock", []any{slot, map[string]any{
		"encoding": "jsonParsed", "transactionDetails": "full", "rewards": false,
		"commitment": "finalized", "maxSupportedTransactionVersion": 0,
	}})
}

// AccountSnapshot preserves request order, including missing and invalid accounts.
type AccountSnapshot struct {
	Slot     uint64
	Accounts []*AccountInfo
}

func (c *RPCClient) MultipleAccounts(ctx context.Context, addresses []string) (AccountSnapshot, error) {
	var snapshot AccountSnapshot
	if len(addresses) < 1 || len(addresses) > 100 {
		return snapshot, errors.New("invalid account batch size")
	}
	for _, address := range addresses {
		if !validPublicKey(address) {
			return snapshot, errors.New("invalid requested account address")
		}
	}
	data, err := c.call(ctx, "getMultipleAccounts", []any{addresses, map[string]any{"encoding": "base64", "commitment": "finalized"}})
	if err != nil {
		return snapshot, err
	}
	var response struct {
		Context struct {
			Slot *uint64 `json:"slot"`
		} `json:"context"`
		Value []json.RawMessage `json:"value"`
	}
	if err := json.Unmarshal(data, &response); err != nil {
		return snapshot, fmt.Errorf("decode account batch: %w", err)
	}
	if response.Context.Slot == nil || response.Value == nil || len(response.Value) != len(addresses) {
		return snapshot, errors.New("invalid account batch context or count")
	}
	snapshot.Slot = *response.Context.Slot
	snapshot.Accounts = make([]*AccountInfo, len(addresses))
	for i, raw := range response.Value {
		if isJSONNull(raw) {
			continue
		}
		a := &AccountInfo{}
		snapshot.Accounts[i] = a
		var encoded struct {
			Owner      string   `json:"owner"`
			Executable *bool    `json:"executable"`
			Data       []string `json:"data"`
		}
		if err := json.Unmarshal(raw, &encoded); err != nil {
			a.DecodeError = errors.New("invalid account response shape")
			continue
		}
		if encoded.Owner == "" || encoded.Executable == nil || len(encoded.Data) != 2 || encoded.Data[1] != "base64" {
			a.DecodeError = errors.New("invalid base64 account response")
			continue
		}
		a.Owner = encoded.Owner
		a.Executable = *encoded.Executable
		a.Data, err = base64.StdEncoding.DecodeString(encoded.Data[0])
		if err != nil {
			a.DecodeError = errors.New("invalid account base64")
		}
	}
	return snapshot, nil
}
func (c *RPCClient) Transaction(ctx context.Context, signature string) (json.RawMessage, error) {
	if signature == "" {
		return nil, errors.New("missing transaction signature")
	}
	return c.call(ctx, "getTransaction", []any{signature, map[string]any{"encoding": "jsonParsed", "commitment": "finalized", "maxSupportedTransactionVersion": 0}})
}
