package reconcile

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
	"github.com/useryege/athena/internal/application/avelogo"
	"github.com/useryege/athena/internal/application/evm"
	appstore "github.com/useryege/athena/internal/application/store"
	solidityapiclient "github.com/useryege/athena/internal/solidity/apiclient"
	athenacontract "github.com/useryege/athena/pkg/abi/ATHENA"
	erc20contract "github.com/useryege/athena/pkg/abi/ERC20"
	utilio "github.com/useryege/athena/util/io"
)

const (
	reconcilerDefaultConcurrency       = 2
	reconcilerSchedulerTickInterval    = 5 * time.Second
	reconcilerScheduledRefreshInterval = time.Minute
	reconcilerCatchUpTTL               = 5 * time.Minute
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
	componentCache      ProjectComponentCache
	projectCache        ProjectSnapshotCache
	discoveryNodeClient projectDiscoveryNodeClient
	projectStore        appstore.ProjectStore
	fetcher             evm.AthenaFetcher
	simulator           ProjectSimulator
	aveDetailFetcher    avelogo.Fetcher
	aveChain            string
	solidityClientSet   solidityapiclient.Clientset
	chainID             int64

	persistencePublisher PersistenceEventPublisher
	policyEngine         ProjectPolicyEngine

	jobSem chan struct{}
	wg     sync.WaitGroup

	scheduleMu   sync.Mutex
	scheduled    map[common.Address]*scheduledProject
	lifecycleCtx context.Context
}

func NewProjectStateReconciler(
	componentCache ProjectComponentCache,
	discoveryNodeClient projectDiscoveryNodeClient,
	projectStore appstore.ProjectStore,
	fetcher evm.AthenaFetcher,
	simulator ProjectSimulator,
	aveDetailFetcher avelogo.Fetcher,
	aveChain string,
	solidityClientSet solidityapiclient.Clientset,
	chainID int64,
	persistencePublisher PersistenceEventPublisher,
	policyEngine ProjectPolicyEngine,
) ProjectStateReconciler {
	return &projectStateReconcilerImpl{
		componentCache:       componentCache,
		discoveryNodeClient:  discoveryNodeClient,
		projectStore:         projectStore,
		fetcher:              fetcher,
		simulator:            simulator,
		aveDetailFetcher:     aveDetailFetcher,
		aveChain:             strings.TrimSpace(aveChain),
		solidityClientSet:    solidityClientSet,
		chainID:              chainID,
		persistencePublisher: persistencePublisher,
		policyEngine:         policyEngine,
		jobSem:               make(chan struct{}, reconcilerDefaultConcurrency),
		scheduled:            map[common.Address]*scheduledProject{},
	}
}

func (r *projectStateReconcilerImpl) Start(ctx context.Context) error {
	r.scheduleMu.Lock()
	r.lifecycleCtx = ctx
	r.scheduleMu.Unlock()

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
		// Here return nil to fix the bug contract at https://bscscan.com/address/0x868e9962fb1d9346d74547603b04f3d66a6f537b
		// avoid out of gas error
		return nil
	}
	if len(snapshots) != len(queries) {
		return fmt.Errorf("athena list returned %d projects for %d queries", len(snapshots), len(queries))
	}
	fetchAt := time.Now().UTC()
	immediateRefreshContracts := make([]common.Address, 0, len(snapshots))
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
		project.Meta.WethPair = snapshot.WethPair.ContractAddress
		project.Meta.UsdtPair = snapshot.UsdtPair.ContractAddress
		project.Meta.FetchAt = fetchAt
		if err := r.persistProjectBase(ctx, project.Meta); err != nil {
			return fmt.Errorf("failed to persist project %s: %w", project.Meta.Contract.Hex(), err)
		}
		if err := r.persistProjectChainState(ctx, project.Meta.Contract, snapshot, fetchAt); err != nil {
			return fmt.Errorf("failed to persist project chain state %s: %w", project.Meta.Contract.Hex(), err)
		}
		if r.componentCache != nil {
			if err := r.componentCache.SetBase(ctx, projectBaseFromMeta(project.Meta)); err != nil {
				return fmt.Errorf("failed to cache project base %s: %w", project.Meta.Contract.Hex(), err)
			}
			if err := r.componentCache.SetChainState(ctx, projectChainStateFromSnapshot(project.Meta.Contract, snapshot, fetchAt)); err != nil {
				return fmt.Errorf("failed to cache project chain state %s: %w", project.Meta.Contract.Hex(), err)
			}
		} else if r.projectCache != nil {
			_, err := r.projectCache.UpdateProject(ctx, project.Meta.Contract, func(current *Project, exists bool) (*Project, bool, error) {
				if exists && current != nil {
					return nil, false, nil
				}
				return project, true, nil
			})
			if err != nil {
				return fmt.Errorf("failed to cache project %s: %w", project.Meta.Contract.Hex(), err)
			}
		}
		if r.componentCache != nil {
			if err := r.refreshProjectGenesisWallets(ctx, project.Meta.Contract); err != nil {
				return err
			}
			if err := r.refreshProjectCreatorHistoricalProjects(ctx, project.Meta.Contract); err != nil {
				return err
			}
			if r.policyEngine != nil {
				if _, err := r.policyEngine.EvaluateProject(ctx, project.Meta.Contract); err != nil {
					return err
				}
			}
		}
		r.scheduleProject(candidate)
		immediateRefreshContracts = append(immediateRefreshContracts, project.Meta.Contract)
	}
	for _, contract := range immediateRefreshContracts {
		r.triggerImmediateScheduledRefresh(ctx, contract)
	}
	return nil
}

