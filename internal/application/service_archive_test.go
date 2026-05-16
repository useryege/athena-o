package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	applicationpkg "github.com/useryege/athena/internal/application/apiclient"
	appstore "github.com/useryege/athena/internal/application/store"
)

type archiveProjectStoreMock struct {
	meta       *appstore.ProjectMeta
	getMetaErr error
}

func (m *archiveProjectStoreMock) SaveProjectMeta(ctx context.Context, meta appstore.ProjectMeta) error {
	return nil
}

func (m *archiveProjectStoreMock) ListProjectMetas(ctx context.Context) ([]appstore.ProjectMeta, error) {
	return nil, nil
}

func (m *archiveProjectStoreMock) ListAllProjectMetas(ctx context.Context) ([]appstore.ProjectMeta, error) {
	return nil, nil
}

func (m *archiveProjectStoreMock) UpdateProjectSourceCode(ctx context.Context, contract common.Address, sourceCode string) error {
	return nil
}

func (m *archiveProjectStoreMock) ArchiveProjectByContract(ctx context.Context, contract common.Address) error {
	return nil
}

func (m *archiveProjectStoreMock) UnarchiveProjectByContract(ctx context.Context, contract common.Address) error {
	return nil
}

func (m *archiveProjectStoreMock) ListArchivedProjectMetas(ctx context.Context, page int32, pageSize int32) ([]appstore.ProjectMeta, int64, int32, int32, error) {
	return nil, 0, 0, 0, nil
}

func (m *archiveProjectStoreMock) GetArchivedProjectMetaByContract(ctx context.Context, contract common.Address) (*appstore.ProjectMeta, error) {
	return nil, nil
}

func (m *archiveProjectStoreMock) GetProjectMetaByContract(ctx context.Context, contract common.Address) (*appstore.ProjectMeta, error) {
	if m.getMetaErr != nil {
		return nil, m.getMetaErr
	}
	return m.meta, nil
}

func (m *archiveProjectStoreMock) ListSourceCodeBlacklistFields(ctx context.Context) ([]string, error) {
	return nil, nil
}

func (m *archiveProjectStoreMock) AddSourceCodeBlacklistField(ctx context.Context, field string) error {
	return nil
}

func (m *archiveProjectStoreMock) DeleteSourceCodeBlacklistField(ctx context.Context, field string) error {
	return nil
}

type archiveProjectPublisherMock struct {
	archiveErr error
	archived   []common.Address
}

func (m *archiveProjectPublisherMock) Publish(ctx context.Context, event PersistenceEvent) error {
	return nil
}

func (m *archiveProjectPublisherMock) PublishProjectMetaSave(ctx context.Context, meta appstore.ProjectMeta) error {
	return nil
}

func (m *archiveProjectPublisherMock) PublishProjectSourceCodeUpdate(ctx context.Context, contract common.Address, sourceCode string) error {
	return nil
}

func (m *archiveProjectPublisherMock) PublishProjectArchive(ctx context.Context, contract common.Address) error {
	if m.archiveErr != nil {
		return m.archiveErr
	}
	m.archived = append(m.archived, contract)
	return nil
}

func (m *archiveProjectPublisherMock) PublishProjectUnarchive(ctx context.Context, contract common.Address) error {
	return nil
}

func (m *archiveProjectPublisherMock) PublishSourceCodeBlacklistAdd(ctx context.Context, field string) error {
	return nil
}

func (m *archiveProjectPublisherMock) PublishSourceCodeBlacklistDelete(ctx context.Context, field string) error {
	return nil
}

type archiveProjectCacheMock struct {
	getProject *Project
	getOK      bool
	getErr     error

	setErr   error
	setCalls int
	lastSet  *Project
}

func (m *archiveProjectCacheMock) ReplaceAll(ctx context.Context, projects []*Project) error {
	return nil
}

func (m *archiveProjectCacheMock) SetProject(ctx context.Context, project *Project) error {
	if m.setErr != nil {
		return m.setErr
	}
	m.setCalls++
	m.lastSet = project
	return nil
}

func (m *archiveProjectCacheMock) DeleteProject(ctx context.Context, contract common.Address) error {
	return nil
}

func (m *archiveProjectCacheMock) GetProject(ctx context.Context, contract common.Address) (*Project, bool, error) {
	if m.getErr != nil {
		return nil, false, m.getErr
	}
	return m.getProject, m.getOK, nil
}

func (m *archiveProjectCacheMock) GetMaxProjectBlockNumber(ctx context.Context) (uint64, bool, error) {
	return 0, false, nil
}

func (m *archiveProjectCacheMock) ListActiveProjects(ctx context.Context) ([]*Project, error) {
	return nil, nil
}

