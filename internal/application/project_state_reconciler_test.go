package application

import (
	"context"
	"errors"
	"math/big"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/redis/go-redis/v9"
	"github.com/useryege/athena/internal/application/redisport"
	appstore "github.com/useryege/athena/internal/application/store"
	athenacontract "github.com/useryege/athena/pkg/abi/ATHENA"
)

type sourceQualityAnalyzerFake struct {
	report string
	err    error
	calls  int
}

type projectPolicyEngineFake struct {
	calls     int
	contracts []common.Address
	report    ProjectReport
	err       error
}

func (f *projectPolicyEngineFake) EvaluateProject(_ context.Context, contract common.Address) (ProjectReport, error) {
	f.calls++
	f.contracts = append(f.contracts, contract)
	return f.report, f.err
}

type simulationFetcherFake struct {
	projects []athenacontract.AthenaProject
	states   []athenacontract.AthenaSimulationState
	err      error
}

func (f *simulationFetcherFake) FetchProject(context.Context, athenacontract.AthenaProjectQuery) (athenacontract.AthenaProject, error) {
	return athenacontract.AthenaProject{}, f.err
}

func (f *simulationFetcherFake) FetchProjects(context.Context, []athenacontract.AthenaProjectQuery) ([]athenacontract.AthenaProject, error) {
	return nil, f.err
}

func (f *simulationFetcherFake) FetchProjectsWithSimulationState(_ context.Context, queries []athenacontract.AthenaProjectQuery) ([]athenacontract.AthenaProjectWithSimulationState, error) {
	if f.err != nil {
		return nil, f.err
	}
	items := make([]athenacontract.AthenaProjectWithSimulationState, 0, len(queries))
	for i := range queries {
		item := athenacontract.AthenaProjectWithSimulationState{}
		if i < len(f.projects) {
			item.Project = f.projects[i]
		}
		if i < len(f.states) {
			item.SimulationState = f.states[i]
		}
		items = append(items, item)
	}
	return items, nil
}

func (f *simulationFetcherFake) FetchSimulationState(context.Context, athenacontract.AthenaProjectQuery) (athenacontract.AthenaSimulationState, error) {
	if len(f.states) == 0 {
		return athenacontract.AthenaSimulationState{}, f.err
	}
	return f.states[0], f.err
}

func (f *simulationFetcherFake) FetchSimulationStates(context.Context, []athenacontract.AthenaProjectQuery) ([]athenacontract.AthenaSimulationState, error) {
	return append([]athenacontract.AthenaSimulationState(nil), f.states...), f.err
}

type initProjectFetcherFake struct {
	snapshots []athenacontract.AthenaProject
	calls     int
	queryLen  int
	err       error
}

func (f *initProjectFetcherFake) FetchProject(context.Context, athenacontract.AthenaProjectQuery) (athenacontract.AthenaProject, error) {
	return athenacontract.AthenaProject{}, f.err
}

func (f *initProjectFetcherFake) FetchProjects(_ context.Context, queries []athenacontract.AthenaProjectQuery) ([]athenacontract.AthenaProject, error) {
	f.calls++
	f.queryLen = len(queries)
	if f.err != nil {
		return nil, f.err
	}
	if len(f.snapshots) > 0 {
		return f.snapshots, nil
	}
	snapshots := make([]athenacontract.AthenaProject, 0, len(queries))
	for _, query := range queries {
		snapshots = append(snapshots, athenacontract.AthenaProject{
			TokenContract: query.TokenContract,
			Token: athenacontract.AthenaToken{
				IsValidERC20: true,
			},
		})
	}
	return snapshots, nil
}

func (f *initProjectFetcherFake) FetchProjectsWithSimulationState(context.Context, []athenacontract.AthenaProjectQuery) ([]athenacontract.AthenaProjectWithSimulationState, error) {
	return nil, f.err
}

func (f *initProjectFetcherFake) FetchSimulationState(context.Context, athenacontract.AthenaProjectQuery) (athenacontract.AthenaSimulationState, error) {
	return athenacontract.AthenaSimulationState{}, f.err
}

func (f *initProjectFetcherFake) FetchSimulationStates(context.Context, []athenacontract.AthenaProjectQuery) ([]athenacontract.AthenaSimulationState, error) {
	return nil, f.err
}

type initProjectNoGetCacheFake struct {
	projects map[common.Address]*Project
}

func (c *initProjectNoGetCacheFake) ReplaceAll(context.Context, []*Project) error { return nil }
func (c *initProjectNoGetCacheFake) SetProject(_ context.Context, project *Project) error {
	if c.projects == nil {
		c.projects = map[common.Address]*Project{}
	}
	c.projects[project.Meta.Contract] = project
	return nil
}
func (c *initProjectNoGetCacheFake) UpdateProject(_ context.Context, contract common.Address, updater ProjectUpdater) (bool, error) {
	var current *Project
	if c.projects != nil {
		current = c.projects[contract]
	}
	next, changed, err := updater(current, current != nil)
	if err != nil || !changed {
		return false, err
	}
	if next != nil && next.Meta.Contract == (common.Address{}) {
		next.Meta.Contract = contract
	}
	if c.projects == nil {
		c.projects = map[common.Address]*Project{}
	}
	if next == nil {
		delete(c.projects, contract)
		return true, nil
	}
	c.projects[contract] = next
	return true, nil
}
func (c *initProjectNoGetCacheFake) DeleteProject(context.Context, common.Address) error {
	c.projects = nil
	return nil
}
func (c *initProjectNoGetCacheFake) GetProject(context.Context, common.Address) (*Project, bool, error) {
	return nil, false, errors.New("GetProject should not be called by InitProject")
}
func (c *initProjectNoGetCacheFake) ListProjects(context.Context) ([]*Project, error) {
	return nil, nil
}
func (c *initProjectNoGetCacheFake) ListProjectsPage(context.Context, int32, int32) ([]*Project, int64, int32, int32, error) {
	return nil, 0, 0, 0, nil
}

