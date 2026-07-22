package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	tokensqlc "github.com/useryege/athena/internal/token/adapters/postgres/sqlc"
	researchapp "github.com/useryege/athena/internal/token/research/application"
	"github.com/useryege/athena/internal/token/shared"
)

func (repository *CollectionRepository) ListPendingWalletSwapTransactionHistoryWallets(ctx context.Context, projectID int64) ([]shared.Address, error) {
	queries, err := repository.querier()
	if err != nil {
		return nil, err
	}
	rows, err := queries.ListPendingProjectWalletSwapTransactionHistoryWallets(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("list pending wallet swap transaction histories: %w", err)
	}
	wallets := make([]shared.Address, 0, len(rows))
	for _, row := range rows {
		wallets = append(wallets, bytesToAddress(row))
	}
	return wallets, nil
}

func (repository *CollectionRepository) SaveWalletSwapTransactionHistory(ctx context.Context, command researchapp.SaveWalletSwapTransactionHistoryCommand) (bool, error) {
	if repository == nil || repository.pool == nil {
		return false, fmt.Errorf("token collection repository is not configured")
	}
	if command.Wallet.IsZero() {
		return false, fmt.Errorf("wallet swap transaction history wallet is required")
	}
	if command.RequestedCount <= 0 || len(command.History.Transactions) > int(command.RequestedCount) {
		return false, fmt.Errorf("wallet swap transaction history count is invalid")
	}
	if command.FetchedAt.IsZero() {
		return false, fmt.Errorf("wallet swap transaction history fetched time is required")
	}
	anchorBlock, err := uint64ToInt64("anchor_block_number", command.Project.CreationBlockNumber)
	if err != nil || anchorBlock == 0 {
		return false, fmt.Errorf("wallet swap transaction history anchor block is invalid")
	}
	indexedThroughBlock, err := uint64ToInt64("indexed_through_block", command.History.IndexedThroughBlock)
	if err != nil {
		return false, err
	}
	indexedThroughTimestamp, err := uint64ToInt64("indexed_through_timestamp", command.History.IndexedThroughTimestamp)
	if err != nil {
		return false, err
	}
	params := make([]tokensqlc.InsertProjectWalletSwapTransactionParams, 0, len(command.History.Transactions))
	seen := make(map[shared.Hash]struct{}, len(command.History.Transactions))
	for index, transaction := range command.History.Transactions {
		if transaction.TransactionHash.IsZero() {
			return false, fmt.Errorf("wallet swap transaction rank=%d has an invalid hash", index)
		}
		if _, exists := seen[transaction.TransactionHash]; exists {
			return false, fmt.Errorf("wallet swap transaction rank=%d has a duplicate hash", index)
		}
		seen[transaction.TransactionHash] = struct{}{}
		params = append(params, tokensqlc.InsertProjectWalletSwapTransactionParams{
			ProjectID: command.Project.ID, Wallet: command.Wallet.Bytes(), RankIndex: int32(index),
			TransactionHash: transaction.TransactionHash.Bytes(),
		})
	}

	tx, err := repository.pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := tokensqlc.New(tx)
	_, err = queries.InsertProjectWalletSwapTransactionHistory(ctx, tokensqlc.InsertProjectWalletSwapTransactionHistoryParams{
		ProjectID: command.Project.ID, Wallet: command.Wallet.Bytes(), AnchorBlockNumber: anchorBlock,
		RequestedTransactionCount: command.RequestedCount, CollectedTransactionCount: int32(len(command.History.Transactions)),
		IndexedThroughBlock: indexedThroughBlock, IndexedThroughTimestamp: indexedThroughTimestamp,
		FetchedAt: nullableTime(command.FetchedAt),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("insert wallet swap transaction history: %w", err)
	}
	for _, item := range params {
		if err = queries.InsertProjectWalletSwapTransaction(ctx, item); err != nil {
			return false, fmt.Errorf("insert wallet swap transaction rank=%d: %w", item.RankIndex, err)
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return false, err
	}
	return true, nil
}

func (repository *CollectionRepository) CompleteWalletSwapTransactionHistory(ctx context.Context, command researchapp.CompleteWalletSwapTransactionHistoryCommand) (bool, error) {
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
		return false, fmt.Errorf("wallet swap transaction history schedule was not found")
	}
	if err = tx.Commit(ctx); err != nil {
		return false, err
	}
	return true, nil
}
