package application

import (
	"context"
	"errors"
	"math/big"
	"testing"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
	appstore "github.com/useryege/athena/internal/application/store"
	athenacontract "github.com/useryege/athena/pkg/abi/ATHENA"
)

type discoveryFetcherFake struct {
	snapshots []athenacontract.AthenaProject
}

func (f *discoveryFetcherFake) FetchProject(context.Context, athenacontract.AthenaProjectQuery) (athenacontract.AthenaProject, error) {
	if len(f.snapshots) == 0 {
		return athenacontract.AthenaProject{}, nil
	}
	return f.snapshots[0], nil
}

func (f *discoveryFetcherFake) FetchProjects(_ context.Context, queries []athenacontract.AthenaProjectQuery) ([]athenacontract.AthenaProject, error) {
	if len(f.snapshots) > 0 {
		return f.snapshots, nil
	}
	snapshots := make([]athenacontract.AthenaProject, 0, len(queries))
	for _, query := range queries {
		snapshots = append(snapshots, athenacontract.AthenaProject{
			TokenContract: query.TokenContract,
			Token: athenacontract.AthenaToken{
				TotalSupply:  big.NewInt(1000),
				IsValidERC20: true,
			},
		})
	}
	return snapshots, nil
}

func (f *discoveryFetcherFake) FetchProjectsWithSimulationState(context.Context, []athenacontract.AthenaProjectQuery) ([]athenacontract.AthenaProjectWithSimulationState, error) {
	return nil, nil
}

func (f *discoveryFetcherFake) FetchSimulationState(context.Context, athenacontract.AthenaProjectQuery) (athenacontract.AthenaSimulationState, error) {
	return athenacontract.AthenaSimulationState{}, nil
}

func (f *discoveryFetcherFake) FetchSimulationStates(context.Context, []athenacontract.AthenaProjectQuery) ([]athenacontract.AthenaSimulationState, error) {
	return nil, nil
}

type discoveryNodeClientFake struct{}

func (f *discoveryNodeClientFake) SubscribeNewHead(context.Context, chan<- *types.Header) (ethereum.Subscription, error) {
	return event.NewSubscription(func(<-chan struct{}) error { return nil }), nil
}

func (f *discoveryNodeClientFake) BlockNumber(context.Context) (uint64, error) { return 0, nil }

func (f *discoveryNodeClientFake) BlockByNumber(context.Context, *big.Int) (*types.Block, error) {
	return nil, nil
}

func (f *discoveryNodeClientFake) TransactionReceipt(context.Context, common.Hash) (*types.Receipt, error) {
	return &types.Receipt{}, nil
}

func (f *discoveryNodeClientFake) FilterLogs(context.Context, ethereum.FilterQuery) ([]types.Log, error) {
	return nil, nil
}

func (f *discoveryNodeClientFake) ChainID(context.Context) (*big.Int, error) {
	return big.NewInt(1), nil
}

type discoveryCursorNodeClientFake struct {
	blockByNumberCalls int
	latestBlock        *types.Block
}

func (f *discoveryCursorNodeClientFake) SubscribeNewHead(context.Context, chan<- *types.Header) (ethereum.Subscription, error) {
	return event.NewSubscription(func(<-chan struct{}) error { return nil }), nil
}

func (f *discoveryCursorNodeClientFake) BlockNumber(context.Context) (uint64, error) { return 0, nil }

func (f *discoveryCursorNodeClientFake) BlockByNumber(context.Context, *big.Int) (*types.Block, error) {
	f.blockByNumberCalls++
	return f.latestBlock, nil
}

func (f *discoveryCursorNodeClientFake) TransactionReceipt(context.Context, common.Hash) (*types.Receipt, error) {
	return &types.Receipt{}, nil
}

func (f *discoveryCursorNodeClientFake) FilterLogs(context.Context, ethereum.FilterQuery) ([]types.Log, error) {
	return nil, nil
}

