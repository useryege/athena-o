package collector

import (
	"context"
	"sync"
	"time"

	ethcommon "github.com/ethereum/go-ethereum/common"
	log "github.com/sirupsen/logrus"
	"github.com/useryege/athena/internal/token/adapters/evm"
	"github.com/useryege/athena/internal/token/research"
	"github.com/useryege/athena/internal/token/telemetry"
	athenacontract "github.com/useryege/athena/pkg/abi/ATHENA"
)

type dataCollectorRunnerOptions struct {
	store           research.CollectorRepository
	clients         *evm.ChainClientRegistry
	dataType        research.DataCollectionType
	chainIDs        []int64
	athenaContracts map[int64]ethcommon.Address
	marketData      research.MarketDataProvider
	sourceCode      research.SourceCodeProvider
	pollInterval    time.Duration
	telemetry       telemetry.Reporter
}

type dataCollectorRunner struct {
	opts dataCollectorRunnerOptions

	resourceMu sync.Mutex
	resources  map[chainResourceKey]*chainResource
}

type chainResourceKey struct {
	dataType research.DataCollectionType
	chainID  int64
}

type chainResource struct {
	mu     sync.Mutex
	caller *athenacontract.ATHENACaller
}

func newDataCollectorRunner(opts dataCollectorRunnerOptions) *dataCollectorRunner {
	return &dataCollectorRunner{
		opts:      opts,
		resources: make(map[chainResourceKey]*chainResource),
	}
}

func (r *dataCollectorRunner) chainResource(key chainResourceKey) *chainResource {
	r.resourceMu.Lock()
	defer r.resourceMu.Unlock()
	resource := r.resources[key]
	if resource == nil {
		resource = &chainResource{}
		r.resources[key] = resource
	}
	return resource
}

func (r *dataCollectorRunner) existingChainResource(key chainResourceKey) *chainResource {
	r.resourceMu.Lock()
	defer r.resourceMu.Unlock()
	return r.resources[key]
}

func (r *dataCollectorRunner) run(ctx context.Context) {
	defer r.close()
	loops := map[research.DataCollectionType]struct {
		name    string
		process func(context.Context) error
	}{
		research.DataCollectionTypeAve:                {name: "ave", process: r.processAveTasks},
		research.DataCollectionTypeContractCodeSource: {name: "contract code source", process: r.processContractCodeSourceTasks},
		research.DataCollectionTypeChainState:         {name: "chain state", process: r.processChainStateTasks},
		research.DataCollectionTypeWalletAssetState:   {name: "wallet asset state", process: r.processWalletAssetStateTasks},
		research.DataCollectionTypeSimulationResult:   {name: "simulation result", process: r.processSimulationResultTasks},
	}
	loop, ok := loops[r.opts.dataType]
	if !ok {
		return
	}
	r.runCollectorLoop(ctx, loop.name, r.opts.dataType, loop.process)
}

func (r *dataCollectorRunner) runCollectorLoop(ctx context.Context, name string, dataType research.DataCollectionType, process func(context.Context) error) {
	scope := telemetry.Scope{Component: "data_collector", DataType: string(dataType)}
	if len(r.opts.chainIDs) > 0 {
		telemetry.Register(r.opts.telemetry, scope)
	}
	for {
		if ctx.Err() != nil {
			return
		}
		if len(r.opts.chainIDs) == 0 {
			log.Debug("token project data collector has no enabled chains")
		} else if err := process(ctx); err != nil && ctx.Err() == nil {
			telemetry.Failure(r.opts.telemetry, scope, err)
			log.WithError(err).WithField("collector", name).Error("token project data collector loop failed")
		} else if ctx.Err() == nil {
			telemetry.Success(r.opts.telemetry, scope)
		}
		if !sleepContext(ctx, r.opts.pollInterval) {
			return
		}
	}
}

func (r *dataCollectorRunner) failTasks(ctx context.Context, tasks []research.ProjectDataCollectionTaskWithProject, err error) {
	for _, task := range tasks {
		r.markTaskFailed(ctx, task.Task, err)
	}
}

func (r *dataCollectorRunner) markTaskFailed(ctx context.Context, task research.ProjectDataCollectionTask, err error) {
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
