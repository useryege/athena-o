package application

import (
	"container/heap"
	"context"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)

const (
	defaultProjectSyncWorkers              = 10
	defaultProjectSyncQueueCapacity        = 4096
	defaultProjectSyncInitialDelay         = 10 * time.Second
	defaultProjectSyncMaxDelay             = 1 * time.Minute
	defaultProjectSourceCodeMaxAttempts    = 999999999
	defaultProjectPairDiscoveryMaxAttempts = 999999999
	defaultProjectPairSnapshotInterval     = time.Minute
)

type ProjectSyncTaskKind string

const (
	sourceCodeTask    ProjectSyncTaskKind = "source_code"
	pairDiscoveryTask ProjectSyncTaskKind = "pair_discovery"
	pairSnapshotTask  ProjectSyncTaskKind = "pair_snapshot"
)

var ErrProjectSchedulerNotStarted = errors.New("project scheduler is not started")

type ProjectSchedulerOptions struct {
	WorkerCount              int
	QueueCapacity            int
	InitialDelay             time.Duration
	MaxDelay                 time.Duration
	SourceCodeMaxAttempts    int
	PairDiscoveryMaxAttempts int
	PairSnapshotInterval     time.Duration
}

type ProjectScheduler interface {
	Start(ctx context.Context) error
	Stop() error
	EnqueueProject(ctx context.Context, project *Project) error
}

var _ ProjectScheduler = &projectSchedulerImpl{}

type projectSchedulerImpl struct {
	registry ProjectRegistry
	syncer   ProjectSync
	opts     ProjectSchedulerOptions

	mu         sync.Mutex
	ctx        context.Context
	cancel     context.CancelFunc
	started    bool
	readyCh    chan ProjectSyncTask
	scheduleCh chan ProjectSyncTask
	wg         sync.WaitGroup
}

type ProjectSyncTask struct {
	projectID uuid.UUID
	kind      ProjectSyncTaskKind
	attempt   int
	nextRunAt time.Time
	index     int
}

func NewProjectScheduler(registry ProjectRegistry, syncer ProjectSync, opts ProjectSchedulerOptions) ProjectScheduler {
	opts = normalizeProjectSchedulerOptions(opts)
	return &projectSchedulerImpl{
		registry:   registry,
		syncer:     syncer,
		opts:       opts,
		readyCh:    make(chan ProjectSyncTask, opts.QueueCapacity),
		scheduleCh: make(chan ProjectSyncTask, opts.QueueCapacity),
	}
}

func normalizeProjectSchedulerOptions(opts ProjectSchedulerOptions) ProjectSchedulerOptions {
	if opts.WorkerCount <= 0 {
		opts.WorkerCount = defaultProjectSyncWorkers
	}
	if opts.QueueCapacity <= 0 {
		opts.QueueCapacity = defaultProjectSyncQueueCapacity
	}
	if opts.InitialDelay <= 0 {
		opts.InitialDelay = defaultProjectSyncInitialDelay
	}
	if opts.MaxDelay <= 0 {
		opts.MaxDelay = defaultProjectSyncMaxDelay
	}
	if opts.SourceCodeMaxAttempts <= 0 {
		opts.SourceCodeMaxAttempts = defaultProjectSourceCodeMaxAttempts
	}
	if opts.PairDiscoveryMaxAttempts <= 0 {
		opts.PairDiscoveryMaxAttempts = defaultProjectPairDiscoveryMaxAttempts
	}
	if opts.PairSnapshotInterval <= 0 {
		opts.PairSnapshotInterval = defaultProjectPairSnapshotInterval
	}
	return opts
}

func (s *projectSchedulerImpl) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.started {
		return nil
	}

	s.ctx, s.cancel = context.WithCancel(ctx)
	s.started = true

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.run()
	}()

	for i := 0; i < s.opts.WorkerCount; i++ {
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			s.worker()
		}()
	}
	return nil
}

func (s *projectSchedulerImpl) Stop() error {
	s.mu.Lock()
	if !s.started {
		s.mu.Unlock()
		return nil
	}
	cancel := s.cancel
	s.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	s.wg.Wait()

	s.mu.Lock()
	s.ctx = nil
	s.cancel = nil
	s.started = false
	s.mu.Unlock()
	return nil
}

func (s *projectSchedulerImpl) EnqueueProject(ctx context.Context, project *Project) error {
	if project == nil {
		return nil
	}

	projectID := project.Meta.ProjectID
	if err := s.schedule(ctx, ProjectSyncTask{
		projectID: projectID,
		kind:      sourceCodeTask,
		attempt:   1,
		nextRunAt: time.Now().Add(withJitter(s.opts.InitialDelay)),
	}); err != nil {
		return err
	}
	return s.schedule(ctx, ProjectSyncTask{
		projectID: projectID,
		kind:      pairDiscoveryTask,
		attempt:   1,
		nextRunAt: time.Now().Add(withJitter(s.opts.InitialDelay)),
	})
}

