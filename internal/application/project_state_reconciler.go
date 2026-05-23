package application

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/big"
	"sort"
	"strings"
	"sync"
	"time"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	log "github.com/sirupsen/logrus"
	"github.com/useryege/athena/internal/application/evm"
	"github.com/useryege/athena/internal/application/sourcequality"
	appstore "github.com/useryege/athena/internal/application/store"
	erc20contract "github.com/useryege/athena/pkg/abi/ERC20"
	"github.com/useryege/athena/util/ethereumapi"
)

const (
	reconcilerDefaultConcurrency       = 2
	reconcilerSchedulerTickInterval    = 5 * time.Second
	reconcilerScheduledRefreshInterval = time.Minute
	reconcilerCatchUpTTL               = 3 * time.Minute
	reconcilerFollowHeadsTTL           = 360 * time.Minute
)

var creatorHistoricalProjectRetryDelays = []time.Duration{
	200 * time.Millisecond,
	500 * time.Millisecond,
	time.Second,
}

var (
	errGenesisReceiptNil   = errors.New("project transaction receipt is nil")
	erc20TransferTopicHash = crypto.Keccak256Hash([]byte("Transfer(address,address,uint256)"))
)

type scheduledProject struct {
	candidate  DiscoveredProjectCandidate
	source     ProjectDiscoverySource
	interval   time.Duration
	expiresAt  time.Time
	nextRunAt  time.Time
	processing bool
}

type GenesisWalletShare struct {
	Wallet   common.Address
	Amount   *big.Int
	Ratio    string
	RatioBPS int64
}

type genesisWalletShareLog struct {
	Wallet   string `json:"wallet"`
	Amount   string `json:"amount"`
	Ratio    string `json:"ratio"`
	RatioBPS int64  `json:"ratio_bps"`
}

type projectStateReconcilerImpl struct {
	projectCache          ProjectSnapshotCache
	discoveryNodeClient   projectDiscoveryNodeClient
	projectStore          appstore.ProjectStore
	fetcher               evm.AthenaFetcher
	simulator             ProjectSimulator
	apiFetcher            ethereumapi.EthereumAPI
	sourceQualityAnalyzer sourcequality.Analyzer

	persistencePublisher PersistenceEventPublisher
	codeAtFunc           func(ctx context.Context, contract common.Address) ([]byte, error)
	policyEngine         ProjectPolicyEngine

	jobSem chan struct{}
	wg     sync.WaitGroup

	scheduleMu sync.Mutex
	scheduled  map[common.Address]*scheduledProject
}

func NewProjectStateReconciler(
	projectCache ProjectSnapshotCache,
	discoveryNodeClient projectDiscoveryNodeClient,
	projectStore appstore.ProjectStore,
	fetcher evm.AthenaFetcher,
	simulator ProjectSimulator,
	apiFetcher ethereumapi.EthereumAPI,
	sourceQualityAnalyzer sourcequality.Analyzer,
	persistencePublisher PersistenceEventPublisher,
	codeAtFunc func(ctx context.Context, contract common.Address) ([]byte, error),
	policyEngine ProjectPolicyEngine,
) ProjectStateReconciler {
	return &projectStateReconcilerImpl{
		projectCache:          projectCache,
		discoveryNodeClient:   discoveryNodeClient,
		projectStore:          projectStore,
		fetcher:               fetcher,
		simulator:             simulator,
		apiFetcher:            apiFetcher,
		sourceQualityAnalyzer: sourceQualityAnalyzer,
		persistencePublisher:  persistencePublisher,
		codeAtFunc:            codeAtFunc,
		policyEngine:          policyEngine,
		jobSem:                make(chan struct{}, reconcilerDefaultConcurrency),
		scheduled:             map[common.Address]*scheduledProject{},
	}
}

func (r *projectStateReconcilerImpl) Start(ctx context.Context) error {
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		r.runScheduler(ctx)
	}()
	return nil
}

func (r *projectStateReconcilerImpl) Stop() error {
	r.wg.Wait()
	return nil
}

