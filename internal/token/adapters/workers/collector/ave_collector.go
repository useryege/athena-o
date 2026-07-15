package collector

import (
	"context"
	"fmt"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/useryege/athena/internal/token/research"
)

func (r *dataCollectorRunner) processAveTasks(ctx context.Context) error {
	tasks, err := r.opts.store.ListDueProjectDataCollectionTasks(ctx, research.DataCollectionTypeAve, r.opts.chainIDs, aveTaskLimit)
	if err != nil {
		return err
	}
	if len(tasks) == 0 {
		log.Debug("token project data collector has no due Ave tasks")
		return nil
	}

	sem := make(chan struct{}, aveFetchConcurrency)
	var wg sync.WaitGroup
	for _, task := range tasks {
		if err := ctx.Err(); err != nil {
			return err
		}
		task := task
		sem <- struct{}{}
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			r.processAveTask(ctx, task)
		}()
	}
	wg.Wait()
	log.WithField("task_count", len(tasks)).Debug("token project data collector processed Ave task batch")
	return nil
}

func (r *dataCollectorRunner) processAveTask(ctx context.Context, item research.ProjectDataCollectionTaskWithProject) {
	observation, err := r.opts.marketData.GetMarketData(ctx, item.Project.ChainID, item.Project.Contract)
	if err != nil {
		r.markTaskFailed(ctx, item.Task, fmt.Errorf("collect Ave market data project_id=%d: %w", item.Project.ID, err))
		return
	}
	if _, err := r.opts.store.CompleteProjectAveDataCollection(ctx, item.Task, observation, time.Now().UTC()); err != nil {
		r.markTaskFailed(ctx, item.Task, err)
		return
	}
	log.WithFields(log.Fields{
		"project_id": item.Project.ID,
		"chain_id":   item.Project.ChainID,
		"contract":   item.Project.Contract.Hex(),
	}).Info("token project data collector fetched Ave data")
}
