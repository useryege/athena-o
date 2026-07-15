package postgres

import (
	"context"
	"fmt"
	"time"

	tokensqlc "github.com/useryege/athena/internal/token/adapters/postgres/sqlc"
	"github.com/useryege/athena/internal/token/research"
	researchapp "github.com/useryege/athena/internal/token/research/application"
)

func (repository *SchedulerRepository) ApplyResearchPolicy(ctx context.Context, intervals map[research.DataCollectionType]time.Duration, ttl time.Duration) error {
	queries, err := repository.querier()
	if err != nil {
		return err
	}
	for dataType, interval := range intervals {
		if _, err = queries.ApplyProjectDataCollectionSchedulePolicy(ctx, tokensqlc.ApplyProjectDataCollectionSchedulePolicyParams{RefreshIntervalSeconds: int64(interval / time.Second), DataType: string(dataType)}); err != nil {
			return fmt.Errorf("apply %s schedule policy: %w", dataType, err)
		}
	}
	if _, err = queries.ApplyProjectResearchTTL(ctx, int64(ttl/time.Second)); err != nil {
		return fmt.Errorf("apply research ttl: %w", err)
	}
	return nil
}

func (repository *SchedulerRepository) MaintainResearchLifecycle(ctx context.Context) error {
	queries, err := repository.querier()
	if err != nil {
		return err
	}
	if _, err = queries.ExpireProjectResearchStates(ctx); err != nil {
		return err
	}
	_, err = queries.PauseTerminalProjectDataCollectionSchedules(ctx)
	return err
}

func (repository *SchedulerRepository) ListDueCollectionSchedules(ctx context.Context, limit int32) ([]researchapp.DueSchedule, error) {
	queries, err := repository.querier()
	if err != nil {
		return nil, err
	}
	rows, err := queries.ListDueProjectDataCollectionSchedules(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("list due collection schedules: %w", err)
	}
	result := make([]researchapp.DueSchedule, 0, len(rows))
	for _, row := range rows {
		result = append(result, researchapp.DueSchedule{ProjectID: row.ProjectID, DataType: research.DataCollectionType(row.DataType), Status: research.DataCollectionScheduleStatus(row.Status), RefreshInterval: time.Duration(row.RefreshIntervalSeconds) * time.Second, NextRunAt: timeValue(row.NextRunAt), LatestTaskRevision: row.LatestTaskRevision})
	}
	return result, nil
}

func (repository *SchedulerRepository) CreateCollectionTaskIfDue(ctx context.Context, command researchapp.CreateCollectionTaskCommand) (bool, error) {
	if repository == nil || repository.pool == nil {
		return false, fmt.Errorf("token scheduler repository is not configured")
	}
	tx, err := repository.pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := tokensqlc.New(tx)
	locked, err := queries.LockProjectDataCollectionSchedule(ctx, tokensqlc.LockProjectDataCollectionScheduleParams{ProjectID: command.ProjectID, DataType: string(command.DataType)})
	if err != nil {
		return false, fmt.Errorf("lock collection schedule: %w", err)
	}
	if locked.Status != string(research.DataCollectionScheduleStatusActive) || locked.LatestTaskRevision != command.ExpectedRevision || timeValue(locked.NextRunAt).After(command.Now) {
		return false, nil
	}
	if _, err = queries.CreateProjectDataCollectionTask(ctx, tokensqlc.CreateProjectDataCollectionTaskParams{ProjectID: command.ProjectID, DataType: string(command.DataType), Revision: command.TaskRevision}); err != nil {
		return false, fmt.Errorf("create collection task: %w", err)
	}
	if _, err = queries.AdvanceProjectDataCollectionSchedule(ctx, tokensqlc.AdvanceProjectDataCollectionScheduleParams{LatestTaskRevision: command.TaskRevision, NextRunAt: nullableTime(command.NextRunAt), ProjectID: command.ProjectID, DataType: string(command.DataType)}); err != nil {
		return false, fmt.Errorf("advance collection schedule: %w", err)
	}
	if err = tx.Commit(ctx); err != nil {
		return false, err
	}
	return true, nil
}

func markScheduleSucceeded(ctx context.Context, queries *tokensqlc.Queries, projectID int64, dataType research.DataCollectionType, checkedAt, nextRunAt time.Time) error {
	_, err := queries.MarkProjectDataCollectionScheduleSucceeded(ctx, tokensqlc.MarkProjectDataCollectionScheduleSucceededParams{LastCheckedAt: nullableTime(checkedAt), NextRunAt: nullableTime(nextRunAt), ProjectID: projectID, DataType: string(dataType)})
	return err
}

func markScheduleFailed(ctx context.Context, queries *tokensqlc.Queries, projectID int64, dataType research.DataCollectionType, lastError string, nextRunAt time.Time) error {
	_, err := queries.MarkProjectDataCollectionScheduleFailed(ctx, tokensqlc.MarkProjectDataCollectionScheduleFailedParams{LastError: nullableText(lastError), NextRunAt: nullableTime(nextRunAt), ProjectID: projectID, DataType: string(dataType)})
	return err
}
