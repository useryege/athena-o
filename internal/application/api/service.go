package api

import (
	"context"
	"errors"
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
	"github.com/useryege/athena/internal/application/persistence"
	"github.com/useryege/athena/internal/application/pipeline"
	"github.com/useryege/athena/internal/application/sourcequality"
	appstore "github.com/useryege/athena/internal/application/store"
	walletapiclient "github.com/useryege/athena/internal/wallet/apiclient"
	"github.com/useryege/athena/pkg/apis/application/v1alpha1"
	"github.com/useryege/athena/util/ave"
	"github.com/useryege/athena/util/ethereumapi"
	"github.com/useryege/athena/util/redisport"
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
	chainID           int64

	aveConfig             ave.Config
	liquidityLocker       []common.Address
	apiFetcher            ethereumapi.EthereumAPI
	sourceQualityAnalyzer sourcequality.Analyzer

	pipeline        *pipeline.ProjectPipeline
	athenaFetcher   evm.AthenaFetcher
	aveComponent    *avecomponent.Component
	walletBlacklist walletBlacklistLister

	store             appstore.Store
	componentCache    appcache.ProjectComponentCache
	componentEventBus appcomponents.EventBus
	persistenceFlush  *persistence.Flusher

	delayedFetchSem chan struct{}
	codeAtFunc      func(ctx context.Context, contract common.Address) ([]byte, error)
	startStopMu     sync.Mutex
	lifecycleCtx    context.Context
	lifecycleStop   context.CancelFunc
	bootstrapStop   context.CancelFunc
	starting        bool
	started         bool

	discoveryMu             sync.Mutex
	discoveryIntake         discovery.DiscoveryIntake
	discoveryIndexer        pipeline.ProjectDiscoveryIndexer
	discoveryStop           context.CancelFunc
	discoveryStarted        bool
	discoveryIndexerFactory func() (pipeline.ProjectDiscoveryIndexer, error)
}

type ServiceOpts struct {
	NodeClient            *ethclient.Client
	V2FactoryContract     common.Address
	WethContract          common.Address
	UsdtContract          common.Address
	WethDecimals          uint8
	UsdtDecimals          uint8
	AthenaContract        common.Address
	ChainID               int64
	AveConfig             ave.Config
	Store                 appstore.Store
	LiquidityLocker       []common.Address
	RedisClient           redisport.Client
	APIFetcher            ethereumapi.EthereumAPI
	SourceQualityAnalyzer sourcequality.Analyzer
	WalletClientset       walletapiclient.Clientset
	CodeAtFunc            func(ctx context.Context, contract common.Address) ([]byte, error)
}

