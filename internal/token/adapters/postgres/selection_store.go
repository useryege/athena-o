package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	tokensqlc "github.com/useryege/athena/internal/token/adapters/postgres/sqlc"
	"github.com/useryege/athena/internal/token/selection"
	selectionapp "github.com/useryege/athena/internal/token/selection/application"
)

func (s *SelectionRepository) ListProjectSelectionsPage(ctx context.Context, chainID, projectID int64, outcome string, page, pageSize int32) (*selection.Page, error) {
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

func (s *SelectionRepository) ClaimSelectionTasks(ctx context.Context, limit int32) ([]selection.ProjectSelectionEvaluationTask, error) {
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
		out = append(out, selection.ProjectSelectionEvaluationTask{ID: r.ID, ProjectID: r.ProjectID, ReportRevision: r.ReportRevision, Status: selection.TaskStatus(r.Status), Attempts: r.Attempts, AvailableAt: timeValue(r.AvailableAt), LeaseExpiresAt: timeValue(r.LeaseExpiresAt), LastError: textValue(r.LastError)})
	}
	return out, nil
}

func (s *SelectionRepository) CommitSelectionAndCompleteTask(ctx context.Context, command selectionapp.CommitSelectionCommand) (*selection.ProjectSelection, bool, error) {
	if s == nil || s.pool == nil {
		return nil, false, fmt.Errorf("token selection repository is not configured")
	}
	tx, e := s.pool.Begin(ctx)
	if e != nil {
		return nil, false, e
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := tokensqlc.New(tx)
	task, selectionItem := command.Task, command.Selection
	latest, e := q.GetLatestProjectSelection(ctx, task.ProjectID)
	duplicate := e == nil && latest.Outcome == string(selectionItem.Outcome) && latest.StrategyKey == selectionItem.StrategyKey && latest.StrategyVersion == selectionItem.StrategyVersion && latest.ReasonDetail == selectionItem.ReasonDetail && equalStrings(latest.ReasonCodes, selectionItem.ReasonCodes)
	now := selectionItem.DecidedAt
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
func (s *SelectionRepository) RetrySelectionTask(ctx context.Context, command selectionapp.RetrySelectionTaskCommand) error {
	q, e := s.querier()
	if e != nil {
		return e
	}
	_, e = q.RetryProjectSelectionEvaluationTask(ctx, tokensqlc.RetryProjectSelectionEvaluationTaskParams{AvailableAt: nullableTime(command.AvailableAt), LastError: nullableText(command.LastError), ID: command.Task.ID})
	if errors.Is(e, pgx.ErrNoRows) {
		return nil
	}
	return e
}

func (s *SelectionRepository) FailSelectionTask(ctx context.Context, command selectionapp.FailSelectionTaskCommand) error {
	q, e := s.querier()
	if e != nil {
		return e
	}
	_, e = q.FailProjectSelectionEvaluationTask(ctx, tokensqlc.FailProjectSelectionEvaluationTaskParams{LastError: nullableText(command.LastError), ID: command.Task.ID})
	return e
}

func (s *SelectionRepository) GetReportSnapshot(ctx context.Context, projectID, revision int64) (*selection.ReportSnapshot, error) {
	q, err := s.querier()
	if err != nil {
		return nil, err
	}
	row, err := q.GetProjectReportRevision(ctx, tokensqlc.GetProjectReportRevisionParams{ProjectID: projectID, Revision: revision})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if !json.Valid(row.Report) {
		return nil, fmt.Errorf("project report revision %d contains invalid JSON", row.ID)
	}
	return &selection.ReportSnapshot{ProjectID: row.ProjectID, Revision: row.Revision, SchemaVersion: row.SchemaVersion, CompletenessStatus: row.CompletenessStatus, Report: append(json.RawMessage(nil), row.Report...)}, nil
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
