package evm

import (
	"bytes"
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/useryege/athena/internal/token/shared"
	"github.com/useryege/athena/internal/token/swap"
	"github.com/useryege/athena/pkg/abi/IPancakePair"
)

type SwapBlockSource struct {
	clients   *ChainClientRegistry
	swapEvent abi.Event
}

func NewSwapBlockSource(clients *ChainClientRegistry) (*SwapBlockSource, error) {
	if clients == nil {
		return nil, fmt.Errorf("token swap EVM client registry is required")
	}
	parsed, err := IPancakePair.IPancakePairMetaData.GetAbi()
	if err != nil {
		return nil, fmt.Errorf("parse Pancake Pair ABI: %w", err)
	}
	if parsed == nil {
		return nil, fmt.Errorf("Pancake Pair ABI is empty")
	}
	event, ok := parsed.Events["Swap"]
	if !ok {
		return nil, fmt.Errorf("Pancake Pair ABI does not contain Swap event")
	}
	return &SwapBlockSource{clients: clients, swapEvent: event}, nil
}

func (source *SwapBlockSource) FilterSwapLogs(ctx context.Context, chainID int64, blockNumber uint64) ([]swap.RawLog, error) {
	if source == nil || source.clients == nil {
		return nil, fmt.Errorf("token swap block source is not configured")
	}
	client, err := source.clients.Client(ctx, chainID)
	if err != nil {
		return nil, err
	}
	number := new(big.Int).SetUint64(blockNumber)
	logs, err := client.FilterLogs(ctx, ethereum.FilterQuery{
		FromBlock: number,
		ToBlock:   new(big.Int).Set(number),
		Topics:    [][]common.Hash{{source.swapEvent.ID}},
	})
	if err != nil {
		source.clients.Reset(chainID)
		return nil, fmt.Errorf("filter Swap logs for block %d: %w", blockNumber, err)
	}
	result := make([]swap.RawLog, 0, len(logs))
	for _, item := range logs {
		if item.Removed {
			source.clients.Reset(chainID)
			return nil, fmt.Errorf("Swap log %s:%d at block %d is marked removed", item.TxHash.Hex(), item.Index, blockNumber)
		}
		if item.BlockNumber != blockNumber {
			source.clients.Reset(chainID)
			return nil, fmt.Errorf("Swap log %s:%d returned block %d for requested block %d", item.TxHash.Hex(), item.Index, item.BlockNumber, blockNumber)
		}
		if len(item.Topics) == 0 || item.Topics[0] != source.swapEvent.ID {
			source.clients.Reset(chainID)
			return nil, fmt.Errorf("Swap log %s:%d at block %d has an unexpected topic", item.TxHash.Hex(), item.Index, blockNumber)
		}
		topics := make([]shared.Hash, len(item.Topics))
		for index, topic := range item.Topics {
			topics[index] = shared.Hash(topic)
		}
		result = append(result, swap.RawLog{
			Address:          shared.Address(item.Address),
			BlockNumber:      item.BlockNumber,
			TransactionHash:  shared.Hash(item.TxHash),
			TransactionIndex: uint64(item.TxIndex),
			LogIndex:         uint64(item.Index),
			Topics:           topics,
			Data:             append([]byte(nil), item.Data...),
			Removed:          item.Removed,
		})
	}
	return result, nil
}

func (source *SwapBlockSource) BlockTimestamp(ctx context.Context, chainID int64, blockNumber uint64) (uint64, error) {
	if source == nil || source.clients == nil {
		return 0, fmt.Errorf("token swap block source is not configured")
	}
	client, err := source.clients.Client(ctx, chainID)
	if err != nil {
		return 0, err
	}
	header, err := client.HeaderByNumber(ctx, new(big.Int).SetUint64(blockNumber))
	if err != nil {
		source.clients.Reset(chainID)
		return 0, fmt.Errorf("fetch Swap block header %d: %w", blockNumber, err)
	}
	if header == nil || header.Number == nil || !header.Number.IsUint64() || header.Number.Uint64() != blockNumber {
		source.clients.Reset(chainID)
		return 0, fmt.Errorf("Swap block header %d is invalid", blockNumber)
	}
	return header.Time, nil
}