func (f *discoveryCursorNodeClientFake) ChainID(context.Context) (*big.Int, error) {
	return big.NewInt(1), nil
}

type swapLogDiscoveryNodeClientFake struct {
	blockByNumberCalls int
	filterLogsCalls    int
	block              *types.Block
	blockErr           error
	logs               []types.Log
	filterErr          error
	gotQuery           ethereum.FilterQuery
}

func (f *swapLogDiscoveryNodeClientFake) SubscribeNewHead(context.Context, chan<- *types.Header) (ethereum.Subscription, error) {
	return event.NewSubscription(func(<-chan struct{}) error { return nil }), nil
}

func (f *swapLogDiscoveryNodeClientFake) BlockNumber(context.Context) (uint64, error) {
	return 0, nil
}

func (f *swapLogDiscoveryNodeClientFake) BlockByNumber(context.Context, *big.Int) (*types.Block, error) {
	f.blockByNumberCalls++
	if f.blockErr != nil {
		return nil, f.blockErr
	}
	return f.block, nil
}

func (f *swapLogDiscoveryNodeClientFake) TransactionReceipt(context.Context, common.Hash) (*types.Receipt, error) {
	return &types.Receipt{}, nil
}

func (f *swapLogDiscoveryNodeClientFake) FilterLogs(_ context.Context, q ethereum.FilterQuery) ([]types.Log, error) {
	f.filterLogsCalls++
	f.gotQuery = q
	if f.filterErr != nil {
		return nil, f.filterErr
	}
	return append([]types.Log(nil), f.logs...), nil
}

func (f *swapLogDiscoveryNodeClientFake) ChainID(context.Context) (*big.Int, error) {
	return big.NewInt(1), nil
}

type discoveryProjectStoreFake struct {
	metas       []appstore.ProjectMeta
	pairMetas   []appstore.ProjectMeta
	errs        []error
	calls       int
	pairCalls   int
	maxBlock    uint64
	maxBlockOK  bool
	maxBlockErr error

	gotCreator     common.Address
	gotBlockNumber uint64
	gotTxIndex     uint64
	gotPairs       []common.Address
}

func (s *discoveryProjectStoreFake) SaveProjectMeta(context.Context, appstore.ProjectMeta) error {
	return nil
}

func (s *discoveryProjectStoreFake) GetMaxProjectBlockNumber(context.Context) (uint64, bool, error) {
	return s.maxBlock, s.maxBlockOK, s.maxBlockErr
}

func (s *discoveryProjectStoreFake) ListProjectMetas(context.Context) ([]appstore.ProjectMeta, error) {
	return nil, nil
}

func (s *discoveryProjectStoreFake) ListAllProjectMetas(context.Context) ([]appstore.ProjectMeta, error) {
	return nil, nil
}

func (s *discoveryProjectStoreFake) ListProjectMetasByPairAddresses(_ context.Context, pairs []common.Address) ([]appstore.ProjectMeta, error) {
	s.pairCalls++
	s.gotPairs = append([]common.Address(nil), pairs...)
	return append([]appstore.ProjectMeta(nil), s.pairMetas...), nil
}

func (s *discoveryProjectStoreFake) UpdateProjectSourceCode(context.Context, common.Address, string) error {
	return nil
}

func (s *discoveryProjectStoreFake) UpdateProjectCodeBinHash(context.Context, common.Address, common.Hash) error {
	return nil
}

func (s *discoveryProjectStoreFake) UpdateProjectSourceQualityReport(context.Context, common.Address, string) error {
	return nil
}

func (s *discoveryProjectStoreFake) UpsertProjectAveDetail(context.Context, common.Address, appstore.ProjectAveDetail) error {
	return nil
}

func (s *discoveryProjectStoreFake) UpdateProjectCreatorResult(context.Context, common.Address, appstore.SimulateResult) error {
	return nil
}

func (s *discoveryProjectStoreFake) UpdateProjectReport(context.Context, common.Address, appstore.ProjectReport) error {
	return nil
}

