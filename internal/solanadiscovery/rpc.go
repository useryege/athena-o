package solanadiscovery

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
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
