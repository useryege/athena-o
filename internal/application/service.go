package application

import (
	"context"
	"errors"
	"sync"

	"github.com/ethereum/go-ethereum/ethclient"
)

type Service struct {
	blockWatcher  *BlockWatcher
	projectFilter *ProjectFilter

	registry ProjectRegistry
	queue    StaticFieldQueue
	// resolver  StaticFieldResolver
	// scheduler StaticFieldRetryScheduler

	workerCount int
	workerWG    sync.WaitGroup

	startStopMu   sync.Mutex
	lifecycleCtx  context.Context
	lifecycleStop context.CancelFunc
	started       bool
}

func NewService(nodeClient *ethclient.Client) *Service {
	// channel 1 is used by block watcher and project filter
	ch1 := make(chan *Project, 24)

	registry := NewProjectRegistry()

	queue := NewStaticFieldQueue(2048)

	return &Service{
		blockWatcher:  NewBlockWatcher(nodeClient, ch1),
		projectFilter: NewProjectFilter(nodeClient, registry, ch1),
		registry:      registry,
		queue:         queue,
		// resolver:       nil,
		// scheduler:      nil,
		workerCount: 4,
	}
}

func (s *Service) Start() error {
	s.startStopMu.Lock()
	defer s.startStopMu.Unlock()
	if s.started {
		return nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	if err := s.blockWatcher.Start(ctx); err != nil {
		cancel()
		return err
	}

	if err := s.projectFilter.Start(ctx); err != nil {
		cancel()
		_ = s.blockWatcher.Stop()
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

	watcherErr := s.blockWatcher.Stop()
	filterErr := s.projectFilter.Stop()
	queueErr := s.queue.Close()
	s.workerWG.Wait()
	return errors.Join(watcherErr, filterErr, queueErr)
}
