package api

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	log "github.com/sirupsen/logrus"
	applicationpkg "github.com/useryege/athena/internal/application/apiclient"
	appcache "github.com/useryege/athena/internal/application/cache"
	appcomponents "github.com/useryege/athena/internal/application/components"
	avecomponent "github.com/useryege/athena/internal/application/components/ave"
	reportcomponent "github.com/useryege/athena/internal/application/components/report"
	"github.com/useryege/athena/internal/application/discovery"
	"github.com/useryege/athena/internal/application/evm"
	"github.com/useryege/athena/internal/application/persistence"
	"github.com/useryege/athena/internal/application/pipeline"
	"github.com/useryege/athena/internal/application/redisport"
	"github.com/useryege/athena/internal/application/simulate"
	appstore "github.com/useryege/athena/internal/application/store"
	solidityapiclient "github.com/useryege/athena/internal/solidity/apiclient"
	walletapiclient "github.com/useryege/athena/internal/wallet/apiclient"
	athenacontract "github.com/useryege/athena/pkg/abi/ATHENA"
	"github.com/useryege/athena/pkg/apis/application/v1alpha1"
	"github.com/useryege/athena/util/ave"
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

	aveConfig         ave.Config
	liquidityLocker   []common.Address
	solidityClientSet solidityapiclient.Clientset

	pipeline        *pipeline.ProjectPipeline
	athenaFetcher   evm.AthenaFetcher
	aveComponent    *avecomponent.Component
	walletBlacklist walletBlacklistLister

	store                appstore.ProjectStore
	projectCache         appcache.ProjectSnapshotCache
	componentCache       appcache.ProjectComponentCache
	persistencePublisher persistence.PersistenceEventPublisher
	persistenceBus       *persistence.RedisPersistenceEventBus
	persistenceWriter    persistence.PersistenceEventWriter
	componentEventBus    appcomponents.EventBus

	delayedFetchSem chan struct{}
	codeAtFunc      func(ctx context.Context, contract common.Address) ([]byte, error)
	startStopMu     sync.Mutex
	lifecycleCtx    context.Context
	lifecycleStop   context.CancelFunc
	bootstrapStop   context.CancelFunc
	starting        bool
	started         bool
}

func NewService(nodeClient *ethclient.Client, v2FactoryContract common.Address, wethContract common.Address, usdtContract common.Address, wethDecimals uint8, usdtDecimals uint8, athenaContract common.Address, aveConfig ave.Config, store appstore.Store, liquidityLocker []common.Address, redisClient redisport.Client, solidityClientSet solidityapiclient.Clientset, walletClientSet walletapiclient.Clientset) (*Service, error) {
	persistenceBus := persistence.NewRedisPersistenceEventBus(redisClient)

	return &Service{
		nodeClient:           nodeClient,
		store:                store,
		projectCache:         appcache.NewProjectSnapshotCache(redisClient),
		componentCache:       appcache.NewProjectComponentCache(redisClient),
		componentEventBus:    appcomponents.NewRedisEventBus(redisClient),
		walletBlacklist:      newWalletBlacklistClientLister(walletClientSet),
		persistencePublisher: persistenceBus,
		persistenceBus:       persistenceBus,
		persistenceWriter:    persistence.NewStorePersistenceWriter(store),
		v2FactoryContract:    v2FactoryContract,
		wethContract:         wethContract,
		usdtContract:         usdtContract,
		wethDecimals:         wethDecimals,
		usdtDecimals:         usdtDecimals,
		athenaContract:       athenaContract,
		aveConfig:            aveConfig,
		liquidityLocker:      liquidityLocker,
		solidityClientSet:    solidityClientSet,
	}, nil
}

func (s *Service) Start() error {
	s.startStopMu.Lock()
	if s.started {
		s.startStopMu.Unlock()
		return nil
	}
	if s.starting {
		s.startStopMu.Unlock()
		return status.Error(codes.Aborted, "service start already in progress")
	}

	ctx, cancel := context.WithCancel(context.Background())
	s.starting = true
	s.bootstrapStop = cancel
	s.startStopMu.Unlock()

	pipeline, athenaFetcher, aveComponent, err := s.startWithContext(ctx)
	if err != nil {
		cancel()
		s.startStopMu.Lock()
		s.starting = false
		s.bootstrapStop = nil
		s.clearPipelineLocked()
		s.startStopMu.Unlock()
		return err
	}

	s.startStopMu.Lock()
	if !s.starting {
		s.startStopMu.Unlock()
		cancel()
		if aveComponent != nil {
			aveComponent.Stop()
		}
		_ = pipeline.Stop()
		return context.Canceled
	}
	s.pipeline = pipeline
	s.athenaFetcher = athenaFetcher
	s.aveComponent = aveComponent
	s.lifecycleCtx = ctx
	s.lifecycleStop = cancel
	s.bootstrapStop = nil
	s.starting = false
	s.started = true
	s.startStopMu.Unlock()

	return nil
}

