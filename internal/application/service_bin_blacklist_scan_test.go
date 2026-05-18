package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	appstore "github.com/useryege/athena/internal/application/store"
)

type binBlacklistStoreMock struct {
	items   []appstore.BytecodeBlacklistContract
	listErr error
}

func (m *binBlacklistStoreMock) SaveProjectMeta(context.Context, appstore.ProjectMeta) error {
	return nil
}

func (m *binBlacklistStoreMock) ListProjectMetas(context.Context) ([]appstore.ProjectMeta, error) {
	return nil, nil
}

func (m *binBlacklistStoreMock) ListAllProjectMetas(context.Context) ([]appstore.ProjectMeta, error) {
	return nil, nil
}

func (m *binBlacklistStoreMock) UpdateProjectSourceCode(context.Context, common.Address, string) error {
	return nil
}

func (m *binBlacklistStoreMock) ArchiveProjectByContract(context.Context, common.Address) error {
	return nil
}

func (m *binBlacklistStoreMock) UnarchiveProjectByContract(context.Context, common.Address) error {
	return nil
}

func (m *binBlacklistStoreMock) ListArchivedProjectMetas(context.Context, int32, int32) ([]appstore.ProjectMeta, int64, int32, int32, error) {
	return nil, 0, 0, 0, nil
}

func (m *binBlacklistStoreMock) GetArchivedProjectMetaByContract(context.Context, common.Address) (*appstore.ProjectMeta, error) {
	return nil, nil
}

func (m *binBlacklistStoreMock) GetProjectMetaByContract(context.Context, common.Address) (*appstore.ProjectMeta, error) {
	return nil, nil
}

func (m *binBlacklistStoreMock) ListSourceCodeBlacklistFields(context.Context) ([]string, error) {
	return nil, nil
}

func (m *binBlacklistStoreMock) AddSourceCodeBlacklistField(context.Context, string) error {
	return nil
}

func (m *binBlacklistStoreMock) DeleteSourceCodeBlacklistField(context.Context, string) error {
	return nil
}

func (m *binBlacklistStoreMock) ListBytecodeBlacklistContracts(context.Context) ([]appstore.BytecodeBlacklistContract, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.items, nil
}

func (m *binBlacklistStoreMock) AddBytecodeBlacklistContract(context.Context, appstore.BytecodeBlacklistContract) error {
	return nil
}

func (m *binBlacklistStoreMock) UpdateBytecodeBlacklistContractNote(context.Context, common.Address, string) error {
	return nil
}

func (m *binBlacklistStoreMock) DeleteBytecodeBlacklistContract(context.Context, common.Address) error {
	return nil
}

type storeWithoutBytecodeBlacklistMock struct{}

func (m *storeWithoutBytecodeBlacklistMock) SaveProjectMeta(context.Context, appstore.ProjectMeta) error {
	return nil
}

func (m *storeWithoutBytecodeBlacklistMock) ListProjectMetas(context.Context) ([]appstore.ProjectMeta, error) {
	return nil, nil
}

func (m *storeWithoutBytecodeBlacklistMock) ListAllProjectMetas(context.Context) ([]appstore.ProjectMeta, error) {
	return nil, nil
}

func (m *storeWithoutBytecodeBlacklistMock) UpdateProjectSourceCode(context.Context, common.Address, string) error {
	return nil
}

func (m *storeWithoutBytecodeBlacklistMock) ArchiveProjectByContract(context.Context, common.Address) error {
	return nil
}

func (m *storeWithoutBytecodeBlacklistMock) UnarchiveProjectByContract(context.Context, common.Address) error {
	return nil
}

func (m *storeWithoutBytecodeBlacklistMock) ListArchivedProjectMetas(context.Context, int32, int32) ([]appstore.ProjectMeta, int64, int32, int32, error) {
	return nil, 0, 0, 0, nil
}

func (m *storeWithoutBytecodeBlacklistMock) GetArchivedProjectMetaByContract(context.Context, common.Address) (*appstore.ProjectMeta, error) {
	return nil, nil
}

func (m *storeWithoutBytecodeBlacklistMock) GetProjectMetaByContract(context.Context, common.Address) (*appstore.ProjectMeta, error) {
	return nil, nil
}

