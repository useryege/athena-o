package workerhost

import (
	"context"
	"fmt"
	"os/signal"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/useryege/athena/internal/token/telemetry"
)

type Worker interface {
	Start(context.Context) error
	Stop(context.Context) error
}

type QueueMetric struct {
	Queue             string
	Status            string
	Count             int64
	OldestAvailableAt time.Time
}

type Options struct {
	Name                string
	HealthListenAddress string
	HealthStaleAfter    time.Duration
	Worker              Worker
	Ping                func(context.Context) error
	Diagnostics         func(context.Context) ([]QueueMetric, error)
	Close               func() error
}

type Host struct {
	opts    Options
	tracker *telemetry.Tracker
	health  *telemetry.Server
}

func New(opts Options) (*Host, telemetry.Reporter, error) {
	if opts.Name == "" {
		return nil, nil, fmt.Errorf("token worker host name is required")
	}
	if opts.HealthListenAddress == "" {
		return nil, nil, fmt.Errorf("token worker host health listen address is required")
	}
	tracker := telemetry.NewTracker(opts.Name, opts.HealthStaleAfter)
	return &Host{opts: opts, tracker: tracker, health: telemetry.NewServer(opts.HealthListenAddress, tracker, opts.Ping)}, tracker, nil
}

func (h *Host) SetWorker(worker Worker) {
	h.opts.Worker = worker
}

func (h *Host) AddClose(closeFn func() error) {
	if closeFn == nil {
		return
	}
	previous := h.opts.Close
	h.opts.Close = func() error {
		var result error
		if previous != nil {
			result = previous()
		}
		if err := closeFn(); err != nil && result == nil {
			result = err
		}
		return result
	}
}

func (h *Host) Reporter() telemetry.Reporter {
	return h.tracker
}

func (h *Host) Run(parent context.Context) error {
	if h.opts.Worker == nil {
		return fmt.Errorf("token worker host worker is required")
	}
	ctx, stopSignal := signal.NotifyContext(parent, syscall.SIGINT, syscall.SIGTERM)
	defer stopSignal()
	if err := h.health.Start(); err != nil {
		return err
	}
	if err := h.opts.Worker.Start(ctx); err != nil {
		_ = h.stopResources()
		return err
	}
	h.tracker.SetRunning(true)
	diagnosticsDone := h.startDiagnostics(ctx)
	<-ctx.Done()
	log.WithField("worker", h.opts.Name).Info("stopping token worker")
	if diagnosticsDone != nil {
		<-diagnosticsDone
	}
	return h.stopResources()
}

func (h *Host) startDiagnostics(ctx context.Context) <-chan struct{} {
	if h.opts.Diagnostics == nil {
		return nil
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()
		for {
			h.refreshDiagnostics(ctx)
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
	return done
}

func (h *Host) refreshDiagnostics(parent context.Context) {
	ctx, cancel := context.WithTimeout(parent, 2*time.Second)
	metrics, err := h.opts.Diagnostics(ctx)
	cancel()
	if err != nil {
		h.tracker.SetDiagnosticsSuccess(false)
		return
	}
	h.tracker.ResetQueues()
	for _, metric := range metrics {
		h.tracker.UpdateQueue(metric.Queue, metric.Status, metric.Count, metric.OldestAvailableAt)
	}
	h.tracker.SetDiagnosticsSuccess(true)
}

func (h *Host) stopResources() error {
	h.tracker.SetRunning(false)
	var result error
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	if h.opts.Worker != nil {
		result = h.opts.Worker.Stop(ctx)
	}
	if err := h.health.Stop(ctx); err != nil && result == nil {
		result = err
	}
	cancel()
	if h.opts.Close != nil {
		if err := h.opts.Close(); err != nil && result == nil {
			result = err
		}
	}
	return result
}
