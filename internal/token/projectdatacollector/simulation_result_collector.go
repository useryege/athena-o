package projectdatacollector

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	ethcommon "github.com/ethereum/go-ethereum/common"
	log "github.com/sirupsen/logrus"
	tokenstore "github.com/useryege/athena/internal/token/store"
	athenacontract "github.com/useryege/athena/pkg/abi/ATHENA"
)

func (r *dataCollectorRunner) processSimulationResultTasks(ctx context.Context) error {
	tasks, err := r.opts.store.ListDueProjectDataCollectionTasks(ctx, tokenstore.ProjectDataCollectionTypeSimulationResult, r.opts.chainIDs, simulationTaskLimit)
	if err != nil {
		return err
	}
	if len(tasks) == 0 {
		log.Debug("token project data collector has no due simulation result tasks")
		return nil
	}
	for _, task := range tasks {
		if err := ctx.Err(); err != nil {
			return err
		}
		r.processSimulationResultTask(ctx, task)
	}
	log.WithField("task_count", len(tasks)).Debug("token project data collector processed simulation result task batch")
	return nil
}

func (r *dataCollectorRunner) processSimulationResultTask(ctx context.Context, task tokenstore.ProjectDataCollectionTaskWithProject) {
	client, err := r.ensureClient(ctx, task.Project.ChainID)
	if err != nil {
		r.markTaskFailed(ctx, task.Task, err)
		return
	}
	blockNumber, err := client.BlockNumber(ctx)
	if err != nil {
		r.markTaskFailed(ctx, task.Task, err)
		return
	}
	wallets, err := r.projectRelatedWallets(ctx, task.Project.ID)
	if err != nil {
		r.markTaskFailed(ctx, task.Task, err)
		return
	}
	if len(wallets) == 0 {
		r.completeEmptySimulationResultTask(ctx, task, blockNumber, "without related wallets")
		return
	}
	if task.Project.WethPair == (ethcommon.Address{}) || task.Project.UsdtPair == (ethcommon.Address{}) {
		r.completeEmptySimulationResultTask(ctx, task, blockNumber, "without token pairs")
		return
	}
	caller, err := r.ensureCaller(ctx, task.Project.ChainID)
	if err != nil {
		r.markTaskFailed(ctx, task.Task, err)
		return
	}
	queries := walletSimulationStateQueries(task.Project.Contract, wallets)
	states, err := caller.ListWalletSimulationStates(&bind.CallOpts{Context: ctx, BlockNumber: new(big.Int).SetUint64(blockNumber)}, queries)
	if err != nil {
		r.resetChain(task.Project.ChainID)
		r.markTaskFailed(ctx, task.Task, fmt.Errorf("fetch ATHENA wallet simulation state chain_id=%d project_id=%d wallet_count=%d: %w", task.Project.ChainID, task.Project.ID, len(wallets), err))
		return
	}
	if len(states) != len(wallets) {
		r.resetChain(task.Project.ChainID)
		r.markTaskFailed(ctx, task.Task, fmt.Errorf("fetch ATHENA wallet simulation state chain_id=%d project_id=%d returned %d items for %d wallets", task.Project.ChainID, task.Project.ID, len(states), len(wallets)))
		return
	}
	results, err := simulateProjectWallets(ctx, client.Client(), task.Project, wallets, states)
	if err != nil {
		r.resetChain(task.Project.ChainID)
		r.markTaskFailed(ctx, task.Task, fmt.Errorf("simulate token project wallets chain_id=%d project_id=%d wallet_count=%d: %w", task.Project.ChainID, task.Project.ID, len(wallets), err))
		return
	}
	if err := r.opts.store.CompleteProjectSimulationResultCollection(ctx, task.Task, results, blockNumber, time.Now().UTC()); err != nil {
		r.markTaskFailed(ctx, task.Task, err)
		return
	}
	log.WithFields(log.Fields{
		"project_id":   task.Project.ID,
		"chain_id":     task.Project.ChainID,
		"contract":     task.Project.Contract.Hex(),
		"wallet_count": len(results),
	}).Info("token project data collector fetched simulation result")
}

func (r *dataCollectorRunner) completeEmptySimulationResultTask(ctx context.Context, task tokenstore.ProjectDataCollectionTaskWithProject, blockNumber uint64, reason string) {
	if err := r.opts.store.CompleteProjectSimulationResultCollection(ctx, task.Task, nil, blockNumber, time.Now().UTC()); err != nil {
		r.markTaskFailed(ctx, task.Task, err)
		return
	}
	log.WithFields(log.Fields{
		"project_id": task.Project.ID,
		"chain_id":   task.Project.ChainID,
		"contract":   task.Project.Contract.Hex(),
		"reason":     reason,
	}).Info("token project data collector skipped simulation result")
}

func walletSimulationStateQueries(tokenContract ethcommon.Address, wallets []ethcommon.Address) []athenacontract.AthenaWalletSimulationStateQuery {
	queries := make([]athenacontract.AthenaWalletSimulationStateQuery, 0, len(wallets))
	for _, wallet := range wallets {
		queries = append(queries, athenacontract.AthenaWalletSimulationStateQuery{
			TokenContract: tokenContract,
			MsgCaller:     wallet,
		})
	}
	return queries
}
