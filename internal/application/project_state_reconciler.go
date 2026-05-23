package application

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	log "github.com/sirupsen/logrus"
	"github.com/useryege/athena/internal/application/evm"
	"github.com/useryege/athena/internal/application/sourcequality"
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
	projectCache          ProjectSnapshotCache
	projectStore          appstore.ProjectStore
	fetcher               evm.AthenaFetcher
	simulator             ProjectSimulator
	apiFetcher            ethereumapi.EthereumAPI
	sourceQualityAnalyzer sourcequality.Analyzer

	persistencePublisher PersistenceEventPublisher
	codeAtFunc           func(ctx context.Context, contract common.Address) ([]byte, error)
	policyTriggerCh      chan<- common.Address

	jobSem                 chan struct{}
	wg                     sync.WaitGroup
	suppressPolicyTriggers atomic.Bool
}

func NewProjectStateReconciler(
	projectCache ProjectSnapshotCache,
	projectStore appstore.ProjectStore,
	fetcher evm.AthenaFetcher,
	simulator ProjectSimulator,
	apiFetcher ethereumapi.EthereumAPI,
	sourceQualityAnalyzer sourcequality.Analyzer,
	persistencePublisher PersistenceEventPublisher,
	codeAtFunc func(ctx context.Context, contract common.Address) ([]byte, error),
	policyTriggerCh chan<- common.Address,
) ProjectStateReconciler {
	return &projectStateReconcilerImpl{
		projectCache:          projectCache,
		projectStore:          projectStore,
		fetcher:               fetcher,
		simulator:             simulator,
		apiFetcher:            apiFetcher,
		sourceQualityAnalyzer: sourceQualityAnalyzer,
		persistencePublisher:  persistencePublisher,
		codeAtFunc:            codeAtFunc,
		policyTriggerCh:       policyTriggerCh,
		jobSem:                make(chan struct{}, reconcilerDefaultConcurrency),
	}
}

func (r *projectStateReconcilerImpl) Start(ctx context.Context) error {
	jobs := r.reconcilerJobs()

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

func (r *projectStateReconcilerImpl) ReconcileOnce(ctx context.Context) error {
	if r == nil {
		return nil
	}
	return r.reconcileOnceJobs(ctx, r.reconcilerJobs())
}

func (r *projectStateReconcilerImpl) reconcileOnceJobs(ctx context.Context, jobs []reconcilerJob) error {
	r.suppressPolicyTriggers.Store(true)
	defer r.suppressPolicyTriggers.Store(false)
	for _, job := range jobs {
		if err := ctx.Err(); err != nil {
			return err
		}
		jobStartedAt := time.Now()
		logger := log.WithFields(log.Fields{
			"component": "project_state_reconciler",
			"phase":     "reconcile_once",
			"job":       job.name,
			"interval":  job.interval.String(),
		})
		logger.Info("project reconciler job started")
		if err := r.runJobOnce(ctx, job); err != nil {
			logger.WithFields(log.Fields{
				"duration": time.Since(jobStartedAt).String(),
				"error":    err.Error(),
			}).Warn("project reconciler job failed")
			return err
		}
		logger.WithField("duration", time.Since(jobStartedAt).String()).Info("project reconciler job completed")
	}
	return nil
}

func (r *projectStateReconcilerImpl) reconcilerJobs() []reconcilerJob {
	return []reconcilerJob{
		{name: "state_refresh_active", interval: activeProjectStateRefreshInterval, run: r.refreshActiveProjectStates},
		{name: "simulation_refresh_active", interval: activeProjectSimulationRefreshInterval, run: r.refreshActiveProjectSimulations},
		{name: "sourcecode_refresh_active", interval: activeProjectSourceCodeRefreshInterval, run: r.refreshActiveProjectSourceCodes},
		{name: "source_quality_refresh_active", interval: sourceCodeRefreshInterval, run: r.refreshActiveProjectSourceQualityReports},
		{name: "code_bin_hash_refresh_active", interval: sourceCodeRefreshInterval, run: r.refreshActiveProjectCodeBinHashes},
	}
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
	if err := r.runJobOnce(ctx, job); err != nil {
		log.WithFields(log.Fields{
			"component": "project_state_reconciler",
			"job":       job.name,
			"error":     err.Error(),
		}).Warn("project reconciler job failed")
	}
}

func (r *projectStateReconcilerImpl) runJobOnce(ctx context.Context, job reconcilerJob) error {
	if err := r.acquireSemaphore(ctx); err != nil {
		return err
	}
	defer r.releaseSemaphore()

	if err := job.run(ctx); err != nil {
		return err
	}
	return nil
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
		changed, err := r.projectCache.UpdateProject(ctx, contract, func(current *Project, exists bool) (*Project, bool, error) {
			if !exists || current == nil {
				return nil, false, nil
			}
			current.Runtime.ChainState = nextState
			return current, true, nil
		})
		if err != nil {
			return err
		}
		if changed {
			r.triggerPolicyEvaluation(contract, "refresh_project_state")
		}
	}
	return nil
}

