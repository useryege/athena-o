package application

import (
	"context"
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

type discoveryProjectStoreFake struct {
	metas       []appstore.ProjectMeta
	errs        []error
	calls       int
	maxBlock    uint64
	maxBlockOK  bool
	maxBlockErr error

	gotCreator     common.Address
	gotBlockNumber uint64
	gotTxIndex     uint64
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

func (s *discoveryProjectStoreFake) UpdateProjectSourceCode(context.Context, common.Address, string) error {
	return nil
}

func (s *discoveryProjectStoreFake) UpdateProjectCodeBinHash(context.Context, common.Address, common.Hash) error {
	return nil
}

func (s *discoveryProjectStoreFake) UpdateProjectSourceQualityReport(context.Context, common.Address, string) error {
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

type intakeReconcilerFake struct {
	calls int
	items []DiscoveredProjectCandidate
}

func (r *intakeReconcilerFake) Start(context.Context) error { return nil }
func (r *intakeReconcilerFake) Stop() error                 { return nil }

func (r *intakeReconcilerFake) InitProject(_ context.Context, items []DiscoveredProjectCandidate) error {
	r.calls++
	r.items = append([]DiscoveredProjectCandidate(nil), items...)
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
