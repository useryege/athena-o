package chainregistry

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

const EnvironmentVariable = "ATHENA_TOKEN_CHAINS_JSON"

type duration struct {
	time.Duration
}

func (d *duration) UnmarshalJSON(data []byte) error {
	var text string
	if err := json.Unmarshal(data, &text); err == nil {
		parsed, err := time.ParseDuration(strings.TrimSpace(text))
		if err != nil {
			return err
		}
		d.Duration = parsed
		return nil
	}
	var nanos int64
	if err := json.Unmarshal(data, &nanos); err != nil {
		return fmt.Errorf("duration must be a duration string or nanoseconds: %w", err)
	}
	d.Duration = time.Duration(nanos)
	return nil
}

type chainJSON struct {
	ID                             int64    `json:"id"`
	Name                           string   `json:"name"`
	Enabled                        bool     `json:"enabled"`
	NodeWSURLs                     []string `json:"nodeWsUrls"`
	UseProxy                       bool     `json:"useProxy"`
	AthenaContract                 string   `json:"athenaContract"`
	ScannerInitialLookbackDuration duration `json:"scannerInitialLookbackDuration"`
	ScannerPollInterval            duration `json:"scannerPollInterval"`
}

type Chain struct {
	ID                             int64
	Name                           string
	Enabled                        bool
	NodeWSURLs                     []string
	UseProxy                       bool
	AthenaContract                 string
	ScannerInitialLookbackDuration time.Duration
	ScannerPollInterval            time.Duration
}

type Registry struct {
	chains []Chain
	byID   map[int64]Chain
}

func Parse(raw string) (*Registry, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("%s is required", EnvironmentVariable)
	}
	var values []chainJSON
	if err := json.Unmarshal([]byte(raw), &values); err != nil {
		return nil, fmt.Errorf("parse %s: %w", EnvironmentVariable, err)
	}
	registry := &Registry{chains: make([]Chain, 0, len(values)), byID: make(map[int64]Chain, len(values))}
	for index, value := range values {
		value.Name = strings.TrimSpace(value.Name)
		if value.ID <= 0 {
			return nil, fmt.Errorf("%s[%d].id must be positive", EnvironmentVariable, index)
		}
		if value.Name == "" {
			return nil, fmt.Errorf("%s[%d].name is required", EnvironmentVariable, index)
		}
		if _, exists := registry.byID[value.ID]; exists {
			return nil, fmt.Errorf("%s contains duplicate chain id %d", EnvironmentVariable, value.ID)
		}
		urls := make([]string, 0, len(value.NodeWSURLs))
		for _, endpoint := range value.NodeWSURLs {
			if endpoint = strings.TrimSpace(endpoint); endpoint != "" {
				urls = append(urls, endpoint)
			}
		}
		if value.ScannerPollInterval.Duration <= 0 {
			return nil, fmt.Errorf("%s[%d].scannerPollInterval must be positive", EnvironmentVariable, index)
		}
		if value.ScannerInitialLookbackDuration.Duration < time.Second {
			return nil, fmt.Errorf("%s[%d].scannerInitialLookbackDuration must be at least 1s", EnvironmentVariable, index)
		}
		chain := Chain{ID: value.ID, Name: value.Name, Enabled: value.Enabled, NodeWSURLs: urls, UseProxy: value.UseProxy, AthenaContract: strings.TrimSpace(value.AthenaContract), ScannerInitialLookbackDuration: value.ScannerInitialLookbackDuration.Duration, ScannerPollInterval: value.ScannerPollInterval.Duration}
		registry.chains = append(registry.chains, chain)
		registry.byID[chain.ID] = chain
	}
	sort.Slice(registry.chains, func(i, j int) bool { return registry.chains[i].ID < registry.chains[j].ID })
	return registry, nil
}

func (r *Registry) Chains() []Chain {
	if r == nil {
		return nil
	}
	result := make([]Chain, len(r.chains))
	copy(result, r.chains)
	return result
}

func (r *Registry) EnabledChains() []Chain {
	result := make([]Chain, 0)
	for _, chain := range r.Chains() {
		if chain.Enabled {
			result = append(result, chain)
		}
	}
	return result
}

func (r *Registry) Chain(id int64) (Chain, bool) {
	if r == nil {
		return Chain{}, false
	}
	chain, ok := r.byID[id]
	return chain, ok
}