func (r *projectStateReconcilerImpl) InitProject(ctx context.Context, candidates []DiscoveredProjectCandidate) error {
	if len(candidates) == 0 {
		return nil
	}
	if r.fetcher == nil {
		return errors.New("athena fetcher is not configured")
	}

	projects := make([]*Project, 0, len(candidates))
	validCandidates := make([]DiscoveredProjectCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		if candidate.Source == "" {
			candidate.Source = ProjectDiscoverySourceCatchUp
		}
		project := projectFromCandidate(candidate)
		if project == nil {
			continue
		}
		projects = append(projects, project)
		validCandidates = append(validCandidates, candidate)
	}

	queries, _ := buildProjectQueries(projects)
	if len(queries) == 0 {
		return nil
	}
	snapshots, err := r.fetcher.FetchProjects(ctx, queries)
	if err != nil {
		log.WithFields(log.Fields{
			"error":   err,
			"queries": queries,
		}).Error("failed to fetch projects")
		return err
	}
	if len(snapshots) != len(queries) {
		return fmt.Errorf("athena list returned %d projects for %d queries", len(snapshots), len(queries))
	}
	for i, snapshot := range snapshots {
		project := projects[i]
		candidate := validCandidates[i]
		if snapshot.TokenContract != (common.Address{}) && snapshot.TokenContract != project.Meta.Contract {
			return fmt.Errorf("athena list result token contract = %s, want %s", snapshot.TokenContract, project.Meta.Contract)
		}
		if !snapshot.Token.IsValidERC20 {
			continue
		}
		project.Meta.ChainState = snapshot
		if err := r.persistProjectMeta(ctx, project.Meta); err != nil {
			return fmt.Errorf("failed to persist project %s: %w", project.Meta.Contract.Hex(), err)
		}
		_, err := r.projectCache.UpdateProject(ctx, project.Meta.Contract, func(current *Project, exists bool) (*Project, bool, error) {
			if exists && current != nil {
				return nil, false, nil
			}
			return project, true, nil
		})
		if err != nil {
			return fmt.Errorf("failed to cache project %s: %w", project.Meta.Contract.Hex(), err)
		}
		r.scheduleProject(candidate)
	}
	return nil
}

func projectFromCandidate(candidate DiscoveredProjectCandidate) *Project {
	if candidate.Contract == (common.Address{}) {
		return nil
	}
	return &Project{
		Meta: ProjectMeta{
			BlockTime:   candidate.BlockTime,
			BlockNumber: candidate.BlockNumber,
			TxIndex:     candidate.TxIndex,
			TxHash:      candidate.TxHash,
			Contract:    candidate.Contract,
			Creator:     candidate.Creator,
			GenesisTx:   candidate.Tx,
		},
	}
}

func (r *projectStateReconcilerImpl) scheduleProject(candidate DiscoveredProjectCandidate) {
	if r == nil || candidate.Contract == (common.Address{}) {
		return
	}
	source := candidate.Source
	if source == "" {
		source = ProjectDiscoverySourceCatchUp
	}
	now := time.Now().UTC()
	ttl := reconcilerCatchUpTTL
	if source == ProjectDiscoverySourceFollowHeads {
		ttl = reconcilerFollowHeadsTTL
	}
	item := &scheduledProject{
		candidate: candidate,
		source:    source,
		interval:  reconcilerScheduledRefreshInterval,
		expiresAt: now.Add(ttl),
		nextRunAt: now.Add(reconcilerScheduledRefreshInterval),
	}
	r.scheduleMu.Lock()
	defer r.scheduleMu.Unlock()
	if r.scheduled == nil {
		r.scheduled = map[common.Address]*scheduledProject{}
	}
	if existing := r.scheduled[candidate.Contract]; existing != nil {
		if existing.expiresAt.After(item.expiresAt) {
			item.expiresAt = existing.expiresAt
		}
		if existing.nextRunAt.Before(item.nextRunAt) {
			item.nextRunAt = existing.nextRunAt
		}
		item.processing = existing.processing
	}
	r.scheduled[candidate.Contract] = item
}

func (r *projectStateReconcilerImpl) runDueProjects(ctx context.Context, now time.Time) {
	for _, contract := range r.dueProjectContracts(now) {
		if err := r.runScheduledProject(ctx, contract); err != nil {
			log.WithFields(log.Fields{
				"component": "project_state_reconciler",
				"contract":  contract.Hex(),
				"error":     err.Error(),
			}).Warn("project scheduled refresh failed")
		}
		r.finishScheduledProject(contract, time.Now().UTC())
	}
}

func (r *projectStateReconcilerImpl) dueProjectContracts(now time.Time) []common.Address {
	r.scheduleMu.Lock()
	defer r.scheduleMu.Unlock()
	contracts := make([]common.Address, 0)
	for contract, item := range r.scheduled {
		if item == nil || !now.Before(item.expiresAt) {
			delete(r.scheduled, contract)
			continue
		}
		if item.processing || now.Before(item.nextRunAt) {
			continue
		}
		item.processing = true
		contracts = append(contracts, contract)
	}
	return contracts
}

