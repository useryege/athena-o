package application

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	applicationpkg "github.com/useryege/athena/internal/application/apiclient"
	"github.com/useryege/athena/internal/application/sourcecode"
	appstore "github.com/useryege/athena/internal/application/store"
)

type policyReevaluationProjectCache struct {
	projects map[common.Address]*Project
	active   []common.Address
	archived []common.Address
}

func newPolicyReevaluationProjectCache(projects ...*Project) *policyReevaluationProjectCache {
	cache := &policyReevaluationProjectCache{
		projects: make(map[common.Address]*Project, len(projects)),
	}
	for _, project := range projects {
		if project == nil {
			continue
		}
		contract := project.Meta.Contract
		cache.projects[contract] = project
		if project.Meta.IsArchived {
			cache.archived = append(cache.archived, contract)
			continue
		}
		cache.active = append(cache.active, contract)
	}
	return cache
}

func (c *policyReevaluationProjectCache) ReplaceAll(context.Context, []*Project) error {
	return nil
}

func (c *policyReevaluationProjectCache) SetProject(_ context.Context, project *Project) error {
	if project == nil {
		return nil
	}
	c.projects[project.Meta.Contract] = project
	return nil
}

func (c *policyReevaluationProjectCache) UpdateProject(_ context.Context, contract common.Address, updater ProjectUpdater) (bool, error) {
	current, exists := c.projects[contract]
	next, changed, err := updater(current, exists)
	if err != nil || !changed {
		return false, err
	}
	if next == nil {
		delete(c.projects, contract)
		return true, nil
	}
	c.projects[contract] = next
	return true, nil
}

func (c *policyReevaluationProjectCache) DeleteProject(_ context.Context, contract common.Address) error {
	delete(c.projects, contract)
	return nil
}

func (c *policyReevaluationProjectCache) GetProject(_ context.Context, contract common.Address) (*Project, bool, error) {
	project, ok := c.projects[contract]
	return project, ok, nil
}

func (c *policyReevaluationProjectCache) GetMaxProjectBlockNumber(context.Context) (uint64, bool, error) {
	return 0, false, nil
}

func (c *policyReevaluationProjectCache) ListActiveProjects(context.Context) ([]*Project, error) {
	projects := make([]*Project, 0, len(c.active))
	for _, contract := range c.active {
		if project := c.projects[contract]; project != nil {
			projects = append(projects, project)
		}
	}
	return projects, nil
}

func (c *policyReevaluationProjectCache) ListArchivedProjects(_ context.Context, page int32, pageSize int32) ([]*Project, int64, int32, int32, error) {
	page, pageSize = normalizeCachePage(page, pageSize)
	total := int64(len(c.archived))
	start := int64(page-1) * int64(pageSize)
	if start >= total {
		return nil, total, page, pageSize, nil
	}
	stop := start + int64(pageSize)
	if stop > total {
		stop = total
	}
	projects := make([]*Project, 0, stop-start)
	for _, contract := range c.archived[start:stop] {
		if project := c.projects[contract]; project != nil {
			projects = append(projects, project)
		}
	}
	return projects, total, page, pageSize, nil
}

type sourceCodeBlacklistModelFake struct {
	added   []string
	deleted []string
}

func (m *sourceCodeBlacklistModelFake) Load(context.Context) error {
	return nil
}

func (m *sourceCodeBlacklistModelFake) List(context.Context) ([]string, error) {
	return nil, nil
}

func (m *sourceCodeBlacklistModelFake) Version(context.Context) (string, error) {
	return "source", nil
}

func (m *sourceCodeBlacklistModelFake) Add(_ context.Context, field string) error {
	m.added = append(m.added, field)
	return nil
}

func (m *sourceCodeBlacklistModelFake) Delete(_ context.Context, field string) error {
	m.deleted = append(m.deleted, field)
	return nil
}

type bytecodeBlacklistModelFake struct {
	items       []appstore.BytecodeBlacklistContract
	updatedNote bool
}

func (m *bytecodeBlacklistModelFake) Load(context.Context) error {
	return nil
}

func (m *bytecodeBlacklistModelFake) List(context.Context) ([]appstore.BytecodeBlacklistContract, error) {
	return append([]appstore.BytecodeBlacklistContract(nil), m.items...), nil
}

func (m *bytecodeBlacklistModelFake) Version(context.Context) (string, error) {
	return "bytecode", nil
}

func (m *bytecodeBlacklistModelFake) Add(_ context.Context, item appstore.BytecodeBlacklistContract) error {
	m.items = append([]appstore.BytecodeBlacklistContract{item}, m.items...)
	return nil
}

func (m *bytecodeBlacklistModelFake) UpdateNote(_ context.Context, contract common.Address, note string) error {
	for i := range m.items {
		if m.items[i].Contract == contract {
			m.items[i].Note = note
			m.updatedNote = true
			return nil
		}
	}
	return appstore.ErrBytecodeBlacklistContractNotFound
}