func (s *Service) startWithContext(ctx context.Context) (projectPipeline *pipeline.ProjectPipeline, athenaFetcher evm.AthenaFetcher, aveComponent *avecomponent.Component, err error) {
	startedAt := time.Now()
	startLogger := log.WithFields(log.Fields{
		"component": "application_start",
		"stage":     "start_with_context",
	})
	startLogger.Info("application startup stage started")
	defer func() {
		fields := log.Fields{
			"duration": time.Since(startedAt).String(),
		}
		if err != nil {
			fields["error"] = err.Error()
			startLogger.WithFields(fields).Warn("application startup stage failed")
			return
		}
		startLogger.WithFields(fields).Info("application startup stage completed")
	}()

	log.Info("athena-application project snapshot cache currently supports a single application writer replica")

	athenaFetcher, err = evm.NewAthenaFetcher(s.nodeClient, s.athenaContract, s.liquidityLocker)
	if err != nil {
		return nil, nil, nil, err
	}

	chainID, err := s.nodeClient.ChainID(ctx)
	if err != nil {
		return nil, nil, nil, err
	}

	aveStore, _ := s.store.(avecomponent.Store)
	aveComponent, err = avecomponent.NewComponent(avecomponent.Options{
		Config:   s.aveConfig,
		ChainID:  chainID.Int64(),
		Store:    aveStore,
		Cache:    s.componentCache,
		EventBus: s.componentEventBus,
	})
	if err != nil {
		return nil, nil, nil, err
	}
	if aveComponent != nil {
		log.Info("Ave component configured")
	}
	projectSimulator := simulate.NewProjectSimulator(s.nodeClient)
	componentStore, _ := s.store.(appstore.Store)

	if s.persistenceBus != nil && s.persistenceWriter != nil {
		go s.runPersistenceEventLoop(ctx)
	}

	initializer := appcomponents.NewInitializer(componentStore, s.componentCache, s.componentEventBus)
	chainStateComponent := appcomponents.NewChainStateComponent(componentStore, s.componentCache, athenaFetcher, s.componentEventBus)
	simulationComponent := appcomponents.NewSimulationComponent(componentStore, s.componentCache, athenaFetcher, projectSimulator, s.componentEventBus)
	genesisWalletComponent := appcomponents.NewGenesisWalletComponent(componentStore, s.componentCache, s.nodeClient, s.componentEventBus)
	creatorHistoryComponent := appcomponents.NewCreatorHistoryComponent(componentStore, s.componentCache, s.componentEventBus)
	bytecodeComponent := appcomponents.NewBytecodeComponent(componentStore, s.componentCache, s.solidityClientSet, chainID.Int64(), s.componentEventBus)
	requiredReportComponents := []string{
		appstore.ProjectComponentInitializer,
		appstore.ProjectComponentChainState,
		appstore.ProjectComponentSimulation,
		appstore.ProjectComponentGenesisWallet,
		appstore.ProjectComponentCreatorHistory,
	}
	if bytecodeComponent != nil {
		requiredReportComponents = append(requiredReportComponents, appstore.ProjectComponentBytecodeFact)
	}
	if aveComponent != nil {
		requiredReportComponents = append(requiredReportComponents, appstore.ProjectComponentAveDetail)
	}
	reportComponent := reportcomponent.NewComponent(reportcomponent.Options{
		Store:              componentStore,
		Cache:              s.componentCache,
		Bus:                s.componentEventBus,
		WalletBlacklist:    s.walletBlacklist,
		RequiredComponents: requiredReportComponents,
	})
	discoveryIntake := discovery.NewDiscoveryIntake(initializer)

	discoveryIndexer, err := discovery.NewProjectDiscoveryIndexer(s.nodeClient, s.componentCache, s.store, discoveryIntake)
	if err != nil {
		return nil, nil, nil, err
	}

	projectPipeline = pipeline.NewProjectPipeline(
		initializer,
		chainStateComponent,
		simulationComponent,
		genesisWalletComponent,
		creatorHistoryComponent,
		bytecodeComponent,
		reportComponent,
		discoveryIndexer,
	)
	if err := projectPipeline.Start(ctx); err != nil {
		return nil, nil, nil, err
	}
	if aveComponent != nil {
		if err := aveComponent.Start(ctx); err != nil {
			_ = projectPipeline.Stop()
			return nil, nil, nil, err
		}
	}
	return projectPipeline, athenaFetcher, aveComponent, nil
}