func (r *projectStateReconcilerImpl) finishScheduledProject(contract common.Address, now time.Time) {
	r.scheduleMu.Lock()
	defer r.scheduleMu.Unlock()
	item := r.scheduled[contract]
	if item == nil {
		return
	}
	if !now.Before(item.expiresAt) {
		delete(r.scheduled, contract)
		return
	}
	item.processing = false
	item.nextRunAt = now.Add(item.interval)
}

func (r *projectStateReconcilerImpl) runScheduler(ctx context.Context) {
	ticker := time.NewTicker(reconcilerSchedulerTickInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.runDueProjects(ctx, time.Now().UTC())
		}
	}
}

func (r *projectStateReconcilerImpl) runScheduledProject(ctx context.Context, contract common.Address) error {
	if err := r.acquireSemaphore(ctx); err != nil {
		return err
	}
	defer r.releaseSemaphore()
	return r.refreshProject(ctx, contract)
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

func (r *projectStateReconcilerImpl) refreshProject(ctx context.Context, contract common.Address) error {
	if r == nil || r.projectCache == nil || contract == (common.Address{}) {
		return nil
	}
	project, ok, err := r.projectCache.GetProject(ctx, contract)
	if err != nil {
		return err
	}
	if !ok || project == nil {
		return nil
	}
	if err := r.refreshProjectChainAndSimulation(ctx, project); err != nil {
		return err
	}
	if err := r.refreshProjectGenesisWallets(ctx, contract); err != nil {
		return err
	}
	if err := r.refreshProjectCreatorHistoricalProjects(ctx, contract); err != nil {
		return err
	}
	if err := r.refreshProjectSourceCode(ctx, contract); err != nil {
		return err
	}
	if err := r.refreshProjectCodeBinHash(ctx, contract); err != nil {
		return err
	}
	if err := r.refreshProjectSourceQualityReport(ctx, contract); err != nil {
		return err
	}
	if r.policyEngine != nil {
		if _, err := r.policyEngine.EvaluateProject(ctx, contract); err != nil {
			return err
		}
	}
	return nil
}

func (r *projectStateReconcilerImpl) refreshProjectChainAndSimulation(ctx context.Context, project *Project) error {
	if r.fetcher == nil || project == nil {
		return nil
	}
	queries, contracts := buildProjectQueries([]*Project{project})
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
	contract := contracts[0]
	nextState := fetched[0].Project
	_, err = r.projectCache.UpdateProject(ctx, contract, func(current *Project, exists bool) (*Project, bool, error) {
		if !exists || current == nil {
			return nil, false, nil
		}
		current.Meta.ChainState = nextState
		return current, true, nil
	})
	if err != nil {
		return err
	}
	if r.simulator == nil {
		return nil
	}
	latest, ok, err := r.projectCache.GetProject(ctx, contract)
	if err != nil {
		return err
	}
	if !ok || latest == nil {
		return nil
	}
	wethPairContract := latest.Meta.ChainState.WethPair.ContractAddress
	usdtPairContract := latest.Meta.ChainState.UsdtPair.ContractAddress
	if wethPairContract == (common.Address{}) || usdtPairContract == (common.Address{}) {
		return nil
	}
	result, err := r.simulator.SimulatePrimary(
		ctx,
		latest.Meta.Creator,
		latest.Meta.Contract,
		wethPairContract,
		usdtPairContract,
		fetched[0].SimulationState,
	)
	if err != nil {
		return nil
	}
	if err := r.persistProjectCreatorResult(ctx, contract, result); err != nil {
		log.WithFields(log.Fields{
			"component": "project_state_reconciler",
			"contract":  contract.Hex(),
			"error":     err.Error(),
		}).Warn("failed to persist project creator result")
		return nil
	}
	fetchedAt := time.Now().UTC()
	_, err = r.projectCache.UpdateProject(ctx, contract, func(current *Project, exists bool) (*Project, bool, error) {
		if !exists || current == nil {
			return nil, false, nil
		}
		if current.Meta.CreatorResult == result && !current.Meta.CreatorResultFetchedAt.IsZero() {
			return nil, false, nil
		}
		current.Meta.CreatorResult = result
		current.Meta.CreatorResultFetchedAt = fetchedAt
		return current, true, nil
	})
	if err != nil {
		return err
	}
	return nil
}

func (r *projectStateReconcilerImpl) refreshProjectGenesisWallets(ctx context.Context, contract common.Address) error {
	project, ok, err := r.projectCache.GetProject(ctx, contract)
	if err != nil {
		return err
	}
	if !ok || project == nil || !project.Meta.GenesisWalletsFetchedAt.IsZero() {
		return nil
	}
	genesisWalletShares, err := r.fetchGenesisWallets(ctx, project, project.Meta.ChainState.Token.TotalSupply)
	if err != nil {
		log.WithFields(log.Fields{
			"component":   "project_state_reconciler",
			"contract":    project.Meta.Contract.Hex(),
			"txHash":      project.Meta.TxHash.Hex(),
			"blockNumber": project.Meta.BlockNumber,
			"creator":     project.Meta.Creator.Hex(),
		}).WithError(err).Warn("failed to fetch genesis wallets")
		return nil
	}
	logGenesisWalletShares(project, genesisWalletShares)
	if err := r.publishProjectGenesisWallets(ctx, project, project.Meta.ChainState.Token.TotalSupply, genesisWalletShares); err != nil {
		log.WithFields(log.Fields{
			"component": "project_state_reconciler",
			"contract":  project.Meta.Contract.Hex(),
			"error":     err.Error(),
		}).Warn("failed to persist project genesis wallets")
		return nil
	}
	genesisWallets := genesisWalletMetasFromShares(genesisWalletShares)
	fetchedAt := time.Now().UTC()
	_, err = r.projectCache.UpdateProject(ctx, contract, func(current *Project, exists bool) (*Project, bool, error) {
		if !exists || current == nil || !current.Meta.GenesisWalletsFetchedAt.IsZero() {
			return nil, false, nil
		}
		current.Meta.GenesisWallets = genesisWallets
		current.Meta.GenesisWalletsFetchedAt = fetchedAt
		return current, true, nil
	})
	if err != nil {
		return err
	}
	return nil
}

func (r *projectStateReconcilerImpl) refreshProjectCreatorHistoricalProjects(ctx context.Context, contract common.Address) error {
	project, ok, err := r.projectCache.GetProject(ctx, contract)
	if err != nil {
		return err
	}
	if !ok || project == nil || !project.Meta.CreatorHistoricalProjectsFetchedAt.IsZero() {
		return nil
	}
	historicalProjects, fetched, err := r.fetchCreatorHistoricalProjects(ctx, project)
	if err != nil {
		log.WithFields(log.Fields{
			"component": "project_state_reconciler",
			"contract":  project.Meta.Contract.Hex(),
			"creator":   project.Meta.Creator.Hex(),
			"error":     err.Error(),
		}).Warn("failed to fetch project creator historical projects")
		return nil
	}
	if !fetched {
		return nil
	}
	project.Meta.CreatorHistoricalProjects = historicalProjects
	if err := r.publishProjectCreatorHistoricalProjects(ctx, project); err != nil {
		log.WithFields(log.Fields{
			"component": "project_state_reconciler",
			"contract":  project.Meta.Contract.Hex(),
			"error":     err.Error(),
		}).Warn("failed to persist project creator historical projects")
		return nil
	}
	fetchedAt := time.Now().UTC()
	_, err = r.projectCache.UpdateProject(ctx, contract, func(current *Project, exists bool) (*Project, bool, error) {
		if !exists || current == nil || !current.Meta.CreatorHistoricalProjectsFetchedAt.IsZero() {
			return nil, false, nil
		}
		current.Meta.CreatorHistoricalProjects = historicalProjects
		current.Meta.CreatorHistoricalProjectsFetchedAt = fetchedAt
		return current, true, nil
	})
	if err != nil {
		return err
	}
	return nil
}

func (r *projectStateReconcilerImpl) refreshProjectSourceCode(ctx context.Context, contract common.Address) error {
	if r.apiFetcher == nil {
		return nil
	}
	project, ok, err := r.projectCache.GetProject(ctx, contract)
	if err != nil {
		return err
	}
	if !ok || project == nil || !project.Meta.SourceCodeFetchedAt.IsZero() {
		return nil
	}
	sourceCode, _, fetchErr := r.fetchSourceCode(ctx, project)
	if fetchErr != nil {
		return nil
	}
	sourceCodeHash := common.Hash{}
	if sourceCode != "" {
		sourceCodeHash = crypto.Keccak256Hash([]byte(sourceCode))
	}
	if err := r.persistProjectSourceCode(ctx, contract, sourceCode); err != nil {
		return nil
	}
	if sourceCode != "" {
		if err := r.persistProjectEventLog(ctx, appstore.ProjectEventLog{
			Contract:       contract,
			EventType:      projectEventTypeOpenSource,
			OccurredAt:     time.Now().UTC(),
			Message:        "Contract source code opened",
			Payload:        "{}",
			IdempotencyKey: projectEventIdempotencyOpenSource,
		}); err != nil {
			return nil
		}
	}
	fetchedAt := time.Now().UTC()
	_, err = r.projectCache.UpdateProject(ctx, contract, func(current *Project, exists bool) (*Project, bool, error) {
		if !exists || current == nil || !current.Meta.SourceCodeFetchedAt.IsZero() {
			return nil, false, nil
		}
		current.Meta.SourceCode = sourceCode
		current.Meta.SourceCodeHash = sourceCodeHash
		current.Meta.SourceCodeFetchedAt = fetchedAt
		return current, true, nil
	})
	if err != nil {
		return err
	}
	return nil
}

func (r *projectStateReconcilerImpl) refreshProjectCodeBinHash(ctx context.Context, contract common.Address) error {
	project, ok, err := r.projectCache.GetProject(ctx, contract)
	if err != nil {
		return err
	}
	if !ok || project == nil || !project.Meta.CodeBinHashFetchedAt.IsZero() {
		return nil
	}
	code, err := r.fetchContractBytecode(ctx, contract)
	if err != nil {
		return nil
	}
	codeBinHash := common.Hash{}
	if len(code) > 0 {
		codeBinHash = crypto.Keccak256Hash(code)
	}
	if err := r.persistProjectCodeBinHash(ctx, contract, codeBinHash); err != nil {
		return nil
	}
	fetchedAt := time.Now().UTC()
	_, err = r.projectCache.UpdateProject(ctx, contract, func(current *Project, exists bool) (*Project, bool, error) {
		if !exists || current == nil || !current.Meta.CodeBinHashFetchedAt.IsZero() {
			return nil, false, nil
		}
		current.Meta.CodeBinHash = codeBinHash
		current.Meta.CodeBinHashFetchedAt = fetchedAt
		return current, true, nil
	})
	if err != nil {
		return err
	}
	return nil
}

func (r *projectStateReconcilerImpl) refreshProjectSourceQualityReport(ctx context.Context, contract common.Address) error {
	if r.sourceQualityAnalyzer == nil {
		return nil
	}
	project, ok, err := r.projectCache.GetProject(ctx, contract)
	if err != nil {
		return err
	}
	if !ok || project == nil || project.Meta.SourceCode == "" || project.Meta.SourceQualityReport != "" || !project.Meta.SourceQualityReportFetchedAt.IsZero() {
		return nil
	}
	report, err := r.sourceQualityAnalyzer.AnalyzeContractSource(ctx, project.Meta.SourceCode)
	if err != nil {
		log.WithFields(log.Fields{
			"component": "project_state_reconciler",
			"contract":  contract.Hex(),
			"error":     err.Error(),
		}).Warn("failed to analyze project source quality")
		return nil
	}
	report = strings.TrimSpace(report)
	if err := r.persistProjectSourceQualityReport(ctx, contract, report); err != nil {
		log.WithFields(log.Fields{
			"component": "project_state_reconciler",
			"contract":  contract.Hex(),
			"error":     err.Error(),
		}).Warn("failed to persist project source quality report")
		return nil
	}
	fetchedAt := time.Now().UTC()
	_, err = r.projectCache.UpdateProject(ctx, contract, func(current *Project, exists bool) (*Project, bool, error) {
		if !exists || current == nil || current.Meta.SourceCode == "" || current.Meta.SourceQualityReport != "" || !current.Meta.SourceQualityReportFetchedAt.IsZero() {
			return nil, false, nil
		}
		current.Meta.SourceQualityReport = report
		current.Meta.SourceQualityReportFetchedAt = fetchedAt
		return current, true, nil
	})
	return err
}

func (r *projectStateReconcilerImpl) persistProjectSourceCode(ctx context.Context, contract common.Address, sourceCode string) error {
	if r.persistencePublisher == nil {
		return nil
	}
	return r.persistencePublisher.PublishProjectSourceCodeUpdate(ctx, contract, sourceCode)
}

func (r *projectStateReconcilerImpl) persistProjectMeta(ctx context.Context, meta ProjectMeta) error {
	if r.persistencePublisher == nil {
		return nil
	}
	return r.persistencePublisher.PublishProjectMetaSave(ctx, projectMetaToStore(meta))
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

func (r *projectStateReconcilerImpl) fetchCreatorHistoricalProjects(ctx context.Context, project *Project) ([]common.Address, bool, error) {
	if project == nil || project.Meta.Creator == (common.Address{}) || project.Meta.Contract == (common.Address{}) {
		return nil, false, nil
	}
	if r.projectStore == nil {
		log.WithFields(log.Fields{
			"component": "project_state_reconciler",
			"contract":  project.Meta.Contract.Hex(),
			"creator":   project.Meta.Creator.Hex(),
		}).Warn("project store is not configured; creator historical projects will be empty")
		return nil, false, nil
	}

	var lastErr error
	attempts := len(creatorHistoricalProjectRetryDelays) + 1
	for attempt := 1; attempt <= attempts; attempt++ {
		metas, err := r.projectStore.ListProjectMetasByCreatorBefore(ctx, project.Meta.Creator, project.Meta.BlockNumber, project.Meta.TxIndex)
		if err == nil {
			return projectContractsFromMetasPreserveOrder(metas, project.Meta.Contract), true, nil
		}
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, false, ctxErr
		}
		lastErr = err
		if attempt > len(creatorHistoricalProjectRetryDelays) {
			break
		}
		delay := creatorHistoricalProjectRetryDelays[attempt-1]
		log.WithFields(log.Fields{
			"component":     "project_state_reconciler",
			"contract":      project.Meta.Contract.Hex(),
			"creator":       project.Meta.Creator.Hex(),
			"attempt":       attempt,
			"next_retry_in": delay.String(),
			"error":         err.Error(),
		}).Warn("list creator historical projects failed, retrying")
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return nil, false, ctx.Err()
		case <-timer.C:
		}
	}

	log.WithFields(log.Fields{
		"component": "project_state_reconciler",
		"contract":  project.Meta.Contract.Hex(),
		"creator":   project.Meta.Creator.Hex(),
		"attempts":  attempts,
		"error":     lastErr.Error(),
	}).Warn("list creator historical projects failed; continuing with empty result")
	return nil, false, nil
}

