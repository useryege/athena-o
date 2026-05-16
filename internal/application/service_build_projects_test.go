package application

import (
	"context"
	"errors"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/useryege/athena/internal/application/sourcecode"
	appstore "github.com/useryege/athena/internal/application/store"
	athenacontract "github.com/useryege/athena/pkg/abi/ATHENA"
)

type buildProjectsBlacklistModelMock struct {
	fields  []string
	listErr error
}

func (m *buildProjectsBlacklistModelMock) Load(context.Context) error {
	return nil
}

func (m *buildProjectsBlacklistModelMock) List(context.Context) ([]string, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	out := make([]string, len(m.fields))
	copy(out, m.fields)
	return out, nil
}

func (m *buildProjectsBlacklistModelMock) Add(context.Context, string) error {
	return nil
}

func (m *buildProjectsBlacklistModelMock) Delete(context.Context, string) error {
	return nil
}

func TestBuildProjectsFromMetasAppliesSourceCodeBlacklistPerMeta(t *testing.T) {
	contractA := common.HexToAddress("0x1000000000000000000000000000000000000001")
	contractB := common.HexToAddress("0x2000000000000000000000000000000000000002")
	creatorA := common.HexToAddress("0x3000000000000000000000000000000000000003")
	creatorB := common.HexToAddress("0x4000000000000000000000000000000000000004")
	wethA := common.HexToAddress("0x5000000000000000000000000000000000000005")
	usdtA := common.HexToAddress("0x6000000000000000000000000000000000000006")
	wethB := common.HexToAddress("0x7000000000000000000000000000000000000007")
	usdtB := common.HexToAddress("0x8000000000000000000000000000000000000008")
	resultA := SimulateResult{CanMintViaTransferToWethPair: true}
	resultB := SimulateResult{CanMintFromZeroViaTransferFrom: true}

	metas := []appstore.ProjectMeta{
		{
			Contract:   contractA,
			Creator:    creatorA,
			SourceCode: "contract T { address public owner; function f() external { owner = msg.sender; } }",
		},
		{
			Contract: contractB,
			Creator:  creatorB,
		},
	}
	fetcher := &activeRefreshFetcherMock{
		projectsWithSimulation: []athenacontract.AthenaProjectWithSimulationState{
			{
				Project: athenacontract.AthenaProject{
					TokenContract: contractA,
					WethPair:      athenacontract.AthenaPair{ContractAddress: wethA},
					UsdtPair:      athenacontract.AthenaPair{ContractAddress: usdtA},
				},
			},
			{
				Project: athenacontract.AthenaProject{
					TokenContract: contractB,
					WethPair:      athenacontract.AthenaPair{ContractAddress: wethB},
					UsdtPair:      athenacontract.AthenaPair{ContractAddress: usdtB},
				},
			},
		},
	}
	simulator := &activeRefreshSimulatorMock{
		results: map[string]SimulateResult{
			contractA.Hex(): resultA,
			contractB.Hex(): resultB,
		},
	}
	service := &Service{
		sourceAnalyzer:  sourcecode.NewAnalyzer(),
		sourceBlacklist: &buildProjectsBlacklistModelMock{fields: []string{"owner"}},
	}

	projects, err := service.buildProjectsFromMetas(context.Background(), metas, fetcher, simulator)
	if err != nil {
		t.Fatalf("build projects from metas: %v", err)
	}
	if len(projects) != 2 {
		t.Fatalf("projects len = %d, want 2", len(projects))
	}

	if projects[0].Meta.CreatorResult != resultA {
		t.Fatalf("project A creator result = %+v, want %+v", projects[0].Meta.CreatorResult, resultA)
	}
	if !projects[0].Meta.SourceCodeBlacklist.HasBlacklistFields {
		t.Fatalf("project A has blacklist fields = false, want true")
	}
	if len(projects[0].Meta.SourceCodeBlacklist.BlacklistFields) != 1 || projects[0].Meta.SourceCodeBlacklist.BlacklistFields[0] != "owner" {
		t.Fatalf("project A blacklist fields = %v, want [owner]", projects[0].Meta.SourceCodeBlacklist.BlacklistFields)
	}
	if projects[0].Meta.SourceCodeBlacklist.ResolvedAt.IsZero() {
		t.Fatalf("project A resolvedAt is zero, want non-zero")
	}

	if projects[1].Meta.CreatorResult != resultB {
		t.Fatalf("project B creator result = %+v, want %+v", projects[1].Meta.CreatorResult, resultB)
	}
	if !projects[1].Meta.SourceCodeBlacklist.ResolvedAt.IsZero() {
		t.Fatalf("project B resolvedAt = %v, want zero", projects[1].Meta.SourceCodeBlacklist.ResolvedAt)
	}
	if len(simulator.calls) != 2 {
		t.Fatalf("simulator calls = %d, want 2", len(simulator.calls))
	}
}

func TestBuildProjectsFromMetasSkipsBlacklistAnalysisWhenFieldsLoadFails(t *testing.T) {
	contract := common.HexToAddress("0x9000000000000000000000000000000000000009")
	creator := common.HexToAddress("0x1000000000000000000000000000000000000010")

	metas := []appstore.ProjectMeta{
		{
			Contract:   contract,
			Creator:    creator,
			SourceCode: "contract T { address owner; }",
		},
	}
	fetcher := &activeRefreshFetcherMock{
		projectsWithSimulation: []athenacontract.AthenaProjectWithSimulationState{
			{
				Project: athenacontract.AthenaProject{
					TokenContract: contract,
				},
			},
		},
	}
	service := &Service{
		sourceAnalyzer:  sourcecode.NewAnalyzer(),
		sourceBlacklist: &buildProjectsBlacklistModelMock{listErr: errors.New("list failed")},
	}

	projects, err := service.buildProjectsFromMetas(context.Background(), metas, fetcher, nil)
	if err != nil {
		t.Fatalf("build projects from metas: %v", err)
	}
	if len(projects) != 1 {
		t.Fatalf("projects len = %d, want 1", len(projects))
	}
	if !projects[0].Meta.SourceCodeBlacklist.ResolvedAt.IsZero() {
		t.Fatalf("resolvedAt = %v, want zero", projects[0].Meta.SourceCodeBlacklist.ResolvedAt)
	}
	if projects[0].Meta.SourceCodeBlacklist.HasBlacklistFields {
		t.Fatalf("has blacklist fields = true, want false")
	}
}