type reconcilerDiscoveryNodeClientFake struct {
	receipt      *types.Receipt
	receiptErr   error
	logs         []types.Log
	filterErr    error
	receiptCalls int
	filterCalls  int
}

func (f *reconcilerDiscoveryNodeClientFake) SubscribeNewHead(context.Context, chan<- *types.Header) (ethereum.Subscription, error) {
	return nil, nil
}

func (f *reconcilerDiscoveryNodeClientFake) BlockNumber(context.Context) (uint64, error) {
	return 0, nil
}

func (f *reconcilerDiscoveryNodeClientFake) BlockByNumber(context.Context, *big.Int) (*types.Block, error) {
	return nil, nil
}

func (f *reconcilerDiscoveryNodeClientFake) TransactionReceipt(context.Context, common.Hash) (*types.Receipt, error) {
	f.receiptCalls++
	if f.receiptErr != nil {
		return nil, f.receiptErr
	}
	return f.receipt, nil
}

func (f *reconcilerDiscoveryNodeClientFake) FilterLogs(context.Context, ethereum.FilterQuery) ([]types.Log, error) {
	f.filterCalls++
	if f.filterErr != nil {
		return nil, f.filterErr
	}
	return append([]types.Log(nil), f.logs...), nil
}

func (f *reconcilerDiscoveryNodeClientFake) ChainID(context.Context) (*big.Int, error) {
	return big.NewInt(1), nil
}

type projectSimulatorFake struct {
	result SimulateResult
	err    error
}

func (s *projectSimulatorFake) SimulatePrimary(context.Context, common.Address, common.Address, common.Address, common.Address, athenacontract.AthenaSimulationState) (SimulateResult, error) {
	return s.result, s.err
}

func (f *sourceQualityAnalyzerFake) AnalyzeContractSource(context.Context, string) (string, error) {
	f.calls++
	if f.err != nil {
		return "", f.err
	}
	return f.report, nil
}

type persistencePublisherFake struct {
	metas                []appstore.ProjectMeta
	sourceQualityReports map[common.Address]string
	codeBinHashes        map[common.Address]common.Hash
	creatorResults       map[common.Address]SimulateResult
	creatorHistorical    map[common.Address][]appstore.ProjectCreatorHistoricalProject
	events               []PersistenceEvent
	err                  error
}

func (p *persistencePublisherFake) Publish(_ context.Context, event PersistenceEvent) error {
	if p.err != nil {
		return p.err
	}
	p.events = append(p.events, event)
	return nil
}
func (p *persistencePublisherFake) PublishProjectMetaSave(_ context.Context, meta appstore.ProjectMeta) error {
	if p.err != nil {
		return p.err
	}
	p.metas = append(p.metas, meta)
	return nil
}
func (p *persistencePublisherFake) PublishProjectEventLog(context.Context, appstore.ProjectEventLog) error {
	return nil
}
func (p *persistencePublisherFake) PublishProjectSourceCodeUpdate(context.Context, common.Address, string) error {
	return nil
}
func (p *persistencePublisherFake) PublishProjectCodeBinHashUpdate(_ context.Context, contract common.Address, codeBinHash common.Hash) error {
	if p.err != nil {
		return p.err
	}
	if p.codeBinHashes == nil {
		p.codeBinHashes = map[common.Address]common.Hash{}
	}
	p.codeBinHashes[contract] = codeBinHash
	return nil
}
func (p *persistencePublisherFake) PublishProjectSourceQualityReportUpdate(_ context.Context, contract common.Address, report string) error {
	if p.err != nil {
		return p.err
	}
	if p.sourceQualityReports == nil {
		p.sourceQualityReports = map[common.Address]string{}
	}
	p.sourceQualityReports[contract] = report
	return nil
}
func (p *persistencePublisherFake) PublishProjectCreatorResultUpdate(_ context.Context, contract common.Address, result SimulateResult) error {
	if p.err != nil {
		return p.err
	}
	if p.creatorResults == nil {
		p.creatorResults = map[common.Address]SimulateResult{}
	}
	p.creatorResults[contract] = result
	return nil
}
func (p *persistencePublisherFake) PublishProjectCreatorHistoricalProjectsReplace(_ context.Context, contract common.Address, items []appstore.ProjectCreatorHistoricalProject) error {
	if p.err != nil {
		return p.err
	}
	if p.creatorHistorical == nil {
		p.creatorHistorical = map[common.Address][]appstore.ProjectCreatorHistoricalProject{}
	}
	p.creatorHistorical[contract] = append([]appstore.ProjectCreatorHistoricalProject(nil), items...)
	return nil
}
func (p *persistencePublisherFake) PublishBytecodeBlacklistAdd(context.Context, appstore.BytecodeBlacklistContract) error {
	return nil
}
func (p *persistencePublisherFake) PublishBytecodeBlacklistUpdateNote(context.Context, common.Address, string) error {
	return nil
}
func (p *persistencePublisherFake) PublishBytecodeBlacklistDelete(context.Context, common.Address) error {
	return nil
}
func (p *persistencePublisherFake) PublishSourcecodeBlacklistContractAdd(context.Context, appstore.SourcecodeBlacklistContract) error {
	return nil
}
func (p *persistencePublisherFake) PublishSourcecodeBlacklistContractUpdateNote(context.Context, common.Address, string) error {
	return nil
}
func (p *persistencePublisherFake) PublishSourcecodeBlacklistContractDelete(context.Context, common.Address) error {
	return nil
}
func (p *persistencePublisherFake) PublishWalletBlacklistAdd(context.Context, appstore.WalletBlacklistEntry) error {
	return nil
}
func (p *persistencePublisherFake) PublishWalletBlacklistUpdateNote(context.Context, common.Address, string) error {
	return nil
}
func (p *persistencePublisherFake) PublishWalletBlacklistDelete(context.Context, common.Address) error {
	return nil
}