func (m *storeWithoutBytecodeBlacklistMock) ListSourceCodeBlacklistFields(context.Context) ([]string, error) {
	return nil, nil
}

func (m *storeWithoutBytecodeBlacklistMock) AddSourceCodeBlacklistField(context.Context, string) error {
	return nil
}

func (m *storeWithoutBytecodeBlacklistMock) DeleteSourceCodeBlacklistField(context.Context, string) error {
	return nil
}

type binBlacklistPublisherMock struct {
	archives             []common.Address
	rawEventLogs         []appstore.ProjectEventLog
	acceptedEventLogs    []appstore.ProjectEventLog
	acceptedEventLogKeys map[string]struct{}
}

func (m *binBlacklistPublisherMock) Publish(context.Context, PersistenceEvent) error {
	return nil
}

func (m *binBlacklistPublisherMock) PublishProjectMetaSave(context.Context, appstore.ProjectMeta) error {
	return nil
}

func (m *binBlacklistPublisherMock) PublishProjectEventLog(_ context.Context, item appstore.ProjectEventLog) error {
	m.rawEventLogs = append(m.rawEventLogs, item)
	if m.acceptedEventLogKeys == nil {
		m.acceptedEventLogKeys = make(map[string]struct{})
	}
	key := item.Contract.Hex() + ":" + item.IdempotencyKey
	if _, exists := m.acceptedEventLogKeys[key]; exists {
		return nil
	}
	m.acceptedEventLogKeys[key] = struct{}{}
	m.acceptedEventLogs = append(m.acceptedEventLogs, item)
	return nil
}

func (m *binBlacklistPublisherMock) PublishProjectSourceCodeUpdate(context.Context, common.Address, string) error {
	return nil
}

func (m *binBlacklistPublisherMock) PublishProjectArchive(_ context.Context, contract common.Address) error {
	m.archives = append(m.archives, contract)
	return nil
}

func (m *binBlacklistPublisherMock) PublishProjectUnarchive(context.Context, common.Address) error {
	return nil
}

func (m *binBlacklistPublisherMock) PublishSourceCodeBlacklistAdd(context.Context, string) error {
	return nil
}

func (m *binBlacklistPublisherMock) PublishSourceCodeBlacklistDelete(context.Context, string) error {
	return nil
}

func TestScanAllProjectsForBINBlacklist_ActiveMatchAutoArchivesAndWritesEvent(t *testing.T) {
	contract := common.HexToAddress("0x1111111111111111111111111111111111111111")
	creator := common.HexToAddress("0x2222222222222222222222222222222222222222")
	code := []byte{0x60, 0x60, 0x60, 0x40}
	codeHash := crypto.Keccak256Hash(code)

	cache := &activeRefreshCacheMock{
		listActive: []*Project{
			{Meta: ProjectMeta{Contract: contract, Creator: creator, IsArchived: false}},
		},
		getByKey: map[string]*Project{
			contract.Hex(): {Meta: ProjectMeta{Contract: contract, Creator: creator, IsArchived: false}},
		},
	}
	store := &binBlacklistStoreMock{
		items: []appstore.BytecodeBlacklistContract{{Contract: contract, CodeHash: codeHash}},
	}
	publisher := &binBlacklistPublisherMock{}
	service := &Service{
		store:                store,
		projectCache:         cache,
		persistencePublisher: publisher,
		codeAtFunc: func(context.Context, common.Address) ([]byte, error) {
			return code, nil
		},
	}

	if err := service.scanAllProjectsForBINBlacklist(context.Background()); err != nil {
		t.Fatalf("scan all projects for BIN blacklist: %v", err)
	}
	if len(publisher.archives) != 1 || publisher.archives[0] != contract {
		t.Fatalf("archive calls = %v, want [%s]", publisher.archives, contract.Hex())
	}
	updated := cache.getByKey[contract.Hex()]
	if updated == nil || !updated.Meta.IsArchived {
		t.Fatalf("project is_archived = false, want true")
	}
	if updated.Meta.ArchivedAt.IsZero() {
		t.Fatalf("project archived_at is zero")
	}
	if len(publisher.acceptedEventLogs) != 1 {
		t.Fatalf("accepted event logs = %d, want 1", len(publisher.acceptedEventLogs))
	}
	item := publisher.acceptedEventLogs[0]
	if item.Contract != contract {
		t.Fatalf("event contract = %s, want %s", item.Contract.Hex(), contract.Hex())
	}
	if item.EventType != projectEventTypeAutoArchiveBIN {
		t.Fatalf("event type = %d, want %d", item.EventType, projectEventTypeAutoArchiveBIN)
	}
	if item.IdempotencyKey != projectEventIdempotencyAutoArchiveBIN {
		t.Fatalf("idempotency key = %q, want %q", item.IdempotencyKey, projectEventIdempotencyAutoArchiveBIN)
	}
}

