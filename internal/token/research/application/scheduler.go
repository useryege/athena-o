package application

import (
	"context"
	"fmt"
	"time"

	"github.com/useryege/athena/internal/token/research"
)

type DueSchedule struct {
	ProjectID          int64
	DataType           research.DataCollectionType
	Status             research.DataCollectionScheduleStatus
	RefreshInterval    time.Duration
	NextRunAt          time.Time
	LatestTaskRevision int64
}

type CreateCollectionTaskCommand struct {
	ProjectID        int64
	DataType         research.DataCollectionType
	ExpectedRevision int64
	TaskRevision     int64
	Now              time.Time
	NextRunAt        time.Time
}

type SchedulerRepository interface {
	ApplyResearchPolicy(context.Context, map[research.DataCollectionType]time.Duration, time.Duration) error
	MaintainResearchLifecycle(context.Context) error
	ListDueCollectionSchedules(context.Context, int32) ([]DueSchedule, error)
	CreateCollectionTaskIfDue(context.Context, CreateCollectionTaskCommand) (bool, error)
}

type SchedulerOptions struct {
	Intervals map[research.DataCollectionType]time.Duration
	TTL       time.Duration
	Limit     int32
	Now       func() time.Time
}

type Scheduler struct {
	repository SchedulerRepository
	options    SchedulerOptions
}

func NewScheduler(repository SchedulerRepository, options SchedulerOptions) *Scheduler {
	if options.Intervals == nil {
		options.Intervals = map[research.DataCollectionType]time.Duration{
			research.DataCollectionTypeChainState:         15 * time.Second,
			research.DataCollectionTypeWalletAssetState:   time.Minute,
			research.DataCollectionTypeSimulationResult:   time.Minute,
			research.DataCollectionTypeAve:                5 * time.Minute,
			research.DataCollectionTypeContractCodeSource: 10 * time.Minute,
		}
	}
	if options.TTL <= 0 {
		options.TTL = 168 * time.Hour
	}
	if options.Limit <= 0 {
		options.Limit = 100
	}
	if options.Now == nil {
		options.Now = time.Now
	}
	return &Scheduler{repository: repository, options: options}
}

func (scheduler *Scheduler) Initialize(ctx context.Context) error {
	if scheduler == nil || scheduler.repository == nil {
		return fmt.Errorf("token scheduler application is not configured")
	}
	return scheduler.repository.ApplyResearchPolicy(ctx, scheduler.options.Intervals, scheduler.options.TTL)
}

func (scheduler *Scheduler) RunOnce(ctx context.Context) (int, error) {
	if scheduler == nil || scheduler.repository == nil {
		return 0, fmt.Errorf("token scheduler application is not configured")
	}
	if err := scheduler.repository.MaintainResearchLifecycle(ctx); err != nil {
		return 0, err
	}
	schedules, err := scheduler.repository.ListDueCollectionSchedules(ctx, scheduler.options.Limit)
	if err != nil {
		return 0, err
	}
	created := 0
	for _, schedule := range schedules {
		now := scheduler.options.Now().UTC()
		if schedule.Status != research.DataCollectionScheduleStatusActive || schedule.NextRunAt.After(now) {
			continue
		}
		applied, err := scheduler.repository.CreateCollectionTaskIfDue(ctx, CreateCollectionTaskCommand{
			ProjectID: schedule.ProjectID, DataType: schedule.DataType, ExpectedRevision: schedule.LatestTaskRevision,
			TaskRevision: schedule.LatestTaskRevision + 1, Now: now, NextRunAt: now.Add(schedule.RefreshInterval),
		})
		if err != nil {
			return created, err
		}
		if applied {
			created++
		}
	}
	return created, nil
}
