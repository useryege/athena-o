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

	delayedFetchSem chan struct{}
	startStopMu     sync.Mutex
	lifecycleCtx    context.Context
	lifecycleStop   context.CancelFunc
	started         bool
}

func NewService(nodeClient *ethclient.Client) *Service {
	registry := NewProjectRegistry()

	return &Service{
		nodeClient:      nodeClient,
		registry:        registry,
		delayedFetchSem: make(chan struct{}, defaultDelayedFetchConcurrency),
	}
}

func (s *Service) Start() error {
	s.startStopMu.Lock()
	defer s.startStopMu.Unlock()
	if s.started {
		return nil
	}

	// channel 1 is used by block watcher and project filter
	ch1 := make(chan *Project, 24)
	s.projectCh = ch1
	s.blockWatcher = NewBlockWatcher(s.nodeClient, ch1)
	evmFetcher := NewEVMFetcher(s.nodeClient, s.registry)
	apiFetcher := NewAPIFetcher()
	s.projectFilter = NewProjectFilter(s.registry, ch1, evmFetcher, apiFetcher, s.delayedFetchSem)

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

	s.lifecycleCtx = nil
	s.lifecycleStop = nil
	s.started = false
	s.clearPipelineLocked()

	return errors.Join(watcherErr, filterErr)
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