func (s *discoveryProjectStoreFake) ListProjectMetasByCreator(context.Context, common.Address) ([]appstore.ProjectMeta, error) {
	return nil, nil
}

func (s *discoveryProjectStoreFake) ListProjectMetasByCreatorBefore(_ context.Context, creator common.Address, blockNumber uint64, txIndex uint64) ([]appstore.ProjectMeta, error) {
	s.calls++
	s.gotCreator = creator
	s.gotBlockNumber = blockNumber
	s.gotTxIndex = txIndex
	if len(s.errs) > 0 {
		err := s.errs[0]
		s.errs = s.errs[1:]
		if err != nil {
			return nil, err
		}
	}
	return append([]appstore.ProjectMeta(nil), s.metas...), nil
}

func (s *discoveryProjectStoreFake) GetProjectMetaByContract(context.Context, common.Address) (*appstore.ProjectMeta, error) {
	return nil, nil
}

func TestProjectDiscoveryIndexerLoadCursorUsesStoreMaxProjectBlock(t *testing.T) {
	ctx := context.Background()
	node := &discoveryCursorNodeClientFake{}
	indexer := &projectDiscoveryIndexerImpl{
		nodeClient:   node,
		projectStore: &discoveryProjectStoreFake{maxBlock: 103, maxBlockOK: true},
	}

	cursor, err := indexer.loadCursor(ctx)
	if err != nil {
		t.Fatalf("load cursor: %v", err)
	}
	if cursor != 102 {
		t.Fatalf("cursor = %d, want 102", cursor)
	}
	if node.blockByNumberCalls != 0 {
		t.Fatalf("block by number calls = %d, want 0", node.blockByNumberCalls)
	}
}

func TestProjectDiscoveryIndexerFetchPancakeV2SwapPairAddresses(t *testing.T) {
	ctx := context.Background()
	pairA := common.HexToAddress("0x00000000000000000000000000000000000000a1")
	pairB := common.HexToAddress("0x00000000000000000000000000000000000000b2")
	node := &swapLogDiscoveryNodeClientFake{
		logs: []types.Log{
			{Address: pairA},
			{Address: pairA},
			{Address: pairB},
		},
	}
	indexer := &projectDiscoveryIndexerImpl{nodeClient: node}

	addresses, err := indexer.fetchPancakeV2SwapPairAddresses(ctx, 123)
	if err != nil {
		t.Fatalf("fetch pancake v2 swap pair addresses: %v", err)
	}
	if node.filterLogsCalls != 1 {
		t.Fatalf("filter logs calls = %d, want 1", node.filterLogsCalls)
	}
	if node.gotQuery.FromBlock == nil || node.gotQuery.FromBlock.Uint64() != 123 {
		t.Fatalf("from block = %v, want 123", node.gotQuery.FromBlock)
	}
	if node.gotQuery.ToBlock == nil || node.gotQuery.ToBlock.Uint64() != 123 {
		t.Fatalf("to block = %v, want 123", node.gotQuery.ToBlock)
	}
	if len(node.gotQuery.Topics) != 1 || len(node.gotQuery.Topics[0]) != 1 || node.gotQuery.Topics[0][0] != pancakeV2SwapTopicHash {
		t.Fatalf("topics = %v, want pancake v2 swap topic", node.gotQuery.Topics)
	}
	if len(addresses) != 2 {
		t.Fatalf("address count = %d, want 2", len(addresses))
	}
	if addresses[0] != pairA || addresses[1] != pairB {
		t.Fatalf("addresses = %v, want %s then %s", addresses, pairA.Hex(), pairB.Hex())
	}
}

