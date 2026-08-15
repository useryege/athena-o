package marketradar

import (
	"github.com/useryege/athena/internal/marketradar/apiclient"
	notificationapiclient "github.com/useryege/athena/internal/notification/apiclient"
	"github.com/useryege/athena/internal/server/version"
	versionpkg "github.com/useryege/athena/pkg/apiclient/version"
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
	NotificationClientset  notificationapiclient.Clientset
	NotificationInviteCode string
	MoverAlertsConfig      MoverAlertsConfig
}

func NewServer(opts ServerOpts) (*Server, error) {
	healthService := health.NewServer()
	healthService.SetServingStatus("", grpc_health_v1.HealthCheckResponse_NOT_SERVING)
	return &Server{
		ServerOpts: opts,
		service: NewService(
			WithNotificationClientset(opts.NotificationClientset),
			WithNotificationInviteCode(opts.NotificationInviteCode),
			WithMoverAlertsConfig(opts.MoverAlertsConfig),
		),
		healthService: healthService,
	}, nil
}

func (s *Server) CreateGRPC() *grpc.Server {
	server := grpc.NewServer(grpc.MaxRecvMsgSize(apiclient.MaxGRPCMessageSize))
	versionpkg.RegisterVersionServiceServer(server, version.NewServer(nil, func() (bool, error) { return true, nil }))
	apiclient.RegisterMarketRadarServiceServer(server, s.service)
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
