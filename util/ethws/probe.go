package ethws

import (
	"context"
	"fmt"
	"time"
)

type ProbeResult struct {
	Endpoint          string
	Available         bool
	Latency           time.Duration
	ReportedChainID   int64
	LatestBlockNumber uint64
	CheckedAt         time.Time
	Err               error
}

// ProbeEndpoints checks all endpoints concurrently and preserves their configured order.
func ProbeEndpoints(ctx context.Context, endpoints []string, expectedChainID int64, useProxy bool) ([]ProbeResult, error) {
	endpoints = NormalizeEndpoints(endpoints)
	if len(endpoints) == 0 {
		return []ProbeResult{}, nil
	}

	type indexedResult struct {
		index  int
		result ProbeResult
	}

	resultCh := make(chan indexedResult, len(endpoints))
	for index, endpoint := range endpoints {
		go func(index int, endpoint string) {
			resultCh <- indexedResult{
				index:  index,
				result: probeEndpoint(ctx, endpoint, expectedChainID, useProxy),
			}
		}(index, endpoint)
	}

	results := make([]ProbeResult, len(endpoints))
	for range endpoints {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case indexed := <-resultCh:
			results[indexed.index] = indexed.result
		}
	}
	return results, nil
}

func probeEndpoint(ctx context.Context, endpoint string, expectedChainID int64, useProxy bool) ProbeResult {
	startedAt := time.Now()
	probeCtx, cancel := context.WithTimeout(ctx, poolDialTimeout)
	defer cancel()

	result := ProbeResult{Endpoint: RedactEndpoint(endpoint)}
	finish := func(err error) ProbeResult {
		result.Latency = time.Since(startedAt)
		result.CheckedAt = time.Now().UTC()
		result.Err = err
		return result
	}

	client, err := DialContext(probeCtx, endpoint, useProxy)
	if err != nil {
		return finish(fmt.Errorf("dial websocket: %w", err))
	}
	defer client.Close()

	chainID, err := client.ChainID(probeCtx)
	if err != nil {
		return finish(fmt.Errorf("get chain id: %w", err))
	}
	if chainID != nil {
		result.ReportedChainID = chainID.Int64()
	}
	if chainID == nil || chainID.Int64() != expectedChainID {
		return finish(fmt.Errorf("unexpected chain id: got %v, expected %d", chainID, expectedChainID))
	}

	latestBlockNumber, err := client.BlockNumber(probeCtx)
	if err != nil {
		return finish(fmt.Errorf("get latest block: %w", err))
	}
	result.LatestBlockNumber = latestBlockNumber
	result.Available = true
	return finish(nil)
}
