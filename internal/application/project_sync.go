package application

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/useryege/athena/internal/application/evm"
	athenacontract "github.com/useryege/athena/pkg/abi/ATHENA"
	"github.com/useryege/athena/util/ethereumapi"
)

const (
	defaultDelayedFetchConcurrency    = 10
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
	delayedFetchSem  chan struct{}
}

func NewProjectSync(
	nodeClient *ethclient.Client,
	registry ProjectRegistry,
	fetcher evm.AthenaFetcher,
	apiFetcher ethereumapi.EthereumAPI,
	projectSimulator ProjectSimulator,
	delayedFetchSem chan struct{},
) ProjectSync {
	if delayedFetchSem == nil {
		delayedFetchSem = make(chan struct{}, defaultDelayedFetchConcurrency)
	}

	return &projectSyncImpl{
		nodeClient:       nodeClient,
		registry:         registry,
		fetcher:          fetcher,
		apiFetcher:       apiFetcher,
		projectSimulator: projectSimulator,
		delayedFetchSem:  delayedFetchSem,
	}
}

func (s *projectSyncImpl) SyncSourceCodeOnce(ctx context.Context, event *Project) (bool, error) {
	if event.SourceCode.SourceCode.IsReady() && event.SourceCode.SourceCodeABI.IsReady() {
		return true, nil
	}

	now := time.Now()
	sourceCode, sourceCodeABI, err := s.fetchSourceCode(ctx, event)
	if err != nil {
		event.SourceCode.SourceCode.MarkFailed(err, now)
		event.SourceCode.SourceCodeABI.MarkFailed(err, now)
		return false, err
	}

	event.SourceCode.SourceCode.MarkReady(sourceCode, now)
	event.SourceCode.SourceCodeABI.MarkReady(sourceCodeABI, now)
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
	refs, err := s.registry.ListProjectContracts(ctx)
	fields["listProjectContractsDuration"] = time.Since(listStartedAt)
	if err != nil {
		return err
	}
	fields["tokenContractCount"] = len(refs)
	if len(refs) == 0 {
		return nil
	}

	buildTokenContractsStartedAt := time.Now()
	tokenContracts := make([]common.Address, 0, len(refs))
	queries := make([]athenacontract.AthenaProjectQuery, 0, len(refs))
	for _, ref := range refs {
		tokenContracts = append(tokenContracts, ref.Contract)
		queries = append(queries, athenacontract.AthenaProjectQuery{
			TokenContract: ref.Contract,
			MsgCaller:     ref.Creator,
		})
	}
	fields["buildTokenContractsDuration"] = time.Since(buildTokenContractsStartedAt)

	fetchStartedAt := time.Now()
	var snapshots []athenacontract.AthenaProject
	var simulationStates []athenacontract.AthenaSimulationState
	if s.projectSimulator == nil {
		snapshots, err = s.fetcher.FetchProjects(ctx, tokenContracts)
	} else {
		projects, err := s.fetcher.FetchProjectsWithSimulationState(ctx, queries)
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
	if len(snapshots) != len(refs) {
		return fmt.Errorf("athena list returned %d projects for %d token contracts", len(snapshots), len(refs))
	}

	buildStatesStartedAt := time.Now()
	states := make(map[uuid.UUID]athenacontract.AthenaProject, len(refs))
	for i, ref := range refs {
		snapshot := snapshots[i]
		if snapshot.TokenContract != (common.Address{}) && snapshot.TokenContract != ref.Contract {
			return fmt.Errorf("athena list result %d token contract = %s, want %s", i, snapshot.TokenContract, ref.Contract)
		}
		states[ref.ProjectID] = snapshot
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
		stats, err := s.syncProjectSimulateStates(ctx, refs, snapshots, simulationStates)
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
	ref   ProjectContractRef
}

func (s *projectSyncImpl) syncProjectSimulateStates(
	ctx context.Context,
	refs []ProjectContractRef,
	snapshots []athenacontract.AthenaProject,
	simulationStates []athenacontract.AthenaSimulationState,
) (projectSimulateStats, error) {
	stats := projectSimulateStats{
		projectCount: len(refs),
		concurrency:  defaultProjectSimulateConcurrency,
	}
	if len(simulationStates) != len(refs) {
		return stats, fmt.Errorf("athena list returned %d simulation states for %d token contracts", len(simulationStates), len(refs))
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

				now := time.Now()
				state := ProjectSimulateState{}

				simulateCallStartedAt := time.Now()
				simulateResult, err := s.projectSimulator.SimulatePrimary(
					ctx,
					job.ref.Creator,
					job.ref.Contract,
					snapshots[job.index].Pair.ContractAddress,
					simulationStates[job.index],
				)
				simulateCallsDuration := time.Since(simulateCallStartedAt)
				if err != nil {
					state.CreatorResult.MarkFailed(err, now)
					recordSimulateError(err)
				} else {
					state.CreatorResult.MarkReady(simulateResult, now)
				}

				simulateUpdateStartedAt := time.Now()
				if err := s.registry.UpdateProjectSimulateState(ctx, job.ref.ProjectID, &state); err != nil {
					recordUpdateError(err)
				}
				simulateUpdatesDuration := time.Since(simulateUpdateStartedAt)
				recordDurations(simulateCallsDuration, simulateUpdatesDuration)
			}
		}()
	}

enqueueJobs:
	for i, ref := range refs {
		select {
		case jobs <- projectSimulateJob{index: i, ref: ref}:
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
	if err := s.acquireDelayedFetch(ctx); err != nil {
		return "", "", err
	}
	defer s.releaseDelayedFetch()

	response, err := s.apiFetcher.GetSourceCode(ctx, event.Meta.Contract.String())
	if err != nil {
		return "", "", err
	}
	if len(response.Result) == 0 {
		return "", "", errors.New("etherscan getsourcecode returned empty result")
	}
	return response.Result[0].SourceCode, response.Result[0].ABI, nil
}

func (s *projectSyncImpl) acquireDelayedFetch(ctx context.Context) error {
	select {
	case s.delayedFetchSem <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *projectSyncImpl) releaseDelayedFetch() {
	<-s.delayedFetchSem
}
