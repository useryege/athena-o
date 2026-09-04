package chainregistry

import (
	"fmt"
	"strings"
	"time"
)

const (
	ethereumChainID   int64 = 1
	ethereumChainName       = "Ethereum Mainnet"
	bscChainID        int64 = 56
	bscChainName            = "BSC Mainnet"
)

type ChainConfig struct {
	Enabled                          bool
	NodeWSURLs                       []string
	AthenaContract                   string
	ProcessorInitialLookbackDuration time.Duration
	ProcessorPollInterval            time.Duration
}

type Config struct {
	Ethereum ChainConfig
	BSC      ChainConfig
}

type Chain struct {
	ID                               int64
	Name                             string
	Enabled                          bool
	NodeWSURLs                       []string
	AthenaContract                   string
	ProcessorInitialLookbackDuration time.Duration
	ProcessorPollInterval            time.Duration
}

type Registry struct {
	chains []Chain
	byID   map[int64]Chain
}

func New(config Config) (*Registry, error) {
	chains := []Chain{
		newChain(ethereumChainID, ethereumChainName, config.Ethereum),
		newChain(bscChainID, bscChainName, config.BSC),
	}
	registry := &Registry{chains: make([]Chain, 0, len(chains)), byID: make(map[int64]Chain, len(chains))}
	for _, chain := range chains {
		if len(chain.NodeWSURLs) == 0 {
			return nil, fmt.Errorf("%s node WebSocket URLs are required", chain.Name)
		}
		if chain.AthenaContract == "" {
			return nil, fmt.Errorf("%s ATHENA contract is required", chain.Name)
		}
		if chain.ProcessorInitialLookbackDuration < time.Second {
			return nil, fmt.Errorf("%s processor initial lookback duration must be at least 1s", chain.Name)
		}
		if chain.ProcessorPollInterval <= 0 {
			return nil, fmt.Errorf("%s processor poll interval must be positive", chain.Name)
		}
		registry.chains = append(registry.chains, chain)
		registry.byID[chain.ID] = chain
	}
	return registry, nil
}

func newChain(id int64, name string, config ChainConfig) Chain {
	urls := make([]string, 0, len(config.NodeWSURLs))
	for _, endpoint := range config.NodeWSURLs {
		if endpoint = strings.TrimSpace(endpoint); endpoint != "" {
			urls = append(urls, endpoint)
		}
	}
	return Chain{
		ID:                               id,
		Name:                             name,
		Enabled:                          config.Enabled,
		NodeWSURLs:                       urls,
		AthenaContract:                   strings.TrimSpace(config.AthenaContract),
		ProcessorInitialLookbackDuration: config.ProcessorInitialLookbackDuration,
		ProcessorPollInterval:            config.ProcessorPollInterval,
	}
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
