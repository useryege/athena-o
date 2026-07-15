package workerhost

import (
	"context"
	"fmt"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/useryege/athena/internal/token/telemetry"
)

type JobResult struct{ Processed int }

type PeriodicJob struct {
	Name       string
	Interval   time.Duration
	Scope      telemetry.Scope
	Initialize func(context.Context) error
	Shutdown   func(context.Context) error
	RunOnce    func(context.Context) (JobResult, error)
}

type PeriodicWorker struct {
	jobs     []PeriodicJob
	reporter telemetry.Reporter
	mu       sync.Mutex
	cancel   context.CancelFunc
	done     chan struct{}
}

func NewPeriodicWorker(jobs []PeriodicJob, reporter telemetry.Reporter) *PeriodicWorker {
	return &PeriodicWorker{jobs: append([]PeriodicJob(nil), jobs...), reporter: reporter}
}

func (worker *PeriodicWorker) Start(ctx context.Context) error {
	worker.mu.Lock()
	defer worker.mu.Unlock()
	if worker.cancel != nil {
		return nil
	}
	if len(worker.jobs) == 0 {
		return fmt.Errorf("token periodic worker requires at least one job")
	}
	for index := range worker.jobs {
		job := &worker.jobs[index]
		if job.Name == "" || job.Interval <= 0 || job.RunOnce == nil {
			return fmt.Errorf("token periodic worker job %d is invalid", index)
		}
		if job.Initialize != nil {
			if err := job.Initialize(ctx); err != nil {
				return fmt.Errorf("initialize %s: %w", job.Name, err)
			}
		}
		telemetry.Register(worker.reporter, job.Scope)
	}
	runCtx, cancel := context.WithCancel(ctx)
	worker.cancel = cancel
	worker.done = make(chan struct{})
	go worker.run(runCtx)
	return nil
}

func (worker *PeriodicWorker) Stop(ctx context.Context) error {
	worker.mu.Lock()
	cancel, done := worker.cancel, worker.done
	worker.cancel, worker.done = nil, nil
	worker.mu.Unlock()
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
	var result error
	for _, job := range worker.jobs {
		if job.Shutdown != nil {
			if err := job.Shutdown(ctx); err != nil && result == nil {
				result = err
			}
		}
	}
	return result
}

func (worker *PeriodicWorker) run(ctx context.Context) {
	defer close(worker.done)
	var group sync.WaitGroup
	for _, job := range worker.jobs {
		job := job
		group.Add(1)
		go func() {
			defer group.Done()
			worker.runJob(ctx, job)
		}()
	}
	group.Wait()
}

func (worker *PeriodicWorker) runJob(ctx context.Context, job PeriodicJob) {
	ticker := time.NewTicker(job.Interval)
	defer ticker.Stop()
	for {
		result, err := job.RunOnce(ctx)
		if err != nil && ctx.Err() == nil {
			telemetry.Failure(worker.reporter, job.Scope, err)
			log.WithError(err).WithField("job", job.Name).Error("token periodic job failed")
		} else if err == nil {
			telemetry.Success(worker.reporter, job.Scope)
			if result.Processed > 0 {
				log.WithFields(log.Fields{"job": job.Name, "processed": result.Processed}).Debug("token periodic job completed")
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
