package evm

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	log "github.com/sirupsen/logrus"
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

func (source *BlockSource) DiscoverProjectBlock(ctx context.Context, chainID int64, blockNumber uint64) (discovery.ProjectCandidateBlock, error) {
	startedAt := time.Now()
	clientStartedAt := time.Now()
	client, err := source.clients.Client(ctx, chainID)
	clientDuration := time.Since(clientStartedAt)
	if err != nil {
		logBlockDiscoveryFailure(ctx, chainID, blockNumber, "client", startedAt, clientDuration, err)
		return discovery.ProjectCandidateBlock{}, err
	}
	blockFetchStartedAt := time.Now()
	block, err := client.BlockByNumber(ctx, new(big.Int).SetUint64(blockNumber))
	blockFetchDuration := time.Since(blockFetchStartedAt)
	if err != nil {
		source.clients.Reset(chainID)
		err = fmt.Errorf("fetch block %d: %w", blockNumber, err)
		logBlockDiscoveryFailure(ctx, chainID, blockNumber, "block_fetch", startedAt, blockFetchDuration, err)
		return discovery.ProjectCandidateBlock{}, err
	}
	if block == nil {
		err = fmt.Errorf("fetch block %d returned nil block", blockNumber)
		logBlockDiscoveryFailure(ctx, chainID, blockNumber, "block_fetch", startedAt, blockFetchDuration, err)
		return discovery.ProjectCandidateBlock{}, err
	}
	if block.NumberU64() != blockNumber {
		err = fmt.Errorf("fetch block %d returned block %d", blockNumber, block.NumberU64())
		logBlockDiscoveryFailure(ctx, chainID, blockNumber, "block_fetch", startedAt, blockFetchDuration, err)
		return discovery.ProjectCandidateBlock{}, err
	}
	candidateExtractionStartedAt := time.Now()
	signer := types.LatestSignerForChainID(big.NewInt(chainID))
	candidates := make([]discovery.ProjectCandidate, 0)
	transactions := block.Transactions()
	for transactionIndex, transaction := range transactions {
		if transaction == nil || transaction.To() != nil {
			continue
		}
		sender, err := types.Sender(signer, transaction)
		if err != nil {
			err = fmt.Errorf("derive transaction sender for tx %s: %w", transaction.Hash().Hex(), err)
			logBlockDiscoveryFailure(ctx, chainID, blockNumber, "candidate_extraction", startedAt, time.Since(candidateExtractionStartedAt), err)
			return discovery.ProjectCandidateBlock{}, err
		}
		deploymentNonce := transaction.Nonce()
		contract := crypto.CreateAddress(sender, deploymentNonce)
		if contract == (common.Address{}) {
			continue
		}
		candidates = append(candidates, discovery.ProjectCandidate{ChainID: chainID, Contract: shared.Address(contract), TxSender: shared.Address(sender), TxHash: shared.Hash(transaction.Hash()), TxIndex: uint64(transactionIndex), DeploymentNonce: deploymentNonce, BlockNumber: block.NumberU64(), BlockTime: block.Time()})
	}
	candidateExtractionDuration := time.Since(candidateExtractionStartedAt)
	log.WithFields(log.Fields{
		"block_number":                     blockNumber,
		"returned_block_number":            block.NumberU64(),
		"chain_id":                         chainID,
		"transaction_count":                len(transactions),
		"candidate_count":                  len(candidates),
		"client_duration_ms":               clientDuration.Milliseconds(),
		"block_fetch_duration_ms":          blockFetchDuration.Milliseconds(),
		"candidate_extraction_duration_ms": candidateExtractionDuration.Milliseconds(),
		"duration_ms":                      time.Since(startedAt).Milliseconds(),
	}).Info("token chain block discovery completed")
	return discovery.ProjectCandidateBlock{
		Header:     discovery.BlockHeader{Number: block.NumberU64(), Timestamp: block.Time()},
		Candidates: candidates,
	}, nil
}

func logBlockDiscoveryFailure(
	ctx context.Context,
	chainID int64,
	blockNumber uint64,
	stage string,
	startedAt time.Time,
	phaseDuration time.Duration,
	err error,
) {
	if ctx.Err() != nil {
		return
	}
	log.WithError(err).WithFields(log.Fields{
		"block_number":      blockNumber,
		"chain_id":          chainID,
		"stage":             stage,
		"phase_duration_ms": phaseDuration.Milliseconds(),
		"duration_ms":       time.Since(startedAt).Milliseconds(),
	}).Error("token chain block discovery failed")
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
