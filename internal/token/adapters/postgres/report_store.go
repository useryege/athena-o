package postgres

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/jackc/pgx/v5"
	tokensqlc "github.com/useryege/athena/internal/token/adapters/postgres/sqlc"
	"github.com/useryege/athena/internal/token/reporting"
	"github.com/useryege/athena/internal/token/research"
)

const researchTaskLease = 90 * time.Second

func (s *Database) ListProjectReportsPage(ctx context.Context, chainID, projectID int64, contract common.Address, buildStatus string, page, pageSize int32) (*reporting.ProjectReportPage, error) {
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
	items := make([]reporting.ProjectReportReadModel, 0, len(rows))
	for _, r := range rows {
		report, e := mapProjectReportRevision(tokensqlc.ProjectReportRevision{ID: r.ID, ProjectID: r.ProjectID, Revision: r.Revision, SchemaVersion: r.SchemaVersion, ContentHash: r.ContentHash, CompletenessStatus: r.CompletenessStatus, Evidence: r.Evidence, Report: r.Report, ObservedBlockNumber: r.ObservedBlockNumber, WethPairIsCreated: r.WethPairIsCreated, WethPairIsRemoveLiquidity: r.WethPairIsRemoveLiquidity, WethPairIsMint: r.WethPairIsMint, WethPairQuoteUsdtValueInt: r.WethPairQuoteUsdtValueInt, WethPairLastSwapTimestamp: r.WethPairLastSwapTimestamp, UsdtPairIsCreated: r.UsdtPairIsCreated, UsdtPairIsRemoveLiquidity: r.UsdtPairIsRemoveLiquidity, UsdtPairIsMint: r.UsdtPairIsMint, UsdtPairQuoteUsdtValueInt: r.UsdtPairQuoteUsdtValueInt, UsdtPairLastSwapTimestamp: r.UsdtPairLastSwapTimestamp, BuiltAt: r.BuiltAt, CreatedAt: r.CreatedAt})
		if e != nil {
			return nil, e
		}
		items = append(items, reporting.ProjectReportReadModel{Report: report, ChainID: r.ChainID, Name: r.Name, Symbol: r.Symbol, Contract: bytesToAddress(r.Contract), BuildStatus: r.BuildStatus, BuildAttempts: r.BuildAttempts, BuildLastError: r.BuildLastError, BuildUpdatedAt: timeValue(r.BuildUpdatedAt)})
	}
	return &reporting.ProjectReportPage{Items: items, Total: total, Page: page, PageSize: pageSize}, nil
}