func TestProjectStateReconcilerInitProjectCachesValidERC20WithoutGetProject(t *testing.T) {
	ctx := context.Background()
	contract := common.HexToAddress("0x00000000000000000000000000000000000000a1")
	cache := &initProjectNoGetCacheFake{}
	fetcher := &initProjectFetcherFake{}
	publisher := &persistencePublisherFake{}
	reconciler := &projectStateReconcilerImpl{
		projectCache:         cache,
		fetcher:              fetcher,
		persistencePublisher: publisher,
		scheduled:            map[common.Address]*scheduledProject{},
		jobSem:               make(chan struct{}, 1),
	}

	err := reconciler.InitProject(ctx, []DiscoveredProjectCandidate{{
		Contract:    contract,
		BlockNumber: 103,
		TxIndex:     7,
	}})
	if err != nil {
		t.Fatalf("init project: %v", err)
	}
	if fetcher.calls != 1 {
		t.Fatalf("fetch projects calls = %d, want 1", fetcher.calls)
	}
	project := cache.projects[contract]
	if project == nil {
		t.Fatal("project missing after init")
	}
	if project.Meta.Contract != contract {
		t.Fatalf("cached contract = %s, want %s", project.Meta.Contract.Hex(), contract.Hex())
	}
	if project.Meta.ChainState.TokenContract != contract {
		t.Fatalf("chain state token contract = %s, want %s", project.Meta.ChainState.TokenContract.Hex(), contract.Hex())
	}
	if len(publisher.metas) != 1 {
		t.Fatalf("persisted project metas = %d, want 1", len(publisher.metas))
	}
	if publisher.metas[0].Contract != contract {
		t.Fatalf("persisted contract = %s, want %s", publisher.metas[0].Contract.Hex(), contract.Hex())
	}
	if publisher.metas[0].BlockNumber != 103 || publisher.metas[0].TxIndex != 7 {
		t.Fatalf("persisted block/tx index = %d/%d, want 103/7", publisher.metas[0].BlockNumber, publisher.metas[0].TxIndex)
	}
	if reconciler.scheduled[contract] == nil {
		t.Fatal("project was not scheduled")
	}
}

func TestProjectStateReconcilerInitProjectCachesBatchBySnapshotIndex(t *testing.T) {
	ctx := context.Background()
	validA := common.HexToAddress("0x00000000000000000000000000000000000000b1")
	invalid := common.HexToAddress("0x00000000000000000000000000000000000000b2")
	validB := common.HexToAddress("0x00000000000000000000000000000000000000b3")
	cache := &initProjectNoGetCacheFake{}
	fetcher := &initProjectFetcherFake{snapshots: []athenacontract.AthenaProject{
		{
			TokenContract: validA,
			Token:         athenacontract.AthenaToken{IsValidERC20: true},
		},
		{
			TokenContract: invalid,
			Token:         athenacontract.AthenaToken{IsValidERC20: false},
		},
		{
			TokenContract: validB,
			Token:         athenacontract.AthenaToken{IsValidERC20: true},
		},
	}}
	publisher := &persistencePublisherFake{}
	reconciler := &projectStateReconcilerImpl{
		projectCache:         cache,
		fetcher:              fetcher,
		persistencePublisher: publisher,
		scheduled:            map[common.Address]*scheduledProject{},
		jobSem:               make(chan struct{}, 1),
	}

	err := reconciler.InitProject(ctx, []DiscoveredProjectCandidate{
		{Contract: validA, BlockNumber: 101},
		{Contract: invalid, BlockNumber: 102},
		{Contract: validB, BlockNumber: 103, Source: ProjectDiscoverySourceFollowHeads},
	})
	if err != nil {
		t.Fatalf("init project: %v", err)
	}
	if fetcher.calls != 1 || fetcher.queryLen != 3 {
		t.Fatalf("fetch projects calls/query len = %d/%d, want 1/3", fetcher.calls, fetcher.queryLen)
	}
	if cache.projects[validA] == nil || cache.projects[validA].Meta.ChainState.TokenContract != validA {
		t.Fatalf("valid A was not cached with matching snapshot")
	}
	if cache.projects[invalid] != nil {
		t.Fatalf("invalid ERC20 project was cached")
	}
	if cache.projects[validB] == nil || cache.projects[validB].Meta.ChainState.TokenContract != validB {
		t.Fatalf("valid B was not cached with matching snapshot")
	}
	if len(publisher.metas) != 2 {
		t.Fatalf("persisted project metas = %d, want 2", len(publisher.metas))
	}
	if publisher.metas[0].Contract != validA || publisher.metas[1].Contract != validB {
		t.Fatalf("persisted contracts = %s/%s, want %s/%s", publisher.metas[0].Contract.Hex(), publisher.metas[1].Contract.Hex(), validA.Hex(), validB.Hex())
	}
	if reconciler.scheduled[validA] == nil || reconciler.scheduled[validB] == nil {
		t.Fatalf("valid projects were not scheduled")
	}
	if reconciler.scheduled[invalid] != nil {
		t.Fatalf("invalid ERC20 project was scheduled")
	}
}

