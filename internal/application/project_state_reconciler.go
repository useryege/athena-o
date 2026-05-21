package application

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	log "github.com/sirupsen/logrus"
	"github.com/useryege/athena/internal/application/evm"
	appstore "github.com/useryege/athena/internal/application/store"
	"github.com/useryege/athena/util/ethereumapi"
)

const reconcilerDefaultConcurrency = 2

type reconcilerJob struct {
	name     string
	interval time.Duration
	run      func(ctx context.Context) error
}

type projectStateReconcilerImpl struct {
	projectCache ProjectSnapshotCache
	fetcher      evm.AthenaFetcher
	simulator    ProjectSimulator
	apiFetcher   ethereumapi.EthereumAPI

	persistencePublisher PersistenceEventPublisher
	codeAtFunc           func(ctx context.Context, contract common.Address) ([]byte, error)

	jobSem chan struct{}
	wg     sync.WaitGroup
}

func NewProjectStateReconciler(
	projectCache ProjectSnapshotCache,
	fetcher evm.AthenaFetcher,
	simulator ProjectSimulator,
	apiFetcher ethereumapi.EthereumAPI,
	persistencePublisher PersistenceEventPublisher,
	codeAtFunc func(ctx context.Context, contract common.Address) ([]byte, error),
) ProjectStateReconciler {
	return &projectStateReconcilerImpl{
		projectCache:         projectCache,
		fetcher:              fetcher,
		simulator:            simulator,
		apiFetcher:           apiFetcher,
		persistencePublisher: persistencePublisher,
		codeAtFunc:           codeAtFunc,
		jobSem:               make(chan struct{}, reconcilerDefaultConcurrency),
	}
}

func (r *projectStateReconcilerImpl) Start(ctx context.Context) error {
	jobs := []reconcilerJob{
		{name: "state_refresh_active", interval: activeProjectStateRefreshInterval, run: r.refreshActiveProjectStates},
		{name: "state_refresh_archived", interval: archivedProjectRefreshInterval, run: r.refreshArchivedProjectStates},
		{name: "simulation_refresh_active", interval: activeProjectSimulationRefreshInterval, run: r.refreshActiveProjectSimulations},
		{name: "simulation_refresh_archived", interval: archivedProjectRefreshInterval, run: r.refreshArchivedProjectSimulations},
		{name: "sourcecode_refresh_active", interval: sourceCodeRefreshInterval, run: r.refreshActiveProjectSourceCodes},
		{name: "sourcecode_refresh_archived", interval: archivedProjectRefreshInterval, run: r.refreshArchivedProjectSourceCodes},
		{name: "runtime_code_hash_refresh_active", interval: sourceCodeRefreshInterval, run: r.refreshActiveProjectRuntimeCodeHashes},
		{name: "runtime_code_hash_refresh_archived", interval: archivedProjectRefreshInterval, run: r.refreshArchivedProjectRuntimeCodeHashes},
	}

	for i := range jobs {
		job := jobs[i]
		r.wg.Add(1)
		go func() {
			defer r.wg.Done()
			r.runLoop(ctx, job)
		}()
	}
	return nil
}

func (r *projectStateReconcilerImpl) Stop() error {
	r.wg.Wait()
	return nil
}

func (r *projectStateReconcilerImpl) runLoop(ctx context.Context, job reconcilerJob) {
	ticker := time.NewTicker(job.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.executeJob(ctx, job)
		}
	}
}

func (r *projectStateReconcilerImpl) executeJob(ctx context.Context, job reconcilerJob) {
	if err := r.acquireSemaphore(ctx); err != nil {
		return
	}
	defer r.releaseSemaphore()

	if err := job.run(ctx); err != nil {
		log.WithFields(log.Fields{
			"component": "project_state_reconciler",
			"job":       job.name,
			"error":     err.Error(),
		}).Warn("project reconciler job failed")
	}
}

func (r *projectStateReconcilerImpl) acquireSemaphore(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case r.jobSem <- struct{}{}:
		return nil
	}
}

func (r *projectStateReconcilerImpl) releaseSemaphore() {
	select {
	case <-r.jobSem:
	default:
	}
}

func (r *projectStateReconcilerImpl) refreshActiveProjectStates(ctx context.Context) error {
	return r.refreshProjectStates(ctx, refreshTargetActive)
}

func (r *projectStateReconcilerImpl) refreshArchivedProjectStates(ctx context.Context) error {
	return r.refreshProjectStates(ctx, refreshTargetArchived)
}

