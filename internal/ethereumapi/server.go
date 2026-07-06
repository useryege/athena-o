package ethereumapi

import (
	"context"
	"time"

	"github.com/useryege/athena/internal/ethereumapi/apiclient"
	ethereumapistore "github.com/useryege/athena/internal/ethereumapi/store"
	"github.com/useryege/athena/internal/server/version"
	versionpkg "github.com/useryege/athena/pkg/apiclient/version"
	utilethereumapi "github.com/useryege/athena/util/ethereumapi"
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
	Store          *ethereumapistore.SQLStore
	EthereumAPI    utilethereumapi.EthereumAPI
	CacheTTL       time.Duration
	CacheRetention time.Duration
	RefreshTimeout time.Duration
}

func NewServer(opts ServerOpts) (*Server, error) {
	healthService := health.NewServer()
	healthService.SetServingStatus("", grpc_health_v1.HealthCheckResponse_NOT_SERVING)
	return &Server{
		ServerOpts: opts,
		service: NewService(ServiceOpts{
			Store:          opts.Store,
			EthereumAPI:    opts.EthereumAPI,
			CacheTTL:       opts.CacheTTL,
			CacheRetention: opts.CacheRetention,
			RefreshTimeout: opts.RefreshTimeout,
		}),
		healthService: healthService,
	}, nil
}

func (s *Server) CreateGRPC() *grpc.Server {
	server := grpc.NewServer(grpc.MaxRecvMsgSize(apiclient.MaxGRPCMessageSize))
	versionService := version.NewServer(nil, func() (bool, error) {
		return true, nil
	})
	versionpkg.RegisterVersionServiceServer(server, versionService)
	apiclient.RegisterEthereumAPIServiceServer(server, s.service)
	grpc_health_v1.RegisterHealthServer(server, s.healthService)
	return server
}

func (s *Server) Start(ctx context.Context) error {
	if err := s.service.Start(ctx); err != nil {
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
