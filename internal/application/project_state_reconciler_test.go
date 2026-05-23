package application

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/redis/go-redis/v9"
	"github.com/useryege/athena/internal/application/redisport"
	appstore "github.com/useryege/athena/internal/application/store"
)

type sourceQualityAnalyzerFake struct {
	report string
	err    error
	calls  int
}

func (f *sourceQualityAnalyzerFake) AnalyzeContractSource(context.Context, string) (string, error) {
	f.calls++
	if f.err != nil {
		return "", f.err
	}
	return f.report, nil
}

type persistencePublisherFake struct {
	sourceQualityReports map[common.Address]string
	codeBinHashes        map[common.Address]common.Hash
}

func (p *persistencePublisherFake) Publish(context.Context, PersistenceEvent) error { return nil }
func (p *persistencePublisherFake) PublishProjectMetaSave(context.Context, appstore.ProjectMeta) error {
	return nil
}
func (p *persistencePublisherFake) PublishProjectEventLog(context.Context, appstore.ProjectEventLog) error {
	return nil
}
func (p *persistencePublisherFake) PublishProjectSourceCodeUpdate(context.Context, common.Address, string) error {
	return nil
}
func (p *persistencePublisherFake) PublishProjectCodeBinHashUpdate(_ context.Context, contract common.Address, codeBinHash common.Hash) error {
	if p.codeBinHashes == nil {
		p.codeBinHashes = map[common.Address]common.Hash{}
	}
	p.codeBinHashes[contract] = codeBinHash
	return nil
}
func (p *persistencePublisherFake) PublishProjectSourceQualityReportUpdate(_ context.Context, contract common.Address, report string) error {
	if p.sourceQualityReports == nil {
		p.sourceQualityReports = map[common.Address]string{}
	}
	p.sourceQualityReports[contract] = report
	return nil
}
func (p *persistencePublisherFake) PublishProjectArchive(context.Context, common.Address) error {
	return nil
}
func (p *persistencePublisherFake) PublishProjectUnarchive(context.Context, common.Address) error {
	return nil
}
func (p *persistencePublisherFake) PublishBytecodeBlacklistAdd(context.Context, appstore.BytecodeBlacklistContract) error {
	return nil
}
func (p *persistencePublisherFake) PublishBytecodeBlacklistUpdateNote(context.Context, common.Address, string) error {
	return nil
}
func (p *persistencePublisherFake) PublishBytecodeBlacklistDelete(context.Context, common.Address) error {
	return nil
}
func (p *persistencePublisherFake) PublishSourcecodeBlacklistContractAdd(context.Context, appstore.SourcecodeBlacklistContract) error {
	return nil
}
func (p *persistencePublisherFake) PublishSourcecodeBlacklistContractUpdateNote(context.Context, common.Address, string) error {
	return nil
}
func (p *persistencePublisherFake) PublishSourcecodeBlacklistContractDelete(context.Context, common.Address) error {
	return nil
}
func (p *persistencePublisherFake) PublishWalletBlacklistAdd(context.Context, appstore.WalletBlacklistEntry) error {
	return nil
}
func (p *persistencePublisherFake) PublishWalletBlacklistUpdateNote(context.Context, common.Address, string) error {
	return nil
}
func (p *persistencePublisherFake) PublishWalletBlacklistDelete(context.Context, common.Address) error {
	return nil
}

func TestProjectStateReconcilerJobIntervals(t *testing.T) {
	reconciler := &projectStateReconcilerImpl{}
	jobs := reconciler.reconcilerJobs()

	intervals := make(map[string]time.Duration, len(jobs))
	for _, job := range jobs {
		if strings.Contains(job.name, "_archived") {
			t.Fatalf("unexpected archived reconciler job %q", job.name)
		}
		intervals[job.name] = job.interval
	}

	assertJobInterval(t, intervals, "state_refresh_active", activeProjectStateRefreshInterval)
	assertJobInterval(t, intervals, "simulation_refresh_active", activeProjectSimulationRefreshInterval)
	assertJobInterval(t, intervals, "sourcecode_refresh_active", 10*time.Second)
	assertJobInterval(t, intervals, "source_quality_refresh_active", sourceCodeRefreshInterval)
	assertJobInterval(t, intervals, "code_bin_hash_refresh_active", sourceCodeRefreshInterval)
	assertJobInterval(t, intervals, "creator_other_projects_refresh_active", sourceCodeRefreshInterval)
}

func assertJobInterval(t *testing.T, intervals map[string]time.Duration, name string, want time.Duration) {
	t.Helper()

	got, ok := intervals[name]
	if !ok {
		t.Fatalf("job %q not found", name)
	}
	if got != want {
		t.Fatalf("job %q interval = %s, want %s", name, got, want)
	}
}

func TestProjectStateReconcilerReconcileOnceRunsJobsInOrderAndReturnsError(t *testing.T) {
	wantErr := errors.New("stop")
	var got []string
	reconciler := &projectStateReconcilerImpl{
		jobSem: make(chan struct{}, 1),
	}
	jobs := []reconcilerJob{
		{name: "first", run: func(context.Context) error {
			got = append(got, "first")
			return nil
		}},
		{name: "second", run: func(context.Context) error {
			got = append(got, "second")
			return wantErr
		}},
		{name: "third", run: func(context.Context) error {
			got = append(got, "third")
			return nil
		}},
	}

	err := reconciler.reconcileOnceJobs(context.Background(), jobs)
	if !errors.Is(err, wantErr) {
		t.Fatalf("ReconcileOnce error = %v, want %v", err, wantErr)
	}
	if len(got) != 2 || got[0] != "first" || got[1] != "second" {
		t.Fatalf("job order = %v, want [first second]", got)
	}
}