func TestProjectDiscoveryIndexerScanBlockScansSwapLogsWithoutCandidatesOrIntake(t *testing.T) {
	ctx := context.Background()
	node := &swapLogDiscoveryNodeClientFake{
		block: types.NewBlockWithHeader(&types.Header{Number: big.NewInt(123)}),
		logs: []types.Log{
			{Address: common.HexToAddress("0x00000000000000000000000000000000000000a1")},
		},
	}
	indexer := &projectDiscoveryIndexerImpl{nodeClient: node}

	if err := indexer.scanBlock(ctx, 123, ProjectDiscoverySourceCatchUp); err != nil {
		t.Fatalf("scan block: %v", err)
	}
	if node.blockByNumberCalls != 1 {
		t.Fatalf("block by number calls = %d, want 1", node.blockByNumberCalls)
	}
	if node.filterLogsCalls != 1 {
		t.Fatalf("filter logs calls = %d, want 1", node.filterLogsCalls)
	}
}

func TestProjectDiscoveryIndexerScanBlockRunsProjectCreationAndSwapTasks(t *testing.T) {
	ctx := context.Background()
	node := &swapLogDiscoveryNodeClientFake{
		block: types.NewBlockWithHeader(&types.Header{Number: big.NewInt(123)}),
		logs: []types.Log{
			{Address: common.HexToAddress("0x00000000000000000000000000000000000000a1")},
		},
	}
	indexer := &projectDiscoveryIndexerImpl{
		nodeClient: node,
		intake:     &discoveryIntakeImpl{reconciler: &intakeReconcilerFake{}},
	}

	if err := indexer.scanBlock(ctx, 123, ProjectDiscoverySourceCatchUp); err != nil {
		t.Fatalf("scan block: %v", err)
	}
	if node.blockByNumberCalls != 1 {
		t.Fatalf("block by number calls = %d, want 1", node.blockByNumberCalls)
	}
	if node.filterLogsCalls != 1 {
		t.Fatalf("filter logs calls = %d, want 1", node.filterLogsCalls)
	}
}

func TestProjectDiscoveryIndexerScanBlockReturnsProjectCreationTaskError(t *testing.T) {
	ctx := context.Background()
	node := &swapLogDiscoveryNodeClientFake{
		blockErr: errors.New("block failed"),
	}
	indexer := &projectDiscoveryIndexerImpl{nodeClient: node}

	if err := indexer.scanBlock(ctx, 123, ProjectDiscoverySourceCatchUp); err == nil {
		t.Fatal("scan block error = nil, want error")
	}
	if node.blockByNumberCalls != 1 {
		t.Fatalf("block by number calls = %d, want 1", node.blockByNumberCalls)
	}
	if node.filterLogsCalls != 1 {
		t.Fatalf("filter logs calls = %d, want 1", node.filterLogsCalls)
	}
}

func TestProjectDiscoveryIndexerScanBlockReturnsPairSwapTaskError(t *testing.T) {
	ctx := context.Background()
	node := &swapLogDiscoveryNodeClientFake{
		block:     types.NewBlockWithHeader(&types.Header{Number: big.NewInt(123)}),
		filterErr: errors.New("filter failed"),
	}
	indexer := &projectDiscoveryIndexerImpl{nodeClient: node}

	if err := indexer.scanBlock(ctx, 123, ProjectDiscoverySourceCatchUp); err == nil {
		t.Fatal("scan block error = nil, want error")
	}
	if node.blockByNumberCalls != 1 {
		t.Fatalf("block by number calls = %d, want 1", node.blockByNumberCalls)
	}
	if node.filterLogsCalls != 1 {
		t.Fatalf("filter logs calls = %d, want 1", node.filterLogsCalls)
	}
}

