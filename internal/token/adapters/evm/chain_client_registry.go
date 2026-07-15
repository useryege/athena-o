package evm

import (
	"context"
	"fmt"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/useryege/athena/internal/token/chainregistry"
	"github.com/useryege/athena/internal/token/discovery"
	"github.com/useryege/athena/util/ethws"
)

type ChainClientRegistry struct {
	registry *chainregistry.Registry
	mu       sync.Mutex
	clients  map[int64]*ethclient.Client
}

func NewChainClientRegistry(registry *chainregistry.Registry) *ChainClientRegistry {
	return &ChainClientRegistry{registry: registry, clients: make(map[int64]*ethclient.Client)}
}

func (r *ChainClientRegistry) Client(ctx context.Context, chainID int64) (*ethclient.Client, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if client := r.clients[chainID]; client != nil {
		return client, nil
	}
	chain, ok := r.registry.Chain(chainID)
	if !ok || !chain.Enabled {
		return nil, fmt.Errorf("token chain %d is not enabled", chainID)
	}
	client, _, err := ethws.DialFastestContext(ctx, chain.NodeWSURLs, chain.ID, chain.UseProxy)
	if err != nil {
		return nil, err
	}
	r.clients[chainID] = client
	return client, nil
}

func (r *ChainClientRegistry) ContractCodeHash(ctx context.Context, chainID int64, contract common.Address) (common.Hash, error) {
	client, err := r.Client(ctx, chainID)
	if err != nil {
		return common.Hash{}, err
	}
	code, err := client.CodeAt(ctx, contract, nil)
	if err != nil {
		return common.Hash{}, err
	}
	if len(code) == 0 {
		return common.Hash{}, fmt.Errorf("contract %s has no deployed bytecode", contract.Hex())
	}
	return crypto.Keccak256Hash(code), nil
}

func (r *ChainClientRegistry) Reset(chainID int64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if client := r.clients[chainID]; client != nil {
		client.Close()
		delete(r.clients, chainID)
	}
}

func (r *ChainClientRegistry) ListNodeStatuses(ctx context.Context) ([]discovery.NodeStatus, error) {
	chains := r.registry.Chains()
	result := make([]discovery.NodeStatus, 0)
	for _, chain := range chains {
		if !chain.Enabled {
			continue
		}
		probes, err := ethws.ProbeEndpoints(ctx, chain.NodeWSURLs, chain.ID, chain.UseProxy)
		if err != nil {
			return nil, err
		}
		for _, probe := range probes {
			status := discovery.NodeStatus{ChainID: chain.ID, ChainName: chain.Name, Endpoint: probe.Endpoint, Available: probe.Available, Latency: probe.Latency, ReportedChainID: probe.ReportedChainID, LatestBlockNumber: probe.LatestBlockNumber, CheckedAt: probe.CheckedAt, ReferenceBlockNumber: probe.ReferenceBlockNumber, BlockLag: probe.BlockLag, LatestBlockTime: probe.LatestBlockTime, Syncing: probe.Syncing}
			if probe.Err != nil {
				status.Error = probe.Err.Error()
			}
			result = append(result, status)
		}
	}
	return result, nil
}

func (r *ChainClientRegistry) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for chainID, client := range r.clients {
		client.Close()
		delete(r.clients, chainID)
	}
	return nil
}
