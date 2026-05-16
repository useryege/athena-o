package application

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/redis/go-redis/v9"
	applicationpkg "github.com/useryege/athena/internal/application/apiclient"
	appcache "github.com/useryege/athena/internal/application/cache"
	"github.com/useryege/athena/internal/application/evm"
	"github.com/useryege/athena/internal/application/sourcecode"
	appstore "github.com/useryege/athena/internal/application/store"
	v1 "github.com/useryege/athena/internal/pkg/proto/v1"
	athenacontract "github.com/useryege/athena/pkg/abi/ATHENA"
	"github.com/useryege/athena/pkg/apis/application/v1alpha1"
	"github.com/useryege/athena/util/ethereumapi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	activeProjectStateRefreshInterval      = 3 * time.Second
	activeProjectSimulationRefreshInterval = time.Minute
	sourceCodeRefreshInterval              = time.Minute
	sourceCodeScanPageSize                 = 200
	bootstrapRetryInterval                 = 3 * time.Second
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
	apiFetcher      ethereumapi.EthereumAPI
	sourceAnalyzer  sourcecode.Analyzer
	sourceBlacklist appcache.SourceCodeBlacklistModel

	store                appstore.Store
	projectCache         ProjectSnapshotCache
	persistencePublisher PersistenceEventPublisher
	persistenceBus       *RedisPersistenceEventBus
	persistenceWriter    PersistenceEventWriter

	delayedFetchSem chan struct{}
	startStopMu     sync.Mutex
	lifecycleCtx    context.Context
	lifecycleStop   context.CancelFunc
	started         bool
}

func NewService(nodeClient *ethclient.Client, v2FactoryContract common.Address, wethContract common.Address, usdtContract common.Address, wethDecimals uint8, usdtDecimals uint8, athenaContract common.Address, etherscanAPIBaseURL string, etherscanAPIKey string, store appstore.Store, liquidityLocker []common.Address, redisClient *redis.Client) *Service {
	persistenceBus := NewRedisPersistenceEventBus(redisClient)
	sourceAnalyzer := sourcecode.NewAnalyzer()
	sourceBlacklist := appcache.NewSourceCodeBlacklistModel(
		store,
		appcache.NewLayeredBlacklistCache(appcache.NewLocalBlacklistCache(), appcache.NewSourceCodeBlacklistRedisCache(redisClient)),
		newSourceCodeBlacklistEventPublisher(persistenceBus),
	)

	return &Service{
		nodeClient:           nodeClient,
		store:                store,
		projectCache:         NewProjectSnapshotCache(redisClient),
		sourceAnalyzer:       sourceAnalyzer,
		sourceBlacklist:      sourceBlacklist,
		persistencePublisher: persistenceBus,
		persistenceBus:       persistenceBus,
		persistenceWriter:    NewStorePersistenceWriter(store),
		v2FactoryContract:    v2FactoryContract,
		wethContract:         wethContract,
		usdtContract:         usdtContract,
		wethDecimals:         wethDecimals,
		usdtDecimals:         usdtDecimals,
		athenaContract:       athenaContract,
		etherscanAPIBaseURL:  etherscanAPIBaseURL,
		etherscanAPIKey:      etherscanAPIKey,
		liquidityLocker:      liquidityLocker,
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
	if s.persistenceBus != nil && s.persistenceWriter != nil {
		go s.runPersistenceEventLoop(ctx)
	}

	s.apiFetcher = apiFetcher
	s.blockSubscriber = NewBlockEventSubscriber(s.nodeClient)
	s.blockWatcher = NewBlockWatcher(s.nodeClient, s.projectCache, athenaFetcher, s.persistencePublisher)
	if err := s.blockSubscriber.Start(ctx); err != nil {
		cancel()
		s.clearPipelineLocked()
		return err
	}

	if err := s.blockWatcher.Start(ctx); err != nil {
		cancel()
		_ = s.blockSubscriber.Stop()
		s.clearPipelineLocked()
		return err
	}

	go s.runActiveProjectStateRefreshLoop(ctx, athenaFetcher)
	go s.runActiveProjectSimulationRefreshLoop(ctx, athenaFetcher, projectSimulator)
	go s.runActiveProjectSourceCodeRefreshLoop(ctx)

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
		return nil
	}
}