func TestProjectDiscoveryIndexerSchedulesCachedProjectBySwapPair(t *testing.T) {
	ctx := context.Background()
	cache := newProjectSnapshotCacheTest(t)
	pair := common.HexToAddress("0x00000000000000000000000000000000000000a1")
	contract := common.HexToAddress("0x00000000000000000000000000000000000000c1")
	if err := cache.SetProject(ctx, &Project{Meta: ProjectMeta{
		BlockNumber: 77,
		Contract:    contract,
		WethPair:    pair,
	}}); err != nil {
		t.Fatalf("set project: %v", err)
	}
	node := &swapLogDiscoveryNodeClientFake{
		block: types.NewBlockWithHeader(&types.Header{Number: big.NewInt(123)}),
		logs:  []types.Log{{Address: pair}},
	}
	store := &discoveryProjectStoreFake{}
	reconciler := &intakeReconcilerFake{}
	indexer := &projectDiscoveryIndexerImpl{
		nodeClient:   node,
		projectCache: cache,
		projectStore: store,
		intake:       &discoveryIntakeImpl{reconciler: reconciler},
	}

	if err := indexer.scanBlock(ctx, 123, ProjectDiscoverySourceCatchUp); err != nil {
		t.Fatalf("scan block: %v", err)
	}
	if store.pairCalls != 0 {
		t.Fatalf("store pair calls = %d, want 0", store.pairCalls)
	}
	if reconciler.scheduleCalls != 1 || len(reconciler.scheduled) != 1 {
		t.Fatalf("scheduled = calls %d items %d, want 1/1", reconciler.scheduleCalls, len(reconciler.scheduled))
	}
	if got := reconciler.scheduled[0]; got.Contract != contract || got.Source != ProjectDiscoverySourcePairSwap {
		t.Fatalf("scheduled candidate = %+v, want contract %s source %s", got, contract.Hex(), ProjectDiscoverySourcePairSwap)
	}
}

func TestProjectDiscoveryIndexerSchedulesStoredProjectBySwapPairAndSeedsCache(t *testing.T) {
	ctx := context.Background()
	cache := newProjectSnapshotCacheTest(t)
	pair := common.HexToAddress("0x00000000000000000000000000000000000000a1")
	contract := common.HexToAddress("0x00000000000000000000000000000000000000c1")
	node := &swapLogDiscoveryNodeClientFake{
		block: types.NewBlockWithHeader(&types.Header{Number: big.NewInt(123)}),
		logs:  []types.Log{{Address: pair}},
	}
	store := &discoveryProjectStoreFake{pairMetas: []appstore.ProjectMeta{{
		BlockNumber: 77,
		Contract:    contract,
		WethPair:    pair,
	}}}
	reconciler := &intakeReconcilerFake{}
	indexer := &projectDiscoveryIndexerImpl{
		nodeClient:   node,
		projectCache: cache,
		projectStore: store,
		intake:       &discoveryIntakeImpl{reconciler: reconciler},
	}

	if err := indexer.scanBlock(ctx, 123, ProjectDiscoverySourceCatchUp); err != nil {
		t.Fatalf("scan block: %v", err)
	}
	if store.pairCalls != 1 || len(store.gotPairs) != 1 || store.gotPairs[0] != pair {
		t.Fatalf("store pair query = calls %d pairs %v, want one call with %s", store.pairCalls, store.gotPairs, pair.Hex())
	}
	if reconciler.scheduleCalls != 1 || len(reconciler.scheduled) != 1 {
		t.Fatalf("scheduled = calls %d items %d, want 1/1", reconciler.scheduleCalls, len(reconciler.scheduled))
	}
	if got := reconciler.scheduled[0]; got.Contract != contract || got.Source != ProjectDiscoverySourcePairSwap {
		t.Fatalf("scheduled candidate = %+v, want contract %s source %s", got, contract.Hex(), ProjectDiscoverySourcePairSwap)
	}
	cached, exists, err := cache.GetProject(ctx, contract)
	if err != nil {
		t.Fatalf("get cached project: %v", err)
	}
	if !exists || cached == nil || cached.Meta.WethPair != pair {
		t.Fatalf("cached project = %+v exists %t, want weth pair %s", cached, exists, pair.Hex())
	}
}

