package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	tokensqlc "github.com/useryege/athena/internal/token/adapters/postgres/sqlc"
	"github.com/useryege/athena/internal/token/research"
	researchapp "github.com/useryege/athena/internal/token/research/application"
	"github.com/useryege/athena/internal/token/shared"
)

func (repository *CollectionRepository) ListPendingWalletFundingSourceHistoryWallets(ctx context.Context, projectID int64) ([]shared.Address, error) {
	queries, err := repository.querier()
	if err != nil {
		return nil, err
	}
	rows, err := queries.ListPendingProjectWalletFundingSourceHistoryWallets(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("list pending wallet funding source histories: %w", err)
	}
	wallets := make([]shared.Address, 0, len(rows))
	for _, row := range rows {
		wallets = append(wallets, bytesToAddress(row))
	}
	return wallets, nil
}

func (repository *CollectionRepository) SaveWalletFundingSourceHistory(ctx context.Context, command researchapp.SaveWalletFundingSourceHistoryCommand) (bool, error) {
	if repository == nil || repository.pool == nil {
		return false, fmt.Errorf("token collection repository is not configured")
	}
	if command.Wallet.IsZero() {
		return false, fmt.Errorf("wallet funding source history wallet is required")
	}
	if command.RequestedCount <= 0 || len(command.History.Transactions) > int(command.RequestedCount) {
		return false, fmt.Errorf("wallet funding source history count is invalid")
	}
	if command.FetchedAt.IsZero() {
		return false, fmt.Errorf("wallet funding source history fetched time is required")
	}
	anchorBlock, err := uint64ToInt64("anchor_block_number", command.Project.CreationBlockNumber)
	if err != nil {
		return false, err
	}
	anchorTransactionIndex, err := uint64ToInt64("anchor_transaction_index", command.Project.CreationTransactionIndex)
	if err != nil {
		return false, err
	}
	indexedThroughBlock, err := uint64ToInt64("indexed_through_block", command.History.IndexedThroughBlock)
	if err != nil {
		return false, err
	}
	indexedThroughTimestamp, err := uint64ToInt64("indexed_through_timestamp", command.History.IndexedThroughTimestamp)
	if err != nil {
		return false, err
	}
	params := make([]tokensqlc.InsertProjectWalletFundingSourceTransactionParams, 0, len(command.History.Transactions))
	for index, transaction := range command.History.Transactions {
		item, err := walletFundingSourceTransactionParams(command.Project.ID, command.Wallet, int32(index), transaction)
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
	_, err = queries.InsertProjectWalletFundingSourceHistory(ctx, tokensqlc.InsertProjectWalletFundingSourceHistoryParams{
		ProjectID: command.Project.ID, Wallet: command.Wallet.Bytes(), AnchorBlockNumber: anchorBlock,
		AnchorTransactionIndex: anchorTransactionIndex, RequestedTransactionCount: command.RequestedCount,
		CollectedTransactionCount: int32(len(command.History.Transactions)), IndexedThroughBlock: indexedThroughBlock,
		IndexedThroughTimestamp: indexedThroughTimestamp, FetchedAt: nullableTime(command.FetchedAt),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("insert wallet funding source history: %w", err)
	}
	for _, item := range params {
		if err = queries.InsertProjectWalletFundingSourceTransaction(ctx, item); err != nil {
			return false, fmt.Errorf("insert wallet funding source transaction rank=%d: %w", item.RankIndex, err)
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return false, err
	}
	return true, nil
}

func (repository *CollectionRepository) CompleteWalletFundingSourceHistory(ctx context.Context, command researchapp.CompleteWalletFundingSourceHistoryCommand) (bool, error) {
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
		return false, fmt.Errorf("wallet funding source history schedule was not found")
	}
	if err = tx.Commit(ctx); err != nil {
		return false, err
	}
	return true, nil
}

func walletFundingSourceTransactionParams(projectID int64, wallet shared.Address, rankIndex int32, transaction research.WalletFundingSourceTransaction) (tokensqlc.InsertProjectWalletFundingSourceTransactionParams, error) {
	blockNumber, err := uint64ToInt64("block_number", transaction.BlockNumber)
	if err != nil {
		return tokensqlc.InsertProjectWalletFundingSourceTransactionParams{}, err
	}
	blockTimestamp, err := uint64ToInt64("block_timestamp", transaction.BlockTimestamp)
	if err != nil {
		return tokensqlc.InsertProjectWalletFundingSourceTransactionParams{}, err
	}
	transactionIndex, err := uint64ToInt64("transaction_index", transaction.TransactionIndex)
	if err != nil {
		return tokensqlc.InsertProjectWalletFundingSourceTransactionParams{}, err
	}
	return tokensqlc.InsertProjectWalletFundingSourceTransactionParams{
		ProjectID: projectID, Wallet: wallet.Bytes(), RankIndex: rankIndex,
		BlockNumber: blockNumber, BlockHash: transaction.BlockHash.Bytes(), BlockTimestamp: blockTimestamp,
		TransactionHash: transaction.TransactionHash.Bytes(), TransactionIndex: transactionIndex,
		FromAddress: transaction.From.Bytes(), ToAddress: transaction.To.Bytes(), ValueWei: numericFromBigInt(transaction.ValueWei),
	}, nil
}
