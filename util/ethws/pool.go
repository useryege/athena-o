package ethws

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
)

const poolDialTimeout = 5 * time.Second

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

// DialFastestContext probes all endpoints and returns the lowest-latency healthy client.
func DialFastestContext(ctx context.Context, endpoints []string, expectedChainID int64, useProxy bool) (*ethclient.Client, string, error) {
	endpoints = NormalizeEndpoints(endpoints)
	if len(endpoints) == 0 {
		return nil, "", fmt.Errorf("node websocket endpoint list is empty")
	}

	probes, err := probeEndpoints(ctx, endpoints, expectedChainID, useProxy)
	if err != nil {
		return nil, "", err
	}

	selectedIndex := -1
	for index := range probes {
		if !probes[index].result.Available {
			continue
		}
		if selectedIndex == -1 || probes[index].result.Latency < probes[selectedIndex].result.Latency {
			selectedIndex = index
		}
	}
	if selectedIndex == -1 {
		err := formatProbeErrors(probes)
		closeProbeClients(probes)
		return nil, "", err
	}

	selectedClient := probes[selectedIndex].client
	selectedEndpoint := endpoints[selectedIndex]
	for index := range probes {
		if index != selectedIndex && probes[index].client != nil {
			probes[index].client.Close()
		}
	}
	return selectedClient, selectedEndpoint, nil
}

func formatProbeErrors(probes []endpointProbe) error {
	parts := make([]string, 0, len(probes))
	for _, probe := range probes {
		err := probe.result.Err
		if err == nil {
			err = fmt.Errorf("node is unavailable")
		}
		parts = append(parts, fmt.Sprintf("%s: %v", probe.result.Endpoint, err))
	}
	return fmt.Errorf("all node websocket endpoints failed: %s", strings.Join(parts, "; "))
}