func TestProjectDiscoveryIndexerSkipsSwapPairsWithoutProjectMatch(t *testing.T) {
	ctx := context.Background()
	cache := newProjectSnapshotCacheTest(t)
	pair := common.HexToAddress("0x00000000000000000000000000000000000000a1")
	node := &swapLogDiscoveryNodeClientFake{
		block: types.NewBlockWithHeader(&types.Header{Number: big.NewInt(123)}),
		logs:  []types.Log{{Address: pair}},
	}
	store := &discoveryProjectStoreFake{}
	reconciler := &intakeReconcilerFake{}
	indexer := &projectDiscoveryIndexerImpl{
		nodeClient:   node,
		projectCache: cache,
		projectStore: store,
		intake:       &discoveryIntakeImpl{reconciler: reconciler},
	}

	if err := indexer.scanBlock(ctx, 123, ProjectDiscoverySourceCatchUp); err != nil {
		t.Fatalf("scan block: %v", err)
	}
	if store.pairCalls != 1 {
		t.Fatalf("store pair calls = %d, want 1", store.pairCalls)
	}
	if reconciler.scheduleCalls != 0 {
		t.Fatalf("schedule calls = %d, want 0", reconciler.scheduleCalls)
	}
}

func TestProjectDiscoveryIndexerDeduplicatesSwapPairProjectSchedules(t *testing.T) {
	ctx := context.Background()
	cache := newProjectSnapshotCacheTest(t)
	wethPair := common.HexToAddress("0x00000000000000000000000000000000000000a1")
	usdtPair := common.HexToAddress("0x00000000000000000000000000000000000000a2")
	contract := common.HexToAddress("0x00000000000000000000000000000000000000c1")
	if err := cache.SetProject(ctx, &Project{Meta: ProjectMeta{
		Contract: contract,
		WethPair: wethPair,
		UsdtPair: usdtPair,
	}}); err != nil {
		t.Fatalf("set project: %v", err)
	}
	node := &swapLogDiscoveryNodeClientFake{
		block: types.NewBlockWithHeader(&types.Header{Number: big.NewInt(123)}),
		logs: []types.Log{
			{Address: wethPair},
			{Address: wethPair},
			{Address: usdtPair},
		},
	}
	reconciler := &intakeReconcilerFake{}
	indexer := &projectDiscoveryIndexerImpl{
		nodeClient:   node,
		projectCache: cache,
		projectStore: &discoveryProjectStoreFake{},
		intake:       &discoveryIntakeImpl{reconciler: reconciler},
	}

	if err := indexer.scanBlock(ctx, 123, ProjectDiscoverySourceCatchUp); err != nil {
		t.Fatalf("scan block: %v", err)
	}
	if reconciler.scheduleCalls != 1 || len(reconciler.scheduled) != 1 {
		t.Fatalf("scheduled = calls %d items %d, want 1/1", reconciler.scheduleCalls, len(reconciler.scheduled))
	}
	if reconciler.scheduled[0].Contract != contract {
		t.Fatalf("scheduled contract = %s, want %s", reconciler.scheduled[0].Contract.Hex(), contract.Hex())
	}
}

type intakeReconcilerFake struct {
	calls         int
	scheduleCalls int
	items         []DiscoveredProjectCandidate
	scheduled     []DiscoveredProjectCandidate
}

func (r *intakeReconcilerFake) Start(context.Context) error { return nil }
func (r *intakeReconcilerFake) Stop() error                 { return nil }

func (r *intakeReconcilerFake) InitProject(_ context.Context, items []DiscoveredProjectCandidate) error {
	r.calls++
	r.items = append([]DiscoveredProjectCandidate(nil), items...)
	return nil
}

func (r *intakeReconcilerFake) ScheduleProjects(_ context.Context, items []DiscoveredProjectCandidate) error {
	r.scheduleCalls++
	r.scheduled = append([]DiscoveredProjectCandidate(nil), items...)
	return nil
}