func genesisWalletAddressesFromMetas(items []GenesisWalletMeta) []common.Address {
	if len(items) == 0 {
		return nil
	}
	addresses := make([]common.Address, 0, len(items))
	for _, item := range items {
		addresses = append(addresses, item.Wallet)
	}
	return addresses
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

func buildProjectQueries(projects []*Project) ([]athenacontract.AthenaProjectQuery, []common.Address) {
	queries := make([]athenacontract.AthenaProjectQuery, 0, len(projects))
	contracts := make([]common.Address, 0, len(projects))
	for _, project := range projects {
		if project == nil {
			continue
		}
		queries = append(queries, athenacontract.AthenaProjectQuery{
			TokenContract:  project.Meta.Contract,
			MsgCaller:      project.Meta.Creator,
			GenesisWallets: genesisWalletAddressesFromMetas(project.Meta.GenesisWallets),
		})
		contracts = append(contracts, project.Meta.Contract)
	}
	return queries, contracts
}

func (s *Service) Stop() error {
	s.startStopMu.Lock()

	if s.starting && !s.started {
		stop := s.bootstrapStop
		s.starting = false
		s.bootstrapStop = nil
		s.startStopMu.Unlock()
		if stop != nil {
			stop()
		}
		return nil
	}

	if !s.started {
		s.startStopMu.Unlock()
		return nil
	}

	stop := s.lifecycleStop
	pipeline := s.pipeline
	aveComponent := s.aveComponent

	if stop != nil {
		stop()
	}

	s.lifecycleCtx = nil
	s.lifecycleStop = nil
	s.bootstrapStop = nil
	s.starting = false
	s.started = false
	s.clearPipelineLocked()
	s.startStopMu.Unlock()

	pipelineErr := error(nil)
	if aveComponent != nil {
		aveComponent.Stop()
	}
	if pipeline != nil {
		pipelineErr = pipeline.Stop()
	}

	return pipelineErr
}

func (s *Service) clearPipelineLocked() {
	s.pipeline = nil
	s.athenaFetcher = nil
	s.aveComponent = nil
}

func projectEventLogToAPI(item appstore.ProjectEventLog) *applicationpkg.ProjectEventLog {
	occurredAt := ""
	if !item.OccurredAt.IsZero() {
		occurredAt = item.OccurredAt.UTC().Format(time.RFC3339Nano)
	}
	createdAt := ""
	if !item.CreatedAt.IsZero() {
		createdAt = item.CreatedAt.UTC().Format(time.RFC3339Nano)
	}
	return &applicationpkg.ProjectEventLog{
		Id:         item.ID,
		Contract:   item.Contract.Hex(),
		EventType:  int32(item.EventType),
		OccurredAt: occurredAt,
		Message:    item.Message,
		Payload:    item.Payload,
		CreatedAt:  createdAt,
	}
}

func projectCommentToAPI(item appstore.ProjectComment) *applicationpkg.ProjectComment {
	return &applicationpkg.ProjectComment{
		Id:        item.ID,
		Contract:  item.Contract.Hex(),
		Username:  item.Username,
		Content:   item.Content,
		CreatedAt: item.CreatedAt.UTC().Format(time.RFC3339Nano),
	}
}

func (s *Service) fetchContractBytecode(ctx context.Context, contract common.Address) ([]byte, error) {
	if s.codeAtFunc != nil {
		return s.codeAtFunc(ctx, contract)
	}
	if s.nodeClient == nil {
		return nil, errors.New("node client is not configured")
	}
	return s.nodeClient.CodeAt(ctx, contract, nil)
}

func (s *Service) getProjectSnapshot(ctx context.Context, contract common.Address) (*Project, bool, error) {
	if s.projectCache != nil {
		project, ok, err := s.projectCache.GetProject(ctx, contract)
		if err != nil {
			return nil, false, err
		}
		if ok {
			return project, true, nil
		}
	}
	project, ok, err := s.loadProjectSnapshotFromDB(ctx, contract)
	if err != nil || !ok {
		return project, ok, err
	}
	if s.projectCache != nil {
		if err := s.projectCache.SetProject(ctx, project); err != nil {
			return nil, false, err
		}
	}
	return project, true, nil
}

func (s *Service) loadProjectSnapshotFromDB(ctx context.Context, contract common.Address) (*Project, bool, error) {
	if s.store == nil {
		return nil, false, nil
	}
	meta, err := s.store.GetProjectMetaByContract(ctx, contract)
	if err != nil {
		return nil, false, err
	}
	if meta == nil {
		return nil, false, nil
	}
	projects, err := s.hydrateProjectSnapshotsFromMetas(ctx, []appstore.ProjectMeta{*meta})
	if err != nil {
		return nil, false, err
	}
	if len(projects) == 0 {
		return nil, false, nil
	}
	return projects[0], true, nil
}

func (s *Service) listProjectSnapshotsFromDBPage(ctx context.Context, page int32, pageSize int32) ([]*Project, int64, int32, int32, error) {
	page, pageSize = normalizeCachePage(page, pageSize)
	if s.store == nil {
		return nil, 0, page, pageSize, nil
	}
	metas, err := s.store.ListProjectMetas(ctx)
	if err != nil {
		return nil, 0, page, pageSize, err
	}
	total := int64(len(metas))
	start := int64(page-1) * int64(pageSize)
	if start >= total {
		return nil, total, page, pageSize, nil
	}
	stop := start + int64(pageSize)
	if stop > total {
		stop = total
	}
	projects, err := s.hydrateProjectSnapshotsFromMetas(ctx, metas[start:stop])
	if err != nil {
		return nil, 0, page, pageSize, err
	}
	for _, project := range projects {
		if s.projectCache == nil || project == nil {
			continue
		}
		if err := s.projectCache.SetProject(ctx, project); err != nil {
			return nil, 0, page, pageSize, err
		}
	}
	return projects, total, page, pageSize, nil
}

func (s *Service) hydrateProjectSnapshotsFromMetas(ctx context.Context, metas []appstore.ProjectMeta) ([]*Project, error) {
	if len(metas) == 0 {
		return nil, nil
	}
	if s.athenaFetcher == nil {
		return nil, status.Error(codes.FailedPrecondition, "athena fetcher is not configured")
	}

	projects := make([]*Project, 0, len(metas))
	contracts := make([]common.Address, 0, len(metas))
	for _, meta := range metas {
		project := &Project{
			Meta:   projectMetaFromStore(meta),
			Report: projectReportFromStore(meta.Report),
		}
		projects = append(projects, project)
		contracts = append(contracts, project.Meta.Contract)
	}

	if store, ok := s.store.(appstore.ProjectGenesisWalletStore); ok && store != nil {
		byContract, err := store.ListProjectGenesisWalletsByContracts(ctx, contracts)
		if err != nil {
			return nil, err
		}
		for _, project := range projects {
			project.Meta.GenesisWallets = genesisWalletMetasFromStore(byContract[project.Meta.Contract])
		}
	}
	if store, ok := s.store.(appstore.ProjectCreatorHistoricalProjectStore); ok && store != nil {
		byContract, err := store.ListProjectCreatorHistoricalProjectsByContracts(ctx, contracts)
		if err != nil {
			return nil, err
		}
		for _, project := range projects {
			project.Meta.CreatorHistoricalProjects = creatorHistoricalProjectContractsFromStore(byContract[project.Meta.Contract])
		}
	}
	if store, ok := s.store.(appstore.ProjectAveDetailStore); ok && store != nil {
		byContract, err := store.ListProjectAveDetailsByContracts(ctx, contracts)
		if err != nil {
			return nil, err
		}
		for _, project := range projects {
			if detail, ok := byContract[project.Meta.Contract]; ok {
				project.AveDetail = projectAveDetailFromStore(detail)
			}
		}
	}

	queries, orderedContracts := buildProjectQueries(projects)
	snapshots, err := s.athenaFetcher.FetchProjects(ctx, queries)
	if err != nil {
		return nil, err
	}
	if len(snapshots) != len(projects) {
		return nil, fmt.Errorf("athena list returned %d projects for %d db projects", len(snapshots), len(projects))
	}
	for i, snapshot := range snapshots {
		if snapshot.TokenContract != (common.Address{}) && snapshot.TokenContract != orderedContracts[i] {
			return nil, fmt.Errorf("athena list result token contract = %s, want %s", snapshot.TokenContract, orderedContracts[i])
		}
		projects[i].Meta.ChainState = snapshot
	}
	return projects, nil
}

func genesisWalletMetasFromStore(items []appstore.ProjectGenesisWallet) []GenesisWalletMeta {
	if len(items) == 0 {
		return nil
	}
	result := make([]GenesisWalletMeta, 0, len(items))
	for _, item := range items {
		netAmount := item.NetAmount
		if netAmount != nil {
			netAmount = new(big.Int).Set(netAmount)
		}
		result = append(result, GenesisWalletMeta{
			Wallet:    item.Wallet,
			NetAmount: netAmount,
			RatioBPS:  item.RatioBPS,
			RankIndex: item.RankIndex,
		})
	}
	return result
}

func creatorHistoricalProjectContractsFromStore(items []appstore.ProjectCreatorHistoricalProject) []common.Address {
	if len(items) == 0 {
		return nil
	}
	result := make([]common.Address, 0, len(items))
	for _, item := range items {
		if item.HistoricalProjectContract == (common.Address{}) {
			continue
		}
		result = append(result, item.HistoricalProjectContract)
	}
	return result
}

func (s *Service) ListProjects(ctx context.Context, req *applicationpkg.ListProjectsRequest) (*applicationpkg.ListProjectsResponse, error) {
	startedAt := time.Now()
	var bases []appstore.ProjectBase
	var total int64
	var page int32
	var pageSize int32
	var err error
	if s.componentCache != nil {
		bases, total, page, pageSize, err = s.componentCache.ListBasePage(ctx, req.GetPage(), req.GetPageSize())
	} else {
		page, pageSize = normalizeCachePage(req.GetPage(), req.GetPageSize())
	}
	projectSnapshotLatency.Observe(float64(time.Since(startedAt).Milliseconds()))
	if err != nil {
		return nil, err
	}
	if shouldFallbackListProjectsToDB(total, page, pageSize, len(bases)) {
		store, ok := s.store.(appstore.ProjectBaseStore)
		if !ok || store == nil {
			projects, total, page, pageSize, err := s.listProjectSnapshotsFromDBPage(ctx, req.GetPage(), req.GetPageSize())
			if err != nil {
				return nil, err
			}
			items := make([]*v1alpha1.ProjectListItem, 0, len(projects))
			for _, project := range projects {
				items = append(items, projectToListItem(project))
			}
			return &applicationpkg.ListProjectsResponse{Items: items, Total: total, Page: page, PageSize: pageSize}, nil
		}
		bases, total, page, pageSize, err = store.ListProjectBasesPage(ctx, req.GetPage(), req.GetPageSize())
		if err != nil {
			return nil, err
		}
		if s.componentCache != nil {
			for _, base := range bases {
				if err := s.componentCache.SetBase(ctx, base); err != nil {
					return nil, err
				}
			}
		}
	}

	reports, err := s.projectReportsByContracts(ctx, projectBaseContracts(bases))
	if err != nil {
		return nil, err
	}
	items := make([]*v1alpha1.ProjectListItem, 0, len(bases))
	for _, base := range bases {
		report := reports[base.Contract]
		items = append(items, projectBaseAndReportToListItem(base, report))
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
	project, ok, err := s.getProjectSnapshot(ctx, contract)
	projectSnapshotLatency.Observe(float64(time.Since(startedAt).Milliseconds()))
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, status.Errorf(codes.NotFound, "project %q not found", req.GetContract())
	}

	return &applicationpkg.GetProjectResponse{Item: projectToView(project, true)}, nil
}

func (s *Service) GetProjectBase(ctx context.Context, req *applicationpkg.GetProjectBaseRequest) (*applicationpkg.GetProjectBaseResponse, error) {
	contract, err := parseProjectContract(req.GetContract())
	if err != nil {
		return nil, err
	}
	base, err := s.projectBase(ctx, contract)
	if err != nil {
		return nil, err
	}
	if base == nil {
		return nil, status.Errorf(codes.NotFound, "project %q not found", req.GetContract())
	}
	return &applicationpkg.GetProjectBaseResponse{Item: projectBaseToView(*base)}, nil
}

func (s *Service) GetProjectReport(ctx context.Context, req *applicationpkg.GetProjectReportRequest) (*applicationpkg.GetProjectReportResponse, error) {
	contract, err := parseProjectContract(req.GetContract())
	if err != nil {
		return nil, err
	}
	report, err := s.projectReport(ctx, contract)
	if err != nil {
		return nil, err
	}
	return &applicationpkg.GetProjectReportResponse{Item: projectReportToView(report)}, nil
}

func (s *Service) GetProjectChainState(ctx context.Context, req *applicationpkg.GetProjectChainStateRequest) (*applicationpkg.GetProjectChainStateResponse, error) {
	contract, err := parseProjectContract(req.GetContract())
	if err != nil {
		return nil, err
	}
	item, err := s.projectChainState(ctx, contract)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, status.Errorf(codes.NotFound, "project chain state %q not found", req.GetContract())
	}
	return &applicationpkg.GetProjectChainStateResponse{Item: projectChainStateToView(*item)}, nil
}