func TestScanAllProjectsForBINBlacklist_ArchivedMatchBackfillsEventOnly(t *testing.T) {
	contract := common.HexToAddress("0x3333333333333333333333333333333333333333")
	creator := common.HexToAddress("0x4444444444444444444444444444444444444444")
	code := []byte{0x60, 0x61, 0x62}
	codeHash := crypto.Keccak256Hash(code)

	cache := &activeRefreshCacheMock{
		listArchived: []*Project{
			{Meta: ProjectMeta{Contract: contract, Creator: creator, IsArchived: true, ArchivedAt: time.Now().UTC()}},
		},
		getByKey: map[string]*Project{
			contract.Hex(): {Meta: ProjectMeta{Contract: contract, Creator: creator, IsArchived: true, ArchivedAt: time.Now().UTC()}},
		},
	}
	store := &binBlacklistStoreMock{
		items: []appstore.BytecodeBlacklistContract{{Contract: contract, CodeHash: codeHash}},
	}
	publisher := &binBlacklistPublisherMock{}
	service := &Service{
		store:                store,
		projectCache:         cache,
		persistencePublisher: publisher,
		codeAtFunc: func(context.Context, common.Address) ([]byte, error) {
			return code, nil
		},
	}

	if err := service.scanAllProjectsForBINBlacklist(context.Background()); err != nil {
		t.Fatalf("scan all projects for BIN blacklist: %v", err)
	}
	if len(publisher.archives) != 0 {
		t.Fatalf("archive calls = %d, want 0", len(publisher.archives))
	}
	if len(publisher.acceptedEventLogs) != 1 {
		t.Fatalf("accepted event logs = %d, want 1", len(publisher.acceptedEventLogs))
	}
}

func TestScanAllProjectsForBINBlacklist_NonMatchDoesNothing(t *testing.T) {
	contract := common.HexToAddress("0x5555555555555555555555555555555555555555")
	creator := common.HexToAddress("0x6666666666666666666666666666666666666666")
	blackCode := []byte{0xaa, 0xbb}
	projectCode := []byte{0xcc, 0xdd}

	cache := &activeRefreshCacheMock{
		listActive: []*Project{
			{Meta: ProjectMeta{Contract: contract, Creator: creator}},
		},
		getByKey: map[string]*Project{
			contract.Hex(): {Meta: ProjectMeta{Contract: contract, Creator: creator}},
		},
	}
	store := &binBlacklistStoreMock{
		items: []appstore.BytecodeBlacklistContract{
			{Contract: common.HexToAddress("0x7777777777777777777777777777777777777777"), CodeHash: crypto.Keccak256Hash(blackCode)},
		},
	}
	publisher := &binBlacklistPublisherMock{}
	service := &Service{
		store:                store,
		projectCache:         cache,
		persistencePublisher: publisher,
		codeAtFunc: func(context.Context, common.Address) ([]byte, error) {
			return projectCode, nil
		},
	}

	if err := service.scanAllProjectsForBINBlacklist(context.Background()); err != nil {
		t.Fatalf("scan all projects for BIN blacklist: %v", err)
	}
	if len(publisher.archives) != 0 {
		t.Fatalf("archive calls = %d, want 0", len(publisher.archives))
	}
	if len(publisher.rawEventLogs) != 0 {
		t.Fatalf("raw event log calls = %d, want 0", len(publisher.rawEventLogs))
	}
}

