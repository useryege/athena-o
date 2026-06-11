package projectdatacollector

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	tokenstore "github.com/useryege/athena/internal/token/store"
)

func (r *dataCollectorRunner) processAveTasks(ctx context.Context) error {
	tasks, err := r.opts.store.ListDueProjectDataCollectionTasks(ctx, tokenstore.ProjectDataCollectionTypeAve, r.opts.chainIDs, aveTaskLimit)
	if err != nil {
		return err
	}
	if len(tasks) == 0 {
		log.Debug("token project data collector has no due ave tasks")
		return nil
	}

	sem := make(chan struct{}, aveFetchConcurrency)
	wg := sync.WaitGroup{}
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
	log.WithField("task_count", len(tasks)).Debug("token project data collector processed ave task batch")
	return nil
}

func (r *dataCollectorRunner) processAveTask(ctx context.Context, item tokenstore.ProjectDataCollectionTaskWithProject) {
	resp, err := r.opts.aveClient.GetTokenDetail(ctx, item.Project.Contract, item.Project.ChainID)
	if err != nil {
		r.markTaskFailed(ctx, item.Task, fmt.Errorf("fetch ave token detail chain_id=%d contract=%s: %w", item.Project.ChainID, item.Project.Contract.Hex(), err))
		return
	}
	payload, err := json.Marshal(resp)
	if err != nil {
		r.markTaskFailed(ctx, item.Task, fmt.Errorf("marshal ave token detail project_id=%d: %w", item.Project.ID, err))
		return
	}
	if _, err := r.opts.store.CompleteProjectAveDataCollection(ctx, item.Task, payload, time.Now().UTC()); err != nil {
		r.markTaskFailed(ctx, item.Task, err)
		return
	}
	log.WithFields(log.Fields{
		"project_id": item.Project.ID,
		"chain_id":   item.Project.ChainID,
		"contract":   item.Project.Contract.Hex(),
	}).Info("token project data collector fetched ave data")
}