func (m *bytecodeBlacklistModelFake) Delete(_ context.Context, contract common.Address) error {
	filtered := make([]appstore.BytecodeBlacklistContract, 0, len(m.items))
	found := false
	for _, item := range m.items {
		if item.Contract == contract {
			found = true
			continue
		}
		filtered = append(filtered, item)
	}
	if !found {
		return appstore.ErrBytecodeBlacklistContractNotFound
	}
	m.items = filtered
	return nil
}

func TestAddSourceCodeBlacklistFieldResetsReportsAndEnqueuesProjects(t *testing.T) {
	activeContract := common.HexToAddress("0x00000000000000000000000000000000000000a1")
	archivedContract := common.HexToAddress("0x00000000000000000000000000000000000000a2")
	cache := newPolicyReevaluationProjectCache(
		&Project{
			Meta: ProjectMeta{Contract: activeContract},
			Runtime: ProjectRuntime{SourceCodeBlacklist: sourcecode.BlacklistReport{
				ResolvedAt: time.Now().Add(-time.Hour),
			}},
		},
		&Project{
			Meta: ProjectMeta{Contract: archivedContract, IsArchived: true},
			Runtime: ProjectRuntime{SourceCodeBlacklist: sourcecode.BlacklistReport{
				HasBlacklistFields: true,
				BlacklistFields:    []string{"owner"},
				ResolvedAt:         time.Now().Add(-time.Hour),
			}},
		},
	)
	triggerCh := make(chan common.Address, 4)
	model := &sourceCodeBlacklistModelFake{}
	service := newStartedPolicyReevaluationService(cache, triggerCh)
	service.sourceBlacklist = model

	if _, err := service.AddSourceCodeBlacklistField(context.Background(), &applicationpkg.AddSourceCodeBlacklistFieldRequest{Field: "owner"}); err != nil {
		t.Fatalf("add source blacklist field: %v", err)
	}

	wantContracts := []common.Address{activeContract, archivedContract}
	if got := waitForContracts(t, triggerCh, len(wantContracts)); !sameAddressSet(got, wantContracts) {
		t.Fatalf("triggered contracts = %v, want %v", got, wantContracts)
	}
	assertSourceCodeReportReset(t, cache.projects[activeContract])
	assertSourceCodeReportReset(t, cache.projects[archivedContract])
	if !reflect.DeepEqual(model.added, []string{"owner"}) {
		t.Fatalf("source blacklist added = %v, want [owner]", model.added)
	}
}

func TestDeleteSourceCodeBlacklistFieldResetsReportsAndEnqueuesProjects(t *testing.T) {
	activeContract := common.HexToAddress("0x00000000000000000000000000000000000000b1")
	archivedContract := common.HexToAddress("0x00000000000000000000000000000000000000b2")
	cache := newPolicyReevaluationProjectCache(
		&Project{
			Meta: ProjectMeta{Contract: activeContract},
			Runtime: ProjectRuntime{SourceCodeBlacklist: sourcecode.BlacklistReport{
				HasBlacklistFields: true,
				BlacklistFields:    []string{"admin"},
				ResolvedAt:         time.Now().Add(-time.Hour),
			}},
		},
		&Project{
			Meta: ProjectMeta{Contract: archivedContract, IsArchived: true},
			Runtime: ProjectRuntime{SourceCodeBlacklist: sourcecode.BlacklistReport{
				ResolvedAt: time.Now().Add(-time.Hour),
			}},
		},
	)
	triggerCh := make(chan common.Address, 4)
	model := &sourceCodeBlacklistModelFake{}
	service := newStartedPolicyReevaluationService(cache, triggerCh)
	service.sourceBlacklist = model

	if _, err := service.DeleteSourceCodeBlacklistField(context.Background(), &applicationpkg.DeleteSourceCodeBlacklistFieldRequest{Field: "admin"}); err != nil {
		t.Fatalf("delete source blacklist field: %v", err)
	}

	wantContracts := []common.Address{activeContract, archivedContract}
	if got := waitForContracts(t, triggerCh, len(wantContracts)); !sameAddressSet(got, wantContracts) {
		t.Fatalf("triggered contracts = %v, want %v", got, wantContracts)
	}
	assertSourceCodeReportReset(t, cache.projects[activeContract])
	assertSourceCodeReportReset(t, cache.projects[archivedContract])
	if !reflect.DeepEqual(model.deleted, []string{"admin"}) {
		t.Fatalf("source blacklist deleted = %v, want [admin]", model.deleted)
	}
}

