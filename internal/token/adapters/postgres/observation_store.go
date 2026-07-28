package postgres

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	tokensqlc "github.com/useryege/athena/internal/token/adapters/postgres/sqlc"
	researchapp "github.com/useryege/athena/internal/token/research/application"
)

func (repository *CollectionRepository) CommitCollection(ctx context.Context, command researchapp.CommitCollectionCommand) (researchapp.CommitCollectionResult, error) {
	if repository == nil || repository.pool == nil {
		return researchapp.CommitCollectionResult{}, fmt.Errorf("token collection repository is not configured")
	}
	tx, err := repository.pool.Begin(ctx)
	if err != nil {
		return researchapp.CommitCollectionResult{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := tokensqlc.New(tx)
	if command.CodeSource != nil && command.CodeSource.SourceCode != "" {
		if _, err = queries.UpdateContractCodeSource(ctx, tokensqlc.UpdateContractCodeSourceParams{CodeHash: command.CodeSource.CodeHash.Bytes(), SourceCode: nullableText(command.CodeSource.SourceCode), SourceCodeFetchedAt: nullableTime(command.CheckedAt)}); err != nil {
			return researchapp.CommitCollectionResult{}, err
		}
	}
	result := researchapp.CommitCollectionResult{}
	if command.RecordObservation {
		result.ObservationID, result.Changed, err = persistObservation(ctx, queries, command)
		if err != nil {
			return researchapp.CommitCollectionResult{}, err
		}
	}
	if err = insertWalletNormalTransactions(ctx, queries, command.Task.ProjectID, command.CheckedAt, command.NormalTransactions); err != nil {
		return researchapp.CommitCollectionResult{}, err
	}
	rows, err := queries.MarkProjectDataCollectionTaskSucceeded(ctx, command.Task.ID)
	if err != nil {
		return researchapp.CommitCollectionResult{}, err
	}
	if rows == 0 {
		return researchapp.CommitCollectionResult{}, nil
	}
	if _, err = queries.CompleteProjectDataCollectionSchedule(ctx, tokensqlc.CompleteProjectDataCollectionScheduleParams{LastCheckedAt: nullableTime(command.CheckedAt), ProjectID: command.Task.ProjectID, DataType: string(command.Task.DataType)}); err != nil {
		return researchapp.CommitCollectionResult{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return researchapp.CommitCollectionResult{}, err
	}
	result.Applied = true
	return result, nil
}

func persistObservation(ctx context.Context, queries *tokensqlc.Queries, command researchapp.CommitCollectionCommand) (int64, bool, error) {
	current, err := queries.GetCurrentProjectObservation(ctx, tokensqlc.GetCurrentProjectObservationParams{ProjectID: command.Task.ProjectID, DataType: string(command.Task.DataType)})
	if err == nil && current.SchemaVersion == command.SchemaVersion && bytes.Equal(current.ContentHash, command.ContentHash.Bytes()) {
		if _, err = queries.TouchCurrentProjectObservation(ctx, tokensqlc.TouchCurrentProjectObservationParams{LastCheckedAt: nullableTime(command.CheckedAt), ProjectID: command.Task.ProjectID, DataType: string(command.Task.DataType)}); err != nil {
			return 0, false, err
		}
		return current.ID, false, nil
	}
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return 0, false, err
	}
	blockNumber, err := nullableUint64(command.BlockNumber)
	if err != nil {
		return 0, false, err
	}
	row, err := queries.InsertProjectObservation(ctx, tokensqlc.InsertProjectObservationParams{ProjectID: command.Task.ProjectID, DataType: string(command.Task.DataType), SchemaVersion: command.SchemaVersion, ContentHash: command.ContentHash.Bytes(), Payload: command.Payload, BlockNumber: blockNumber, ObservedAt: nullableTime(command.CheckedAt)})
	if err != nil {
		return 0, false, err
	}
	if _, err = queries.UpsertCurrentProjectObservation(ctx, tokensqlc.UpsertCurrentProjectObservationParams{ProjectID: command.Task.ProjectID, DataType: string(command.Task.DataType), ObservationID: row.ID, LastCheckedAt: nullableTime(command.CheckedAt)}); err != nil {
		return 0, false, err
	}
	state, err := queries.IncrementProjectEvidenceRevision(ctx, command.Task.ProjectID)
	if err != nil {
		return 0, false, err
	}
	if _, err = queries.EnqueueProjectReportBuildTask(ctx, tokensqlc.EnqueueProjectReportBuildTaskParams{ProjectID: command.Task.ProjectID, EvidenceRevision: state.EvidenceRevision}); err != nil {
		return 0, false, err
	}
	return row.ID, true, nil
}