func (r *projectStateReconcilerImpl) ScheduleProjects(ctx context.Context, candidates []DiscoveredProjectCandidate) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	for _, candidate := range candidates {
		if candidate.Source == "" {
			candidate.Source = ProjectDiscoverySourceCatchUp
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

func (r *projectStateReconcilerImpl) triggerImmediateScheduledRefresh(ctx context.Context, contract common.Address) {
	if r == nil || contract == (common.Address{}) {
		return
	}
	runCtx := ctx
	r.scheduleMu.Lock()
	if r.lifecycleCtx != nil {
		runCtx = r.lifecycleCtx
	}
	item := r.scheduled[contract]
	now := time.Now().UTC()
	if item == nil || !now.Before(item.expiresAt) || item.processing {
		if item != nil && !now.Before(item.expiresAt) {
			delete(r.scheduled, contract)
		}
		r.scheduleMu.Unlock()
		return
	}
	item.processing = true
	r.scheduleMu.Unlock()

	if runCtx == nil {
		runCtx = context.Background()
	}
	r.wg.Add(1)
	go func(ctx context.Context) {
		defer r.wg.Done()
		if err := r.runScheduledProject(ctx, contract); err != nil {
			log.WithFields(log.Fields{
				"component": "project_state_reconciler",
				"contract":  contract.Hex(),
				"error":     err.Error(),
			}).Warn("project immediate refresh failed")
		}
		r.finishScheduledProject(contract, time.Now().UTC())
	}(runCtx)
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
	if r == nil || contract == (common.Address{}) {
		return nil
	}
	if r.componentCache == nil && r.projectCache != nil {
		return r.refreshProjectLegacy(ctx, contract)
	}
	project, ok, err := r.getProjectBaseForRefresh(ctx, contract)
	if err != nil || !ok || project == nil {
		return err
	}
	if err := r.refreshProjectChainState(ctx, project); err != nil {
		return err
	}
	if err := r.refreshProjectSimulation(ctx, contract); err != nil {
		return err
	}
	if err := r.refreshProjectAveDetail(ctx, contract); err != nil {
		return err
	}
	if err := r.refreshProjectSoliditySource(ctx, contract); err != nil {
		return err
	}
	if r.policyEngine != nil {
		if _, err := r.policyEngine.EvaluateProject(ctx, contract); err != nil {
			return err
		}
	}
	return nil
}

func (r *projectStateReconcilerImpl) getProjectBaseForRefresh(ctx context.Context, contract common.Address) (*Project, bool, error) {
	if r.componentCache != nil {
		if base, ok, err := r.componentCache.GetBase(ctx, contract); err != nil {
			return nil, false, err
		} else if ok && base != nil {
			return projectFromBase(*base), true, nil
		}
	}
	if store, ok := r.projectStore.(appstore.ProjectBaseStore); ok && store != nil {
		base, err := store.GetProjectBaseByContract(ctx, contract)
		if err != nil || base == nil {
			return nil, false, err
		}
		if r.componentCache != nil {
			if err := r.componentCache.SetBase(ctx, *base); err != nil {
				return nil, false, err
			}
		}
		return projectFromBase(*base), true, nil
	}
	if r.projectCache == nil {
		return nil, false, nil
	}
	return r.projectCache.GetProject(ctx, contract)
}

func (r *projectStateReconcilerImpl) refreshProjectChainState(ctx context.Context, project *Project) error {
	if r.fetcher == nil || project == nil {
		return nil
	}
	queries, contracts := buildProjectQueries([]*Project{project})
	if len(queries) == 0 {
		return nil
	}
	snapshots, err := r.fetcher.FetchProjects(ctx, queries)
	if err != nil {
		return err
	}
	if len(snapshots) != len(queries) {
		return fmt.Errorf("fetch projects size mismatch: got %d want %d", len(snapshots), len(queries))
	}
	contract := contracts[0]
	item := projectChainStateFromSnapshot(contract, snapshots[0], time.Now().UTC())
	if err := r.persistProjectChainState(ctx, contract, snapshots[0], item.FetchedAt); err != nil {
		return err
	}
	if r.componentCache != nil {
		return r.componentCache.SetChainState(ctx, item)
	}
	return nil
}

func (r *projectStateReconcilerImpl) refreshProjectSimulation(ctx context.Context, contract common.Address) error {
	if r.fetcher == nil || r.simulator == nil {
		return nil
	}
	project, ok, err := r.getProjectBaseForRefresh(ctx, contract)
	if err != nil || !ok || project == nil {
		return err
	}
	chainState, ok, err := r.getProjectChainStateForRefresh(ctx, contract)
	if err != nil || !ok || chainState == nil {
		return err
	}
	project.Meta.GenesisWallets = nil
	query := athenacontract.AthenaProjectQuery{
		TokenContract: contract,
		MsgCaller:     project.Meta.Creator,
	}
	simulationState, err := r.fetcher.FetchSimulationState(ctx, query)
	if err != nil {
		return nil
	}
	wethPairContract := chainState.WethPair
	usdtPairContract := chainState.UsdtPair
	if wethPairContract == (common.Address{}) || usdtPairContract == (common.Address{}) {
		return nil
	}
	result, err := r.simulator.SimulatePrimary(
		ctx,
		project.Meta.Creator,
		contract,
		wethPairContract,
		usdtPairContract,
		simulationState,
	)
	if err != nil {
		return nil
	}
	item := appstore.ProjectSimulationResult{ProjectContract: contract, Result: simulateResultToStore(result), FetchedAt: time.Now().UTC()}
	if store, ok := r.projectStore.(appstore.ProjectSimulationStore); ok && store != nil {
		if err := store.UpsertProjectSimulationResult(ctx, item); err != nil {
			log.WithFields(log.Fields{
				"component": "project_state_reconciler",
				"contract":  contract.Hex(),
				"error":     err.Error(),
			}).Warn("failed to persist project simulation result")
			return nil
		}
	} else if err := r.persistProjectCreatorResult(ctx, contract, result); err != nil {
		return nil
	}
	if r.componentCache != nil {
		if err := r.componentCache.SetSimulation(ctx, item); err != nil {
			return err
		}
	}
	return nil
}

func (r *projectStateReconcilerImpl) getProjectChainStateForRefresh(ctx context.Context, contract common.Address) (*appstore.ProjectChainState, bool, error) {
	if r.componentCache != nil {
		if item, ok, err := r.componentCache.GetChainState(ctx, contract); err != nil {
			return nil, false, err
		} else if ok && item != nil {
			return item, true, nil
		}
	}
	if store, ok := r.projectStore.(appstore.ProjectChainStateStore); ok && store != nil {
		item, err := store.GetProjectChainState(ctx, contract)
		if err != nil || item == nil {
			return nil, false, err
		}
		if r.componentCache != nil {
			if err := r.componentCache.SetChainState(ctx, *item); err != nil {
				return nil, false, err
			}
		}
		return item, true, nil
	}
	return nil, false, nil
}

func (r *projectStateReconcilerImpl) refreshProjectLegacy(ctx context.Context, contract common.Address) error {
	if r == nil || r.projectCache == nil || contract == (common.Address{}) {
		return nil
	}
	project, ok, err := r.projectCache.GetProject(ctx, contract)
	if err != nil || !ok || project == nil {
		return err
	}
	if err := r.refreshProjectChainAndSimulation(ctx, project); err != nil {
		return err
	}
	if err := r.refreshProjectAveDetail(ctx, contract); err != nil {
		return err
	}
	if err := r.refreshProjectGenesisWallets(ctx, contract); err != nil {
		return err
	}
	if err := r.refreshProjectCreatorHistoricalProjects(ctx, contract); err != nil {
		return err
	}
	if err := r.refreshProjectSoliditySource(ctx, contract); err != nil {
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
	_, err = r.projectCache.UpdateProject(ctx, contract, func(current *Project, exists bool) (*Project, bool, error) {
		if !exists || current == nil {
			return nil, false, nil
		}
		if current.Meta.CreatorResult == result {
			return nil, false, nil
		}
		current.Meta.CreatorResult = result
		return current, true, nil
	})
	if err != nil {
		return err
	}
	return nil
}

func (r *projectStateReconcilerImpl) refreshProjectGenesisWallets(ctx context.Context, contract common.Address) error {
	if r.componentCache == nil && r.projectCache != nil {
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
		return err
	}
	if done, err := r.componentSucceeded(ctx, contract, "genesis_wallet"); err != nil || done {
		return err
	}
	project, ok, err := r.getProjectBaseForRefresh(ctx, contract)
	if err != nil || !ok || project == nil {
		return err
	}
	chainState, ok, err := r.getProjectChainStateForRefresh(ctx, contract)
	if err != nil || !ok || chainState == nil {
		return nil
	}
	project.Meta.ChainState = chainState.ChainState
	genesisWalletShares, err := r.fetchGenesisWallets(ctx, project, chainState.ChainState.Token.TotalSupply)
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
	if err := r.publishProjectGenesisWallets(ctx, project, chainState.ChainState.Token.TotalSupply, genesisWalletShares); err != nil {
		log.WithFields(log.Fields{
			"component": "project_state_reconciler",
			"contract":  project.Meta.Contract.Hex(),
			"error":     err.Error(),
		}).Warn("failed to persist project genesis wallets")
		return nil
	}
	genesisWallets := genesisWalletMetasFromShares(genesisWalletShares)
	fetchedAt := time.Now().UTC()
	if r.componentCache != nil {
		storeItems := projectGenesisWalletsToStore(project, chainState.ChainState.Token.TotalSupply, genesisWallets)
		if err := r.componentCache.SetGenesisWallets(ctx, contract, storeItems); err != nil {
			return err
		}
	} else if r.projectCache != nil {
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
	}
	return r.markComponentSuccess(ctx, contract, "genesis_wallet", fetchedAt)
}

func (r *projectStateReconcilerImpl) refreshProjectAveDetail(ctx context.Context, contract common.Address) error {
	if r.aveDetailFetcher == nil || strings.TrimSpace(r.aveChain) == "" {
		return nil
	}
	if r.componentCache != nil {
		if _, ok, err := r.componentCache.GetAveDetail(ctx, contract); err != nil || ok {
			return err
		}
	} else if r.projectCache != nil {
		project, ok, err := r.projectCache.GetProject(ctx, contract)
		if err != nil || !ok || project == nil || project.AveDetail != nil {
			return err
		}
	} else {
		return nil
	}
	tokenID := strings.ToLower(contract.Hex()) + "-" + r.aveChain
	response, err := r.aveDetailFetcher.FetchDetail(ctx, tokenID)
	if err != nil {
		return nil
	}
	detail := projectAveDetailFromAveResponse(response, time.Now().UTC())
	if detail == nil {
		return nil
	}
	if err := r.persistProjectAveDetail(ctx, contract, detail); err != nil {
		return nil
	}
	if r.componentCache != nil {
		return r.componentCache.SetAveDetail(ctx, contract, projectAveDetailToStore(detail))
	}
	if r.projectCache != nil {
		_, err = r.projectCache.UpdateProject(ctx, contract, func(current *Project, exists bool) (*Project, bool, error) {
			if !exists || current == nil || current.AveDetail != nil {
				return nil, false, nil
			}
			current.AveDetail = detail
			return current, true, nil
		})
		return err
	}
	return nil
}

func (r *projectStateReconcilerImpl) refreshProjectCreatorHistoricalProjects(ctx context.Context, contract common.Address) error {
	if r.componentCache == nil && r.projectCache != nil {
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
		return err
	}
	if done, err := r.componentSucceeded(ctx, contract, "creator_history"); err != nil || done {
		return err
	}
	project, ok, err := r.getProjectBaseForRefresh(ctx, contract)
	if err != nil || !ok || project == nil {
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
	if r.componentCache != nil {
		items := creatorHistoricalProjectsToStore(contract, historicalProjects)
		if err := r.componentCache.SetCreatorHistory(ctx, contract, items); err != nil {
			return err
		}
	} else if r.projectCache != nil {
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
	}
	return r.markComponentSuccess(ctx, contract, "creator_history", fetchedAt)
}

func (r *projectStateReconcilerImpl) refreshProjectSoliditySource(ctx context.Context, contract common.Address) error {
	if r.solidityClientSet == nil || r.chainID <= 0 {
		return nil
	}
	closer, client, err := r.solidityClientSet.NewSolidityServiceClient()
	if err != nil {
		return nil
	}
	defer utilio.Close(closer)

	info, err := client.GetContractSourceInfo(ctx, &solidityapiclient.GetContractSourceInfoRequest{
		ChainId:  r.chainID,
		Contract: contract.Hex(),
	})
	if err != nil || info == nil {
		return nil
	}
	fact := appstore.ProjectBytecodeFact{
		ProjectContract:       contract,
		IsBytecodeBlacklisted: info.IsBytecodeBlacklisted,
		FetchedAt:             time.Now().UTC(),
	}
	if strings.TrimSpace(info.CodeBinHash) != "" {
		fact.CodeHash = common.HexToHash(info.CodeBinHash)
	}
	if store, ok := r.projectStore.(appstore.ProjectBytecodeFactStore); ok && store != nil {
		if err := store.UpsertProjectBytecodeFact(ctx, fact); err != nil {
			return err
		}
	}
	if r.componentCache != nil {
		return r.componentCache.SetBytecodeFact(ctx, fact)
	}
	if r.projectCache != nil {
		_, err = r.projectCache.UpdateProject(ctx, contract, func(current *Project, exists bool) (*Project, bool, error) {
			if !exists || current == nil || current.Meta.IsBytecodeBlacklisted == info.IsBytecodeBlacklisted {
				return nil, false, nil
			}
			current.Meta.IsBytecodeBlacklisted = info.IsBytecodeBlacklisted
			return current, true, nil
		})
		return err
	}
	return nil
}

func (r *projectStateReconcilerImpl) persistProjectMeta(ctx context.Context, meta ProjectMeta) error {
	if r.persistencePublisher == nil {
		return nil
	}
	return r.persistencePublisher.PublishProjectMetaSave(ctx, projectMetaToStore(meta))
}

func (r *projectStateReconcilerImpl) persistProjectBase(ctx context.Context, meta ProjectMeta) error {
	if store, ok := r.projectStore.(appstore.ProjectBaseStore); ok && store != nil {
		return store.SaveProjectBase(ctx, projectBaseFromMeta(meta))
	}
	return r.persistProjectMeta(ctx, meta)
}

func (r *projectStateReconcilerImpl) persistProjectChainState(ctx context.Context, contract common.Address, snapshot athenacontract.AthenaProject, fetchedAt time.Time) error {
	if store, ok := r.projectStore.(appstore.ProjectChainStateStore); ok && store != nil {
		return store.UpsertProjectChainState(ctx, projectChainStateFromSnapshot(contract, snapshot, fetchedAt))
	}
	return nil
}

func (r *projectStateReconcilerImpl) persistProjectAveDetail(ctx context.Context, contract common.Address, detail *ProjectAveDetail) error {
	if store, ok := r.projectStore.(appstore.ProjectAveDetailStore); ok && store != nil {
		return store.UpsertProjectAveDetail(ctx, contract, projectAveDetailToStore(detail))
	}
	if r.persistencePublisher == nil {
		return nil
	}
	return r.persistencePublisher.PublishProjectAveDetailUpsert(ctx, contract, projectAveDetailToStore(detail))
}

func (r *projectStateReconcilerImpl) persistProjectCreatorResult(ctx context.Context, contract common.Address, result SimulateResult) error {
	if r.persistencePublisher == nil {
		return nil
	}
	return r.persistencePublisher.PublishProjectCreatorResultUpdate(ctx, contract, result)
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

func projectGenesisWalletsToStore(project *Project, totalSupply *big.Int, items []GenesisWalletMeta) []appstore.ProjectGenesisWallet {
	if project == nil {
		return nil
	}
	txHash := projectTxHash(project)
	result := make([]appstore.ProjectGenesisWallet, 0, len(items))
	for _, item := range items {
		if item.Wallet == (common.Address{}) || item.NetAmount == nil || item.NetAmount.Sign() <= 0 {
			continue
		}
		result = append(result, appstore.ProjectGenesisWallet{
			ProjectContract:   project.Meta.Contract,
			Wallet:            item.Wallet,
			NetAmount:         new(big.Int).Set(item.NetAmount),
			RatioBPS:          item.RatioBPS,
			RankIndex:         item.RankIndex,
			TotalSupply:       newBigIntOrZero(totalSupply),
			SourceTxHash:      txHash,
			SourceBlockNumber: project.Meta.BlockNumber,
		})
	}
	return result
}

func creatorHistoricalProjectsToStore(contract common.Address, items []common.Address) []appstore.ProjectCreatorHistoricalProject {
	result := make([]appstore.ProjectCreatorHistoricalProject, 0, len(items))
	for i, item := range items {
		if item == (common.Address{}) {
			continue
		}
		result = append(result, appstore.ProjectCreatorHistoricalProject{
			ProjectContract:           contract,
			HistoricalProjectContract: item,
			RankIndex:                 int32(i),
		})
	}
	return result
}

func newBigIntOrZero(value *big.Int) *big.Int {
	if value == nil {
		return new(big.Int)
	}
	return new(big.Int).Set(value)
}

func (r *projectStateReconcilerImpl) componentSucceeded(ctx context.Context, contract common.Address, component string) (bool, error) {
	if store, ok := r.projectStore.(appstore.ProjectComponentStateStore); ok && store != nil {
		state, err := store.GetProjectComponentState(ctx, contract, component)
		if err != nil || state == nil {
			return false, err
		}
		return state.Status == "success" && !state.LastSuccessAt.IsZero(), nil
	}
	return false, nil
}

func (r *projectStateReconcilerImpl) markComponentSuccess(ctx context.Context, contract common.Address, component string, now time.Time) error {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if store, ok := r.projectStore.(appstore.ProjectComponentStateStore); ok && store != nil {
		return store.UpsertProjectComponentState(ctx, appstore.ProjectComponentState{
			ProjectContract: contract,
			Component:       component,
			Status:          "success",
			LastAttemptAt:   now,
			LastSuccessAt:   now,
		})
	}
	return nil
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
	if project == nil {
		return errors.New("project is nil")
	}
	genesisWallets := genesisWalletMetasFromShares(shares)
	if store, ok := r.projectStore.(appstore.ProjectGenesisWalletStore); ok && store != nil {
		if err := store.ReplaceProjectGenesisWallets(ctx, project.Meta.Contract, projectGenesisWalletsToStore(project, totalSupply, genesisWallets)); err != nil {
			return err
		}
		return nil
	}
	if r.persistencePublisher == nil {
		return errors.New("persistence publisher is not configured")
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
	if project == nil {
		return errors.New("project is nil")
	}
	items := creatorHistoricalProjectsToStore(project.Meta.Contract, project.Meta.CreatorHistoricalProjects)
	if store, ok := r.projectStore.(appstore.ProjectCreatorHistoricalProjectStore); ok && store != nil {
		return store.ReplaceProjectCreatorHistoricalProjects(ctx, project.Meta.Contract, items)
	}
	if r.persistencePublisher == nil {
		return errors.New("persistence publisher is not configured")
	}
	return r.persistencePublisher.PublishProjectCreatorHistoricalProjectsReplace(ctx, project.Meta.Contract, items)
}
