package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	tokensqlc "github.com/useryege/athena/internal/token/adapters/postgres/sqlc"
	"github.com/useryege/athena/internal/token/research"
)

const collectionTaskLease = 90 * time.Second

func (s *Database) ListDueProjectDataCollectionTasks(ctx context.Context, dataType research.DataCollectionType, chainIDs []int64, limit int32) ([]research.ProjectDataCollectionTaskWithProject, error) {
	if len(chainIDs) == 0 {
		return nil, nil
	}
	q, e := s.querier()
	if e != nil {
		return nil, e
	}
	rows, e := q.ClaimProjectDataCollectionTasks(ctx, tokensqlc.ClaimProjectDataCollectionTasksParams{LeaseSeconds: int64(collectionTaskLease / time.Second), DataType: string(dataType), ChainIds: chainIDs, Limit: limit})
	if e != nil {
		return nil, fmt.Errorf("claim collection tasks: %w", e)
	}
	out := make([]research.ProjectDataCollectionTaskWithProject, 0, len(rows))
	for _, row := range rows {
		projectRow, e := q.GetProjectForDataCollectionTask(ctx, row.ID)
		if e != nil {
			return nil, e
		}
		project, e := mapProject(projectRow)
		if e != nil {
			return nil, e
		}
		out = append(out, research.ProjectDataCollectionTaskWithProject{Task: mapProjectDataCollectionTask(row), Project: *project})
	}
	return out, nil
}
func (s *Database) GetProjectDataCollectionTask(ctx context.Context, id int64) (*research.ProjectDataCollectionTask, error) {
	q, e := s.querier()
	if e != nil {
		return nil, e
	}
	row, e := q.GetProjectDataCollectionTask(ctx, id)
	if errors.Is(e, pgx.ErrNoRows) {
		return nil, nil
	}
	if e != nil {
		return nil, e
	}
	v := mapProjectDataCollectionTask(row)
	return &v, nil
}
func (s *Database) ListProjectDataCollectionTasks(ctx context.Context, projectID int64, dataType, status string, page, pageSize int32) (*research.CollectionTaskPage, error) {
	q, e := s.querier()
	if e != nil {
		return nil, e
	}
	page, pageSize, offset := normalizePage(page, pageSize)
	p := tokensqlc.CountProjectDataCollectionTasksParams{ProjectID: projectID, DataType: dataType, Status: status}
	total, e := q.CountProjectDataCollectionTasks(ctx, p)
	if e != nil {
		return nil, e
	}
	rows, e := q.ListProjectDataCollectionTasks(ctx, tokensqlc.ListProjectDataCollectionTasksParams{ProjectID: projectID, DataType: dataType, Status: status, Offset: offset, Limit: pageSize})
	if e != nil {
		return nil, e
	}
	items := make([]research.ProjectDataCollectionTask, 0, len(rows))
	for _, r := range rows {
		items = append(items, mapProjectDataCollectionTask(r))
	}
	return &research.CollectionTaskPage{Items: items, Total: total, Page: page, PageSize: pageSize}, nil
}

func (s *Database) MarkProjectDataCollectionTaskSucceeded(ctx context.Context, task research.ProjectDataCollectionTask) (bool, error) {
	if s == nil || s.pool == nil {
		return false, fmt.Errorf("token postgres database is not configured")
	}
	tx, e := s.pool.Begin(ctx)
	if e != nil {
		return false, e
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := tokensqlc.New(tx)
	n, e := q.MarkProjectDataCollectionTaskSucceeded(ctx, task.ID)
	if e != nil {
		return false, e
	}
	if n == 0 {
		return false, nil
	}
	now := time.Now().UTC()
	if task.DataType == research.DataCollectionTypeContractCodeSource {
		_, e = q.CompleteProjectDataCollectionSchedule(ctx, tokensqlc.CompleteProjectDataCollectionScheduleParams{LastCheckedAt: nullableTime(now), ProjectID: task.ProjectID, DataType: string(task.DataType)})
	} else {
		e = markScheduleSucceeded(ctx, q, task.ProjectID, task.DataType, now)
	}
	if e != nil {
		return false, e
	}
	if e = tx.Commit(ctx); e != nil {
		return false, e
	}
	return true, nil
}

func (s *Database) MarkProjectDataCollectionTaskFailed(ctx context.Context, task research.ProjectDataCollectionTask, lastError string) (*research.ProjectDataCollectionTask, bool, error) {
	if s == nil || s.pool == nil {
		return nil, false, fmt.Errorf("token postgres database is not configured")
	}
	tx, e := s.pool.Begin(ctx)
	if e != nil {
		return nil, false, e
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := tokensqlc.New(tx)
	backoff := time.Duration(1<<min(int(task.Attempts), 4)) * time.Second
	row, e := q.RetryProjectDataCollectionTask(ctx, tokensqlc.RetryProjectDataCollectionTaskParams{AvailableAt: nullableTime(time.Now().UTC().Add(backoff)), LastError: nullableText(lastError), ID: task.ID})
	if e == nil {
		if e = tx.Commit(ctx); e != nil {
			return nil, false, e
		}
		v := mapProjectDataCollectionTask(row)
		return &v, true, nil
	}
	if !errors.Is(e, pgx.ErrNoRows) {
		return nil, false, e
	}
	n, e := q.FailProjectDataCollectionTask(ctx, tokensqlc.FailProjectDataCollectionTaskParams{LastError: nullableText(lastError), ID: task.ID})
	if e != nil {
		return nil, false, e
	}
	if n == 0 {
		return nil, false, nil
	}
	if e = markScheduleFailed(ctx, q, task.ProjectID, task.DataType, lastError, time.Now().UTC()); e != nil {
		return nil, false, e
	}
	row, e = q.GetProjectDataCollectionTask(ctx, task.ID)
	if e != nil {
		return nil, false, e
	}
	if e = tx.Commit(ctx); e != nil {
		return nil, false, e
	}
	v := mapProjectDataCollectionTask(row)
	return &v, true, nil
}
