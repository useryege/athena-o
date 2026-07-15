package collector

import (
	"context"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/useryege/athena/internal/token/research"
)

func (r *dataCollectorRunner) processContractCodeSourceTasks(ctx context.Context) error {
	tasks, err := r.opts.store.ListDueProjectDataCollectionTasks(ctx, research.DataCollectionTypeContractCodeSource, r.opts.chainIDs, contractCodeSourceLimit)
	if err != nil {
		return err
	}
	if len(tasks) == 0 {
		log.Debug("token project data collector has no due contract code source tasks")
		return nil
	}
	for _, task := range tasks {
		if err := ctx.Err(); err != nil {
			return err
		}
		r.processContractCodeSourceTask(ctx, task)
	}
	log.WithField("task_count", len(tasks)).Debug("token project data collector processed contract code source task batch")
	return nil
}

func (r *dataCollectorRunner) processContractCodeSourceTask(ctx context.Context, task research.ProjectDataCollectionTaskWithProject) {
	record, err := r.opts.store.GetContractCode(ctx, task.Project.CodeHash)
	if err != nil {
		r.markTaskFailed(ctx, task.Task, err)
		return
	}
	if record != nil && !record.SourceCodeFetchedAt.IsZero() {
		if err := r.opts.store.CompleteProjectContractCodeSourceCollection(ctx, task.Task, task.Project.CodeHash, record.SourceCode, time.Now().UTC()); err != nil {
			r.markTaskFailed(ctx, task.Task, err)
			return
		}
		log.WithFields(log.Fields{
			"project_id": task.Project.ID,
			"chain_id":   task.Project.ChainID,
			"contract":   task.Project.Contract.Hex(),
			"code_hash":  task.Project.CodeHash.Hex(),
		}).Info("token project data collector skipped contract code source because source already exists")
		return
	}
	if r.opts.sourceCode == nil {
		r.markTaskFailed(ctx, task.Task, errEthereumAPIServerAddressRequired())
		return
	}
	sourceCode, err := r.opts.sourceCode.GetSourceCode(ctx, task.Project.ChainID, task.Project.Contract)
	if err != nil {
		r.markTaskFailed(ctx, task.Task, err)
		return
	}
	if err := r.opts.store.CompleteProjectContractCodeSourceCollection(ctx, task.Task, task.Project.CodeHash, sourceCode, time.Now().UTC()); err != nil {
		r.markTaskFailed(ctx, task.Task, err)
		return
	}
	log.WithFields(log.Fields{
		"project_id": task.Project.ID,
		"chain_id":   task.Project.ChainID,
		"contract":   task.Project.Contract.Hex(),
		"code_hash":  task.Project.CodeHash.Hex(),
	}).Info("token project data collector fetched contract code source")
}
