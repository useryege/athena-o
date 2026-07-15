package store

import (
	"context"
	"fmt"
	"time"

	"github.com/useryege/athena/internal/token/domain"
	tokensqlc "github.com/useryege/athena/internal/token/store/sqlc"
)

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
		if locked.Status != string(domain.DataCollectionScheduleStatusActive) || timeValue(locked.NextRunAt).After(time.Now().UTC()) {
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

func (s *SQLStore) ApplyResearchPolicy(ctx context.Context, intervals map[domain.DataCollectionType]time.Duration, ttl time.Duration) error {
	q, err := s.querier()
	if err != nil {
		return err
	}
	for dataType, interval := range intervals {
		if _, err = q.ApplyProjectDataCollectionSchedulePolicy(ctx, tokensqlc.ApplyProjectDataCollectionSchedulePolicyParams{RefreshIntervalSeconds: int64(interval / time.Second), DataType: string(dataType)}); err != nil {
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

func markScheduleSucceeded(ctx context.Context, q *tokensqlc.Queries, projectID int64, dataType domain.DataCollectionType, checkedAt time.Time) error {
	schedule, e := q.GetProjectDataCollectionSchedule(ctx, tokensqlc.GetProjectDataCollectionScheduleParams{ProjectID: projectID, DataType: string(dataType)})
	if e != nil {
		return e
	}
	_, e = q.MarkProjectDataCollectionScheduleSucceeded(ctx, tokensqlc.MarkProjectDataCollectionScheduleSucceededParams{LastCheckedAt: nullableTime(checkedAt), NextRunAt: nullableTime(checkedAt.Add(time.Duration(schedule.RefreshIntervalSeconds) * time.Second)), ProjectID: projectID, DataType: string(dataType)})
	return e
}
func markScheduleFailed(ctx context.Context, q *tokensqlc.Queries, projectID int64, dataType domain.DataCollectionType, lastError string, failedAt time.Time) error {
	schedule, e := q.GetProjectDataCollectionSchedule(ctx, tokensqlc.GetProjectDataCollectionScheduleParams{ProjectID: projectID, DataType: string(dataType)})
	if e != nil {
		return e
	}
	_, e = q.MarkProjectDataCollectionScheduleFailed(ctx, tokensqlc.MarkProjectDataCollectionScheduleFailedParams{LastError: nullableText(lastError), NextRunAt: nullableTime(failedAt.Add(time.Duration(schedule.RefreshIntervalSeconds) * time.Second)), ProjectID: projectID, DataType: string(dataType)})
	return e
}
