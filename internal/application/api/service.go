package api

import (
	"context"
	"errors"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	log "github.com/sirupsen/logrus"
	applicationpkg "github.com/useryege/athena/internal/application/apiclient"
	appcache "github.com/useryege/athena/internal/application/cache"
	appcomponents "github.com/useryege/athena/internal/application/components"
	avecomponent "github.com/useryege/athena/internal/application/components/ave"
	bytecodecomponent "github.com/useryege/athena/internal/application/components/bytecode"
	chainstatecomponent "github.com/useryege/athena/internal/application/components/chainstate"
	creatorhistorycomponent "github.com/useryege/athena/internal/application/components/creatorhistory"
	genesiswalletcomponent "github.com/useryege/athena/internal/application/components/genesiswallet"
	initializercomponent "github.com/useryege/athena/internal/application/components/initializer"
	reportcomponent "github.com/useryege/athena/internal/application/components/report"
	simulationcomponent "github.com/useryege/athena/internal/application/components/simulation"
	"github.com/useryege/athena/internal/application/discovery"
	"github.com/useryege/athena/internal/application/evm"
	"github.com/useryege/athena/internal/application/model"
	"github.com/useryege/athena/internal/application/persistence"
	"github.com/useryege/athena/internal/application/pipeline"
	appstore "github.com/useryege/athena/internal/application/store"
	solidityapiclient "github.com/useryege/athena/internal/solidity/apiclient"
	walletapiclient "github.com/useryege/athena/internal/wallet/apiclient"
	"github.com/useryege/athena/pkg/apis/application/v1alpha1"
	"github.com/useryege/athena/util/ave"
	"github.com/useryege/athena/util/redisport"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	projectRefreshThrottleInterval = 30 * time.Second
	projectRefreshPublishTimeout   = 5 * time.Second
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

	dbStore           appstore.Store
	store             appstore.ProjectStore
	componentCache    appcache.ProjectComponentCache
	componentEventBus appcomponents.EventBus
	persistenceFlush  *persistence.Flusher

	projectRefreshMu      sync.Mutex
	projectRefreshLastRun map[common.Address]time.Time
	projectRefreshWindow  time.Duration

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
	bufferedStore, err := persistence.NewRedisBufferedStore(store, redisClient)
	if err != nil {
		return nil, err
	}
	return &Service{
		nodeClient:            nodeClient,
		dbStore:               store,
		store:                 bufferedStore,
		componentCache:        appcache.NewProjectComponentCache(redisClient),
		componentEventBus:     appcomponents.NewRedisEventBus(redisClient),
		persistenceFlush:      persistence.NewFlusher(bufferedStore, time.Minute),
		walletBlacklist:       newWalletBlacklistClientLister(walletClientSet),
		v2FactoryContract:     v2FactoryContract,
		wethContract:          wethContract,
		usdtContract:          usdtContract,
		wethDecimals:          wethDecimals,
		usdtDecimals:          usdtDecimals,
		athenaContract:        athenaContract,
		aveConfig:             aveConfig,
		liquidityLocker:       liquidityLocker,
		solidityClientSet:     solidityClientSet,
		projectRefreshLastRun: map[common.Address]time.Time{},
		projectRefreshWindow:  projectRefreshThrottleInterval,
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
	componentStore, _ := s.store.(appstore.Store)

	initializer := initializercomponent.NewComponent(initializercomponent.Options{
		Store: componentStore,
		Cache: s.componentCache,
		Bus:   s.componentEventBus,
	})
	chainStateComponent := chainstatecomponent.NewComponent(chainstatecomponent.Options{
		Store:   componentStore,
		Cache:   s.componentCache,
		Fetcher: athenaFetcher,
		Bus:     s.componentEventBus,
	})
	simulationComponent := simulationcomponent.NewComponent(simulationcomponent.Options{
		Store:      componentStore,
		Cache:      s.componentCache,
		Fetcher:    athenaFetcher,
		NodeClient: s.nodeClient.Client(),
		Bus:        s.componentEventBus,
	})
	genesisWalletComponent := genesiswalletcomponent.NewComponent(genesiswalletcomponent.Options{
		Store:      componentStore,
		Cache:      s.componentCache,
		NodeClient: s.nodeClient,
		Bus:        s.componentEventBus,
	})
	creatorHistoryComponent := creatorhistorycomponent.NewComponent(creatorhistorycomponent.Options{
		Store: componentStore,
		Cache: s.componentCache,
		Bus:   s.componentEventBus,
	})
	bytecodeComponent := bytecodecomponent.NewComponent(bytecodecomponent.Options{
		Store:   componentStore,
		Cache:   s.componentCache,
		Clients: s.solidityClientSet,
		ChainID: chainID.Int64(),
		Bus:     s.componentEventBus,
	})
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
		s.persistenceFlush,
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

func (s *Service) fetchContractBytecode(ctx context.Context, contract common.Address) ([]byte, error) {
	if s.codeAtFunc != nil {
		return s.codeAtFunc(ctx, contract)
	}
	if s.nodeClient == nil {
		return nil, errors.New("node client is not configured")
	}
	return s.nodeClient.CodeAt(ctx, contract, nil)
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
	baseStore, ok := s.dbStore.(appstore.ProjectBaseStore)
	if !ok || baseStore == nil {
		return nil, status.Error(codes.FailedPrecondition, "project base db store is not configured")
	}
	reportStore, _ := s.dbStore.(appstore.ProjectReportStore)

	bases, total, page, pageSize, err := baseStore.ListProjectBasesPage(ctx, req.GetPage(), req.GetPageSize())
	projectSnapshotLatency.Observe(float64(time.Since(startedAt).Milliseconds()))
	if err != nil {
		return nil, err
	}

	reports := map[common.Address]appstore.ProjectReportState{}
	if reportStore != nil {
		reports, err = reportStore.ListProjectReportStatesByContracts(ctx, projectBaseContracts(bases))
		if err != nil {
			return nil, err
		}
	}
	items := make([]*v1alpha1.ProjectListItem, 0, len(bases))
	for _, base := range bases {
		var reportPtr *appstore.ProjectReportState
		if report, ok := reports[base.Contract]; ok {
			item := report
			reportPtr = &item
		}
		items = append(items, projectBaseAndReportToListItem(base, reportPtr))
	}
	s.publishProjectRefreshEventsAsync(projectBaseContracts(bases), model.ProjectDiscoverySourceFollowHeads)

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
	project, ok, err := s.projectFromComponentCache(ctx, contract)
	cacheMiss := !ok
	if err == nil && !ok {
		project, ok, err = s.loadProjectInitialFromDB(ctx, contract)
	}
	projectSnapshotLatency.Observe(float64(time.Since(startedAt).Milliseconds()))
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, status.Errorf(codes.NotFound, "project %q not found", req.GetContract())
	}
	if cacheMiss {
		s.publishProjectRefreshEventsAsync([]common.Address{contract}, model.ProjectDiscoverySourceFollowHeads)
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

func parseProjectContract(value string) (common.Address, error) {
	if !common.IsHexAddress(value) {
		return common.Address{}, status.Errorf(codes.InvalidArgument, "invalid contract %q", value)
	}
	return common.HexToAddress(value), nil
}

func (s *Service) projectFromComponentCache(ctx context.Context, contract common.Address) (*Project, bool, error) {
	if s.componentCache == nil {
		return nil, false, nil
	}
	base, ok, err := s.componentCache.GetBase(ctx, contract)
	if err != nil {
		return nil, false, err
	}
	if !ok || base == nil {
		return nil, false, nil
	}
	project := projectFromBase(*base)

	chainState, chainOK, err := s.componentCache.GetChainState(ctx, contract)
	if err != nil {
		return nil, false, err
	}
	if chainOK && chainState != nil {
		applyProjectChainState(project, chainState)
	}
	simulation, simulationOK, err := s.componentCache.GetSimulation(ctx, contract)
	if err != nil {
		return nil, false, err
	}
	if simulationOK && simulation != nil {
		project.Meta.CreatorResult = simulation.Result
	}
	report, reportOK, err := s.componentCache.GetReport(ctx, contract)
	if err != nil {
		return nil, false, err
	}
	if reportOK && report != nil {
		project.Report = report.Report
	}
	aveDetail, aveOK, err := s.componentCache.GetAveDetail(ctx, contract)
	if err != nil {
		return nil, false, err
	}
	if aveOK && aveDetail != nil {
		project.AveDetail = projectAveDetailFromStore(*aveDetail)
	}
	genesisWallets, genesisOK, err := s.componentCache.GetGenesisWallets(ctx, contract)
	if err != nil {
		return nil, false, err
	}
	if genesisOK {
		project.Meta.GenesisWallets = genesisWalletMetasFromStore(genesisWallets)
	}
	creatorHistory, creatorOK, err := s.componentCache.GetCreatorHistory(ctx, contract)
	if err != nil {
		return nil, false, err
	}
	if creatorOK {
		project.Meta.CreatorHistoricalProjects = creatorHistoricalProjectContractsFromStore(creatorHistory)
	}
	if state, ok, err := s.componentCache.GetComponentState(ctx, contract, appstore.ProjectComponentGenesisWallet); err != nil {
		return nil, false, err
	} else if ok && state != nil {
		project.Meta.GenesisWalletsFetchedAt = state.LastSuccessAt
	}
	if state, ok, err := s.componentCache.GetComponentState(ctx, contract, appstore.ProjectComponentCreatorHistory); err != nil {
		return nil, false, err
	} else if ok && state != nil {
		project.Meta.CreatorHistoricalProjectsFetchedAt = state.LastSuccessAt
	}
	return project, true, nil
}

func (s *Service) loadProjectInitialFromDB(ctx context.Context, contract common.Address) (*Project, bool, error) {
	baseStore, ok := s.dbStore.(appstore.ProjectBaseStore)
	if !ok || baseStore == nil {
		return nil, false, status.Error(codes.FailedPrecondition, "project base db store is not configured")
	}
	base, err := baseStore.GetProjectBaseByContract(ctx, contract)
	if err != nil {
		return nil, false, err
	}
	if base == nil {
		return nil, false, nil
	}
	project := projectFromBase(*base)

	var chainState *appstore.ProjectChainState
	if store, ok := s.dbStore.(appstore.ProjectChainStateStore); ok && store != nil {
		item, err := store.GetProjectChainState(ctx, contract)
		if err != nil {
			return nil, false, err
		}
		chainState = item
		if chainState != nil {
			applyProjectChainState(project, chainState)
		}
	}

	var simulation *appstore.ProjectSimulationResult
	if store, ok := s.dbStore.(appstore.ProjectSimulationStore); ok && store != nil {
		item, err := store.GetProjectSimulationResult(ctx, contract)
		if err != nil {
			return nil, false, err
		}
		simulation = item
		if simulation != nil {
			project.Meta.CreatorResult = simulation.Result
		}
	}

	var report *appstore.ProjectReportState
	if store, ok := s.dbStore.(appstore.ProjectReportStore); ok && store != nil {
		item, err := store.GetProjectReportState(ctx, contract)
		if err != nil {
			return nil, false, err
		}
		report = item
		if report != nil {
			project.Report = report.Report
		}
	}

	var aveDetail *appstore.ProjectAveDetail
	if store, ok := s.dbStore.(appstore.ProjectAveDetailStore); ok && store != nil {
		item, err := store.GetProjectAveDetail(ctx, contract)
		if err != nil {
			return nil, false, err
		}
		aveDetail = item
		if aveDetail != nil {
			project.AveDetail = projectAveDetailFromStore(*aveDetail)
		}
	}

	var genesisWallets []appstore.ProjectGenesisWallet
	if store, ok := s.dbStore.(appstore.ProjectGenesisWalletStore); ok && store != nil {
		items, err := store.ListProjectGenesisWalletsByContract(ctx, contract)
		if err != nil {
			return nil, false, err
		}
		genesisWallets = items
		project.Meta.GenesisWallets = genesisWalletMetasFromStore(items)
	}
	var creatorHistory []appstore.ProjectCreatorHistoricalProject
	if store, ok := s.dbStore.(appstore.ProjectCreatorHistoricalProjectStore); ok && store != nil {
		items, err := store.ListProjectCreatorHistoricalProjectsByContract(ctx, contract)
		if err != nil {
			return nil, false, err
		}
		creatorHistory = items
		project.Meta.CreatorHistoricalProjects = creatorHistoricalProjectContractsFromStore(items)
	}

	var genesisState *appstore.ProjectComponentState
	var creatorState *appstore.ProjectComponentState
	if stateStore, ok := s.dbStore.(appstore.ProjectComponentStateStore); ok && stateStore != nil {
		state, err := stateStore.GetProjectComponentState(ctx, contract, appstore.ProjectComponentGenesisWallet)
		if err != nil {
			return nil, false, err
		}
		genesisState = state
		if genesisState != nil {
			project.Meta.GenesisWalletsFetchedAt = genesisState.LastSuccessAt
		}
		state, err = stateStore.GetProjectComponentState(ctx, contract, appstore.ProjectComponentCreatorHistory)
		if err != nil {
			return nil, false, err
		}
		creatorState = state
		if creatorState != nil {
			project.Meta.CreatorHistoricalProjectsFetchedAt = creatorState.LastSuccessAt
		}
	}

	if s.componentCache != nil {
		if err := s.componentCache.SetBase(ctx, *base); err != nil {
			return nil, false, err
		}
		if chainState != nil {
			if err := s.componentCache.SetChainState(ctx, *chainState); err != nil {
				return nil, false, err
			}
		}
		if simulation != nil {
			if err := s.componentCache.SetSimulation(ctx, *simulation); err != nil {
				return nil, false, err
			}
		}
		if report != nil {
			if err := s.componentCache.SetReport(ctx, *report); err != nil {
				return nil, false, err
			}
		}
		if aveDetail != nil {
			if err := s.componentCache.SetAveDetail(ctx, contract, *aveDetail); err != nil {
				return nil, false, err
			}
		}
		if genesisWallets != nil {
			if err := s.componentCache.SetGenesisWallets(ctx, contract, genesisWallets); err != nil {
				return nil, false, err
			}
		}
		if creatorHistory != nil {
			if err := s.componentCache.SetCreatorHistory(ctx, contract, creatorHistory); err != nil {
				return nil, false, err
			}
		}
		if genesisState != nil {
			if err := s.componentCache.SetComponentState(ctx, *genesisState); err != nil {
				return nil, false, err
			}
		}
		if creatorState != nil {
			if err := s.componentCache.SetComponentState(ctx, *creatorState); err != nil {
				return nil, false, err
			}
		}
	}
	return project, true, nil
}

func projectFromBase(base appstore.ProjectBase) *Project {
	return &Project{
		Meta: ProjectMeta{
			BlockTime:   base.BlockTime,
			BlockNumber: base.BlockNumber,
			Contract:    base.Contract,
			Creator:     base.Creator,
			FetchAt:     base.CreatedAt,
			TxHash:      base.TxHash,
			TxIndex:     base.TxIndex,
		},
	}
}

func applyProjectChainState(project *Project, item *appstore.ProjectChainState) {
	if project == nil || item == nil {
		return
	}
	project.Meta.FetchAt = item.FetchedAt
	project.Meta.ChainState = item.ChainState
	project.Meta.WethPair = item.WethPair
	project.Meta.UsdtPair = item.UsdtPair
}

func (s *Service) publishProjectRefreshEventsAsync(contracts []common.Address, source model.ProjectDiscoverySource) {
	if s == nil || s.componentEventBus == nil {
		return
	}
	candidates := s.nextProjectRefreshContracts(uniqueNonZeroContracts(contracts))
	if len(candidates) == 0 {
		return
	}
	go func(items []common.Address, refreshSource model.ProjectDiscoverySource) {
		ctx, cancel := context.WithTimeout(context.Background(), projectRefreshPublishTimeout)
		defer cancel()
		for _, contract := range items {
			if err := s.componentEventBus.Publish(ctx, appcomponents.ProjectRefreshEvent(contract, refreshSource)); err != nil {
				log.WithFields(log.Fields{
					"contract": contract.Hex(),
					"source":   refreshSource,
				}).WithError(err).Warn("failed to publish project refresh event")
			}
		}
	}(candidates, source)
}

func (s *Service) nextProjectRefreshContracts(contracts []common.Address) []common.Address {
	if len(contracts) == 0 {
		return nil
	}
	window := s.projectRefreshWindow
	if window <= 0 {
		window = projectRefreshThrottleInterval
	}
	now := time.Now().UTC()
	s.projectRefreshMu.Lock()
	defer s.projectRefreshMu.Unlock()
	if s.projectRefreshLastRun == nil {
		s.projectRefreshLastRun = map[common.Address]time.Time{}
	}
	items := make([]common.Address, 0, len(contracts))
	for _, contract := range contracts {
		last := s.projectRefreshLastRun[contract]
		if !last.IsZero() && now.Sub(last) < window {
			continue
		}
		s.projectRefreshLastRun[contract] = now
		items = append(items, contract)
	}
	return items
}

func uniqueNonZeroContracts(contracts []common.Address) []common.Address {
	seen := make(map[common.Address]struct{}, len(contracts))
	items := make([]common.Address, 0, len(contracts))
	for _, contract := range contracts {
		if contract == (common.Address{}) {
			continue
		}
		if _, ok := seen[contract]; ok {
			continue
		}
		seen[contract] = struct{}{}
		items = append(items, contract)
	}
	return items
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
