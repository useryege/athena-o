package projectdatacollector

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	log "github.com/sirupsen/logrus"
	athenacommon "github.com/useryege/athena/common"
	tokenstore "github.com/useryege/athena/internal/token/store"
	athenacontract "github.com/useryege/athena/pkg/abi/ATHENA"
	"github.com/useryege/athena/util/ave"
	"github.com/useryege/athena/util/ethws"
)

type dataCollectorRunnerOptions struct {
	store           *tokenstore.SQLStore
	chainIDs        []int64
	nodeWSURLs      map[int64]string
	athenaContracts map[int64]ethcommon.Address
	nodeWSUseProxy  bool
	aveClient       ave.Client
	lockers         []ethcommon.Address
	pollInterval    time.Duration
}

type dataCollectorRunner struct {
	opts dataCollectorRunnerOptions

	clientMu sync.Mutex
	clients  map[int64]*ethclient.Client
	callers  map[int64]*athenacontract.ATHENACaller
}

func newDataCollectorRunner(opts dataCollectorRunnerOptions) *dataCollectorRunner {
	return &dataCollectorRunner{
		opts:    opts,
		clients: make(map[int64]*ethclient.Client),
		callers: make(map[int64]*athenacontract.ATHENACaller),
	}
}

func (r *dataCollectorRunner) run(ctx context.Context) {
	defer r.close()
	for {
		if err := ctx.Err(); err != nil {
			return
		}
		r.processAvailableTasks(ctx)
		if !sleepContext(ctx, r.opts.pollInterval) {
			return
		}
	}
}

func (r *dataCollectorRunner) processAvailableTasks(ctx context.Context) {
	if len(r.opts.chainIDs) == 0 {
		log.Debug("token project data collector has no enabled chains")
		return
	}
	if err := r.processAveTasks(ctx); err != nil {
		log.WithError(err).Error("token project data collector ave task loop failed")
	}
	if err := r.processChainStateTasks(ctx); err != nil {
		log.WithError(err).Error("token project data collector chain state task loop failed")
	}
}

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
	if _, err := r.opts.store.CompleteProjectAveDataCollection(ctx, item.Project.ID, payload, time.Now().UTC()); err != nil {
		r.markTaskFailed(ctx, item.Task, err)
		return
	}
	log.WithFields(log.Fields{
		"project_id": item.Project.ID,
		"chain_id":   item.Project.ChainID,
		"contract":   item.Project.Contract.Hex(),
	}).Info("token project data collector fetched ave data")
}

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
	queries := make([]athenacontract.AthenaProjectQuery, 0, len(tasks))
	for _, task := range tasks {
		queries = append(queries, athenacontract.AthenaProjectQuery{
			TokenContract:  task.Project.Contract,
			MsgCaller:      task.Project.Creator,
			GenesisWallets: []ethcommon.Address{},
		})
	}
	items, err := caller.List(&bind.CallOpts{Context: ctx}, queries, r.opts.lockers)
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

func (r *dataCollectorRunner) failTasks(ctx context.Context, tasks []tokenstore.ProjectDataCollectionTaskWithProject, err error) {
	for _, task := range tasks {
		r.markTaskFailed(ctx, task.Task, err)
	}
}

func (r *dataCollectorRunner) markTaskFailed(ctx context.Context, task tokenstore.ProjectDataCollectionTask, err error) {
	if err == nil {
		return
	}
	updated, updateErr := r.opts.store.MarkProjectDataCollectionTaskFailed(ctx, task.ProjectID, task.DataType, err.Error())
	if updateErr != nil {
		log.WithError(updateErr).WithFields(log.Fields{
			"project_id": task.ProjectID,
			"data_type":  task.DataType,
		}).Error("token project data collector failed to mark task failed")
		return
	}
	log.WithError(err).WithFields(log.Fields{
		"project_id": task.ProjectID,
		"data_type":  task.DataType,
		"attempts":   updated.Attempts,
		"status":     updated.Status,
	}).Warn("token project data collector task failed")
}

func (r *dataCollectorRunner) ensureClient(ctx context.Context, chainID int64) (*ethclient.Client, error) {
	r.clientMu.Lock()
	defer r.clientMu.Unlock()
	if client := r.clients[chainID]; client != nil {
		return client, nil
	}
	nodeWSURL := r.opts.nodeWSURLs[chainID]
	if nodeWSURL == "" {
		return nil, errNodeWSURLRequired(chainID)
	}
	client, err := ethws.DialContext(ctx, nodeWSURL, r.opts.nodeWSUseProxy)
	if err != nil {
		return nil, err
	}
	nodeChainID, err := client.ChainID(ctx)
	if err != nil {
		client.Close()
		return nil, err
	}
	if nodeChainID == nil || nodeChainID.Int64() != chainID {
		client.Close()
		return nil, fmt.Errorf("token project data collector node returned chain_id %v for configured chain_id %d", nodeChainID, chainID)
	}
	r.clients[chainID] = client
	log.WithFields(log.Fields{
		"chain_id":   chainID,
		"chain_name": athenacommon.ChainName(chainID),
	}).Info("token project data collector connected to node websocket")
	return client, nil
}

func (r *dataCollectorRunner) ensureCaller(ctx context.Context, chainID int64) (*athenacontract.ATHENACaller, error) {
	r.clientMu.Lock()
	if caller := r.callers[chainID]; caller != nil {
		r.clientMu.Unlock()
		return caller, nil
	}
	r.clientMu.Unlock()

	client, err := r.ensureClient(ctx, chainID)
	if err != nil {
		return nil, err
	}
	athenaContract := r.opts.athenaContracts[chainID]
	if athenaContract == (ethcommon.Address{}) {
		return nil, errAthenaContractRequired(chainID)
	}
	caller, err := athenacontract.NewATHENACaller(athenaContract, client)
	if err != nil {
		return nil, err
	}
	r.clientMu.Lock()
	defer r.clientMu.Unlock()
	if current := r.callers[chainID]; current != nil {
		return current, nil
	}
	r.callers[chainID] = caller
	log.WithFields(log.Fields{
		"chain_id":        chainID,
		"chain_name":      athenacommon.ChainName(chainID),
		"athena_contract": athenaContract.Hex(),
	}).Info("token project data collector initialized ATHENA caller")
	return caller, nil
}

func (r *dataCollectorRunner) resetChain(chainID int64) {
	r.clientMu.Lock()
	defer r.clientMu.Unlock()
	if client := r.clients[chainID]; client != nil {
		client.Close()
		delete(r.clients, chainID)
	}
	delete(r.callers, chainID)
}

func (r *dataCollectorRunner) close() {
	r.clientMu.Lock()
	defer r.clientMu.Unlock()
	for chainID, client := range r.clients {
		if client != nil {
			client.Close()
		}
		delete(r.clients, chainID)
	}
	for chainID := range r.callers {
		delete(r.callers, chainID)
	}
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
