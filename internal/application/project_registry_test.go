package application

import (
	"context"
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/google/uuid"
	appstore "github.com/useryege/athena/internal/application/store"
	athenacontract "github.com/useryege/athena/pkg/abi/ATHENA"
)

type registryPublisherMock struct {
	saveMetas        []appstore.ProjectMeta
	sourceCodeWrites []struct {
		projectID  uuid.UUID
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

func (m *registryPublisherMock) PublishProjectSourceCodeUpdate(ctx context.Context, projectID uuid.UUID, sourceCode string) error {
	if m.sourceCodeErr != nil {
		return m.sourceCodeErr
	}
	m.sourceCodeWrites = append(m.sourceCodeWrites, struct {
		projectID  uuid.UUID
		sourceCode string
	}{projectID: projectID, sourceCode: sourceCode})
	return nil
}

func (m *registryPublisherMock) PublishProjectArchive(ctx context.Context, projectID uuid.UUID) error {
	return nil
}

func (m *registryPublisherMock) PublishProjectUnarchive(ctx context.Context, projectID uuid.UUID) error {
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

	projectID := uuid.New()
	project := &Project{Meta: ProjectMeta{
		ProjectID:   projectID,
		Contract:    common.HexToAddress("0x1111111111111111111111111111111111111111"),
		Creator:     common.HexToAddress("0x2222222222222222222222222222222222222222"),
		BlockNumber: 10,
		BlockTime:   11,
		TxHash:      common.HexToHash("0x1234"),
		TxIndex:     1,
	}}

	if err := registry.SetProject(context.Background(), projectID, project); err != nil {
		t.Fatalf("set project: %v", err)
	}
	if len(publisher.saveMetas) != 1 {
		t.Fatalf("meta save publishes = %d, want 1", len(publisher.saveMetas))
	}
	stored, ok, err := registry.GetProject(context.Background(), projectID)
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

	projectID := uuid.New()
	project := &Project{Meta: ProjectMeta{
		ProjectID: projectID,
		Contract:  common.HexToAddress("0x1111111111111111111111111111111111111111"),
	}}

	err := registry.SetProject(context.Background(), projectID, project)
	if !errors.Is(err, expectedErr) {
		t.Fatalf("set project err = %v, want %v", err, expectedErr)
	}
	_, ok, getErr := registry.GetProject(context.Background(), projectID)
	if getErr != nil {
		t.Fatalf("get project: %v", getErr)
	}
	if ok {
		t.Fatalf("project should not be present when publish fails")
	}
}

func TestProjectRegistryUpdateProjectMetaStatePublishFailureKeepsOldSourceCode(t *testing.T) {
	publisher := &registryPublisherMock{}
	registry := NewProjectRegistry(publisher)

	projectID := uuid.New()
	project := &Project{Meta: ProjectMeta{
		ProjectID:   projectID,
		Contract:    common.HexToAddress("0x1111111111111111111111111111111111111111"),
		SourceCode:  "old",
		BlockNumber: 1,
	}}
	if err := registry.LoadProject(context.Background(), projectID, project); err != nil {
		t.Fatalf("load project: %v", err)
	}

	expectedErr := errors.New("source update failed")
	publisher.sourceCodeErr = expectedErr
	err := registry.UpdateProjectMetaState(context.Background(), projectID, &ProjectMeta{SourceCode: "new"})
	if !errors.Is(err, expectedErr) {
		t.Fatalf("update err = %v, want %v", err, expectedErr)
	}
	stored, ok, getErr := registry.GetProject(context.Background(), projectID)
	if getErr != nil {
		t.Fatalf("get project: %v", getErr)
	}
	if !ok || stored == nil {
		t.Fatalf("project not found")
	}
	if stored.Meta.SourceCode != "old" {
		t.Fatalf("source code = %q, want %q", stored.Meta.SourceCode, "old")
	}
}

func TestProjectRegistryListProjectContractsSnapshotAlignedAfterRemove(t *testing.T) {
	publisher := &registryPublisherMock{}
	registry := NewProjectRegistry(publisher)

	projectID1 := uuid.New()
	projectID2 := uuid.New()
	projectID3 := uuid.New()
	contract1 := common.HexToAddress("0x1111111111111111111111111111111111111111")
	contract2 := common.HexToAddress("0x2222222222222222222222222222222222222222")
	contract3 := common.HexToAddress("0x3333333333333333333333333333333333333333")
	creator1 := common.HexToAddress("0xa111111111111111111111111111111111111111")
	creator2 := common.HexToAddress("0xa222222222222222222222222222222222222222")
	creator3 := common.HexToAddress("0xa333333333333333333333333333333333333333")

	for _, tc := range []struct {
		id       uuid.UUID
		contract common.Address
		creator  common.Address
	}{
		{id: projectID1, contract: contract1, creator: creator1},
		{id: projectID2, contract: contract2, creator: creator2},
		{id: projectID3, contract: contract3, creator: creator3},
	} {
		if err := registry.LoadProject(context.Background(), tc.id, &Project{
			Meta: ProjectMeta{
				ProjectID: tc.id,
				Contract:  tc.contract,
				Creator:   tc.creator,
			},
		}); err != nil {
			t.Fatalf("load project %s: %v", tc.id, err)
		}
	}

	if err := registry.RemoveProject(context.Background(), projectID2); err != nil {
		t.Fatalf("remove project: %v", err)
	}

	snapshot, err := registry.ListProjectContracts(context.Background())
	if err != nil {
		t.Fatalf("list project contracts: %v", err)
	}
	if len(snapshot.ProjectIDs) != 2 || len(snapshot.ProjectContracts) != 2 || len(snapshot.ProjectQueries) != 2 {
		t.Fatalf(
			"unexpected snapshot lengths: ids=%d contracts=%d queries=%d",
			len(snapshot.ProjectIDs), len(snapshot.ProjectContracts), len(snapshot.ProjectQueries),
		)
	}
	for i, id := range snapshot.ProjectIDs {
		if snapshot.ProjectQueries[i].TokenContract != snapshot.ProjectContracts[i] {
			t.Fatalf("query/contract mismatch at %d: query=%s contract=%s", i, snapshot.ProjectQueries[i].TokenContract, snapshot.ProjectContracts[i])
		}
		stored, ok, getErr := registry.GetProject(context.Background(), id)
		if getErr != nil {
			t.Fatalf("get project %s: %v", id, getErr)
		}
		if !ok || stored == nil {
			t.Fatalf("project %s not found", id)
		}
		if stored.Meta.Contract != snapshot.ProjectContracts[i] {
			t.Fatalf("contract mismatch at %d: meta=%s snapshot=%s", i, stored.Meta.Contract, snapshot.ProjectContracts[i])
		}
		if stored.Meta.Creator != snapshot.ProjectQueries[i].MsgCaller {
			t.Fatalf("creator mismatch at %d: meta=%s query=%s", i, stored.Meta.Creator, snapshot.ProjectQueries[i].MsgCaller)
		}
	}
}

func TestProjectRegistryListProjectContractsReturnsSharedSlices(t *testing.T) {
	publisher := &registryPublisherMock{}
	registry := NewProjectRegistry(publisher)

	projectID := uuid.New()
	contract := common.HexToAddress("0x1111111111111111111111111111111111111111")
	creator := common.HexToAddress("0xa111111111111111111111111111111111111111")
	if err := registry.LoadProject(context.Background(), projectID, &Project{
		Meta: ProjectMeta{
			ProjectID: projectID,
			Contract:  contract,
			Creator:   creator,
		},
	}); err != nil {
		t.Fatalf("load project: %v", err)
	}

	snapshot, err := registry.ListProjectContracts(context.Background())
	if err != nil {
		t.Fatalf("list project contracts: %v", err)
	}

	snapshot.ProjectIDs[0] = uuid.Nil
	snapshot.ProjectContracts[0] = common.HexToAddress("0xffffffffffffffffffffffffffffffffffffffff")
	snapshot.ProjectQueries[0] = athenacontract.AthenaProjectQuery{}

	latest, err := registry.ListProjectContracts(context.Background())
	if err != nil {
		t.Fatalf("list project contracts: %v", err)
	}
	if latest.ProjectIDs[0] != uuid.Nil {
		t.Fatalf("project id not shared: got %s want %s", latest.ProjectIDs[0], uuid.Nil)
	}
	if latest.ProjectContracts[0] != common.HexToAddress("0xffffffffffffffffffffffffffffffffffffffff") {
		t.Fatalf("contract slice not shared: got %s", latest.ProjectContracts[0])
	}
	if latest.ProjectQueries[0] != (athenacontract.AthenaProjectQuery{}) {
		t.Fatalf("query slice not shared: got token=%s caller=%s", latest.ProjectQueries[0].TokenContract, latest.ProjectQueries[0].MsgCaller)
	}
}

func TestProjectRegistryGetProjectReturnsSharedReference(t *testing.T) {
	publisher := &registryPublisherMock{}
	registry := NewProjectRegistry(publisher)

	projectID := uuid.New()
	project := &Project{Meta: ProjectMeta{
		ProjectID:  projectID,
		Contract:   common.HexToAddress("0x1111111111111111111111111111111111111111"),
		SourceCode: "old",
	}}
	if err := registry.LoadProject(context.Background(), projectID, project); err != nil {
		t.Fatalf("load project: %v", err)
	}

	stored, ok, err := registry.GetProject(context.Background(), projectID)
	if err != nil {
		t.Fatalf("get project: %v", err)
	}
	if !ok || stored == nil {
		t.Fatalf("project not found")
	}

	stored.Meta.SourceCode = "new"

	latest, ok, err := registry.GetProject(context.Background(), projectID)
	if err != nil {
		t.Fatalf("get project again: %v", err)
	}
	if !ok || latest == nil {
		t.Fatalf("project not found on second get")
	}
	if latest.Meta.SourceCode != "new" {
		t.Fatalf("shared reference not observed: got %q want %q", latest.Meta.SourceCode, "new")
	}
}

func TestProjectRegistryListProjectsReturnsSharedReferences(t *testing.T) {
	publisher := &registryPublisherMock{}
	registry := NewProjectRegistry(publisher)

	projectID := uuid.New()
	project := &Project{Meta: ProjectMeta{
		ProjectID:  projectID,
		Contract:   common.HexToAddress("0x1111111111111111111111111111111111111111"),
		SourceCode: "old",
	}}
	if err := registry.LoadProject(context.Background(), projectID, project); err != nil {
		t.Fatalf("load project: %v", err)
	}

	projects, err := registry.ListProjects(context.Background())
	if err != nil {
		t.Fatalf("list projects: %v", err)
	}
	if len(projects) != 1 || projects[0] == nil {
		t.Fatalf("unexpected projects list")
	}

	projects[0].Meta.SourceCode = "new"

	latest, ok, err := registry.GetProject(context.Background(), projectID)
	if err != nil {
		t.Fatalf("get project: %v", err)
	}
	if !ok || latest == nil {
		t.Fatalf("project not found after list update")
	}
	if latest.Meta.SourceCode != "new" {
		t.Fatalf("shared list reference not observed: got %q want %q", latest.Meta.SourceCode, "new")
	}
}

func TestProjectRegistryUpdateProjectChainStatesRetainsInputReferences(t *testing.T) {
	publisher := &registryPublisherMock{}
	registry := NewProjectRegistry(publisher)

	projectID := uuid.New()
	project := &Project{Meta: ProjectMeta{
		ProjectID: projectID,
		Contract:  common.HexToAddress("0x1111111111111111111111111111111111111111"),
	}}
	if err := registry.LoadProject(context.Background(), projectID, project); err != nil {
		t.Fatalf("load project: %v", err)
	}

	totalSupply := big.NewInt(100)
	state := athenacontract.AthenaProject{}
	state.Token.TotalSupply = totalSupply
	if err := registry.UpdateProjectChainStates(context.Background(), map[uuid.UUID]athenacontract.AthenaProject{
		projectID: state,
	}); err != nil {
		t.Fatalf("update chain states: %v", err)
	}

	totalSupply.SetInt64(200)
	latest, ok, err := registry.GetProject(context.Background(), projectID)
	if err != nil {
		t.Fatalf("get project: %v", err)
	}
	if !ok || latest == nil {
		t.Fatalf("project not found")
	}
	if latest.ChainState.Token.TotalSupply == nil || latest.ChainState.Token.TotalSupply.Int64() != 200 {
		t.Fatalf("chain state reference not retained")
	}
}
