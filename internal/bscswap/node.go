package bscswap

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
)

const bscMainnetChainID = int64(56)

var V2SwapTopic = common.HexToHash("0xd78ad95fa46c994b6551d0da85fc275fe613ce37657fb8d5e3d130840159d822")

type Node struct {
	client *ethclient.Client
}

func DialNode(ctx context.Context, rpcURL string) (*Node, error) {
	rpcURL = strings.TrimSpace(rpcURL)
	if rpcURL == "" {
		return nil, fmt.Errorf("BSC node RPC URL is required")
	}
	client, err := ethclient.DialContext(ctx, rpcURL)
	if err != nil {
		return nil, fmt.Errorf("dial BSC node: %w", err)
	}
	node := &Node{client: client}
	chainID, err := node.client.ChainID(ctx)
	if err != nil {
		node.Close()
		return nil, fmt.Errorf("query BSC node chain ID: %w", err)
	}
	if !chainID.IsInt64() || chainID.Int64() != bscMainnetChainID {
		node.Close()
		return nil, fmt.Errorf("BSC node must report chain ID %d, got %s", bscMainnetChainID, chainID.String())
	}
	return node, nil
}

func (n *Node) Close() error {
	if n != nil && n.client != nil {
		n.client.Close()
		n.client = nil
	}
	return nil
}

func (n *Node) FinalizedHeader(ctx context.Context) (*types.Header, error) {
	if n == nil || n.client == nil {
		return nil, fmt.Errorf("BSC node is not configured")
	}
	header, err := n.client.HeaderByNumber(ctx, big.NewInt(int64(rpc.FinalizedBlockNumber)))
	if err != nil {
		return nil, fmt.Errorf("fetch finalized BSC header: %w", err)
	}
	if header == nil || header.Number == nil || !header.Number.IsUint64() {
		return nil, fmt.Errorf("finalized BSC header is invalid")
	}
	return header, nil
}

func (n *Node) HeaderByNumber(ctx context.Context, number uint64) (*types.Header, error) {
	if n == nil || n.client == nil {
		return nil, fmt.Errorf("BSC node is not configured")
	}
	header, err := n.client.HeaderByNumber(ctx, new(big.Int).SetUint64(number))
	if err != nil {
		return nil, fmt.Errorf("fetch BSC header %d: %w", number, err)
	}
	if header == nil || header.Number == nil || header.Number.Uint64() != number {
		return nil, fmt.Errorf("BSC header %d is invalid", number)
	}
	return header, nil
}

func (n *Node) BlockByNumber(ctx context.Context, number uint64) (*types.Block, error) {
	if n == nil || n.client == nil {
		return nil, fmt.Errorf("BSC node is not configured")
	}
	block, err := n.client.BlockByNumber(ctx, new(big.Int).SetUint64(number))
	if err != nil {
		return nil, fmt.Errorf("fetch BSC block %d: %w", number, err)
	}
	if block == nil || block.NumberU64() != number {
		return nil, fmt.Errorf("BSC block %d is invalid", number)
	}
	return block, nil
}

func (n *Node) FilterSwapLogs(ctx context.Context, start, end uint64) ([]types.Log, error) {
	if n == nil || n.client == nil {
		return nil, fmt.Errorf("BSC node is not configured")
	}
	if end < start {
		return nil, fmt.Errorf("BSC swap log range %d-%d is invalid", start, end)
	}
	logs, err := n.client.FilterLogs(ctx, ethereum.FilterQuery{
		FromBlock: new(big.Int).SetUint64(start),
		ToBlock:   new(big.Int).SetUint64(end),
		Topics:    [][]common.Hash{{V2SwapTopic}},
	})
	if err != nil {
		return nil, fmt.Errorf("filter BSC V2 swap logs for blocks %d-%d: %w", start, end, err)
	}
	return logs, nil
}