func NewService(opts ServiceOpts) (*Service, error) {
	bufferedStore, err := persistence.NewRedisBufferedStore(opts.Store, opts.RedisClient)
	if err != nil {
		return nil, err
	}
	return &Service{
		nodeClient:            opts.NodeClient,
		store:                 bufferedStore,
		componentCache:        appcache.NewProjectComponentCache(opts.RedisClient),
		componentEventBus:     appcomponents.NewRedisEventBus(opts.RedisClient),
		persistenceFlush:      persistence.NewFlusher(bufferedStore, time.Minute),
		walletBlacklist:       newWalletBlacklistClientLister(opts.WalletClientset),
		v2FactoryContract:     opts.V2FactoryContract,
		wethContract:          opts.WethContract,
		usdtContract:          opts.UsdtContract,
		wethDecimals:          opts.WethDecimals,
		usdtDecimals:          opts.UsdtDecimals,
		athenaContract:        opts.AthenaContract,
		chainID:               opts.ChainID,
		aveConfig:             opts.AveConfig,
		liquidityLocker:       opts.LiquidityLocker,
		apiFetcher:            opts.APIFetcher,
		sourceQualityAnalyzer: opts.SourceQualityAnalyzer,
		codeAtFunc:            opts.CodeAtFunc,
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

	pipeline, athenaFetcher, aveComponent, discoveryIntake, err := s.startWithContext(ctx)
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
	s.discoveryIntake = discoveryIntake
	s.lifecycleCtx = ctx
	s.lifecycleStop = cancel
	s.bootstrapStop = nil
	s.starting = false
	s.started = true
	s.startStopMu.Unlock()

	return nil
}

func (s *Service) startWithContext(ctx context.Context) (projectPipeline *pipeline.ProjectPipeline, athenaFetcher evm.AthenaFetcher, aveComponent *avecomponent.Component, discoveryIntake discovery.DiscoveryIntake, err error) {
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
		return nil, nil, nil, nil, err
	}

	chainID, err := s.nodeClient.ChainID(ctx)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	s.chainID = chainID.Int64()
	if _, err := s.store.EnsureDefaultSourceQualityPrompt(ctx, "Default Solidity Source Quality Prompt", sourcequality.DefaultSystemPrompt); err != nil {
		return nil, nil, nil, nil, err
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
		return nil, nil, nil, nil, err
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
		Store:    componentStore,
		Cache:    s.componentCache,
		Resolver: s,
		ChainID:  chainID.Int64(),
		Bus:      s.componentEventBus,
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
	discoveryIntake = discovery.NewDiscoveryIntake(initializer)

	projectPipeline = pipeline.NewProjectPipeline(
		s.persistenceFlush,
		initializer,
		chainStateComponent,
		simulationComponent,
		genesisWalletComponent,
		creatorHistoryComponent,
		bytecodeComponent,
		reportComponent,
	)
	if err := projectPipeline.Start(ctx); err != nil {
		return nil, nil, nil, nil, err
	}
	if aveComponent != nil {
		if err := aveComponent.Start(ctx); err != nil {
			_ = projectPipeline.Stop()
			return nil, nil, nil, nil, err
		}
	}
	return projectPipeline, athenaFetcher, aveComponent, discoveryIntake, nil
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
	discoveryIndexer, discoveryStop := s.stopProjectDiscoveryLocked()

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
	if discoveryStop != nil {
		discoveryStop()
	}
	discoveryErr := error(nil)
	if discoveryIndexer != nil {
		discoveryErr = discoveryIndexer.Stop()
	}
	if stop != nil {
		stop()
	}
	if pipeline != nil {
		pipelineErr = pipeline.Stop()
	}

	return errors.Join(discoveryErr, pipelineErr)
}

func (s *Service) clearPipelineLocked() {
	s.pipeline = nil
	s.athenaFetcher = nil
	s.aveComponent = nil
	s.discoveryIntake = nil
}

func (s *Service) stopProjectDiscoveryLocked() (pipeline.ProjectDiscoveryIndexer, context.CancelFunc) {
	s.discoveryMu.Lock()
	defer s.discoveryMu.Unlock()
	indexer := s.discoveryIndexer
	stop := s.discoveryStop
	s.discoveryIndexer = nil
	s.discoveryStop = nil
	s.discoveryStarted = false
	return indexer, stop
}

func (s *Service) GetProjectDiscoveryStatus(context.Context, *applicationpkg.GetProjectDiscoveryStatusRequest) (*v1alpha1.ProjectDiscoveryStatus, error) {
	s.discoveryMu.Lock()
	started := s.discoveryStarted
	s.discoveryMu.Unlock()
	return projectDiscoveryStatus(started), nil
}

func (s *Service) StartProjectDiscovery(_ context.Context, _ *applicationpkg.StartProjectDiscoveryRequest) (*v1alpha1.ProjectDiscoveryStatus, error) {
	s.startStopMu.Lock()
	defer s.startStopMu.Unlock()
	if !s.started || s.lifecycleCtx == nil {
		return nil, status.Error(codes.FailedPrecondition, "application service is not started")
	}

	s.discoveryMu.Lock()
	defer s.discoveryMu.Unlock()
	if s.discoveryStarted {
		return projectDiscoveryStatus(true), nil
	}

	indexer, err := s.newProjectDiscoveryIndexer()
	if err != nil {
		return nil, err
	}
	runCtx, cancel := context.WithCancel(s.lifecycleCtx)
	if err := indexer.Start(runCtx); err != nil {
		cancel()
		return nil, err
	}
	s.discoveryIndexer = indexer
	s.discoveryStop = cancel
	s.discoveryStarted = true
	return projectDiscoveryStatus(true), nil
}

func (s *Service) StopProjectDiscovery(context.Context, *applicationpkg.StopProjectDiscoveryRequest) (*v1alpha1.ProjectDiscoveryStatus, error) {
	indexer, stop := s.stopProjectDiscoveryLocked()
	if stop != nil {
		stop()
	}
	if indexer != nil {
		if err := indexer.Stop(); err != nil {
			return nil, err
		}
	}
	return projectDiscoveryStatus(false), nil
}

func (s *Service) newProjectDiscoveryIndexer() (pipeline.ProjectDiscoveryIndexer, error) {
	if s.discoveryIndexerFactory != nil {
		return s.discoveryIndexerFactory()
	}
	return discovery.NewProjectDiscoveryIndexer(s.nodeClient, s.componentCache, s.store, s.discoveryIntake)
}

func projectDiscoveryStatus(started bool) *v1alpha1.ProjectDiscoveryStatus {
	statusText := "stopped"
	if started {
		statusText = "running"
	}
	return &v1alpha1.ProjectDiscoveryStatus{
		Started: started,
		Status:  statusText,
	}
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
