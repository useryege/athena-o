package evm

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/useryege/athena/internal/token/discovery"
	"github.com/useryege/athena/internal/token/shared"
)

type BlockSource struct{ clients *ChainClientRegistry }

func NewBlockSource(clients *ChainClientRegistry) *BlockSource { return &BlockSource{clients: clients} }

func (source *BlockSource) LatestBlockHeader(ctx context.Context, chainID int64) (discovery.BlockHeader, error) {
	client, err := source.clients.Client(ctx, chainID)
	if err != nil {
		return discovery.BlockHeader{}, err
	}
	header, err := client.HeaderByNumber(ctx, nil)
	if err != nil {
		source.clients.Reset(chainID)
		return discovery.BlockHeader{}, fmt.Errorf("fetch latest block header: %w", err)
	}
	mapped, err := mapBlockHeader(header, nil)
	if err != nil {
		source.clients.Reset(chainID)
		return discovery.BlockHeader{}, err
	}
	return mapped, nil
}

func (source *BlockSource) BlockHeaderByNumber(ctx context.Context, chainID int64, number uint64) (discovery.BlockHeader, error) {
	client, err := source.clients.Client(ctx, chainID)
	if err != nil {
		return discovery.BlockHeader{}, err
	}
	header, err := client.HeaderByNumber(ctx, new(big.Int).SetUint64(number))
	if err != nil {
		source.clients.Reset(chainID)
		return discovery.BlockHeader{}, fmt.Errorf("fetch block header %d: %w", number, err)
	}
	mapped, err := mapBlockHeader(header, &number)
	if err != nil {
		source.clients.Reset(chainID)
		return discovery.BlockHeader{}, err
	}
	return mapped, nil
}

func (source *BlockSource) DiscoverProjectCandidates(ctx context.Context, chainID int64, blockNumber uint64) ([]discovery.ProjectCandidate, error) {
	client, err := source.clients.Client(ctx, chainID)
	if err != nil {
		return nil, err
	}
	block, err := client.BlockByNumber(ctx, new(big.Int).SetUint64(blockNumber))
	if err != nil {
		source.clients.Reset(chainID)
		return nil, fmt.Errorf("fetch block %d: %w", blockNumber, err)
	}
	if block == nil {
		return nil, fmt.Errorf("fetch block %d returned nil block", blockNumber)
	}
	signer := types.LatestSignerForChainID(big.NewInt(chainID))
	candidates := make([]discovery.ProjectCandidate, 0)
	for transactionIndex, transaction := range block.Transactions() {
		if transaction == nil || transaction.To() != nil {
			continue
		}
		sender, err := types.Sender(signer, transaction)
		if err != nil {
			return nil, fmt.Errorf("derive transaction sender for tx %s: %w", transaction.Hash().Hex(), err)
		}
		contract := crypto.CreateAddress(sender, transaction.Nonce())
		if contract == (common.Address{}) {
			continue
		}
		candidates = append(candidates, discovery.ProjectCandidate{ChainID: chainID, Contract: shared.Address(contract), TxSender: shared.Address(sender), TxHash: shared.Hash(transaction.Hash()), TxIndex: uint64(transactionIndex), BlockNumber: block.NumberU64(), BlockTime: block.Time(), Status: discovery.ProjectCandidateStatusPending})
	}
	return candidates, nil
}

func mapBlockHeader(header *types.Header, expectedNumber *uint64) (discovery.BlockHeader, error) {
	if header == nil || header.Number == nil || !header.Number.IsUint64() {
		return discovery.BlockHeader{}, fmt.Errorf("block header is invalid")
	}
	number := header.Number.Uint64()
	if expectedNumber != nil && number != *expectedNumber {
		return discovery.BlockHeader{}, fmt.Errorf("block header %d returned number %d", *expectedNumber, number)
	}
	return discovery.BlockHeader{Number: number, Timestamp: header.Time}, nil
}