func (s *Service) buildProjectsFromMetas(ctx context.Context, metas []appstore.ProjectMeta, fetcher evm.AthenaFetcher, simulator ProjectSimulator) ([]*Project, error) {
	projects := make([]*Project, 0, len(metas))
	if len(metas) == 0 {
		return projects, nil
	}

	blacklistFields, blacklistFieldsErr := s.sourceCodeBlacklistFields(ctx)

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
		if blacklistFieldsErr == nil && meta.SourceCode != "" && s.sourceAnalyzer != nil {
			project.Meta.SourceCodeBlacklist = s.sourceAnalyzer.AnalyzeSourceCode(meta.SourceCode, blacklistFields)
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

func (s *Service) runPersistenceEventLoop(ctx context.Context) {
	for {
		if err := ctx.Err(); err != nil {
			return
		}
		if s.persistenceBus == nil || s.persistenceWriter == nil {
			return
		}
		err := s.persistenceBus.Start(ctx, s.persistenceWriter)
		if err == nil || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Second):
		}
	}
}

func (s *Service) runActiveProjectStateRefreshLoop(ctx context.Context, fetcher evm.AthenaFetcher) {
	ticker := time.NewTicker(activeProjectStateRefreshInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_ = s.refreshActiveProjectStates(ctx, fetcher)
		}
	}
}

