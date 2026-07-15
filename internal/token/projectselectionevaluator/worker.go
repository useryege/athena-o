package projectselectionevaluator

import (
	"context"
	"fmt"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/useryege/athena/internal/token/domain"
)

type Store interface {
	ClaimProjectSelectionEvaluationTasks(context.Context, int32) ([]domain.ProjectSelectionEvaluationTask, error)
	GetProjectReportRevision(context.Context, int64, int64) (*domain.ProjectReportRevision, error)
	CompleteProjectSelectionEvaluation(context.Context, domain.ProjectSelectionEvaluationTask, domain.ProjectSelection) (*domain.ProjectSelection, bool, error)
	MarkProjectSelectionEvaluationTaskFailed(context.Context, domain.ProjectSelectionEvaluationTask, string) error
}

type Strategy interface {
	Key() string
	Version() string
	Evaluate(context.Context, domain.StrategyInput) (domain.SelectionDecision, error)
}
type Registry struct {
	mu         sync.RWMutex
	strategies []Strategy
}

func NewRegistry(strategies ...Strategy) *Registry {
	r := &Registry{}
	for _, strategy := range strategies {
		r.Register(strategy)
	}
	if len(r.strategies) == 0 {
		r.Register(NotConfiguredStrategy{})
	}
	return r
}
func (r *Registry) Register(strategy Strategy) {
	if strategy == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.strategies = append(r.strategies, strategy)
}
func (r *Registry) Active() Strategy { r.mu.RLock(); defer r.mu.RUnlock(); return r.strategies[0] }

type NotConfiguredStrategy struct{}

func (NotConfiguredStrategy) Key() string     { return "default" }
func (NotConfiguredStrategy) Version() string { return "1" }
func (NotConfiguredStrategy) Evaluate(context.Context, domain.StrategyInput) (domain.SelectionDecision, error) {
	return domain.SelectionDecision{Outcome: domain.SelectionOutcomeDeferred, ReasonCodes: []string{"strategy_not_configured"}}, nil
}

type Options struct {
	Store        Store
	Registry     *Registry
	PollInterval time.Duration
	TaskLimit    int32
}
type Worker struct {
	opts   Options
	mu     sync.Mutex
	cancel context.CancelFunc
	done   chan struct{}
}

func NewWorker(opts Options) *Worker {
	if opts.Registry == nil {
		opts.Registry = NewRegistry()
	}
	if opts.PollInterval <= 0 {
		opts.PollInterval = time.Second
	}
	if opts.TaskLimit <= 0 {
		opts.TaskLimit = 20
	}
	return &Worker{opts: opts}
}
func (w *Worker) Start(ctx context.Context) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.cancel != nil {
		return nil
	}
	if w.opts.Store == nil {
		return fmt.Errorf("token selection evaluator store is required")
	}
	runCtx, cancel := context.WithCancel(ctx)
	w.cancel = cancel
	w.done = make(chan struct{})
	go w.run(runCtx)
	return nil
}
func (w *Worker) Stop(ctx context.Context) error {
	w.mu.Lock()
	cancel, done := w.cancel, w.done
	w.cancel = nil
	w.done = nil
	w.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	if done != nil {
		select {
		case <-done:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}
func (w *Worker) run(ctx context.Context) {
	defer close(w.done)
	ticker := time.NewTicker(w.opts.PollInterval)
	defer ticker.Stop()
	for {
		tasks, err := w.opts.Store.ClaimProjectSelectionEvaluationTasks(ctx, w.opts.TaskLimit)
		if err != nil {
			log.WithError(err).Error("token selection evaluator claim failed")
		} else {
			for _, task := range tasks {
				w.process(ctx, task)
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
func (w *Worker) process(ctx context.Context, task domain.ProjectSelectionEvaluationTask) {
	report, err := w.opts.Store.GetProjectReportRevision(ctx, task.ProjectID, task.ReportRevision)
	if err != nil || report == nil {
		if err == nil {
			err = fmt.Errorf("report revision not found")
		}
		_ = w.opts.Store.MarkProjectSelectionEvaluationTaskFailed(ctx, task, err.Error())
		return
	}
	strategy := w.opts.Registry.Active()
	decision, err := strategy.Evaluate(ctx, domain.StrategyInput{ProjectID: report.ProjectID, ReportRevision: report.Revision, Report: report.Report})
	if err != nil {
		_ = w.opts.Store.MarkProjectSelectionEvaluationTaskFailed(ctx, task, err.Error())
		return
	}
	selection := domain.ProjectSelection{ProjectID: task.ProjectID, Outcome: decision.Outcome, StrategyKey: strategy.Key(), StrategyVersion: strategy.Version(), ReportRevision: task.ReportRevision, ReasonCodes: decision.ReasonCodes, ReasonDetail: decision.ReasonDetail, DecidedAt: time.Now().UTC()}
	_, created, err := w.opts.Store.CompleteProjectSelectionEvaluation(ctx, task, selection)
	if err != nil {
		_ = w.opts.Store.MarkProjectSelectionEvaluationTaskFailed(ctx, task, err.Error())
		return
	}
	log.WithFields(log.Fields{"project_id": task.ProjectID, "report_revision": task.ReportRevision, "outcome": decision.Outcome, "created": created}).Info("token project selection evaluation completed")
}
