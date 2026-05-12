package application

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)

const (
	defaultProjectSourceCodeInterval  = time.Minute
	defaultBlockRefreshQueueCapacity  = 16
	defaultInitialStateRefreshTimeout = 10 * time.Second
	initialStateRefreshBlockNumber    = uint64(0)
	sourceCodeProjectQueueCapacity    = 1024
)

var ErrProjectSchedulerNotStarted = errors.New("project scheduler is not started")

type ProjectSchedulerOptions struct {
	SourceCodeInterval         time.Duration
	BlockRefreshQueueCapacity  int
	InitialStateRefreshTimeout time.Duration
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
	blockCh  <-chan uint64
	opts     ProjectSchedulerOptions

	mu              sync.Mutex
	ctx             context.Context
	cancel          context.CancelFunc
	started         bool
	sourceProjectCh chan uuid.UUID
	refreshCh       chan uint64
	wg              sync.WaitGroup
}

func NewProjectScheduler(registry ProjectRegistry, syncer ProjectSync, blockCh <-chan uint64, opts ProjectSchedulerOptions) ProjectScheduler {
	opts = normalizeProjectSchedulerOptions(opts)
	return &projectSchedulerImpl{
		registry:        registry,
		syncer:          syncer,
		blockCh:         blockCh,
		opts:            opts,
		sourceProjectCh: make(chan uuid.UUID, sourceCodeProjectQueueCapacity),
		refreshCh:       make(chan uint64, opts.BlockRefreshQueueCapacity),
	}
}

func normalizeProjectSchedulerOptions(opts ProjectSchedulerOptions) ProjectSchedulerOptions {
	if opts.SourceCodeInterval <= 0 {
		opts.SourceCodeInterval = defaultProjectSourceCodeInterval
	}
	if opts.BlockRefreshQueueCapacity <= 0 {
		opts.BlockRefreshQueueCapacity = defaultBlockRefreshQueueCapacity
	}
	if opts.InitialStateRefreshTimeout <= 0 {
		opts.InitialStateRefreshTimeout = defaultInitialStateRefreshTimeout
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

	s.wg.Add(2)
	go func() {
		defer s.wg.Done()
		s.sourceCodeLoop()
	}()
	go func() {
		defer s.wg.Done()
		s.blockRefreshLoop()
	}()
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

	s.mu.Lock()
	schedulerCtx := s.ctx
	s.mu.Unlock()
	if schedulerCtx == nil {
		return ErrProjectSchedulerNotStarted
	}

	select {
	case s.sourceProjectCh <- project.Meta.ProjectID:
	case <-schedulerCtx.Done():
		return schedulerCtx.Err()
	case <-ctx.Done():
		return ctx.Err()
	}

	return s.requestStateRefresh(ctx, initialStateRefreshBlockNumber)
}

func (s *projectSchedulerImpl) requestStateRefresh(ctx context.Context, blockNumber uint64) error {
	s.mu.Lock()
	schedulerCtx := s.ctx
	s.mu.Unlock()
	if schedulerCtx == nil {
		return ErrProjectSchedulerNotStarted
	}

	select {
	case s.refreshCh <- blockNumber:
		return nil
	case <-schedulerCtx.Done():
		return schedulerCtx.Err()
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *projectSchedulerImpl) sourceCodeLoop() {
	ticker := time.NewTicker(s.opts.SourceCodeInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case projectID := <-s.sourceProjectCh:
			s.syncProjectSourceCode(projectID)
		case <-ticker.C:
			s.syncAllProjectSourceCode()
		}
	}
}

func (s *projectSchedulerImpl) syncAllProjectSourceCode() {
	projects, err := s.registry.ListProjects(s.ctx)
	if err != nil {
		if !errors.Is(err, context.Canceled) {
			log.WithError(err).Warn("failed to list projects for source code sync")
		}
		return
	}
	for _, project := range projects {
		if err := s.ctx.Err(); err != nil {
			return
		}
		s.syncProjectSourceCode(project.Meta.ProjectID)
	}
}

func (s *projectSchedulerImpl) syncProjectSourceCode(projectID uuid.UUID) {
	project, ok, err := s.registry.GetProject(s.ctx, projectID)
	if err != nil {
		if !errors.Is(err, context.Canceled) {
			log.WithFields(log.Fields{
				"projectID": projectID,
				"error":     err,
			}).Warn("failed to get project for source code sync")
		}
		return
	}
	if !ok {
		return
	}

	done, err := s.syncer.SyncSourceCodeOnce(s.ctx, project)
	if err != nil {
		log.WithFields(log.Fields{
			"projectID": project.Meta.ProjectID,
			"contract":  project.Meta.Contract,
			"error":     err,
		}).Warn("failed to sync project source code")
	}
	if updateErr := s.registry.UpdateProjectSourceCodeState(s.ctx, project.Meta.ProjectID, &project.SourceCode); updateErr != nil && !errors.Is(updateErr, context.Canceled) {
		log.WithFields(log.Fields{
			"projectID": project.Meta.ProjectID,
			"error":     updateErr,
		}).Warn("failed to update project source code state")
	}
	if done {
		log.WithFields(log.Fields{
			"projectID": project.Meta.ProjectID,
			"contract":  project.Meta.Contract,
		}).Debug("project source code sync completed")
	}
}

func (s *projectSchedulerImpl) blockRefreshLoop() {
	doneCh := make(chan error, 1)
	blockCh := s.blockCh
	refreshing := false

	for {
		select {
		case <-s.ctx.Done():
			return
		case blockNumber, ok := <-blockCh:
			if !ok {
				blockCh = nil
				continue
			}
			refreshing = s.handleBlockRefreshTrigger(blockNumber, refreshing, doneCh)
		case blockNumber := <-s.refreshCh:
			refreshing = s.handleBlockRefreshTrigger(blockNumber, refreshing, doneCh)
		case err := <-doneCh:
			refreshing = false
			if err != nil && !errors.Is(err, context.Canceled) {
				log.WithError(err).Warn("failed to refresh project chain states")
			}
		}
	}
}

func (s *projectSchedulerImpl) handleBlockRefreshTrigger(blockNumber uint64, refreshing bool, doneCh chan<- error) bool {
	if refreshing {
		log.WithField("blockNumber", blockNumber).Debug("dropping project state refresh trigger while refresh is running")
		return true
	}
	s.startProjectStateRefresh(blockNumber, doneCh)
	return true
}

func (s *projectSchedulerImpl) startProjectStateRefresh(blockNumber uint64, doneCh chan<- error) {
	refreshCtx := s.ctx
	cancel := func() {}
	if blockNumber == initialStateRefreshBlockNumber && s.opts.InitialStateRefreshTimeout > 0 {
		refreshCtx, cancel = context.WithTimeout(s.ctx, s.opts.InitialStateRefreshTimeout)
	}

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		defer cancel()
		err := s.syncer.SyncProjectStatesOnce(refreshCtx)
		select {
		case doneCh <- err:
		case <-s.ctx.Done():
		}
	}()
}