func (m *archiveProjectCacheMock) ListArchivedProjects(ctx context.Context, page int32, pageSize int32) ([]*Project, int64, int32, int32, error) {
	return nil, 0, 0, 0, nil
}

func TestArchiveProjectCacheHit(t *testing.T) {
	contract := common.HexToAddress("0x1000000000000000000000000000000000000001")
	meta := &appstore.ProjectMeta{
		Contract:    contract,
		Creator:     common.HexToAddress("0x2000000000000000000000000000000000000002"),
		BlockNumber: 10,
	}
	project := &Project{Meta: projectMetaFromStore(*meta)}
	store := &archiveProjectStoreMock{meta: meta}
	publisher := &archiveProjectPublisherMock{}
	cache := &archiveProjectCacheMock{getProject: project, getOK: true}
	service := &Service{
		store:                store,
		persistencePublisher: publisher,
		projectCache:         cache,
	}

	_, err := service.ArchiveProject(context.Background(), &applicationpkg.ArchiveProjectRequest{Contract: contract.Hex()})
	if err != nil {
		t.Fatalf("archive project: %v", err)
	}
	if len(publisher.archived) != 1 || publisher.archived[0] != contract {
		t.Fatalf("archive publish contracts = %v, want [%s]", publisher.archived, contract.Hex())
	}
	if cache.setCalls != 1 {
		t.Fatalf("set project calls = %d, want 1", cache.setCalls)
	}
	if cache.lastSet == nil {
		t.Fatalf("set project received nil project")
	}
	if !cache.lastSet.Meta.IsArchived {
		t.Fatalf("is archived = false, want true")
	}
	if cache.lastSet.Meta.ArchivedAt.IsZero() {
		t.Fatalf("archived at is zero")
	}
	if cache.lastSet.Meta.ArchivedAt.Location() != time.UTC {
		t.Fatalf("archived at location = %s, want UTC", cache.lastSet.Meta.ArchivedAt.Location())
	}
}

func TestArchiveProjectCacheMissBuildsFromStoreMeta(t *testing.T) {
	contract := common.HexToAddress("0x5000000000000000000000000000000000000005")
	creator := common.HexToAddress("0x6000000000000000000000000000000000000006")
	meta := &appstore.ProjectMeta{
		Contract:    contract,
		Creator:     creator,
		BlockNumber: 33,
		BlockTime:   44,
		TxIndex:     3,
	}
	store := &archiveProjectStoreMock{meta: meta}
	publisher := &archiveProjectPublisherMock{}
	cache := &archiveProjectCacheMock{getProject: nil, getOK: false}
	service := &Service{
		store:                store,
		persistencePublisher: publisher,
		projectCache:         cache,
	}

	_, err := service.ArchiveProject(context.Background(), &applicationpkg.ArchiveProjectRequest{Contract: contract.Hex()})
	if err != nil {
		t.Fatalf("archive project: %v", err)
	}
	if cache.lastSet == nil {
		t.Fatalf("set project received nil project")
	}
	if cache.lastSet.Meta.Contract != contract {
		t.Fatalf("contract = %s, want %s", cache.lastSet.Meta.Contract, contract)
	}
	if cache.lastSet.Meta.Creator != creator {
		t.Fatalf("creator = %s, want %s", cache.lastSet.Meta.Creator, creator)
	}
	if cache.lastSet.Meta.BlockNumber != meta.BlockNumber {
		t.Fatalf("block number = %d, want %d", cache.lastSet.Meta.BlockNumber, meta.BlockNumber)
	}
	if !cache.lastSet.Meta.IsArchived || cache.lastSet.Meta.ArchivedAt.IsZero() {
		t.Fatalf("archived fields not set correctly: %+v", cache.lastSet.Meta)
	}
}

func TestArchiveProjectSetProjectErrorStopsFlow(t *testing.T) {
	contract := common.HexToAddress("0x7000000000000000000000000000000000000007")
	meta := &appstore.ProjectMeta{Contract: contract}
	expectedErr := errors.New("set project failed")
	store := &archiveProjectStoreMock{meta: meta}
	publisher := &archiveProjectPublisherMock{}
	cache := &archiveProjectCacheMock{
		getProject: &Project{Meta: projectMetaFromStore(*meta)},
		getOK:      true,
		setErr:     expectedErr,
	}
	service := &Service{
		store:                store,
		persistencePublisher: publisher,
		projectCache:         cache,
	}

	_, err := service.ArchiveProject(context.Background(), &applicationpkg.ArchiveProjectRequest{Contract: contract.Hex()})
	if !errors.Is(err, expectedErr) {
		t.Fatalf("archive project err = %v, want %v", err, expectedErr)
	}
}
