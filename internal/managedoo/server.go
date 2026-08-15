package managedoo

import (
	"github.com/useryege/athena/internal/managedoo/apiclient"
	managedoostore "github.com/useryege/athena/internal/managedoo/store"
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
	Store                  *managedoostore.SQLStore
	NotificationClientset  notificationapiclient.Clientset
	NotificationInviteCode string
	PolygonRPCURL          string
	ProposedAlertsConfig   ManagedOOProposedAlertsConfig
	DisputedAlertsConfig   ManagedOODisputedAlertsConfig
}

func NewServer(opts ServerOpts) (*Server, error) {
	healthService := health.NewServer()
	healthService.SetServingStatus("", grpc_health_v1.HealthCheckResponse_NOT_SERVING)
	return &Server{
		ServerOpts: opts,
		service: NewService(
			opts.Store,
			WithNotificationClientset(opts.NotificationClientset),
			WithNotificationInviteCode(opts.NotificationInviteCode),
			WithPolygonRPCURL(opts.PolygonRPCURL),
			WithProposedAlertsConfig(opts.ProposedAlertsConfig),
			WithDisputedAlertsConfig(opts.DisputedAlertsConfig),
		),
		healthService: healthService,
	}, nil
}

func (s *Server) CreateGRPC() *grpc.Server {
	server := grpc.NewServer(grpc.MaxRecvMsgSize(apiclient.MaxGRPCMessageSize))
	versionpkg.RegisterVersionServiceServer(server, version.NewServer(nil, func() (bool, error) { return true, nil }))
	apiclient.RegisterManagedOOServiceServer(server, s.service)
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
