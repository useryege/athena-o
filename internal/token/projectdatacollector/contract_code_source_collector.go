package projectdatacollector

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	log "github.com/sirupsen/logrus"
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
		if _, err := r.opts.store.MarkProjectDataCollectionTaskSucceeded(ctx, task.Project.ID, tokenstore.ProjectDataCollectionTypeContractCodeSource); err != nil {
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
	if r.opts.etherscanClient == nil {
		r.markTaskFailed(ctx, task.Task, errEtherscanAPIKeyRequired())
		return
	}
	sourceCode, err := r.fetchContractSourceCode(ctx, task.Project.ChainID, task.Project.Contract)
	if err != nil {
		r.markTaskFailed(ctx, task.Task, err)
		return
	}
	sourceCodeHash := crypto.Keccak256Hash([]byte(sourceCode))
	if err := r.opts.store.CompleteProjectContractCodeSourceCollection(ctx, task.Project.ID, task.Project.CodeHash, sourceCode, sourceCodeHash, time.Now().UTC()); err != nil {
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
	response, err := r.opts.etherscanClient.GetSourceCode(ctx, chainID, contract.Hex())
	if err != nil {
		return "", fmt.Errorf("fetch etherscan source code chain_id=%d contract=%s: %w", chainID, contract.Hex(), err)
	}
	if response == nil || len(response.Result) == 0 {
		return "", fmt.Errorf("fetch etherscan source code chain_id=%d contract=%s returned empty result", chainID, contract.Hex())
	}
	return strings.TrimSpace(response.Result[0].SourceCode), nil
}
