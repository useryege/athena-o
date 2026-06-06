package ingest

import (
	"context"
	"errors"
	"math/big"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

var uniswapV2SwapTopic = common.HexToHash("0xd78ad95fa46c994b6551d0da85fc275fe613f1621ef4b3a6c36c5b3313c4")

type EVMClient interface {
	BlockNumber(ctx context.Context) (uint64, error)
	BlockByNumber(ctx context.Context, number *big.Int) (*types.Block, error)
	TransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error)
	CodeAt(ctx context.Context, account common.Address, blockNumber *big.Int) ([]byte, error)
}

type EVMReader struct {
	client  EVMClient
	chainID int64
	signer  types.Signer
}

func NewEVMReader(client EVMClient, chainID int64) *EVMReader {
	if client == nil || chainID <= 0 {
		return nil
	}
	return &EVMReader{
		client:  client,
		chainID: chainID,
		signer:  types.LatestSignerForChainID(big.NewInt(chainID)),
	}
}

func (r *EVMReader) LatestBlockNumber(ctx context.Context) (uint64, error) {
	return r.client.BlockNumber(ctx)
}

func (r *EVMReader) ReadBlock(ctx context.Context, number uint64) (Block, error) {
	block, err := r.client.BlockByNumber(ctx, new(big.Int).SetUint64(number))
	if err != nil {
		return Block{}, err
	}
	item := Block{
		ChainID: r.chainID,
		Number:  block.NumberU64(),
		Hash:    block.Hash(),
		Time:    block.Time(),
	}
	for _, tx := range block.Transactions() {
		receipt, err := r.client.TransactionReceipt(ctx, tx.Hash())
		if err != nil {
			if errors.Is(err, ethereum.NotFound) {
				continue
			}
			return Block{}, err
		}
		if tx.To() == nil && receipt != nil && receipt.ContractAddress != (common.Address{}) {
			creator, _ := types.Sender(r.signer, tx)
			item.ContractCreations = append(item.ContractCreations, ContractCreated{
				Contract: receipt.ContractAddress,
				Creator:  creator,
				TxHash:   tx.Hash(),
				TxIndex:  uint64(receipt.TransactionIndex),
			})
		}
		for _, logItem := range receipt.Logs {
			if len(logItem.Topics) == 0 || logItem.Topics[0] != uniswapV2SwapTopic {
				continue
			}
			item.DexSwaps = append(item.DexSwaps, DexSwap{
				Pair:   logItem.Address,
				TxHash: tx.Hash(),
			})
		}
	}
	return item, nil
}

func (r *EVMReader) ReadContractCode(ctx context.Context, contract common.Address, blockNumber uint64) ([]byte, error) {
	return r.client.CodeAt(ctx, contract, new(big.Int).SetUint64(blockNumber))
}
