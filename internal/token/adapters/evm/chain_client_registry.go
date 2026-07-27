package evm

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	log "github.com/sirupsen/logrus"
	"github.com/useryege/athena/internal/token/chainregistry"
	"github.com/useryege/athena/internal/token/discovery"
	"github.com/useryege/athena/internal/token/shared"
	"github.com/useryege/athena/util/ethws"
)

type ChainClientRegistry struct {
	registry *chainregistry.Registry
	proxyURL string
	mu       sync.Mutex
	clients  map[int64]*ethclient.Client
}

func NewChainClientRegistry(registry *chainregistry.Registry, proxyURL string) *ChainClientRegistry {
	return &ChainClientRegistry{registry: registry, proxyURL: strings.TrimSpace(proxyURL), clients: make(map[int64]*ethclient.Client)}
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
	startedAt := time.Now()
	client, endpoint, err := ethws.DialFastestContext(ctx, chain.NodeWSURLs, chain.ID, r.proxyURL)
	if err != nil {
		return nil, err
	}
	r.clients[chainID] = client
	log.WithFields(log.Fields{
		"chain_id":       chainID,
		"endpoint":       ethws.RedactEndpoint(endpoint),
		"endpoint_count": len(ethws.NormalizeEndpoints(chain.NodeWSURLs)),
		"proxy_enabled":  r.proxyURL != "",
		"proxy_endpoint": r.proxyURL,
		"duration_ms":    time.Since(startedAt).Milliseconds(),
	}).Info("token EVM client selected")
	return client, nil
}

func (r *ChainClientRegistry) ContractCodeHash(ctx context.Context, chainID int64, contract shared.Address) (shared.Hash, error) {
	client, err := r.Client(ctx, chainID)
	if err != nil {
		return shared.Hash{}, err
	}
	commonContract := common.Address(contract)
	code, err := client.CodeAt(ctx, commonContract, nil)
	if err != nil {
		return shared.Hash{}, err
	}
	if len(code) == 0 {
		return shared.Hash{}, fmt.Errorf("contract %s has no deployed bytecode", contract.Hex())
	}
	return shared.Hash(crypto.Keccak256Hash(code)), nil
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
		probes, err := ethws.ProbeEndpoints(ctx, chain.NodeWSURLs, chain.ID, r.proxyURL)
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