func TestAddBytecodeBlacklistContractTriggersFullPolicyReevaluation(t *testing.T) {
	projectContract := common.HexToAddress("0x00000000000000000000000000000000000000c1")
	blacklistContract := common.HexToAddress("0x00000000000000000000000000000000000000c2")
	cache := newPolicyReevaluationProjectCache(&Project{Meta: ProjectMeta{Contract: projectContract}})
	triggerCh := make(chan common.Address, 2)
	service := newStartedPolicyReevaluationService(cache, triggerCh)
	service.bytecodeBlacklist = &bytecodeBlacklistModelFake{}
	service.codeAtFunc = func(context.Context, common.Address) ([]byte, error) {
		return []byte{0x1, 0x2}, nil
	}

	if _, err := service.AddBytecodeBlacklistContract(context.Background(), &applicationpkg.AddBytecodeBlacklistContractRequest{Contract: blacklistContract.Hex()}); err != nil {
		t.Fatalf("add bytecode blacklist contract: %v", err)
	}

	if got := waitForContracts(t, triggerCh, 1); !sameAddressSet(got, []common.Address{projectContract}) {
		t.Fatalf("triggered contracts = %v, want [%s]", got, projectContract.Hex())
	}
}

func TestDeleteBytecodeBlacklistContractTriggersFullPolicyReevaluation(t *testing.T) {
	projectContract := common.HexToAddress("0x00000000000000000000000000000000000000d1")
	blacklistContract := common.HexToAddress("0x00000000000000000000000000000000000000d2")
	cache := newPolicyReevaluationProjectCache(&Project{Meta: ProjectMeta{Contract: projectContract}})
	triggerCh := make(chan common.Address, 2)
	service := newStartedPolicyReevaluationService(cache, triggerCh)
	service.bytecodeBlacklist = &bytecodeBlacklistModelFake{items: []appstore.BytecodeBlacklistContract{{Contract: blacklistContract}}}

	if _, err := service.DeleteBytecodeBlacklistContract(context.Background(), &applicationpkg.DeleteBytecodeBlacklistContractRequest{Contract: blacklistContract.Hex()}); err != nil {
		t.Fatalf("delete bytecode blacklist contract: %v", err)
	}

	if got := waitForContracts(t, triggerCh, 1); !sameAddressSet(got, []common.Address{projectContract}) {
		t.Fatalf("triggered contracts = %v, want [%s]", got, projectContract.Hex())
	}
}

func TestUpdateBytecodeBlacklistContractNoteDoesNotTriggerPolicyReevaluation(t *testing.T) {
	projectContract := common.HexToAddress("0x00000000000000000000000000000000000000e1")
	blacklistContract := common.HexToAddress("0x00000000000000000000000000000000000000e2")
	cache := newPolicyReevaluationProjectCache(&Project{Meta: ProjectMeta{Contract: projectContract}})
	triggerCh := make(chan common.Address, 2)
	model := &bytecodeBlacklistModelFake{items: []appstore.BytecodeBlacklistContract{{Contract: blacklistContract}}}
	service := newStartedPolicyReevaluationService(cache, triggerCh)
	service.bytecodeBlacklist = model

	if _, err := service.UpdateBytecodeBlacklistContractNote(context.Background(), &applicationpkg.UpdateBytecodeBlacklistContractNoteRequest{
		Contract: blacklistContract.Hex(),
		Note:     "new note",
	}); err != nil {
		t.Fatalf("update bytecode blacklist note: %v", err)
	}
	if !model.updatedNote {
		t.Fatalf("bytecode blacklist note was not updated")
	}
	assertNoContractTriggered(t, triggerCh)
}

func newStartedPolicyReevaluationService(projectCache ProjectSnapshotCache, triggerCh chan common.Address) *Service {
	return &Service{
		projectCache:    projectCache,
		lifecycleCtx:    context.Background(),
		policyTriggerCh: triggerCh,
		started:         true,
	}
}

func waitForContracts(t *testing.T, ch <-chan common.Address, count int) []common.Address {
	t.Helper()
	got := make([]common.Address, 0, count)
	timeout := time.After(2 * time.Second)
	for len(got) < count {
		select {
		case contract := <-ch:
			got = append(got, contract)
		case <-timeout:
			t.Fatalf("timed out waiting for %d policy triggers, got %d", count, len(got))
		}
	}
	return got
}

func assertNoContractTriggered(t *testing.T, ch <-chan common.Address) {
	t.Helper()
	select {
	case contract := <-ch:
		t.Fatalf("unexpected policy trigger for %s", contract.Hex())
	case <-time.After(50 * time.Millisecond):
	}
}

func assertSourceCodeReportReset(t *testing.T, project *Project) {
	t.Helper()
	if project == nil {
		t.Fatalf("project is nil")
	}
	report := project.Runtime.SourceCodeBlacklist
	if report.HasBlacklistFields || len(report.BlacklistFields) != 0 || !report.ResolvedAt.IsZero() {
		t.Fatalf("source code blacklist report = %+v, want zero value", report)
	}
}

func sameAddressSet(left []common.Address, right []common.Address) bool {
	if len(left) != len(right) {
		return false
	}
	counts := make(map[common.Address]int, len(left))
	for _, item := range left {
		counts[item]++
	}
	for _, item := range right {
		counts[item]--
		if counts[item] < 0 {
			return false
		}
	}
	for _, count := range counts {
		if count != 0 {
			return false
		}
	}
	return true
}

var _ ProjectSnapshotCache = (*policyReevaluationProjectCache)(nil)
