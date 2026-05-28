package application

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/useryege/athena/internal/application/api"
	applicationpkg "github.com/useryege/athena/internal/application/apiclient"
	"github.com/useryege/athena/internal/application/redisport"
	appstore "github.com/useryege/athena/internal/application/store"
	"github.com/useryege/athena/internal/server/version"
	solidityapiclient "github.com/useryege/athena/internal/solidity/apiclient"
	walletapiclient "github.com/useryege/athena/internal/wallet/apiclient"
	versionpkg "github.com/useryege/athena/pkg/apiclient/version"
	"github.com/useryege/athena/util/ave"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
)

type ApplicationServer struct {
	ApplicationServerOpts
	service       *api.Service
	healthService *health.Server
}

type ApplicationServerOpts struct {
	NodeClient        *ethclient.Client
	AthenaContract    common.Address
	AveConfig         ave.Config
	Store             appstore.Store
	LiquidityLocker   []common.Address
	RedisClient       redisport.Client
	SolidityClientset solidityapiclient.Clientset
	WalletClientset   walletapiclient.Clientset

	// Fetch from Athena contract
	V2FactoryContract common.Address
	WethContract      common.Address
	UsdtContract      common.Address
	WethDecimals      uint8
	UsdtDecimals      uint8
}

func NewServer(opts ApplicationServerOpts) (*ApplicationServer, error) {
	service, err := api.NewService(opts.NodeClient,
		opts.V2FactoryContract,
		opts.WethContract,
		opts.UsdtContract,
		opts.WethDecimals,
		opts.UsdtDecimals,
		opts.AthenaContract,
		opts.AveConfig,
		opts.Store,
		opts.LiquidityLocker,
		opts.RedisClient,
		opts.SolidityClientset,
		opts.WalletClientset)
	if err != nil {
		return nil, err
	}
	healthService := health.NewServer()
	healthService.SetServingStatus("", grpc_health_v1.HealthCheckResponse_NOT_SERVING)
	return &ApplicationServer{
		ApplicationServerOpts: opts,
		service:               service,
		healthService:         healthService,
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
	grpc_health_v1.RegisterHealthServer(server, a.healthService)
	return server
}

func (a *ApplicationServer) Start() error {
	if err := a.service.Start(); err != nil {
		return err
	}
	a.setHealthStatus(grpc_health_v1.HealthCheckResponse_SERVING)
	return nil
}

func (a *ApplicationServer) Stop() error {
	a.setHealthStatus(grpc_health_v1.HealthCheckResponse_NOT_SERVING)
	if err := a.service.Stop(); err != nil {
		return err
	}
	return nil
}

func (a *ApplicationServer) setHealthStatus(status grpc_health_v1.HealthCheckResponse_ServingStatus) {
	if a.healthService != nil {
		a.healthService.SetServingStatus("", status)
	}
}
