package application

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/google/uuid"
	"github.com/useryege/athena/pkg/apis/application/v1alpha1"
)

type fakeProjectRegistry struct {
	mu       sync.RWMutex
	projects map[uuid.UUID]*Project
}

func newFakeProjectRegistry(projects ...*Project) *fakeProjectRegistry {
	registry := &fakeProjectRegistry{projects: make(map[uuid.UUID]*Project, len(projects))}
	for _, project := range projects {
		registry.projects[project.Meta.ProjectID] = project
	}
	return registry
}

func (r *fakeProjectRegistry) GetProject(ctx context.Context, projectID uuid.UUID) (*Project, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	project, ok := r.projects[projectID]
	return project, ok, nil
}

func (r *fakeProjectRegistry) ListProjects(ctx context.Context) ([]*v1alpha1.ProjectView, error) {
	return nil, nil
}

func (r *fakeProjectRegistry) SetProject(ctx context.Context, projectID uuid.UUID, project *Project) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.projects[projectID] = project
	return nil
}

func (r *fakeProjectRegistry) RemoveProject(ctx context.Context, projectID uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.projects, projectID)
	return nil
}

type fakeProjectSyncer struct {
	sourceCodeFunc    func(context.Context, *Project) (bool, error)
	pairDiscoveryFunc func(context.Context, *Project) (bool, bool, error)
	pairSnapshotFunc  func(context.Context, *Project) (bool, error)
}

var _ ProjectSync = &fakeProjectSyncer{}

func (s *fakeProjectSyncer) SyncSourceCodeOnce(ctx context.Context, project *Project) (bool, error) {
	if s.sourceCodeFunc == nil {
		return true, nil
	}
	return s.sourceCodeFunc(ctx, project)
}

func (s *fakeProjectSyncer) SyncPairDiscoveryOnce(ctx context.Context, project *Project) (bool, bool, error) {
	if s.pairDiscoveryFunc == nil {
		return false, true, nil
	}
	return s.pairDiscoveryFunc(ctx, project)
}

func (s *fakeProjectSyncer) SyncPairSnapshotOnce(ctx context.Context, project *Project) (bool, error) {
	if s.pairSnapshotFunc == nil {
		return true, nil
	}
	return s.pairSnapshotFunc(ctx, project)
}