func (r *projectStateReconcilerImpl) refreshProjectStates(ctx context.Context, target refreshTarget) error {
	projects, err := r.listProjectsByTarget(ctx, target)
	if err != nil {
		return err
	}
	if len(projects) == 0 {
		return nil
	}

	queries, contracts := buildProjectQueries(projects)
	if len(queries) == 0 {
		return nil
	}

	fetched, err := r.fetcher.FetchProjectsWithSimulationState(ctx, queries)
	if err != nil {
		return err
	}
	if len(fetched) != len(queries) {
		return fmt.Errorf("fetch projects with simulation state size mismatch: got %d want %d", len(fetched), len(queries))
	}

	for i, contract := range contracts {
		nextState := fetched[i].Project
		_, err := r.projectCache.UpdateProject(ctx, contract, func(current *Project, exists bool) (*Project, bool, error) {
			if !exists || current == nil || !matchesTarget(current.Meta.IsArchived, target) {
				return nil, false, nil
			}
			current.Runtime.ChainState = nextState
			return current, true, nil
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *projectStateReconcilerImpl) refreshActiveProjectSimulations(ctx context.Context) error {
	return r.refreshProjectSimulations(ctx, refreshTargetActive)
}

func (r *projectStateReconcilerImpl) refreshArchivedProjectSimulations(ctx context.Context) error {
	return r.refreshProjectSimulations(ctx, refreshTargetArchived)
}

func (r *projectStateReconcilerImpl) refreshProjectSimulations(ctx context.Context, target refreshTarget) error {
	if r.simulator == nil {
		return nil
	}

	projects, err := r.listProjectsByTarget(ctx, target)
	if err != nil {
		return err
	}
	if len(projects) == 0 {
		return nil
	}

	queries, contracts := buildProjectQueries(projects)
	if len(queries) == 0 {
		return nil
	}

	states, err := r.fetcher.FetchSimulationStates(ctx, queries)
	if err != nil {
		return err
	}
	if len(states) != len(queries) {
		return fmt.Errorf("fetch simulation states size mismatch: got %d want %d", len(states), len(queries))
	}

	for i, contract := range contracts {
		latest, ok, err := r.projectCache.GetProject(ctx, contract)
		if err != nil {
			return err
		}
		if !ok || latest == nil || !matchesTarget(latest.Meta.IsArchived, target) {
			continue
		}

		wethPairContract := latest.Runtime.ChainState.WethPair.ContractAddress
		usdtPairContract := latest.Runtime.ChainState.UsdtPair.ContractAddress
		if wethPairContract == (common.Address{}) || usdtPairContract == (common.Address{}) {
			continue
		}

		result, err := r.simulator.SimulatePrimary(
			ctx,
			latest.Meta.Creator,
			latest.Meta.Contract,
			wethPairContract,
			usdtPairContract,
			states[i],
		)
		if err != nil {
			continue
		}

		_, err = r.projectCache.UpdateProject(ctx, contract, func(current *Project, exists bool) (*Project, bool, error) {
			if !exists || current == nil || !matchesTarget(current.Meta.IsArchived, target) {
				return nil, false, nil
			}
			current.Runtime.CreatorResult = result
			return current, true, nil
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *projectStateReconcilerImpl) refreshActiveProjectSourceCodes(ctx context.Context) error {
	return r.refreshProjectSourceCodes(ctx, refreshTargetActive)
}

func (r *projectStateReconcilerImpl) refreshArchivedProjectSourceCodes(ctx context.Context) error {
	return r.refreshProjectSourceCodes(ctx, refreshTargetArchived)
}

func (r *projectStateReconcilerImpl) refreshProjectSourceCodes(ctx context.Context, target refreshTarget) error {
	projects, err := r.listProjectsByTarget(ctx, target)
	if err != nil {
		return err
	}
	if err := r.fetchProjectSourceCodeBatch(ctx, projects, target); err != nil {
		return err
	}
	return nil
}

func (r *projectStateReconcilerImpl) refreshActiveProjectRuntimeCodeHashes(ctx context.Context) error {
	return r.refreshProjectRuntimeCodeHashes(ctx, refreshTargetActive)
}

func (r *projectStateReconcilerImpl) refreshArchivedProjectRuntimeCodeHashes(ctx context.Context) error {
	return r.refreshProjectRuntimeCodeHashes(ctx, refreshTargetArchived)
}

func (r *projectStateReconcilerImpl) refreshProjectRuntimeCodeHashes(ctx context.Context, target refreshTarget) error {
	projects, err := r.listProjectsByTarget(ctx, target)
	if err != nil {
		return err
	}
	for _, project := range projects {
		if err := ctx.Err(); err != nil {
			return err
		}
		if project == nil || project.Runtime.RuntimeCodeHash != (common.Hash{}) {
			continue
		}
		code, err := r.fetchContractBytecode(ctx, project.Meta.Contract)
		if err != nil || len(code) == 0 {
			continue
		}
		codeHash := crypto.Keccak256Hash(code)
		_, err = r.projectCache.UpdateProject(ctx, project.Meta.Contract, func(current *Project, exists bool) (*Project, bool, error) {
			if !exists || current == nil || !matchesTarget(current.Meta.IsArchived, target) || current.Runtime.RuntimeCodeHash != (common.Hash{}) {
				return nil, false, nil
			}
			current.Runtime.RuntimeCodeHash = codeHash
			return current, true, nil
		})
		if err != nil {
			continue
		}
	}
	return nil
}

func (r *projectStateReconcilerImpl) listProjectsByTarget(ctx context.Context, target refreshTarget) ([]*Project, error) {
	if target == refreshTargetActive {
		return r.projectCache.ListActiveProjects(ctx)
	}
	return r.listAllArchivedProjects(ctx)
}

func (r *projectStateReconcilerImpl) listAllArchivedProjects(ctx context.Context) ([]*Project, error) {
	projects := make([]*Project, 0)
	page := int32(1)

	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		items, total, _, pageSize, err := r.projectCache.ListArchivedProjects(ctx, page, sourceCodeScanPageSize)
		if err != nil {
			return nil, err
		}
		projects = append(projects, items...)
		if len(items) == 0 {
			return projects, nil
		}
		if int64(page)*int64(pageSize) >= total {
			return projects, nil
		}
		page++
	}
}

func (r *projectStateReconcilerImpl) fetchProjectSourceCodeBatch(ctx context.Context, projects []*Project, target refreshTarget) error {
	for _, project := range projects {
		if err := ctx.Err(); err != nil {
			return err
		}
		if project == nil || project.Meta.SourceCode != "" || r.apiFetcher == nil {
			continue
		}

		sourceCode, _, fetchErr := r.fetchSourceCode(ctx, project)
		if fetchErr != nil || sourceCode == "" {
			continue
		}

		shouldPersistSourceCode := false
		_, err := r.projectCache.UpdateProject(ctx, project.Meta.Contract, func(current *Project, exists bool) (*Project, bool, error) {
			if !exists || current == nil || !matchesTarget(current.Meta.IsArchived, target) || current.Meta.SourceCode != "" {
				return nil, false, nil
			}
			current.Meta.SourceCode = sourceCode
			shouldPersistSourceCode = true
			return current, true, nil
		})
		if err != nil {
			continue
		}

		if !shouldPersistSourceCode {
			continue
		}
		if err := r.persistProjectSourceCode(ctx, project.Meta.Contract, sourceCode); err != nil {
			continue
		}
		if err := r.persistProjectEventLog(ctx, appstore.ProjectEventLog{
			Contract:       project.Meta.Contract,
			EventType:      projectEventTypeOpenSource,
			OccurredAt:     time.Now().UTC(),
			Message:        "Contract source code opened",
			Payload:        "{}",
			IdempotencyKey: projectEventIdempotencyOpenSource,
		}); err != nil {
			continue
		}
	}
	return nil
}

func (r *projectStateReconcilerImpl) persistProjectSourceCode(ctx context.Context, contract common.Address, sourceCode string) error {
	if sourceCode == "" || r.persistencePublisher == nil {
		return nil
	}
	return r.persistencePublisher.PublishProjectSourceCodeUpdate(ctx, contract, sourceCode)
}

func (r *projectStateReconcilerImpl) persistProjectEventLog(ctx context.Context, item appstore.ProjectEventLog) error {
	if r.persistencePublisher == nil {
		return nil
	}
	return r.persistencePublisher.PublishProjectEventLog(ctx, item)
}

func (r *projectStateReconcilerImpl) fetchSourceCode(ctx context.Context, project *Project) (string, string, error) {
	response, err := r.apiFetcher.GetSourceCode(ctx, project.Meta.Contract.String())
	if err != nil {
		return "", "", err
	}
	if len(response.Result) == 0 {
		return "", "", errors.New("etherscan getsourcecode returned empty result")
	}
	return response.Result[0].SourceCode, response.Result[0].ABI, nil
}

func (r *projectStateReconcilerImpl) fetchContractBytecode(ctx context.Context, contract common.Address) ([]byte, error) {
	if r.codeAtFunc != nil {
		return r.codeAtFunc(ctx, contract)
	}
	return nil, errors.New("codeAt function is not configured")
}
