package application

import (
	"context"
	"errors"
	"sync"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/core/types"
	log "github.com/sirupsen/logrus"
	"github.com/useryege/athena/internal/application/evm"
)

const (
	defaultBlockHeaderQueueCapacity   = 16
	defaultProjectSimulateConcurrency = 30
)

type NewHeadSubscriber interface {
	SubscribeNewHead(ctx context.Context, ch chan<- *types.Header) (ethereum.Subscription, error)
	BlockNumber(ctx context.Context) (uint64, error)
}

type BlockEventSubscriber struct {
	client           NewHeadSubscriber
	registry         ProjectRegistry
	fetcher          evm.AthenaFetcher
	projectSimulator ProjectSimulator
	wg               sync.WaitGroup
}

func NewBlockEventSubscriber(
	client NewHeadSubscriber,
	registry ProjectRegistry,
	fetcher evm.AthenaFetcher,
	projectSimulator ProjectSimulator,
) *BlockEventSubscriber {
	return &BlockEventSubscriber{
		client:           client,
		registry:         registry,
		fetcher:          fetcher,
		projectSimulator: projectSimulator,
	}
}

func (s *BlockEventSubscriber) Start(ctx context.Context) error {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		if err := s.run(ctx); err != nil && !errors.Is(err, context.Canceled) {
			log.WithError(err).Error("failed to subscribe new block headers")
		}
	}()
	return nil
}

func (s *BlockEventSubscriber) Stop() error {
	s.wg.Wait()
	return nil
}

func (s *BlockEventSubscriber) run(ctx context.Context) error {
	headers := make(chan *types.Header, defaultBlockHeaderQueueCapacity)
	subscription, err := s.client.SubscribeNewHead(ctx, headers)
	if err != nil {
		return err
	}
	defer subscription.Unsubscribe()

	syncDone := make(chan error, 1)
	syncRunning := false

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-subscription.Err():
			if err != nil {
				return err
			}
			return nil
		case header := <-headers:
			if header == nil || header.Number == nil {
				continue
			}

			triggerBlockNumber := header.Number.Uint64()
			log.WithFields(log.Fields{
				"event":              "header_received",
				"triggerBlockNumber": triggerBlockNumber,
			}).Info("received new block header")

			latestBlockNumber, err := s.client.BlockNumber(ctx)
			if err != nil {
				if errors.Is(err, context.Canceled) {
					return err
				}
				log.WithFields(log.Fields{
					"event":              "header_received",
					"triggerBlockNumber": triggerBlockNumber,
					"error":              err,
				}).Warn("failed to get latest block number after new block header")
			} else {
				log.WithFields(log.Fields{
					"event":              "header_received",
					"triggerBlockNumber": triggerBlockNumber,
					"latestBlockNumber":  latestBlockNumber,
				}).Info("resolved latest block number after new block header")
			}

			if syncRunning {
				continue
			}

			syncRunning = true
			s.wg.Add(1)
			// go func() {
			// 	defer s.wg.Done()
			// 	syncDone <- s.refreshProjectStates(ctx, triggerBlockNumber)
			// }()
		case err := <-syncDone:
			syncRunning = false
			if err != nil {
				if errors.Is(err, context.Canceled) {
					return err
				}
				log.WithFields(log.Fields{
					"event": "sync_failed",
					"error": err,
				}).Warn("project state sync failed; continuing subscriber loop")
			}
		}
	}
}

// func (s *BlockEventSubscriber) refreshProjectStates(ctx context.Context, triggerBlockNumber uint64) error {
// 	syncStartedAt := time.Now()
// 	syncID := fmt.Sprintf("%d-%d", triggerBlockNumber, syncStartedAt.UnixNano())
// 	fields := log.Fields{
// 		"event":              "sync_started",
// 		"syncID":             syncID,
// 		"triggerBlockNumber": triggerBlockNumber,
// 	}
// 	log.WithFields(fields).Info("started project state sync")

// 	var syncErr error

// 	startBlockNumberStartedAt := time.Now()
// 	startBlockNumber, err := s.client.BlockNumber(ctx)
// 	fields["startBlockNumberDuration"] = time.Since(startBlockNumberStartedAt)
// 	fields["startBlockNumber"] = startBlockNumber
// 	if err != nil {
// 		if errors.Is(err, context.Canceled) {
// 			return err
// 		}
// 		fields["error"] = err
// 		log.WithFields(fields).Warn("failed to get start block number before refreshing project chain states")
// 		delete(fields, "error")
// 	}

// 	listStartedAt := time.Now()
// 	snapshot, err := s.registry.ListProjectContracts(ctx)
// 	fields["listProjectContractsDuration"] = time.Since(listStartedAt)
// 	if err != nil {
// 		syncErr = err
// 		return s.logSyncCompletion(ctx, fields, syncStartedAt, syncErr)
// 	}
// 	if len(snapshot.ProjectContracts) != len(snapshot.ProjectQueries) {
// 		syncErr = fmt.Errorf(
// 			"project contract snapshot has inconsistent lengths: contracts=%d queries=%d",
// 			len(snapshot.ProjectContracts),
// 			len(snapshot.ProjectQueries),
// 		)
// 		return s.logSyncCompletion(ctx, fields, syncStartedAt, syncErr)
// 	}
// 	fields["tokenContractCount"] = len(snapshot.ProjectContracts)
// 	if len(snapshot.ProjectContracts) == 0 {
// 		return s.logSyncCompletion(ctx, fields, syncStartedAt, nil)
// 	}

