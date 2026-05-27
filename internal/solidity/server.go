package solidity

import (
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/useryege/athena/internal/server/version"
	"github.com/useryege/athena/internal/solidity/apiclient"
	"github.com/useryege/athena/internal/solidity/sourcequality"
	soliditystore "github.com/useryege/athena/internal/solidity/store"
	versionpkg "github.com/useryege/athena/pkg/apiclient/version"
	"github.com/useryege/athena/util/ethereumapi"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
)

type Server struct {
	ServerOpts
	service       *Service
	healthService *health.Server
}

type ServerOpts struct {
	Store                 *soliditystore.SQLStore
	NodeClient            *ethclient.Client
	ChainID               int64
	APIFetcher            ethereumapi.EthereumAPI
	SourceQualityAnalyzer sourcequality.Analyzer
}

func NewServer(opts ServerOpts) (*Server, error) {
	healthService := health.NewServer()
	healthService.SetServingStatus("", grpc_health_v1.HealthCheckResponse_NOT_SERVING)
	return &Server{
		ServerOpts:    opts,
		service:       NewService(ServiceOpts{Store: opts.Store, NodeClient: opts.NodeClient, ChainID: opts.ChainID, APIFetcher: opts.APIFetcher, SourceQualityAnalyzer: opts.SourceQualityAnalyzer}),
		healthService: healthService,
	}, nil
}

func (s *Server) CreateGRPC() *grpc.Server {
	server := grpc.NewServer(grpc.MaxRecvMsgSize(apiclient.MaxGRPCMessageSize))
	versionService := version.NewServer(nil, func() (bool, error) {
		return true, nil
	})
	versionpkg.RegisterVersionServiceServer(server, versionService)
	apiclient.RegisterSolidityServiceServer(server, s.service)
	grpc_health_v1.RegisterHealthServer(server, s.healthService)
	return server
}

func (s *Server) Start() error {
	if err := s.service.Start(); err != nil {
		return err
	}
	s.setHealthStatus(grpc_health_v1.HealthCheckResponse_SERVING)
	return nil
}

func (s *Server) Stop() error {
	s.setHealthStatus(grpc_health_v1.HealthCheckResponse_NOT_SERVING)
	return s.service.Stop()
}

func (s *Server) setHealthStatus(status grpc_health_v1.HealthCheckResponse_ServingStatus) {
	if s.healthService != nil {
		s.healthService.SetServingStatus("", status)
	}
}
