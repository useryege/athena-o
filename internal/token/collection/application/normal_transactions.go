package application

import (
	"fmt"
	"math/big"
	"sort"
	"strings"

	"github.com/useryege/athena/internal/token/collection"
	"github.com/useryege/athena/internal/token/shared"
)

const (
	normalTransactionPageSize = 300
	normalTransactionTopLimit = 10
)

type methodCount struct {
	methodID     string
	functionName string
	count        int
}

type counterpartyCount struct {
	address shared.Address
	count   int
}

type walletTransactionKey struct {
	wallet          shared.Address
	transactionHash shared.Hash
}

func uniqueWalletsInOrder(wallets []shared.Address) []shared.Address {
	seen := make(map[shared.Address]struct{}, len(wallets))
	result := make([]shared.Address, 0, len(wallets))
	for _, wallet := range wallets {
		if wallet.IsZero() {
			continue
		}
		if _, ok := seen[wallet]; ok {
			continue
		}
		seen[wallet] = struct{}{}
		result = append(result, wallet)
	}
	return result
}

func uniqueNormalTransactionAssociationsInOrder(
	associations []collection.WalletNormalTransaction,
) ([]collection.WalletNormalTransaction, error) {
	seen := make(map[walletTransactionKey]collection.WalletNormalTransaction, len(associations))
	result := make([]collection.WalletNormalTransaction, 0, len(associations))
	for _, transaction := range associations {
		key := walletTransactionKey{wallet: transaction.Wallet, transactionHash: transaction.TransactionHash}
		if existing, ok := seen[key]; ok {
			if !sameNormalTransactionAssociation(existing, transaction) {
				return nil, fmt.Errorf(
					"wallet %s transaction %s has conflicting duplicate rows",
					transaction.Wallet.Hex(),
					transaction.TransactionHash.Hex(),
				)
			}
			continue
		}
		seen[key] = transaction
		result = append(result, transaction)
	}
	return result, nil
}

func sameNormalTransactionAssociation(left, right collection.WalletNormalTransaction) bool {
	return left.Wallet == right.Wallet &&
		left.TransactionHash == right.TransactionHash &&
		left.BlockNumber == right.BlockNumber &&
		left.BlockTimestamp == right.BlockTimestamp &&
		left.TransactionIndex == right.TransactionIndex &&
		left.Nonce == right.Nonce &&
		left.FromAddress == right.FromAddress &&
		left.ToAddress == right.ToAddress &&
		sameBigInt(left.Value, right.Value) &&
		left.Gas == right.Gas &&
		sameBigInt(left.GasPrice, right.GasPrice) &&
		left.GasUsed == right.GasUsed &&
		left.Input == right.Input &&
		left.MethodID == right.MethodID &&
		left.FunctionName == right.FunctionName &&
		left.ReceiptStatus == right.ReceiptStatus &&
		left.IsError == right.IsError &&
		left.CollectedAt.Equal(right.CollectedAt)
}

func sameBigInt(left, right *big.Int) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return left.Cmp(right) == 0
}

func summarizeNormalTransactions(
	wallets []shared.Address,
	associations []collection.WalletNormalTransaction,
	cappedWallets []shared.Address,
) collection.WalletNormalTransactionsResultV1 {
	walletSet := make(map[shared.Address]struct{}, len(wallets))
	for _, wallet := range wallets {
		walletSet[wallet] = struct{}{}
	}

	unique := make(map[shared.Hash]collection.WalletNormalTransaction, len(associations))
	for _, transaction := range associations {
		if _, exists := unique[transaction.TransactionHash]; !exists {
			unique[transaction.TransactionHash] = transaction
		}
	}

	result := collection.WalletNormalTransactionsResultV1{
		WalletCount:                 len(wallets),
		TransactionAssociationCount: len(associations),
		UniqueTransactionCount:      len(unique),
		TotalInflowNativeValue:      new(big.Int),
		TotalOutflowNativeValue:     new(big.Int),
		TopMethods:                  []collection.NormalTransactionMethodV1{},
		TopCounterparties:           []collection.NormalTransactionCounterpartyV1{},
		CappedWallets:               append([]shared.Address(nil), cappedWallets...),
	}
	methods := make(map[string]*methodCount)
	counterparties := make(map[shared.Address]*counterpartyCount)
	for _, transaction := range unique {
		switch {
		case transaction.IsError || transaction.ReceiptStatus == collection.NormalTransactionReceiptStatusFailed:
			result.FailedTransactionCount++
		case transaction.ReceiptStatus == collection.NormalTransactionReceiptStatusSuccess:
			result.SucceededTransactionCount++
		}

		methodKey := transaction.MethodID + "\x00" + transaction.FunctionName
		method := methods[methodKey]
		if method == nil {
			method = &methodCount{methodID: transaction.MethodID, functionName: transaction.FunctionName}
			methods[methodKey] = method
		}
		method.count++

		_, fromRelated := walletSet[transaction.FromAddress]
		_, toRelated := walletSet[transaction.ToAddress]
		value := transaction.Value
		if value == nil {
			value = new(big.Int)
		}
		switch {
		case !fromRelated && toRelated:
			result.TotalInflowNativeValue.Add(result.TotalInflowNativeValue, value)
			incrementCounterparty(counterparties, transaction.FromAddress)
		case fromRelated && !toRelated:
			result.TotalOutflowNativeValue.Add(result.TotalOutflowNativeValue, value)
			incrementCounterparty(counterparties, transaction.ToAddress)
		}
	}

	methodValues := make([]methodCount, 0, len(methods))
	for _, value := range methods {
		methodValues = append(methodValues, *value)
	}
	sort.Slice(methodValues, func(i, j int) bool {
		if methodValues[i].count != methodValues[j].count {
			return methodValues[i].count > methodValues[j].count
		}
		if methodValues[i].methodID != methodValues[j].methodID {
			return methodValues[i].methodID < methodValues[j].methodID
		}
		return methodValues[i].functionName < methodValues[j].functionName
	})
	for _, value := range methodValues[:min(len(methodValues), normalTransactionTopLimit)] {
		result.TopMethods = append(result.TopMethods, collection.NormalTransactionMethodV1{
			MethodID: value.methodID, FunctionName: value.functionName, Count: value.count,
		})
	}

	counterpartyValues := make([]counterpartyCount, 0, len(counterparties))
	for _, value := range counterparties {
		counterpartyValues = append(counterpartyValues, *value)
	}
	sort.Slice(counterpartyValues, func(i, j int) bool {
		if counterpartyValues[i].count != counterpartyValues[j].count {
			return counterpartyValues[i].count > counterpartyValues[j].count
		}
		return strings.Compare(counterpartyValues[i].address.Hex(), counterpartyValues[j].address.Hex()) < 0
	})
	for _, value := range counterpartyValues[:min(len(counterpartyValues), normalTransactionTopLimit)] {
		result.TopCounterparties = append(result.TopCounterparties, collection.NormalTransactionCounterpartyV1{
			Address: value.address, Count: value.count,
		})
	}
	return result
}

func incrementCounterparty(values map[shared.Address]*counterpartyCount, address shared.Address) {
	if address.IsZero() {
		return
	}
	value := values[address]
	if value == nil {
		value = &counterpartyCount{address: address}
		values[address] = value
	}
	value.count++
}
