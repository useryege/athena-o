package application

import (
	"database/sql"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	applicationpkg "github.com/useryege/athena/internal/application/apiclient"
	"github.com/useryege/athena/internal/server/version"
	versionpkg "github.com/useryege/athena/pkg/apiclient/version"
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
	V2FactoryContract   common.Address
	WethContract        common.Address
	EtherscanAPIBaseURL string
	EtherscanAPIKey     string
	PostgresDB          *sql.DB
}

func NewServer(opts ApplicationServerOpts) *ApplicationServer {
	if opts.WethContract == (common.Address{}) {
		panic("WETH contract address is required.Set it by --weth-contract flag or ATHENA_APPLICATION_WETH_CONTRACT environment variable")
	}
	if opts.V2FactoryContract == (common.Address{}) {
		panic("V2 Factory contract address is required.Set it by --v2-factory-contract flag or ATHENA_APPLICATION_V2_FACTORY_CONTRACT environment variable")
	}
	if opts.EtherscanAPIBaseURL == "" {
		panic("Etherscan API base URL is required.Set it by --etherscan-api-base-url flag or ATHENA_APPLICATION_ETHERSCAN_API_BASE_URL environment variable")
	}
	if opts.EtherscanAPIKey == "" {
		panic("Etherscan API key is required.Set it by --etherscan-api-key flag or ATHENA_APPLICATION_ETHERSCAN_API_KEY environment variable")
	}

	return &ApplicationServer{
		ApplicationServerOpts: opts,
		service:               NewService(opts.NodeClient, opts.V2FactoryContract, opts.WethContract, opts.EtherscanAPIBaseURL, opts.EtherscanAPIKey, NewProjectMetaStore(opts.PostgresDB)),
	}
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
