package projectselectionevaluator

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/useryege/athena/internal/token/domain"
	"github.com/useryege/athena/internal/token/telemetry"
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
	strategies map[strategyIdentity]Strategy
	active     strategyIdentity
}

type strategyIdentity struct {
	key     string
	version string
}

func NewRegistry(activeKey, activeVersion string, strategies ...Strategy) (*Registry, error) {
	r := &Registry{strategies: make(map[strategyIdentity]Strategy)}
	if err := r.Register(NotConfiguredStrategy{}); err != nil {
		return nil, err
	}
	for _, strategy := range strategies {
		if err := r.Register(strategy); err != nil {
			return nil, err
		}
	}
	active := strategyIdentity{key: strings.TrimSpace(activeKey), version: strings.TrimSpace(activeVersion)}
	if active.key == "" || active.version == "" {
		return nil, fmt.Errorf("token selection active strategy key and version are required")
	}
	if _, exists := r.strategies[active]; !exists {
		return nil, fmt.Errorf("token selection strategy %s/%s is not registered", active.key, active.version)
	}
	r.active = active
	return r, nil
}
func (r *Registry) Register(strategy Strategy) error {
	if strategy == nil {
		return fmt.Errorf("token selection strategy is required")
	}
	identity := strategyIdentity{key: strings.TrimSpace(strategy.Key()), version: strings.TrimSpace(strategy.Version())}
	if identity.key == "" || identity.version == "" {
		return fmt.Errorf("token selection strategy key and version are required")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.strategies[identity]; exists {
		return fmt.Errorf("token selection strategy %s/%s is already registered", identity.key, identity.version)
	}
	r.strategies[identity] = strategy
	return nil
}
func (r *Registry) Active() (Strategy, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	strategy := r.strategies[r.active]
	if strategy == nil {
		return nil, fmt.Errorf("token selection strategy %s/%s is not registered", r.active.key, r.active.version)
	}
	return strategy, nil
}

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
	Telemetry    telemetry.Reporter
}
type Worker struct {
	opts   Options
	mu     sync.Mutex
	cancel context.CancelFunc
	done   chan struct{}
}

func NewWorker(opts Options) *Worker {
	if opts.Registry == nil {
		opts.Registry, _ = NewRegistry("default", "1")
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
	if _, err := w.opts.Registry.Active(); err != nil {
		return err
	}
	telemetry.Register(w.opts.Telemetry, telemetry.Scope{Component: "selection_evaluator"})
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
	scope := telemetry.Scope{Component: "selection_evaluator"}
	telemetry.Register(w.opts.Telemetry, scope)
	ticker := time.NewTicker(w.opts.PollInterval)
	defer ticker.Stop()
	for {
		tasks, err := w.opts.Store.ClaimProjectSelectionEvaluationTasks(ctx, w.opts.TaskLimit)
		if err != nil {
			telemetry.Failure(w.opts.Telemetry, scope, err)
			log.WithError(err).Error("token selection evaluator claim failed")
		} else {
			for _, task := range tasks {
				w.process(ctx, task)
			}
			telemetry.Success(w.opts.Telemetry, scope)
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
	strategy, err := w.opts.Registry.Active()
	if err != nil {
		_ = w.opts.Store.MarkProjectSelectionEvaluationTaskFailed(ctx, task, err.Error())
		return
	}
	decision, err := strategy.Evaluate(ctx, domain.StrategyInput{ProjectID: report.ProjectID, ReportRevision: report.Revision, Report: report.Report})
	if err != nil {
		_ = w.opts.Store.MarkProjectSelectionEvaluationTaskFailed(ctx, task, err.Error())
		return
	}
	decision, err = normalizeSelectionDecision(decision)
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

var reasonCodePattern = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

func normalizeSelectionDecision(decision domain.SelectionDecision) (domain.SelectionDecision, error) {
	switch decision.Outcome {
	case domain.SelectionOutcomeSelected, domain.SelectionOutcomeRejected, domain.SelectionOutcomeDeferred:
	default:
		return domain.SelectionDecision{}, fmt.Errorf("token selection strategy returned invalid outcome %q", decision.Outcome)
	}
	seen := make(map[string]struct{}, len(decision.ReasonCodes))
	reasonCodes := make([]string, 0, len(decision.ReasonCodes))
	for _, reasonCode := range decision.ReasonCodes {
		reasonCode = strings.TrimSpace(reasonCode)
		if !reasonCodePattern.MatchString(reasonCode) {
			return domain.SelectionDecision{}, fmt.Errorf("token selection strategy returned invalid reason code %q", reasonCode)
		}
		if _, exists := seen[reasonCode]; exists {
			continue
		}
		seen[reasonCode] = struct{}{}
		reasonCodes = append(reasonCodes, reasonCode)
	}
	if len(reasonCodes) == 0 {
		return domain.SelectionDecision{}, fmt.Errorf("token selection strategy must return at least one reason code")
	}
	sort.Strings(reasonCodes)
	decision.ReasonCodes = reasonCodes
	decision.ReasonDetail = strings.TrimSpace(decision.ReasonDetail)
	return decision, nil
}