func TestScanAllProjectsForBINBlacklist_IdempotencyKeyPreventsDuplicateEventInsert(t *testing.T) {
	contract := common.HexToAddress("0x8888888888888888888888888888888888888888")
	creator := common.HexToAddress("0x9999999999999999999999999999999999999999")
	code := []byte{0x01, 0x02, 0x03}
	codeHash := crypto.Keccak256Hash(code)

	cache := &activeRefreshCacheMock{
		listActive: []*Project{
			{Meta: ProjectMeta{Contract: contract, Creator: creator, IsArchived: false}},
		},
		getByKey: map[string]*Project{
			contract.Hex(): {Meta: ProjectMeta{Contract: contract, Creator: creator, IsArchived: false}},
		},
	}
	store := &binBlacklistStoreMock{
		items: []appstore.BytecodeBlacklistContract{{Contract: contract, CodeHash: codeHash}},
	}
	publisher := &binBlacklistPublisherMock{}
	service := &Service{
		store:                store,
		projectCache:         cache,
		persistencePublisher: publisher,
		codeAtFunc: func(context.Context, common.Address) ([]byte, error) {
			return code, nil
		},
	}

	if err := service.scanAllProjectsForBINBlacklist(context.Background()); err != nil {
		t.Fatalf("first scan all projects for BIN blacklist: %v", err)
	}
	if err := service.scanAllProjectsForBINBlacklist(context.Background()); err != nil {
		t.Fatalf("second scan all projects for BIN blacklist: %v", err)
	}

	if len(publisher.archives) != 1 {
		t.Fatalf("archive calls = %d, want 1", len(publisher.archives))
	}
	if len(publisher.rawEventLogs) != 2 {
		t.Fatalf("raw event log calls = %d, want 2", len(publisher.rawEventLogs))
	}
	if len(publisher.acceptedEventLogs) != 1 {
		t.Fatalf("accepted event log inserts = %d, want 1", len(publisher.acceptedEventLogs))
	}
}

func TestScanAllProjectsForBINBlacklist_NoBytecodeBlacklistStoreNoop(t *testing.T) {
	service := &Service{store: &storeWithoutBytecodeBlacklistMock{}}
	if err := service.scanAllProjectsForBINBlacklist(context.Background()); err != nil {
		t.Fatalf("scan without bytecode blacklist store: %v", err)
	}
}

func TestScanAllProjectsForBINBlacklist_BytecodeFetchFailureContinues(t *testing.T) {
	contractA := common.HexToAddress("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	contractB := common.HexToAddress("0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")
	creatorA := common.HexToAddress("0xcccccccccccccccccccccccccccccccccccccccc")
	creatorB := common.HexToAddress("0xdddddddddddddddddddddddddddddddddddddddd")
	codeA := []byte{0x60, 0x00}
	codeB := []byte{0x60, 0x01}

	cache := &activeRefreshCacheMock{
		listActive: []*Project{
			{Meta: ProjectMeta{Contract: contractA, Creator: creatorA}},
			{Meta: ProjectMeta{Contract: contractB, Creator: creatorB}},
		},
		getByKey: map[string]*Project{
			contractA.Hex(): {Meta: ProjectMeta{Contract: contractA, Creator: creatorA}},
			contractB.Hex(): {Meta: ProjectMeta{Contract: contractB, Creator: creatorB}},
		},
	}
	store := &binBlacklistStoreMock{
		items: []appstore.BytecodeBlacklistContract{
			{Contract: contractA, CodeHash: crypto.Keccak256Hash(codeA)},
			{Contract: contractB, CodeHash: crypto.Keccak256Hash(codeB)},
		},
	}
	publisher := &binBlacklistPublisherMock{}
	service := &Service{
		store:                store,
		projectCache:         cache,
		persistencePublisher: publisher,
		codeAtFunc: func(_ context.Context, contract common.Address) ([]byte, error) {
			if contract == contractA {
				return nil, errors.New("rpc failure")
			}
			return codeB, nil
		},
	}

	if err := service.scanAllProjectsForBINBlacklist(context.Background()); err != nil {
		t.Fatalf("scan all projects for BIN blacklist: %v", err)
	}
	if len(publisher.archives) != 1 || publisher.archives[0] != contractB {
		t.Fatalf("archive calls = %v, want [%s]", publisher.archives, contractB.Hex())
	}
	if len(publisher.acceptedEventLogs) != 1 || publisher.acceptedEventLogs[0].Contract != contractB {
		t.Fatalf("accepted event logs = %+v, want one item for %s", publisher.acceptedEventLogs, contractB.Hex())
	}
}
