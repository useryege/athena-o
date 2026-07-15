package projectdatacollector

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	log "github.com/sirupsen/logrus"
	ethereumapiapiclient "github.com/useryege/athena/internal/ethereumapi/apiclient"
	tokenstore "github.com/useryege/athena/internal/token/store"
)

func (r *dataCollectorRunner) processContractCodeSourceTasks(ctx context.Context) error {
	tasks, err := r.opts.store.ListDueProjectDataCollectionTasks(ctx, tokenstore.ProjectDataCollectionTypeContractCodeSource, r.opts.chainIDs, contractCodeSourceLimit)
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

func (r *dataCollectorRunner) processContractCodeSourceTask(ctx context.Context, task tokenstore.ProjectDataCollectionTaskWithProject) {
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
	if r.opts.ethereumAPI == nil {
		r.markTaskFailed(ctx, task.Task, errEthereumAPIServerAddressRequired())
		return
	}
	sourceCode, err := r.fetchContractSourceCode(ctx, task.Project.ChainID, task.Project.Contract)
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

func (r *dataCollectorRunner) fetchContractSourceCode(ctx context.Context, chainID int64, contract common.Address) (string, error) {
	response, err := r.opts.ethereumAPI.GetSourceCode(ctx, &ethereumapiapiclient.GetSourceCodeRequest{
		ChainId:         chainID,
		ContractAddress: contract.Hex(),
	})
	if err != nil {
		return "", fmt.Errorf("fetch ethereum-api source code chain_id=%d contract=%s: %w", chainID, contract.Hex(), err)
	}
	if response == nil || len(response.GetItems()) == 0 {
		return "", fmt.Errorf("fetch ethereum-api source code chain_id=%d contract=%s returned empty result", chainID, contract.Hex())
	}
	return strings.TrimSpace(response.GetItems()[0].GetSourceCode()), nil
}
