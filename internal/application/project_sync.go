package application

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	log "github.com/sirupsen/logrus"
	"github.com/useryege/athena/internal/application/evm"
	athenacontract "github.com/useryege/athena/pkg/abi/ATHENA"
	"github.com/useryege/athena/util/ethereumapi"
)

const (
	defaultProjectSimulateConcurrency = 30
)

type ProjectSync interface {
	SyncSourceCodeOnce(ctx context.Context, event *Project) (bool, error)
	SyncProjectStatesOnce(ctx context.Context, triggerBlockNumber uint64) error
}

var _ ProjectSync = &projectSyncImpl{}

type projectSyncImpl struct {
	nodeClient       *ethclient.Client
	registry         ProjectRegistry
	fetcher          evm.AthenaFetcher
	apiFetcher       ethereumapi.EthereumAPI
	projectSimulator ProjectSimulator
}

func NewProjectSync(
	nodeClient *ethclient.Client,
	registry ProjectRegistry,
	fetcher evm.AthenaFetcher,
	apiFetcher ethereumapi.EthereumAPI,
	projectSimulator ProjectSimulator,
) ProjectSync {
	return &projectSyncImpl{
		nodeClient:       nodeClient,
		registry:         registry,
		fetcher:          fetcher,
		apiFetcher:       apiFetcher,
		projectSimulator: projectSimulator,
	}
}

func (s *projectSyncImpl) SyncSourceCodeOnce(ctx context.Context, event *Project) (bool, error) {
	if event.Meta.SourceCode != "" {
		return true, nil
	}

	sourceCode, _, err := s.fetchSourceCode(ctx, event)
	if err != nil {
		return false, err
	}

	event.Meta.SourceCode = sourceCode
	return true, nil
}

func (s *projectSyncImpl) SyncProjectStatesOnce(ctx context.Context, triggerBlockNumber uint64) (syncErr error) {
	syncStartedAt := time.Now()
	fields := log.Fields{
		"blockNumber": triggerBlockNumber,
	}

	startBlockNumberStartedAt := time.Now()
	startBlockNumber, err := s.nodeClient.BlockNumber(ctx)
	fields["startBlockNumberDuration"] = time.Since(startBlockNumberStartedAt)
	fields["startBlockNumber"] = startBlockNumber
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return err
		}
		fields["error"] = err
		log.WithFields(fields).Warn("failed to get start block number before refreshing project chain states")
		delete(fields, "error")
	}

	defer func() {
		endBlockNumberStartedAt := time.Now()
		endBlockNumber, err := s.nodeClient.BlockNumber(ctx)
		fields["endBlockNumberDuration"] = time.Since(endBlockNumberStartedAt)
		fields["totalDuration"] = time.Since(syncStartedAt)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				syncErr = err
				return
			}
			log.WithFields(log.Fields{
				"blockNumber":      triggerBlockNumber,
				"startBlockNumber": startBlockNumber,
				"error":            err,
			}).Warn("failed to get end block number after refreshing project chain states")
		}
		fields["endBlockNumber"] = endBlockNumber

		if syncErr != nil {
			if errors.Is(syncErr, context.Canceled) {
				return
			}
			fields["error"] = syncErr
			log.WithFields(fields).Warn("failed to refresh project chain states")
			return
		}

		log.WithFields(fields).Info("refreshed project chain states")
	}()

	listStartedAt := time.Now()
	snapshot, err := s.registry.ListProjectContracts(ctx)
	fields["listProjectContractsDuration"] = time.Since(listStartedAt)
	if err != nil {
		return err
	}
	if len(snapshot.ProjectContracts) != len(snapshot.ProjectQueries) {
		return fmt.Errorf(
			"project contract snapshot has inconsistent lengths: contracts=%d queries=%d",
			len(snapshot.ProjectContracts),
			len(snapshot.ProjectQueries),
		)
	}
	fields["tokenContractCount"] = len(snapshot.ProjectContracts)
	if len(snapshot.ProjectContracts) == 0 {
		return nil
	}

	fetchStartedAt := time.Now()
	var snapshots []athenacontract.AthenaProject
	var simulationStates []athenacontract.AthenaSimulationState
	if s.projectSimulator == nil {
		snapshots, err = s.fetcher.FetchProjects(ctx, snapshot.ProjectContracts)
	} else {
		projects, err := s.fetcher.FetchProjectsWithSimulationState(ctx, snapshot.ProjectQueries)
		if err != nil {
			fields["fetchDuration"] = time.Since(fetchStartedAt)
			return err
		}
		snapshots = make([]athenacontract.AthenaProject, len(projects))
		simulationStates = make([]athenacontract.AthenaSimulationState, len(projects))
		for i, project := range projects {
			snapshots[i] = project.Project
			simulationStates[i] = project.SimulationState
		}
	}
	fields["fetchDuration"] = time.Since(fetchStartedAt)
	if err != nil {
		return err
	}
	if len(snapshots) != len(snapshot.ProjectContracts) {
		return fmt.Errorf("athena list returned %d projects for %d token contracts", len(snapshots), len(snapshot.ProjectContracts))
	}

	buildStatesStartedAt := time.Now()
	states := make(map[common.Address]athenacontract.AthenaProject, len(snapshot.ProjectContracts))
	for i, contract := range snapshot.ProjectContracts {
		projectSnapshot := snapshots[i]
		if projectSnapshot.TokenContract != (common.Address{}) && projectSnapshot.TokenContract != contract {
			return fmt.Errorf("athena list result %d token contract = %s, want %s", i, projectSnapshot.TokenContract, contract)
		}
		states[contract] = projectSnapshot
	}
	fields["buildStatesDuration"] = time.Since(buildStatesStartedAt)

	updateStatesStartedAt := time.Now()
	if err := s.registry.UpdateProjectChainStates(ctx, states); err != nil {
		fields["updateProjectChainStatesDuration"] = time.Since(updateStatesStartedAt)
		return err
	}
	fields["updateProjectChainStatesDuration"] = time.Since(updateStatesStartedAt)

	if s.projectSimulator != nil {
		simulateStartedAt := time.Now()
		stats, err := s.syncProjectSimulateStates(ctx, snapshot, snapshots, simulationStates)
		if err != nil {
			syncErr = errors.Join(syncErr, err)
		}
		fields["simulateProjectCount"] = stats.projectCount
		fields["simulateConcurrency"] = stats.concurrency
		fields["simulateDuration"] = time.Since(simulateStartedAt)
		fields["simulateCallsDuration"] = stats.callsDuration
		fields["updateProjectSimulateStatesDuration"] = stats.updatesDuration
		fields["simulateErrorCount"] = stats.simulateErrorCount
		fields["simulateUpdateErrorCount"] = stats.updateErrorCount
	}

	return syncErr
}

