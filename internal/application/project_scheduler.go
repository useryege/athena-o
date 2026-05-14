package application

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	appcache "github.com/useryege/athena/internal/application/cache"
	"github.com/useryege/athena/internal/application/sourcecode"
)

const (
	defaultProjectSourceCodeInterval = time.Minute
	sourceCodeProjectQueueCapacity   = 1024
)

var ErrProjectSchedulerNotStarted = errors.New("project scheduler is not started")

type ProjectSchedulerOptions struct {
	SourceCodeInterval time.Duration
}

type ProjectScheduler interface {
	Start(ctx context.Context) error
	Stop() error
	EnqueueProject(ctx context.Context, project *Project) error
}

var _ ProjectScheduler = &projectSchedulerImpl{}

type projectSchedulerImpl struct {
	registry  ProjectRegistry
	syncer    ProjectSync
	analyzer  sourcecode.Analyzer
	blacklist appcache.SourceCodeBlacklistModel
	opts      ProjectSchedulerOptions

	mu              sync.Mutex
	ctx             context.Context
	cancel          context.CancelFunc
	started         bool
	sourceProjectCh chan uuid.UUID
	wg              sync.WaitGroup
}

func NewProjectScheduler(registry ProjectRegistry, syncer ProjectSync, analyzer sourcecode.Analyzer, blacklist appcache.SourceCodeBlacklistModel, opts ProjectSchedulerOptions) ProjectScheduler {
	opts = normalizeProjectSchedulerOptions(opts)
	return &projectSchedulerImpl{
		registry:        registry,
		syncer:          syncer,
		analyzer:        analyzer,
		blacklist:       blacklist,
		opts:            opts,
		sourceProjectCh: make(chan uuid.UUID, sourceCodeProjectQueueCapacity),
	}
}

func normalizeProjectSchedulerOptions(opts ProjectSchedulerOptions) ProjectSchedulerOptions {
	if opts.SourceCodeInterval <= 0 {
		opts.SourceCodeInterval = defaultProjectSourceCodeInterval
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
		s.sourceCodeLoop()
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

	return nil
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
	sourceCodeUpdateErr := s.registry.UpdateProjectMetaState(s.ctx, project.Meta.ProjectID, &project.Meta)
	if sourceCodeUpdateErr != nil && !errors.Is(sourceCodeUpdateErr, context.Canceled) {
		log.WithFields(log.Fields{
			"projectID": project.Meta.ProjectID,
			"error":     sourceCodeUpdateErr,
		}).Warn("failed to update project source code state")
	}
	if done && sourceCodeUpdateErr == nil {
		s.analyzeProjectSourceCode(project)
		// log.WithFields(log.Fields{
		// 	"projectID": project.Meta.ProjectID,
		// 	"contract":  project.Meta.Contract,
		// }).Debug("project source code sync completed")
	}
}

func (s *projectSchedulerImpl) analyzeProjectSourceCode(project *Project) {
	if s.analyzer == nil || project == nil {
		return
	}
	if project.Meta.SourceCode == "" || project.Meta.SourceCodeBlacklist.IsReady() {
		return
	}

	meta := cloneProjectMeta(project.Meta)
	sourceCode := project.Meta.SourceCode
	fields, err := s.sourceCodeBlacklistFields()
	if err != nil {
		meta.SourceCodeBlacklist.MarkFailed(err, time.Now())
	} else {
		meta.SourceCodeBlacklist.MarkReady(s.analyzer.AnalyzeSourceCode(sourceCode, fields), time.Now())
	}
	if updateErr := s.registry.UpdateProjectMetaState(s.ctx, project.Meta.ProjectID, &meta); updateErr != nil && !errors.Is(updateErr, context.Canceled) {
		log.WithFields(log.Fields{
			"projectID": project.Meta.ProjectID,
			"error":     updateErr,
		}).Warn("failed to update project source code analysis state")
	}
}

func (s *projectSchedulerImpl) sourceCodeBlacklistFields() ([]string, error) {
	if s.blacklist == nil {
		return nil, nil
	}
	return s.blacklist.List(s.ctx)
}
