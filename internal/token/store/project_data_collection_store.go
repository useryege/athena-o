package store

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/jackc/pgx/v5"
	tokensqlc "github.com/useryege/athena/internal/token/store/sqlc"
)

const collectionTaskLease = 90 * time.Second

func (s *SQLStore) ScheduleDueProjectDataCollectionTasks(ctx context.Context, limit int32) (int, error) {
	q, err := s.querier()
	if err != nil {
		return 0, err
	}
	if limit <= 0 {
		limit = 100
	}
	schedules, err := q.ListDueProjectDataCollectionSchedules(ctx, limit)
	if err != nil {
		return 0, fmt.Errorf("list due collection schedules: %w", err)
	}
	created := 0
	for _, candidate := range schedules {
		tx, err := s.pool.Begin(ctx)
		if err != nil {
			return created, err
		}
		tq := tokensqlc.New(tx)
		locked, err := tq.LockProjectDataCollectionSchedule(ctx, tokensqlc.LockProjectDataCollectionScheduleParams{ProjectID: candidate.ProjectID, DataType: candidate.DataType})
		if err != nil {
			_ = tx.Rollback(ctx)
			return created, fmt.Errorf("lock collection schedule: %w", err)
		}
		if locked.Status != ProjectDataCollectionScheduleStatusActive || timeValue(locked.NextRunAt).After(time.Now().UTC()) {
			_ = tx.Rollback(ctx)
			continue
		}
		revision := locked.LatestTaskRevision + 1
		if _, err = tq.CreateProjectDataCollectionTask(ctx, tokensqlc.CreateProjectDataCollectionTaskParams{ProjectID: locked.ProjectID, DataType: locked.DataType, Revision: revision}); err != nil {
			_ = tx.Rollback(ctx)
			return created, fmt.Errorf("create collection task: %w", err)
		}
		next := time.Now().UTC().Add(time.Duration(locked.RefreshIntervalSeconds) * time.Second)
		if _, err = tq.AdvanceProjectDataCollectionSchedule(ctx, tokensqlc.AdvanceProjectDataCollectionScheduleParams{LatestTaskRevision: revision, NextRunAt: nullableTime(next), ProjectID: locked.ProjectID, DataType: locked.DataType}); err != nil {
			_ = tx.Rollback(ctx)
			return created, fmt.Errorf("advance collection schedule: %w", err)
		}
		if err = tx.Commit(ctx); err != nil {
			return created, err
		}
		created++
	}
	return created, nil
}

func (s *SQLStore) ApplyResearchPolicy(ctx context.Context, intervals map[string]time.Duration, ttl time.Duration) error {
	q, err := s.querier()
	if err != nil {
		return err
	}
	for dataType, interval := range intervals {
		if _, err = q.ApplyProjectDataCollectionSchedulePolicy(ctx, tokensqlc.ApplyProjectDataCollectionSchedulePolicyParams{RefreshIntervalSeconds: int64(interval / time.Second), DataType: dataType}); err != nil {
			return fmt.Errorf("apply %s schedule policy: %w", dataType, err)
		}
	}
	if _, err = q.ApplyProjectResearchTTL(ctx, int64(ttl/time.Second)); err != nil {
		return fmt.Errorf("apply research ttl: %w", err)
	}
	return nil
}

func (s *SQLStore) MaintainResearchLifecycle(ctx context.Context) error {
	q, e := s.querier()
	if e != nil {
		return e
	}
	if _, e = q.ExpireProjectResearchStates(ctx); e != nil {
		return e
	}
	_, e = q.PauseTerminalProjectDataCollectionSchedules(ctx)
	return e
}

