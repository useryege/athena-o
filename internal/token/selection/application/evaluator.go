package application

import (
	"context"
	"fmt"
	"time"

	"github.com/useryege/athena/internal/token/selection"
)

type SelectionRepository interface {
	ClaimSelectionTasks(context.Context, int32) ([]selection.ProjectSelectionEvaluationTask, error)
	GetReportSnapshot(context.Context, int64, int64) (*selection.ReportSnapshot, error)
	CommitSelectionAndCompleteTask(context.Context, CommitSelectionCommand) (*selection.ProjectSelection, bool, error)
	RetrySelectionTask(context.Context, RetrySelectionTaskCommand) error
	FailSelectionTask(context.Context, FailSelectionTaskCommand) error
}

type CommitSelectionCommand struct {
	Task      selection.ProjectSelectionEvaluationTask
	Selection selection.ProjectSelection
}

type RetrySelectionTaskCommand struct {
	Task        selection.ProjectSelectionEvaluationTask
	LastError   string
	AvailableAt time.Time
}

type FailSelectionTaskCommand struct {
	Task      selection.ProjectSelectionEvaluationTask
	LastError string
	FailedAt  time.Time
}

type EvaluatorOptions struct {
	TaskLimit int32
	Now       func() time.Time
}

type Evaluator struct {
	repository SelectionRepository
	registry   *selection.StrategyRegistry
	options    EvaluatorOptions
}

func NewEvaluator(repository SelectionRepository, registry *selection.StrategyRegistry, options EvaluatorOptions) *Evaluator {
	if registry == nil {
		registry, _ = selection.NewStrategyRegistry("default", "1")
	}
	if options.TaskLimit <= 0 {
		options.TaskLimit = 20
	}
	if options.Now == nil {
		options.Now = time.Now
	}
	return &Evaluator{repository: repository, registry: registry, options: options}
}

func (evaluator *Evaluator) RunOnce(ctx context.Context) (int, error) {
	if evaluator == nil || evaluator.repository == nil || evaluator.registry == nil {
		return 0, fmt.Errorf("token selection evaluator application is not configured")
	}
	tasks, err := evaluator.repository.ClaimSelectionTasks(ctx, evaluator.options.TaskLimit)
	if err != nil {
		return 0, err
	}
	processed := 0
	for _, task := range tasks {
		if err := evaluator.process(ctx, task); err != nil {
			return processed, err
		}
		processed++
	}
	return processed, nil
}

func (evaluator *Evaluator) process(ctx context.Context, task selection.ProjectSelectionEvaluationTask) error {
	report, err := evaluator.repository.GetReportSnapshot(ctx, task.ProjectID, task.ReportRevision)
	if err != nil {
		return evaluator.fail(ctx, task, err)
	}
	if report == nil {
		return evaluator.fail(ctx, task, fmt.Errorf("report revision not found"))
	}
	strategy, err := evaluator.registry.Active()
	if err != nil {
		return evaluator.fail(ctx, task, err)
	}
	decision, err := strategy.Evaluate(ctx, selection.StrategyInput{ProjectID: report.ProjectID, ReportRevision: report.Revision, Report: *report})
	if err != nil {
		return evaluator.fail(ctx, task, err)
	}
	decision, err = selection.NormalizeDecision(decision)
	if err != nil {
		return evaluator.fail(ctx, task, err)
	}
	item := selection.ProjectSelection{ProjectID: task.ProjectID, Outcome: decision.Outcome, StrategyKey: strategy.Key(), StrategyVersion: strategy.Version(), ReportRevision: task.ReportRevision, ReasonCodes: decision.ReasonCodes, ReasonDetail: decision.ReasonDetail, DecidedAt: evaluator.options.Now().UTC()}
	if _, _, err = evaluator.repository.CommitSelectionAndCompleteTask(ctx, CommitSelectionCommand{Task: task, Selection: item}); err != nil {
		return evaluator.fail(ctx, task, err)
	}
	return nil
}

func (evaluator *Evaluator) fail(ctx context.Context, task selection.ProjectSelectionEvaluationTask, failure error) error {
	failedAt := evaluator.options.Now().UTC()
	backoff := time.Duration(1<<min(int(task.Attempts), 4)) * time.Second
	if task.Attempts < 4 {
		return evaluator.repository.RetrySelectionTask(ctx, RetrySelectionTaskCommand{Task: task, LastError: failure.Error(), AvailableAt: failedAt.Add(backoff)})
	}
	return evaluator.repository.FailSelectionTask(ctx, FailSelectionTaskCommand{Task: task, LastError: failure.Error(), FailedAt: failedAt})
}
