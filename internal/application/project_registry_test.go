package application

import (
	"context"
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	appstore "github.com/useryege/athena/internal/application/store"
	athenacontract "github.com/useryege/athena/pkg/abi/ATHENA"
)

type registryPublisherMock struct {
	saveMetas        []appstore.ProjectMeta
	sourceCodeWrites []struct {
		contract   common.Address
		sourceCode string
	}
	saveErr       error
	sourceCodeErr error
}

func (m *registryPublisherMock) Publish(ctx context.Context, event PersistenceEvent) error {
	return nil
}

func (m *registryPublisherMock) PublishProjectMetaSave(ctx context.Context, meta appstore.ProjectMeta) error {
	if m.saveErr != nil {
		return m.saveErr
	}
	m.saveMetas = append(m.saveMetas, meta)
	return nil
}

func (m *registryPublisherMock) PublishProjectSourceCodeUpdate(ctx context.Context, contract common.Address, sourceCode string) error {
	if m.sourceCodeErr != nil {
		return m.sourceCodeErr
	}
	m.sourceCodeWrites = append(m.sourceCodeWrites, struct {
		contract   common.Address
		sourceCode string
	}{contract: contract, sourceCode: sourceCode})
	return nil
}

func (m *registryPublisherMock) PublishProjectArchive(ctx context.Context, contract common.Address) error {
	return nil
}

func (m *registryPublisherMock) PublishProjectUnarchive(ctx context.Context, contract common.Address) error {
	return nil
}

func (m *registryPublisherMock) PublishSourceCodeBlacklistAdd(ctx context.Context, field string) error {
	return nil
}

func (m *registryPublisherMock) PublishSourceCodeBlacklistDelete(ctx context.Context, field string) error {
	return nil
}

func TestProjectRegistrySetProjectPublishesMetaSave(t *testing.T) {
	publisher := &registryPublisherMock{}
	registry := NewProjectRegistry(publisher)

	contract := common.HexToAddress("0x1111111111111111111111111111111111111111")
	project := &Project{Meta: ProjectMeta{
		Contract:    contract,
		Creator:     common.HexToAddress("0x2222222222222222222222222222222222222222"),
		BlockNumber: 10,
		BlockTime:   11,
		TxHash:      common.HexToHash("0x1234"),
		TxIndex:     1,
	}}

	if err := registry.SetProject(context.Background(), project); err != nil {
		t.Fatalf("set project: %v", err)
	}
	if len(publisher.saveMetas) != 1 {
		t.Fatalf("meta save publishes = %d, want 1", len(publisher.saveMetas))
	}
	stored, ok, err := registry.GetProject(context.Background(), contract)
	if err != nil {
		t.Fatalf("get project: %v", err)
	}
	if !ok || stored == nil {
		t.Fatalf("project not found after set")
	}
}

func TestProjectRegistrySetProjectPublishFailureDoesNotMutateRegistry(t *testing.T) {
	expectedErr := errors.New("publish failed")
	publisher := &registryPublisherMock{saveErr: expectedErr}
	registry := NewProjectRegistry(publisher)

	contract := common.HexToAddress("0x1111111111111111111111111111111111111111")
	project := &Project{Meta: ProjectMeta{Contract: contract}}

	err := registry.SetProject(context.Background(), project)
	if !errors.Is(err, expectedErr) {
		t.Fatalf("set project err = %v, want %v", err, expectedErr)
	}
	_, ok, getErr := registry.GetProject(context.Background(), contract)
	if getErr != nil {
		t.Fatalf("get project: %v", getErr)
	}
	if ok {
		t.Fatalf("project should not be present when publish fails")
	}
}

