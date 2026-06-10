package ethws

import (
	"context"
	"fmt"
	"math/big"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
)

const poolDialTimeout = 5 * time.Second

type dialResult struct {
	client   *ethclient.Client
	endpoint string
}

type dialError struct {
	endpoint string
	err      error
}

// NormalizeEndpoints trims endpoint values, removes empty entries, and deduplicates them.
func NormalizeEndpoints(endpoints []string) []string {
	normalized := make([]string, 0, len(endpoints))
	seen := make(map[string]struct{}, len(endpoints))
	for _, endpoint := range endpoints {
		endpoint = strings.TrimSpace(endpoint)
		if endpoint == "" {
			continue
		}
		if _, ok := seen[endpoint]; ok {
			continue
		}
		seen[endpoint] = struct{}{}
		normalized = append(normalized, endpoint)
	}
	return normalized
}

// RedactEndpoint removes credentials and query parameters before an endpoint is logged.
func RedactEndpoint(endpoint string) string {
	parsed, err := url.Parse(strings.TrimSpace(endpoint))
	if err != nil {
		return "<invalid>"
	}
	parsed.User = nil
	parsed.RawQuery = ""
	parsed.ForceQuery = false
	parsed.Fragment = ""
	return parsed.String()
}

// DialFastestContext connects to all endpoints concurrently and returns the first
// client that reports the expected chain ID.
func DialFastestContext(ctx context.Context, endpoints []string, expectedChainID int64, useProxy bool) (*ethclient.Client, string, error) {
	endpoints = NormalizeEndpoints(endpoints)
	if len(endpoints) == 0 {
		return nil, "", fmt.Errorf("node websocket endpoint list is empty")
	}

	probeCtx, cancel := context.WithTimeout(ctx, poolDialTimeout)
	defer cancel()

	successes := make(chan dialResult)
	failures := make(chan dialError, len(endpoints))
	done := make(chan struct{})
	var wg sync.WaitGroup
	for _, endpoint := range endpoints {
		wg.Add(1)
		go func(endpoint string) {
			defer wg.Done()
			client, err := DialContext(probeCtx, endpoint, useProxy)
			if err != nil {
				failures <- dialError{endpoint: endpoint, err: err}
				return
			}
			chainID, err := client.ChainID(probeCtx)
			if err != nil {
				client.Close()
				failures <- dialError{endpoint: endpoint, err: err}
				return
			}
			if chainID == nil || chainID.Cmp(big.NewInt(expectedChainID)) != 0 {
				client.Close()
				failures <- dialError{
					endpoint: endpoint,
					err:      fmt.Errorf("returned chain_id %v, expected %d", chainID, expectedChainID),
				}
				return
			}
			select {
			case successes <- dialResult{client: client, endpoint: endpoint}:
			case <-probeCtx.Done():
				client.Close()
			}
		}(endpoint)
	}
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case result := <-successes:
		cancel()
		return result.client, result.endpoint, nil
	case <-done:
		return nil, "", formatDialErrors(endpoints, failures, context.DeadlineExceeded)
	case <-probeCtx.Done():
		return nil, "", formatDialErrors(endpoints, failures, probeCtx.Err())
	}
}

func formatDialErrors(endpoints []string, failures <-chan dialError, fallback error) error {
	errorByEndpoint := make(map[string]error, len(endpoints))
	for {
		select {
		case failure := <-failures:
			errorByEndpoint[failure.endpoint] = failure.err
		default:
			parts := make([]string, 0, len(endpoints))
			for _, endpoint := range endpoints {
				err := errorByEndpoint[endpoint]
				if err == nil {
					err = fallback
				}
				parts = append(parts, fmt.Sprintf("%s: %v", RedactEndpoint(endpoint), err))
			}
			return fmt.Errorf("all node websocket endpoints failed: %s", strings.Join(parts, "; "))
		}
	}
}
