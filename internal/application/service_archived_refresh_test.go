package application

import (
	"context"
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/useryege/athena/internal/application/sourcecode"
	appstore "github.com/useryege/athena/internal/application/store"
	athenacontract "github.com/useryege/athena/pkg/abi/ATHENA"
	"github.com/useryege/athena/util/ethereumapi"
)

type archivedRefreshAPIMock struct {
	sourceByContract map[string]string
}

func (m *archivedRefreshAPIMock) GetSourceCode(_ context.Context, contractAddress string) (*ethereumapi.SourceCodeResponse, error) {
	sourceCode, ok := m.sourceByContract[contractAddress]
	if !ok {
		return nil, errors.New("source code not found")
	}
	response := &ethereumapi.SourceCodeResponse{
		Status:  "1",
		Message: "OK",
		Result: []struct {
			SourceCode           string `json:"SourceCode"`
			ABI                  string `json:"ABI"`
			ContractName         string `json:"ContractName"`
			CompilerVersion      string `json:"CompilerVersion"`
			OptimizationUsed     string `json:"OptimizationUsed"`
			Runs                 string `json:"Runs"`
			ConstructorArguments string `json:"ConstructorArguments"`
			EVMVersion           string `json:"EVMVersion"`
			Library              string `json:"Library"`
			LicenseType          string `json:"LicenseType"`
			Proxy                string `json:"Proxy"`
			Implementation       string `json:"Implementation"`
			SwarmSource          string `json:"SwarmSource"`
		}{
			{SourceCode: sourceCode},
		},
	}
	return response, nil
}

func (m *archivedRefreshAPIMock) GetABI(context.Context, string) (*ethereumapi.ABIResponse, error) {
	return &ethereumapi.ABIResponse{Status: "1", Message: "OK", Result: "[]"}, nil
}

type archivedSourceUpdate struct {
	contract   common.Address
	sourceCode string
}

type archivedRefreshPublisherMock struct {
	sourceUpdates []archivedSourceUpdate
}

func (m *archivedRefreshPublisherMock) Publish(context.Context, PersistenceEvent) error {
	return nil
}

func (m *archivedRefreshPublisherMock) PublishProjectMetaSave(context.Context, appstore.ProjectMeta) error {
	return nil
}

func (m *archivedRefreshPublisherMock) PublishProjectEventLog(context.Context, appstore.ProjectEventLog) error {
	return nil
}

func (m *archivedRefreshPublisherMock) PublishProjectSourceCodeUpdate(_ context.Context, contract common.Address, sourceCode string) error {
	m.sourceUpdates = append(m.sourceUpdates, archivedSourceUpdate{contract: contract, sourceCode: sourceCode})
	return nil
}

func (m *archivedRefreshPublisherMock) PublishProjectArchive(context.Context, common.Address) error {
	return nil
}

func (m *archivedRefreshPublisherMock) PublishProjectUnarchive(context.Context, common.Address) error {
	return nil
}

func (m *archivedRefreshPublisherMock) PublishSourceCodeBlacklistAdd(context.Context, string) error {
	return nil
}

func (m *archivedRefreshPublisherMock) PublishSourceCodeBlacklistDelete(context.Context, string) error {
	return nil
}

func TestMatchesTarget(t *testing.T) {
	if !matchesTarget(false, refreshTargetActive) {
		t.Fatalf("active target should match non-archived item")
	}
	if matchesTarget(true, refreshTargetActive) {
		t.Fatalf("active target should not match archived item")
	}
	if !matchesTarget(true, refreshTargetArchived) {
		t.Fatalf("archived target should match archived item")
	}
	if matchesTarget(false, refreshTargetArchived) {
		t.Fatalf("archived target should not match non-archived item")
	}
}