func projectContractsFromMetasPreserveOrder(metas []appstore.ProjectMeta, currentContract common.Address) []common.Address {
	if len(metas) == 0 {
		return nil
	}
	contracts := make([]common.Address, 0, len(metas))
	for _, meta := range metas {
		contracts = appendProjectContractPreserveOrder(contracts, meta.Contract, currentContract)
	}
	return contracts
}

func appendProjectContractPreserveOrder(contracts []common.Address, contract common.Address, currentContract common.Address) []common.Address {
	if contract == (common.Address{}) || contract == currentContract {
		return contracts
	}
	for _, existing := range contracts {
		if existing == contract {
			return contracts
		}
	}
	return append(contracts, contract)
}

func genesisWalletMetasFromShares(shares []GenesisWalletShare) []GenesisWalletMeta {
	if len(shares) == 0 {
		return nil
	}
	metas := make([]GenesisWalletMeta, 0, len(shares))
	for i, item := range shares {
		amount := new(big.Int)
		if item.Amount != nil {
			amount = new(big.Int).Set(item.Amount)
		}
		metas = append(metas, GenesisWalletMeta{
			Wallet:    item.Wallet,
			NetAmount: amount,
			RatioBPS:  item.RatioBPS,
			RankIndex: int32(i),
		})
	}
	return metas
}

