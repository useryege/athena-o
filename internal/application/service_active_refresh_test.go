package application

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	athenacontract "github.com/useryege/athena/pkg/abi/ATHENA"
)

type activeRefreshCacheMock struct {
	listActive   []*Project
	listArchived []*Project
	getByKey     map[string]*Project

	listArchivedErr   error
	listArchivedCalls [][2]int32

	setCalls []*Project
	setErr   error
}

func (m *activeRefreshCacheMock) ReplaceAll(context.Context, []*Project) error {
	return nil
}

func (m *activeRefreshCacheMock) SetProject(_ context.Context, project *Project) error {
	if m.setErr != nil {
		return m.setErr
	}
	cloned := cloneProjectForTest(project)
	m.setCalls = append(m.setCalls, cloned)
	if m.getByKey == nil {
		m.getByKey = map[string]*Project{}
	}
	m.getByKey[project.Meta.Contract.Hex()] = cloned
	return nil
}

func (m *activeRefreshCacheMock) UpdateProject(ctx context.Context, contract common.Address, updater ProjectUpdater) (bool, error) {
	current, ok, err := m.GetProject(ctx, contract)
	if err != nil {
		return false, err
	}
	next, changed, err := updater(current, ok)
	if err != nil {
		return false, err
	}
	if !changed {
		return false, nil
	}
	if next == nil {
		if m.getByKey != nil {
			delete(m.getByKey, contract.Hex())
		}
		return true, nil
	}
	if next.Meta.Contract == (common.Address{}) {
		next.Meta.Contract = contract
	}
	if err := m.SetProject(ctx, next); err != nil {
		return false, err
	}
	return true, nil
}

func (m *activeRefreshCacheMock) DeleteProject(context.Context, common.Address) error {
	return nil
}

func (m *activeRefreshCacheMock) GetProject(_ context.Context, contract common.Address) (*Project, bool, error) {
	if m.getByKey == nil {
		return nil, false, nil
	}
	project, ok := m.getByKey[contract.Hex()]
	if !ok || project == nil {
		return nil, false, nil
	}
	return cloneProjectForTest(project), true, nil
}

func (m *activeRefreshCacheMock) GetMaxProjectBlockNumber(context.Context) (uint64, bool, error) {
	return 0, false, nil
}

func (m *activeRefreshCacheMock) ListActiveProjects(context.Context) ([]*Project, error) {
	items := make([]*Project, 0, len(m.listActive))
	for _, project := range m.listActive {
		items = append(items, cloneProjectForTest(project))
	}
	return items, nil
}

func (m *activeRefreshCacheMock) ListArchivedProjects(_ context.Context, page int32, pageSize int32) ([]*Project, int64, int32, int32, error) {
	if m.listArchivedErr != nil {
		return nil, 0, 0, 0, m.listArchivedErr
	}
	page, pageSize = normalizeCachePage(page, pageSize)
	m.listArchivedCalls = append(m.listArchivedCalls, [2]int32{page, pageSize})

	total := int64(len(m.listArchived))
	if total == 0 {
		return nil, total, page, pageSize, nil
	}

	start := int((page - 1) * pageSize)
	if start >= len(m.listArchived) {
		return nil, total, page, pageSize, nil
	}
	end := start + int(pageSize)
	if end > len(m.listArchived) {
		end = len(m.listArchived)
	}

	items := make([]*Project, 0, end-start)
	for _, project := range m.listArchived[start:end] {
		items = append(items, cloneProjectForTest(project))
	}
	return items, total, page, pageSize, nil
}

type activeRefreshFetcherMock struct {
	projects               []athenacontract.AthenaProject
	projectsWithSimulation []athenacontract.AthenaProjectWithSimulationState
	states                 []athenacontract.AthenaSimulationState

	projectErr               error
	projectWithSimulationErr error
	stateErr                 error

	projectCalls               [][]athenacontract.AthenaProjectQuery
	projectWithSimulationCalls [][]athenacontract.AthenaProjectQuery
	stateCalls                 [][]athenacontract.AthenaProjectQuery
}

func (m *activeRefreshFetcherMock) FetchProject(context.Context, athenacontract.AthenaProjectQuery) (athenacontract.AthenaProject, error) {
	return athenacontract.AthenaProject{}, nil
}

