package application

import (
	"context"
	"fmt"
	"time"

	"github.com/useryege/athena/internal/token/reporting"
)

type ReportBuildRepository interface {
	ClaimReportBuildTasks(context.Context, int32) ([]reporting.ProjectReportBuildTask, error)
	ListObservationSnapshots(context.Context, int64) ([]reporting.ObservationSnapshot, error)
	CommitReportAndEnqueueSelection(context.Context, CommitReportCommand) (*reporting.ProjectReportRevision, bool, error)
	RetryReportBuildTask(context.Context, RetryReportTaskCommand) error
	FailReportBuildTask(context.Context, FailReportTaskCommand) error
}

type CommitReportCommand struct {
	Task   reporting.ProjectReportBuildTask
	Report reporting.ProjectReportRevision
}

type RetryReportTaskCommand struct {
	Task        reporting.ProjectReportBuildTask
	LastError   string
	AvailableAt time.Time
}

type FailReportTaskCommand struct {
	Task      reporting.ProjectReportBuildTask
	LastError string
	FailedAt  time.Time
}

type BuilderOptions struct {
	TaskLimit int32
	Now       func() time.Time
}

type Builder struct {
	repository ReportBuildRepository
	options    BuilderOptions
}

func NewBuilder(repository ReportBuildRepository, options BuilderOptions) *Builder {
	if options.TaskLimit <= 0 {
		options.TaskLimit = 20
	}
	if options.Now == nil {
		options.Now = time.Now
	}
	return &Builder{repository: repository, options: options}
}

func (builder *Builder) RunOnce(ctx context.Context) (int, error) {
	if builder == nil || builder.repository == nil {
		return 0, fmt.Errorf("token report builder application is not configured")
	}
	tasks, err := builder.repository.ClaimReportBuildTasks(ctx, builder.options.TaskLimit)
	if err != nil {
		return 0, err
	}
	processed := 0
	for _, task := range tasks {
		if err := builder.process(ctx, task); err != nil {
			return processed, err
		}
		processed++
	}
	return processed, nil
}

func (builder *Builder) process(ctx context.Context, task reporting.ProjectReportBuildTask) error {
	observations, err := builder.repository.ListObservationSnapshots(ctx, task.ProjectID)
	if err != nil {
		return builder.fail(ctx, task, err)
	}
	report, err := reporting.BuildReport(task.ProjectID, observations, builder.options.Now().UTC())
	if err != nil {
		return builder.fail(ctx, task, err)
	}
	if _, _, err = builder.repository.CommitReportAndEnqueueSelection(ctx, CommitReportCommand{Task: task, Report: report}); err != nil {
		return builder.fail(ctx, task, err)
	}
	return nil
}

func (builder *Builder) fail(ctx context.Context, task reporting.ProjectReportBuildTask, failure error) error {
	failedAt := builder.options.Now().UTC()
	backoff := time.Duration(1<<min(int(task.Attempts), 4)) * time.Second
	if task.Attempts < 4 {
		return builder.repository.RetryReportBuildTask(ctx, RetryReportTaskCommand{Task: task, LastError: failure.Error(), AvailableAt: failedAt.Add(backoff)})
	}
	return builder.repository.FailReportBuildTask(ctx, FailReportTaskCommand{Task: task, LastError: failure.Error(), FailedAt: failedAt})
}
