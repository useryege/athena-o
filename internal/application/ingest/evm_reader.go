package ingest

import (
	"context"
	"errors"
	"fmt"
	"math/big"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

var uniswapV2SwapTopic = common.HexToHash("0xd78ad95fa46c994b6551d0da85fc275fe613f1621ef4b3a6c36c5b3313c4")

type EVMReader struct {
	client  *ethclient.Client
	chainID int64
	signer  types.Signer
}

func NewEVMReader(client *ethclient.Client, chainID int64) *EVMReader {
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
	for txIndex, tx := range block.Transactions() {
		creation, err := contractCreationFromTransaction(r.signer, tx, txIndex)
		if err != nil {
			return Block{}, err
		}
		if creation != nil {
			item.ContractCreations = append(item.ContractCreations, *creation)
		}
		receipt, err := r.client.TransactionReceipt(ctx, tx.Hash())
		if err != nil {
			if errors.Is(err, ethereum.NotFound) {
				continue
			}
			return Block{}, err
		}
		item.DexSwaps = append(item.DexSwaps, dexSwapsFromReceipt(tx.Hash(), receipt)...)
	}
	return item, nil
}

func (r *EVMReader) ReadContractCode(ctx context.Context, contract common.Address) ([]byte, error) {
	return r.client.CodeAt(ctx, contract, nil)
}

func contractCreationFromTransaction(signer types.Signer, tx *types.Transaction, txIndex int) (*ContractCreated, error) {
	if tx == nil || tx.To() != nil {
		return nil, nil
	}
	creator, err := types.Sender(signer, tx)
	if err != nil {
		return nil, fmt.Errorf("derive contract creator for tx %s: %w", tx.Hash().Hex(), err)
	}
	contract := crypto.CreateAddress(creator, tx.Nonce())
	if contract == (common.Address{}) {
		return nil, nil
	}
	return &ContractCreated{
		Contract: contract,
		Creator:  creator,
		TxHash:   tx.Hash(),
		TxIndex:  uint64(txIndex),
	}, nil
}

func dexSwapsFromReceipt(txHash common.Hash, receipt *types.Receipt) []DexSwap {
	if receipt == nil {
		return nil
	}
	items := make([]DexSwap, 0)
	for _, logItem := range receipt.Logs {
		if logItem == nil || len(logItem.Topics) == 0 || logItem.Topics[0] != uniswapV2SwapTopic {
			continue
		}
		items = append(items, DexSwap{
			Pair:   logItem.Address,
			TxHash: txHash,
		})
	}
	return items
}
