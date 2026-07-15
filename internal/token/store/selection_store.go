package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/useryege/athena/internal/token/domain"
	tokensqlc "github.com/useryege/athena/internal/token/store/sqlc"
)

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
	items := make([]domain.ProjectSelection, 0, len(rows))
	for _, r := range rows {
		v := mapProjectSelection(tokensqlc.ProjectSelection{ID: r.ID, ProjectID: r.ProjectID, Outcome: r.Outcome, StrategyKey: r.StrategyKey, StrategyVersion: r.StrategyVersion, ReportRevision: r.ReportRevision, ReasonCodes: r.ReasonCodes, ReasonDetail: r.ReasonDetail, DecidedAt: r.DecidedAt, CreatedAt: r.CreatedAt})
		v.ChainID = r.ChainID
		v.Contract = bytesToAddress(r.Contract)
		items = append(items, v)
	}
	return &ProjectSelectionPage{Items: items, Total: total, Page: page, PageSize: pageSize}, nil
}

func (s *SQLStore) ClaimProjectSelectionEvaluationTasks(ctx context.Context, limit int32) ([]domain.ProjectSelectionEvaluationTask, error) {
	q, e := s.querier()
	if e != nil {
		return nil, e
	}
	rows, e := q.ClaimProjectSelectionEvaluationTasks(ctx, tokensqlc.ClaimProjectSelectionEvaluationTasksParams{LeaseSeconds: int64(researchTaskLease / time.Second), Limit: limit})
	if e != nil {
		return nil, e
	}
	out := make([]domain.ProjectSelectionEvaluationTask, 0, len(rows))
	for _, r := range rows {
		out = append(out, domain.ProjectSelectionEvaluationTask{ID: r.ID, ProjectID: r.ProjectID, ReportRevision: r.ReportRevision, Status: domain.TaskStatus(r.Status), Attempts: r.Attempts, AvailableAt: timeValue(r.AvailableAt), LeaseExpiresAt: timeValue(r.LeaseExpiresAt), LastError: textValue(r.LastError)})
	}
	return out, nil
}

func (s *SQLStore) CompleteProjectSelectionEvaluation(ctx context.Context, task domain.ProjectSelectionEvaluationTask, selection domain.ProjectSelection) (*domain.ProjectSelection, bool, error) {
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
	duplicate := e == nil && latest.Outcome == string(selection.Outcome) && latest.StrategyKey == selection.StrategyKey && latest.StrategyVersion == selection.StrategyVersion && latest.ReasonDetail == selection.ReasonDetail && equalStrings(latest.ReasonCodes, selection.ReasonCodes)
	now := selection.DecidedAt
	if now.IsZero() {
		now = time.Now().UTC()
	}
	selectionID := int64(0)
	var result domain.ProjectSelection
	if duplicate {
		selectionID = latest.ID
		result = mapProjectSelection(latest)
	} else {
		row, e := q.InsertProjectSelection(ctx, tokensqlc.InsertProjectSelectionParams{ProjectID: task.ProjectID, Outcome: string(selection.Outcome), StrategyKey: selection.StrategyKey, StrategyVersion: selection.StrategyVersion, ReportRevision: task.ReportRevision, ReasonCodes: selection.ReasonCodes, ReasonDetail: selection.ReasonDetail, DecidedAt: nullableTime(now)})
		if e != nil {
			return nil, false, e
		}
		selectionID = row.ID
		result = mapProjectSelection(row)
	}
	if _, e = q.UpdateProjectResearchSelection(ctx, tokensqlc.UpdateProjectResearchSelectionParams{Outcome: string(selection.Outcome), SelectionID: nullableInt64(selectionID), ReportRevision: nullableInt64(task.ReportRevision), EvaluatedAt: nullableTime(now), ProjectID: task.ProjectID}); e != nil {
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
func (s *SQLStore) MarkProjectSelectionEvaluationTaskFailed(ctx context.Context, task domain.ProjectSelectionEvaluationTask, lastError string) error {
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