func TestProjectStateReconcilerInitProjectReturnsErrorOnIndexedSnapshotMismatch(t *testing.T) {
	ctx := context.Background()
	contractA := common.HexToAddress("0x00000000000000000000000000000000000000c1")
	contractB := common.HexToAddress("0x00000000000000000000000000000000000000c2")
	reconciler := &projectStateReconcilerImpl{
		projectCache: &initProjectNoGetCacheFake{},
		fetcher: &initProjectFetcherFake{snapshots: []athenacontract.AthenaProject{
			{
				TokenContract: contractA,
				Token:         athenacontract.AthenaToken{IsValidERC20: true},
			},
			{
				TokenContract: contractA,
				Token:         athenacontract.AthenaToken{IsValidERC20: true},
			},
		}},
		scheduled: map[common.Address]*scheduledProject{},
	}

	err := reconciler.InitProject(ctx, []DiscoveredProjectCandidate{
		{Contract: contractA},
		{Contract: contractB},
	})
	if err == nil {
		t.Fatal("init project succeeded, want mismatch error")
	}
}

func TestProjectStateReconcilerInitProjectDoesNotOverwriteExistingCache(t *testing.T) {
	ctx := context.Background()
	contract := common.HexToAddress("0x00000000000000000000000000000000000000a2")
	cache := &initProjectNoGetCacheFake{projects: map[common.Address]*Project{
		contract: {Meta: ProjectMeta{
			Contract:    contract,
			BlockNumber: 11,
		}},
	}}
	publisher := &persistencePublisherFake{}
	reconciler := &projectStateReconcilerImpl{
		projectCache:         cache,
		fetcher:              &initProjectFetcherFake{},
		persistencePublisher: publisher,
		scheduled:            map[common.Address]*scheduledProject{},
		jobSem:               make(chan struct{}, 1),
	}

	err := reconciler.InitProject(ctx, []DiscoveredProjectCandidate{{
		Contract:    contract,
		BlockNumber: 103,
		Source:      ProjectDiscoverySourceFollowHeads,
	}})
	if err != nil {
		t.Fatalf("init project: %v", err)
	}
	project := cache.projects[contract]
	if project == nil {
		t.Fatal("existing project missing")
	}
	if project.Meta.BlockNumber != 11 {
		t.Fatalf("cached block number = %d, want existing 11", project.Meta.BlockNumber)
	}
	if len(publisher.metas) != 1 {
		t.Fatalf("persisted project metas = %d, want 1", len(publisher.metas))
	}
	if publisher.metas[0].Contract != contract || publisher.metas[0].BlockNumber != 103 {
		t.Fatalf("persisted meta = %s/%d, want %s/103", publisher.metas[0].Contract.Hex(), publisher.metas[0].BlockNumber, contract.Hex())
	}
	if reconciler.scheduled[contract] == nil {
		t.Fatal("project was not scheduled")
	}
}

func TestProjectStateReconcilerSchedulePolicies(t *testing.T) {
	reconciler := &projectStateReconcilerImpl{}
	catchUpContract := common.HexToAddress("0x00000000000000000000000000000000000000a1")
	followContract := common.HexToAddress("0x00000000000000000000000000000000000000a2")

	reconciler.scheduleProject(DiscoveredProjectCandidate{Contract: catchUpContract, Source: ProjectDiscoverySourceCatchUp})
	reconciler.scheduleProject(DiscoveredProjectCandidate{Contract: followContract, Source: ProjectDiscoverySourceFollowHeads})

	catchUp := reconciler.scheduled[catchUpContract]
	follow := reconciler.scheduled[followContract]
	if catchUp == nil || catchUp.interval != time.Minute {
		t.Fatalf("catch-up schedule = %+v, want 1m interval", catchUp)
	}
	if ttl := catchUp.expiresAt.Sub(catchUp.nextRunAt); ttl < 2*time.Minute || ttl > 3*time.Minute {
		t.Fatalf("catch-up ttl after first run = %s, want about 2m", ttl)
	}
	if follow == nil || follow.interval != time.Minute {
		t.Fatalf("follow-heads schedule = %+v, want 1m interval", follow)
	}
	if ttl := follow.expiresAt.Sub(follow.nextRunAt); ttl < 359*time.Minute || ttl > 360*time.Minute {
		t.Fatalf("follow-heads ttl after first run = %s, want about 359m", ttl)
	}
}