func (m *activeRefreshFetcherMock) FetchProjects(_ context.Context, queries []athenacontract.AthenaProjectQuery) ([]athenacontract.AthenaProject, error) {
	call := make([]athenacontract.AthenaProjectQuery, len(queries))
	copy(call, queries)
	m.projectCalls = append(m.projectCalls, call)
	return m.projects, m.projectErr
}

func (m *activeRefreshFetcherMock) FetchProjectsWithSimulationState(_ context.Context, queries []athenacontract.AthenaProjectQuery) ([]athenacontract.AthenaProjectWithSimulationState, error) {
	call := make([]athenacontract.AthenaProjectQuery, len(queries))
	copy(call, queries)
	m.projectWithSimulationCalls = append(m.projectWithSimulationCalls, call)
	return m.projectsWithSimulation, m.projectWithSimulationErr
}

func (m *activeRefreshFetcherMock) FetchSimulationState(context.Context, athenacontract.AthenaProjectQuery) (athenacontract.AthenaSimulationState, error) {
	return athenacontract.AthenaSimulationState{}, nil
}

func (m *activeRefreshFetcherMock) FetchSimulationStates(_ context.Context, queries []athenacontract.AthenaProjectQuery) ([]athenacontract.AthenaSimulationState, error) {
	call := make([]athenacontract.AthenaProjectQuery, len(queries))
	copy(call, queries)
	m.stateCalls = append(m.stateCalls, call)
	return m.states, m.stateErr
}

type simulateCall struct {
	msgCaller        common.Address
	tokenAddress     common.Address
	wethPairContract common.Address
	usdtPairContract common.Address
}

type activeRefreshSimulatorMock struct {
	results map[string]SimulateResult
	errs    map[string]error

	calls []simulateCall
}

func (m *activeRefreshSimulatorMock) SimulatePrimary(_ context.Context, msgCaller common.Address, tokenAddress common.Address, wethPairContract common.Address, usdtPairContract common.Address, _ athenacontract.AthenaSimulationState) (SimulateResult, error) {
	m.calls = append(m.calls, simulateCall{
		msgCaller:        msgCaller,
		tokenAddress:     tokenAddress,
		wethPairContract: wethPairContract,
		usdtPairContract: usdtPairContract,
	})
	if err, ok := m.errs[tokenAddress.Hex()]; ok {
		return SimulateResult{}, err
	}
	if result, ok := m.results[tokenAddress.Hex()]; ok {
		return result, nil
	}
	return SimulateResult{}, nil
}

func cloneProjectForTest(project *Project) *Project {
	if project == nil {
		return nil
	}
	cloned := *project
	return &cloned
}

func TestRefreshActiveProjectStatesUpdatesChainStateAndKeepsMeta(t *testing.T) {
	contract := common.HexToAddress("0x1000000000000000000000000000000000000001")
	creator := common.HexToAddress("0x2000000000000000000000000000000000000002")
	originalResult := SimulateResult{CanMintViaTransferToWethPair: true}

	cache := &activeRefreshCacheMock{
		listActive: []*Project{
			{
				Meta: ProjectMeta{Contract: contract, Creator: creator, CreatorResult: originalResult},
				ChainState: athenacontract.AthenaProject{
					TokenContract: contract,
					Token:         athenacontract.AthenaToken{Symbol: "OLD"},
				},
			},
		},
		getByKey: map[string]*Project{
			contract.Hex(): {
				Meta: ProjectMeta{Contract: contract, Creator: creator, CreatorResult: originalResult},
				ChainState: athenacontract.AthenaProject{
					TokenContract: contract,
					Token:         athenacontract.AthenaToken{Symbol: "OLD"},
				},
			},
		},
	}
	fetcher := &activeRefreshFetcherMock{
		projectsWithSimulation: []athenacontract.AthenaProjectWithSimulationState{
			{
				Project: athenacontract.AthenaProject{
					TokenContract: contract,
					Token:         athenacontract.AthenaToken{Symbol: "NEW"},
				},
			},
		},
	}
	service := &Service{projectCache: cache}

	if err := service.refreshActiveProjectStates(context.Background(), fetcher); err != nil {
		t.Fatalf("refresh active states: %v", err)
	}
	if len(cache.setCalls) != 1 {
		t.Fatalf("set calls = %d, want 1", len(cache.setCalls))
	}
	got := cache.setCalls[0]
	if got.Meta.CreatorResult != originalResult {
		t.Fatalf("creator result changed: got %+v want %+v", got.Meta.CreatorResult, originalResult)
	}
	if got.ChainState.Token.Symbol != "NEW" {
		t.Fatalf("chain state symbol = %q, want NEW", got.ChainState.Token.Symbol)
	}
}