func TestBuildProjectQueriesSkipsNilAndKeepsAlignment(t *testing.T) {
	contractA := common.HexToAddress("0x0100000000000000000000000000000000000001")
	contractB := common.HexToAddress("0x0200000000000000000000000000000000000002")
	creatorA := common.HexToAddress("0x0300000000000000000000000000000000000003")
	creatorB := common.HexToAddress("0x0400000000000000000000000000000000000004")

	queries, contracts := buildProjectQueries([]*Project{
		{Meta: ProjectMeta{Contract: contractA, Creator: creatorA}},
		nil,
		{Meta: ProjectMeta{Contract: contractB, Creator: creatorB}},
	})

	if len(queries) != 2 || len(contracts) != 2 {
		t.Fatalf("lens = (%d,%d), want (2,2)", len(queries), len(contracts))
	}
	if queries[0].TokenContract != contractA || queries[0].MsgCaller != creatorA || contracts[0] != contractA {
		t.Fatalf("index 0 alignment mismatch: query=%+v contract=%s", queries[0], contracts[0].Hex())
	}
	if queries[1].TokenContract != contractB || queries[1].MsgCaller != creatorB || contracts[1] != contractB {
		t.Fatalf("index 1 alignment mismatch: query=%+v contract=%s", queries[1], contracts[1].Hex())
	}
}

func TestListProjectsByTarget(t *testing.T) {
	activeContract := common.HexToAddress("0x1111000000000000000000000000000000000001")
	archivedContract := common.HexToAddress("0x2222000000000000000000000000000000000002")
	cache := &activeRefreshCacheMock{
		listActive: []*Project{
			{Meta: ProjectMeta{Contract: activeContract}},
		},
		listArchived: []*Project{
			{Meta: ProjectMeta{Contract: archivedContract, IsArchived: true}},
		},
	}
	service := &Service{projectCache: cache}

	active, err := service.listProjectsByTarget(context.Background(), refreshTargetActive)
	if err != nil {
		t.Fatalf("list active by target: %v", err)
	}
	if len(active) != 1 || active[0].Meta.Contract != activeContract {
		t.Fatalf("active projects = %+v, want one contract %s", active, activeContract.Hex())
	}

	archived, err := service.listProjectsByTarget(context.Background(), refreshTargetArchived)
	if err != nil {
		t.Fatalf("list archived by target: %v", err)
	}
	if len(archived) != 1 || archived[0].Meta.Contract != archivedContract {
		t.Fatalf("archived projects = %+v, want one contract %s", archived, archivedContract.Hex())
	}
	if len(cache.listArchivedCalls) == 0 {
		t.Fatalf("expected archived pagination calls for archived target")
	}
}

func TestListAllArchivedProjectsPaginatesAll(t *testing.T) {
	const total = 401
	archived := make([]*Project, 0, total)
	for i := 0; i < total; i++ {
		contract := common.BigToAddress(big.NewInt(int64(i + 1)))
		creator := common.BigToAddress(big.NewInt(int64(i + 10_000)))
		archived = append(archived, &Project{
			Meta: ProjectMeta{
				Contract:   contract,
				Creator:    creator,
				IsArchived: true,
			},
		})
	}

	cache := &activeRefreshCacheMock{listArchived: archived}
	service := &Service{projectCache: cache}

	projects, err := service.listAllArchivedProjects(context.Background())
	if err != nil {
		t.Fatalf("list all archived projects: %v", err)
	}
	if len(projects) != total {
		t.Fatalf("projects len = %d, want %d", len(projects), total)
	}
	if len(cache.listArchivedCalls) != 3 {
		t.Fatalf("listArchived calls = %d, want 3", len(cache.listArchivedCalls))
	}
	if cache.listArchivedCalls[0] != [2]int32{1, 200} || cache.listArchivedCalls[1] != [2]int32{2, 200} || cache.listArchivedCalls[2] != [2]int32{3, 200} {
		t.Fatalf("listArchived calls = %v, want [[1 200] [2 200] [3 200]]", cache.listArchivedCalls)
	}
}

