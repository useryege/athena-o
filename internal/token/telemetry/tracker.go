package telemetry

import (
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type Scope struct {
	Component string `json:"component"`
	DataType  string `json:"dataType,omitempty"`
	ChainID   int64  `json:"chainId,omitempty"`
}

type Reporter interface {
	Register(Scope)
	RecordSuccess(Scope)
	RecordFailure(Scope, error)
}

type ScopeStatus struct {
	Scope               Scope     `json:"scope"`
	Status              string    `json:"status"`
	LastSuccessAt       time.Time `json:"lastSuccessAt,omitempty"`
	LastErrorAt         time.Time `json:"lastErrorAt,omitempty"`
	LastError           string    `json:"lastError,omitempty"`
	ConsecutiveFailures int64     `json:"consecutiveFailures"`
}

type scopeState struct {
	lastSuccessAt       time.Time
	lastErrorAt         time.Time
	lastError           string
	consecutiveFailures int64
}

type Tracker struct {
	mode       string
	staleAfter time.Duration

	mu      sync.RWMutex
	running bool
	states  map[Scope]*scopeState

	registry            *prometheus.Registry
	lastSuccess         *prometheus.GaugeVec
	lastError           *prometheus.GaugeVec
	consecutiveFailures *prometheus.GaugeVec
	runs                *prometheus.CounterVec
	queueItems          *prometheus.GaugeVec
	queueOldestAge      *prometheus.GaugeVec
	diagnosticsSuccess  prometheus.Gauge
}

func NewTracker(mode string, staleAfter time.Duration) *Tracker {
	if staleAfter <= 0 {
		staleAfter = 2 * time.Minute
	}
	labels := []string{"mode", "component", "data_type", "chain_id"}
	t := &Tracker{
		mode:       mode,
		staleAfter: staleAfter,
		states:     make(map[Scope]*scopeState),
		registry:   prometheus.NewRegistry(),
		lastSuccess: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "athena_token_loop_last_success_timestamp_seconds",
			Help: "Unix timestamp of the last successful Token loop iteration.",
		}, labels),
		lastError: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "athena_token_loop_last_error_timestamp_seconds",
			Help: "Unix timestamp of the last failed Token loop iteration.",
		}, labels),
		consecutiveFailures: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "athena_token_loop_consecutive_failures",
			Help: "Current consecutive Token loop failure count.",
		}, labels),
		runs: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "athena_token_loop_runs_total",
			Help: "Token loop iterations by result.",
		}, append(labels, "result")),
		queueItems: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "athena_token_queue_items",
			Help: "Current Token pipeline queue item count.",
		}, []string{"mode", "queue", "status"}),
		queueOldestAge: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "athena_token_queue_oldest_available_age_seconds",
			Help: "Age in seconds of the oldest Token pipeline queue item.",
		}, []string{"mode", "queue", "status"}),
		diagnosticsSuccess: prometheus.NewGauge(prometheus.GaugeOpts{
			Name:        "athena_token_diagnostics_snapshot_success",
			Help:        "Whether the latest Token diagnostics database snapshot succeeded.",
			ConstLabels: prometheus.Labels{"mode": mode},
		}),
	}
	t.registry.MustRegister(t.lastSuccess, t.lastError, t.consecutiveFailures, t.runs, t.queueItems, t.queueOldestAge, t.diagnosticsSuccess)
	return t
}

func (t *Tracker) Registry() *prometheus.Registry { return t.registry }

func (t *Tracker) SetRunning(running bool) {
	t.mu.Lock()
	t.running = running
	t.mu.Unlock()
}

func (t *Tracker) IsRunning() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.running
}

func (t *Tracker) Register(scope Scope) {
	created := false
	t.mu.Lock()
	if t.states[scope] == nil {
		t.states[scope] = &scopeState{}
		created = true
	}
	t.mu.Unlock()
	if !created {
		return
	}
	labels := t.scopeLabels(scope)
	t.lastSuccess.WithLabelValues(labels...).Set(0)
	t.lastError.WithLabelValues(labels...).Set(0)
	t.consecutiveFailures.WithLabelValues(labels...).Set(0)
}

