package store

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/jackc/pgx/v5"
	tokensqlc "github.com/useryege/athena/internal/token/store/sqlc"
)

const researchTaskLease = 90 * time.Second

func (s *SQLStore) ListProjectResearchStatesPage(ctx context.Context, chainID, projectID int64, status string, page, pageSize int32) (*ProjectResearchStatePage, error) {
	q, e := s.querier()
	if e != nil {
		return nil, e
	}
	page, pageSize, offset := normalizePage(page, pageSize)
	p := tokensqlc.CountProjectResearchStatesParams{ChainID: chainID, ProjectID: projectID, Status: status}
	total, e := q.CountProjectResearchStates(ctx, p)
	if e != nil {
		return nil, e
	}
	rows, e := q.ListProjectResearchStates(ctx, tokensqlc.ListProjectResearchStatesParams{ChainID: chainID, ProjectID: projectID, Status: status, Offset: offset, Limit: pageSize})
	if e != nil {
		return nil, e
	}
	items := make([]ProjectResearchState, 0, len(rows))
	for _, r := range rows {
		items = append(items, mapProjectResearchState(r))
	}
	return &ProjectResearchStatePage{Items: items, Total: total, Page: page, PageSize: pageSize}, nil
}

func (s *SQLStore) ListCurrentProjectObservations(ctx context.Context, projectID int64) ([]ProjectObservation, error) {
	q, e := s.querier()
	if e != nil {
		return nil, e
	}
	rows, e := q.ListCurrentProjectObservations(ctx, projectID)
	if e != nil {
		return nil, e
	}
	return mapCurrentProjectObservations(rows)
}

func (s *SQLStore) ListProjectReportsPage(ctx context.Context, chainID, projectID int64, contract common.Address, buildStatus string, page, pageSize int32) (*ProjectReportPage, error) {
	q, e := s.querier()
	if e != nil {
		return nil, e
	}
	page, pageSize, offset := normalizePage(page, pageSize)
	count := tokensqlc.CountCurrentProjectReportsParams{ChainID: chainID, ProjectID: projectID, Contract: optionalAddressBytes(contract), BuildStatus: buildStatus}
	total, e := q.CountCurrentProjectReports(ctx, count)
	if e != nil {
		return nil, e
	}
	rows, e := q.ListCurrentProjectReports(ctx, tokensqlc.ListCurrentProjectReportsParams{ChainID: chainID, ProjectID: projectID, Contract: count.Contract, BuildStatus: buildStatus, Offset: offset, Limit: pageSize})
	if e != nil {
		return nil, e
	}
	items := make([]ProjectReportListItem, 0, len(rows))
	for _, r := range rows {
		report, e := mapProjectReportRevision(tokensqlc.ProjectReportRevision{ID: r.ID, ProjectID: r.ProjectID, Revision: r.Revision, ContentHash: r.ContentHash, CompletenessStatus: r.CompletenessStatus, Evidence: r.Evidence, Report: r.Report, ObservedBlockNumber: r.ObservedBlockNumber, WethPairIsCreated: r.WethPairIsCreated, WethPairIsRemoveLiquidity: r.WethPairIsRemoveLiquidity, WethPairIsMint: r.WethPairIsMint, WethPairQuoteUsdtValueInt: r.WethPairQuoteUsdtValueInt, WethPairLastSwapTimestamp: r.WethPairLastSwapTimestamp, UsdtPairIsCreated: r.UsdtPairIsCreated, UsdtPairIsRemoveLiquidity: r.UsdtPairIsRemoveLiquidity, UsdtPairIsMint: r.UsdtPairIsMint, UsdtPairQuoteUsdtValueInt: r.UsdtPairQuoteUsdtValueInt, UsdtPairLastSwapTimestamp: r.UsdtPairLastSwapTimestamp, BuiltAt: r.BuiltAt, CreatedAt: r.CreatedAt})
		if e != nil {
			return nil, e
		}
		items = append(items, ProjectReportListItem{Report: report, ChainID: r.ChainID, Name: r.Name, Symbol: r.Symbol, Contract: bytesToAddress(r.Contract), BuildStatus: r.BuildStatus, BuildAttempts: r.BuildAttempts, BuildLastError: r.BuildLastError, BuildUpdatedAt: timeValue(r.BuildUpdatedAt)})
	}
	return &ProjectReportPage{Items: items, Total: total, Page: page, PageSize: pageSize}, nil
}

