package application

import (
	"context"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	applicationpkg "github.com/useryege/athena/internal/application/apiclient"
	appstore "github.com/useryege/athena/internal/application/store"
)

type policyReevaluationProjectCache struct {
	projects map[common.Address]*Project
	active   []common.Address
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

func (c *policyReevaluationProjectCache) ListProjects(context.Context) ([]*Project, error) {
	projects := make([]*Project, 0, len(c.active))
	for _, contract := range c.active {
		if project := c.projects[contract]; project != nil {
			projects = append(projects, project)
		}
	}
	return projects, nil
}

func (c *policyReevaluationProjectCache) ListProjectsPage(_ context.Context, page int32, pageSize int32) ([]*Project, int64, int32, int32, error) {
	projects, err := c.ListProjects(context.Background())
	if err != nil {
		return nil, 0, 0, 0, err
	}
	total, page, pageSize := paginateProjects(page, pageSize, &projects)
	return projects, total, page, pageSize, nil
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

type sourcecodeBlacklistContractModelFake struct {
	items       []appstore.SourcecodeBlacklistContract
	updatedNote bool
}

func (m *sourcecodeBlacklistContractModelFake) Load(context.Context) error {
	return nil
}

func (m *sourcecodeBlacklistContractModelFake) List(context.Context) ([]appstore.SourcecodeBlacklistContract, error) {
	return append([]appstore.SourcecodeBlacklistContract(nil), m.items...), nil
}

func (m *sourcecodeBlacklistContractModelFake) Version(context.Context) (string, error) {
	return "sourcecode-contract", nil
}

func (m *sourcecodeBlacklistContractModelFake) Add(_ context.Context, item appstore.SourcecodeBlacklistContract) error {
	m.items = append([]appstore.SourcecodeBlacklistContract{item}, m.items...)
	return nil
}

func (m *sourcecodeBlacklistContractModelFake) UpdateNote(_ context.Context, contract common.Address, note string) error {
	for i := range m.items {
		if m.items[i].Contract == contract {
			m.items[i].Note = note
			m.updatedNote = true
			return nil
		}
	}
	return appstore.ErrSourcecodeBlacklistContractNotFound
}

func (m *sourcecodeBlacklistContractModelFake) Delete(_ context.Context, contract common.Address) error {
	filtered := make([]appstore.SourcecodeBlacklistContract, 0, len(m.items))
	found := false
	for _, item := range m.items {
		if item.Contract == contract {
			found = true
			continue
		}
		filtered = append(filtered, item)
	}
	if !found {
		return appstore.ErrSourcecodeBlacklistContractNotFound
	}
	m.items = filtered
	return nil
}

type walletBlacklistModelFake struct {
	items       []appstore.WalletBlacklistEntry
	updatedNote bool
}

func (m *walletBlacklistModelFake) Load(context.Context) error {
	return nil
}

func (m *walletBlacklistModelFake) List(context.Context) ([]appstore.WalletBlacklistEntry, error) {
	return append([]appstore.WalletBlacklistEntry(nil), m.items...), nil
}

func (m *walletBlacklistModelFake) Version(context.Context) (string, error) {
	return "wallet", nil
}

func (m *walletBlacklistModelFake) Add(_ context.Context, item appstore.WalletBlacklistEntry) error {
	m.items = append([]appstore.WalletBlacklistEntry{item}, m.items...)
	return nil
}

func (m *walletBlacklistModelFake) UpdateNote(_ context.Context, wallet common.Address, note string) error {
	for i := range m.items {
		if m.items[i].Wallet == wallet {
			m.items[i].Note = note
			m.updatedNote = true
			return nil
		}
	}
	return appstore.ErrWalletBlacklistEntryNotFound
}

func (m *walletBlacklistModelFake) Delete(_ context.Context, wallet common.Address) error {
	filtered := make([]appstore.WalletBlacklistEntry, 0, len(m.items))
	found := false
	for _, item := range m.items {
		if item.Wallet == wallet {
			found = true
			continue
		}
		filtered = append(filtered, item)
	}
	if !found {
		return appstore.ErrWalletBlacklistEntryNotFound
	}
	m.items = filtered
	return nil
}

func TestAddBytecodeBlacklistContractDoesNotTriggerPolicyReevaluation(t *testing.T) {
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

	assertNoContractTriggered(t, triggerCh)
}

func TestDeleteBytecodeBlacklistContractDoesNotTriggerPolicyReevaluation(t *testing.T) {
	projectContract := common.HexToAddress("0x00000000000000000000000000000000000000d1")
	blacklistContract := common.HexToAddress("0x00000000000000000000000000000000000000d2")
	cache := newPolicyReevaluationProjectCache(&Project{Meta: ProjectMeta{Contract: projectContract}})
	triggerCh := make(chan common.Address, 2)
	service := newStartedPolicyReevaluationService(cache, triggerCh)
	service.bytecodeBlacklist = &bytecodeBlacklistModelFake{items: []appstore.BytecodeBlacklistContract{{Contract: blacklistContract}}}

	if _, err := service.DeleteBytecodeBlacklistContract(context.Background(), &applicationpkg.DeleteBytecodeBlacklistContractRequest{Contract: blacklistContract.Hex()}); err != nil {
		t.Fatalf("delete bytecode blacklist contract: %v", err)
	}

	assertNoContractTriggered(t, triggerCh)
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

func TestAddSourcecodeBlacklistContractDoesNotTriggerPolicyReevaluation(t *testing.T) {
	projectContract := common.HexToAddress("0x00000000000000000000000000000000000000f1")
	blacklistContract := common.HexToAddress("0x00000000000000000000000000000000000000f2")
	cache := newPolicyReevaluationProjectCache(
		&Project{Meta: ProjectMeta{Contract: projectContract}},
		&Project{Meta: ProjectMeta{Contract: blacklistContract, SourceCode: "contract Source {}"}},
	)
	triggerCh := make(chan common.Address, 3)
	service := newStartedPolicyReevaluationService(cache, triggerCh)
	service.sourcecodeBlacklist = &sourcecodeBlacklistContractModelFake{}

	if _, err := service.AddSourcecodeBlacklistContract(context.Background(), &applicationpkg.AddSourcecodeBlacklistContractRequest{Contract: blacklistContract.Hex()}); err != nil {
		t.Fatalf("add sourcecode blacklist contract: %v", err)
	}

	assertNoContractTriggered(t, triggerCh)
}

func TestDeleteSourcecodeBlacklistContractDoesNotTriggerPolicyReevaluation(t *testing.T) {
	projectContract := common.HexToAddress("0x0000000000000000000000000000000000000101")
	blacklistContract := common.HexToAddress("0x0000000000000000000000000000000000000102")
	cache := newPolicyReevaluationProjectCache(&Project{Meta: ProjectMeta{Contract: projectContract}})
	triggerCh := make(chan common.Address, 2)
	service := newStartedPolicyReevaluationService(cache, triggerCh)
	service.sourcecodeBlacklist = &sourcecodeBlacklistContractModelFake{items: []appstore.SourcecodeBlacklistContract{{Contract: blacklistContract, SourceHash: common.HexToHash("0x1234")}}}

	if _, err := service.DeleteSourcecodeBlacklistContract(context.Background(), &applicationpkg.DeleteSourcecodeBlacklistContractRequest{Contract: blacklistContract.Hex()}); err != nil {
		t.Fatalf("delete sourcecode blacklist contract: %v", err)
	}

	assertNoContractTriggered(t, triggerCh)
}

func TestUpdateSourcecodeBlacklistContractNoteDoesNotTriggerPolicyReevaluation(t *testing.T) {
	projectContract := common.HexToAddress("0x0000000000000000000000000000000000000111")
	blacklistContract := common.HexToAddress("0x0000000000000000000000000000000000000112")
	cache := newPolicyReevaluationProjectCache(&Project{Meta: ProjectMeta{Contract: projectContract}})
	triggerCh := make(chan common.Address, 2)
	model := &sourcecodeBlacklistContractModelFake{items: []appstore.SourcecodeBlacklistContract{{Contract: blacklistContract, SourceHash: common.HexToHash("0x1234")}}}
	service := newStartedPolicyReevaluationService(cache, triggerCh)
	service.sourcecodeBlacklist = model

	if _, err := service.UpdateSourcecodeBlacklistContractNote(context.Background(), &applicationpkg.UpdateSourcecodeBlacklistContractNoteRequest{
		Contract: blacklistContract.Hex(),
		Note:     "new note",
	}); err != nil {
		t.Fatalf("update sourcecode blacklist note: %v", err)
	}
	if !model.updatedNote {
		t.Fatalf("sourcecode blacklist note was not updated")
	}
	assertNoContractTriggered(t, triggerCh)
}

func TestAddWalletBlacklistEntryDoesNotTriggerPolicyReevaluation(t *testing.T) {
	projectContract := common.HexToAddress("0x0000000000000000000000000000000000000121")
	wallet := common.HexToAddress("0x0000000000000000000000000000000000000122")
	cache := newPolicyReevaluationProjectCache(&Project{Meta: ProjectMeta{Contract: projectContract}})
	triggerCh := make(chan common.Address, 2)
	service := newStartedPolicyReevaluationService(cache, triggerCh)
	service.walletBlacklist = &walletBlacklistModelFake{}
	service.codeAtFunc = func(context.Context, common.Address) ([]byte, error) {
		return nil, nil
	}

	if _, err := service.AddWalletBlacklistEntry(context.Background(), &applicationpkg.AddWalletBlacklistEntryRequest{Wallet: wallet.Hex()}); err != nil {
		t.Fatalf("add wallet blacklist entry: %v", err)
	}

	assertNoContractTriggered(t, triggerCh)
}

func TestDeleteWalletBlacklistEntryDoesNotTriggerPolicyReevaluation(t *testing.T) {
	projectContract := common.HexToAddress("0x0000000000000000000000000000000000000131")
	wallet := common.HexToAddress("0x0000000000000000000000000000000000000132")
	cache := newPolicyReevaluationProjectCache(&Project{Meta: ProjectMeta{Contract: projectContract}})
	triggerCh := make(chan common.Address, 2)
	service := newStartedPolicyReevaluationService(cache, triggerCh)
	service.walletBlacklist = &walletBlacklistModelFake{items: []appstore.WalletBlacklistEntry{{Wallet: wallet}}}

	if _, err := service.DeleteWalletBlacklistEntry(context.Background(), &applicationpkg.DeleteWalletBlacklistEntryRequest{Wallet: wallet.Hex()}); err != nil {
		t.Fatalf("delete wallet blacklist entry: %v", err)
	}

	assertNoContractTriggered(t, triggerCh)
}

func TestUpdateWalletBlacklistEntryNoteDoesNotTriggerPolicyReevaluation(t *testing.T) {
	projectContract := common.HexToAddress("0x0000000000000000000000000000000000000141")
	wallet := common.HexToAddress("0x0000000000000000000000000000000000000142")
	cache := newPolicyReevaluationProjectCache(&Project{Meta: ProjectMeta{Contract: projectContract}})
	triggerCh := make(chan common.Address, 2)
	model := &walletBlacklistModelFake{items: []appstore.WalletBlacklistEntry{{Wallet: wallet}}}
	service := newStartedPolicyReevaluationService(cache, triggerCh)
	service.walletBlacklist = model

	if _, err := service.UpdateWalletBlacklistEntryNote(context.Background(), &applicationpkg.UpdateWalletBlacklistEntryNoteRequest{
		Wallet: wallet.Hex(),
		Note:   "new note",
	}); err != nil {
		t.Fatalf("update wallet blacklist note: %v", err)
	}
	if !model.updatedNote {
		t.Fatalf("wallet blacklist note was not updated")
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

func assertNoContractTriggered(t *testing.T, ch <-chan common.Address) {
	t.Helper()
	select {
	case contract := <-ch:
		t.Fatalf("unexpected policy trigger for %s", contract.Hex())
	case <-time.After(50 * time.Millisecond):
	}
}

var _ ProjectSnapshotCache = (*policyReevaluationProjectCache)(nil)
