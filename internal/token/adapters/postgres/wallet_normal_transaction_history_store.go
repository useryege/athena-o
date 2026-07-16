package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	tokensqlc "github.com/useryege/athena/internal/token/adapters/postgres/sqlc"
	"github.com/useryege/athena/internal/token/research"
	researchapp "github.com/useryege/athena/internal/token/research/application"
	"github.com/useryege/athena/internal/token/shared"
)

func (repository *CollectionRepository) ListPendingWalletNormalTransactionHistoryWallets(ctx context.Context, projectID int64) ([]shared.Address, error) {
	queries, err := repository.querier()
	if err != nil {
		return nil, err
	}
	rows, err := queries.ListPendingProjectWalletNormalTransactionHistoryWallets(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("list pending wallet normal transaction histories: %w", err)
	}
	wallets := make([]shared.Address, 0, len(rows))
	for _, row := range rows {
		wallets = append(wallets, bytesToAddress(row))
	}
	return wallets, nil
}

func (repository *CollectionRepository) SaveWalletNormalTransactionHistory(ctx context.Context, command researchapp.SaveWalletNormalTransactionHistoryCommand) (bool, error) {
	if repository == nil || repository.pool == nil {
		return false, fmt.Errorf("token collection repository is not configured")
	}
	if command.Wallet.IsZero() {
		return false, fmt.Errorf("wallet normal transaction history wallet is required")
	}
	if command.RequestedCount <= 0 || len(command.Transactions) > int(command.RequestedCount) {
		return false, fmt.Errorf("wallet normal transaction history count is invalid")
	}
	if command.FetchedAt.IsZero() {
		return false, fmt.Errorf("wallet normal transaction history fetched time is required")
	}
	anchorBlock, err := uint64ToInt64("anchor_block_number", command.Project.CreationBlockNumber)
	if err != nil {
		return false, err
	}
	anchorTransactionIndex, err := uint64ToInt64("anchor_transaction_index", command.Project.CreationTransactionIndex)
	if err != nil {
		return false, err
	}
	params := make([]tokensqlc.InsertProjectWalletNormalTransactionParams, 0, len(command.Transactions))
	for index, transaction := range command.Transactions {
		item, err := walletNormalTransactionParams(command.Project.ID, command.Wallet, int32(index), transaction)
		if err != nil {
			return false, err
		}
		params = append(params, item)
	}

	tx, err := repository.pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := tokensqlc.New(tx)
	_, err = queries.InsertProjectWalletNormalTransactionHistory(ctx, tokensqlc.InsertProjectWalletNormalTransactionHistoryParams{
		ProjectID: command.Project.ID, Wallet: command.Wallet.Bytes(), AnchorBlockNumber: anchorBlock,
		AnchorTransactionIndex: anchorTransactionIndex, RequestedTransactionCount: command.RequestedCount,
		CollectedTransactionCount: int32(len(command.Transactions)), FetchedAt: nullableTime(command.FetchedAt),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("insert wallet normal transaction history: %w", err)
	}
	for _, item := range params {
		if err = queries.InsertProjectWalletNormalTransaction(ctx, item); err != nil {
			return false, fmt.Errorf("insert wallet normal transaction rank=%d: %w", item.RankIndex, err)
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return false, err
	}
	return true, nil
}

func (repository *CollectionRepository) RenewCollectionTaskLease(ctx context.Context, taskID int64, lease time.Duration) error {
	if lease <= 0 {
		return fmt.Errorf("collection task lease must be positive")
	}
	queries, err := repository.querier()
	if err != nil {
		return err
	}
	rows, err := queries.RenewProjectDataCollectionTaskLease(ctx, tokensqlc.RenewProjectDataCollectionTaskLeaseParams{LeaseSeconds: int64(lease / time.Second), ID: taskID})
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("collection task %d is no longer running", taskID)
	}
	return nil
}

func (repository *CollectionRepository) CompleteWalletNormalTransactionHistory(ctx context.Context, command researchapp.CompleteWalletNormalTransactionHistoryCommand) (bool, error) {
	if repository == nil || repository.pool == nil {
		return false, fmt.Errorf("token collection repository is not configured")
	}
	tx, err := repository.pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := tokensqlc.New(tx)
	rows, err := queries.MarkProjectDataCollectionTaskSucceeded(ctx, command.Task.ID)
	if err != nil {
		return false, err
	}
	if rows == 0 {
		return false, nil
	}
	rows, err = queries.CompleteProjectDataCollectionSchedule(ctx, tokensqlc.CompleteProjectDataCollectionScheduleParams{
		LastCheckedAt: nullableTime(command.CompletedAt), ProjectID: command.Task.ProjectID, DataType: string(command.Task.DataType),
	})
	if err != nil {
		return false, err
	}
	if rows == 0 {
		return false, fmt.Errorf("wallet normal transaction history schedule was not found")
	}
	if err = tx.Commit(ctx); err != nil {
		return false, err
	}
	return true, nil
}

func walletNormalTransactionParams(projectID int64, wallet shared.Address, rankIndex int32, transaction research.WalletNormalTransaction) (tokensqlc.InsertProjectWalletNormalTransactionParams, error) {
	blockNumber, err := uint64ToInt64("block_number", transaction.BlockNumber)
	if err != nil {
		return tokensqlc.InsertProjectWalletNormalTransactionParams{}, err
	}
	blockTimestamp, err := uint64ToInt64("block_timestamp", transaction.BlockTimestamp)
	if err != nil {
		return tokensqlc.InsertProjectWalletNormalTransactionParams{}, err
	}
	nonce, err := uint64ToInt64("nonce", transaction.Nonce)
	if err != nil {
		return tokensqlc.InsertProjectWalletNormalTransactionParams{}, err
	}
	transactionIndex, err := uint64ToInt64("transaction_index", transaction.TransactionIndex)
	if err != nil {
		return tokensqlc.InsertProjectWalletNormalTransactionParams{}, err
	}
	gas, err := uint64ToInt64("gas", transaction.Gas)
	if err != nil {
		return tokensqlc.InsertProjectWalletNormalTransactionParams{}, err
	}
	cumulativeGasUsed, err := uint64ToInt64("cumulative_gas_used", transaction.CumulativeGasUsed)
	if err != nil {
		return tokensqlc.InsertProjectWalletNormalTransactionParams{}, err
	}
	gasUsed, err := uint64ToInt64("gas_used", transaction.GasUsed)
	if err != nil {
		return tokensqlc.InsertProjectWalletNormalTransactionParams{}, err
	}
	confirmations, err := uint64ToInt64("confirmations", transaction.Confirmations)
	if err != nil {
		return tokensqlc.InsertProjectWalletNormalTransactionParams{}, err
	}
	if len(transaction.MethodID) != 0 && len(transaction.MethodID) != 4 {
		return tokensqlc.InsertProjectWalletNormalTransactionParams{}, fmt.Errorf("method_id must contain 4 bytes")
	}
	return tokensqlc.InsertProjectWalletNormalTransactionParams{
		ProjectID: projectID, Wallet: wallet.Bytes(), RankIndex: rankIndex,
		BlockNumber: blockNumber, BlockHash: transaction.BlockHash.Bytes(), BlockTimestamp: blockTimestamp,
		TransactionHash: transaction.TransactionHash.Bytes(), Nonce: nonce, TransactionIndex: transactionIndex,
		FromAddress: transaction.From.Bytes(), ToAddress: optionalAddressBytes(transaction.To),
		Value: numericFromBigInt(transaction.Value), Gas: gas, GasPrice: numericFromBigInt(transaction.GasPrice),
		Input: transaction.Input, MethodID: append([]byte(nil), transaction.MethodID...), FunctionName: transaction.FunctionName,
		ContractAddress: optionalAddressBytes(transaction.ContractAddress), CumulativeGasUsed: cumulativeGasUsed,
		ReceiptStatus: string(transaction.ReceiptStatus), GasUsed: gasUsed, Confirmations: confirmations, IsError: transaction.IsError,
	}, nil
}