func (s *Service) GetProjectSimulation(ctx context.Context, req *applicationpkg.GetProjectSimulationRequest) (*applicationpkg.GetProjectSimulationResponse, error) {
	contract, err := parseProjectContract(req.GetContract())
	if err != nil {
		return nil, err
	}
	item, err := s.projectSimulation(ctx, contract)
	if err != nil {
		return nil, err
	}
	return &applicationpkg.GetProjectSimulationResponse{Item: projectSimulationToView(item)}, nil
}

func (s *Service) GetProjectAveState(ctx context.Context, req *applicationpkg.GetProjectAveStateRequest) (*applicationpkg.GetProjectAveStateResponse, error) {
	contract, err := parseProjectContract(req.GetContract())
	if err != nil {
		return nil, err
	}
	if err := s.ensureProjectExists(ctx, contract, req.GetContract()); err != nil {
		return nil, err
	}
	if s.aveComponent == nil {
		return nil, status.Error(codes.FailedPrecondition, "Ave component is not configured")
	}
	state, err := s.aveComponent.State(ctx, contract)
	if err != nil {
		return nil, err
	}
	return &applicationpkg.GetProjectAveStateResponse{Item: projectAveStateToView(state)}, nil
}

func (s *Service) RefreshProjectAveDetail(ctx context.Context, req *applicationpkg.RefreshProjectAveDetailRequest) (*applicationpkg.RefreshProjectAveDetailResponse, error) {
	contract, err := parseProjectContract(req.GetContract())
	if err != nil {
		return nil, err
	}
	if err := s.ensureProjectExists(ctx, contract, req.GetContract()); err != nil {
		return nil, err
	}
	if s.aveComponent == nil {
		return nil, status.Error(codes.FailedPrecondition, "Ave component is not configured")
	}
	if err := s.aveComponent.ScheduleRefresh(ctx, contract); err != nil {
		return nil, err
	}
	state, err := s.aveComponent.State(ctx, contract)
	if err != nil {
		return nil, err
	}
	return &applicationpkg.RefreshProjectAveDetailResponse{Item: projectAveStateToView(state)}, nil
}