func logGenesisWalletShares(project *Project, shares []GenesisWalletShare) {
	if project == nil {
		return
	}
	genesisWalletHex := make([]string, 0, len(shares))
	genesisWalletShareItems := make([]genesisWalletShareLog, 0, len(shares))
	for _, item := range shares {
		genesisWalletHex = append(genesisWalletHex, item.Wallet.Hex())
		amount := "0"
		if item.Amount != nil {
			amount = item.Amount.String()
		}
		genesisWalletShareItems = append(genesisWalletShareItems, genesisWalletShareLog{
			Wallet:   item.Wallet.Hex(),
			Amount:   amount,
			Ratio:    item.Ratio,
			RatioBPS: item.RatioBPS,
		})
	}
	log.WithFields(log.Fields{
		"contract":            project.Meta.Contract.Hex(),
		"txHash":              project.Meta.TxHash.Hex(),
		"blockNumber":         project.Meta.BlockNumber,
		"creator":             project.Meta.Creator.Hex(),
		"genesisWalletCount":  len(genesisWalletHex),
		"genesisWallets":      genesisWalletHex,
		"genesisWalletShares": genesisWalletShareItems,
	}).Info("extracted genesis wallets from project creation receipt")
}

func (r *projectStateReconcilerImpl) fetchGenesisWallets(ctx context.Context, project *Project, totalSupply *big.Int) ([]GenesisWalletShare, error) {
	logs, err := r.fetchGenesisWalletsFromReceipt(ctx, project)
	if err == nil {
		return extractGenesisWalletShares(logs, project.Meta.Contract, totalSupply), nil
	}
	if !shouldFallbackToLogs(err) {
		return nil, err
	}
	log.WithFields(log.Fields{
		"contract":    project.Meta.Contract.Hex(),
		"txHash":      project.Meta.TxHash.Hex(),
		"blockNumber": project.Meta.BlockNumber,
		"creator":     project.Meta.Creator.Hex(),
		"fallback":    "eth_getLogs",
		"reason":      "receipt_not_found",
	}).WithError(err).Warn("failed to fetch genesis wallets from receipt, attempting logs fallback")
	fallbackLogs, fallbackErr := r.fetchGenesisWalletsFromLogsFallback(ctx, project)
	if fallbackErr != nil {
		log.WithFields(log.Fields{
			"contract":    project.Meta.Contract.Hex(),
			"txHash":      project.Meta.TxHash.Hex(),
			"blockNumber": project.Meta.BlockNumber,
			"creator":     project.Meta.Creator.Hex(),
			"fallback":    "eth_getLogs",
		}).WithError(fallbackErr).Warn("failed to fetch genesis wallets from logs fallback")
		return nil, fmt.Errorf("fetch genesis wallets by logs fallback: %w", fallbackErr)
	}
	return extractGenesisWalletShares(fallbackLogs, project.Meta.Contract, totalSupply), nil
}

