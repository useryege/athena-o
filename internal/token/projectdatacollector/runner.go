package projectdatacollector

import (
	"context"
	"sync"
	"time"

	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	log "github.com/sirupsen/logrus"
	tokenstore "github.com/useryege/athena/internal/token/store"
	athenacontract "github.com/useryege/athena/pkg/abi/ATHENA"
	"github.com/useryege/athena/util/ave"
)

type dataCollectorRunnerOptions struct {
	store           *tokenstore.SQLStore
	chainIDs        []int64
	nodeWSURLs      map[int64]string
	athenaContracts map[int64]ethcommon.Address
	nodeWSUseProxy  bool
	aveClient       ave.Client
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
	if err := r.processWalletAssetStateTasks(ctx); err != nil {
		log.WithError(err).Error("token project data collector wallet asset state task loop failed")
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