func TestProjectStateReconcilerScheduleDeduplicatesAndExpires(t *testing.T) {
	contract := common.HexToAddress("0x00000000000000000000000000000000000000a1")
	reconciler := &projectStateReconcilerImpl{}

	reconciler.scheduleProject(DiscoveredProjectCandidate{Contract: contract, Source: ProjectDiscoverySourceCatchUp})
	reconciler.scheduleProject(DiscoveredProjectCandidate{Contract: contract, Source: ProjectDiscoverySourceFollowHeads})
	if len(reconciler.scheduled) != 1 {
		t.Fatalf("scheduled count = %d, want 1", len(reconciler.scheduled))
	}

	item := reconciler.scheduled[contract]
	item.nextRunAt = time.Now().Add(-time.Minute)
	item.expiresAt = time.Now().Add(-time.Second)
	due := reconciler.dueProjectContracts(time.Now())
	if len(due) != 0 {
		t.Fatalf("due contracts = %v, want empty", due)
	}
	if len(reconciler.scheduled) != 0 {
		t.Fatalf("scheduled count after expiry = %d, want 0", len(reconciler.scheduled))
	}
}

func TestProjectStateReconcilerRefreshProjectGenesisWalletsPersistsAndCaches(t *testing.T) {
	cache := newProjectSnapshotCacheTest(t)
	ctx := context.Background()
	contract := common.HexToAddress("0x00000000000000000000000000000000000000a6")
	creator := common.HexToAddress("0x00000000000000000000000000000000000000b6")
	wallet := common.HexToAddress("0x00000000000000000000000000000000000000c6")
	txHash := common.HexToHash("0x0606060606060606060606060606060606060606060606060606060606060606")
	if err := cache.SetProject(ctx, &Project{Meta: ProjectMeta{
		Contract:    contract,
		Creator:     creator,
		TxHash:      txHash,
		BlockNumber: 106,
		ChainState: athenacontract.AthenaProject{
			Token: athenacontract.AthenaToken{TotalSupply: big.NewInt(1000)},
		},
	}}); err != nil {
		t.Fatalf("set project: %v", err)
	}

	publisher := &persistencePublisherFake{}
	policyEngine := &projectPolicyEngineFake{}
	reconciler := &projectStateReconcilerImpl{
		projectCache:          cache,
		discoveryNodeClient:   &reconcilerDiscoveryNodeClientFake{receipt: &types.Receipt{Logs: []*types.Log{erc20TransferLog(contract, common.Address{}, wallet, big.NewInt(250), txHash)}}},
		persistencePublisher:  publisher,
		policyEngine:          policyEngine,
		sourceQualityAnalyzer: nil,
	}

	if err := reconciler.refreshProject(ctx, contract); err != nil {
		t.Fatalf("refresh project: %v", err)
	}
	project, ok, err := cache.GetProject(ctx, contract)
	if err != nil {
		t.Fatalf("get project: %v", err)
	}
	if !ok || len(project.Meta.GenesisWallets) != 1 {
		t.Fatalf("genesis wallets = %+v, want one", project.Meta.GenesisWallets)
	}
	if project.Meta.GenesisWallets[0].Wallet != wallet || project.Meta.GenesisWallets[0].RatioBPS != 2500 {
		t.Fatalf("genesis wallet = %+v, want wallet %s 2500bps", project.Meta.GenesisWallets[0], wallet.Hex())
	}
	if project.Meta.GenesisWalletsFetchedAt.IsZero() {
		t.Fatal("genesis wallets fetched at is zero")
	}
	if len(publisher.events) != 1 || publisher.events[0].Op != PersistenceOpProjectGenesisReplace {
		t.Fatalf("published events = %+v, want genesis replace", publisher.events)
	}
	if policyEngine.calls != 1 || len(policyEngine.contracts) != 1 || policyEngine.contracts[0] != contract {
		t.Fatalf("policy evaluations = %d/%v, want one for %s", policyEngine.calls, policyEngine.contracts, contract.Hex())
	}
}

