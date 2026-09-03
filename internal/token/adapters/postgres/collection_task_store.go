package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	tokensqlc "github.com/useryege/athena/internal/token/adapters/postgres/sqlc"
	"github.com/useryege/athena/internal/token/collection"
	collectionapp "github.com/useryege/athena/internal/token/collection/application"
	"github.com/useryege/athena/internal/token/shared"
)

var (
	_ collectionapp.TaskRepository = (*CollectionRepository)(nil)
	_ collectionapp.ReadRepository = (*CollectionRepository)(nil)
)

func (repository *CollectionRepository) ClaimCollectionTask(
	ctx context.Context,
	dataType collection.DataType,
	chainIDs []int64,
	lease time.Duration,
) (*collection.TaskWithProject, error) {
	if len(chainIDs) == 0 {
		return nil, nil
	}
	leaseSeconds, err := positiveWholeSeconds("collection task lease", lease)
	if err != nil {
		return nil, err
	}
	queries, err := repository.querier()
	if err != nil {
		return nil, err
	}
	row, err := queries.ClaimProjectDataCollectionTask(ctx, tokensqlc.ClaimProjectDataCollectionTaskParams{
		LeaseSeconds: leaseSeconds,
		DataType:     string(dataType),
		ChainIds:     chainIDs,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("claim %s collection task: %w", dataType, err)
	}
	task, err := mapCollectionTask(row)
	if err != nil {
		return nil, fmt.Errorf("map claimed collection task: %w", err)
	}
	project, err := queries.GetProjectForDataCollectionTask(ctx, task.ID)
	if err != nil {
		return nil, fmt.Errorf("get project for collection task %d: %w", task.ID, err)
	}
	contextValue, err := mapCollectionProjectContext(project)
	if err != nil {
		return nil, fmt.Errorf("map project for collection task %d: %w", task.ID, err)
	}
	walletRows, err := queries.ListProjectRelatedWalletsByProject(ctx, task.ProjectID)
	if err != nil {
		return nil, fmt.Errorf("list related wallets for collection task %d: %w", task.ID, err)
	}
	seen := make(map[shared.Address]struct{}, len(walletRows))
	for _, walletRow := range walletRows {
		wallet := bytesToAddress(walletRow.Wallet)
		if wallet.IsZero() {
			continue
		}
		if _, exists := seen[wallet]; exists {
			continue
		}
		seen[wallet] = struct{}{}
		contextValue.RelatedWallets = append(contextValue.RelatedWallets, wallet)
	}
	return &collection.TaskWithProject{Task: task, Project: contextValue}, nil
}

func (repository *CollectionRepository) RenewCollectionTaskLease(
	ctx context.Context,
	taskID int64,
	claimGeneration int64,
	lease time.Duration,
) (bool, error) {
	leaseSeconds, err := positiveWholeSeconds("collection task lease", lease)
	if err != nil {
		return false, err
	}
	queries, err := repository.querier()
	if err != nil {
		return false, err
	}
	rows, err := queries.RenewProjectDataCollectionTaskLease(ctx, tokensqlc.RenewProjectDataCollectionTaskLeaseParams{
		LeaseSeconds:    leaseSeconds,
		ID:              taskID,
		ClaimGeneration: claimGeneration,
	})
	if err != nil {
		return false, fmt.Errorf("renew collection task %d lease: %w", taskID, err)
	}
	return rows == 1, nil
}

func (repository *CollectionRepository) RetryCollectionTask(
	ctx context.Context,
	command collectionapp.RetryCollectionTaskCommand,
) (bool, error) {
	if err := validateCollectionFailureTransition(command.Task, command.FailureCount); err != nil {
		return false, err
	}
	if command.FailureCount >= collectionapp.MaxFailureCount {
		return false, fmt.Errorf("collection task %d retry failure count must be below %d", command.Task.ID, collectionapp.MaxFailureCount)
	}
	if command.AvailableAt.IsZero() {
		return false, fmt.Errorf("collection task %d retry availability is required", command.Task.ID)
	}
	queries, err := repository.querier()
	if err != nil {
		return false, err
	}
	row, err := queries.RetryProjectDataCollectionTask(ctx, tokensqlc.RetryProjectDataCollectionTaskParams{
		AvailableAt:     nullableTime(command.AvailableAt.UTC()),
		LastError:       nullableText(command.LastError),
		ID:              command.Task.ID,
		ClaimGeneration: command.Task.ClaimGeneration,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("retry collection task %d: %w", command.Task.ID, err)
	}
	if row.ProjectID != command.Task.ProjectID || row.DataType != string(command.Task.DataType) || row.FailureCount != command.FailureCount {
		return false, fmt.Errorf("collection task %d retry transition returned inconsistent identity or failure count", command.Task.ID)
	}
	return true, nil
}

func (repository *CollectionRepository) GetCollectionTask(ctx context.Context, id int64) (*collection.TaskDetail, error) {
	queries, err := repository.querier()
	if err != nil {
		return nil, err
	}
	row, err := queries.GetProjectDataCollectionTaskWithResult(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get collection task %d: %w", id, err)
	}
	task, err := mapCollectionTask(tokensqlc.ProjectDataCollectionTask{
		ID: row.ID, ProjectID: row.ProjectID, DataType: row.DataType, Status: row.Status,
		FailureCount: row.FailureCount, AvailableAt: row.AvailableAt, ClaimGeneration: row.ClaimGeneration,
		LockedAt: row.LockedAt, LeaseExpiresAt: row.LeaseExpiresAt, LastError: row.LastError,
		FinishedAt: row.FinishedAt, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	})
	if err != nil {
		return nil, fmt.Errorf("map collection task %d: %w", id, err)
	}
	detail := &collection.TaskDetail{Task: task}
	if row.ResultSchemaVersion.Valid {
		result, mapErr := mapCollectionResult(tokensqlc.ProjectDataCollectionResult{
			TaskID: row.ID, ProjectID: row.ProjectID, DataType: row.DataType,
			SchemaVersion: row.ResultSchemaVersion.Int32, Payload: row.ResultPayload,
			ContentHash: row.ResultContentHash, BlockNumber: row.ResultBlockNumber,
			CollectedAt: row.ResultCollectedAt, CreatedAt: row.ResultCreatedAt,
		})
		if mapErr != nil {
			return nil, fmt.Errorf("map result for collection task %d: %w", id, mapErr)
		}
		detail.Result = &result
	} else if row.ResultPayload != nil || row.ResultContentHash != nil || row.ResultBlockNumber.Valid || row.ResultCollectedAt.Valid || row.ResultCreatedAt.Valid {
		return nil, fmt.Errorf("collection task %d has a partially populated result", id)
	}
	return detail, nil
}

func (repository *CollectionRepository) ListCollectionTasks(
	ctx context.Context,
	projectID int64,
	dataType, status string,
	page, pageSize int32,
) (*collection.TaskPage, error) {
	queries, err := repository.querier()
	if err != nil {
		return nil, err
	}
	page, pageSize, offset := normalizePage(page, pageSize)
	filter := tokensqlc.CountProjectDataCollectionTasksParams{
		ProjectID: projectID,
		DataType:  dataType,
		Status:    status,
	}
	total, err := queries.CountProjectDataCollectionTasks(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("count collection tasks: %w", err)
	}
	rows, err := queries.ListProjectDataCollectionTasks(ctx, tokensqlc.ListProjectDataCollectionTasksParams{
		ProjectID: filter.ProjectID,
		DataType:  filter.DataType,
		Status:    filter.Status,
		Offset:    offset,
		Limit:     pageSize,
	})
	if err != nil {
		return nil, fmt.Errorf("list collection tasks: %w", err)
	}
	items := make([]collection.Task, 0, len(rows))
	for _, row := range rows {
		item, mapErr := mapCollectionTask(row)
		if mapErr != nil {
			return nil, fmt.Errorf("map collection task %d: %w", row.ID, mapErr)
		}
		items = append(items, item)
	}
	return &collection.TaskPage{Items: items, Total: total, Page: page, PageSize: pageSize}, nil
}

func mapCollectionTask(row tokensqlc.ProjectDataCollectionTask) (collection.Task, error) {
	dataType, ok := collection.ParseDataType(row.DataType)
	if !ok {
		return collection.Task{}, fmt.Errorf("unsupported data type %q", row.DataType)
	}
	status := collection.TaskStatus(row.Status)
	switch status {
	case collection.TaskStatusPending, collection.TaskStatusRunning, collection.TaskStatusSucceeded, collection.TaskStatusFailed:
	default:
		return collection.Task{}, fmt.Errorf("unsupported status %q", row.Status)
	}
	if row.FailureCount < 0 || row.FailureCount > collectionapp.MaxFailureCount {
		return collection.Task{}, fmt.Errorf("failure count %d is outside 0..%d", row.FailureCount, collectionapp.MaxFailureCount)
	}
	return collection.Task{
		ID: row.ID, ProjectID: row.ProjectID, DataType: dataType, Status: status,
		FailureCount: row.FailureCount, AvailableAt: timeValue(row.AvailableAt),
		ClaimGeneration: row.ClaimGeneration, LockedAt: timeValue(row.LockedAt),
		LeaseExpiresAt: timeValue(row.LeaseExpiresAt), LastError: textValue(row.LastError),
		FinishedAt: timeValue(row.FinishedAt), CreatedAt: timeValue(row.CreatedAt), UpdatedAt: timeValue(row.UpdatedAt),
	}, nil
}

func mapCollectionProjectContext(row tokensqlc.Project) (collection.ProjectContext, error) {
	blockNumber, err := int64ToUint64("deployment_block_number", row.BlockNumber)
	if err != nil {
		return collection.ProjectContext{}, err
	}
	return collection.ProjectContext{
		ID: row.ID, ChainID: row.ChainID, Contract: bytesToAddress(row.Contract),
		CodeHash: bytesToHash(row.CodeHash), WethPair: bytesToAddress(row.WethPair),
		UsdtPair: bytesToAddress(row.UsdtPair), DeploymentBlockNumber: blockNumber,
	}, nil
}

func validateCollectionFailureTransition(task collection.Task, failureCount int32) error {
	if task.ID <= 0 || task.ProjectID <= 0 {
		return fmt.Errorf("collection task identity must be positive")
	}
	if task.Status != collection.TaskStatusRunning {
		return fmt.Errorf("collection task %d must be running", task.ID)
	}
	if task.ClaimGeneration <= 0 {
		return fmt.Errorf("collection task %d claim generation must be positive", task.ID)
	}
	if failureCount != task.FailureCount+1 {
		return fmt.Errorf("collection task %d failure count must advance from %d to %d", task.ID, task.FailureCount, task.FailureCount+1)
	}
	return nil
}

func positiveWholeSeconds(field string, value time.Duration) (int64, error) {
	if value < time.Second || value%time.Second != 0 {
		return 0, fmt.Errorf("%s must be a positive whole number of seconds", field)
	}
	return int64(value / time.Second), nil
}
