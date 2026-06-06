package token

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/useryege/athena/internal/server/version"
	"github.com/useryege/athena/internal/token/apiclient"
	"github.com/useryege/athena/internal/token/chainingestor"
	tokenstore "github.com/useryege/athena/internal/token/store"
	versionpkg "github.com/useryege/athena/pkg/apiclient/version"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
)

const (
	ModeGRPC          = "grpc"
	ModeChainIngestor = "chain-ingestor"
)

type Server struct {
	ServerOpts
	service       *Service
	healthService *health.Server
	store         *tokenstore.SQLStore
	chainWorker   *chainingestor.Worker
}

type ServerOpts struct {
	Mode           string
	StoreSrc       func(context.Context) (*tokenstore.SQLStore, error)
	EthNodeWSURL   string
	BSCNodeWSURL   string
	NodeWSUseProxy bool
}

func NormalizeMode(mode string) string {
	mode = strings.TrimSpace(mode)
	if mode == "" {
		return ModeGRPC
	}
	return mode
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
	apiclient.RegisterTokenServiceServer(server, s.service)
	grpc_health_v1.RegisterHealthServer(server, s.healthService)
	return server
}

func (s *Server) Start(ctx context.Context) error {
	mode := NormalizeMode(s.Mode)
	switch mode {
	case ModeGRPC, ModeChainIngestor:
	default:
		return fmt.Errorf("unsupported athena-token mode %q", s.Mode)
	}

	if err := s.service.Start(); err != nil {
		return err
	}
	switch mode {
	case ModeGRPC:
	case ModeChainIngestor:
		if s.StoreSrc == nil {
			_ = s.service.Stop()
			return fmt.Errorf("token store source is required for chain-ingestor mode")
		}
		store, err := s.StoreSrc(ctx)
		if err != nil {
			_ = s.service.Stop()
			return err
		}
		s.store = store
		s.chainWorker = chainingestor.NewWorker(chainingestor.Options{
			Store:          store,
			EthNodeWSURL:   s.EthNodeWSURL,
			BSCNodeWSURL:   s.BSCNodeWSURL,
			NodeWSUseProxy: s.NodeWSUseProxy,
		})
		if err := s.chainWorker.Start(ctx); err != nil {
			_ = store.Close()
			_ = s.service.Stop()
			s.store = nil
			s.chainWorker = nil
			return err
		}
	}
	s.setHealthStatus(grpc_health_v1.HealthCheckResponse_SERVING)
	return nil
}

func (s *Server) Stop() error {
	s.setHealthStatus(grpc_health_v1.HealthCheckResponse_NOT_SERVING)
	var result error
	if s.chainWorker != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		if err := s.chainWorker.Stop(ctx); err != nil {
			result = err
		}
		cancel()
		s.chainWorker = nil
	}
	if err := s.service.Stop(); err != nil && result == nil {
		result = err
	}
	if s.store != nil {
		if err := s.store.Close(); err != nil && result == nil {
			result = err
		}
		s.store = nil
	}
	return result
}

func (s *Server) setHealthStatus(status grpc_health_v1.HealthCheckResponse_ServingStatus) {
	if s.healthService != nil {
		s.healthService.SetServingStatus("", status)
	}
}