func (s *Database) ListProjectReportRevisionsPage(ctx context.Context, chainID, projectID int64, page, pageSize int32) (*reporting.ReportRevisionPage, error) {
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
	items := make([]reporting.ProjectReportRevision, 0, len(rows))
	for _, r := range rows {
		v, e := mapProjectReportRevision(tokensqlc.ProjectReportRevision{ID: r.ID, ProjectID: r.ProjectID, Revision: r.Revision, SchemaVersion: r.SchemaVersion, ContentHash: r.ContentHash, CompletenessStatus: r.CompletenessStatus, Evidence: r.Evidence, Report: r.Report, ObservedBlockNumber: r.ObservedBlockNumber, WethPairIsCreated: r.WethPairIsCreated, WethPairIsRemoveLiquidity: r.WethPairIsRemoveLiquidity, WethPairIsMint: r.WethPairIsMint, WethPairQuoteUsdtValueInt: r.WethPairQuoteUsdtValueInt, WethPairLastSwapTimestamp: r.WethPairLastSwapTimestamp, UsdtPairIsCreated: r.UsdtPairIsCreated, UsdtPairIsRemoveLiquidity: r.UsdtPairIsRemoveLiquidity, UsdtPairIsMint: r.UsdtPairIsMint, UsdtPairQuoteUsdtValueInt: r.UsdtPairQuoteUsdtValueInt, UsdtPairLastSwapTimestamp: r.UsdtPairLastSwapTimestamp, BuiltAt: r.BuiltAt, CreatedAt: r.CreatedAt})
		if e != nil {
			return nil, e
		}
		v.ChainID = r.ChainID
		v.Contract = bytesToAddress(r.Contract)
		items = append(items, v)
	}
	return &reporting.ReportRevisionPage{Items: items, Total: total, Page: page, PageSize: pageSize}, nil
}
func (s *Database) GetProjectReportRevision(ctx context.Context, projectID, revision int64) (*reporting.ProjectReportRevision, error) {
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

func (s *Database) ClaimProjectReportBuildTasks(ctx context.Context, limit int32) ([]reporting.ProjectReportBuildTask, error) {
	q, e := s.querier()
	if e != nil {
		return nil, e
	}
	rows, e := q.ClaimProjectReportBuildTasks(ctx, tokensqlc.ClaimProjectReportBuildTasksParams{LeaseSeconds: int64(researchTaskLease / time.Second), Limit: limit})
	if e != nil {
		return nil, e
	}
	out := make([]reporting.ProjectReportBuildTask, 0, len(rows))
	for _, r := range rows {
		out = append(out, reporting.ProjectReportBuildTask{ID: r.ID, ProjectID: r.ProjectID, EvidenceRevision: r.EvidenceRevision, Status: research.TaskStatus(r.Status), Attempts: r.Attempts, AvailableAt: timeValue(r.AvailableAt), LeaseExpiresAt: timeValue(r.LeaseExpiresAt), LastError: textValue(r.LastError)})
	}
	return out, nil
}

func (s *Database) CompleteProjectReportBuild(ctx context.Context, task reporting.ProjectReportBuildTask, report reporting.ProjectReportRevision) (*reporting.ProjectReportRevision, bool, error) {
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
	risk := report.Report.RiskSummary
	wethSwap, e := nullableUint64(risk.WethPairLastSwapTimestamp)
	if e != nil {
		return nil, false, e
	}
	usdtSwap, e := nullableUint64(risk.UsdtPairLastSwapTimestamp)
	if e != nil {
		return nil, false, e
	}
	evidenceJSON, e := json.Marshal(report.Evidence)
	if e != nil {
		return nil, false, e
	}
	reportJSON, e := json.Marshal(report.Report)
	if e != nil {
		return nil, false, e
	}
	row, e := q.InsertProjectReportRevision(ctx, tokensqlc.InsertProjectReportRevisionParams{ProjectID: task.ProjectID, Revision: revision, SchemaVersion: report.SchemaVersion, ContentHash: report.ContentHash.Bytes(), CompletenessStatus: report.CompletenessStatus, Evidence: evidenceJSON, Report: reportJSON, ObservedBlockNumber: observedBlock, WethPairIsCreated: nullableBool(risk.WethPairIsCreated), WethPairIsRemoveLiquidity: nullableBool(risk.WethPairIsRemoveLiquidity), WethPairIsMint: nullableBool(risk.WethPairIsMint), WethPairQuoteUsdtValueInt: nullableNumericFromBigInt(risk.WethPairQuoteUsdtValueInt), WethPairLastSwapTimestamp: wethSwap, UsdtPairIsCreated: nullableBool(risk.UsdtPairIsCreated), UsdtPairIsRemoveLiquidity: nullableBool(risk.UsdtPairIsRemoveLiquidity), UsdtPairIsMint: nullableBool(risk.UsdtPairIsMint), UsdtPairQuoteUsdtValueInt: nullableNumericFromBigInt(risk.UsdtPairQuoteUsdtValueInt), UsdtPairLastSwapTimestamp: usdtSwap, BuiltAt: nullableTime(report.BuiltAt)})
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

func (s *Database) MarkProjectReportBuildTaskFailed(ctx context.Context, task reporting.ProjectReportBuildTask, lastError string) error {
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
