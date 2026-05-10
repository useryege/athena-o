package application

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/google/uuid"
	applicationpkg "github.com/useryege/athena/internal/application/apiclient"
	"github.com/useryege/athena/internal/application/evm"
	"github.com/useryege/athena/util/ethereumapi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Service struct {
	applicationpkg.UnimplementedApplicationServiceServer

	nodeClient        *ethclient.Client
	v2FactoryContract common.Address
	wethContract      common.Address

	etherscanAPIBaseURL string
	etherscanAPIKey     string

	projectCh     chan *Project
	blockWatcher  *BlockWatcher
	projectFilter *ProjectFilter

	registry     ProjectRegistry
	projectStore ProjectStore

	delayedFetchSem chan struct{}
	startStopMu     sync.Mutex
	lifecycleCtx    context.Context
	lifecycleStop   context.CancelFunc
	started         bool
}

func NewService(nodeClient *ethclient.Client, v2FactoryContract common.Address, wethContract common.Address, etherscanAPIBaseURL string, etherscanAPIKey string, projectStore ProjectStore) *Service {
	registry := NewProjectRegistry(projectStore)

	return &Service{
		nodeClient:          nodeClient,
		registry:            registry,
		projectStore:        projectStore,
		delayedFetchSem:     make(chan struct{}, defaultDelayedFetchConcurrency),
		v2FactoryContract:   v2FactoryContract,
		wethContract:        wethContract,
		etherscanAPIBaseURL: etherscanAPIBaseURL,
		etherscanAPIKey:     etherscanAPIKey,
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
	evmFetcher := evm.NewEVMFetcher(s.nodeClient, s.v2FactoryContract)

	chainID, err := s.nodeClient.ChainID(context.Background())
	if err != nil {
		return err
	}

	apiFetcher := ethereumapi.NewEthereumAPI(s.etherscanAPIBaseURL, s.etherscanAPIKey, chainID.Int64())

	s.projectFilter = NewProjectFilter(s.registry, ch1, evmFetcher, apiFetcher, s.delayedFetchSem, s.wethContract)

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

func (s *Service) GetProject(ctx context.Context, req *applicationpkg.GetProjectRequest) (*applicationpkg.GetProjectResponse, error) {
	projectID, err := uuid.Parse(req.GetProjectID())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid projectID %q: %v", req.GetProjectID(), err)
	}

	startedAt := time.Now()
	project, ok, err := s.registry.GetProject(ctx, projectID)
	projectSnapshotLatency.Observe(float64(time.Since(startedAt).Milliseconds()))
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, status.Errorf(codes.NotFound, "project %q not found", req.GetProjectID())
	}

	return &applicationpkg.GetProjectResponse{Item: projectToView(project)}, nil
}
