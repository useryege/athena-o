package profitsharing

import (
	"github.com/useryege/athena/internal/profitsharing/apiclient"
	profitsharingstore "github.com/useryege/athena/internal/profitsharing/store"
	"github.com/useryege/athena/internal/server/version"
	versionpkg "github.com/useryege/athena/pkg/apiclient/version"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
)

type ServerOpts struct {
	Store *profitsharingstore.SQLStore
}

type Server struct {
	ServerOpts
	service       *Service
	healthService *health.Server
}

func NewServer(opts ServerOpts) (*Server, error) {
	healthService := health.NewServer()
	healthService.SetServingStatus("", grpc_health_v1.HealthCheckResponse_NOT_SERVING)
	return &Server{
		ServerOpts:    opts,
		service:       NewService(opts.Store),
		healthService: healthService,
	}, nil
}

func (s *Server) CreateGRPC() *grpc.Server {
	server := grpc.NewServer(grpc.MaxRecvMsgSize(apiclient.MaxGRPCMessageSize))
	versionpkg.RegisterVersionServiceServer(server, version.NewServer(nil, func() (bool, error) { return true, nil }))
	apiclient.RegisterProfitSharingServiceServer(server, s.service)
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

func (s *Server) setHealthStatus(servingStatus grpc_health_v1.HealthCheckResponse_ServingStatus) {
	if s.healthService != nil {
		s.healthService.SetServingStatus("", servingStatus)
	}
}
