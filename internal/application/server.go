package application

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	applicationpkg "github.com/useryege/athena/internal/application/apiclient"
	"github.com/useryege/athena/internal/application/redisport"
	appstore "github.com/useryege/athena/internal/application/store"
	"github.com/useryege/athena/internal/server/version"
	versionpkg "github.com/useryege/athena/pkg/apiclient/version"
	"github.com/useryege/athena/util/deepseek"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
)

type ApplicationServer struct {
	ApplicationServerOpts
	service *Service
}

type ApplicationServerOpts struct {
	NodeClient          *ethclient.Client
	AthenaContract      common.Address
	EtherscanAPIBaseURL string
	EtherscanAPIKey     string
	DeepSeekConfig      deepseek.Config
	Store               appstore.Store
	LiquidityLocker     []common.Address
	RedisClient         redisport.Client

	// Fetch from Athena contract
	V2FactoryContract common.Address
	WethContract      common.Address
	UsdtContract      common.Address
	WethDecimals      uint8
	UsdtDecimals      uint8
}

func NewServer(opts ApplicationServerOpts) (*ApplicationServer, error) {
	service, err := NewService(opts.NodeClient,
		opts.V2FactoryContract,
		opts.WethContract,
		opts.UsdtContract,
		opts.WethDecimals,
		opts.UsdtDecimals,
		opts.AthenaContract,
		opts.EtherscanAPIBaseURL,
		opts.EtherscanAPIKey,
		opts.DeepSeekConfig,
		opts.Store,
		opts.LiquidityLocker,
		opts.RedisClient)
	if err != nil {
		return nil, err
	}
	return &ApplicationServer{
		ApplicationServerOpts: opts,
		service:               service,
	}, nil
}

// CreateGRPC creates a new gRPC server.
func (a *ApplicationServer) CreateGRPC() *grpc.Server {
	server := grpc.NewServer(grpc.MaxRecvMsgSize(applicationpkg.MaxGRPCMessageSize))

	// register the version service to the gRPC server
	versionService := version.NewServer(nil, func() (bool, error) {
		return true, nil
	})
	versionpkg.RegisterVersionServiceServer(server, versionService)

	// register the project controller service to the gRPC server
	applicationpkg.RegisterApplicationServiceServer(server, a.service)

	// register the health service to the gRPC server
	healthService := health.NewServer()
	grpc_health_v1.RegisterHealthServer(server, healthService)
	return server
}

func (a *ApplicationServer) Start() error {
	if err := a.service.Start(); err != nil {
		return err
	}
	return nil
}

func (a *ApplicationServer) Stop() error {
	if err := a.service.Stop(); err != nil {
		return err
	}
	return nil
}
