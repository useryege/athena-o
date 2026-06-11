package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/jackc/pgx/v5"
	tokensqlc "github.com/useryege/athena/internal/token/store/sqlc"
)

const (
	ProjectReportEvaluationStatusNotStarted = "not_started"
	ProjectReportEvaluationStatusPending    = "pending"
	ProjectReportEvaluationStatusSucceeded  = "succeeded"
	ProjectReportEvaluationStatusFailed     = "failed"
)

func (s *SQLStore) ListProjectReportsPage(ctx context.Context, chainID, projectID int64, contract common.Address, evaluationStatus string, page, pageSize int32) (*ProjectReportPage, error) {
	q, err := s.querier()
	if err != nil {
		return nil, err
	}
	page, pageSize, offset := normalizePage(page, pageSize)
	params := tokensqlc.CountProjectReportsParams{
		ChainID:          chainID,
		ProjectID:        projectID,
		Contract:         optionalAddressBytes(contract),
		EvaluationStatus: evaluationStatus,
	}
	total, err := q.CountProjectReports(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("count project reports: %w", err)
	}
	rows, err := q.ListProjectReports(ctx, tokensqlc.ListProjectReportsParams{
		ChainID:          params.ChainID,
		ProjectID:        params.ProjectID,
		Contract:         params.Contract,
		EvaluationStatus: params.EvaluationStatus,
		Offset:           offset,
		Limit:            pageSize,
	})
	if err != nil {
		return nil, fmt.Errorf("list project reports: %w", err)
	}
	items, err := mapProjectReportListItems(rows)
	if err != nil {
		return nil, fmt.Errorf("map project reports: %w", err)
	}
	return &ProjectReportPage{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func (s *SQLStore) ListDueProjectReportEvaluationTasks(ctx context.Context, limit int32) ([]ProjectReportEvaluationTask, error) {
	q, err := s.querier()
	if err != nil {
		return nil, err
	}
	rows, err := q.ListDueProjectReportEvaluationTasks(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("list due project report evaluation tasks: %w", err)
	}
	items := make([]ProjectReportEvaluationTask, 0, len(rows))
	for _, row := range rows {
		items = append(items, ProjectReportEvaluationTask{
			ProjectID:       row.ProjectID,
			Status:          ProjectReportEvaluationStatusPending,
			Revision:        row.Revision,
			Attempts:        row.Attempts,
			SourceUpdatedAt: time.UnixMicro(row.SourceUpdatedAtUnixMicro).UTC(),
			HasChainState:   row.ChainStateProjectID != 0,
			ChainState:      row.ChainState,
		})
	}
	return items, nil
}

func (s *SQLStore) CompleteProjectReportEvaluation(ctx context.Context, task ProjectReportEvaluationTask, report ProjectReport) (*ProjectReport, bool, error) {
	if s == nil || s.pool == nil {
		return nil, false, fmt.Errorf("token postgres database is not configured")
	}
	wethLastSwapTimestamp, err := nullableUint64(report.WethPairLastSwapTimestamp)
	if err != nil {
		return nil, false, fmt.Errorf("convert weth pair last swap timestamp: %w", err)
	}
	usdtLastSwapTimestamp, err := nullableUint64(report.UsdtPairLastSwapTimestamp)
	if err != nil {
		return nil, false, fmt.Errorf("convert usdt pair last swap timestamp: %w", err)
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, false, fmt.Errorf("begin project report evaluation transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()
	q := tokensqlc.New(tx)
	if _, err := q.LockPendingProjectReportEvaluationTask(ctx, tokensqlc.LockPendingProjectReportEvaluationTaskParams{
		ProjectID: task.ProjectID,
		Revision:  task.Revision,
	}); errors.Is(err, pgx.ErrNoRows) {
		return nil, false, nil
	} else if err != nil {
		return nil, false, fmt.Errorf("lock project report evaluation task: %w", err)
	}
	row, err := q.UpdateProjectReportEvaluation(ctx, tokensqlc.UpdateProjectReportEvaluationParams{
		WethPairIsCreated:         nullableBool(report.WethPairIsCreated),
		WethPairIsRemoveLiquidity: nullableBool(report.WethPairIsRemoveLiquidity),
		WethPairIsMint:            nullableBool(report.WethPairIsMint),
		WethPairQuoteUsdtValueInt: nullableNumericFromBigInt(report.WethPairQuoteUsdtValueInt),
		WethPairLastSwapTimestamp: wethLastSwapTimestamp,
		UsdtPairIsCreated:         nullableBool(report.UsdtPairIsCreated),
		UsdtPairIsRemoveLiquidity: nullableBool(report.UsdtPairIsRemoveLiquidity),
		UsdtPairIsMint:            nullableBool(report.UsdtPairIsMint),
		UsdtPairQuoteUsdtValueInt: nullableNumericFromBigInt(report.UsdtPairQuoteUsdtValueInt),
		UsdtPairLastSwapTimestamp: usdtLastSwapTimestamp,
		SourceUpdatedAt:           nullableTime(report.SourceUpdatedAt),
		EvaluatedAt:               nullableTime(report.EvaluatedAt),
		ProjectID:                 report.ProjectID,
	})
	if err != nil {
		return nil, false, fmt.Errorf("update project report evaluation: %w", err)
	}
	if err := q.MarkProjectReportEvaluationTaskSucceeded(ctx, tokensqlc.MarkProjectReportEvaluationTaskSucceededParams{
		ProjectID: task.ProjectID,
		Revision:  task.Revision,
	}); err != nil {
		return nil, false, fmt.Errorf("mark project report evaluation task succeeded: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, false, fmt.Errorf("commit project report evaluation transaction: %w", err)
	}
	mapped, err := mapProjectReport(row)
	if err != nil {
		return nil, false, fmt.Errorf("map project report: %w", err)
	}
	return mapped, true, nil
}

func (s *SQLStore) MarkProjectReportEvaluationTaskFailed(ctx context.Context, projectID, revision int64, lastError string) (*ProjectReportEvaluationTask, bool, error) {
	q, err := s.querier()
	if err != nil {
		return nil, false, err
	}
	row, err := q.MarkProjectReportEvaluationTaskFailed(ctx, tokensqlc.MarkProjectReportEvaluationTaskFailedParams{
		LastError: nullableText(lastError),
		ProjectID: projectID,
		Revision:  revision,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("mark project report evaluation task failed: %w", err)
	}
	return mapProjectReportEvaluationTask(row), true, nil
}

func enqueueProjectReportEvaluationTask(ctx context.Context, q *tokensqlc.Queries, projectID int64) error {
	if _, err := q.EnqueueProjectReportEvaluationTask(ctx, projectID); err != nil {
		return fmt.Errorf("enqueue project report evaluation task: %w", err)
	}
	return nil
}