func TestRefreshActiveProjectSimulationsUpdatesOnlyCreatorResult(t *testing.T) {
	contract := common.HexToAddress("0x3000000000000000000000000000000000000003")
	creator := common.HexToAddress("0x4000000000000000000000000000000000000004")
	wethPair := common.HexToAddress("0x5000000000000000000000000000000000000005")
	usdtPair := common.HexToAddress("0x6000000000000000000000000000000000000006")
	oldResult := SimulateResult{CanMintViaTransferToUsdtPair: true}
	newResult := SimulateResult{CanMintFromDeadViaTransferFrom: true}

	cache := &activeRefreshCacheMock{
		listActive: []*Project{
			{
				Meta: ProjectMeta{Contract: contract, Creator: creator, CreatorResult: oldResult},
				ChainState: athenacontract.AthenaProject{
					TokenContract: contract,
					Token:         athenacontract.AthenaToken{Symbol: "KEEP"},
					WethPair:      athenacontract.AthenaPair{ContractAddress: wethPair},
					UsdtPair:      athenacontract.AthenaPair{ContractAddress: usdtPair},
				},
			},
		},
		getByKey: map[string]*Project{
			contract.Hex(): {
				Meta: ProjectMeta{Contract: contract, Creator: creator, CreatorResult: oldResult},
				ChainState: athenacontract.AthenaProject{
					TokenContract: contract,
					Token:         athenacontract.AthenaToken{Symbol: "KEEP"},
					WethPair:      athenacontract.AthenaPair{ContractAddress: wethPair},
					UsdtPair:      athenacontract.AthenaPair{ContractAddress: usdtPair},
				},
			},
		},
	}
	fetcher := &activeRefreshFetcherMock{
		states: []athenacontract.AthenaSimulationState{{}},
	}
	simulator := &activeRefreshSimulatorMock{
		results: map[string]SimulateResult{contract.Hex(): newResult},
	}
	service := &Service{projectCache: cache}

	if err := service.refreshActiveProjectSimulations(context.Background(), fetcher, simulator); err != nil {
		t.Fatalf("refresh active simulations: %v", err)
	}
	if len(cache.setCalls) != 1 {
		t.Fatalf("set calls = %d, want 1", len(cache.setCalls))
	}
	got := cache.setCalls[0]
	if got.Meta.CreatorResult != newResult {
		t.Fatalf("creator result = %+v, want %+v", got.Meta.CreatorResult, newResult)
	}
	if got.ChainState.Token.Symbol != "KEEP" {
		t.Fatalf("chain state symbol changed: got %q want KEEP", got.ChainState.Token.Symbol)
	}
}

func TestRefreshActiveProjectSimulationsSkipsMissingPairs(t *testing.T) {
	contract := common.HexToAddress("0x7000000000000000000000000000000000000007")
	creator := common.HexToAddress("0x8000000000000000000000000000000000000008")

	cache := &activeRefreshCacheMock{
		listActive: []*Project{
			{
				Meta:       ProjectMeta{Contract: contract, Creator: creator},
				ChainState: athenacontract.AthenaProject{TokenContract: contract},
			},
		},
		getByKey: map[string]*Project{
			contract.Hex(): {
				Meta:       ProjectMeta{Contract: contract, Creator: creator},
				ChainState: athenacontract.AthenaProject{TokenContract: contract},
			},
		},
	}
	fetcher := &activeRefreshFetcherMock{states: []athenacontract.AthenaSimulationState{{}}}
	simulator := &activeRefreshSimulatorMock{}
	service := &Service{projectCache: cache}

	if err := service.refreshActiveProjectSimulations(context.Background(), fetcher, simulator); err != nil {
		t.Fatalf("refresh active simulations: %v", err)
	}
	if len(simulator.calls) != 0 {
		t.Fatalf("simulator calls = %d, want 0", len(simulator.calls))
	}
	if len(cache.setCalls) != 0 {
		t.Fatalf("set calls = %d, want 0", len(cache.setCalls))
	}
}

