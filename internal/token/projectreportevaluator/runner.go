package projectreportevaluator

import (
	"context"
	"time"

	log "github.com/sirupsen/logrus"
	tokenstore "github.com/useryege/athena/internal/token/store"
)

type reportEvaluatorRunner struct {
	store        *tokenstore.SQLStore
	pollInterval time.Duration
	taskLimit    int32
}

func (r *reportEvaluatorRunner) run(ctx context.Context) {
	for {
		if err := ctx.Err(); err != nil {
			return
		}
		if err := r.processAvailableTasks(ctx); err != nil {
			log.WithError(err).Error("token project report evaluator task loop failed")
		}
		if !sleepContext(ctx, r.pollInterval) {
			return
		}
	}
}

func (r *reportEvaluatorRunner) processAvailableTasks(ctx context.Context) error {
	tasks, err := r.store.ListDueProjectReportEvaluationTasks(ctx, r.taskLimit)
	if err != nil {
		return err
	}
	if len(tasks) == 0 {
		log.Debug("token project report evaluator has no due tasks")
		return nil
	}
	for _, task := range tasks {
		if err := ctx.Err(); err != nil {
			return err
		}
		r.processTask(ctx, task)
	}
	log.WithField("task_count", len(tasks)).Debug("token project report evaluator processed task batch")
	return nil
}

func sleepContext(ctx context.Context, interval time.Duration) bool {
	timer := time.NewTimer(interval)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