func (s *SQLStore) ListProjectReportRevisionsPage(ctx context.Context, chainID, projectID int64, page, pageSize int32) (*ProjectReportRevisionPage, error) {
	q, e := s.querier()
	if e != nil {
		return nil, e
	}
	page, pageSize, offset := normalizePage(page, pageSize)
	p := tokensqlc.CountProjectReportRevisionsParams{ChainID: chainID, ProjectID: projectID}
	total, e := q.CountProjectReportRevisions(ctx, p)
	if e != nil {
		return nil, e
	}
	rows, e := q.ListProjectReportRevisions(ctx, tokensqlc.ListProjectReportRevisionsParams{ChainID: chainID, ProjectID: projectID, Offset: offset, Limit: pageSize})
	if e != nil {
		return nil, e
	}
	items := make([]ProjectReportRevision, 0, len(rows))
	for _, r := range rows {
		v, e := mapProjectReportRevision(tokensqlc.ProjectReportRevision{ID: r.ID, ProjectID: r.ProjectID, Revision: r.Revision, ContentHash: r.ContentHash, CompletenessStatus: r.CompletenessStatus, Evidence: r.Evidence, Report: r.Report, ObservedBlockNumber: r.ObservedBlockNumber, WethPairIsCreated: r.WethPairIsCreated, WethPairIsRemoveLiquidity: r.WethPairIsRemoveLiquidity, WethPairIsMint: r.WethPairIsMint, WethPairQuoteUsdtValueInt: r.WethPairQuoteUsdtValueInt, WethPairLastSwapTimestamp: r.WethPairLastSwapTimestamp, UsdtPairIsCreated: r.UsdtPairIsCreated, UsdtPairIsRemoveLiquidity: r.UsdtPairIsRemoveLiquidity, UsdtPairIsMint: r.UsdtPairIsMint, UsdtPairQuoteUsdtValueInt: r.UsdtPairQuoteUsdtValueInt, UsdtPairLastSwapTimestamp: r.UsdtPairLastSwapTimestamp, BuiltAt: r.BuiltAt, CreatedAt: r.CreatedAt})
		if e != nil {
			return nil, e
		}
		v.ChainID = r.ChainID
		v.Contract = bytesToAddress(r.Contract)
		items = append(items, v)
	}
	return &ProjectReportRevisionPage{Items: items, Total: total, Page: page, PageSize: pageSize}, nil
}
func (s *SQLStore) GetProjectReportRevision(ctx context.Context, projectID, revision int64) (*ProjectReportRevision, error) {
	q, e := s.querier()
	if e != nil {
		return nil, e
	}
	r, e := q.GetProjectReportRevision(ctx, tokensqlc.GetProjectReportRevisionParams{ProjectID: projectID, Revision: revision})
	if errors.Is(e, pgx.ErrNoRows) {
		return nil, nil
	}
	if e != nil {
		return nil, e
	}
	v, e := mapProjectReportRevision(r)
	return &v, e
}

func (s *SQLStore) ClaimProjectReportBuildTasks(ctx context.Context, limit int32) ([]ProjectReportBuildTask, error) {
	q, e := s.querier()
	if e != nil {
		return nil, e
	}
	rows, e := q.ClaimProjectReportBuildTasks(ctx, tokensqlc.ClaimProjectReportBuildTasksParams{LeaseSeconds: int64(researchTaskLease / time.Second), Limit: limit})
	if e != nil {
		return nil, e
	}
	out := make([]ProjectReportBuildTask, 0, len(rows))
	for _, r := range rows {
		out = append(out, ProjectReportBuildTask{ID: r.ID, ProjectID: r.ProjectID, EvidenceRevision: r.EvidenceRevision, Status: r.Status, Attempts: r.Attempts, AvailableAt: timeValue(r.AvailableAt), LeaseExpiresAt: timeValue(r.LeaseExpiresAt), LastError: textValue(r.LastError)})
	}
	return out, nil
}

