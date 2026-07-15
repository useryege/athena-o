package projectdatacollector

import (
	"context"
	"sync"
	"time"

	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	log "github.com/sirupsen/logrus"
	ethereumapiapiclient "github.com/useryege/athena/internal/ethereumapi/apiclient"
	"github.com/useryege/athena/internal/token/domain"
	athenacontract "github.com/useryege/athena/pkg/abi/ATHENA"
	"github.com/useryege/athena/util/ave"
	utilio "github.com/useryege/athena/util/io"
)

type dataCollectorRunnerOptions struct {
	store           Store
	chainIDs        []int64
	nodeWSURLs      map[int64][]string
	athenaContracts map[int64]ethcommon.Address
	nodeWSUseProxy  bool
	aveClient       ave.Client
	ethereumAPI     ethereumapiapiclient.EthereumAPIServiceClient
	ethereumAPIConn utilio.Closer
	pollInterval    time.Duration
}

type dataCollectorRunner struct {
	opts dataCollectorRunnerOptions

	clientMu sync.Mutex
	clients  map[chainResourceKey]*ethclient.Client
	callers  map[chainResourceKey]*athenacontract.ATHENACaller
}

type chainResourceKey struct {
	dataType domain.DataCollectionType
	chainID  int64
}

func newDataCollectorRunner(opts dataCollectorRunnerOptions) *dataCollectorRunner {
	return &dataCollectorRunner{
		opts:    opts,
		clients: make(map[chainResourceKey]*ethclient.Client),
		callers: make(map[chainResourceKey]*athenacontract.ATHENACaller),
	}
}

func (r *dataCollectorRunner) run(ctx context.Context) {
	defer r.close()
	loops := []struct {
		name    string
		process func(context.Context) error
	}{
		{name: "ave", process: r.processAveTasks},
		{name: "contract code source", process: r.processContractCodeSourceTasks},
		{name: "chain state", process: r.processChainStateTasks},
		{name: "wallet asset state", process: r.processWalletAssetStateTasks},
		{name: "simulation result", process: r.processSimulationResultTasks},
	}
	var wg sync.WaitGroup
	for _, loop := range loops {
		loop := loop
		wg.Add(1)
		go func() {
			defer wg.Done()
			r.runCollectorLoop(ctx, loop.name, loop.process)
		}()
	}
	wg.Wait()
}

func (r *dataCollectorRunner) runCollectorLoop(ctx context.Context, name string, process func(context.Context) error) {
	for {
		if ctx.Err() != nil {
			return
		}
		if len(r.opts.chainIDs) == 0 {
			log.Debug("token project data collector has no enabled chains")
		} else if err := process(ctx); err != nil && ctx.Err() == nil {
			log.WithError(err).WithField("collector", name).Error("token project data collector loop failed")
		}
		if !sleepContext(ctx, r.opts.pollInterval) {
			return
		}
	}
}

func (r *dataCollectorRunner) failTasks(ctx context.Context, tasks []domain.ProjectDataCollectionTaskWithProject, err error) {
	for _, task := range tasks {
		r.markTaskFailed(ctx, task.Task, err)
	}
}

func (r *dataCollectorRunner) markTaskFailed(ctx context.Context, task domain.ProjectDataCollectionTask, err error) {
	if err == nil {
		return
	}
	updated, applied, updateErr := r.opts.store.MarkProjectDataCollectionTaskFailed(ctx, task, err.Error())
	if updateErr != nil {
		log.WithError(updateErr).WithFields(log.Fields{
			"project_id": task.ProjectID,
			"data_type":  task.DataType,
			"revision":   task.Revision,
		}).Error("token project data collector failed to mark task failed")
		return
	}
	if !applied {
		return
	}
	log.WithError(err).WithFields(log.Fields{
		"project_id": task.ProjectID,
		"data_type":  task.DataType,
		"revision":   task.Revision,
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
