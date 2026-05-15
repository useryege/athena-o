package application

import (
	"context"
	"errors"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	appcache "github.com/useryege/athena/internal/application/cache"
	"github.com/useryege/athena/internal/application/sourcecode"
)

const (
	defaultProjectSourceCodeInterval = time.Minute
)

type ProjectSchedulerOptions struct {
	SourceCodeInterval time.Duration
}

type ProjectScheduler interface {
	Start(ctx context.Context) error
	Stop() error
}

var _ ProjectScheduler = &projectSchedulerImpl{}

type projectSchedulerImpl struct {
	registry  ProjectRegistry
	syncer    ProjectSync
	analyzer  sourcecode.Analyzer
	blacklist appcache.SourceCodeBlacklistModel
	opts      ProjectSchedulerOptions

	mu      sync.Mutex
	ctx     context.Context
	cancel  context.CancelFunc
	started bool
	wg      sync.WaitGroup
}

func NewProjectScheduler(registry ProjectRegistry, syncer ProjectSync, analyzer sourcecode.Analyzer, blacklist appcache.SourceCodeBlacklistModel, opts ProjectSchedulerOptions) ProjectScheduler {
	opts = normalizeProjectSchedulerOptions(opts)
	return &projectSchedulerImpl{
		registry:  registry,
		syncer:    syncer,
		analyzer:  analyzer,
		blacklist: blacklist,
		opts:      opts,
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

func (s *projectSchedulerImpl) sourceCodeLoop() {
	ticker := time.NewTicker(s.opts.SourceCodeInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
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
		s.syncProjectSourceCode(project)
	}
}

func (s *projectSchedulerImpl) syncProjectSourceCode(project *Project) {
	if project == nil {
		return
	}

	if project.Meta.SourceCode == "" {
		if _, err := s.syncer.SyncSourceCodeOnce(s.ctx, project); err != nil {
			log.WithFields(log.Fields{
				"projectID": project.Meta.ProjectID,
				"contract":  project.Meta.Contract,
				"error":     err,
			}).Warn("failed to sync project source code")
			return
		}
		if err := s.registry.UpdateProjectMetaState(s.ctx, project.Meta.ProjectID, &project.Meta); err != nil {
			if !errors.Is(err, context.Canceled) {
				log.WithFields(log.Fields{
					"projectID": project.Meta.ProjectID,
					"error":     err,
				}).Warn("failed to update project source code state")
			}
			return
		}
	}

	if project.Meta.SourceCode != "" {
		s.analyzeProjectSourceCode(project)
	}
}

func (s *projectSchedulerImpl) analyzeProjectSourceCode(project *Project) {
	if s.analyzer == nil || project == nil {
		return
	}
	if project.Meta.SourceCode == "" || !project.Meta.SourceCodeBlacklist.ResolvedAt.IsZero() {
		return
	}

	meta := cloneProjectMeta(project.Meta)
	sourceCode := project.Meta.SourceCode
	fields, err := s.sourceCodeBlacklistFields()
	if err != nil {
		log.WithFields(log.Fields{
			"projectID": project.Meta.ProjectID,
			"error":     err,
		}).Warn("failed to load source code blacklist fields")
		return
	}
	meta.SourceCodeBlacklist = s.analyzer.AnalyzeSourceCode(sourceCode, fields)
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