func (s *SQLStore) CompleteProjectReportBuild(ctx context.Context, task ProjectReportBuildTask, report ProjectReportRevision) (*ProjectReportRevision, bool, error) {
	if s == nil || s.pool == nil {
		return nil, false, fmt.Errorf("token postgres database is not configured")
	}
	tx, e := s.pool.Begin(ctx)
	if e != nil {
		return nil, false, e
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := tokensqlc.New(tx)
	latest, e := q.GetLatestProjectReportRevision(ctx, task.ProjectID)
	if e == nil && bytes.Equal(latest.ContentHash, report.ContentHash.Bytes()) {
		if _, e = q.MarkProjectReportBuildTaskSucceeded(ctx, task.ID); e != nil {
			return nil, false, e
		}
		if e = tx.Commit(ctx); e != nil {
			return nil, false, e
		}
		v, e := mapProjectReportRevision(latest)
		return &v, false, e
	}
	if e != nil && !errors.Is(e, pgx.ErrNoRows) {
		return nil, false, e
	}
	revision := int64(1)
	if e == nil {
		revision = latest.Revision + 1
	}
	observedBlock, e := nullableUint64(report.ObservedBlockNumber)
	if e != nil {
		return nil, false, e
	}
	wethSwap, e := nullableUint64(report.WethPairLastSwapTimestamp)
	if e != nil {
		return nil, false, e
	}
	usdtSwap, e := nullableUint64(report.UsdtPairLastSwapTimestamp)
	if e != nil {
		return nil, false, e
	}
	row, e := q.InsertProjectReportRevision(ctx, tokensqlc.InsertProjectReportRevisionParams{ProjectID: task.ProjectID, Revision: revision, ContentHash: report.ContentHash.Bytes(), CompletenessStatus: report.CompletenessStatus, Evidence: report.Evidence, Report: report.Report, ObservedBlockNumber: observedBlock, WethPairIsCreated: nullableBool(report.WethPairIsCreated), WethPairIsRemoveLiquidity: nullableBool(report.WethPairIsRemoveLiquidity), WethPairIsMint: nullableBool(report.WethPairIsMint), WethPairQuoteUsdtValueInt: nullableNumericFromBigInt(report.WethPairQuoteUsdtValueInt), WethPairLastSwapTimestamp: wethSwap, UsdtPairIsCreated: nullableBool(report.UsdtPairIsCreated), UsdtPairIsRemoveLiquidity: nullableBool(report.UsdtPairIsRemoveLiquidity), UsdtPairIsMint: nullableBool(report.UsdtPairIsMint), UsdtPairQuoteUsdtValueInt: nullableNumericFromBigInt(report.UsdtPairQuoteUsdtValueInt), UsdtPairLastSwapTimestamp: usdtSwap, BuiltAt: nullableTime(report.BuiltAt)})
	if e != nil {
		return nil, false, e
	}
	if _, e = q.UpdateProjectResearchCurrentReport(ctx, tokensqlc.UpdateProjectResearchCurrentReportParams{CurrentReportRevision: nullableInt64(revision), ProjectID: task.ProjectID}); e != nil {
		return nil, false, e
	}
	if _, e = q.EnqueueProjectSelectionEvaluationTask(ctx, tokensqlc.EnqueueProjectSelectionEvaluationTaskParams{ProjectID: task.ProjectID, ReportRevision: revision}); e != nil {
		return nil, false, e
	}
	if _, e = q.MarkProjectReportBuildTaskSucceeded(ctx, task.ID); e != nil {
		return nil, false, e
	}
	if e = tx.Commit(ctx); e != nil {
		return nil, false, e
	}
	v, e := mapProjectReportRevision(row)
	return &v, true, e
}

func (s *SQLStore) MarkProjectReportBuildTaskFailed(ctx context.Context, task ProjectReportBuildTask, lastError string) error {
	q, e := s.querier()
	if e != nil {
		return e
	}
	backoff := time.Duration(1<<min(int(task.Attempts), 4)) * time.Second
	if _, e = q.RetryProjectReportBuildTask(ctx, tokensqlc.RetryProjectReportBuildTaskParams{AvailableAt: nullableTime(time.Now().UTC().Add(backoff)), LastError: nullableText(lastError), ID: task.ID}); e == nil {
		return nil
	} else if !errors.Is(e, pgx.ErrNoRows) {
		return e
	}
	_, e = q.FailProjectReportBuildTask(ctx, tokensqlc.FailProjectReportBuildTaskParams{LastError: nullableText(lastError), ID: task.ID})
	return e
}

func (s *SQLStore) ListProjectSelectionsPage(ctx context.Context, chainID, projectID int64, outcome string, page, pageSize int32) (*ProjectSelectionPage, error) {
	q, e := s.querier()
	if e != nil {
		return nil, e
	}
	page, pageSize, offset := normalizePage(page, pageSize)
	p := tokensqlc.CountProjectSelectionsParams{ChainID: chainID, ProjectID: projectID, Outcome: outcome}
	total, e := q.CountProjectSelections(ctx, p)
	if e != nil {
		return nil, e
	}
	rows, e := q.ListProjectSelections(ctx, tokensqlc.ListProjectSelectionsParams{ChainID: chainID, ProjectID: projectID, Outcome: outcome, Offset: offset, Limit: pageSize})
	if e != nil {
		return nil, e
	}
	items := make([]ProjectSelection, 0, len(rows))
	for _, r := range rows {
		v := mapProjectSelection(tokensqlc.ProjectSelection{ID: r.ID, ProjectID: r.ProjectID, Outcome: r.Outcome, StrategyKey: r.StrategyKey, StrategyVersion: r.StrategyVersion, ReportRevision: r.ReportRevision, ReasonCodes: r.ReasonCodes, ReasonDetail: r.ReasonDetail, DecidedAt: r.DecidedAt, CreatedAt: r.CreatedAt})
		v.ChainID = r.ChainID
		v.Contract = bytesToAddress(r.Contract)
		items = append(items, v)
	}
	return &ProjectSelectionPage{Items: items, Total: total, Page: page, PageSize: pageSize}, nil
}

func (s *SQLStore) ClaimProjectSelectionEvaluationTasks(ctx context.Context, limit int32) ([]ProjectSelectionEvaluationTask, error) {
	q, e := s.querier()
	if e != nil {
		return nil, e
	}
	rows, e := q.ClaimProjectSelectionEvaluationTasks(ctx, tokensqlc.ClaimProjectSelectionEvaluationTasksParams{LeaseSeconds: int64(researchTaskLease / time.Second), Limit: limit})
	if e != nil {
		return nil, e
	}
	out := make([]ProjectSelectionEvaluationTask, 0, len(rows))
	for _, r := range rows {
		out = append(out, ProjectSelectionEvaluationTask{ID: r.ID, ProjectID: r.ProjectID, ReportRevision: r.ReportRevision, Status: r.Status, Attempts: r.Attempts, AvailableAt: timeValue(r.AvailableAt), LeaseExpiresAt: timeValue(r.LeaseExpiresAt), LastError: textValue(r.LastError)})
	}
	return out, nil
}

func (s *SQLStore) CompleteProjectSelectionEvaluation(ctx context.Context, task ProjectSelectionEvaluationTask, selection ProjectSelection) (*ProjectSelection, bool, error) {
	if s == nil || s.pool == nil {
		return nil, false, fmt.Errorf("token postgres database is not configured")
	}
	tx, e := s.pool.Begin(ctx)
	if e != nil {
		return nil, false, e
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := tokensqlc.New(tx)
	latest, e := q.GetLatestProjectSelection(ctx, task.ProjectID)
	duplicate := e == nil && latest.Outcome == selection.Outcome && latest.StrategyKey == selection.StrategyKey && latest.StrategyVersion == selection.StrategyVersion && latest.ReasonDetail == selection.ReasonDetail && equalStrings(latest.ReasonCodes, selection.ReasonCodes)
	now := selection.DecidedAt
	if now.IsZero() {
		now = time.Now().UTC()
	}
	selectionID := int64(0)
	var result ProjectSelection
	if duplicate {
		selectionID = latest.ID
		result = mapProjectSelection(latest)
	} else {
		row, e := q.InsertProjectSelection(ctx, tokensqlc.InsertProjectSelectionParams{ProjectID: task.ProjectID, Outcome: selection.Outcome, StrategyKey: selection.StrategyKey, StrategyVersion: selection.StrategyVersion, ReportRevision: task.ReportRevision, ReasonCodes: selection.ReasonCodes, ReasonDetail: selection.ReasonDetail, DecidedAt: nullableTime(now)})
		if e != nil {
			return nil, false, e
		}
		selectionID = row.ID
		result = mapProjectSelection(row)
	}
	if _, e = q.UpdateProjectResearchSelection(ctx, tokensqlc.UpdateProjectResearchSelectionParams{Outcome: selection.Outcome, SelectionID: nullableInt64(selectionID), ReportRevision: nullableInt64(task.ReportRevision), EvaluatedAt: nullableTime(now), ProjectID: task.ProjectID}); e != nil {
		return nil, false, e
	}
	if _, e = q.MarkProjectSelectionEvaluationTaskSucceeded(ctx, task.ID); e != nil {
		return nil, false, e
	}
	if e = tx.Commit(ctx); e != nil {
		return nil, false, e
	}
	return &result, !duplicate, nil
}
func (s *SQLStore) MarkProjectSelectionEvaluationTaskFailed(ctx context.Context, task ProjectSelectionEvaluationTask, lastError string) error {
	q, e := s.querier()
	if e != nil {
		return e
	}
	backoff := time.Duration(1<<min(int(task.Attempts), 4)) * time.Second
	if _, e = q.RetryProjectSelectionEvaluationTask(ctx, tokensqlc.RetryProjectSelectionEvaluationTaskParams{AvailableAt: nullableTime(time.Now().UTC().Add(backoff)), LastError: nullableText(lastError), ID: task.ID}); e == nil {
		return nil
	} else if !errors.Is(e, pgx.ErrNoRows) {
		return e
	}
	_, e = q.FailProjectSelectionEvaluationTask(ctx, tokensqlc.FailProjectSelectionEvaluationTaskParams{LastError: nullableText(lastError), ID: task.ID})
	return e
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