func (s *Service) ListProjectGenesisWallets(ctx context.Context, req *applicationpkg.ListProjectGenesisWalletsRequest) (*applicationpkg.ListProjectGenesisWalletsResponse, error) {
	contract, err := parseProjectContract(req.GetContract())
	if err != nil {
		return nil, err
	}
	items, err := s.projectGenesisWallets(ctx, contract)
	if err != nil {
		return nil, err
	}
	return &applicationpkg.ListProjectGenesisWalletsResponse{Items: genesisWalletsToView(items)}, nil
}

func (s *Service) ListProjectCreatorHistoricalProjects(ctx context.Context, req *applicationpkg.ListProjectCreatorHistoricalProjectsRequest) (*applicationpkg.ListProjectCreatorHistoricalProjectsResponse, error) {
	contract, err := parseProjectContract(req.GetContract())
	if err != nil {
		return nil, err
	}
	items, err := s.projectCreatorHistory(ctx, contract)
	if err != nil {
		return nil, err
	}
	result := make([]string, 0, len(items))
	for _, item := range items {
		if item.HistoricalProjectContract != (common.Address{}) {
			result = append(result, item.HistoricalProjectContract.Hex())
		}
	}
	return &applicationpkg.ListProjectCreatorHistoricalProjectsResponse{Items: result}, nil
}

