package application

import (
	"context"
	"errors"
	"math/big"
	"testing"
	"time"

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

type discoveryProjectStoreFake struct {
	metas []appstore.ProjectMeta
	errs  []error
	calls int

	gotCreator     common.Address
	gotBlockNumber uint64
	gotTxIndex     uint64
}

func (s *discoveryProjectStoreFake) SaveProjectMeta(context.Context, appstore.ProjectMeta) error {
	return nil
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

func TestDiscoverySyncProjectsSetsCreatorHistoricalProjects(t *testing.T) {
	ctx := context.Background()
	cache := newProjectSnapshotCacheTest(t)
	creator := common.HexToAddress("0x00000000000000000000000000000000000000a0")
	contract := common.HexToAddress("0x00000000000000000000000000000000000000c0")
	previousA := common.HexToAddress("0x00000000000000000000000000000000000000a1")
	previousB := common.HexToAddress("0x00000000000000000000000000000000000000a2")
	store := &discoveryProjectStoreFake{metas: []appstore.ProjectMeta{
		{Contract: previousA, Creator: creator},
		{Contract: previousB, Creator: creator},
		{Contract: previousA, Creator: creator},
		{Contract: contract, Creator: creator},
	}}
	intake := &discoveryIntakeImpl{
		nodeClient:   &discoveryNodeClientFake{},
		projectCache: cache,
		projectStore: store,
		fetcher:      &discoveryFetcherFake{},
		publisher:    &persistencePublisherFake{},
	}

	err := intake.syncProjects(ctx, []*Project{{Meta: ProjectMeta{
		Contract:    contract,
		Creator:     creator,
		TxHash:      common.HexToHash("0x0101010101010101010101010101010101010101010101010101010101010101"),
		BlockNumber: 103,
		TxIndex:     7,
	}}})
	if err != nil {
		t.Fatalf("sync projects: %v", err)
	}

	project, exists, err := cache.GetProject(ctx, contract)
	if err != nil {
		t.Fatalf("get project: %v", err)
	}
	if !exists || project == nil {
		t.Fatal("project missing after sync")
	}
	want := []common.Address{previousA, previousB}
	if got := project.Meta.CreatorHistoricalProjects; len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("creator historical projects = %v, want %v", addressHexes(got), addressHexes(want))
	}
	if store.calls != 1 || store.gotCreator != creator || store.gotBlockNumber != 103 || store.gotTxIndex != 7 {
		t.Fatalf("store query = calls %d creator %s block %d tx %d", store.calls, store.gotCreator.Hex(), store.gotBlockNumber, store.gotTxIndex)
	}
}

func TestDiscoverySyncProjectsRetriesCreatorHistoricalProjects(t *testing.T) {
	restore := setCreatorHistoricalProjectRetryDelaysForTest(t)
	defer restore()

	ctx := context.Background()
	cache := newProjectSnapshotCacheTest(t)
	creator := common.HexToAddress("0x00000000000000000000000000000000000000a0")
	contract := common.HexToAddress("0x00000000000000000000000000000000000000c0")
	previous := common.HexToAddress("0x00000000000000000000000000000000000000a1")
	store := &discoveryProjectStoreFake{
		metas: []appstore.ProjectMeta{{Contract: previous, Creator: creator}},
		errs:  []error{errors.New("one"), errors.New("two"), errors.New("three"), nil},
	}
	intake := &discoveryIntakeImpl{
		nodeClient:   &discoveryNodeClientFake{},
		projectCache: cache,
		projectStore: store,
		fetcher:      &discoveryFetcherFake{},
		publisher:    &persistencePublisherFake{},
	}

	err := intake.syncProjects(ctx, []*Project{{Meta: ProjectMeta{
		Contract:    contract,
		Creator:     creator,
		TxHash:      common.HexToHash("0x0202020202020202020202020202020202020202020202020202020202020202"),
		BlockNumber: 103,
		TxIndex:     7,
	}}})
	if err != nil {
		t.Fatalf("sync projects: %v", err)
	}

	project, exists, err := cache.GetProject(ctx, contract)
	if err != nil {
		t.Fatalf("get project: %v", err)
	}
	if !exists || project == nil {
		t.Fatal("project missing after sync")
	}
	if store.calls != 4 {
		t.Fatalf("store calls = %d, want 4", store.calls)
	}
	if got := project.Meta.CreatorHistoricalProjects; len(got) != 1 || got[0] != previous {
		t.Fatalf("creator historical projects = %v, want [%s]", addressHexes(got), previous.Hex())
	}
}

func TestDiscoverySyncProjectsContinuesWithEmptyCreatorHistoricalProjectsAfterRetryFailure(t *testing.T) {
	restore := setCreatorHistoricalProjectRetryDelaysForTest(t)
	defer restore()

	ctx := context.Background()
	cache := newProjectSnapshotCacheTest(t)
	creator := common.HexToAddress("0x00000000000000000000000000000000000000a0")
	contract := common.HexToAddress("0x00000000000000000000000000000000000000c0")
	store := &discoveryProjectStoreFake{
		errs: []error{errors.New("one"), errors.New("two"), errors.New("three"), errors.New("four")},
	}
	intake := &discoveryIntakeImpl{
		nodeClient:   &discoveryNodeClientFake{},
		projectCache: cache,
		projectStore: store,
		fetcher:      &discoveryFetcherFake{},
		publisher:    &persistencePublisherFake{},
	}

	err := intake.syncProjects(ctx, []*Project{{Meta: ProjectMeta{
		Contract:    contract,
		Creator:     creator,
		TxHash:      common.HexToHash("0x0303030303030303030303030303030303030303030303030303030303030303"),
		BlockNumber: 103,
		TxIndex:     7,
	}}})
	if err != nil {
		t.Fatalf("sync projects: %v", err)
	}

	project, exists, err := cache.GetProject(ctx, contract)
	if err != nil {
		t.Fatalf("get project: %v", err)
	}
	if !exists || project == nil {
		t.Fatal("project missing after sync")
	}
	if store.calls != 4 {
		t.Fatalf("store calls = %d, want 4", store.calls)
	}
	if len(project.Meta.CreatorHistoricalProjects) != 0 {
		t.Fatalf("creator historical projects = %v, want empty", addressHexes(project.Meta.CreatorHistoricalProjects))
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