func (s *projectSchedulerImpl) schedule(ctx context.Context, task ProjectSyncTask) error {
	s.mu.Lock()
	schedulerCtx := s.ctx
	s.mu.Unlock()
	if schedulerCtx == nil {
		return ErrProjectSchedulerNotStarted
	}
	if task.nextRunAt.IsZero() {
		task.nextRunAt = time.Now()
	}

	select {
	case s.scheduleCh <- task:
		return nil
	case <-schedulerCtx.Done():
		return schedulerCtx.Err()
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *projectSchedulerImpl) run() {
	var tasks projectSyncTaskHeap
	heap.Init(&tasks)

	for {
		if tasks.Len() == 0 {
			select {
			case <-s.ctx.Done():
				return
			case task := <-s.scheduleCh:
				heap.Push(&tasks, task)
			}
			continue
		}

		next := tasks[0]
		delay := time.Until(next.nextRunAt)
		if delay <= 0 {
			task := heap.Pop(&tasks).(ProjectSyncTask)
			select {
			case s.readyCh <- task:
			case <-s.ctx.Done():
				return
			}
			continue
		}

		timer := time.NewTimer(delay)
		select {
		case <-s.ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return
		case task := <-s.scheduleCh:
			if !timer.Stop() {
				<-timer.C
			}
			heap.Push(&tasks, task)
		case <-timer.C:
		}
	}
}

func (s *projectSchedulerImpl) worker() {
	for {
		select {
		case <-s.ctx.Done():
			return
		case task := <-s.readyCh:
			s.handleTask(task)
		}
	}
}

func (s *projectSchedulerImpl) handleTask(task ProjectSyncTask) {
	project, ok, err := s.registry.GetProject(s.ctx, task.projectID)
	if err != nil {
		log.WithFields(log.Fields{
			"projectID": task.projectID,
			"task":      task.kind,
			"error":     err,
		}).Error("failed to get project for scheduled sync")
		return
	}
	if !ok {
		return
	}

	switch task.kind {
	case sourceCodeTask:
		s.handleSourceCodeTask(task, project)
	case pairDiscoveryTask:
		s.handlePairDiscoveryTask(task, project)
	case pairSnapshotTask:
		s.handlePairSnapshotTask(task, project)
	default:
		log.WithField("task", task.kind).Warn("unknown project sync task")
	}
}

func (s *projectSchedulerImpl) handleSourceCodeTask(task ProjectSyncTask, project *Project) {
	done, err := s.syncer.SyncSourceCodeOnce(s.ctx, project)
	if err != nil {
		log.WithFields(log.Fields{
			"projectID": project.Meta.ProjectID,
			"contract":  project.Meta.Contract,
			"attempt":   task.attempt,
			"error":     err,
		}).Warn("failed to sync project source code")
	}
	if done || task.attempt >= s.opts.SourceCodeMaxAttempts {
		return
	}
	s.reschedule(task, s.backoff(task.attempt))
}

func (s *projectSchedulerImpl) handlePairDiscoveryTask(task ProjectSyncTask, project *Project) {
	created, done, err := s.syncer.SyncPairDiscoveryOnce(s.ctx, project)
	if err != nil {
		log.WithFields(log.Fields{
			"projectID": project.Meta.ProjectID,
			"contract":  project.Meta.Contract,
			"attempt":   task.attempt,
			"error":     err,
		}).Warn("failed to discover project v2 pair")
	}
	if created {
		s.reschedule(ProjectSyncTask{
			projectID: project.Meta.ProjectID,
			kind:      pairSnapshotTask,
			attempt:   1,
		}, 0)
		return
	}
	if done || task.attempt >= s.opts.PairDiscoveryMaxAttempts {
		return
	}
	s.reschedule(task, s.backoff(task.attempt))
}

func (s *projectSchedulerImpl) handlePairSnapshotTask(task ProjectSyncTask, project *Project) {
	done, err := s.syncer.SyncPairSnapshotOnce(s.ctx, project)
	if err != nil {
		log.WithFields(log.Fields{
			"projectID": project.Meta.ProjectID,
			"contract":  project.Meta.Contract,
			"attempt":   task.attempt,
			"error":     err,
		}).Warn("failed to sync project v2 pair snapshot")
	}
	if done {
		return
	}
	s.reschedule(task, withJitter(s.opts.PairSnapshotInterval))
}

func (s *projectSchedulerImpl) reschedule(task ProjectSyncTask, delay time.Duration) {
	task.attempt++
	task.nextRunAt = time.Now().Add(delay)
	if err := s.schedule(s.ctx, task); err != nil && !errors.Is(err, context.Canceled) {
		log.WithFields(log.Fields{
			"projectID": task.projectID,
			"task":      task.kind,
			"error":     err,
		}).Warn("failed to reschedule project sync task")
	}
}

func (s *projectSchedulerImpl) backoff(attempt int) time.Duration {
	delay := s.opts.InitialDelay
	for i := 1; i < attempt; i++ {
		delay *= 2
		if delay >= s.opts.MaxDelay {
			return withJitter(s.opts.MaxDelay)
		}
	}
	return withJitter(delay)
}

type projectSyncTaskHeap []ProjectSyncTask

func (h projectSyncTaskHeap) Len() int {
	return len(h)
}

func (h projectSyncTaskHeap) Less(i, j int) bool {
	return h[i].nextRunAt.Before(h[j].nextRunAt)
}

func (h projectSyncTaskHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].index = i
	h[j].index = j
}

func (h *projectSyncTaskHeap) Push(x any) {
	task := x.(ProjectSyncTask)
	task.index = len(*h)
	*h = append(*h, task)
}

func (h *projectSyncTaskHeap) Pop() any {
	old := *h
	n := len(old)
	task := old[n-1]
	old[n-1] = ProjectSyncTask{}
	task.index = -1
	*h = old[:n-1]
	return task
}