func TestProjectStateReconcilerRefreshProjectCreatorHistoricalProjectsPersistsAndCaches(t *testing.T) {
	cache := newProjectSnapshotCacheTest(t)
	ctx := context.Background()
	creator := common.HexToAddress("0x00000000000000000000000000000000000000b7")
	contract := common.HexToAddress("0x00000000000000000000000000000000000000a7")
	previousA := common.HexToAddress("0x00000000000000000000000000000000000000c7")
	previousB := common.HexToAddress("0x00000000000000000000000000000000000000d7")
	if err := cache.SetProject(ctx, &Project{Meta: ProjectMeta{
		Contract:    contract,
		Creator:     creator,
		BlockNumber: 107,
		TxIndex:     2,
	}}); err != nil {
		t.Fatalf("set project: %v", err)
	}

	publisher := &persistencePublisherFake{}
	policyEngine := &projectPolicyEngineFake{}
	reconciler := &projectStateReconcilerImpl{
		projectCache: cache,
		projectStore: &discoveryProjectStoreFake{metas: []appstore.ProjectMeta{
			{Contract: previousA, Creator: creator},
			{Contract: previousB, Creator: creator},
			{Contract: previousA, Creator: creator},
			{Contract: contract, Creator: creator},
		}},
		persistencePublisher: publisher,
		policyEngine:         policyEngine,
	}

	if err := reconciler.refreshProject(ctx, contract); err != nil {
		t.Fatalf("refresh project: %v", err)
	}
	project, ok, err := cache.GetProject(ctx, contract)
	if err != nil {
		t.Fatalf("get project: %v", err)
	}
	if !ok || len(project.Meta.CreatorHistoricalProjects) != 2 || project.Meta.CreatorHistoricalProjects[0] != previousA || project.Meta.CreatorHistoricalProjects[1] != previousB {
		t.Fatalf("creator historical projects = %v, want [%s %s]", addressHexes(project.Meta.CreatorHistoricalProjects), previousA.Hex(), previousB.Hex())
	}
	if project.Meta.CreatorHistoricalProjectsFetchedAt.IsZero() {
		t.Fatal("creator historical projects fetched at is zero")
	}
	if got := publisher.creatorHistorical[contract]; len(got) != 2 || got[0].HistoricalProjectContract != previousA || got[1].HistoricalProjectContract != previousB {
		t.Fatalf("persisted creator historical projects = %+v", got)
	}
	if policyEngine.calls != 1 || len(policyEngine.contracts) != 1 || policyEngine.contracts[0] != contract {
		t.Fatalf("policy evaluations = %d/%v, want one for %s", policyEngine.calls, policyEngine.contracts, contract.Hex())
	}
}

func TestProjectStateReconcilerRefreshProjectSkipsFetchedGenesisAndHistory(t *testing.T) {
	cache := newProjectSnapshotCacheTest(t)
	ctx := context.Background()
	contract := common.HexToAddress("0x00000000000000000000000000000000000000a8")
	fetchedAt := time.Now().UTC()
	if err := cache.SetProject(ctx, &Project{Meta: ProjectMeta{
		Contract:                           contract,
		GenesisWalletsFetchedAt:            fetchedAt,
		CreatorHistoricalProjectsFetchedAt: fetchedAt,
	}}); err != nil {
		t.Fatalf("set project: %v", err)
	}
	nodeClient := &reconcilerDiscoveryNodeClientFake{}
	store := &discoveryProjectStoreFake{}
	reconciler := &projectStateReconcilerImpl{
		projectCache:        cache,
		discoveryNodeClient: nodeClient,
		projectStore:        store,
	}

	if err := reconciler.refreshProject(ctx, contract); err != nil {
		t.Fatalf("refresh project: %v", err)
	}
	if nodeClient.receiptCalls != 0 || nodeClient.filterCalls != 0 {
		t.Fatalf("genesis fetch calls = receipt %d filter %d, want zero", nodeClient.receiptCalls, nodeClient.filterCalls)
	}
	if store.calls != 0 {
		t.Fatalf("creator historical store calls = %d, want zero", store.calls)
	}
}

func TestProjectStateReconcilerRefreshProjectIgnoresGenesisAndHistoryFailures(t *testing.T) {
	restore := setCreatorHistoricalProjectRetryDelaysForTest(t)
	defer restore()

	cache := newProjectSnapshotCacheTest(t)
	ctx := context.Background()
	creator := common.HexToAddress("0x00000000000000000000000000000000000000b9")
	contract := common.HexToAddress("0x00000000000000000000000000000000000000a9")
	txHash := common.HexToHash("0x0909090909090909090909090909090909090909090909090909090909090909")
	if err := cache.SetProject(ctx, &Project{Meta: ProjectMeta{
		Contract:    contract,
		Creator:     creator,
		TxHash:      txHash,
		BlockNumber: 109,
		TxIndex:     1,
	}}); err != nil {
		t.Fatalf("set project: %v", err)
	}
	reconciler := &projectStateReconcilerImpl{
		projectCache:        cache,
		discoveryNodeClient: &reconcilerDiscoveryNodeClientFake{receiptErr: errors.New("receipt failed")},
		projectStore:        &discoveryProjectStoreFake{errs: []error{errors.New("one"), errors.New("two"), errors.New("three"), errors.New("four")}},
	}

	if err := reconciler.refreshProject(ctx, contract); err != nil {
		t.Fatalf("refresh project: %v", err)
	}
	project, ok, err := cache.GetProject(ctx, contract)
	if err != nil {
		t.Fatalf("get project: %v", err)
	}
	if !ok {
		t.Fatal("project missing")
	}
	if !project.Meta.GenesisWalletsFetchedAt.IsZero() {
		t.Fatalf("genesis wallets fetched at = %s, want zero", project.Meta.GenesisWalletsFetchedAt)
	}
	if !project.Meta.CreatorHistoricalProjectsFetchedAt.IsZero() {
		t.Fatalf("creator historical projects fetched at = %s, want zero", project.Meta.CreatorHistoricalProjectsFetchedAt)
	}
}

