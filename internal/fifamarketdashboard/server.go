package fifamarketdashboard

import (
	"time"

	"github.com/useryege/athena/internal/fifamarketdashboard/apiclient"
	fifamarketdashboardstore "github.com/useryege/athena/internal/fifamarketdashboard/store"
	"github.com/useryege/athena/internal/server/version"
	walletapiclient "github.com/useryege/athena/internal/wallet/apiclient"
	wormmarketsapiclient "github.com/useryege/athena/internal/wormmarkets/apiclient"
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
	Store                        *fifamarketdashboardstore.SQLStore
	WormMarketsClientset         wormmarketsapiclient.Clientset
	WalletClientset              walletapiclient.Clientset
	FIFADashboardRefreshInterval time.Duration
	FIFAWalletBalanceConfig      FIFAWalletBalanceConfig
}

func NewServer(opts ServerOpts) (*Server, error) {
	healthService := health.NewServer()
	healthService.SetServingStatus("", grpc_health_v1.HealthCheckResponse_NOT_SERVING)
	return &Server{
		ServerOpts: opts,
		service: NewService(
			opts.Store,
			WithWormMarketsClientset(opts.WormMarketsClientset),
			WithWalletClientset(opts.WalletClientset),
			WithFIFADashboardRefreshInterval(opts.FIFADashboardRefreshInterval),
			WithFIFAWalletBalanceConfig(opts.FIFAWalletBalanceConfig),
		),
		healthService: healthService,
	}, nil
}

func (s *Server) CreateGRPC() *grpc.Server {
	server := grpc.NewServer(grpc.MaxRecvMsgSize(apiclient.MaxGRPCMessageSize))
	versionService := version.NewServer(nil, func() (bool, error) {
		return true, nil
	})
	versionpkg.RegisterVersionServiceServer(server, versionService)
	apiclient.RegisterFIFAMarketDashboardServiceServer(server, s.service)
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