func (t *Tracker) RecordSuccess(scope Scope) {
	t.Register(scope)
	now := time.Now().UTC()
	t.mu.Lock()
	state := t.states[scope]
	state.lastSuccessAt = now
	state.consecutiveFailures = 0
	t.mu.Unlock()
	labels := t.scopeLabels(scope)
	t.lastSuccess.WithLabelValues(labels...).Set(float64(now.Unix()))
	t.consecutiveFailures.WithLabelValues(labels...).Set(0)
	t.runs.WithLabelValues(append(labels, "success")...).Inc()
}

func (t *Tracker) RecordFailure(scope Scope, err error) {
	t.Register(scope)
	now := time.Now().UTC()
	t.mu.Lock()
	state := t.states[scope]
	state.lastErrorAt = now
	state.consecutiveFailures++
	if err != nil {
		state.lastError = err.Error()
	}
	failures := state.consecutiveFailures
	t.mu.Unlock()
	labels := t.scopeLabels(scope)
	t.lastError.WithLabelValues(labels...).Set(float64(now.Unix()))
	t.consecutiveFailures.WithLabelValues(labels...).Set(float64(failures))
	t.runs.WithLabelValues(append(labels, "failure")...).Inc()
}

func (t *Tracker) ScopeStatuses(now time.Time) ([]ScopeStatus, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	statuses := make([]ScopeStatus, 0, len(t.states))
	ready := t.running
	for scope, state := range t.states {
		status := "healthy"
		if state.lastSuccessAt.IsZero() || now.Sub(state.lastSuccessAt) > t.staleAfter {
			status = "unhealthy"
			ready = false
		} else if state.consecutiveFailures > 0 {
			status = "degraded"
		}
		statuses = append(statuses, ScopeStatus{Scope: scope, Status: status, LastSuccessAt: state.lastSuccessAt, LastErrorAt: state.lastErrorAt, LastError: state.lastError, ConsecutiveFailures: state.consecutiveFailures})
	}
	sort.Slice(statuses, func(i, j int) bool {
		if statuses[i].Scope.Component != statuses[j].Scope.Component {
			return statuses[i].Scope.Component < statuses[j].Scope.Component
		}
		if statuses[i].Scope.DataType != statuses[j].Scope.DataType {
			return statuses[i].Scope.DataType < statuses[j].Scope.DataType
		}
		return statuses[i].Scope.ChainID < statuses[j].Scope.ChainID
	})
	return statuses, ready
}

func (t *Tracker) UpdateQueue(queue, status string, count int64, oldestAvailableAt time.Time) {
	t.queueItems.WithLabelValues(t.mode, queue, status).Set(float64(count))
	age := float64(0)
	if !oldestAvailableAt.IsZero() {
		age = time.Since(oldestAvailableAt).Seconds()
		if age < 0 {
			age = 0
		}
	}
	t.queueOldestAge.WithLabelValues(t.mode, queue, status).Set(age)
}

func (t *Tracker) ResetQueues() {
	t.queueItems.Reset()
	t.queueOldestAge.Reset()
}

func (t *Tracker) SetDiagnosticsSuccess(success bool) {
	if success {
		t.diagnosticsSuccess.Set(1)
		return
	}
	t.diagnosticsSuccess.Set(0)
}

func (t *Tracker) scopeLabels(scope Scope) []string {
	return []string{t.mode, scope.Component, scope.DataType, strconv.FormatInt(scope.ChainID, 10)}
}

func Register(reporter Reporter, scope Scope) {
	if reporter != nil {
		reporter.Register(scope)
	}
}

func Success(reporter Reporter, scope Scope) {
	if reporter != nil {
		reporter.RecordSuccess(scope)
	}
}

func Failure(reporter Reporter, scope Scope, err error) {
	if reporter != nil {
		reporter.RecordFailure(scope, err)
	}
}