func (source *SwapBlockSource) DecodeBlockSwapEvents(ctx context.Context, chainID int64, blockNumber uint64, logs []swap.RawLog) (swap.BlockObservation, error) {
	if source == nil || source.clients == nil {
		return swap.BlockObservation{}, fmt.Errorf("token swap block source is not configured")
	}
	if len(logs) == 0 {
		return swap.BlockObservation{}, fmt.Errorf("Swap block %d has no relevant logs to decode", blockNumber)
	}
	client, err := source.clients.Client(ctx, chainID)
	if err != nil {
		return swap.BlockObservation{}, err
	}
	rpcStartedAt := time.Now()
	block, err := client.BlockByNumber(ctx, new(big.Int).SetUint64(blockNumber))
	rpcDuration := time.Since(rpcStartedAt)
	if err != nil {
		source.clients.Reset(chainID)
		return swap.BlockObservation{}, fmt.Errorf("fetch Swap block %d: %w", blockNumber, err)
	}
	if block == nil || block.NumberU64() != blockNumber {
		source.clients.Reset(chainID)
		return swap.BlockObservation{}, fmt.Errorf("Swap block %d is invalid", blockNumber)
	}

	decodeStartedAt := time.Now()
	transactions := block.Transactions()
	signer := types.LatestSignerForChainID(big.NewInt(chainID))
	senders := make(map[shared.Hash]shared.Address)
	result := make([]swap.DecodedEvent, 0, len(logs))
	for _, item := range logs {
		if err := source.validateRelevantLog(blockNumber, item); err != nil {
			return swap.BlockObservation{}, err
		}
		if item.TransactionIndex >= uint64(len(transactions)) {
			source.clients.Reset(chainID)
			return swap.BlockObservation{}, fmt.Errorf("Swap log %s:%d transaction index %d is outside block %d", item.TransactionHash.Hex(), item.LogIndex, item.TransactionIndex, blockNumber)
		}
		transaction := transactions[item.TransactionIndex]
		if transaction == nil || shared.Hash(transaction.Hash()) != item.TransactionHash {
			source.clients.Reset(chainID)
			return swap.BlockObservation{}, fmt.Errorf("Swap log %s:%d transaction does not match block %d index %d", item.TransactionHash.Hex(), item.LogIndex, blockNumber, item.TransactionIndex)
		}
		txFrom, ok := senders[item.TransactionHash]
		if !ok {
			sender, err := types.Sender(signer, transaction)
			if err != nil {
				return swap.BlockObservation{}, fmt.Errorf("derive sender for Swap transaction %s: %w", item.TransactionHash.Hex(), err)
			}
			txFrom = shared.Address(sender)
			senders[item.TransactionHash] = txFrom
		}
		decoded, err := source.decodeLog(item, txFrom)
		if err != nil {
			return swap.BlockObservation{}, err
		}
		result = append(result, decoded)
	}
	return swap.BlockObservation{
		BlockTime:      block.Time(),
		Events:         result,
		RPCDuration:    rpcDuration,
		DecodeDuration: time.Since(decodeStartedAt),
	}, nil
}

func (source *SwapBlockSource) validateRelevantLog(blockNumber uint64, item swap.RawLog) error {
	if item.Removed {
		return fmt.Errorf("relevant Swap log %s:%d is marked removed", item.TransactionHash.Hex(), item.LogIndex)
	}
	if item.BlockNumber != blockNumber {
		return fmt.Errorf("relevant Swap log %s:%d belongs to block %d instead of %d", item.TransactionHash.Hex(), item.LogIndex, item.BlockNumber, blockNumber)
	}
	if item.TransactionHash.IsZero() {
		return fmt.Errorf("relevant Swap log at block %d index %d has an empty transaction hash", blockNumber, item.LogIndex)
	}
	if len(item.Topics) != 3 {
		return fmt.Errorf("relevant Swap log %s:%d must contain exactly three topics", item.TransactionHash.Hex(), item.LogIndex)
	}
	if common.Hash(item.Topics[0]) != source.swapEvent.ID {
		return fmt.Errorf("relevant Swap log %s:%d has an unexpected topic", item.TransactionHash.Hex(), item.LogIndex)
	}
	if len(item.Data) != 4*common.HashLength {
		return fmt.Errorf("relevant Swap log %s:%d data must contain exactly four uint256 values", item.TransactionHash.Hex(), item.LogIndex)
	}
	return nil
}

func (source *SwapBlockSource) decodeLog(item swap.RawLog, txFrom shared.Address) (swap.DecodedEvent, error) {
	values, err := source.swapEvent.Inputs.NonIndexed().Unpack(item.Data)
	if err != nil {
		return swap.DecodedEvent{}, fmt.Errorf("decode relevant Swap log %s:%d: %w", item.TransactionHash.Hex(), item.LogIndex, err)
	}
	if len(values) != 4 {
		return swap.DecodedEvent{}, fmt.Errorf("relevant Swap log %s:%d decoded %d values instead of four", item.TransactionHash.Hex(), item.LogIndex, len(values))
	}
	amounts := make([]*big.Int, len(values))
	for index, value := range values {
		amount, ok := value.(*big.Int)
		if !ok || amount == nil || amount.Sign() < 0 {
			return swap.DecodedEvent{}, fmt.Errorf("relevant Swap log %s:%d amount %d is invalid", item.TransactionHash.Hex(), item.LogIndex, index)
		}
		amounts[index] = new(big.Int).Set(amount)
	}
	sender, err := strictTopicAddress(item.Topics[1])
	if err != nil {
		return swap.DecodedEvent{}, fmt.Errorf("decode relevant Swap log %s:%d sender: %w", item.TransactionHash.Hex(), item.LogIndex, err)
	}
	toAddress, err := strictTopicAddress(item.Topics[2])
	if err != nil {
		return swap.DecodedEvent{}, fmt.Errorf("decode relevant Swap log %s:%d recipient: %w", item.TransactionHash.Hex(), item.LogIndex, err)
	}
	return swap.DecodedEvent{
		PairAddress:      item.Address,
		TransactionHash:  item.TransactionHash,
		TransactionIndex: item.TransactionIndex,
		LogIndex:         item.LogIndex,
		TxFrom:           txFrom,
		Sender:           sender,
		ToAddress:        toAddress,
		Amount0In:        amounts[0],
		Amount1In:        amounts[1],
		Amount0Out:       amounts[2],
		Amount1Out:       amounts[3],
	}, nil
}

func strictTopicAddress(topic shared.Hash) (shared.Address, error) {
	if !bytes.Equal(topic[:12], make([]byte, 12)) {
		return shared.Address{}, fmt.Errorf("indexed address has non-zero padding")
	}
	return shared.BytesToAddress(topic[12:]), nil
}
