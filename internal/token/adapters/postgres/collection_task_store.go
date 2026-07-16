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
)

const collectionTaskLease = 90 * time.Second

func (repository *CollectionRepository) ClaimCollectionTasks(ctx context.Context, dataType research.DataCollectionType, chainIDs []int64, limit int32) ([]research.ProjectDataCollectionTaskWithProject, error) {
	if len(chainIDs) == 0 {
		return nil, nil
	}
	queries, err := repository.querier()
	if err != nil {
		return nil, err
	}
	rows, err := queries.ClaimProjectDataCollectionTasks(ctx, tokensqlc.ClaimProjectDataCollectionTasksParams{LeaseSeconds: int64(collectionTaskLease / time.Second), DataType: string(dataType), ChainIds: chainIDs, Limit: limit})
	if err != nil {
		return nil, fmt.Errorf("claim collection tasks: %w", err)
	}
	result := make([]research.ProjectDataCollectionTaskWithProject, 0, len(rows))
	for _, row := range rows {
		projectRow, err := queries.GetProjectForDataCollectionTask(ctx, row.ID)
		if err != nil {
			return nil, err
		}
		project, err := mapProject(projectRow)
		if err != nil {
			return nil, err
		}
		schedule, err := queries.GetProjectDataCollectionSchedule(ctx, tokensqlc.GetProjectDataCollectionScheduleParams{ProjectID: row.ProjectID, DataType: row.DataType})
		if err != nil {
			return nil, err
		}
		walletRows, err := queries.ListProjectRelatedWalletsByProject(ctx, row.ProjectID)
		if err != nil {
			return nil, err
		}
		contextValue := research.ProjectCollectionContext{ID: project.ID, ChainID: project.ChainID, Contract: project.Contract, CreationBlockNumber: project.BlockNumber, CreationTransactionIndex: project.TxIndex, CodeHash: project.CodeHash, WethPair: project.WethPair, UsdtPair: project.UsdtPair, RefreshInterval: time.Duration(schedule.RefreshIntervalSeconds) * time.Second}
		for _, wallet := range walletRows {
			contextValue.RelatedWallets = append(contextValue.RelatedWallets, bytesToAddress(wallet.Wallet))
		}
		result = append(result, research.ProjectDataCollectionTaskWithProject{Task: mapProjectDataCollectionTask(row), Project: contextValue})
	}
	return result, nil
}

func (repository *ResearchReadRepository) GetProjectDataCollectionTask(ctx context.Context, id int64) (*research.ProjectDataCollectionTask, error) {
	queries, err := repository.querier()
	if err != nil {
		return nil, err
	}
	row, err := queries.GetProjectDataCollectionTask(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	value := mapProjectDataCollectionTask(row)
	return &value, nil
}

func (repository *ResearchReadRepository) ListProjectDataCollectionTasks(ctx context.Context, projectID int64, dataType, status string, page, pageSize int32) (*research.CollectionTaskPage, error) {
	queries, err := repository.querier()
	if err != nil {
		return nil, err
	}
	page, pageSize, offset := normalizePage(page, pageSize)
	params := tokensqlc.CountProjectDataCollectionTasksParams{ProjectID: projectID, DataType: dataType, Status: status}
	total, err := queries.CountProjectDataCollectionTasks(ctx, params)
	if err != nil {
		return nil, err
	}
	rows, err := queries.ListProjectDataCollectionTasks(ctx, tokensqlc.ListProjectDataCollectionTasksParams{ProjectID: projectID, DataType: dataType, Status: status, Offset: offset, Limit: pageSize})
	if err != nil {
		return nil, err
	}
	items := make([]research.ProjectDataCollectionTask, 0, len(rows))
	for _, row := range rows {
		items = append(items, mapProjectDataCollectionTask(row))
	}
	return &research.CollectionTaskPage{Items: items, Total: total, Page: page, PageSize: pageSize}, nil
}

func (repository *CollectionRepository) RetryCollectionTask(ctx context.Context, command researchapp.RetryCollectionTaskCommand) error {
	queries, err := repository.querier()
	if err != nil {
		return err
	}
	_, err = queries.RetryProjectDataCollectionTask(ctx, tokensqlc.RetryProjectDataCollectionTaskParams{AvailableAt: nullableTime(command.AvailableAt), LastError: nullableText(command.LastError), ID: command.Task.ID})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	return err
}

func (repository *CollectionRepository) FailCollectionTask(ctx context.Context, command researchapp.FailCollectionTaskCommand) error {
	if repository == nil || repository.pool == nil {
		return fmt.Errorf("token collection repository is not configured")
	}
	tx, err := repository.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := tokensqlc.New(tx)
	rows, err := queries.FailProjectDataCollectionTask(ctx, tokensqlc.FailProjectDataCollectionTaskParams{LastError: nullableText(command.LastError), ID: command.Task.ID})
	if err != nil || rows == 0 {
		return err
	}
	if err = markScheduleFailed(ctx, queries, command.Task.ProjectID, command.Task.DataType, command.LastError, command.NextRunAt); err != nil {
		return err
	}
	if err = tx.Commit(ctx); err != nil {
		return err
	}
	return nil
}
