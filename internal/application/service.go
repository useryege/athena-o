package application

import (
	"context"
	"errors"
	"sync"

	"github.com/ethereum/go-ethereum/ethclient"
)

type Service struct {
	watcher        *Watcher
	projectFilter  *ProjectFilter
	projectManager *ProjectManager

	startStopMu   sync.Mutex
	lifecycleCtx  context.Context
	lifecycleStop context.CancelFunc
	started       bool
}

func NewService(nodeClient *ethclient.Client) *Service {

	watcherToProjectFilterCh := make(chan *Project, 512)
	projectFilterToProjectManagerCh := make(chan *Project, 512)

	registry := NewProjectRegistry()

	return &Service{
		watcher:        NewWatcher(nodeClient, watcherToProjectFilterCh),
		projectFilter:  NewProjectFilter(nodeClient, watcherToProjectFilterCh, projectFilterToProjectManagerCh),
		projectManager: NewProjectManager(nodeClient, projectFilterToProjectManagerCh, registry),
	}
}

func (s *Service) Start() error {
	s.startStopMu.Lock()
	defer s.startStopMu.Unlock()
	if s.started {
		return nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	if err := s.watcher.Start(ctx); err != nil {
		cancel()
		return err
	}
	if err := s.projectFilter.Start(ctx); err != nil {
		cancel()
		_ = s.watcher.Stop()
		return err
	}
	if err := s.projectManager.Start(ctx); err != nil {
		cancel()
		_ = s.projectFilter.Stop()
		_ = s.watcher.Stop()
		return err
	}

	s.lifecycleCtx = ctx
	s.lifecycleStop = cancel
	s.started = true
	return nil
}

func (s *Service) Stop() error {
	s.startStopMu.Lock()
	if !s.started {
		s.startStopMu.Unlock()
		return nil
	}
	stop := s.lifecycleStop
	s.lifecycleCtx = nil
	s.lifecycleStop = nil
	s.started = false
	s.startStopMu.Unlock()

	if stop != nil {
		stop()
	}

	watcherErr := s.watcher.Stop()
	projectFilterErr := s.projectFilter.Stop()
	projectManagerErr := s.projectManager.Stop()
	return errors.Join(watcherErr, projectFilterErr, projectManagerErr)
}
