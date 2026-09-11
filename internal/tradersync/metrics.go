package tradersync

import (
	tm "github.com/useryege/athena/internal/tradersync/types"
	"strconv"
	"sync"
	"time"
)

type TimingSample struct {
	PublicEarliest, PublicLatest   *time.Time
	ReceivedAt, RecordedAt         time.Time
	AuthorizedAt, StartedAt, AckAt *time.Time
	Cohort, Outcome, ClockSource   string
	ClockUncertainty               time.Duration
}
type TimingResult struct {
	PublicAssessable                 bool
	Lower, Upper, ReceivedToRecorded time.Duration
	Reason                           string
}

// EvaluateTiming preserves measured values; unknown public time is never inferred
// from block or receipt time. Bounds retain uncertainty rather than choosing a percentile verdict.
func EvaluateTiming(sample TimingSample) TimingResult {
	r := TimingResult{ReceivedToRecorded: sample.RecordedAt.Sub(sample.ReceivedAt)}
	if sample.ReceivedAt.IsZero() || sample.RecordedAt.IsZero() {
		r.Reason = "local_time_missing"
		return r
	}
	if r.ReceivedToRecorded < 0 {
		r.Reason = "received_recorded_clock_anomaly"
		return r
	}
	if sample.PublicEarliest == nil || sample.PublicLatest == nil || sample.ClockSource == "" || sample.ClockSource == "untrusted" {
		r.Reason = "public_time_or_clock_unverified"
		return r
	}
	if sample.ClockUncertainty < 0 {
		r.Reason = "clock_uncertainty_invalid"
		return r
	}
	if sample.PublicEarliest.IsZero() || sample.PublicLatest.IsZero() || sample.PublicLatest.Before(*sample.PublicEarliest) {
		r.Reason = "public_interval_invalid"
		return r
	}
	r.Lower = sample.RecordedAt.Sub(*sample.PublicLatest) - sample.ClockUncertainty
	r.Upper = sample.RecordedAt.Sub(*sample.PublicEarliest) + sample.ClockUncertainty
	if sample.RecordedAt.Before(*sample.PublicLatest) {
		r.Reason = "public_recorded_clock_anomaly"
		return r
	}
	r.PublicAssessable = true
	return r
}

// Bounded, process-local counters include failed/cancelled rounds. Persistent
// activity/delivery/attempt totals are queried separately; these are not cohorts.
type projectorMetrics struct {
	mu     sync.Mutex
	stages map[string]phaseMetric
	active int64
}
type phaseMetric struct {
	count   int64
	elapsed time.Duration
}

var projectorStages = []string{"source_round", "confirmation_round", "version_round", "metadata_extra_wait", "candidate_transaction"}

func (m *projectorMetrics) observe(stage string, began time.Time) {
	elapsed := time.Since(began)
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.stages == nil {
		m.stages = map[string]phaseMetric{}
	}
	v := m.stages[stage]
	v.count++
	v.elapsed += elapsed
	m.stages[stage] = v
}
func (m *projectorMetrics) inFlight(delta int64) { m.mu.Lock(); m.active += delta; m.mu.Unlock() }
func (m *projectorMetrics) snapshot() []tm.RuntimeMetric {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := []tm.RuntimeMetric{{Name: "projector_sources_in_flight", Value: strconv.FormatInt(m.active, 10), Unit: "sources", Kind: "gauge"}}
	for _, name := range projectorStages {
		v := m.stages[name]
		out = append(out,
			tm.RuntimeMetric{Name: "projector_" + name + "_count", Value: strconv.FormatInt(v.count, 10), Unit: "rounds", Kind: "counter"},
			tm.RuntimeMetric{Name: "projector_" + name + "_elapsed_ns_total", Value: strconv.FormatInt(v.elapsed.Nanoseconds(), 10), Unit: "nanoseconds_sender_independent_monotonic", Kind: "counter"})
	}
	return out
}