func TestProjectRegistryUpdateProjectMetaState(t *testing.T) {
	publisher := &registryPublisherMock{}
	registry := NewProjectRegistry(publisher)

	contract := common.HexToAddress("0x1111111111111111111111111111111111111111")
	project := &Project{Meta: ProjectMeta{
		Contract:   contract,
		SourceCode: "old",
	}}
	if err := registry.LoadProject(context.Background(), project); err != nil {
		t.Fatalf("load project: %v", err)
	}

	err := registry.UpdateProjectMetaState(context.Background(), contract, &ProjectMeta{SourceCode: "new"})
	if err != nil {
		t.Fatalf("update meta state: %v", err)
	}
	if len(publisher.sourceCodeWrites) != 1 {
		t.Fatalf("source code publishes = %d, want 1", len(publisher.sourceCodeWrites))
	}
	if publisher.sourceCodeWrites[0].contract != contract {
		t.Fatalf("published contract = %s, want %s", publisher.sourceCodeWrites[0].contract, contract)
	}
	stored, ok, getErr := registry.GetProject(context.Background(), contract)
	if getErr != nil {
		t.Fatalf("get project: %v", getErr)
	}
	if !ok || stored == nil || stored.Meta.SourceCode != "new" {
		t.Fatalf("source code = %q, want %q", stored.Meta.SourceCode, "new")
	}
}

func TestProjectRegistryRemoveProjectUpdatesContractSnapshot(t *testing.T) {
	publisher := &registryPublisherMock{}
	registry := NewProjectRegistry(publisher)

	contract1 := common.HexToAddress("0x00000000000000000000000000000000000000A1")
	contract2 := common.HexToAddress("0x00000000000000000000000000000000000000A2")
	contract3 := common.HexToAddress("0x00000000000000000000000000000000000000A3")
	creator := common.HexToAddress("0x00000000000000000000000000000000000000B1")

	for _, contract := range []common.Address{contract1, contract2, contract3} {
		if err := registry.LoadProject(context.Background(), &Project{
			Meta: ProjectMeta{Contract: contract, Creator: creator},
		}); err != nil {
			t.Fatalf("load project(%s): %v", contract, err)
		}
	}

	if err := registry.RemoveProject(context.Background(), contract2); err != nil {
		t.Fatalf("remove project: %v", err)
	}
	snapshot, err := registry.ListProjectContracts(context.Background())
	if err != nil {
		t.Fatalf("list project contracts: %v", err)
	}
	if len(snapshot.ProjectContracts) != 2 || len(snapshot.ProjectQueries) != 2 {
		t.Fatalf("snapshot sizes = contracts:%d queries:%d, want 2 each", len(snapshot.ProjectContracts), len(snapshot.ProjectQueries))
	}
	for _, contract := range snapshot.ProjectContracts {
		if contract == contract2 {
			t.Fatalf("removed contract still exists in snapshot")
		}
	}
}

func TestProjectRegistryUpdateProjectChainStates(t *testing.T) {
	publisher := &registryPublisherMock{}
	registry := NewProjectRegistry(publisher)

	contract := common.HexToAddress("0x1111111111111111111111111111111111111111")
	project := &Project{Meta: ProjectMeta{Contract: contract}}
	if err := registry.LoadProject(context.Background(), project); err != nil {
		t.Fatalf("load project: %v", err)
	}

	state := athenacontract.AthenaProject{
		Token: athenacontract.AthenaToken{
			Name:        "Token",
			Symbol:      "TKN",
			Decimals:    18,
			TotalSupply: big.NewInt(123),
		},
	}
	if err := registry.UpdateProjectChainStates(context.Background(), map[common.Address]athenacontract.AthenaProject{
		contract: state,
	}); err != nil {
		t.Fatalf("update chain states: %v", err)
	}

	latest, ok, err := registry.GetProject(context.Background(), contract)
	if err != nil {
		t.Fatalf("get project: %v", err)
	}
	if !ok || latest == nil {
		t.Fatalf("project not found")
	}
	if latest.ChainState.Token.Symbol != "TKN" {
		t.Fatalf("symbol = %q, want %q", latest.ChainState.Token.Symbol, "TKN")
	}
}