func TestDiscoveryIntakeCandidatesPassesBatchToInitProject(t *testing.T) {
	ctx := context.Background()
	reconciler := &intakeReconcilerFake{}
	intake := &discoveryIntakeImpl{reconciler: reconciler}
	items := []DiscoveredProjectCandidate{
		{Contract: common.HexToAddress("0x00000000000000000000000000000000000000a1")},
		{Contract: common.HexToAddress("0x00000000000000000000000000000000000000a2")},
	}

	if err := intake.IntakeCandidates(ctx, items); err != nil {
		t.Fatalf("intake candidates: %v", err)
	}
	if reconciler.calls != 1 {
		t.Fatalf("init project calls = %d, want 1", reconciler.calls)
	}
	if len(reconciler.items) != len(items) {
		t.Fatalf("init project item count = %d, want %d", len(reconciler.items), len(items))
	}
}

func TestDiscoveryInitProjectCachesValidERC20WithoutCreatorHistoryQuery(t *testing.T) {
	ctx := context.Background()
	cache := newProjectSnapshotCacheTest(t)
	creator := common.HexToAddress("0x00000000000000000000000000000000000000a0")
	contract := common.HexToAddress("0x00000000000000000000000000000000000000c0")
	store := &discoveryProjectStoreFake{metas: []appstore.ProjectMeta{
		{Contract: common.HexToAddress("0x00000000000000000000000000000000000000a1"), Creator: creator},
		{Contract: contract, Creator: creator},
	}}
	reconciler := &projectStateReconcilerImpl{
		discoveryNodeClient: &discoveryNodeClientFake{},
		projectCache:        cache,
		projectStore:        store,
		fetcher:             &discoveryFetcherFake{},
		scheduled:           map[common.Address]*scheduledProject{},
	}

	err := reconciler.InitProject(ctx, []DiscoveredProjectCandidate{{
		Contract:    contract,
		Creator:     creator,
		TxHash:      common.HexToHash("0x0101010101010101010101010101010101010101010101010101010101010101"),
		BlockNumber: 103,
		TxIndex:     7,
		Source:      ProjectDiscoverySourceCatchUp,
	}})
	if err != nil {
		t.Fatalf("init project: %v", err)
	}

	project, exists, err := cache.GetProject(ctx, contract)
	if err != nil {
		t.Fatalf("get project: %v", err)
	}
	if !exists || project == nil {
		t.Fatal("project missing after sync")
	}
	if project.Meta.ChainState.TokenContract != contract {
		t.Fatalf("chain state token contract = %s, want %s", project.Meta.ChainState.TokenContract.Hex(), contract.Hex())
	}
	if len(project.Meta.CreatorHistoricalProjects) != 0 {
		t.Fatalf("creator historical projects = %v, want empty", addressHexes(project.Meta.CreatorHistoricalProjects))
	}
	if store.calls != 0 {
		t.Fatalf("store calls = %d, want 0", store.calls)
	}
	if reconciler.scheduled[contract] == nil {
		t.Fatal("project was not scheduled")
	}
}

func TestDiscoveryInitProjectSkipsInvalidERC20(t *testing.T) {
	ctx := context.Background()
	cache := newProjectSnapshotCacheTest(t)
	contract := common.HexToAddress("0x00000000000000000000000000000000000000c0")
	reconciler := &projectStateReconcilerImpl{
		projectCache: cache,
		fetcher: &discoveryFetcherFake{snapshots: []athenacontract.AthenaProject{{
			TokenContract: contract,
			Token: athenacontract.AthenaToken{
				IsValidERC20: false,
			},
		}}},
		scheduled: map[common.Address]*scheduledProject{},
	}

	err := reconciler.InitProject(ctx, []DiscoveredProjectCandidate{{
		Contract: contract,
		Source:   ProjectDiscoverySourceCatchUp,
	}})
	if err != nil {
		t.Fatalf("init project: %v", err)
	}

	_, exists, err := cache.GetProject(ctx, contract)
	if err != nil {
		t.Fatalf("get project: %v", err)
	}
	if exists {
		t.Fatal("project exists after invalid ERC20 init")
	}
	if len(reconciler.scheduled) != 0 {
		t.Fatalf("scheduled count = %d, want 0", len(reconciler.scheduled))
	}
}
