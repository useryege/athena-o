package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	tokensqlc "github.com/useryege/athena/internal/token/adapters/postgres/sqlc"
	"github.com/useryege/athena/internal/token/collection"
	"github.com/useryege/athena/internal/token/projectview"
	"github.com/useryege/athena/internal/token/shared"
)

func (repository *ProjectViewRepository) GetProjectDetail(ctx context.Context, projectID int64) (*projectview.Detail, error) {
	if repository == nil || repository.pool == nil {
		return nil, fmt.Errorf("token project view repository is not configured")
	}
	tx, err := repository.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return nil, fmt.Errorf("begin project %d detail snapshot: %w", projectID, err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := tokensqlc.New(tx)

	projectRow, err := queries.GetProject(ctx, projectID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get project %d detail: %w", projectID, err)
	}
	project, err := mapProject(projectRow)
	if err != nil {
		return nil, fmt.Errorf("map project %d detail: %w", projectID, err)
	}
	detail := &projectview.Detail{Project: *project}

	profileRow, err := queries.GetProjectProfile(ctx, projectID)
	if err == nil {
		profileValue, mapErr := mapProjectProfile(profileRow)
		if mapErr != nil {
			return nil, fmt.Errorf("map project %d profile: %w", projectID, mapErr)
		}
		detail.Profile = &profileValue
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("get project %d profile: %w", projectID, err)
	}

	taskRows, err := queries.ListProjectDataCollectionTasksByProject(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("list project %d collection tasks: %w", projectID, err)
	}
	resultRows, err := queries.ListProjectDataCollectionResultsByProject(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("list project %d collection results: %w", projectID, err)
	}
	results := make(map[int64]collection.Result, len(resultRows))
	for _, row := range resultRows {
		result, mapErr := mapCollectionResult(row)
		if mapErr != nil {
			return nil, fmt.Errorf("map project %d collection result %d: %w", projectID, row.TaskID, mapErr)
		}
		results[row.TaskID] = result
	}
	detail.CollectionTasks = make([]collection.TaskDetail, 0, len(taskRows))
	for _, row := range taskRows {
		task, mapErr := mapCollectionTask(row)
		if mapErr != nil {
			return nil, fmt.Errorf("map project %d collection task %d: %w", projectID, row.ID, mapErr)
		}
		item := collection.TaskDetail{Task: task}
		if result, exists := results[task.ID]; exists {
			value := result
			item.Result = &value
			delete(results, task.ID)
		}
		detail.CollectionTasks = append(detail.CollectionTasks, item)
	}
	if len(results) != 0 {
		return nil, fmt.Errorf("project %d contains collection results without tasks", projectID)
	}

	relatedWalletRows, err := queries.ListProjectRelatedWalletsByProject(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("list project %d related wallets: %w", projectID, err)
	}
	detail.RelatedWallets = mapProjectRelatedWallets(relatedWalletRows)
	recipientRows, err := queries.ListProjectInitialRecipientsByProject(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("list project %d initial recipients: %w", projectID, err)
	}
	detail.InitialRecipients, err = mapProjectInitialRecipients(recipientRows)
	if err != nil {
		return nil, fmt.Errorf("map project %d initial recipients: %w", projectID, err)
	}
	detail.TransactionCount, err = queries.CountProjectWalletNormalTransactions(ctx, tokensqlc.CountProjectWalletNormalTransactionsParams{ProjectID: projectID})
	if err != nil {
		return nil, fmt.Errorf("count project %d wallet transactions: %w", projectID, err)
	}
	countRows, err := queries.CountProjectWalletNormalTransactionsByWallet(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("count project %d wallet transactions by wallet: %w", projectID, err)
	}
	detail.WalletTransactionCounts = make([]projectview.WalletTransactionCount, 0, len(countRows))
	for _, row := range countRows {
		detail.WalletTransactionCounts = append(detail.WalletTransactionCounts, projectview.WalletTransactionCount{
			Wallet: bytesToAddress(row.Wallet), TransactionCount: row.TransactionCount,
		})
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit project %d detail snapshot: %w", projectID, err)
	}
	return detail, nil
}

func (repository *ProjectViewRepository) ListProjectWalletNormalTransactionsPage(
	ctx context.Context,
	projectID int64,
	wallet shared.Address,
	receiptStatus string,
	methodID string,
	page, pageSize int32,
) (*projectview.WalletNormalTransactionPage, error) {
	queries, err := repository.querier()
	if err != nil {
		return nil, err
	}
	page, pageSize, offset := normalizePage(page, pageSize)
	receiptStatus = strings.TrimSpace(receiptStatus)
	methodID = strings.TrimSpace(methodID)
	params := tokensqlc.CountProjectWalletNormalTransactionsParams{
		ProjectID: projectID, Wallet: optionalAddressBytes(wallet), ReceiptStatus: receiptStatus, MethodID: methodID,
	}
	total, err := queries.CountProjectWalletNormalTransactions(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("count project wallet normal transactions: %w", err)
	}
	rows, err := queries.ListProjectWalletNormalTransactions(ctx, tokensqlc.ListProjectWalletNormalTransactionsParams{
		ProjectID: projectID, Wallet: params.Wallet, ReceiptStatus: receiptStatus, MethodID: methodID,
		Offset: offset, Limit: pageSize,
	})
	if err != nil {
		return nil, fmt.Errorf("list project wallet normal transactions: %w", err)
	}
	items := make([]collection.WalletNormalTransaction, 0, len(rows))
	for _, row := range rows {
		item, mapErr := mapWalletNormalTransaction(row)
		if mapErr != nil {
			return nil, fmt.Errorf("map project wallet normal transaction: %w", mapErr)
		}
		items = append(items, item)
	}
	return &projectview.WalletNormalTransactionPage{Items: items, Total: total, Page: page, PageSize: pageSize}, nil
}

func mapWalletNormalTransaction(row tokensqlc.ProjectWalletNormalTransaction) (collection.WalletNormalTransaction, error) {
	blockNumber, err := int64ToUint64("block_number", row.BlockNumber)
	if err != nil {
		return collection.WalletNormalTransaction{}, err
	}
	blockTimestamp, err := int64ToUint64("block_timestamp", row.BlockTimestamp)
	if err != nil {
		return collection.WalletNormalTransaction{}, err
	}
	transactionIndex, err := int64ToUint64("transaction_index", row.TransactionIndex)
	if err != nil {
		return collection.WalletNormalTransaction{}, err
	}
	nonce, err := int64ToUint64("nonce", row.Nonce)
	if err != nil {
		return collection.WalletNormalTransaction{}, err
	}
	gas, err := int64ToUint64("gas", row.Gas)
	if err != nil {
		return collection.WalletNormalTransaction{}, err
	}
	gasUsed, err := int64ToUint64("gas_used", row.GasUsed)
	if err != nil {
		return collection.WalletNormalTransaction{}, err
	}
	return collection.WalletNormalTransaction{
		Wallet: bytesToAddress(row.Wallet), TransactionHash: bytesToHash(row.TransactionHash),
		BlockNumber: blockNumber, BlockTimestamp: blockTimestamp, TransactionIndex: transactionIndex,
		Nonce: nonce, FromAddress: bytesToAddress(row.FromAddress), ToAddress: bytesToAddress(row.ToAddress),
		Value: bigIntFromNumeric(row.Value), Gas: gas, GasPrice: bigIntFromNumeric(row.GasPrice), GasUsed: gasUsed,
		Input: row.Input, MethodID: row.MethodID, FunctionName: row.FunctionName,
		ReceiptStatus: collection.NormalTransactionReceiptStatus(row.ReceiptStatus), IsError: row.IsError,
		CollectedAt: timeValue(row.CollectedAt),
	}, nil
}