func TestRefreshArchivedProjectStatesUpdatesOnlyArchivedOnReRead(t *testing.T) {
	contractA := common.HexToAddress("0x1000000000000000000000000000000000000001")
	contractB := common.HexToAddress("0x2000000000000000000000000000000000000002")
	creatorA := common.HexToAddress("0x3000000000000000000000000000000000000003")
	creatorB := common.HexToAddress("0x4000000000000000000000000000000000000004")

	cache := &activeRefreshCacheMock{
		listArchived: []*Project{
			{Meta: ProjectMeta{Contract: contractA, Creator: creatorA, IsArchived: true}},
			{Meta: ProjectMeta{Contract: contractB, Creator: creatorB, IsArchived: true}},
		},
		getByKey: map[string]*Project{
			contractA.Hex(): {
				Meta: ProjectMeta{Contract: contractA, Creator: creatorA, IsArchived: true},
				ChainState: athenacontract.AthenaProject{
					TokenContract: contractA,
					Token:         athenacontract.AthenaToken{Symbol: "OLD_A"},
				},
			},
			contractB.Hex(): {
				Meta: ProjectMeta{Contract: contractB, Creator: creatorB, IsArchived: false},
				ChainState: athenacontract.AthenaProject{
					TokenContract: contractB,
					Token:         athenacontract.AthenaToken{Symbol: "OLD_B"},
				},
			},
		},
	}
	fetcher := &activeRefreshFetcherMock{
		projectsWithSimulation: []athenacontract.AthenaProjectWithSimulationState{
			{Project: athenacontract.AthenaProject{TokenContract: contractA, Token: athenacontract.AthenaToken{Symbol: "NEW_A"}}},
			{Project: athenacontract.AthenaProject{TokenContract: contractB, Token: athenacontract.AthenaToken{Symbol: "NEW_B"}}},
		},
	}
	service := &Service{projectCache: cache}

	if err := service.refreshArchivedProjectStates(context.Background(), fetcher); err != nil {
		t.Fatalf("refresh archived states: %v", err)
	}
	if len(cache.setCalls) != 1 {
		t.Fatalf("set calls = %d, want 1", len(cache.setCalls))
	}
	if cache.setCalls[0].Meta.Contract != contractA {
		t.Fatalf("updated contract = %s, want %s", cache.setCalls[0].Meta.Contract.Hex(), contractA.Hex())
	}
	if cache.setCalls[0].ChainState.Token.Symbol != "NEW_A" {
		t.Fatalf("updated symbol = %q, want NEW_A", cache.setCalls[0].ChainState.Token.Symbol)
	}
}