func (r *projectStateReconcilerImpl) fetchGenesisWalletsFromReceipt(ctx context.Context, project *Project) ([]*types.Log, error) {
	if project == nil {
		return nil, errors.New("project is nil")
	}
	if r.discoveryNodeClient == nil {
		return nil, errors.New("project discovery node client is not configured")
	}
	txHash := projectTxHash(project)
	if txHash == (common.Hash{}) {
		return nil, errors.New("project tx hash is empty")
	}
	receipt, err := r.discoveryNodeClient.TransactionReceipt(ctx, txHash)
	if err != nil {
		return nil, fmt.Errorf("fetch transaction receipt %s: %w", txHash.Hex(), err)
	}
	if receipt == nil {
		return nil, fmt.Errorf("%w: %s", errGenesisReceiptNil, txHash.Hex())
	}
	return receipt.Logs, nil
}

func (r *projectStateReconcilerImpl) fetchGenesisWalletsFromLogsFallback(ctx context.Context, project *Project) ([]*types.Log, error) {
	if project == nil {
		return nil, errors.New("project is nil")
	}
	if r.discoveryNodeClient == nil {
		return nil, errors.New("project discovery node client is not configured")
	}
	txHash := projectTxHash(project)
	if txHash == (common.Hash{}) {
		return nil, errors.New("project tx hash is empty")
	}
	blockNumber := new(big.Int).SetUint64(project.Meta.BlockNumber)
	query := ethereum.FilterQuery{
		FromBlock: blockNumber,
		ToBlock:   blockNumber,
		Addresses: []common.Address{project.Meta.Contract},
		Topics: [][]common.Hash{
			{erc20TransferTopicHash},
		},
	}
	logs, err := r.discoveryNodeClient.FilterLogs(ctx, query)
	if err != nil {
		return nil, err
	}
	return filterLogsByTxHash(logs, txHash), nil
}

