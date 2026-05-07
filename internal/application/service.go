package application

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
	applicationpkg "github.com/useryege/athena/internal/application/apiclient"
)

type Service struct {
	applicationpkg.UnimplementedApplicationServiceServer

	nodeClient *ethclient.Client

	projectCh     chan *Project
	blockWatcher  *BlockWatcher
	projectFilter *ProjectFilter

	registry ProjectRegistry
	eventHub *ProjectEventHub

	retryQueue     RetryUntilReadyQueue
	retryPool      RetryUntilReadyPool
	retryScheduler *RetryScheduler

	workerCount   int
	workerWG      sync.WaitGroup
	startStopMu   sync.Mutex
	lifecycleCtx  context.Context
	lifecycleStop context.CancelFunc
	started       bool
}

func NewService(nodeClient *ethclient.Client) *Service {
	registry := NewProjectRegistry()
	retryQueue := NewRetryUntilReadyQueue(1024)
	retryPool := NewRetryUntilReadyPool()
	retryScheduler := NewRetryScheduler(retryPool, retryQueue, 1*time.Minute, 100)

	return &Service{
		nodeClient:     nodeClient,
		registry:       registry,
		eventHub:       NewProjectEventHub(),
		retryQueue:     retryQueue,
		retryPool:      retryPool,
		retryScheduler: retryScheduler,
		workerCount:    10,
	}
}

func (s *Service) Start() error {
	s.startStopMu.Lock()
	defer s.startStopMu.Unlock()
	if s.started {
		return nil
	}

	s.ensureRetryRuntimeLocked()

	// channel 1 is used by block watcher and project filter
	ch1 := make(chan *Project, 24)
	s.projectCh = ch1
	s.blockWatcher = NewBlockWatcher(s.nodeClient, ch1)
	evmFetcher := NewEVMFetcher(s.nodeClient, s.registry)
	apiFetcher := NewAPIFetcher()
	s.projectFilter = NewProjectFilter(s.registry, ch1, s.retryPool, s.retryQueue, evmFetcher, s.eventHub)

	ctx, cancel := context.WithCancel(context.Background())
	if err := s.blockWatcher.Start(ctx); err != nil {
		cancel()
		close(ch1)
		s.clearPipelineLocked()
		return err
	}

	if err := s.projectFilter.Start(ctx); err != nil {
		cancel()
		_ = s.blockWatcher.Stop()
		close(ch1)
		s.clearPipelineLocked()
		return err
	}

	s.workerWG.Add(1)
	go func() {
		defer s.workerWG.Done()
		s.retryScheduler.Start(ctx)
	}()

	resolver := NewRetryUntilReadyResolver(s.registry, s.retryPool, evmFetcher, apiFetcher)
	for i := 0; i < s.workerCount; i++ {
		worker := NewRetryUntilReadyWorker(s.retryQueue, resolver)
		s.workerWG.Add(1)
		go func() {
			defer s.workerWG.Done()
			worker.Run(ctx)
		}()
	}

	s.lifecycleCtx = ctx
	s.lifecycleStop = cancel
	s.started = true
	return nil
}

func (s *Service) Stop() error {
	s.startStopMu.Lock()
	defer s.startStopMu.Unlock()

	if !s.started {
		return nil
	}

	stop := s.lifecycleStop

	if stop != nil {
		stop()
	}

	watcherErr := s.blockWatcher.Stop()
	if s.projectCh != nil {
		close(s.projectCh)
	}
	filterErr := s.projectFilter.Stop()
	s.workerWG.Wait()
	retryQueueErr := s.retryQueue.Close()

	s.lifecycleCtx = nil
	s.lifecycleStop = nil
	s.started = false
	s.clearPipelineLocked()

	return errors.Join(watcherErr, filterErr, retryQueueErr)
}

func (s *Service) ensureRetryRuntimeLocked() {
	if s.retryQueue != nil && !s.retryQueue.Stats().Closed {
		return
	}

	s.retryQueue = NewRetryUntilReadyQueue(1024)
	s.retryScheduler = NewRetryScheduler(s.retryPool, s.retryQueue, 1*time.Minute, 100)
}

func (s *Service) clearPipelineLocked() {
	s.projectCh = nil
	s.blockWatcher = nil
	s.projectFilter = nil
}

func (s *Service) ListProjects(ctx context.Context, _ *applicationpkg.ListProjectsRequest) (*applicationpkg.ListProjectsResponse, error) {
	startedAt := time.Now()
	projects, err := s.registry.ListProjects(ctx)
	projectSnapshotLatency.Observe(float64(time.Since(startedAt).Milliseconds()))
	if err != nil {
		return nil, err
	}

	return &applicationpkg.ListProjectsResponse{Items: projects}, nil
}

func (s *Service) WatchProjects(_ *applicationpkg.WatchProjectsRequest, stream applicationpkg.ApplicationService_WatchProjectsServer) error {
	events, unsubscribe := s.eventHub.Subscribe()
	defer unsubscribe()

	for {
		select {
		case <-stream.Context().Done():
			return stream.Context().Err()
		case event, ok := <-events:
			if !ok {
				return nil
			}
			resp := &applicationpkg.WatchProjectsResponse{Event: event}
			if err := stream.Send(resp); err != nil {
				return err
			}
		}
	}
}
