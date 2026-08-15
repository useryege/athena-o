package sportslive

import (
	notificationapiclient "github.com/useryege/athena/internal/notification/apiclient"
	"github.com/useryege/athena/internal/server/version"
	"github.com/useryege/athena/internal/sportslive/apiclient"
	sportslivestore "github.com/useryege/athena/internal/sportslive/store"
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
	Store                  *sportslivestore.SQLStore
	NotificationClientset  notificationapiclient.Clientset
	NotificationInviteCode string
	PriceAlertsConfig      SportsLivePriceAlertsConfig
	ScoreAlertsConfig      SportsLiveScoreAlertsConfig
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
			WithPriceAlertsConfig(opts.PriceAlertsConfig),
			WithScoreAlertsConfig(opts.ScoreAlertsConfig),
		),
		healthService: healthService,
	}, nil
}

func (s *Server) CreateGRPC() *grpc.Server {
	server := grpc.NewServer(grpc.MaxRecvMsgSize(apiclient.MaxGRPCMessageSize))
	versionpkg.RegisterVersionServiceServer(server, version.NewServer(nil, func() (bool, error) { return true, nil }))
	apiclient.RegisterSportsLiveServiceServer(server, s.service)
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
