package application

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	applicationpkg "github.com/useryege/athena/internal/application/apiclient"
	appcache "github.com/useryege/athena/internal/application/cache"
	"github.com/useryege/athena/internal/application/evm"
	"github.com/useryege/athena/internal/application/sourcecode"
	appstore "github.com/useryege/athena/internal/application/store"
	athenacontract "github.com/useryege/athena/pkg/abi/ATHENA"
	"github.com/useryege/athena/pkg/apis/application/v1alpha1"
	"github.com/useryege/athena/util/ethereumapi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	activeRedisFlushInterval = 3 * time.Second
	archivedRefreshInterval  = time.Hour
	bootstrapRetryInterval   = 3 * time.Second
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

	registry     ProjectRegistry
	store        appstore.Store
	projectCache ProjectSnapshotCache

	delayedFetchSem chan struct{}
	startStopMu     sync.Mutex
	lifecycleCtx    context.Context
	lifecycleStop   context.CancelFunc
	started         bool
}

func NewService(nodeClient *ethclient.Client, v2FactoryContract common.Address, wethContract common.Address, usdtContract common.Address, wethDecimals uint8, usdtDecimals uint8, athenaContract common.Address, etherscanAPIBaseURL string, etherscanAPIKey string, store appstore.Store, liquidityLocker []common.Address, redisClient *redis.Client) *Service {
	registry := NewProjectRegistry(store)
	sourceAnalyzer := sourcecode.NewAnalyzer()
	sourceBlacklist := appcache.NewSourceCodeBlacklistModel(
		store,
		appcache.NewLayeredBlacklistCache(appcache.NewLocalBlacklistCache(), appcache.NewSourceCodeBlacklistRedisCache(redisClient)),
	)

	return &Service{
		nodeClient:          nodeClient,
		registry:            registry,
		store:               store,
		projectCache:        NewProjectSnapshotCache(redisClient),
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

	if err := s.bootstrapProjectCaches(ctx, athenaFetcher, projectSimulator); err != nil {
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

	s.blockWatcher = NewBlockWatcher(s.nodeClient, s.registry, athenaFetcher)
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

	go s.runActiveRedisFlushLoop(ctx)
	go s.runArchivedRefreshLoop(ctx, athenaFetcher, projectSimulator)

	s.lifecycleCtx = ctx
	s.lifecycleStop = cancel
	s.started = true
	return nil
}

func (s *Service) bootstrapProjectCaches(ctx context.Context, fetcher evm.AthenaFetcher, simulator ProjectSimulator) error {
	store, ok := s.store.(appstore.ProjectStore)
	if !ok || store == nil {
		return status.Error(codes.FailedPrecondition, "project store is not configured")
	}

	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		metas, err := store.ListAllProjectMetas(ctx)
		if err != nil {
			time.Sleep(bootstrapRetryInterval)
			continue
		}
		projects, err := s.buildProjectsFromMetas(ctx, metas, fetcher, simulator)
		if err != nil {
			time.Sleep(bootstrapRetryInterval)
			continue
		}
		if err := s.projectCache.ReplaceAll(ctx, projects); err != nil {
			time.Sleep(bootstrapRetryInterval)
			continue
		}
		activeProjects, err := s.projectCache.ListActiveProjects(ctx)
		if err != nil {
			time.Sleep(bootstrapRetryInterval)
			continue
		}
		for _, project := range activeProjects {
			if err := s.registry.SetProject(ctx, project.Meta.ProjectID, project); err != nil {
				time.Sleep(bootstrapRetryInterval)
				continue
			}
		}
		return nil
	}
}

func (s *Service) buildProjectsFromMetas(ctx context.Context, metas []appstore.ProjectMeta, fetcher evm.AthenaFetcher, simulator ProjectSimulator) ([]*Project, error) {
	projects := make([]*Project, 0, len(metas))
	if len(metas) == 0 {
		return projects, nil
	}

	queries := make([]athenacontract.AthenaProjectQuery, 0, len(metas))
	for _, meta := range metas {
		queries = append(queries, athenacontract.AthenaProjectQuery{TokenContract: meta.Contract, MsgCaller: meta.Creator})
	}

	fetched, err := fetcher.FetchProjectsWithSimulationState(ctx, queries)
	if err != nil {
		return nil, err
	}
	if len(fetched) != len(metas) {
		return nil, fmt.Errorf("fetch projects with simulation state size mismatch: got %d want %d", len(fetched), len(metas))
	}

	for i, meta := range metas {
		project := &Project{
			Meta:       projectMetaFromStore(meta),
			ChainState: fetched[i].Project,
		}
		if simulator != nil {
			result, err := simulator.SimulatePrimary(
				ctx,
				meta.Creator,
				meta.Contract,
				fetched[i].Project.WethPair.ContractAddress,
				fetched[i].Project.UsdtPair.ContractAddress,
				fetched[i].SimulationState,
			)
			if err != nil {
				return nil, err
			}
			project.Meta.CreatorResult = result
		}
		projects = append(projects, project)
	}
	return projects, nil
}

func (s *Service) runActiveRedisFlushLoop(ctx context.Context) {
	ticker := time.NewTicker(activeRedisFlushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			projects, err := s.registry.ListProjects(ctx)
			if err != nil {
				continue
			}
			for _, project := range projects {
				_ = s.projectCache.SetProject(ctx, project)
			}
		}
	}
}