func shouldFallbackToLogs(err error) bool {
	return errors.Is(err, ethereum.NotFound) || errors.Is(err, errGenesisReceiptNil)
}

func projectTxHash(project *Project) common.Hash {
	if project == nil {
		return common.Hash{}
	}
	txHash := project.Meta.TxHash
	if txHash == (common.Hash{}) && project.Meta.GenesisTx != nil {
		txHash = project.Meta.GenesisTx.Hash()
	}
	return txHash
}

func filterLogsByTxHash(logs []types.Log, txHash common.Hash) []*types.Log {
	filtered := make([]*types.Log, 0, len(logs))
	for i := range logs {
		if logs[i].TxHash != txHash {
			continue
		}
		entry := logs[i]
		filtered = append(filtered, &entry)
	}
	return filtered
}

func extractGenesisWalletShares(logs []*types.Log, tokenContract common.Address, totalSupply *big.Int) []GenesisWalletShare {
	filterer, err := erc20contract.NewERC20Filterer(tokenContract, nil)
	if err != nil {
		return nil
	}
	candidates := make(map[common.Address]struct{})
	netBalance := make(map[common.Address]*big.Int)
	for _, entry := range logs {
		if entry == nil {
			continue
		}
		if entry.Address != tokenContract {
			continue
		}
		transferEvent, err := filterer.ParseTransfer(*entry)
		if err != nil {
			continue
		}
		if transferEvent.Tokens == nil || transferEvent.Tokens.Sign() <= 0 {
			continue
		}

		if transferEvent.To != (common.Address{}) {
			candidates[transferEvent.To] = struct{}{}
			if _, exists := netBalance[transferEvent.To]; !exists {
				netBalance[transferEvent.To] = new(big.Int)
			}
			netBalance[transferEvent.To].Add(netBalance[transferEvent.To], transferEvent.Tokens)
		}

		if transferEvent.From != (common.Address{}) {
			if _, exists := netBalance[transferEvent.From]; !exists {
				netBalance[transferEvent.From] = new(big.Int)
			}
			netBalance[transferEvent.From].Sub(netBalance[transferEvent.From], transferEvent.Tokens)
		}
	}

	shares := make([]GenesisWalletShare, 0, len(candidates))
	for wallet := range candidates {
		amount, exists := netBalance[wallet]
		if !exists || amount.Sign() <= 0 {
			continue
		}
		shares = append(shares, GenesisWalletShare{
			Wallet:   wallet,
			Amount:   new(big.Int).Set(amount),
			Ratio:    formatGenesisWalletRatio(amount, totalSupply),
			RatioBPS: ratioBPS(amount, totalSupply),
		})
	}
	sort.Slice(shares, func(i, j int) bool {
		amountCmp := shares[i].Amount.Cmp(shares[j].Amount)
		if amountCmp != 0 {
			return amountCmp > 0
		}
		return bytes.Compare(shares[i].Wallet.Bytes(), shares[j].Wallet.Bytes()) < 0
	})
	return shares
}