// 	fetchStartedAt := time.Now()
// 	var snapshots []athenacontract.AthenaProject
// 	var simulationStates []athenacontract.AthenaSimulationState
// 	if s.projectSimulator == nil {
// 		snapshots, err = s.fetcher.FetchProjects(ctx, snapshot.ProjectContracts)
// 	} else {
// 		projects, fetchErr := s.fetcher.FetchProjectsWithSimulationState(ctx, snapshot.ProjectQueries)
// 		if fetchErr != nil {
// 			fields["fetchDuration"] = time.Since(fetchStartedAt)
// 			syncErr = fetchErr
// 			return s.logSyncCompletion(ctx, fields, syncStartedAt, syncErr)
// 		}
// 		snapshots = make([]athenacontract.AthenaProject, len(projects))
// 		simulationStates = make([]athenacontract.AthenaSimulationState, len(projects))
// 		for i, project := range projects {
// 			snapshots[i] = project.Project
// 			simulationStates[i] = project.SimulationState
// 		}
// 	}
// 	fields["fetchDuration"] = time.Since(fetchStartedAt)
// 	if err != nil {
// 		syncErr = err
// 		return s.logSyncCompletion(ctx, fields, syncStartedAt, syncErr)
// 	}
// 	if len(snapshots) != len(snapshot.ProjectContracts) {
// 		syncErr = fmt.Errorf("athena list returned %d projects for %d token contracts", len(snapshots), len(snapshot.ProjectContracts))
// 		return s.logSyncCompletion(ctx, fields, syncStartedAt, syncErr)
// 	}

// 	buildStatesStartedAt := time.Now()
// 	states := make(map[common.Address]athenacontract.AthenaProject, len(snapshot.ProjectContracts))
// 	for i, contract := range snapshot.ProjectContracts {
// 		projectSnapshot := snapshots[i]
// 		if projectSnapshot.TokenContract != (common.Address{}) && projectSnapshot.TokenContract != contract {
// 			syncErr = fmt.Errorf("athena list result %d token contract = %s, want %s", i, projectSnapshot.TokenContract, contract)
// 			return s.logSyncCompletion(ctx, fields, syncStartedAt, syncErr)
// 		}
// 		states[contract] = projectSnapshot
// 	}
// 	fields["buildStatesDuration"] = time.Since(buildStatesStartedAt)

// 	updateStatesStartedAt := time.Now()
// 	if err := s.registry.UpdateProjectChainStates(ctx, states); err != nil {
// 		fields["updateProjectChainStatesDuration"] = time.Since(updateStatesStartedAt)
// 		syncErr = err
// 		return s.logSyncCompletion(ctx, fields, syncStartedAt, syncErr)
// 	}
// 	fields["updateProjectChainStatesDuration"] = time.Since(updateStatesStartedAt)

// 	if s.projectSimulator != nil {
// 		simulateStartedAt := time.Now()
// 		stats, simulateErr := s.syncProjectSimulateStates(ctx, snapshot, snapshots, simulationStates)
// 		if simulateErr != nil {
// 			syncErr = errors.Join(syncErr, simulateErr)
// 		}
// 		fields["simulateProjectCount"] = stats.projectCount
// 		fields["simulateConcurrency"] = stats.concurrency
// 		fields["simulateDuration"] = time.Since(simulateStartedAt)
// 		fields["simulateCallsDuration"] = stats.callsDuration
// 		fields["updateProjectSimulateStatesDuration"] = stats.updatesDuration
// 		fields["simulateErrorCount"] = stats.simulateErrorCount
// 		fields["simulateUpdateErrorCount"] = stats.updateErrorCount
// 	}

// 	return s.logSyncCompletion(ctx, fields, syncStartedAt, syncErr)
// }

// func (s *BlockEventSubscriber) logSyncCompletion(ctx context.Context, fields log.Fields, syncStartedAt time.Time, syncErr error) error {
// 	endBlockNumberStartedAt := time.Now()
// 	endBlockNumber, endErr := s.client.BlockNumber(ctx)
// 	fields["endBlockNumberDuration"] = time.Since(endBlockNumberStartedAt)
// 	fields["totalDuration"] = time.Since(syncStartedAt)
// 	if endErr != nil {
// 		if errors.Is(endErr, context.Canceled) {
// 			return endErr
// 		}
// 		log.WithFields(log.Fields{
// 			"event":              "sync_failed",
// 			"syncID":             fields["syncID"],
// 			"triggerBlockNumber": fields["triggerBlockNumber"],
// 			"startBlockNumber":   fields["startBlockNumber"],
// 			"error":              endErr,
// 		}).Warn("failed to get end block number after refreshing project chain states")
// 	}
// 	fields["endBlockNumber"] = endBlockNumber