func TestProjectSchedulerStopsSourceCodeAfterMaxAttempts(t *testing.T) {
	project := newSchedulerTestProject()
	registry := newFakeProjectRegistry(project)
	var sourceCalls atomic.Int32
	syncer := &fakeProjectSyncer{
		sourceCodeFunc: func(context.Context, *Project) (bool, error) {
			sourceCalls.Add(1)
			return false, errors.New("temporary etherscan failure")
		},
	}
	scheduler := NewProjectScheduler(registry, syncer, ProjectSchedulerOptions{
		WorkerCount:           1,
		QueueCapacity:         16,
		InitialDelay:          time.Millisecond,
		MaxDelay:              time.Millisecond,
		SourceCodeMaxAttempts: 3,
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := scheduler.Start(ctx); err != nil {
		t.Fatalf("start scheduler: %v", err)
	}
	if err := scheduler.EnqueueProject(ctx, project); err != nil {
		t.Fatalf("enqueue project: %v", err)
	}

	waitForCondition(t, func() bool {
		return sourceCalls.Load() == 3
	})
	time.Sleep(10 * time.Millisecond)
	if got := sourceCalls.Load(); got != 3 {
		t.Fatalf("source code calls = %d, want 3", got)
	}
	if err := scheduler.Stop(); err != nil {
		t.Fatalf("stop scheduler: %v", err)
	}
}

func TestProjectSchedulerSchedulesSnapshotAfterPairDiscovery(t *testing.T) {
	project := newSchedulerTestProject()
	registry := newFakeProjectRegistry(project)
	var snapshotCalls atomic.Int32
	syncer := &fakeProjectSyncer{
		pairDiscoveryFunc: func(context.Context, *Project) (bool, bool, error) {
			return true, true, nil
		},
		pairSnapshotFunc: func(context.Context, *Project) (bool, error) {
			snapshotCalls.Add(1)
			return true, nil
		},
	}
	scheduler := NewProjectScheduler(registry, syncer, ProjectSchedulerOptions{
		WorkerCount:   1,
		QueueCapacity: 16,
		InitialDelay:  time.Millisecond,
		MaxDelay:      time.Millisecond,
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := scheduler.Start(ctx); err != nil {
		t.Fatalf("start scheduler: %v", err)
	}
	if err := scheduler.EnqueueProject(ctx, project); err != nil {
		t.Fatalf("enqueue project: %v", err)
	}

	waitForCondition(t, func() bool {
		return snapshotCalls.Load() > 0
	})
	if err := scheduler.Stop(); err != nil {
		t.Fatalf("stop scheduler: %v", err)
	}
}

func TestProjectSchedulerRetriesPairDiscoveryBeforeSnapshot(t *testing.T) {
	project := newSchedulerTestProject()
	registry := newFakeProjectRegistry(project)
	var discoveryCalls atomic.Int32
	var snapshotCalls atomic.Int32
	syncer := &fakeProjectSyncer{
		pairDiscoveryFunc: func(context.Context, *Project) (bool, bool, error) {
			if discoveryCalls.Add(1) < 3 {
				return false, false, nil
			}
			return true, true, nil
		},
		pairSnapshotFunc: func(context.Context, *Project) (bool, error) {
			snapshotCalls.Add(1)
			return true, nil
		},
	}
	scheduler := NewProjectScheduler(registry, syncer, ProjectSchedulerOptions{
		WorkerCount:              1,
		QueueCapacity:            16,
		InitialDelay:             time.Millisecond,
		MaxDelay:                 time.Millisecond,
		PairDiscoveryMaxAttempts: 4,
		PairSnapshotInterval:     time.Millisecond,
		SourceCodeMaxAttempts:    1,
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := scheduler.Start(ctx); err != nil {
		t.Fatalf("start scheduler: %v", err)
	}
	if err := scheduler.EnqueueProject(ctx, project); err != nil {
		t.Fatalf("enqueue project: %v", err)
	}

	waitForCondition(t, func() bool {
		return discoveryCalls.Load() == 3 && snapshotCalls.Load() > 0
	})
	if err := scheduler.Stop(); err != nil {
		t.Fatalf("stop scheduler: %v", err)
	}
}

func TestProjectSchedulerStopExits(t *testing.T) {
	project := newSchedulerTestProject()
	registry := newFakeProjectRegistry(project)
	scheduler := NewProjectScheduler(registry, &fakeProjectSyncer{}, ProjectSchedulerOptions{
		WorkerCount:   2,
		QueueCapacity: 16,
		InitialDelay:  time.Hour,
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := scheduler.Start(ctx); err != nil {
		t.Fatalf("start scheduler: %v", err)
	}
	if err := scheduler.EnqueueProject(ctx, project); err != nil {
		t.Fatalf("enqueue project: %v", err)
	}

	done := make(chan error, 1)
	go func() {
		done <- scheduler.Stop()
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("stop scheduler: %v", err)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("scheduler stop timed out")
	}
}

func newSchedulerTestProject() *Project {
	return &Project{
		Meta: ProjectMeta{
			ProjectID: uuid.New(),
			Contract:  common.HexToAddress("0x0000000000000000000000000000000000000001"),
		},
	}
}

func waitForCondition(t *testing.T, condition func() bool) {
	t.Helper()

	deadline := time.After(500 * time.Millisecond)
	ticker := time.NewTicker(time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-deadline:
			t.Fatal("condition was not met before timeout")
		case <-ticker.C:
			if condition() {
				return
			}
		}
	}
}