func (s *SQLStore) ListDueProjectDataCollectionTasks(ctx context.Context, dataType string, chainIDs []int64, limit int32) ([]ProjectDataCollectionTaskWithProject, error) {
	if len(chainIDs) == 0 {
		return nil, nil
	}
	q, e := s.querier()
	if e != nil {
		return nil, e
	}
	rows, e := q.ClaimProjectDataCollectionTasks(ctx, tokensqlc.ClaimProjectDataCollectionTasksParams{LeaseSeconds: int64(collectionTaskLease / time.Second), DataType: dataType, ChainIds: chainIDs, Limit: limit})
	if e != nil {
		return nil, fmt.Errorf("claim collection tasks: %w", e)
	}
	out := make([]ProjectDataCollectionTaskWithProject, 0, len(rows))
	for _, row := range rows {
		projectRow, e := q.GetProjectForDataCollectionTask(ctx, row.ID)
		if e != nil {
			return nil, e
		}
		project, e := mapProject(projectRow)
		if e != nil {
			return nil, e
		}
		out = append(out, ProjectDataCollectionTaskWithProject{Task: mapProjectDataCollectionTask(row), Project: *project})
	}
	return out, nil
}
func (s *SQLStore) GetProjectDataCollectionTask(ctx context.Context, id int64) (*ProjectDataCollectionTask, error) {
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
func (s *SQLStore) ListProjectDataCollectionTasks(ctx context.Context, projectID int64, dataType, status string, page, pageSize int32) (*ProjectDataCollectionTaskPage, error) {
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
	items := make([]ProjectDataCollectionTask, 0, len(rows))
	for _, r := range rows {
		items = append(items, mapProjectDataCollectionTask(r))
	}
	return &ProjectDataCollectionTaskPage{Items: items, Total: total, Page: page, PageSize: pageSize}, nil
}

func (s *SQLStore) MarkProjectDataCollectionTaskSucceeded(ctx context.Context, task ProjectDataCollectionTask) (bool, error) {
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
	if task.DataType == ProjectDataCollectionTypeContractCodeSource {
		_, e = q.CompleteProjectDataCollectionSchedule(ctx, tokensqlc.CompleteProjectDataCollectionScheduleParams{LastCheckedAt: nullableTime(now), ProjectID: task.ProjectID, DataType: task.DataType})
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

func (s *SQLStore) MarkProjectDataCollectionTaskFailed(ctx context.Context, task ProjectDataCollectionTask, lastError string) (*ProjectDataCollectionTask, bool, error) {
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

func (s *SQLStore) CompleteProjectAveDataCollection(ctx context.Context, task ProjectDataCollectionTask, payload json.RawMessage, checkedAt time.Time) (*ProjectObservation, error) {
	return s.completeObservation(ctx, task, payload, nil, checkedAt, false)
}
func (s *SQLStore) CompleteProjectChainStateCollection(ctx context.Context, task ProjectDataCollectionTask, payload json.RawMessage, blockNumber uint64, checkedAt time.Time) (*ProjectObservation, error) {
	return s.completeObservation(ctx, task, payload, &blockNumber, checkedAt, false)
}
func (s *SQLStore) CompleteProjectWalletAssetStateCollection(ctx context.Context, task ProjectDataCollectionTask, states []WalletAssetState, blockNumber uint64, checkedAt time.Time) error {
	payload, e := json.Marshal(states)
	if e != nil {
		return e
	}
	_, e = s.completeObservation(ctx, task, payload, &blockNumber, checkedAt, false)
	return e
}
func (s *SQLStore) CompleteProjectSimulationResultCollection(ctx context.Context, task ProjectDataCollectionTask, results []ProjectSimulationResult, blockNumber uint64, checkedAt time.Time) error {
	payload, e := json.Marshal(results)
	if e != nil {
		return e
	}
	_, e = s.completeObservation(ctx, task, payload, &blockNumber, checkedAt, false)
	return e
}

func (s *SQLStore) CompleteProjectContractCodeSourceCollection(ctx context.Context, task ProjectDataCollectionTask, codeHash common.Hash, sourceCode string, checkedAt time.Time) error {
	if strings.TrimSpace(sourceCode) == "" {
		return s.completeUnavailableContractSource(ctx, task, checkedAt)
	}
	if s == nil || s.pool == nil {
		return fmt.Errorf("token postgres database is not configured")
	}
	tx, e := s.pool.Begin(ctx)
	if e != nil {
		return e
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := tokensqlc.New(tx)
	if _, e = q.UpdateContractCodeSource(ctx, tokensqlc.UpdateContractCodeSourceParams{CodeHash: codeHash.Bytes(), SourceCode: nullableText(sourceCode), SourceCodeFetchedAt: nullableTime(checkedAt)}); e != nil {
		return e
	}
	payload, _ := json.Marshal(map[string]any{"codeHash": codeHash.Hex(), "sourceAvailable": true})
	if _, e = recordObservation(ctx, q, task.ProjectID, task.DataType, payload, nil, checkedAt); e != nil {
		return e
	}
	if n, e := q.MarkProjectDataCollectionTaskSucceeded(ctx, task.ID); e != nil {
		return e
	} else if n == 0 {
		return nil
	}
	if _, e = q.CompleteProjectDataCollectionSchedule(ctx, tokensqlc.CompleteProjectDataCollectionScheduleParams{LastCheckedAt: nullableTime(checkedAt), ProjectID: task.ProjectID, DataType: task.DataType}); e != nil {
		return e
	}
	return tx.Commit(ctx)
}

func (s *SQLStore) completeUnavailableContractSource(ctx context.Context, task ProjectDataCollectionTask, checkedAt time.Time) error {
	if checkedAt.IsZero() {
		checkedAt = time.Now().UTC()
	}
	tx, e := s.pool.Begin(ctx)
	if e != nil {
		return e
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := tokensqlc.New(tx)
	n, e := q.MarkProjectDataCollectionTaskSucceeded(ctx, task.ID)
	if e != nil {
		return e
	}
	if n == 0 {
		return nil
	}
	if e = markScheduleSucceeded(ctx, q, task.ProjectID, task.DataType, checkedAt); e != nil {
		return e
	}
	return tx.Commit(ctx)
}

func (s *SQLStore) completeObservation(ctx context.Context, task ProjectDataCollectionTask, payload json.RawMessage, blockNumber *uint64, checkedAt time.Time, completeSchedule bool) (*ProjectObservation, error) {
	if checkedAt.IsZero() {
		checkedAt = time.Now().UTC()
	}
	tx, e := s.pool.Begin(ctx)
	if e != nil {
		return nil, e
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := tokensqlc.New(tx)
	obs, e := recordObservation(ctx, q, task.ProjectID, task.DataType, payload, blockNumber, checkedAt)
	if e != nil {
		return nil, e
	}
	n, e := q.MarkProjectDataCollectionTaskSucceeded(ctx, task.ID)
	if e != nil {
		return nil, e
	}
	if n == 0 {
		return nil, nil
	}
	if completeSchedule {
		_, e = q.CompleteProjectDataCollectionSchedule(ctx, tokensqlc.CompleteProjectDataCollectionScheduleParams{LastCheckedAt: nullableTime(checkedAt), ProjectID: task.ProjectID, DataType: task.DataType})
	} else {
		e = markScheduleSucceeded(ctx, q, task.ProjectID, task.DataType, checkedAt)
	}
	if e != nil {
		return nil, e
	}
	if e = tx.Commit(ctx); e != nil {
		return nil, e
	}
	return obs, nil
}

func recordObservation(ctx context.Context, q *tokensqlc.Queries, projectID int64, dataType string, payload json.RawMessage, blockNumber *uint64, checkedAt time.Time) (*ProjectObservation, error) {
	canonical, e := canonicalJSON(payload)
	if e != nil {
		return nil, fmt.Errorf("canonicalize %s observation: %w", dataType, e)
	}
	digest := sha256.Sum256(canonical)
	current, e := q.GetCurrentProjectObservation(ctx, tokensqlc.GetCurrentProjectObservationParams{ProjectID: projectID, DataType: dataType})
	if e == nil && bytes.Equal(current.ContentHash, digest[:]) {
		if _, e = q.TouchCurrentProjectObservation(ctx, tokensqlc.TouchCurrentProjectObservationParams{LastCheckedAt: nullableTime(checkedAt), ProjectID: projectID, DataType: dataType}); e != nil {
			return nil, e
		}
		v, e := mapCurrentProjectObservation(current)
		v.LastCheckedAt = checkedAt
		return &v, e
	}
	if e != nil && !errors.Is(e, pgx.ErrNoRows) {
		return nil, e
	}
	block, e := nullableUint64(blockNumber)
	if e != nil {
		return nil, e
	}
	row, e := q.InsertProjectObservation(ctx, tokensqlc.InsertProjectObservationParams{ProjectID: projectID, DataType: dataType, ContentHash: digest[:], Payload: canonical, BlockNumber: block, ObservedAt: nullableTime(checkedAt)})
	if e != nil {
		return nil, e
	}
	if _, e = q.UpsertCurrentProjectObservation(ctx, tokensqlc.UpsertCurrentProjectObservationParams{ProjectID: projectID, DataType: dataType, ObservationID: row.ID, LastCheckedAt: nullableTime(checkedAt)}); e != nil {
		return nil, e
	}
	state, e := q.IncrementProjectEvidenceRevision(ctx, projectID)
	if e != nil {
		return nil, e
	}
	if _, e = q.EnqueueProjectReportBuildTask(ctx, tokensqlc.EnqueueProjectReportBuildTaskParams{ProjectID: projectID, EvidenceRevision: state.EvidenceRevision}); e != nil {
		return nil, e
	}
	v, e := mapProjectObservation(row)
	v.LastCheckedAt = checkedAt
	return &v, e
}

func canonicalJSON(payload []byte) ([]byte, error) {
	var value any
	if e := json.Unmarshal(payload, &value); e != nil {
		return nil, e
	}
	return json.Marshal(value)
}
func markScheduleSucceeded(ctx context.Context, q *tokensqlc.Queries, projectID int64, dataType string, checkedAt time.Time) error {
	schedule, e := q.GetProjectDataCollectionSchedule(ctx, tokensqlc.GetProjectDataCollectionScheduleParams{ProjectID: projectID, DataType: dataType})
	if e != nil {
		return e
	}
	_, e = q.MarkProjectDataCollectionScheduleSucceeded(ctx, tokensqlc.MarkProjectDataCollectionScheduleSucceededParams{LastCheckedAt: nullableTime(checkedAt), NextRunAt: nullableTime(checkedAt.Add(time.Duration(schedule.RefreshIntervalSeconds) * time.Second)), ProjectID: projectID, DataType: dataType})
	return e
}
func markScheduleFailed(ctx context.Context, q *tokensqlc.Queries, projectID int64, dataType, lastError string, failedAt time.Time) error {
	schedule, e := q.GetProjectDataCollectionSchedule(ctx, tokensqlc.GetProjectDataCollectionScheduleParams{ProjectID: projectID, DataType: dataType})
	if e != nil {
		return e
	}
	_, e = q.MarkProjectDataCollectionScheduleFailed(ctx, tokensqlc.MarkProjectDataCollectionScheduleFailedParams{LastError: nullableText(lastError), NextRunAt: nullableTime(failedAt.Add(time.Duration(schedule.RefreshIntervalSeconds) * time.Second)), ProjectID: projectID, DataType: dataType})
	return e
}