// 	if syncErr != nil {
// 		if errors.Is(syncErr, context.Canceled) {
// 			return syncErr
// 		}
// 		fields["event"] = "sync_failed"
// 		fields["error"] = syncErr
// 		log.WithFields(fields).Warn("failed to refresh project chain states")
// 		return syncErr
// 	}

// 	fields["event"] = "sync_completed"
// 	log.WithFields(fields).Info("refreshed project chain states")
// 	return nil
// }

// type projectSimulateStats struct {
// 	projectCount       int
// 	concurrency        int
// 	simulateErrorCount int
// 	updateErrorCount   int
// 	callsDuration      time.Duration
// 	updatesDuration    time.Duration
// }

// type projectSimulateJob struct {
// 	index int
// }

// func (s *BlockEventSubscriber) syncProjectSimulateStates(
// 	ctx context.Context,
// 	contractSnapshot ProjectContractSnapshot,
// 	snapshots []athenacontract.AthenaProject,
// 	simulationStates []athenacontract.AthenaSimulationState,
// ) (projectSimulateStats, error) {
// 	stats := projectSimulateStats{
// 		projectCount: len(contractSnapshot.ProjectContracts),
// 		concurrency:  defaultProjectSimulateConcurrency,
// 	}
// 	if len(simulationStates) != len(contractSnapshot.ProjectContracts) {
// 		return stats, fmt.Errorf("athena list returned %d simulation states for %d token contracts", len(simulationStates), len(contractSnapshot.ProjectContracts))
// 	}
// 	if stats.projectCount < stats.concurrency {
// 		stats.concurrency = stats.projectCount
// 	}
// 	if stats.concurrency <= 0 {
// 		return stats, nil
// 	}

// 	jobs := make(chan projectSimulateJob, stats.concurrency)
// 	var wg sync.WaitGroup
// 	var mu sync.Mutex
// 	var syncErr error

// 	recordDurations := func(callsDuration time.Duration, updatesDuration time.Duration) {
// 		mu.Lock()
// 		stats.callsDuration += callsDuration
// 		stats.updatesDuration += updatesDuration
// 		mu.Unlock()
// 	}
// 	recordSimulateError := func(err error) {
// 		mu.Lock()
// 		stats.simulateErrorCount++
// 		syncErr = errors.Join(syncErr, err)
// 		mu.Unlock()
// 	}
// 	recordUpdateError := func(err error) {
// 		mu.Lock()
// 		stats.updateErrorCount++
// 		syncErr = errors.Join(syncErr, err)
// 		mu.Unlock()
// 	}
// 	recordError := func(err error) {
// 		mu.Lock()
// 		syncErr = errors.Join(syncErr, err)
// 		mu.Unlock()
// 	}

// 	for worker := 0; worker < stats.concurrency; worker++ {
// 		wg.Add(1)
// 		go func() {
// 			defer wg.Done()
// 			for job := range jobs {
// 				if err := ctx.Err(); err != nil {
// 					return
// 				}

// 				simulateCallStartedAt := time.Now()
// 				simulateResult, err := s.projectSimulator.SimulatePrimary(
// 					ctx,
// 					contractSnapshot.ProjectQueries[job.index].MsgCaller,
// 					contractSnapshot.ProjectContracts[job.index],
// 					snapshots[job.index].WethPair.ContractAddress,
// 					snapshots[job.index].UsdtPair.ContractAddress,
// 					simulationStates[job.index],
// 				)
// 				simulateCallsDuration := time.Since(simulateCallStartedAt)
// 				if err != nil {
// 					recordSimulateError(err)
// 					recordDurations(simulateCallsDuration, 0)
// 					continue
// 				}

// 				simulateUpdateStartedAt := time.Now()
// 				contract := contractSnapshot.ProjectContracts[job.index]
// 				project, ok, getErr := s.registry.GetProject(ctx, contract)
// 				if getErr != nil {
// 					recordUpdateError(getErr)
// 					recordDurations(simulateCallsDuration, time.Since(simulateUpdateStartedAt))
// 					continue
// 				}
// 				if !ok || project == nil {
// 					recordDurations(simulateCallsDuration, time.Since(simulateUpdateStartedAt))
// 					continue
// 				}
// 				metaState := project.Meta
// 				metaState.CreatorResult = simulateResult
// 				if err := s.registry.UpdateProjectMetaState(ctx, contract, &metaState); err != nil {
// 					recordUpdateError(err)
// 				}
// 				simulateUpdatesDuration := time.Since(simulateUpdateStartedAt)
// 				recordDurations(simulateCallsDuration, simulateUpdatesDuration)
// 			}
// 		}()
// 	}

// enqueueJobs:
// 	for i := range contractSnapshot.ProjectContracts {
// 		select {
// 		case jobs <- projectSimulateJob{index: i}:
// 		case <-ctx.Done():
// 			recordError(ctx.Err())
// 			break enqueueJobs
// 		}
// 	}
// 	close(jobs)
// 	wg.Wait()

// 	return stats, syncErr
// }