func erc20TransferLog(contract common.Address, from common.Address, to common.Address, amount *big.Int, txHash common.Hash) *types.Log {
	return &types.Log{
		Address: contract,
		Topics: []common.Hash{
			erc20TransferTopicHash,
			common.BytesToHash(from.Bytes()),
			common.BytesToHash(to.Bytes()),
		},
		Data:   common.LeftPadBytes(amount.Bytes(), 32),
		TxHash: txHash,
	}
}

func setCreatorHistoricalProjectRetryDelaysForTest(t *testing.T) func() {
	t.Helper()
	original := creatorHistoricalProjectRetryDelays
	creatorHistoricalProjectRetryDelays = []time.Duration{0, 0, 0}
	return func() {
		creatorHistoricalProjectRetryDelays = original
	}
}

func TestProjectStateReconcilerRefreshProjectSourceQualityReports(t *testing.T) {
	cache := newProjectSnapshotCacheTest(t)
	ctx := context.Background()
	contract := common.HexToAddress("0x00000000000000000000000000000000000000a1")
	if err := cache.SetProject(ctx, &Project{Meta: ProjectMeta{Contract: contract, SourceCode: "contract A {}"}}); err != nil {
		t.Fatalf("set project: %v", err)
	}

	analyzer := &sourceQualityAnalyzerFake{report: "## Report"}
	publisher := &persistencePublisherFake{}
	reconciler := &projectStateReconcilerImpl{
		projectCache:          cache,
		sourceQualityAnalyzer: analyzer,
		persistencePublisher:  publisher,
	}

	if err := reconciler.refreshProjectSourceQualityReport(ctx, contract); err != nil {
		t.Fatalf("refresh source quality reports: %v", err)
	}
	if analyzer.calls != 1 {
		t.Fatalf("analyzer calls = %d, want 1", analyzer.calls)
	}
	project, ok, err := cache.GetProject(ctx, contract)
	if err != nil {
		t.Fatalf("get project: %v", err)
	}
	if !ok || project.Meta.SourceQualityReport != "## Report" {
		t.Fatalf("source quality report = %q, want report", project.Meta.SourceQualityReport)
	}
	if project.Meta.SourceQualityReportFetchedAt.IsZero() {
		t.Fatal("source quality report fetched at is zero")
	}
	if publisher.sourceQualityReports[contract] != "## Report" {
		t.Fatalf("persisted report = %q, want report", publisher.sourceQualityReports[contract])
	}
}

func TestProjectStateReconcilerRefreshProjectCodeBinHashesPersistsMetaHash(t *testing.T) {
	cache := newProjectSnapshotCacheTest(t)
	ctx := context.Background()
	contract := common.HexToAddress("0x00000000000000000000000000000000000000a3")
	if err := cache.SetProject(ctx, &Project{Meta: ProjectMeta{Contract: contract}}); err != nil {
		t.Fatalf("set project: %v", err)
	}

	code := []byte{0x60, 0x60, 0x60, 0x40}
	wantHash := crypto.Keccak256Hash(code)
	publisher := &persistencePublisherFake{}
	reconciler := &projectStateReconcilerImpl{
		projectCache:         cache,
		persistencePublisher: publisher,
		codeAtFunc: func(context.Context, common.Address) ([]byte, error) {
			return code, nil
		},
	}

	if err := reconciler.refreshProjectCodeBinHash(ctx, contract); err != nil {
		t.Fatalf("refresh code bin hashes: %v", err)
	}
	project, ok, err := cache.GetProject(ctx, contract)
	if err != nil {
		t.Fatalf("get project: %v", err)
	}
	if !ok || project.Meta.CodeBinHash != wantHash {
		t.Fatalf("code bin hash = %s, want %s", project.Meta.CodeBinHash.Hex(), wantHash.Hex())
	}
	if publisher.codeBinHashes[contract] != wantHash {
		t.Fatalf("persisted code bin hash = %s, want %s", publisher.codeBinHashes[contract].Hex(), wantHash.Hex())
	}
}

func TestProjectStateReconcilerRefreshProjectSimulationsPersistsCreatorResult(t *testing.T) {
	cache := newProjectSnapshotCacheTest(t)
	ctx := context.Background()
	contract := common.HexToAddress("0x00000000000000000000000000000000000000a4")
	creator := common.HexToAddress("0x00000000000000000000000000000000000000b4")
	wethPair := common.HexToAddress("0x00000000000000000000000000000000000000c4")
	usdtPair := common.HexToAddress("0x00000000000000000000000000000000000000d4")
	if err := cache.SetProject(ctx, &Project{
		Meta: ProjectMeta{Contract: contract, Creator: creator, ChainState: athenacontract.AthenaProject{
			WethPair: athenacontract.AthenaPair{ContractAddress: wethPair},
			UsdtPair: athenacontract.AthenaPair{ContractAddress: usdtPair},
		}},
	}); err != nil {
		t.Fatalf("set project: %v", err)
	}

	want := SimulateResult{
		CanMintFromDeadViaTransferFrom: true,
		CanMintViaTransferToWethPair:   true,
	}
	publisher := &persistencePublisherFake{}
	chainState := athenacontract.AthenaProject{
		WethPair: athenacontract.AthenaPair{ContractAddress: wethPair},
		UsdtPair: athenacontract.AthenaPair{ContractAddress: usdtPair},
	}
	reconciler := &projectStateReconcilerImpl{
		projectCache:         cache,
		fetcher:              &simulationFetcherFake{projects: []athenacontract.AthenaProject{chainState}, states: []athenacontract.AthenaSimulationState{{}}},
		simulator:            &projectSimulatorFake{result: want},
		persistencePublisher: publisher,
	}

	if err := reconciler.refreshProject(ctx, contract); err != nil {
		t.Fatalf("refresh project simulations: %v", err)
	}
	project, ok, err := cache.GetProject(ctx, contract)
	if err != nil {
		t.Fatalf("get project: %v", err)
	}
	if !ok || project.Meta.CreatorResult != want {
		t.Fatalf("creator result = %+v, want %+v", project.Meta.CreatorResult, want)
	}
	if project.Meta.CreatorResultFetchedAt.IsZero() {
		t.Fatal("creator result fetched at is zero")
	}
	if publisher.creatorResults[contract] != want {
		t.Fatalf("persisted creator result = %+v, want %+v", publisher.creatorResults[contract], want)
	}
}