func (s *Service) refreshActiveProjectStates(ctx context.Context, fetcher evm.AthenaFetcher) error {
	activeProjects, err := s.projectCache.ListActiveProjects(ctx)
	if err != nil {
		return err
	}
	if len(activeProjects) == 0 {
		return nil
	}

	queries := make([]athenacontract.AthenaProjectQuery, 0, len(activeProjects))
	contracts := make([]common.Address, 0, len(activeProjects))
	for _, project := range activeProjects {
		if project == nil {
			continue
		}
		queries = append(queries, athenacontract.AthenaProjectQuery{
			TokenContract: project.Meta.Contract,
			MsgCaller:     project.Meta.Creator,
		})
		contracts = append(contracts, project.Meta.Contract)
	}
	if len(queries) == 0 {
		return nil
	}

	fetched, err := fetcher.FetchProjectsWithSimulationState(ctx, queries)
	if err != nil {
		return err
	}
	if len(fetched) != len(queries) {
		return fmt.Errorf("fetch projects with simulation state size mismatch: got %d want %d", len(fetched), len(queries))
	}

	for i, contract := range contracts {
		nextState := fetched[i].Project
		_, err := s.projectCache.UpdateProject(ctx, contract, func(current *Project, exists bool) (*Project, bool, error) {
			if !exists || current == nil || current.Meta.IsArchived {
				return nil, false, nil
			}
			current.ChainState = nextState
			return current, true, nil
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) runActiveProjectSimulationRefreshLoop(ctx context.Context, fetcher evm.AthenaFetcher, simulator ProjectSimulator) {
	ticker := time.NewTicker(activeProjectSimulationRefreshInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_ = s.refreshActiveProjectSimulations(ctx, fetcher, simulator)
		}
	}
}

func (s *Service) refreshActiveProjectSimulations(ctx context.Context, fetcher evm.AthenaFetcher, simulator ProjectSimulator) error {
	if simulator == nil {
		return nil
	}

	activeProjects, err := s.projectCache.ListActiveProjects(ctx)
	if err != nil {
		return err
	}
	if len(activeProjects) == 0 {
		return nil
	}

	queries := make([]athenacontract.AthenaProjectQuery, 0, len(activeProjects))
	contracts := make([]common.Address, 0, len(activeProjects))
	for _, project := range activeProjects {
		if project == nil {
			continue
		}
		queries = append(queries, athenacontract.AthenaProjectQuery{
			TokenContract: project.Meta.Contract,
			MsgCaller:     project.Meta.Creator,
		})
		contracts = append(contracts, project.Meta.Contract)
	}
	if len(queries) == 0 {
		return nil
	}

	states, err := fetcher.FetchSimulationStates(ctx, queries)
	if err != nil {
		return err
	}
	if len(states) != len(queries) {
		return fmt.Errorf("fetch simulation states size mismatch: got %d want %d", len(states), len(queries))
	}

	for i, contract := range contracts {
		latest, ok, err := s.projectCache.GetProject(ctx, contract)
		if err != nil {
			return err
		}
		if !ok || latest == nil || latest.Meta.IsArchived {
			continue
		}

		wethPairContract := latest.ChainState.WethPair.ContractAddress
		usdtPairContract := latest.ChainState.UsdtPair.ContractAddress
		if wethPairContract == (common.Address{}) || usdtPairContract == (common.Address{}) {
			continue
		}

		result, err := simulator.SimulatePrimary(
			ctx,
			latest.Meta.Creator,
			latest.Meta.Contract,
			wethPairContract,
			usdtPairContract,
			states[i],
		)
		if err != nil {
			continue
		}

		_, err = s.projectCache.UpdateProject(ctx, contract, func(current *Project, exists bool) (*Project, bool, error) {
			if !exists || current == nil || current.Meta.IsArchived {
				return nil, false, nil
			}
			current.Meta.CreatorResult = result
			return current, true, nil
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) runActiveProjectSourceCodeRefreshLoop(ctx context.Context) {
	ticker := time.NewTicker(sourceCodeRefreshInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_ = s.refreshActiveProjectSourceCodes(ctx)
		}
	}
}

func (s *Service) refreshActiveProjectSourceCodes(ctx context.Context) error {
	fields, err := s.sourceCodeBlacklistFields(ctx)
	if err != nil {
		return err
	}

	activeProjects, err := s.projectCache.ListActiveProjects(ctx)
	if err != nil {
		return err
	}
	if err := s.processProjectSourceCodeBatch(ctx, activeProjects, fields); err != nil {
		return err
	}
	return nil
}

func (s *Service) processProjectSourceCodeBatch(ctx context.Context, projects []*Project, fields []string) error {
	for _, project := range projects {
		if err := ctx.Err(); err != nil {
			return err
		}
		if project == nil {
			continue
		}

		sourceCode := project.Meta.SourceCode
		if sourceCode == "" && s.apiFetcher != nil {
			fetchedSourceCode, _, fetchErr := s.fetchSourceCode(ctx, project)
			if fetchErr != nil {
				continue
			}
			sourceCode = fetchedSourceCode
		}

		var analyzedSourceCode string
		var blacklistReport sourcecode.BlacklistReport
		if sourceCode != "" && s.sourceAnalyzer != nil && project.Meta.SourceCodeBlacklist.ResolvedAt.IsZero() {
			analyzedSourceCode = sourceCode
			blacklistReport = s.sourceAnalyzer.AnalyzeSourceCode(sourceCode, fields)
		}

		shouldPersistSourceCode := false
		_, err := s.projectCache.UpdateProject(ctx, project.Meta.Contract, func(current *Project, exists bool) (*Project, bool, error) {
			if !exists || current == nil || current.Meta.IsArchived {
				return nil, false, nil
			}

			changed := false
			if sourceCode != "" && current.Meta.SourceCode == "" {
				current.Meta.SourceCode = sourceCode
				changed = true
				shouldPersistSourceCode = true
			}

			if analyzedSourceCode != "" && current.Meta.SourceCode == analyzedSourceCode && current.Meta.SourceCodeBlacklist.ResolvedAt.IsZero() {
				current.Meta.SourceCodeBlacklist = blacklistReport
				changed = true
			}

			return current, changed, nil
		})
		if err != nil {
			continue
		}
		if shouldPersistSourceCode {
			if err := s.persistProjectSourceCode(ctx, project.Meta.Contract, sourceCode); err != nil {
				continue
			}
		}
	}
	return nil
}

func (s *Service) persistProjectSourceCode(ctx context.Context, contract common.Address, sourceCode string) error {
	if sourceCode == "" || s.persistencePublisher == nil {
		return nil
	}
	return s.persistencePublisher.PublishProjectSourceCodeUpdate(ctx, contract, sourceCode)
}

func (s *Service) fetchSourceCode(ctx context.Context, project *Project) (string, string, error) {
	response, err := s.apiFetcher.GetSourceCode(ctx, project.Meta.Contract.String())
	if err != nil {
		return "", "", err
	}
	if len(response.Result) == 0 {
		return "", "", errors.New("etherscan getsourcecode returned empty result")
	}
	return response.Result[0].SourceCode, response.Result[0].ABI, nil
}

func (s *Service) sourceCodeBlacklistFields(ctx context.Context) ([]string, error) {
	if s.sourceBlacklist == nil {
		return nil, nil
	}
	return s.sourceBlacklist.List(ctx)
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

	s.lifecycleCtx = nil
	s.lifecycleStop = nil
	s.started = false
	s.clearPipelineLocked()

	return errors.Join(subscriberErr, watcherErr)
}

func (s *Service) clearPipelineLocked() {
	s.blockWatcher = nil
	s.blockSubscriber = nil
	s.apiFetcher = nil
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

func (s *Service) ListProjects(ctx context.Context, req *applicationpkg.ListProjectsRequest) (*applicationpkg.ListProjectsResponse, error) {
	scope, err := normalizeProjectScope(req.GetScope())
	if err != nil {
		return nil, err
	}

	startedAt := time.Now()
	var (
		projects []*Project
		total    int64
		page     int32
		pageSize int32
	)
	switch scope {
	case v1.ProjectScope_PROJECT_SCOPE_ACTIVE:
		projects, err = s.projectCache.ListActiveProjects(ctx)
		projectSnapshotLatency.Observe(float64(time.Since(startedAt).Milliseconds()))
		if err != nil {
			return nil, err
		}
		total, page, pageSize = paginateActiveProjects(req.GetPage(), req.GetPageSize(), &projects)
	case v1.ProjectScope_PROJECT_SCOPE_ARCHIVED:
		projects, total, page, pageSize, err = s.projectCache.ListArchivedProjects(ctx, req.GetPage(), req.GetPageSize())
		projectSnapshotLatency.Observe(float64(time.Since(startedAt).Milliseconds()))
		if err != nil {
			return nil, err
		}
	default:
		return nil, status.Errorf(codes.Internal, "unsupported project scope %v", scope)
	}

	items := make([]*v1alpha1.ProjectView, 0, len(projects))
	for _, project := range projects {
		items = append(items, projectToView(project))
	}

	return &applicationpkg.ListProjectsResponse{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func (s *Service) GetProject(ctx context.Context, req *applicationpkg.GetProjectRequest) (*applicationpkg.GetProjectResponse, error) {
	if !common.IsHexAddress(req.GetContract()) {
		return nil, status.Errorf(codes.InvalidArgument, "invalid contract %q", req.GetContract())
	}
	contract := common.HexToAddress(req.GetContract())

	startedAt := time.Now()
	project, ok, err := s.projectCache.GetProject(ctx, contract)
	projectSnapshotLatency.Observe(float64(time.Since(startedAt).Milliseconds()))
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, status.Errorf(codes.NotFound, "project %q not found", req.GetContract())
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
	if !common.IsHexAddress(req.GetContract()) {
		return nil, status.Errorf(codes.InvalidArgument, "invalid contract %q", req.GetContract())
	}
	contract := common.HexToAddress(req.GetContract())
	if s.store == nil {
		return nil, status.Error(codes.FailedPrecondition, "project store is not configured")
	}
	meta, err := s.store.GetProjectMetaByContract(ctx, contract)
	if err != nil {
		return nil, err
	}
	if meta == nil {
		return nil, status.Errorf(codes.NotFound, "project %q not found", req.GetContract())
	}
	if s.persistencePublisher == nil {
		return nil, status.Error(codes.FailedPrecondition, "persistence publisher is not configured")
	}
	if err := s.persistencePublisher.PublishProjectArchive(ctx, contract); err != nil {
		return nil, err
	}
	_, err = s.projectCache.UpdateProject(ctx, contract, func(current *Project, exists bool) (*Project, bool, error) {
		if !exists || current == nil {
			current = &Project{Meta: projectMetaFromStore(*meta)}
		}
		current.Meta.IsArchived = true
		current.Meta.ArchivedAt = time.Now().UTC()
		return current, true, nil
	})
	if err != nil {
		return nil, err
	}
	return &applicationpkg.ArchiveProjectResponse{}, nil
}

func (s *Service) UnarchiveProject(ctx context.Context, req *applicationpkg.UnarchiveProjectRequest) (*applicationpkg.UnarchiveProjectResponse, error) {
	if !common.IsHexAddress(req.GetContract()) {
		return nil, status.Errorf(codes.InvalidArgument, "invalid contract %q", req.GetContract())
	}
	contract := common.HexToAddress(req.GetContract())
	if s.store == nil {
		return nil, status.Error(codes.FailedPrecondition, "project store is not configured")
	}
	meta, err := s.store.GetProjectMetaByContract(ctx, contract)
	if err != nil {
		return nil, err
	}
	if meta == nil {
		return nil, status.Errorf(codes.NotFound, "project %q not found", req.GetContract())
	}
	if s.persistencePublisher == nil {
		return nil, status.Error(codes.FailedPrecondition, "persistence publisher is not configured")
	}
	if err := s.persistencePublisher.PublishProjectUnarchive(ctx, contract); err != nil {
		return nil, err
	}
	_, err = s.projectCache.UpdateProject(ctx, contract, func(current *Project, exists bool) (*Project, bool, error) {
		if !exists || current == nil {
			current = &Project{Meta: projectMetaFromStore(*meta)}
		}
		current.Meta.IsArchived = false
		current.Meta.ArchivedAt = time.Time{}
		return current, true, nil
	})
	if err != nil {
		return nil, err
	}
	return &applicationpkg.UnarchiveProjectResponse{}, nil
}

func normalizeProjectScope(scope v1.ProjectScope) (v1.ProjectScope, error) {
	switch scope {
	case v1.ProjectScope_PROJECT_SCOPE_UNSPECIFIED, v1.ProjectScope_PROJECT_SCOPE_ACTIVE:
		return v1.ProjectScope_PROJECT_SCOPE_ACTIVE, nil
	case v1.ProjectScope_PROJECT_SCOPE_ARCHIVED:
		return v1.ProjectScope_PROJECT_SCOPE_ARCHIVED, nil
	case v1.ProjectScope_PROJECT_SCOPE_ALL:
		return 0, status.Error(codes.InvalidArgument, "scope PROJECT_SCOPE_ALL is not supported")
	default:
		return 0, status.Errorf(codes.InvalidArgument, "invalid scope %q", scope.String())
	}
}

func normalizeProjectPage(page int32, pageSize int32) (int32, int32) {
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 200 {
		pageSize = 200
	}
	return page, pageSize
}

func paginateActiveProjects(page int32, pageSize int32, projects *[]*Project) (int64, int32, int32) {
	normalizedPage, normalizedPageSize := normalizeProjectPage(page, pageSize)
	total := int64(len(*projects))
	start := int64(normalizedPage-1) * int64(normalizedPageSize)
	if start >= total {
		*projects = (*projects)[:0]
		return total, normalizedPage, normalizedPageSize
	}

	stop := start + int64(normalizedPageSize)
	if stop > total {
		stop = total
	}

	*projects = (*projects)[start:stop]
	return total, normalizedPage, normalizedPageSize
}