func shouldFallbackListProjectsToDB(total int64, page int32, pageSize int32, projectCount int) bool {
	if total == 0 {
		return true
	}
	if page < 1 || pageSize <= 0 {
		page, pageSize = normalizeCachePage(page, pageSize)
	}
	start := int64(page-1) * int64(pageSize)
	if start >= total {
		return false
	}
	expected := total - start
	if expected > int64(pageSize) {
		expected = int64(pageSize)
	}
	return int64(projectCount) < expected
}

func parseProjectContract(value string) (common.Address, error) {
	if !common.IsHexAddress(value) {
		return common.Address{}, status.Errorf(codes.InvalidArgument, "invalid contract %q", value)
	}
	return common.HexToAddress(value), nil
}

func projectBaseContracts(items []appstore.ProjectBase) []common.Address {
	contracts := make([]common.Address, 0, len(items))
	for _, item := range items {
		if item.Contract != (common.Address{}) {
			contracts = append(contracts, item.Contract)
		}
	}
	return contracts
}

func (s *Service) projectBase(ctx context.Context, contract common.Address) (*appstore.ProjectBase, error) {
	if s.componentCache != nil {
		if item, ok, err := s.componentCache.GetBase(ctx, contract); err != nil {
			return nil, err
		} else if ok && item != nil {
			return item, nil
		}
	}
	store, ok := s.store.(appstore.ProjectBaseStore)
	if !ok || store == nil {
		return nil, status.Error(codes.FailedPrecondition, "project base store is not configured")
	}
	item, err := store.GetProjectBaseByContract(ctx, contract)
	if err != nil || item == nil {
		return item, err
	}
	if s.componentCache != nil {
		if err := s.componentCache.SetBase(ctx, *item); err != nil {
			return nil, err
		}
	}
	return item, nil
}