func TestProjectStateReconcilerReconcileOnceSuppressesPolicyTriggers(t *testing.T) {
	triggerCh := make(chan common.Address, 1)
	contract := common.HexToAddress("0x00000000000000000000000000000000000000a1")
	reconciler := &projectStateReconcilerImpl{
		policyTriggerCh: triggerCh,
		jobSem:          make(chan struct{}, 1),
	}

	err := reconciler.reconcileOnceJobs(context.Background(), []reconcilerJob{{
		name: "trigger",
		run: func(context.Context) error {
			reconciler.triggerPolicyEvaluation(contract, "test")
			return nil
		},
	}})
	if err != nil {
		t.Fatalf("ReconcileOnce: %v", err)
	}
	select {
	case got := <-triggerCh:
		t.Fatalf("unexpected policy trigger %s", got.Hex())
	default:
	}
}

func TestProjectStateReconcilerRefreshProjectSourceQualityReports(t *testing.T) {
	cache := newProjectSnapshotCacheTest(t)
	ctx := context.Background()
	contract := common.HexToAddress("0x00000000000000000000000000000000000000a1")
	if err := cache.SetProject(ctx, &Project{Meta: ProjectMeta{Contract: contract, SourceCode: "contract A {}"}}); err != nil {
		t.Fatalf("set project: %v", err)
	}

	analyzer := &sourceQualityAnalyzerFake{report: "## Report"}
	publisher := &persistencePublisherFake{}
	reconciler := &projectStateReconcilerImpl{
		projectCache:          cache,
		sourceQualityAnalyzer: analyzer,
		persistencePublisher:  publisher,
	}

	if err := reconciler.refreshProjectSourceQualityReports(ctx, refreshTargetActive); err != nil {
		t.Fatalf("refresh source quality reports: %v", err)
	}
	if analyzer.calls != 1 {
		t.Fatalf("analyzer calls = %d, want 1", analyzer.calls)
	}
	project, ok, err := cache.GetProject(ctx, contract)
	if err != nil {
		t.Fatalf("get project: %v", err)
	}
	if !ok || project.Meta.SourceQualityReport != "## Report" {
		t.Fatalf("source quality report = %q, want report", project.Meta.SourceQualityReport)
	}
	if project.Meta.SourceQualityReportedAt.IsZero() {
		t.Fatal("source quality reported at is zero")
	}
	if publisher.sourceQualityReports[contract] != "## Report" {
		t.Fatalf("persisted report = %q, want report", publisher.sourceQualityReports[contract])
	}
}

func TestProjectStateReconcilerRefreshProjectCodeBinHashesPersistsMetaHash(t *testing.T) {
	cache := newProjectSnapshotCacheTest(t)
	ctx := context.Background()
	contract := common.HexToAddress("0x00000000000000000000000000000000000000a3")
	if err := cache.SetProject(ctx, &Project{Meta: ProjectMeta{Contract: contract}}); err != nil {
		t.Fatalf("set project: %v", err)
	}

	code := []byte{0x60, 0x60, 0x60, 0x40}
	wantHash := crypto.Keccak256Hash(code)
	publisher := &persistencePublisherFake{}
	reconciler := &projectStateReconcilerImpl{
		projectCache:         cache,
		persistencePublisher: publisher,
		codeAtFunc: func(context.Context, common.Address) ([]byte, error) {
			return code, nil
		},
	}

	if err := reconciler.refreshProjectCodeBinHashes(ctx, refreshTargetActive); err != nil {
		t.Fatalf("refresh code bin hashes: %v", err)
	}
	project, ok, err := cache.GetProject(ctx, contract)
	if err != nil {
		t.Fatalf("get project: %v", err)
	}
	if !ok || project.Meta.CodeBinHash != wantHash {
		t.Fatalf("code bin hash = %s, want %s", project.Meta.CodeBinHash.Hex(), wantHash.Hex())
	}
	if publisher.codeBinHashes[contract] != wantHash {
		t.Fatalf("persisted code bin hash = %s, want %s", publisher.codeBinHashes[contract].Hex(), wantHash.Hex())
	}
}

func TestProjectStateReconcilerRefreshProjectSourceQualityReportsSkipsCompletedAndClosedSource(t *testing.T) {
	cache := newProjectSnapshotCacheTest(t)
	ctx := context.Background()
	closedSource := common.HexToAddress("0x00000000000000000000000000000000000000a1")
	completed := common.HexToAddress("0x00000000000000000000000000000000000000a2")
	if err := cache.SetProject(ctx, &Project{Meta: ProjectMeta{Contract: closedSource}}); err != nil {
		t.Fatalf("set closed source project: %v", err)
	}
	if err := cache.SetProject(ctx, &Project{Meta: ProjectMeta{Contract: completed, SourceCode: "contract A {}", SourceQualityReport: "existing"}}); err != nil {
		t.Fatalf("set completed project: %v", err)
	}

	analyzer := &sourceQualityAnalyzerFake{report: "## Report"}
	reconciler := &projectStateReconcilerImpl{
		projectCache:          cache,
		sourceQualityAnalyzer: analyzer,
		persistencePublisher:  &persistencePublisherFake{},
	}

	if err := reconciler.refreshProjectSourceQualityReports(ctx, refreshTargetActive); err != nil {
		t.Fatalf("refresh source quality reports: %v", err)
	}
	if analyzer.calls != 0 {
		t.Fatalf("analyzer calls = %d, want 0", analyzer.calls)
	}
}

func newProjectSnapshotCacheTest(t *testing.T) ProjectSnapshotCache {
	t.Helper()
	mini := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	return NewProjectSnapshotCache(redisport.NewGoRedisAdapter(client))
}
