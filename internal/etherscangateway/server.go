package etherscangateway

import (
	"context"
	"time"

	internalversion "github.com/useryege/athena/internal/server/version"
	"github.com/useryege/athena/pkg/apiclient/etherscangateway"
	versionpkg "github.com/useryege/athena/pkg/apiclient/version"
	"github.com/useryege/athena/util/etherscanapi"
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
	EtherscanBaseURL string
	AuthToken        string
	Timeout          time.Duration
}

func NewServer(opts ServerOpts) (*Server, error) {
	baseURL := opts.EtherscanBaseURL
	if baseURL == "" {
		baseURL = etherscanapi.DefaultBaseURL
	}
	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = etherscanapi.DefaultTimeout
	}
	healthService := health.NewServer()
	healthService.SetServingStatus("", grpc_health_v1.HealthCheckResponse_NOT_SERVING)
	return &Server{
		ServerOpts: ServerOpts{
			EtherscanBaseURL: baseURL,
			AuthToken:        opts.AuthToken,
			Timeout:          timeout,
		},
		service: NewService(ServiceOpts{
			EtherscanBaseURL: baseURL,
			Timeout:          timeout,
		}),
		healthService: healthService,
	}, nil
}

func (s *Server) CreateGRPC() *grpc.Server {
	server := grpc.NewServer(grpc.UnaryInterceptor(bearerAuthUnaryInterceptor(s.AuthToken)))
	versionService := internalversion.NewServer(nil, func() (bool, error) {
		return true, nil
	})
	versionpkg.RegisterVersionServiceServer(server, versionService)
	etherscangateway.RegisterEtherscanGatewayServiceServer(server, s.service)
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
