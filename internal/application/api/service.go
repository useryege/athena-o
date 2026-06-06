package api

import (
	"context"
	"errors"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	applicationpkg "github.com/useryege/athena/internal/application/apiclient"
	appstore "github.com/useryege/athena/internal/application/store"
	"github.com/useryege/athena/pkg/apis/application/v1alpha1"
	"github.com/useryege/athena/util/ave"
	"github.com/useryege/athena/util/ethereumapi"
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

	store appstore.Store

	delayedFetchSem chan struct{}
	codeAtFunc      func(ctx context.Context, contract common.Address) ([]byte, error)
	startStopMu     sync.Mutex
	lifecycleCtx    context.Context
	lifecycleStop   context.CancelFunc
	started         bool
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
	}, nil
}

func (s *Service) Start() error {
	s.startStopMu.Lock()
	defer s.startStopMu.Unlock()
	if s.started {
		return nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.lifecycleCtx = ctx
	s.lifecycleStop = cancel
	s.started = true
	return nil
}

func (s *Service) Stop() error {
	s.startStopMu.Lock()
	stop := s.lifecycleStop
	s.lifecycleCtx = nil
	s.lifecycleStop = nil
	s.started = false
	s.startStopMu.Unlock()
	if stop != nil {
		stop()
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