func formatGenesisWalletRatio(amount *big.Int, totalSupply *big.Int) string {
	if amount == nil || amount.Sign() <= 0 || totalSupply == nil || totalSupply.Sign() <= 0 {
		return "0.0000%"
	}
	ratio := new(big.Rat).SetFrac(amount, totalSupply)
	ratio.Mul(ratio, big.NewRat(100, 1))
	return ratio.FloatString(4) + "%"
}

func ratioBPS(amount *big.Int, totalSupply *big.Int) int64 {
	if amount == nil || amount.Sign() <= 0 || totalSupply == nil || totalSupply.Sign() <= 0 {
		return 0
	}
	numerator := new(big.Int).Mul(amount, big.NewInt(10000))
	halfDenominator := new(big.Int).Div(new(big.Int).Set(totalSupply), big.NewInt(2))
	numerator.Add(numerator, halfDenominator)
	result := new(big.Int).Div(numerator, totalSupply)
	if !result.IsInt64() {
		return math.MaxInt64
	}
	return result.Int64()
}

func (r *projectStateReconcilerImpl) publishProjectGenesisWallets(ctx context.Context, project *Project, totalSupply *big.Int, shares []GenesisWalletShare) error {
	if r.persistencePublisher == nil {
		return errors.New("persistence publisher is not configured")
	}
	if project == nil {
		return errors.New("project is nil")
	}
	txHash := projectTxHash(project)
	totalSupplyText := "0"
	if totalSupply != nil {
		totalSupplyText = totalSupply.String()
	}
	items := make([]projectGenesisWalletItemPayload, 0, len(shares))
	for i, item := range shares {
		if item.Amount == nil {
			continue
		}
		items = append(items, projectGenesisWalletItemPayload{
			Wallet:    item.Wallet.Hex(),
			NetAmount: item.Amount.String(),
			RatioBPS:  item.RatioBPS,
			RankIndex: int32(i),
		})
	}
	payload := projectGenesisWalletReplacePayload{
		Contract:          project.Meta.Contract.Hex(),
		SourceTxHash:      txHash.Hex(),
		SourceBlockNumber: project.Meta.BlockNumber,
		TotalSupply:       totalSupplyText,
		Items:             items,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal project genesis wallet replace payload: %w", err)
	}
	return r.persistencePublisher.Publish(ctx, PersistenceEvent{
		Version:    persistenceEventVersion,
		Op:         PersistenceOpProjectGenesisReplace,
		Contract:   project.Meta.Contract.Hex(),
		Payload:    data,
		OccurredAt: time.Now().UTC(),
	})
}

func (r *projectStateReconcilerImpl) publishProjectCreatorHistoricalProjects(ctx context.Context, project *Project) error {
	if r.persistencePublisher == nil {
		return errors.New("persistence publisher is not configured")
	}
	if project == nil {
		return errors.New("project is nil")
	}
	items := make([]appstore.ProjectCreatorHistoricalProject, 0, len(project.Meta.CreatorHistoricalProjects))
	for i, contract := range project.Meta.CreatorHistoricalProjects {
		if contract == (common.Address{}) {
			continue
		}
		items = append(items, appstore.ProjectCreatorHistoricalProject{
			ProjectContract:           project.Meta.Contract,
			HistoricalProjectContract: contract,
			RankIndex:                 int32(i),
		})
	}
	return r.persistencePublisher.PublishProjectCreatorHistoricalProjectsReplace(ctx, project.Meta.Contract, items)
}