func (r *projectStateReconcilerImpl) refreshActiveProjectSimulations(ctx context.Context) error {
	return r.refreshProjectSimulations(ctx, refreshTargetActive)
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
		if !ok || latest == nil {
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
		if err := r.persistProjectCreatorResult(ctx, contract, result); err != nil {
			log.WithFields(log.Fields{
				"component": "project_state_reconciler",
				"contract":  contract.Hex(),
				"error":     err.Error(),
			}).Warn("failed to persist project creator result")
			continue
		}
		fetchedAt := time.Now().UTC()

		changed, err := r.projectCache.UpdateProject(ctx, contract, func(current *Project, exists bool) (*Project, bool, error) {
			if !exists || current == nil {
				return nil, false, nil
			}
			current.Runtime.CreatorResult = result
			current.Runtime.CreatorResultFetchedAt = fetchedAt
			return current, true, nil
		})
		if err != nil {
			return err
		}
		if changed {
			r.triggerPolicyEvaluation(contract, "refresh_project_simulation")
		}
	}
	return nil
}

func (r *projectStateReconcilerImpl) refreshActiveProjectSourceCodes(ctx context.Context) error {
	return r.refreshProjectSourceCodes(ctx, refreshTargetActive)
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

func (r *projectStateReconcilerImpl) refreshActiveProjectCodeBinHashes(ctx context.Context) error {
	return r.refreshProjectCodeBinHashes(ctx, refreshTargetActive)
}

func (r *projectStateReconcilerImpl) refreshActiveProjectSourceQualityReports(ctx context.Context) error {
	return r.refreshProjectSourceQualityReports(ctx, refreshTargetActive)
}

func (r *projectStateReconcilerImpl) refreshProjectSourceQualityReports(ctx context.Context, target refreshTarget) error {
	if r.sourceQualityAnalyzer == nil {
		return nil
	}
	projects, err := r.listProjectsByTarget(ctx, target)
	if err != nil {
		return err
	}
	for _, project := range projects {
		if err := ctx.Err(); err != nil {
			return err
		}
		if project == nil || project.Meta.SourceCode == "" || project.Meta.SourceQualityReport != "" || !project.Meta.SourceQualityReportFetchedAt.IsZero() {
			continue
		}
		report, err := r.sourceQualityAnalyzer.AnalyzeContractSource(ctx, project.Meta.SourceCode)
		if err != nil {
			log.WithFields(log.Fields{
				"component": "project_state_reconciler",
				"contract":  project.Meta.Contract.Hex(),
				"error":     err.Error(),
			}).Warn("failed to analyze project source quality")
			continue
		}
		report = strings.TrimSpace(report)
		if err := r.persistProjectSourceQualityReport(ctx, project.Meta.Contract, report); err != nil {
			log.WithFields(log.Fields{
				"component": "project_state_reconciler",
				"contract":  project.Meta.Contract.Hex(),
				"error":     err.Error(),
			}).Warn("failed to persist project source quality report")
			continue
		}
		fetchedAt := time.Now().UTC()
		_, err = r.projectCache.UpdateProject(ctx, project.Meta.Contract, func(current *Project, exists bool) (*Project, bool, error) {
			if !exists || current == nil || current.Meta.SourceCode == "" || current.Meta.SourceQualityReport != "" || !current.Meta.SourceQualityReportFetchedAt.IsZero() {
				return nil, false, nil
			}
			current.Meta.SourceQualityReport = report
			current.Meta.SourceQualityReportFetchedAt = fetchedAt
			return current, true, nil
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *projectStateReconcilerImpl) refreshProjectCodeBinHashes(ctx context.Context, target refreshTarget) error {
	projects, err := r.listProjectsByTarget(ctx, target)
	if err != nil {
		return err
	}
	for _, project := range projects {
		if err := ctx.Err(); err != nil {
			return err
		}
		if project == nil || !project.Meta.CodeBinHashFetchedAt.IsZero() {
			continue
		}
		code, err := r.fetchContractBytecode(ctx, project.Meta.Contract)
		if err != nil {
			continue
		}
		codeBinHash := common.Hash{}
		if len(code) > 0 {
			codeBinHash = crypto.Keccak256Hash(code)
		}
		fetchedAt := time.Now().UTC()
		shouldPersistCodeBinHash := false
		changed, err := r.projectCache.UpdateProject(ctx, project.Meta.Contract, func(current *Project, exists bool) (*Project, bool, error) {
			if !exists || current == nil || !current.Meta.CodeBinHashFetchedAt.IsZero() {
				return nil, false, nil
			}
			current.Meta.CodeBinHash = codeBinHash
			current.Meta.CodeBinHashFetchedAt = fetchedAt
			shouldPersistCodeBinHash = true
			return current, true, nil
		})
		if err != nil {
			continue
		}
		if shouldPersistCodeBinHash {
			if err := r.persistProjectCodeBinHash(ctx, project.Meta.Contract, codeBinHash); err != nil {
				continue
			}
		}
		if changed {
			r.triggerPolicyEvaluation(project.Meta.Contract, "refresh_project_code_bin_hash")
		}
	}
	return nil
}

func (r *projectStateReconcilerImpl) triggerPolicyEvaluation(contract common.Address, source string) {
	if r == nil || r.policyTriggerCh == nil || contract == (common.Address{}) {
		return
	}
	if r.suppressPolicyTriggers.Load() {
		return
	}
	select {
	case r.policyTriggerCh <- contract:
	default:
		log.WithFields(log.Fields{
			"component": "project_state_reconciler",
			"contract":  contract.Hex(),
			"source":    source,
		}).Warn("project policy trigger channel is full, dropping trigger")
	}
}

func (r *projectStateReconcilerImpl) listProjectsByTarget(ctx context.Context, target refreshTarget) ([]*Project, error) {
	return r.projectCache.ListActiveProjects(ctx)
}

func (r *projectStateReconcilerImpl) fetchProjectSourceCodeBatch(ctx context.Context, projects []*Project, target refreshTarget) error {
	for _, project := range projects {
		if err := ctx.Err(); err != nil {
			return err
		}
		if project == nil || !project.Meta.SourceCodeFetchedAt.IsZero() || r.apiFetcher == nil {
			continue
		}

		sourceCode, _, fetchErr := r.fetchSourceCode(ctx, project)
		if fetchErr != nil {
			continue
		}
		sourceCodeHash := common.Hash{}
		if sourceCode != "" {
			sourceCodeHash = crypto.Keccak256Hash([]byte(sourceCode))
		}
		fetchedAt := time.Now().UTC()

		shouldPersistSourceCode := false
		changed, err := r.projectCache.UpdateProject(ctx, project.Meta.Contract, func(current *Project, exists bool) (*Project, bool, error) {
			if !exists || current == nil || !current.Meta.SourceCodeFetchedAt.IsZero() {
				return nil, false, nil
			}
			current.Meta.SourceCode = sourceCode
			current.Meta.SourceCodeHash = sourceCodeHash
			current.Meta.SourceCodeFetchedAt = fetchedAt
			shouldPersistSourceCode = true
			return current, true, nil
		})
		if err != nil {
			continue
		}
		if changed {
			r.triggerPolicyEvaluation(project.Meta.Contract, "refresh_project_sourcecode")
		}

		if !shouldPersistSourceCode {
			continue
		}
		if err := r.persistProjectSourceCode(ctx, project.Meta.Contract, sourceCode); err != nil {
			continue
		}
		if sourceCode != "" {
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
	}
	return nil
}

func (r *projectStateReconcilerImpl) persistProjectSourceCode(ctx context.Context, contract common.Address, sourceCode string) error {
	if r.persistencePublisher == nil {
		return nil
	}
	return r.persistencePublisher.PublishProjectSourceCodeUpdate(ctx, contract, sourceCode)
}

func (r *projectStateReconcilerImpl) persistProjectCodeBinHash(ctx context.Context, contract common.Address, codeBinHash common.Hash) error {
	if r.persistencePublisher == nil {
		return nil
	}
	return r.persistencePublisher.PublishProjectCodeBinHashUpdate(ctx, contract, codeBinHash)
}

func (r *projectStateReconcilerImpl) persistProjectSourceQualityReport(ctx context.Context, contract common.Address, report string) error {
	if r.persistencePublisher == nil {
		return nil
	}
	return r.persistencePublisher.PublishProjectSourceQualityReportUpdate(ctx, contract, report)
}

func (r *projectStateReconcilerImpl) persistProjectCreatorResult(ctx context.Context, contract common.Address, result SimulateResult) error {
	if r.persistencePublisher == nil {
		return nil
	}
	return r.persistencePublisher.PublishProjectCreatorResultUpdate(ctx, contract, result)
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
