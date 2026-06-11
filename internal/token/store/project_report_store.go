package store

import (
	"context"
	"fmt"
	"time"

	tokensqlc "github.com/useryege/athena/internal/token/store/sqlc"
)

func (s *SQLStore) ListProjectReportsDueForEvaluation(ctx context.Context, limit int32) ([]ProjectReportEvaluationCandidate, error) {
	q, err := s.querier()
	if err != nil {
		return nil, err
	}
	rows, err := q.ListProjectReportsDueForEvaluation(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("list project reports due for evaluation: %w", err)
	}
	items := make([]ProjectReportEvaluationCandidate, 0, len(rows))
	for _, row := range rows {
		items = append(items, ProjectReportEvaluationCandidate{
			ProjectID:       row.ProjectID,
			IsComplete:      row.IsComplete,
			SourceUpdatedAt: time.UnixMicro(row.SourceUpdatedAtUnixMicro).UTC(),
			HasChainState:   row.ChainStateSucceeded && row.ChainStateProjectID != 0,
			ChainState:      row.ChainState,
		})
	}
	return items, nil
}

func (s *SQLStore) UpdateProjectReportEvaluation(ctx context.Context, report ProjectReport) (*ProjectReport, error) {
	q, err := s.querier()
	if err != nil {
		return nil, err
	}
	wethLastSwapTimestamp, err := nullableUint64(report.WethPairLastSwapTimestamp)
	if err != nil {
		return nil, fmt.Errorf("convert weth pair last swap timestamp: %w", err)
	}
	usdtLastSwapTimestamp, err := nullableUint64(report.UsdtPairLastSwapTimestamp)
	if err != nil {
		return nil, fmt.Errorf("convert usdt pair last swap timestamp: %w", err)
	}
	row, err := q.UpdateProjectReportEvaluation(ctx, tokensqlc.UpdateProjectReportEvaluationParams{
		IsComplete:                report.IsComplete,
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
		return nil, fmt.Errorf("update project report evaluation: %w", err)
	}
	mapped, err := mapProjectReport(row)
	if err != nil {
		return nil, fmt.Errorf("map project report: %w", err)
	}
	return mapped, nil
}
