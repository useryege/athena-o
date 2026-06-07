package projectdatacollector

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	ethcommon "github.com/ethereum/go-ethereum/common"
	log "github.com/sirupsen/logrus"
	tokenstore "github.com/useryege/athena/internal/token/store"
)

func (r *dataCollectorRunner) processChainStateTasks(ctx context.Context) error {
	tasks, err := r.opts.store.ListDueProjectDataCollectionTasks(ctx, tokenstore.ProjectDataCollectionTypeChainState, r.opts.chainIDs, chainStateTaskLimit)
	if err != nil {
		return err
	}
	if len(tasks) == 0 {
		log.Debug("token project data collector has no due chain state tasks")
		return nil
	}
	byChain := make(map[int64][]tokenstore.ProjectDataCollectionTaskWithProject)
	for _, task := range tasks {
		byChain[task.Project.ChainID] = append(byChain[task.Project.ChainID], task)
	}
	for chainID, chainTasks := range byChain {
		if err := ctx.Err(); err != nil {
			return err
		}
		r.processChainStateTaskBatch(ctx, chainID, chainTasks)
	}
	log.WithField("task_count", len(tasks)).Debug("token project data collector processed chain state task batch")
	return nil
}

func (r *dataCollectorRunner) processChainStateTaskBatch(ctx context.Context, chainID int64, tasks []tokenstore.ProjectDataCollectionTaskWithProject) {
	caller, err := r.ensureCaller(ctx, chainID)
	if err != nil {
		r.failTasks(ctx, tasks, err)
		return
	}
	tokenContracts := make([]ethcommon.Address, 0, len(tasks))
	for _, task := range tasks {
		tokenContracts = append(tokenContracts, task.Project.Contract)
	}
	items, err := caller.ListProjectStates(&bind.CallOpts{Context: ctx}, tokenContracts)
	if err != nil {
		r.resetChain(chainID)
		r.failTasks(ctx, tasks, fmt.Errorf("fetch ATHENA chain state chain_id=%d task_count=%d: %w", chainID, len(tasks), err))
		return
	}
	if len(items) != len(tasks) {
		r.resetChain(chainID)
		r.failTasks(ctx, tasks, fmt.Errorf("fetch ATHENA chain state chain_id=%d returned %d items for %d tasks", chainID, len(items), len(tasks)))
		return
	}
	for i, state := range items {
		task := tasks[i]
		payload, err := json.Marshal(state)
		if err != nil {
			r.markTaskFailed(ctx, task.Task, fmt.Errorf("marshal ATHENA chain state project_id=%d: %w", task.Project.ID, err))
			continue
		}
		if _, err := r.opts.store.CompleteProjectChainStateCollection(ctx, task.Project.ID, payload, time.Now().UTC()); err != nil {
			r.markTaskFailed(ctx, task.Task, err)
			continue
		}
		log.WithFields(log.Fields{
			"project_id": task.Project.ID,
			"chain_id":   task.Project.ChainID,
			"contract":   task.Project.Contract.Hex(),
		}).Info("token project data collector fetched chain state")
	}
}
