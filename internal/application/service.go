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
	"github.com/useryege/athena/pkg/apis/application/v1alpha1"
	"github.com/useryege/athena/util/ethereumapi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Service struct {
	applicationpkg.UnimplementedApplicationServiceServer

	nodeClient        *ethclient.Client
	v2FactoryContract common.Address
	wethContract      common.Address
	athenaContract    common.Address

	etherscanAPIBaseURL string
	etherscanAPIKey     string
	liquidityLocker     []common.Address

	projectCh       chan *Project
	blockWatcher    *BlockWatcher
	blockSubscriber *BlockEventSubscriber
	projectFilter   *ProjectFilter
	projectSync     ProjectSync
	scheduler       ProjectScheduler

	registry     ProjectRegistry
	projectStore ProjectStore

	delayedFetchSem chan struct{}
	startStopMu     sync.Mutex
	lifecycleCtx    context.Context
	lifecycleStop   context.CancelFunc
	started         bool
}

func NewService(nodeClient *ethclient.Client, v2FactoryContract common.Address, wethContract common.Address, athenaContract common.Address, etherscanAPIBaseURL string, etherscanAPIKey string, projectStore ProjectStore, liquidityLocker []common.Address) *Service {
	registry := NewProjectRegistry(projectStore)

	return &Service{
		nodeClient:          nodeClient,
		registry:            registry,
		projectStore:        projectStore,
		delayedFetchSem:     make(chan struct{}, defaultDelayedFetchConcurrency),
		v2FactoryContract:   v2FactoryContract,
		wethContract:        wethContract,
		athenaContract:      athenaContract,
		etherscanAPIBaseURL: etherscanAPIBaseURL,
		etherscanAPIKey:     etherscanAPIKey,
		liquidityLocker:     liquidityLocker,
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
	blockCh := make(chan uint64, defaultBlockRefreshQueueCapacity)
	s.projectCh = ch1
	s.blockWatcher = NewBlockWatcher(s.nodeClient, ch1)
	s.blockSubscriber = NewBlockEventSubscriber(s.nodeClient, blockCh)
	athenaFetcher, err := evm.NewAthenaFetcher(s.nodeClient, s.athenaContract, s.liquidityLocker)
	if err != nil {
		close(ch1)
		s.clearPipelineLocked()
		return err
	}

	chainID, err := s.nodeClient.ChainID(context.Background())
	if err != nil {
		close(ch1)
		s.clearPipelineLocked()
		return err
	}

	apiFetcher := ethereumapi.NewEthereumAPI(s.etherscanAPIBaseURL, s.etherscanAPIKey, chainID.Int64())
	projectSimulator := NewProjectSimulator(s.nodeClient)

	ctx, cancel := context.WithCancel(context.Background())
	s.projectSync = NewProjectSync(s.registry, athenaFetcher, apiFetcher, projectSimulator, s.delayedFetchSem)
	s.scheduler = NewProjectScheduler(s.registry, s.projectSync, blockCh, ProjectSchedulerOptions{})
	if err := s.scheduler.Start(ctx); err != nil {
		cancel()
		close(ch1)
		s.clearPipelineLocked()
		return err
	}

	s.projectFilter = NewProjectFilter(s.registry, ch1, athenaFetcher, s.scheduler)
	if err := s.blockSubscriber.Start(ctx); err != nil {
		cancel()
		_ = s.scheduler.Stop()
		close(ch1)
		s.clearPipelineLocked()
		return err
	}

	if err := s.blockWatcher.Start(ctx); err != nil {
		cancel()
		_ = s.blockSubscriber.Stop()
		_ = s.scheduler.Stop()
		close(ch1)
		s.clearPipelineLocked()
		return err
	}

	if err := s.projectFilter.Start(ctx); err != nil {
		cancel()
		_ = s.blockWatcher.Stop()
		_ = s.blockSubscriber.Stop()
		_ = s.scheduler.Stop()
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

	subscriberErr := s.blockSubscriber.Stop()
	watcherErr := s.blockWatcher.Stop()
	if s.projectCh != nil {
		close(s.projectCh)
	}
	filterErr := s.projectFilter.Stop()
	schedulerErr := s.scheduler.Stop()

	s.lifecycleCtx = nil
	s.lifecycleStop = nil
	s.started = false
	s.clearPipelineLocked()

	return errors.Join(subscriberErr, watcherErr, filterErr, schedulerErr)
}

func (s *Service) clearPipelineLocked() {
	s.projectCh = nil
	s.blockWatcher = nil
	s.blockSubscriber = nil
	s.projectFilter = nil
	s.projectSync = nil
	s.scheduler = nil
}

func (s *Service) ListProjects(ctx context.Context, _ *applicationpkg.ListProjectsRequest) (*applicationpkg.ListProjectsResponse, error) {
	startedAt := time.Now()
	projects, err := s.registry.ListProjects(ctx)
	projectSnapshotLatency.Observe(float64(time.Since(startedAt).Milliseconds()))
	if err != nil {
		return nil, err
	}

	items := make([]*v1alpha1.ProjectView, 0, len(projects))
	for _, project := range projects {
		items = append(items, projectToView(project))
	}

	return &applicationpkg.ListProjectsResponse{Items: items}, nil
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
