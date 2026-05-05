package application

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
)

type Service struct {
	watcher        *Watcher
	projectFilter  *ProjectFilter
	projectManager *ProjectManager

	registry  ProjectRegistry
	queue     StaticFieldQueue
	resolver  StaticFieldResolver
	scheduler StaticFieldRetryScheduler

	workerCount int
	workerWG    sync.WaitGroup

	startStopMu   sync.Mutex
	lifecycleCtx  context.Context
	lifecycleStop context.CancelFunc
	started       bool
}

func NewService(nodeClient *ethclient.Client) *Service {

	watcherToProjectFilterCh := make(chan *Project, 512)
	projectFilterToProjectManagerCh := make(chan *Project, 512)

	registry := NewProjectRegistry()
	queue := NewStaticFieldQueue(2048)
	resolver := NewStaticFieldResolver(NewStaticFieldFetcher(), registry, nil)
	scheduler := NewStaticFieldRetryScheduler(registry, queue, 3*time.Second, 500)

	return &Service{
		watcher:        NewWatcher(nodeClient, watcherToProjectFilterCh),
		projectFilter:  NewProjectFilter(nodeClient, watcherToProjectFilterCh, projectFilterToProjectManagerCh),
		projectManager: NewProjectManager(nodeClient, projectFilterToProjectManagerCh, registry, queue),
		registry:       registry,
		queue:          queue,
		resolver:       resolver,
		scheduler:      scheduler,
		workerCount:    4,
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

	for i := 0; i < s.workerCount; i++ {
		worker := NewStaticFieldWorker(s.queue, s.resolver)
		s.workerWG.Add(1)
		go func() {
			defer s.workerWG.Done()
			worker.Run(ctx)
		}()
	}
	s.workerWG.Add(1)
	go func() {
		defer s.workerWG.Done()
		s.scheduler.Run(ctx)
	}()

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
	queueErr := s.queue.Close()
	s.workerWG.Wait()
	return errors.Join(watcherErr, projectFilterErr, projectManagerErr, queueErr)
}
