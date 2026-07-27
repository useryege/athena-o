package ethws

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
)

const (
	ethereumMainnetChainID = 1
	bscMainnetChainID      = 56
)

type ProbeResult struct {
	Endpoint             string
	Available            bool
	Latency              time.Duration
	ReportedChainID      int64
	LatestBlockNumber    uint64
	ReferenceBlockNumber uint64
	BlockLag             uint64
	LatestBlockTime      time.Time
	Syncing              bool
	CheckedAt            time.Time
	Err                  error
}

type endpointProbe struct {
	result ProbeResult
	client *ethclient.Client
}

type healthPolicy struct {
	maxBlockLag uint64
	maxBlockAge time.Duration
}

// ProbeEndpoints checks all endpoints concurrently and preserves their configured order.
func ProbeEndpoints(ctx context.Context, endpoints []string, expectedChainID int64, proxyURL string) ([]ProbeResult, error) {
	probes, err := probeEndpoints(ctx, endpoints, expectedChainID, proxyURL)
	if err != nil {
		return nil, err
	}

	results := make([]ProbeResult, len(probes))
	for index, probe := range probes {
		if probe.client != nil {
			probe.client.Close()
		}
		results[index] = probe.result
	}
	return results, nil
}

func probeEndpoints(ctx context.Context, endpoints []string, expectedChainID int64, proxyURL string) ([]endpointProbe, error) {
	endpoints = NormalizeEndpoints(endpoints)
	if len(endpoints) == 0 {
		return []endpointProbe{}, nil
	}

	probeCtx, cancel := context.WithTimeout(ctx, endpointProbeTimeout)
	defer cancel()

	type indexedProbe struct {
		index int
		probe endpointProbe
	}

	probeCh := make(chan indexedProbe, len(endpoints))
	for index, endpoint := range endpoints {
		go func(index int, endpoint string) {
			probeCh <- indexedProbe{
				index: index,
				probe: collectEndpointProbe(probeCtx, endpoint, expectedChainID, proxyURL),
			}
		}(index, endpoint)
	}

	probes := make([]endpointProbe, len(endpoints))
	for range endpoints {
		indexed := <-probeCh
		probes[indexed.index] = indexed.probe
	}
	if err := ctx.Err(); err != nil {
		closeProbeClients(probes)
		return nil, err
	}

	evaluateEndpointProbes(probes, expectedChainID, time.Now())
	return probes, nil
}

func collectEndpointProbe(ctx context.Context, endpoint string, expectedChainID int64, proxyURL string) endpointProbe {
	startedAt := time.Now()
	probe := endpointProbe{
		result: ProbeResult{Endpoint: RedactEndpoint(endpoint)},
	}
	finish := func(err error) endpointProbe {
		probe.result.Latency = time.Since(startedAt)
		probe.result.CheckedAt = time.Now().UTC()
		probe.result.Err = err
		return probe
	}

	client, err := DialContext(ctx, endpoint, proxyURL)
	if err != nil {
		return finish(fmt.Errorf("dial websocket: %w", err))
	}
	probe.client = client

	chainID, err := client.ChainID(ctx)
	if err != nil {
		return finish(fmt.Errorf("get chain id: %w", err))
	}
	if chainID != nil {
		probe.result.ReportedChainID = chainID.Int64()
	}
	if chainID == nil || chainID.Cmp(big.NewInt(expectedChainID)) != 0 {
		return finish(fmt.Errorf("unexpected chain id: got %v, expected %d", chainID, expectedChainID))
	}

	syncProgress, err := client.SyncProgress(ctx)
	if err != nil {
		return finish(fmt.Errorf("get sync progress: %w", err))
	}
	probe.result.Syncing = syncProgress != nil

	latestBlockNumber, err := client.BlockNumber(ctx)
	if err != nil {
		return finish(fmt.Errorf("get latest block number: %w", err))
	}
	probe.result.LatestBlockNumber = latestBlockNumber

	header, err := client.HeaderByNumber(ctx, new(big.Int).SetUint64(latestBlockNumber))
	if err != nil {
		return finish(fmt.Errorf("get latest block header: %w", err))
	}
	if header == nil {
		return finish(fmt.Errorf("get latest block header: empty response"))
	}
	probe.result.LatestBlockTime = time.Unix(int64(header.Time), 0).UTC()
	return finish(nil)
}

func evaluateEndpointProbes(probes []endpointProbe, chainID int64, now time.Time) {
	var referenceBlockNumber uint64
	for _, probe := range probes {
		if probe.result.Err == nil && probe.result.LatestBlockNumber > referenceBlockNumber {
			referenceBlockNumber = probe.result.LatestBlockNumber
		}
	}

	policy := healthPolicyForChain(chainID)
	for index := range probes {
		result := &probes[index].result
		result.ReferenceBlockNumber = referenceBlockNumber
		if result.Err != nil {
			continue
		}
		if referenceBlockNumber >= result.LatestBlockNumber {
			result.BlockLag = referenceBlockNumber - result.LatestBlockNumber
		}
		switch {
		case result.Syncing:
			result.Err = fmt.Errorf("node is still syncing")
		case now.Sub(result.LatestBlockTime) > policy.maxBlockAge:
			result.Err = fmt.Errorf(
				"latest block is stale: block_time=%s age=%s max_age=%s",
				result.LatestBlockTime.Format(time.RFC3339),
				now.Sub(result.LatestBlockTime).Round(time.Second),
				policy.maxBlockAge,
			)
		case result.BlockLag > policy.maxBlockLag:
			result.Err = fmt.Errorf(
				"latest block is behind reference: block=%d reference=%d lag=%d max_lag=%d",
				result.LatestBlockNumber,
				referenceBlockNumber,
				result.BlockLag,
				policy.maxBlockLag,
			)
		default:
			result.Available = true
		}
	}
}

func healthPolicyForChain(chainID int64) healthPolicy {
	switch chainID {
	case bscMainnetChainID:
		return healthPolicy{maxBlockLag: 3, maxBlockAge: 30 * time.Second}
	case ethereumMainnetChainID:
		fallthrough
	default:
		return healthPolicy{maxBlockLag: 2, maxBlockAge: 60 * time.Second}
	}
}

func closeProbeClients(probes []endpointProbe) {
	for _, probe := range probes {
		if probe.client != nil {
			probe.client.Close()
		}
	}
}