type projectSimulateStats struct {
	projectCount       int
	concurrency        int
	simulateErrorCount int
	updateErrorCount   int
	callsDuration      time.Duration
	updatesDuration    time.Duration
}

type projectSimulateJob struct {
	index int
}

func (s *projectSyncImpl) syncProjectSimulateStates(
	ctx context.Context,
	contractSnapshot ProjectContractSnapshot,
	snapshots []athenacontract.AthenaProject,
	simulationStates []athenacontract.AthenaSimulationState,
) (projectSimulateStats, error) {
	stats := projectSimulateStats{
		projectCount: len(contractSnapshot.ProjectContracts),
		concurrency:  defaultProjectSimulateConcurrency,
	}
	if len(simulationStates) != len(contractSnapshot.ProjectContracts) {
		return stats, fmt.Errorf("athena list returned %d simulation states for %d token contracts", len(simulationStates), len(contractSnapshot.ProjectContracts))
	}
	if stats.projectCount < stats.concurrency {
		stats.concurrency = stats.projectCount
	}
	if stats.concurrency <= 0 {
		return stats, nil
	}

	jobs := make(chan projectSimulateJob, stats.concurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var syncErr error

	recordDurations := func(callsDuration time.Duration, updatesDuration time.Duration) {
		mu.Lock()
		stats.callsDuration += callsDuration
		stats.updatesDuration += updatesDuration
		mu.Unlock()
	}
	recordSimulateError := func(err error) {
		mu.Lock()
		stats.simulateErrorCount++
		syncErr = errors.Join(syncErr, err)
		mu.Unlock()
	}
	recordUpdateError := func(err error) {
		mu.Lock()
		stats.updateErrorCount++
		syncErr = errors.Join(syncErr, err)
		mu.Unlock()
	}
	recordError := func(err error) {
		mu.Lock()
		syncErr = errors.Join(syncErr, err)
		mu.Unlock()
	}

	for worker := 0; worker < stats.concurrency; worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				if err := ctx.Err(); err != nil {
					return
				}

				simulateCallStartedAt := time.Now()
				simulateResult, err := s.projectSimulator.SimulatePrimary(
					ctx,
					contractSnapshot.ProjectQueries[job.index].MsgCaller,
					contractSnapshot.ProjectContracts[job.index],
					snapshots[job.index].WethPair.ContractAddress,
					snapshots[job.index].UsdtPair.ContractAddress,
					simulationStates[job.index],
				)
				simulateCallsDuration := time.Since(simulateCallStartedAt)
				if err != nil {
					recordSimulateError(err)
					recordDurations(simulateCallsDuration, 0)
					continue
				}

				simulateUpdateStartedAt := time.Now()
				contract := contractSnapshot.ProjectContracts[job.index]
				project, ok, getErr := s.registry.GetProject(ctx, contract)
				if getErr != nil {
					recordUpdateError(getErr)
					recordDurations(simulateCallsDuration, time.Since(simulateUpdateStartedAt))
					continue
				}
				if !ok || project == nil {
					recordDurations(simulateCallsDuration, time.Since(simulateUpdateStartedAt))
					continue
				}
				metaState := project.Meta
				metaState.CreatorResult = simulateResult
				if err := s.registry.UpdateProjectMetaState(ctx, contract, &metaState); err != nil {
					recordUpdateError(err)
				}
				simulateUpdatesDuration := time.Since(simulateUpdateStartedAt)
				recordDurations(simulateCallsDuration, simulateUpdatesDuration)
			}
		}()
	}

enqueueJobs:
	for i := range contractSnapshot.ProjectContracts {
		select {
		case jobs <- projectSimulateJob{index: i}:
		case <-ctx.Done():
			recordError(ctx.Err())
			break enqueueJobs
		}
	}
	close(jobs)
	wg.Wait()

	return stats, syncErr
}

func (s *projectSyncImpl) fetchSourceCode(ctx context.Context, event *Project) (string, string, error) {
	response, err := s.apiFetcher.GetSourceCode(ctx, event.Meta.Contract.String())
	if err != nil {
		return "", "", err
	}
	if len(response.Result) == 0 {
		return "", "", errors.New("etherscan getsourcecode returned empty result")
	}
	return response.Result[0].SourceCode, response.Result[0].ABI, nil
}