func (s *Service) projectReportsByContracts(ctx context.Context, contracts []common.Address) (map[common.Address]*appstore.ProjectReportState, error) {
	result := make(map[common.Address]*appstore.ProjectReportState, len(contracts))
	missing := make([]common.Address, 0)
	if s.componentCache != nil {
		for _, contract := range contracts {
			item, ok, err := s.componentCache.GetReport(ctx, contract)
			if err != nil {
				return nil, err
			}
			if ok && item != nil {
				result[contract] = item
				continue
			}
			missing = append(missing, contract)
		}
	} else {
		missing = contracts
	}
	if len(missing) == 0 {
		return result, nil
	}
	store, ok := s.store.(appstore.ProjectReportStore)
	if !ok || store == nil {
		return result, nil
	}
	items, err := store.ListProjectReportStatesByContracts(ctx, missing)
	if err != nil {
		return nil, err
	}
	for contract, item := range items {
		itemCopy := item
		result[contract] = &itemCopy
		if s.componentCache != nil {
			if err := s.componentCache.SetReport(ctx, item); err != nil {
				return nil, err
			}
		}
	}
	return result, nil
}

func (s *Service) projectReport(ctx context.Context, contract common.Address) (*appstore.ProjectReportState, error) {
	reports, err := s.projectReportsByContracts(ctx, []common.Address{contract})
	if err != nil {
		return nil, err
	}
	return reports[contract], nil
}

func (s *Service) projectChainState(ctx context.Context, contract common.Address) (*appstore.ProjectChainState, error) {
	if s.componentCache != nil {
		if item, ok, err := s.componentCache.GetChainState(ctx, contract); err != nil {
			return nil, err
		} else if ok && item != nil {
			return item, nil
		}
	}
	store, ok := s.store.(appstore.ProjectChainStateStore)
	if !ok || store == nil {
		return nil, status.Error(codes.FailedPrecondition, "project chain state store is not configured")
	}
	item, err := store.GetProjectChainState(ctx, contract)
	if err != nil || item == nil {
		return item, err
	}
	if s.componentCache != nil {
		if err := s.componentCache.SetChainState(ctx, *item); err != nil {
			return nil, err
		}
	}
	return item, nil
}

func (s *Service) projectSimulation(ctx context.Context, contract common.Address) (*appstore.ProjectSimulationResult, error) {
	if s.componentCache != nil {
		if item, ok, err := s.componentCache.GetSimulation(ctx, contract); err != nil {
			return nil, err
		} else if ok && item != nil {
			return item, nil
		}
	}
	store, ok := s.store.(appstore.ProjectSimulationStore)
	if !ok || store == nil {
		return nil, nil
	}
	item, err := store.GetProjectSimulationResult(ctx, contract)
	if err != nil || item == nil {
		return item, err
	}
	if s.componentCache != nil {
		if err := s.componentCache.SetSimulation(ctx, *item); err != nil {
			return nil, err
		}
	}
	return item, nil
}

func (s *Service) projectAveDetail(ctx context.Context, contract common.Address) (*appstore.ProjectAveDetail, error) {
	if s.componentCache != nil {
		if item, ok, err := s.componentCache.GetAveDetail(ctx, contract); err != nil {
			return nil, err
		} else if ok && item != nil {
			return item, nil
		}
	}
	store, ok := s.store.(appstore.ProjectAveDetailStore)
	if !ok || store == nil {
		return nil, nil
	}
	item, err := store.GetProjectAveDetail(ctx, contract)
	if err != nil || item == nil {
		return item, err
	}
	if s.componentCache != nil {
		if err := s.componentCache.SetAveDetail(ctx, contract, *item); err != nil {
			return nil, err
		}
	}
	return item, nil
}

func (s *Service) projectGenesisWallets(ctx context.Context, contract common.Address) ([]appstore.ProjectGenesisWallet, error) {
	if s.componentCache != nil {
		if items, ok, err := s.componentCache.GetGenesisWallets(ctx, contract); err != nil {
			return nil, err
		} else if ok {
			return items, nil
		}
	}
	store, ok := s.store.(appstore.ProjectGenesisWalletStore)
	if !ok || store == nil {
		return nil, nil
	}
	items, err := store.ListProjectGenesisWalletsByContract(ctx, contract)
	if err != nil {
		return nil, err
	}
	if s.componentCache != nil {
		if err := s.componentCache.SetGenesisWallets(ctx, contract, items); err != nil {
			return nil, err
		}
	}
	return items, nil
}

