package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/google/uuid"
	appcache "github.com/useryege/athena/internal/application/cache"
	"github.com/useryege/athena/internal/application/sourcecode"
	athenacontract "github.com/useryege/athena/pkg/abi/ATHENA"
)

type projectSyncMock struct {
	sourceByProjectID map[uuid.UUID]string
	sourceErrByID     map[uuid.UUID]error
	syncCalls         []uuid.UUID
}

func (m *projectSyncMock) SyncSourceCodeOnce(ctx context.Context, event *Project) (bool, error) {
	if event == nil {
		return false, nil
	}
	m.syncCalls = append(m.syncCalls, event.Meta.ProjectID)
	if err, ok := m.sourceErrByID[event.Meta.ProjectID]; ok {
		return false, err
	}
	if sourceCode, ok := m.sourceByProjectID[event.Meta.ProjectID]; ok {
		event.Meta.SourceCode = sourceCode
		return true, nil
	}
	return false, nil
}

func (m *projectSyncMock) SyncProjectStatesOnce(ctx context.Context, triggerBlockNumber uint64) error {
	return nil
}

type sourceCodeBlacklistModelMock struct {
	fields   []string
	listErr  error
	listCall int
}

var _ appcache.SourceCodeBlacklistModel = &sourceCodeBlacklistModelMock{}

func (m *sourceCodeBlacklistModelMock) Load(ctx context.Context) error {
	return nil
}

func (m *sourceCodeBlacklistModelMock) List(ctx context.Context) ([]string, error) {
	m.listCall++
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.fields, nil
}

func (m *sourceCodeBlacklistModelMock) Add(ctx context.Context, field string) error {
	return nil
}

func (m *sourceCodeBlacklistModelMock) Delete(ctx context.Context, field string) error {
	return nil
}

type analyzerMock struct {
	report sourcecode.BlacklistReport
	calls  int
}

func (m *analyzerMock) AnalyzeSourceCode(sourceCode string, blacklistFields []string) sourcecode.BlacklistReport {
	m.calls++
	return m.report
}

func TestSchedulerSyncAllProjectSourceCode(t *testing.T) {
	registry := NewProjectRegistry(nil)
	pendingSourceID := uuid.New()
	pendingAnalyzeID := uuid.New()
	analyzedID := uuid.New()

	mustSetProject(t, registry, newProjectForTest(pendingSourceID, "0x00000000000000000000000000000000000000A1", ""))
	mustSetProject(t, registry, newProjectForTest(pendingAnalyzeID, "0x00000000000000000000000000000000000000A2", "contract Token { address owner; }"))
	alreadyAnalyzed := newProjectForTest(analyzedID, "0x00000000000000000000000000000000000000A3", "contract Token { address owner; }")
	alreadyAnalyzed.Meta.SourceCodeBlacklist.ResolvedAt = time.Now()
	mustSetProject(t, registry, alreadyAnalyzed)

	syncer := &projectSyncMock{
		sourceByProjectID: map[uuid.UUID]string{
			pendingSourceID: "contract Token { address blacklist; }",
		},
	}
	blacklist := &sourceCodeBlacklistModelMock{fields: []string{"owner", "blacklist"}}
	analyzer := &analyzerMock{
		report: sourcecode.BlacklistReport{
			HasBlacklistFields: true,
			BlacklistFields:    []string{"blacklist"},
			ResolvedAt:         time.Now(),
		},
	}

	scheduler := NewProjectScheduler(registry, syncer, analyzer, blacklist, ProjectSchedulerOptions{}).(*projectSchedulerImpl)
	scheduler.ctx = context.Background()
	scheduler.syncAllProjectSourceCode()

	p1, ok, err := registry.GetProject(context.Background(), pendingSourceID)
	if err != nil || !ok {
		t.Fatalf("get project 1: ok=%v err=%v", ok, err)
	}
	if p1.Meta.SourceCode == "" {
		t.Fatal("project 1 source code is empty, want synced source code")
	}
	if p1.Meta.SourceCodeBlacklist.ResolvedAt.IsZero() {
		t.Fatal("project 1 blacklist resolvedAt is zero, want analyzed")
	}

	p2, ok, err := registry.GetProject(context.Background(), pendingAnalyzeID)
	if err != nil || !ok {
		t.Fatalf("get project 2: ok=%v err=%v", ok, err)
	}
	if p2.Meta.SourceCodeBlacklist.ResolvedAt.IsZero() {
		t.Fatal("project 2 blacklist resolvedAt is zero, want analyzed")
	}

	p3, ok, err := registry.GetProject(context.Background(), analyzedID)
	if err != nil || !ok {
		t.Fatalf("get project 3: ok=%v err=%v", ok, err)
	}
	if p3.Meta.SourceCodeBlacklist.ResolvedAt.IsZero() {
		t.Fatal("project 3 blacklist resolvedAt is zero, want preserved non-zero")
	}

	if analyzer.calls != 2 {
		t.Fatalf("analyzer calls = %d, want 2", analyzer.calls)
	}
	if blacklist.listCall != 2 {
		t.Fatalf("blacklist list calls = %d, want 2", blacklist.listCall)
	}
}

