package api

import (
	"context"
	"errors"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	applicationpkg "github.com/useryege/athena/internal/application/apiclient"
	appcache "github.com/useryege/athena/internal/application/cache"
	"github.com/useryege/athena/internal/application/discovery"
	appstore "github.com/useryege/athena/internal/application/store"
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

	aveConfig       ave.Config
	liquidityLocker []common.Address
	apiFetcher      ethereumapi.EthereumAPI

	store          appstore.Store
	componentCache appcache.ProjectComponentCache

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
	discoveryIndexer        discovery.ProjectDiscoveryIndexer
	discoveryStop           context.CancelFunc
	discoveryStarted        bool
	discoveryIndexerFactory func() (discovery.ProjectDiscoveryIndexer, error)
}

type ServiceOpts struct {
	NodeClient        *ethclient.Client
	V2FactoryContract common.Address
	WethContract      common.Address
	UsdtContract      common.Address
	WethDecimals      uint8
	UsdtDecimals      uint8
	AthenaContract    common.Address
	ChainID           int64
	AveConfig         ave.Config
	Store             appstore.Store
	LiquidityLocker   []common.Address
	RedisClient       redisport.Client
	APIFetcher        ethereumapi.EthereumAPI
	CodeAtFunc        func(ctx context.Context, contract common.Address) ([]byte, error)
}

func NewService(opts ServiceOpts) (*Service, error) {
	if opts.Store == nil {
		return nil, errors.New("application store is nil")
	}
	return &Service{
		nodeClient:        opts.NodeClient,
		store:             opts.Store,
		componentCache:    appcache.NewProjectComponentCache(opts.RedisClient),
		v2FactoryContract: opts.V2FactoryContract,
		wethContract:      opts.WethContract,
		usdtContract:      opts.UsdtContract,
		wethDecimals:      opts.WethDecimals,
		usdtDecimals:      opts.UsdtDecimals,
		athenaContract:    opts.AthenaContract,
		chainID:           opts.ChainID,
		aveConfig:         opts.AveConfig,
		liquidityLocker:   opts.LiquidityLocker,
		apiFetcher:        opts.APIFetcher,
		codeAtFunc:        opts.CodeAtFunc,
		discoveryIntake:   discovery.NewDiscoveryIntake(opts.Store, opts.ChainID),
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

	s.startStopMu.Lock()
	if !s.starting {
		s.startStopMu.Unlock()
		cancel()
		return context.Canceled
	}
	s.lifecycleCtx = ctx
	s.lifecycleStop = cancel
	s.bootstrapStop = nil
	s.starting = false
	s.started = true
	s.startStopMu.Unlock()

	return nil
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
	discoveryIndexer, discoveryStop := s.stopProjectDiscoveryLocked()

	s.lifecycleCtx = nil
	s.lifecycleStop = nil
	s.bootstrapStop = nil
	s.starting = false
	s.started = false
	s.startStopMu.Unlock()

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

	return discoveryErr
}

func (s *Service) stopProjectDiscoveryLocked() (discovery.ProjectDiscoveryIndexer, context.CancelFunc) {
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
	if s.discoveryIndexerFactory == nil {
		return nil, status.Error(codes.FailedPrecondition, "application discovery is handled by chain-ingestor and kafka-consumer modes")
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

func (s *Service) newProjectDiscoveryIndexer() (discovery.ProjectDiscoveryIndexer, error) {
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