func (s *Service) runArchivedRefreshLoop(ctx context.Context, fetcher evm.AthenaFetcher, simulator ProjectSimulator) {
	ticker := time.NewTicker(archivedRefreshInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_ = s.refreshArchivedProjects(ctx, fetcher, simulator)
		}
	}
}

func (s *Service) refreshArchivedProjects(ctx context.Context, fetcher evm.AthenaFetcher, simulator ProjectSimulator) error {
	page := int32(1)
	for {
		items, total, current, pageSize, err := s.projectCache.ListArchivedProjects(ctx, page, 200)
		if err != nil {
			return err
		}
		if len(items) == 0 {
			return nil
		}
		metas := make([]appstore.ProjectMeta, 0, len(items))
		for _, item := range items {
			meta := projectMetaToStore(item.Meta)
			meta.IsArchived = true
			meta.ArchivedAt = item.Meta.ArchivedAt
			metas = append(metas, meta)
		}
		updated, err := s.buildProjectsFromMetas(ctx, metas, fetcher, simulator)
		if err != nil {
			return err
		}
		for _, project := range updated {
			project.Meta.IsArchived = true
			if err := s.projectCache.SetProject(ctx, project); err != nil {
				return err
			}
		}
		if int64(current*pageSize) >= total {
			return nil
		}
		page++
	}
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
	projects, err := s.projectCache.ListActiveProjects(ctx)
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
	project, ok, err := s.projectCache.GetProject(ctx, projectID)
	projectSnapshotLatency.Observe(float64(time.Since(startedAt).Milliseconds()))
	if err != nil {
		return nil, err
	}
	if !ok || project.Meta.IsArchived {
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

func (s *Service) ArchiveProject(ctx context.Context, req *applicationpkg.ArchiveProjectRequest) (*applicationpkg.ArchiveProjectResponse, error) {
	projectID, err := uuid.Parse(req.GetProjectID())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid projectID %q: %v", req.GetProjectID(), err)
	}
	if s.store == nil {
		return nil, status.Error(codes.FailedPrecondition, "project store is not configured")
	}
	meta, err := s.store.GetProjectMetaByID(ctx, projectID)
	if err != nil {
		return nil, err
	}
	if meta == nil {
		return nil, status.Errorf(codes.NotFound, "project %q not found", req.GetProjectID())
	}
	if err := s.store.ArchiveProjectByID(ctx, projectID); err != nil {
		return nil, err
	}
	project, ok, err := s.registry.GetProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	if ok && project != nil {
		project.Meta.IsArchived = true
		project.Meta.ArchivedAt = time.Now()
		if err := s.projectCache.SetProject(ctx, project); err != nil {
			return nil, err
		}
	}
	if err := s.registry.RemoveProject(ctx, projectID); err != nil {
		return nil, err
	}
	return &applicationpkg.ArchiveProjectResponse{}, nil
}

func (s *Service) UnarchiveProject(ctx context.Context, req *applicationpkg.UnarchiveProjectRequest) (*applicationpkg.UnarchiveProjectResponse, error) {
	projectID, err := uuid.Parse(req.GetProjectID())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid projectID %q: %v", req.GetProjectID(), err)
	}
	if s.store == nil {
		return nil, status.Error(codes.FailedPrecondition, "project store is not configured")
	}
	meta, err := s.store.GetProjectMetaByID(ctx, projectID)
	if err != nil {
		return nil, err
	}
	if meta == nil {
		return nil, status.Errorf(codes.NotFound, "project %q not found", req.GetProjectID())
	}
	if err := s.store.UnarchiveProjectByID(ctx, projectID); err != nil {
		return nil, err
	}
	project, ok, err := s.projectCache.GetProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	if !ok || project == nil {
		project = &Project{Meta: projectMetaFromStore(*meta)}
	}
	project.Meta.IsArchived = false
	project.Meta.ArchivedAt = time.Time{}
	if err := s.projectCache.SetProject(ctx, project); err != nil {
		return nil, err
	}
	if err := s.registry.SetProject(ctx, projectID, project); err != nil {
		return nil, err
	}
	return &applicationpkg.UnarchiveProjectResponse{}, nil
}

func (s *Service) ListArchivedProjects(ctx context.Context, req *applicationpkg.ListArchivedProjectsRequest) (*applicationpkg.ListArchivedProjectsResponse, error) {
	projects, total, page, pageSize, err := s.projectCache.ListArchivedProjects(ctx, req.GetPage(), req.GetPageSize())
	if err != nil {
		return nil, err
	}
	items := make([]*v1alpha1.ProjectView, 0, len(projects))
	for _, project := range projects {
		items = append(items, projectToView(project))
	}
	return &applicationpkg.ListArchivedProjectsResponse{Items: items, Total: total, Page: page, PageSize: pageSize}, nil
}

func (s *Service) GetArchivedProject(ctx context.Context, req *applicationpkg.GetArchivedProjectRequest) (*applicationpkg.GetArchivedProjectResponse, error) {
	projectID, err := uuid.Parse(req.GetProjectID())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid projectID %q: %v", req.GetProjectID(), err)
	}
	project, ok, err := s.projectCache.GetProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	if !ok || !project.Meta.IsArchived {
		return nil, status.Errorf(codes.NotFound, "archived project %q not found", req.GetProjectID())
	}
	return &applicationpkg.GetArchivedProjectResponse{Item: projectToView(project)}, nil
}