func TestProjectStateReconcilerRefreshProjectSimulationsSkipsCacheWhenPersistFails(t *testing.T) {
	cache := newProjectSnapshotCacheTest(t)
	ctx := context.Background()
	contract := common.HexToAddress("0x00000000000000000000000000000000000000a5")
	wethPair := common.HexToAddress("0x00000000000000000000000000000000000000c5")
	usdtPair := common.HexToAddress("0x00000000000000000000000000000000000000d5")
	if err := cache.SetProject(ctx, &Project{
		Meta: ProjectMeta{Contract: contract, ChainState: athenacontract.AthenaProject{
			WethPair: athenacontract.AthenaPair{ContractAddress: wethPair},
			UsdtPair: athenacontract.AthenaPair{ContractAddress: usdtPair},
		}},
	}); err != nil {
		t.Fatalf("set project: %v", err)
	}

	reconciler := &projectStateReconcilerImpl{
		projectCache: cache,
		fetcher: &simulationFetcherFake{projects: []athenacontract.AthenaProject{{
			WethPair: athenacontract.AthenaPair{ContractAddress: wethPair},
			UsdtPair: athenacontract.AthenaPair{ContractAddress: usdtPair},
		}}, states: []athenacontract.AthenaSimulationState{{}}},
		simulator: &projectSimulatorFake{result: SimulateResult{
			CanMintFromDeadViaTransferFrom: true,
		}},
		persistencePublisher: &persistencePublisherFake{err: errors.New("persist failed")},
	}

	if err := reconciler.refreshProject(ctx, contract); err != nil {
		t.Fatalf("refresh project simulations: %v", err)
	}
	project, ok, err := cache.GetProject(ctx, contract)
	if err != nil {
		t.Fatalf("get project: %v", err)
	}
	if !ok {
		t.Fatal("project missing")
	}
	if project.Meta.CreatorResult.CanMintFromDeadViaTransferFrom {
		t.Fatalf("creator result = %+v, want unchanged zero value", project.Meta.CreatorResult)
	}
	if !project.Meta.CreatorResultFetchedAt.IsZero() {
		t.Fatalf("creator result fetched at = %s, want zero", project.Meta.CreatorResultFetchedAt)
	}
}

func TestProjectStateReconcilerRefreshProjectSourceQualityReportsSkipsCompletedAndClosedSource(t *testing.T) {
	cache := newProjectSnapshotCacheTest(t)
	ctx := context.Background()
	closedSource := common.HexToAddress("0x00000000000000000000000000000000000000a1")
	completed := common.HexToAddress("0x00000000000000000000000000000000000000a2")
	if err := cache.SetProject(ctx, &Project{Meta: ProjectMeta{Contract: closedSource}}); err != nil {
		t.Fatalf("set closed source project: %v", err)
	}
	if err := cache.SetProject(ctx, &Project{Meta: ProjectMeta{Contract: completed, SourceCode: "contract A {}", SourceQualityReport: "existing"}}); err != nil {
		t.Fatalf("set completed project: %v", err)
	}

	analyzer := &sourceQualityAnalyzerFake{report: "## Report"}
	reconciler := &projectStateReconcilerImpl{
		projectCache:          cache,
		sourceQualityAnalyzer: analyzer,
		persistencePublisher:  &persistencePublisherFake{},
	}

	if err := reconciler.refreshProjectSourceQualityReport(ctx, closedSource); err != nil {
		t.Fatalf("refresh source quality reports: %v", err)
	}
	if err := reconciler.refreshProjectSourceQualityReport(ctx, completed); err != nil {
		t.Fatalf("refresh source quality reports: %v", err)
	}
	if analyzer.calls != 0 {
		t.Fatalf("analyzer calls = %d, want 0", analyzer.calls)
	}
}

func addressHexes(addresses []common.Address) []string {
	hexes := make([]string, 0, len(addresses))
	for _, address := range addresses {
		hexes = append(hexes, address.Hex())
	}
	return hexes
}

func newProjectSnapshotCacheTest(t *testing.T) ProjectSnapshotCache {
	t.Helper()
	mini := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	return NewProjectSnapshotCache(redisport.NewGoRedisAdapter(client))
}