func TestRefreshArchivedProjectSimulationsUpdatesOnlyArchivedOnReRead(t *testing.T) {
	contractA := common.HexToAddress("0x5000000000000000000000000000000000000005")
	contractB := common.HexToAddress("0x6000000000000000000000000000000000000006")
	creatorA := common.HexToAddress("0x7000000000000000000000000000000000000007")
	creatorB := common.HexToAddress("0x8000000000000000000000000000000000000008")
	wethA := common.HexToAddress("0x9000000000000000000000000000000000000009")
	usdtA := common.HexToAddress("0x1000000000000000000000000000000000000010")
	wethB := common.HexToAddress("0x1100000000000000000000000000000000000011")
	usdtB := common.HexToAddress("0x1200000000000000000000000000000000000012")
	newResult := SimulateResult{CanMintFromDeadViaTransferFrom: true}

	cache := &activeRefreshCacheMock{
		listArchived: []*Project{
			{
				Meta: ProjectMeta{Contract: contractA, Creator: creatorA, IsArchived: true},
				ChainState: athenacontract.AthenaProject{
					WethPair: athenacontract.AthenaPair{ContractAddress: wethA},
					UsdtPair: athenacontract.AthenaPair{ContractAddress: usdtA},
				},
			},
			{
				Meta: ProjectMeta{Contract: contractB, Creator: creatorB, IsArchived: true},
				ChainState: athenacontract.AthenaProject{
					WethPair: athenacontract.AthenaPair{ContractAddress: wethB},
					UsdtPair: athenacontract.AthenaPair{ContractAddress: usdtB},
				},
			},
		},
		getByKey: map[string]*Project{
			contractA.Hex(): {
				Meta: ProjectMeta{Contract: contractA, Creator: creatorA, IsArchived: true},
				ChainState: athenacontract.AthenaProject{
					WethPair: athenacontract.AthenaPair{ContractAddress: wethA},
					UsdtPair: athenacontract.AthenaPair{ContractAddress: usdtA},
				},
			},
			contractB.Hex(): {
				Meta: ProjectMeta{Contract: contractB, Creator: creatorB, IsArchived: false},
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
		results: map[string]SimulateResult{
			contractA.Hex(): newResult,
		},
	}
	service := &Service{projectCache: cache}

	if err := service.refreshArchivedProjectSimulations(context.Background(), fetcher, simulator); err != nil {
		t.Fatalf("refresh archived simulations: %v", err)
	}
	if len(cache.setCalls) != 1 {
		t.Fatalf("set calls = %d, want 1", len(cache.setCalls))
	}
	if cache.setCalls[0].Meta.Contract != contractA {
		t.Fatalf("updated contract = %s, want %s", cache.setCalls[0].Meta.Contract.Hex(), contractA.Hex())
	}
	if cache.setCalls[0].Meta.CreatorResult != newResult {
		t.Fatalf("creator result = %+v, want %+v", cache.setCalls[0].Meta.CreatorResult, newResult)
	}
	if len(simulator.calls) != 1 {
		t.Fatalf("simulator calls = %d, want 1", len(simulator.calls))
	}
}

func TestRefreshArchivedProjectSourceCodesUpdatesAndPersistsOnlyArchived(t *testing.T) {
	contractA := common.HexToAddress("0x1300000000000000000000000000000000000013")
	contractB := common.HexToAddress("0x1400000000000000000000000000000000000014")
	creatorA := common.HexToAddress("0x1500000000000000000000000000000000000015")
	creatorB := common.HexToAddress("0x1600000000000000000000000000000000000016")
	sourceA := "contract T { address public owner; }"
	sourceB := "contract U { address public owner; }"

	cache := &activeRefreshCacheMock{
		listArchived: []*Project{
			{Meta: ProjectMeta{Contract: contractA, Creator: creatorA, IsArchived: true}},
			{Meta: ProjectMeta{Contract: contractB, Creator: creatorB, IsArchived: true}},
		},
		getByKey: map[string]*Project{
			contractA.Hex(): {Meta: ProjectMeta{Contract: contractA, Creator: creatorA, IsArchived: true}},
			contractB.Hex(): {Meta: ProjectMeta{Contract: contractB, Creator: creatorB, IsArchived: false}},
		},
	}
	api := &archivedRefreshAPIMock{
		sourceByContract: map[string]string{
			contractA.String(): sourceA,
			contractB.String(): sourceB,
		},
	}
	publisher := &archivedRefreshPublisherMock{}
	service := &Service{
		projectCache:         cache,
		apiFetcher:           api,
		sourceAnalyzer:       sourcecode.NewAnalyzer(),
		sourceBlacklist:      &buildProjectsBlacklistModelMock{fields: []string{"owner"}},
		persistencePublisher: publisher,
	}

	if err := service.refreshArchivedProjectSourceCodes(context.Background()); err != nil {
		t.Fatalf("refresh archived source codes: %v", err)
	}
	if len(cache.setCalls) != 1 {
		t.Fatalf("set calls = %d, want 1", len(cache.setCalls))
	}
	updated := cache.setCalls[0]
	if updated.Meta.Contract != contractA {
		t.Fatalf("updated contract = %s, want %s", updated.Meta.Contract.Hex(), contractA.Hex())
	}
	if updated.Meta.SourceCode != sourceA {
		t.Fatalf("source code = %q, want %q", updated.Meta.SourceCode, sourceA)
	}
	if !updated.Meta.SourceCodeBlacklist.HasBlacklistFields {
		t.Fatalf("has blacklist fields = false, want true")
	}
	if len(updated.Meta.SourceCodeBlacklist.BlacklistFields) != 1 || updated.Meta.SourceCodeBlacklist.BlacklistFields[0] != "owner" {
		t.Fatalf("blacklist fields = %v, want [owner]", updated.Meta.SourceCodeBlacklist.BlacklistFields)
	}
	if len(publisher.sourceUpdates) != 1 {
		t.Fatalf("source update calls = %d, want 1", len(publisher.sourceUpdates))
	}
	if publisher.sourceUpdates[0].contract != contractA || publisher.sourceUpdates[0].sourceCode != sourceA {
		t.Fatalf("source update = %+v, want contract=%s source=%q", publisher.sourceUpdates[0], contractA.Hex(), sourceA)
	}
}
