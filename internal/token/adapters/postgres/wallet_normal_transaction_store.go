package postgres

import (
	"context"
	"fmt"
	"time"

	tokensqlc "github.com/useryege/athena/internal/token/adapters/postgres/sqlc"
	"github.com/useryege/athena/internal/token/research"
)

func insertWalletNormalTransactions(
	ctx context.Context,
	queries *tokensqlc.Queries,
	projectID int64,
	collectedAt time.Time,
	transactions []research.WalletNormalTransaction,
) error {
	for index, transaction := range transactions {
		blockNumber, err := uint64ToInt64("block_number", transaction.BlockNumber)
		if err != nil {
			return fmt.Errorf("map wallet normal transaction %d: %w", index, err)
		}
		blockTimestamp, err := uint64ToInt64("block_timestamp", transaction.BlockTimestamp)
		if err != nil {
			return fmt.Errorf("map wallet normal transaction %d: %w", index, err)
		}
		transactionIndex, err := uint64ToInt64("transaction_index", transaction.TransactionIndex)
		if err != nil {
			return fmt.Errorf("map wallet normal transaction %d: %w", index, err)
		}
		nonce, err := uint64ToInt64("nonce", transaction.Nonce)
		if err != nil {
			return fmt.Errorf("map wallet normal transaction %d: %w", index, err)
		}
		gas, err := uint64ToInt64("gas", transaction.Gas)
		if err != nil {
			return fmt.Errorf("map wallet normal transaction %d: %w", index, err)
		}
		gasUsed, err := uint64ToInt64("gas_used", transaction.GasUsed)
		if err != nil {
			return fmt.Errorf("map wallet normal transaction %d: %w", index, err)
		}
		if err = queries.InsertProjectWalletNormalTransaction(ctx, tokensqlc.InsertProjectWalletNormalTransactionParams{
			ProjectID:        projectID,
			Wallet:           transaction.Wallet.Bytes(),
			TransactionHash:  transaction.TransactionHash.Bytes(),
			BlockNumber:      blockNumber,
			BlockTimestamp:   blockTimestamp,
			TransactionIndex: transactionIndex,
			Nonce:            nonce,
			FromAddress:      transaction.FromAddress.Bytes(),
			ToAddress:        optionalAddressBytes(transaction.ToAddress),
			Value:            numericFromBigInt(transaction.Value),
			Gas:              gas,
			GasPrice:         numericFromBigInt(transaction.GasPrice),
			GasUsed:          gasUsed,
			Input:            transaction.Input,
			MethodID:         transaction.MethodID,
			FunctionName:     transaction.FunctionName,
			ReceiptStatus:    string(transaction.ReceiptStatus),
			IsError:          transaction.IsError,
			CollectedAt:      nullableTime(collectedAt),
		}); err != nil {
			return fmt.Errorf("insert wallet normal transaction %d: %w", index, err)
		}
	}
	return nil
}