func TestRefreshActiveProjectSimulationsContinuesWhenItemFails(t *testing.T) {
	contractA := common.HexToAddress("0x9000000000000000000000000000000000000009")
	contractB := common.HexToAddress("0x1000000000000000000000000000000000000010")
	creatorA := common.HexToAddress("0x1100000000000000000000000000000000000011")
	creatorB := common.HexToAddress("0x1200000000000000000000000000000000000012")
	wethA := common.HexToAddress("0x1300000000000000000000000000000000000013")
	usdtA := common.HexToAddress("0x1400000000000000000000000000000000000014")
	wethB := common.HexToAddress("0x1500000000000000000000000000000000000015")
	usdtB := common.HexToAddress("0x1600000000000000000000000000000000000016")
	successResult := SimulateResult{CanMintFromZeroViaTransferFrom: true}

	cache := &activeRefreshCacheMock{
		listActive: []*Project{
			{
				Meta: ProjectMeta{Contract: contractA, Creator: creatorA},
				ChainState: athenacontract.AthenaProject{
					WethPair: athenacontract.AthenaPair{ContractAddress: wethA},
					UsdtPair: athenacontract.AthenaPair{ContractAddress: usdtA},
				},
			},
			{
				Meta: ProjectMeta{Contract: contractB, Creator: creatorB},
				ChainState: athenacontract.AthenaProject{
					WethPair: athenacontract.AthenaPair{ContractAddress: wethB},
					UsdtPair: athenacontract.AthenaPair{ContractAddress: usdtB},
				},
			},
		},
		getByKey: map[string]*Project{
			contractA.Hex(): {
				Meta: ProjectMeta{Contract: contractA, Creator: creatorA},
				ChainState: athenacontract.AthenaProject{
					WethPair: athenacontract.AthenaPair{ContractAddress: wethA},
					UsdtPair: athenacontract.AthenaPair{ContractAddress: usdtA},
				},
			},
			contractB.Hex(): {
				Meta: ProjectMeta{Contract: contractB, Creator: creatorB},
				ChainState: athenacontract.AthenaProject{
					WethPair: athenacontract.AthenaPair{ContractAddress: wethB},
					UsdtPair: athenacontract.AthenaPair{ContractAddress: usdtB},
				},
			},
		},
	}
	fetcher := &activeRefreshFetcherMock{
		states: []athenacontract.AthenaSimulationState{{}, {}},
	}
	simulator := &activeRefreshSimulatorMock{
		errs: map[string]error{
			contractA.Hex(): errors.New("simulate failed"),
		},
		results: map[string]SimulateResult{
			contractB.Hex(): successResult,
		},
	}
	service := &Service{projectCache: cache}

	if err := service.refreshActiveProjectSimulations(context.Background(), fetcher, simulator); err != nil {
		t.Fatalf("refresh active simulations: %v", err)
	}
	if len(simulator.calls) != 2 {
		t.Fatalf("simulator calls = %d, want 2", len(simulator.calls))
	}
	if len(cache.setCalls) != 1 {
		t.Fatalf("set calls = %d, want 1", len(cache.setCalls))
	}
	if cache.setCalls[0].Meta.Contract != contractB {
		t.Fatalf("updated contract = %s, want %s", cache.setCalls[0].Meta.Contract.Hex(), contractB.Hex())
	}
	if cache.setCalls[0].Meta.CreatorResult != successResult {
		t.Fatalf("creator result = %+v, want %+v", cache.setCalls[0].Meta.CreatorResult, successResult)
	}
}