func TestSchedulerSyncSourceCodeErrorDoesNotBlockOthers(t *testing.T) {
	registry := NewProjectRegistry(nil)
	failedSyncID := uuid.New()
	pendingAnalyzeID := uuid.New()
	mustSetProject(t, registry, newProjectForTest(failedSyncID, "0x00000000000000000000000000000000000000B1", ""))
	mustSetProject(t, registry, newProjectForTest(pendingAnalyzeID, "0x00000000000000000000000000000000000000B2", "contract Token { address owner; }"))

	syncer := &projectSyncMock{
		sourceErrByID: map[uuid.UUID]error{
			failedSyncID: errors.New("sync failed"),
		},
	}
	blacklist := &sourceCodeBlacklistModelMock{fields: []string{"owner"}}
	analyzer := &analyzerMock{
		report: sourcecode.BlacklistReport{
			HasBlacklistFields: true,
			BlacklistFields:    []string{"owner"},
			ResolvedAt:         time.Now(),
		},
	}

	scheduler := NewProjectScheduler(registry, syncer, analyzer, blacklist, ProjectSchedulerOptions{}).(*projectSchedulerImpl)
	scheduler.ctx = context.Background()
	scheduler.syncAllProjectSourceCode()

	failedProject, ok, err := registry.GetProject(context.Background(), failedSyncID)
	if err != nil || !ok {
		t.Fatalf("get failed project: ok=%v err=%v", ok, err)
	}
	if failedProject.Meta.SourceCode != "" {
		t.Fatalf("failed project source code = %q, want empty", failedProject.Meta.SourceCode)
	}

	analyzedProject, ok, err := registry.GetProject(context.Background(), pendingAnalyzeID)
	if err != nil || !ok {
		t.Fatalf("get pending analyze project: ok=%v err=%v", ok, err)
	}
	if analyzedProject.Meta.SourceCodeBlacklist.ResolvedAt.IsZero() {
		t.Fatal("pending analyze project resolvedAt is zero, want analyzed")
	}
	if analyzer.calls != 1 {
		t.Fatalf("analyzer calls = %d, want 1", analyzer.calls)
	}
}

func TestSchedulerAnalyzeSkippedWhenBlacklistListFails(t *testing.T) {
	registry := NewProjectRegistry(nil)
	projectID := uuid.New()
	mustSetProject(t, registry, newProjectForTest(projectID, "0x00000000000000000000000000000000000000C1", "contract Token { address owner; }"))

	syncer := &projectSyncMock{}
	blacklist := &sourceCodeBlacklistModelMock{listErr: errors.New("list failed")}
	analyzer := &analyzerMock{
		report: sourcecode.BlacklistReport{
			HasBlacklistFields: true,
			BlacklistFields:    []string{"owner"},
			ResolvedAt:         time.Now(),
		},
	}

	scheduler := NewProjectScheduler(registry, syncer, analyzer, blacklist, ProjectSchedulerOptions{}).(*projectSchedulerImpl)
	scheduler.ctx = context.Background()
	scheduler.syncAllProjectSourceCode()

	project, ok, err := registry.GetProject(context.Background(), projectID)
	if err != nil || !ok {
		t.Fatalf("get project: ok=%v err=%v", ok, err)
	}
	if !project.Meta.SourceCodeBlacklist.ResolvedAt.IsZero() {
		t.Fatalf("resolvedAt = %v, want zero because analyze should be skipped", project.Meta.SourceCodeBlacklist.ResolvedAt)
	}
	if analyzer.calls != 0 {
		t.Fatalf("analyzer calls = %d, want 0", analyzer.calls)
	}
}

func newProjectForTest(projectID uuid.UUID, contractHex string, sourceCode string) *Project {
	return &Project{
		Meta: ProjectMeta{
			ProjectID:  projectID,
			Contract:   common.HexToAddress(contractHex),
			Creator:    common.HexToAddress("0x00000000000000000000000000000000000000A2"),
			SourceCode: sourceCode,
		},
		ChainState: athenacontract.AthenaProject{},
	}
}

func mustSetProject(t *testing.T, registry ProjectRegistry, project *Project) {
	t.Helper()
	if err := registry.SetProject(context.Background(), project.Meta.ProjectID, project); err != nil {
		t.Fatalf("set project %s: %v", project.Meta.ProjectID, err)
	}
}
