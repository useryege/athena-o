package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	log "github.com/sirupsen/logrus"
	"github.com/useryege/athena/internal/application/evm"
	"github.com/useryege/athena/internal/application/sourcecode"
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
	store        appstore.Store
	fetcher      evm.AthenaFetcher
	simulator    ProjectSimulator
	apiFetcher   ethereumapi.EthereumAPI

	sourceAnalyzer  sourcecode.Analyzer
	sourceBlacklist sourceCodeBlacklistLister

	persistencePublisher PersistenceEventPublisher
	codeAtFunc           func(ctx context.Context, contract common.Address) ([]byte, error)

	jobSem chan struct{}
	wg     sync.WaitGroup
}

type sourceCodeBlacklistLister interface {
	List(ctx context.Context) ([]string, error)
}

func NewProjectStateReconciler(
	projectCache ProjectSnapshotCache,
	store appstore.Store,
	fetcher evm.AthenaFetcher,
	simulator ProjectSimulator,
	apiFetcher ethereumapi.EthereumAPI,
	sourceAnalyzer sourcecode.Analyzer,
	sourceBlacklist sourceCodeBlacklistLister,
	persistencePublisher PersistenceEventPublisher,
	codeAtFunc func(ctx context.Context, contract common.Address) ([]byte, error),
) ProjectStateReconciler {
	return &projectStateReconcilerImpl{
		projectCache:         projectCache,
		store:                store,
		fetcher:              fetcher,
		simulator:            simulator,
		apiFetcher:           apiFetcher,
		sourceAnalyzer:       sourceAnalyzer,
		sourceBlacklist:      sourceBlacklist,
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
		{name: "bin_blacklist_scan", interval: binBlacklistScanInterval, run: r.scanAllProjectsForBINBlacklist},
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
			current.ChainState = nextState
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

		wethPairContract := latest.ChainState.WethPair.ContractAddress
		usdtPairContract := latest.ChainState.UsdtPair.ContractAddress
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
			current.Meta.CreatorResult = result
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
	fields, err := r.sourceCodeBlacklistFields(ctx)
	if err != nil {
		return err
	}

	projects, err := r.listProjectsByTarget(ctx, target)
	if err != nil {
		return err
	}
	if err := r.processProjectSourceCodeBatch(ctx, projects, fields, target); err != nil {
		return err
	}
	return nil
}

func (r *projectStateReconcilerImpl) scanAllProjectsForBINBlacklist(ctx context.Context) error {
	blacklistStore, ok := r.store.(appstore.BytecodeBlacklistContractStore)
	if !ok || blacklistStore == nil {
		return nil
	}

	records, err := blacklistStore.ListBytecodeBlacklistContracts(ctx)
	if err != nil {
		return err
	}
	if len(records) == 0 {
		return nil
	}

	blacklistedCodeHashes := make(map[common.Hash]struct{}, len(records))
	for _, item := range records {
		blacklistedCodeHashes[item.CodeHash] = struct{}{}
	}

	projects, err := r.listAllProjectsForBINBlacklistScan(ctx)
	if err != nil {
		return err
	}
	for _, project := range projects {
		if err := ctx.Err(); err != nil {
			return err
		}
		if project == nil {
			continue
		}

		contract := project.Meta.Contract
		code, err := r.fetchContractBytecode(ctx, contract)
		if err != nil || len(code) == 0 {
			continue
		}

		codeHash := crypto.Keccak256Hash(code)
		if _, matched := blacklistedCodeHashes[codeHash]; !matched {
			continue
		}

		_ = r.applyBINBlacklistAutoArchive(ctx, contract, codeHash)
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

func (r *projectStateReconcilerImpl) listAllProjectsForBINBlacklistScan(ctx context.Context) ([]*Project, error) {
	activeProjects, err := r.projectCache.ListActiveProjects(ctx)
	if err != nil {
		return nil, err
	}
	archivedProjects, err := r.listAllArchivedProjects(ctx)
	if err != nil {
		return nil, err
	}

	all := make([]*Project, 0, len(activeProjects)+len(archivedProjects))
	seen := make(map[common.Address]struct{}, len(activeProjects)+len(archivedProjects))
	appendUnique := func(items []*Project) {
		for _, project := range items {
			if project == nil {
				continue
			}
			contract := project.Meta.Contract
			if _, ok := seen[contract]; ok {
				continue
			}
			seen[contract] = struct{}{}
			all = append(all, project)
		}
	}
	appendUnique(activeProjects)
	appendUnique(archivedProjects)
	return all, nil
}

func (r *projectStateReconcilerImpl) applyBINBlacklistAutoArchive(ctx context.Context, contract common.Address, codeHash common.Hash) error {
	current, exists, err := r.projectCache.GetProject(ctx, contract)
	if err != nil || !exists || current == nil {
		return err
	}

	now := time.Now().UTC()
	if !current.Meta.IsArchived && r.persistencePublisher != nil {
		if err := r.persistencePublisher.PublishProjectArchive(ctx, contract); err == nil {
			_, _ = r.projectCache.UpdateProject(ctx, contract, func(latest *Project, exists bool) (*Project, bool, error) {
				if !exists || latest == nil || latest.Meta.IsArchived {
					return nil, false, nil
				}
				latest.Meta.IsArchived = true
				latest.Meta.ArchivedAt = now
				return latest, true, nil
			})
		}
	}

	payload := "{}"
	if encoded, err := json.Marshal(map[string]string{
		"source":    "bin_blacklist_scan",
		"code_hash": codeHash.Hex(),
	}); err == nil {
		payload = string(encoded)
	}

	return r.persistProjectEventLog(ctx, appstore.ProjectEventLog{
		Contract:       contract,
		EventType:      projectEventTypeAutoArchiveBIN,
		OccurredAt:     now,
		Message:        "Project auto archived by BIN blacklist match",
		Payload:        payload,
		IdempotencyKey: projectEventIdempotencyAutoArchiveBIN,
	})
}

func (r *projectStateReconcilerImpl) processProjectSourceCodeBatch(ctx context.Context, projects []*Project, fields []string, target refreshTarget) error {
	for _, project := range projects {
		if err := ctx.Err(); err != nil {
			return err
		}
		if project == nil {
			continue
		}

		sourceCode := project.Meta.SourceCode
		if sourceCode == "" && r.apiFetcher != nil {
			fetchedSourceCode, _, fetchErr := r.fetchSourceCode(ctx, project)
			if fetchErr != nil {
				continue
			}
			sourceCode = fetchedSourceCode
		}

		var analyzedSourceCode string
		var blacklistReport sourcecode.BlacklistReport
		if sourceCode != "" && r.sourceAnalyzer != nil && project.Meta.SourceCodeBlacklist.ResolvedAt.IsZero() {
			analyzedSourceCode = sourceCode
			blacklistReport = r.sourceAnalyzer.AnalyzeSourceCode(sourceCode, fields)
		}

		shouldPersistSourceCode := false
		_, err := r.projectCache.UpdateProject(ctx, project.Meta.Contract, func(current *Project, exists bool) (*Project, bool, error) {
			if !exists || current == nil || !matchesTarget(current.Meta.IsArchived, target) {
				return nil, false, nil
			}

			changed := false
			if sourceCode != "" && current.Meta.SourceCode == "" {
				current.Meta.SourceCode = sourceCode
				changed = true
				shouldPersistSourceCode = true
			}

			if analyzedSourceCode != "" && current.Meta.SourceCode == analyzedSourceCode && current.Meta.SourceCodeBlacklist.ResolvedAt.IsZero() {
				current.Meta.SourceCodeBlacklist = blacklistReport
				changed = true
			}

			return current, changed, nil
		})
		if err != nil {
			continue
		}
		if shouldPersistSourceCode {
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

func (r *projectStateReconcilerImpl) sourceCodeBlacklistFields(ctx context.Context) ([]string, error) {
	if r.sourceBlacklist == nil {
		return nil, nil
	}
	return r.sourceBlacklist.List(ctx)
}

func (r *projectStateReconcilerImpl) fetchContractBytecode(ctx context.Context, contract common.Address) ([]byte, error) {
	if r.codeAtFunc != nil {
		return r.codeAtFunc(ctx, contract)
	}
	return nil, errors.New("codeAt function is not configured")
}