func TestRefreshActiveProjectStateAndSimulationSkipArchivedOnReRead(t *testing.T) {
	contract := common.HexToAddress("0x1700000000000000000000000000000000000017")
	creator := common.HexToAddress("0x1800000000000000000000000000000000000018")
	weth := common.HexToAddress("0x1900000000000000000000000000000000000019")
	usdt := common.HexToAddress("0x2000000000000000000000000000000000000020")

	stateCache := &activeRefreshCacheMock{
		listActive: []*Project{
			{
				Meta:       ProjectMeta{Contract: contract, Creator: creator},
				ChainState: athenacontract.AthenaProject{TokenContract: contract},
			},
		},
		getByKey: map[string]*Project{
			contract.Hex(): {
				Meta:       ProjectMeta{Contract: contract, Creator: creator, IsArchived: true},
				ChainState: athenacontract.AthenaProject{TokenContract: contract},
			},
		},
	}
	stateFetcher := &activeRefreshFetcherMock{
		projectsWithSimulation: []athenacontract.AthenaProjectWithSimulationState{
			{Project: athenacontract.AthenaProject{TokenContract: contract}},
		},
	}
	service := &Service{projectCache: stateCache}

	if err := service.refreshActiveProjectStates(context.Background(), stateFetcher); err != nil {
		t.Fatalf("refresh active states: %v", err)
	}
	if len(stateCache.setCalls) != 0 {
		t.Fatalf("state set calls = %d, want 0", len(stateCache.setCalls))
	}

	simCache := &activeRefreshCacheMock{
		listActive: []*Project{
			{
				Meta: ProjectMeta{Contract: contract, Creator: creator},
				ChainState: athenacontract.AthenaProject{
					WethPair: athenacontract.AthenaPair{ContractAddress: weth},
					UsdtPair: athenacontract.AthenaPair{ContractAddress: usdt},
				},
			},
		},
		getByKey: map[string]*Project{
			contract.Hex(): {
				Meta: ProjectMeta{Contract: contract, Creator: creator, IsArchived: true},
				ChainState: athenacontract.AthenaProject{
					WethPair: athenacontract.AthenaPair{ContractAddress: weth},
					UsdtPair: athenacontract.AthenaPair{ContractAddress: usdt},
				},
			},
		},
	}
	simFetcher := &activeRefreshFetcherMock{
		states: []athenacontract.AthenaSimulationState{{}},
	}
	simulator := &activeRefreshSimulatorMock{
		results: map[string]SimulateResult{contract.Hex(): {CanMintViaTransferToWethPair: true}},
	}
	service.projectCache = simCache

	if err := service.refreshActiveProjectSimulations(context.Background(), simFetcher, simulator); err != nil {
		t.Fatalf("refresh active simulations: %v", err)
	}
	if len(simCache.setCalls) != 0 {
		t.Fatalf("simulation set calls = %d, want 0", len(simCache.setCalls))
	}
}

func TestRefreshActiveProjectRoundFailsOnSizeMismatch(t *testing.T) {
	contract := common.HexToAddress("0x2100000000000000000000000000000000000021")
	creator := common.HexToAddress("0x2200000000000000000000000000000000000022")
	weth := common.HexToAddress("0x2300000000000000000000000000000000000023")
	usdt := common.HexToAddress("0x2400000000000000000000000000000000000024")

	t.Run("state size mismatch", func(t *testing.T) {
		cache := &activeRefreshCacheMock{
			listActive: []*Project{
				{Meta: ProjectMeta{Contract: contract, Creator: creator}},
			},
			getByKey: map[string]*Project{
				contract.Hex(): {Meta: ProjectMeta{Contract: contract, Creator: creator}},
			},
		}
		fetcher := &activeRefreshFetcherMock{projectsWithSimulation: nil}
		service := &Service{projectCache: cache}

		err := service.refreshActiveProjectStates(context.Background(), fetcher)
		if err == nil {
			t.Fatalf("expected size mismatch error")
		}
		if !strings.Contains(err.Error(), "size mismatch") {
			t.Fatalf("error = %v, want contains size mismatch", err)
		}
	})

	t.Run("simulation size mismatch", func(t *testing.T) {
		cache := &activeRefreshCacheMock{
			listActive: []*Project{
				{
					Meta: ProjectMeta{Contract: contract, Creator: creator},
					ChainState: athenacontract.AthenaProject{
						WethPair: athenacontract.AthenaPair{ContractAddress: weth},
						UsdtPair: athenacontract.AthenaPair{ContractAddress: usdt},
					},
				},
			},
			getByKey: map[string]*Project{
				contract.Hex(): {
					Meta: ProjectMeta{Contract: contract, Creator: creator},
					ChainState: athenacontract.AthenaProject{
						WethPair: athenacontract.AthenaPair{ContractAddress: weth},
						UsdtPair: athenacontract.AthenaPair{ContractAddress: usdt},
					},
				},
			},
		}
		fetcher := &activeRefreshFetcherMock{states: nil}
		simulator := &activeRefreshSimulatorMock{}
		service := &Service{projectCache: cache}

		err := service.refreshActiveProjectSimulations(context.Background(), fetcher, simulator)
		if err == nil {
			t.Fatalf("expected size mismatch error")
		}
		if !strings.Contains(err.Error(), "size mismatch") {
			t.Fatalf("error = %v, want contains size mismatch", err)
		}
	})
}