func (s *Service) projectCreatorHistory(ctx context.Context, contract common.Address) ([]appstore.ProjectCreatorHistoricalProject, error) {
	if s.componentCache != nil {
		if items, ok, err := s.componentCache.GetCreatorHistory(ctx, contract); err != nil {
			return nil, err
		} else if ok {
			return items, nil
		}
	}
	store, ok := s.store.(appstore.ProjectCreatorHistoricalProjectStore)
	if !ok || store == nil {
		return nil, nil
	}
	items, err := store.ListProjectCreatorHistoricalProjectsByContract(ctx, contract)
	if err != nil {
		return nil, err
	}
	if s.componentCache != nil {
		if err := s.componentCache.SetCreatorHistory(ctx, contract, items); err != nil {
			return nil, err
		}
	}
	return items, nil
}

func (s *Service) ListProjectEventLogs(ctx context.Context, req *applicationpkg.ListProjectEventLogsRequest) (*applicationpkg.ListProjectEventLogsResponse, error) {
	if !common.IsHexAddress(req.GetContract()) {
		return nil, status.Errorf(codes.InvalidArgument, "invalid contract %q", req.GetContract())
	}
	store, ok := s.store.(appstore.ProjectEventLogStore)
	if !ok || store == nil {
		return &applicationpkg.ListProjectEventLogsResponse{}, status.Error(codes.FailedPrecondition, "project event log store is not configured")
	}

	items, err := store.ListProjectEventLogsByContract(ctx, common.HexToAddress(req.GetContract()))
	if err != nil {
		return nil, err
	}

	result := make([]*applicationpkg.ProjectEventLog, 0, len(items))
	for _, item := range items {
		result = append(result, projectEventLogToAPI(item))
	}
	return &applicationpkg.ListProjectEventLogsResponse{Items: result}, nil
}

func (s *Service) AddProjectComment(ctx context.Context, req *applicationpkg.AddProjectCommentRequest) (*applicationpkg.AddProjectCommentResponse, error) {
	if !common.IsHexAddress(req.GetContract()) {
		return nil, status.Errorf(codes.InvalidArgument, "invalid contract %q", req.GetContract())
	}

	username := strings.TrimSpace(req.GetUsername())
	if username == "" {
		return nil, status.Error(codes.InvalidArgument, "comment username is empty")
	}

	content := strings.TrimSpace(req.GetContent())
	if content == "" {
		return nil, status.Error(codes.InvalidArgument, "comment content is empty")
	}
	if utf8.RuneCountInString(content) > appstore.MaxProjectCommentContentLength {
		return nil, status.Errorf(codes.InvalidArgument, "comment content exceeds max length %d", appstore.MaxProjectCommentContentLength)
	}

	store, ok := s.store.(appstore.ProjectCommentStore)
	if !ok || store == nil {
		return &applicationpkg.AddProjectCommentResponse{}, status.Error(codes.FailedPrecondition, "project comment store is not configured")
	}

	contract := common.HexToAddress(req.GetContract())
	if err := s.ensureProjectExists(ctx, contract, req.GetContract()); err != nil {
		return nil, err
	}

	created, err := store.AddProjectComment(ctx, appstore.ProjectComment{
		Contract: contract,
		Username: username,
		Content:  content,
	})
	if err != nil {
		return nil, err
	}
	return &applicationpkg.AddProjectCommentResponse{Item: projectCommentToAPI(created)}, nil
}

func (s *Service) ListProjectComments(ctx context.Context, req *applicationpkg.ListProjectCommentsRequest) (*applicationpkg.ListProjectCommentsResponse, error) {
	if !common.IsHexAddress(req.GetContract()) {
		return nil, status.Errorf(codes.InvalidArgument, "invalid contract %q", req.GetContract())
	}

	store, ok := s.store.(appstore.ProjectCommentStore)
	if !ok || store == nil {
		return &applicationpkg.ListProjectCommentsResponse{}, status.Error(codes.FailedPrecondition, "project comment store is not configured")
	}

	contract := common.HexToAddress(req.GetContract())
	if err := s.ensureProjectExists(ctx, contract, req.GetContract()); err != nil {
		return nil, err
	}

	items, total, page, pageSize, err := store.ListProjectCommentsByContract(ctx, contract, req.GetPage(), req.GetPageSize())
	if err != nil {
		return nil, err
	}

	result := make([]*applicationpkg.ProjectComment, 0, len(items))
	for _, item := range items {
		result = append(result, projectCommentToAPI(item))
	}

	return &applicationpkg.ListProjectCommentsResponse{
		Items:    result,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func (s *Service) ensureProjectExists(ctx context.Context, contract common.Address, rawContract string) error {
	if s.store == nil {
		return status.Error(codes.FailedPrecondition, "project store is not configured")
	}
	meta, err := s.store.GetProjectMetaByContract(ctx, contract)
	if err != nil {
		return err
	}
	if meta == nil {
		return status.Errorf(codes.NotFound, "project %q not found", rawContract)
	}
	return nil
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

func paginateProjects(page int32, pageSize int32, projects *[]*Project) (int64, int32, int32) {
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
