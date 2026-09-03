package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	tokensqlc "github.com/useryege/athena/internal/token/adapters/postgres/sqlc"
	"github.com/useryege/athena/internal/token/collection"
	collectionapp "github.com/useryege/athena/internal/token/collection/application"
)

func (repository *CollectionRepository) CommitCollection(
	ctx context.Context,
	command collectionapp.CommitCollectionCommand,
) (collectionapp.CommitCollectionResult, error) {
	if repository == nil || repository.pool == nil {
		return collectionapp.CommitCollectionResult{}, fmt.Errorf("token collection repository is not configured")
	}
	if err := validateCollectionCommit(command); err != nil {
		return collectionapp.CommitCollectionResult{}, err
	}
	tx, err := repository.pool.Begin(ctx)
	if err != nil {
		return collectionapp.CommitCollectionResult{}, fmt.Errorf("begin collection completion: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := tokensqlc.New(tx)

	if _, err := queries.LockProjectForCollectionCompletion(ctx, command.Task.ProjectID); err != nil {
		return collectionapp.CommitCollectionResult{}, fmt.Errorf("lock project %d for collection completion: %w", command.Task.ProjectID, err)
	}
	locked, err := queries.LockProjectDataCollectionTaskForCompletion(ctx, tokensqlc.LockProjectDataCollectionTaskForCompletionParams{
		ID: command.Task.ID, ProjectID: command.Task.ProjectID, ClaimGeneration: command.Task.ClaimGeneration,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return collectionapp.CommitCollectionResult{}, nil
	}
	if err != nil {
		return collectionapp.CommitCollectionResult{}, fmt.Errorf("lock collection task %d for completion: %w", command.Task.ID, err)
	}
	if locked.DataType != string(command.Task.DataType) || locked.FailureCount != command.Task.FailureCount {
		return collectionapp.CommitCollectionResult{}, fmt.Errorf("collection task %d changed while completing", command.Task.ID)
	}
	if locked.Status == string(collection.TaskStatusSucceeded) {
		stored, getErr := queries.GetProjectDataCollectionResultByTaskID(ctx, command.Task.ID)
		if getErr != nil {
			return collectionapp.CommitCollectionResult{}, fmt.Errorf("get succeeded collection task %d result: %w", command.Task.ID, getErr)
		}
		if stored.ProjectID != command.Task.ProjectID || stored.DataType != string(command.Task.DataType) ||
			stored.SchemaVersion != command.SchemaVersion || bytesToHash(stored.ContentHash) != command.ContentHash {
			return collectionapp.CommitCollectionResult{}, fmt.Errorf("collection task %d result conflicts with immutable stored evidence", command.Task.ID)
		}
		if err := tx.Commit(ctx); err != nil {
			return collectionapp.CommitCollectionResult{}, fmt.Errorf("commit collection task %d idempotent completion: %w", command.Task.ID, err)
		}
		return collectionapp.CommitCollectionResult{Applied: true}, nil
	}

	if command.CodeSource != nil {
		if _, err := queries.UpdateContractCodeSource(ctx, tokensqlc.UpdateContractCodeSourceParams{
			CodeHash: command.CodeSource.CodeHash.Bytes(), SourceCode: nullableText(command.CodeSource.SourceCode),
			SourceCodeFetchedAt: nullableTime(command.CollectedAt.UTC()),
		}); err != nil {
			return collectionapp.CommitCollectionResult{}, fmt.Errorf("persist contract source for collection task %d: %w", command.Task.ID, err)
		}
	}
	if err := insertWalletNormalTransactions(ctx, queries, command.Task.ProjectID, command.CollectedAt.UTC(), command.NormalTransactions); err != nil {
		return collectionapp.CommitCollectionResult{}, err
	}

	blockNumber, err := nullableUint64(command.BlockNumber)
	if err != nil {
		return collectionapp.CommitCollectionResult{}, fmt.Errorf("map collection task %d block number: %w", command.Task.ID, err)
	}
	resultRow, err := queries.InsertProjectDataCollectionResult(ctx, tokensqlc.InsertProjectDataCollectionResultParams{
		TaskID: command.Task.ID, ProjectID: command.Task.ProjectID, DataType: string(command.Task.DataType),
		SchemaVersion: command.SchemaVersion, Payload: command.Payload, ContentHash: command.ContentHash.Bytes(),
		BlockNumber: blockNumber, CollectedAt: nullableTime(command.CollectedAt.UTC()),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return collectionapp.CommitCollectionResult{}, fmt.Errorf("collection task %d result conflicts with immutable stored evidence", command.Task.ID)
	}
	if err != nil {
		return collectionapp.CommitCollectionResult{}, fmt.Errorf("insert collection task %d result: %w", command.Task.ID, err)
	}
	if resultRow.TaskID != command.Task.ID || resultRow.ProjectID != command.Task.ProjectID ||
		resultRow.DataType != string(command.Task.DataType) || bytesToHash(resultRow.ContentHash) != command.ContentHash {
		return collectionapp.CommitCollectionResult{}, fmt.Errorf("collection task %d result returned inconsistent identity", command.Task.ID)
	}

	if _, err := queries.MarkProjectDataCollectionTaskSucceeded(ctx, tokensqlc.MarkProjectDataCollectionTaskSucceededParams{
		FinishedAt: nullableTime(command.CollectedAt.UTC()), ID: command.Task.ID, ClaimGeneration: command.Task.ClaimGeneration,
	}); errors.Is(err, pgx.ErrNoRows) {
		return collectionapp.CommitCollectionResult{}, nil
	} else if err != nil {
		return collectionapp.CommitCollectionResult{}, fmt.Errorf("mark collection task %d succeeded: %w", command.Task.ID, err)
	}
	profileTaskCreated, err := enqueueProfileBuildTaskIfReady(ctx, queries, command.Task.ProjectID)
	if err != nil {
		return collectionapp.CommitCollectionResult{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return collectionapp.CommitCollectionResult{}, fmt.Errorf("commit collection task %d completion: %w", command.Task.ID, err)
	}
	return collectionapp.CommitCollectionResult{Applied: true, ProfileBuildTaskCreated: profileTaskCreated}, nil
}

func (repository *CollectionRepository) FailCollectionTask(
	ctx context.Context,
	command collectionapp.FailCollectionTaskCommand,
) (bool, error) {
	if repository == nil || repository.pool == nil {
		return false, fmt.Errorf("token collection repository is not configured")
	}
	if err := validateCollectionFailureTransition(command.Task, command.FailureCount); err != nil {
		return false, err
	}
	if command.FailureCount != collectionapp.MaxFailureCount {
		return false, fmt.Errorf("collection task %d terminal failure count must be %d", command.Task.ID, collectionapp.MaxFailureCount)
	}
	if command.FailedAt.IsZero() {
		return false, fmt.Errorf("collection task %d failure time is required", command.Task.ID)
	}
	tx, err := repository.pool.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("begin collection failure: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := tokensqlc.New(tx)

	if _, err := queries.LockProjectForCollectionCompletion(ctx, command.Task.ProjectID); err != nil {
		return false, fmt.Errorf("lock project %d for collection failure: %w", command.Task.ProjectID, err)
	}
	locked, err := queries.LockProjectDataCollectionTaskForCompletion(ctx, tokensqlc.LockProjectDataCollectionTaskForCompletionParams{
		ID: command.Task.ID, ProjectID: command.Task.ProjectID, ClaimGeneration: command.Task.ClaimGeneration,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("lock collection task %d for failure: %w", command.Task.ID, err)
	}
	if locked.DataType != string(command.Task.DataType) || locked.FailureCount != command.Task.FailureCount {
		return false, fmt.Errorf("collection task %d changed while failing", command.Task.ID)
	}
	if locked.Status != string(collection.TaskStatusRunning) {
		return false, nil
	}
	row, err := queries.FailProjectDataCollectionTask(ctx, tokensqlc.FailProjectDataCollectionTaskParams{
		LastError: nullableText(command.LastError), FinishedAt: nullableTime(command.FailedAt.UTC()),
		ID: command.Task.ID, ClaimGeneration: command.Task.ClaimGeneration,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("fail collection task %d: %w", command.Task.ID, err)
	}
	if row.FailureCount != command.FailureCount || row.Status != string(collection.TaskStatusFailed) {
		return false, fmt.Errorf("collection task %d terminal failure returned inconsistent state", command.Task.ID)
	}
	if _, err := enqueueProfileBuildTaskIfReady(ctx, queries, command.Task.ProjectID); err != nil {
		return false, err
	}
	if err := tx.Commit(ctx); err != nil {
		return false, fmt.Errorf("commit collection task %d failure: %w", command.Task.ID, err)
	}
	return true, nil
}

func validateCollectionCommit(command collectionapp.CommitCollectionCommand) error {
	if command.Task.ID <= 0 || command.Task.ProjectID <= 0 {
		return fmt.Errorf("collection task identity must be positive")
	}
	if command.Task.Status != collection.TaskStatusRunning || command.Task.ClaimGeneration <= 0 {
		return fmt.Errorf("collection task %d must have an active claim", command.Task.ID)
	}
	if command.CollectedAt.IsZero() {
		return fmt.Errorf("collection task %d collection time is required", command.Task.ID)
	}
	if command.SchemaVersion <= 0 {
		return fmt.Errorf("collection task %d schema version must be positive", command.Task.ID)
	}
	canonical, contentHash, err := collection.NormalizeResult(command.SchemaVersion, command.Payload)
	if err != nil {
		return fmt.Errorf("normalize collection task %d result: %w", command.Task.ID, err)
	}
	if string(canonical) != string(command.Payload) || contentHash != command.ContentHash {
		return fmt.Errorf("collection task %d result is not canonical or has a mismatched content hash", command.Task.ID)
	}
	if command.CodeSource != nil {
		if command.Task.DataType != collection.DataTypeContractCodeSource {
			return fmt.Errorf("collection task %d includes contract source for data type %s", command.Task.ID, command.Task.DataType)
		}
		if command.CodeSource.CodeHash.IsZero() {
			return fmt.Errorf("collection task %d contract source code hash is required", command.Task.ID)
		}
	}
	if len(command.NormalTransactions) > 0 && command.Task.DataType != collection.DataTypeWalletNormalTransactions {
		return fmt.Errorf("collection task %d includes normal transactions for data type %s", command.Task.ID, command.Task.DataType)
	}
	return nil
}

func enqueueProfileBuildTaskIfReady(ctx context.Context, queries *tokensqlc.Queries, projectID int64) (bool, error) {
	barrier, err := queries.GetProjectCollectionBarrierState(ctx, projectID)
	if err != nil {
		return false, fmt.Errorf("get project %d collection barrier: %w", projectID, err)
	}
	expected := int64(len(collection.AllDataTypes()))
	if barrier.TaskCount != expected {
		return false, fmt.Errorf("project %d collection barrier has %d tasks, expected %d", projectID, barrier.TaskCount, expected)
	}
	if barrier.TerminalCount != expected {
		return false, nil
	}
	if barrier.SucceededCount+barrier.FailedCount != expected {
		return false, fmt.Errorf("project %d collection barrier terminal counts are inconsistent", projectID)
	}
	if _, err := queries.GetProjectProfileBuildTask(ctx, projectID); err == nil {
		return false, nil
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return false, fmt.Errorf("get project %d profile build task: %w", projectID, err)
	}
	if err := queries.EnqueueProjectProfileBuildTask(ctx, projectID); err != nil {
		return false, fmt.Errorf("enqueue project %d profile build task: %w", projectID, err)
	}
	if _, err := queries.GetProjectProfileBuildTask(ctx, projectID); err != nil {
		return false, fmt.Errorf("verify project %d profile build task: %w", projectID, err)
	}
	return true, nil
}
