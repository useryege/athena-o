package tokenapi

import (
	"context"

	"github.com/useryege/athena/internal/server/version"
	tokenstore "github.com/useryege/athena/internal/token/store"
	"github.com/useryege/athena/internal/tokenapi/apiclient"
	versionpkg "github.com/useryege/athena/pkg/apiclient/version"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
)

type Server struct {
	ServerOpts
	service       *Service
	healthService *health.Server
	store         *tokenstore.SQLStore
}

type ServerOpts struct {
	StoreSrc func(context.Context) (*tokenstore.SQLStore, error)
}

func NewServer(opts ServerOpts) (*Server, error) {
	healthService := health.NewServer()
	healthService.SetServingStatus("", grpc_health_v1.HealthCheckResponse_NOT_SERVING)
	return &Server{
		ServerOpts:    opts,
		service:       NewService(),
		healthService: healthService,
	}, nil
}

func (s *Server) CreateGRPC() *grpc.Server {
	server := grpc.NewServer(grpc.MaxRecvMsgSize(apiclient.MaxGRPCMessageSize))
	versionService := version.NewServer(nil, func() (bool, error) {
		return true, nil
	})
	versionpkg.RegisterVersionServiceServer(server, versionService)
	apiclient.RegisterTokenAPIServiceServer(server, s.service)
	grpc_health_v1.RegisterHealthServer(server, s.healthService)
	return server
}

func (s *Server) Start(ctx context.Context) error {
	if s.StoreSrc != nil {
		store, err := s.StoreSrc(ctx)
		if err != nil {
			return err
		}
		s.store = store
		s.service.SetStore(store)
	}
	if err := s.service.Start(); err != nil {
		if s.store != nil {
			_ = s.store.Close()
			s.store = nil
			s.service.SetStore(nil)
		}
		return err
	}
	s.setHealthStatus(grpc_health_v1.HealthCheckResponse_SERVING)
	return nil
}

func (s *Server) Stop() error {
	s.setHealthStatus(grpc_health_v1.HealthCheckResponse_NOT_SERVING)
	err := s.service.Stop()
	if s.store != nil {
		closeErr := s.store.Close()
		s.store = nil
		s.service.SetStore(nil)
		if err == nil {
			err = closeErr
		}
	}
	return err
}

func (s *Server) setHealthStatus(status grpc_health_v1.HealthCheckResponse_ServingStatus) {
	if s.healthService != nil {
		s.healthService.SetServingStatus("", status)
	}
}
