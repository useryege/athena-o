package bscinbound

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
)

const bscMainnetChainID = int64(56)

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

func (n *Node) TransactionReceipt(ctx context.Context, transactionHash common.Hash) (*types.Receipt, error) {
	if n == nil || n.client == nil {
		return nil, fmt.Errorf("BSC node is not configured")
	}
	receipt, err := n.client.TransactionReceipt(ctx, transactionHash)
	if err != nil {
		return nil, fmt.Errorf("fetch BSC transaction receipt %x: %w", transactionHash, err)
	}
	if receipt == nil {
		return nil, fmt.Errorf("BSC transaction receipt %x is empty", transactionHash)
	}
	return receipt, nil
}
