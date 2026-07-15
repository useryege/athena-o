package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	tokensqlc "github.com/useryege/athena/internal/token/adapters/postgres/sqlc"
	"github.com/useryege/athena/internal/token/research"
	"github.com/useryege/athena/internal/token/selection"
)

func (s *Database) ListProjectSelectionsPage(ctx context.Context, chainID, projectID int64, outcome string, page, pageSize int32) (*selection.Page, error) {
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
	items := make([]selection.ProjectSelection, 0, len(rows))
	for _, r := range rows {
		v := mapProjectSelection(tokensqlc.ProjectSelection{ID: r.ID, ProjectID: r.ProjectID, Outcome: r.Outcome, StrategyKey: r.StrategyKey, StrategyVersion: r.StrategyVersion, ReportRevision: r.ReportRevision, ReasonCodes: r.ReasonCodes, ReasonDetail: r.ReasonDetail, DecidedAt: r.DecidedAt, CreatedAt: r.CreatedAt})
		v.ChainID = r.ChainID
		v.Contract = bytesToAddress(r.Contract)
		items = append(items, v)
	}
	return &selection.Page{Items: items, Total: total, Page: page, PageSize: pageSize}, nil
}

func (s *Database) ClaimProjectSelectionEvaluationTasks(ctx context.Context, limit int32) ([]selection.ProjectSelectionEvaluationTask, error) {
	q, e := s.querier()
	if e != nil {
		return nil, e
	}
	rows, e := q.ClaimProjectSelectionEvaluationTasks(ctx, tokensqlc.ClaimProjectSelectionEvaluationTasksParams{LeaseSeconds: int64(researchTaskLease / time.Second), Limit: limit})
	if e != nil {
		return nil, e
	}
	out := make([]selection.ProjectSelectionEvaluationTask, 0, len(rows))
	for _, r := range rows {
		out = append(out, selection.ProjectSelectionEvaluationTask{ID: r.ID, ProjectID: r.ProjectID, ReportRevision: r.ReportRevision, Status: research.TaskStatus(r.Status), Attempts: r.Attempts, AvailableAt: timeValue(r.AvailableAt), LeaseExpiresAt: timeValue(r.LeaseExpiresAt), LastError: textValue(r.LastError)})
	}
	return out, nil
}

func (s *Database) CompleteProjectSelectionEvaluation(ctx context.Context, task selection.ProjectSelectionEvaluationTask, selectionItem selection.ProjectSelection) (*selection.ProjectSelection, bool, error) {
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
	duplicate := e == nil && latest.Outcome == string(selectionItem.Outcome) && latest.StrategyKey == selectionItem.StrategyKey && latest.StrategyVersion == selectionItem.StrategyVersion && latest.ReasonDetail == selectionItem.ReasonDetail && equalStrings(latest.ReasonCodes, selectionItem.ReasonCodes)
	now := selectionItem.DecidedAt
	if now.IsZero() {
		now = time.Now().UTC()
	}
	selectionID := int64(0)
	var result selection.ProjectSelection
	if duplicate {
		selectionID = latest.ID
		result = mapProjectSelection(latest)
	} else {
		row, e := q.InsertProjectSelection(ctx, tokensqlc.InsertProjectSelectionParams{ProjectID: task.ProjectID, Outcome: string(selectionItem.Outcome), StrategyKey: selectionItem.StrategyKey, StrategyVersion: selectionItem.StrategyVersion, ReportRevision: task.ReportRevision, ReasonCodes: selectionItem.ReasonCodes, ReasonDetail: selectionItem.ReasonDetail, DecidedAt: nullableTime(now)})
		if e != nil {
			return nil, false, e
		}
		selectionID = row.ID
		result = mapProjectSelection(row)
	}
	if _, e = q.UpdateProjectResearchSelection(ctx, tokensqlc.UpdateProjectResearchSelectionParams{Outcome: string(selectionItem.Outcome), SelectionID: nullableInt64(selectionID), ReportRevision: nullableInt64(task.ReportRevision), EvaluatedAt: nullableTime(now), ProjectID: task.ProjectID}); e != nil {
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
func (s *Database) MarkProjectSelectionEvaluationTaskFailed(ctx context.Context, task selection.ProjectSelectionEvaluationTask, lastError string) error {
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
