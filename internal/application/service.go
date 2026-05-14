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
	appcache "github.com/useryege/athena/internal/application/cache"
	"github.com/useryege/athena/internal/application/evm"
	"github.com/useryege/athena/internal/application/sourcecode"
	appstore "github.com/useryege/athena/internal/application/store"
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
	usdtContract      common.Address
	wethDecimals      uint8
	usdtDecimals      uint8
	athenaContract    common.Address

	etherscanAPIBaseURL string
	etherscanAPIKey     string
	liquidityLocker     []common.Address

	blockWatcher    *BlockWatcher
	blockSubscriber *BlockEventSubscriber
	projectSync     ProjectSync
	scheduler       ProjectScheduler
	sourceAnalyzer  sourcecode.Analyzer
	sourceBlacklist appcache.SourceCodeBlacklistModel

	registry ProjectRegistry

	delayedFetchSem chan struct{}
	startStopMu     sync.Mutex
	lifecycleCtx    context.Context
	lifecycleStop   context.CancelFunc
	started         bool
}

func NewService(nodeClient *ethclient.Client, v2FactoryContract common.Address, wethContract common.Address, usdtContract common.Address, wethDecimals uint8, usdtDecimals uint8, athenaContract common.Address, etherscanAPIBaseURL string, etherscanAPIKey string, store appstore.Store, liquidityLocker []common.Address) *Service {
	registry := NewProjectRegistry(store)
	sourceAnalyzer := sourcecode.NewAnalyzer()
	sourceBlacklist := appcache.NewSourceCodeBlacklistModel(
		store,
		appcache.NewLayeredBlacklistCache(appcache.NewLocalBlacklistCache(), nil),
	)

	return &Service{
		nodeClient:          nodeClient,
		registry:            registry,
		sourceAnalyzer:      sourceAnalyzer,
		sourceBlacklist:     sourceBlacklist,
		delayedFetchSem:     make(chan struct{}, defaultDelayedFetchConcurrency),
		v2FactoryContract:   v2FactoryContract,
		wethContract:        wethContract,
		usdtContract:        usdtContract,
		wethDecimals:        wethDecimals,
		usdtDecimals:        usdtDecimals,
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

	athenaFetcher, err := evm.NewAthenaFetcher(s.nodeClient, s.athenaContract, s.liquidityLocker)
	if err != nil {
		s.clearPipelineLocked()
		return err
	}

	chainID, err := s.nodeClient.ChainID(context.Background())
	if err != nil {
		s.clearPipelineLocked()
		return err
	}

	apiFetcher := ethereumapi.NewEthereumAPI(s.etherscanAPIBaseURL, s.etherscanAPIKey, chainID.Int64())
	projectSimulator := NewProjectSimulator(s.nodeClient)

	ctx, cancel := context.WithCancel(context.Background())
	if s.sourceBlacklist != nil {
		if err := s.sourceBlacklist.Load(ctx); err != nil {
			cancel()
			s.clearPipelineLocked()
			return err
		}
	}
	if err := s.registry.LoadProjects(ctx); err != nil {
		cancel()
		s.clearPipelineLocked()
		return err
	}

	s.projectSync = NewProjectSync(s.nodeClient, s.registry, athenaFetcher, apiFetcher, projectSimulator, s.delayedFetchSem)
	s.blockSubscriber = NewBlockEventSubscriber(s.nodeClient, s.projectSync)
	s.scheduler = NewProjectScheduler(s.registry, s.projectSync, s.sourceAnalyzer, s.sourceBlacklist, ProjectSchedulerOptions{})
	if err := s.scheduler.Start(ctx); err != nil {
		cancel()
		s.clearPipelineLocked()
		return err
	}

	s.blockWatcher = NewBlockWatcher(s.nodeClient, s.registry, athenaFetcher, s.scheduler)
	if err := s.blockSubscriber.Start(ctx); err != nil {
		cancel()
		_ = s.scheduler.Stop()
		s.clearPipelineLocked()
		return err
	}

	if err := s.blockWatcher.Start(ctx); err != nil {
		cancel()
		_ = s.blockSubscriber.Stop()
		_ = s.scheduler.Stop()
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
	schedulerErr := s.scheduler.Stop()

	s.lifecycleCtx = nil
	s.lifecycleStop = nil
	s.started = false
	s.clearPipelineLocked()

	return errors.Join(subscriberErr, watcherErr, schedulerErr)
}

func (s *Service) clearPipelineLocked() {
	s.blockWatcher = nil
	s.blockSubscriber = nil
	s.projectSync = nil
	s.scheduler = nil
}

func (s *Service) ListSourceCodeBlacklistFields(ctx context.Context, _ *applicationpkg.ListSourceCodeBlacklistFieldsRequest) (*applicationpkg.ListSourceCodeBlacklistFieldsResponse, error) {
	if s.sourceBlacklist == nil {
		return &applicationpkg.ListSourceCodeBlacklistFieldsResponse{}, nil
	}

	fields, err := s.sourceBlacklist.List(ctx)
	if err != nil {
		return nil, err
	}

	items := make([]*applicationpkg.SourceCodeBlacklistField, 0, len(fields))
	for _, field := range fields {
		items = append(items, &applicationpkg.SourceCodeBlacklistField{Field: field})
	}
	return &applicationpkg.ListSourceCodeBlacklistFieldsResponse{Items: items}, nil
}

func (s *Service) AddSourceCodeBlacklistField(ctx context.Context, req *applicationpkg.AddSourceCodeBlacklistFieldRequest) (*applicationpkg.AddSourceCodeBlacklistFieldResponse, error) {
	if s.sourceBlacklist == nil {
		return &applicationpkg.AddSourceCodeBlacklistFieldResponse{}, nil
	}
	field := req.GetField()
	if err := s.sourceBlacklist.Add(ctx, field); err != nil {
		return nil, err
	}
	return &applicationpkg.AddSourceCodeBlacklistFieldResponse{Item: &applicationpkg.SourceCodeBlacklistField{Field: field}}, nil
}

func (s *Service) DeleteSourceCodeBlacklistField(ctx context.Context, req *applicationpkg.DeleteSourceCodeBlacklistFieldRequest) (*applicationpkg.DeleteSourceCodeBlacklistFieldResponse, error) {
	if s.sourceBlacklist == nil {
		return &applicationpkg.DeleteSourceCodeBlacklistFieldResponse{}, nil
	}
	if err := s.sourceBlacklist.Delete(ctx, req.GetField()); err != nil {
		return nil, err
	}
	return &applicationpkg.DeleteSourceCodeBlacklistFieldResponse{}, nil
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

func (s *Service) GetProjectOptions(context.Context, *applicationpkg.GetProjectOptionsRequest) (*applicationpkg.GetProjectOptionsResponse, error) {
	return &applicationpkg.GetProjectOptionsResponse{
		Options: &v1alpha1.ProjectOption{
			FactoryContract: s.v2FactoryContract.Hex(),
			WethContract:    s.wethContract.Hex(),
			UsdtContract:    s.usdtContract.Hex(),
			WethDecimals:    uint32(s.wethDecimals),
			UsdtDecimals:    uint32(s.usdtDecimals),
		},
	}, nil
}
