package token

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/useryege/athena/internal/server/version"
	"github.com/useryege/athena/internal/token/apiclient"
	"github.com/useryege/athena/internal/token/chainingestor"
	"github.com/useryege/athena/internal/token/projectdatacollector"
	"github.com/useryege/athena/internal/token/projectqualifier"
	"github.com/useryege/athena/internal/token/projectreportevaluator"
	tokenstore "github.com/useryege/athena/internal/token/store"
	versionpkg "github.com/useryege/athena/pkg/apiclient/version"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
)

const (
	ModeGRPC                   = "grpc"
	ModeChainIngestor          = "chain-ingestor"
	ModeProjectQualifier       = "project-qualifier"
	ModeProjectDataCollector   = "project-data-collector"
	ModeProjectReportEvaluator = "project-report-evaluator"
)

type Server struct {
	ServerOpts
	service         *Service
	healthService   *health.Server
	store           *tokenstore.SQLStore
	chainWorker     *chainingestor.Worker
	qualifier       *projectqualifier.Worker
	dataCollector   *projectdatacollector.Worker
	reportEvaluator *projectreportevaluator.Worker
}

type ServerOpts struct {
	Mode               string
	StoreSrc           func(context.Context) (*tokenstore.SQLStore, error)
	EthNodeWSURLs      []string
	BSCNodeWSURLs      []string
	EthAthenaContract  string
	BSCAthenaContract  string
	EthEnabled         bool
	BSCEnabled         bool
	NodeWSUseProxy     bool
	AveAPIKey          string
	AveAPIBaseURL      string
	EthereumAPIAddress string
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
	case ModeGRPC, ModeChainIngestor, ModeProjectQualifier, ModeProjectDataCollector, ModeProjectReportEvaluator:
	default:
		return fmt.Errorf("unsupported athena-token mode %q", s.Mode)
	}

	if err := s.service.Start(); err != nil {
		return err
	}
	switch mode {
	case ModeGRPC:
	case ModeChainIngestor:
		store, err := s.startStore(ctx, mode)
		if err != nil {
			_ = s.service.Stop()
			return err
		}
		s.chainWorker = chainingestor.NewWorker(chainingestor.Options{
			Store:          store,
			EthNodeWSURLs:  s.EthNodeWSURLs,
			BSCNodeWSURLs:  s.BSCNodeWSURLs,
			EthEnabled:     s.EthEnabled,
			BSCEnabled:     s.BSCEnabled,
			NodeWSUseProxy: s.NodeWSUseProxy,
		})
		if err := s.chainWorker.Start(ctx); err != nil {
			_ = store.Close()
			_ = s.service.Stop()
			s.store = nil
			s.chainWorker = nil
			return err
		}
	case ModeProjectQualifier:
		store, err := s.startStore(ctx, mode)
		if err != nil {
			_ = s.service.Stop()
			return err
		}
		s.qualifier = projectqualifier.NewWorker(projectqualifier.Options{
			Store:             store,
			EthNodeWSURLs:     s.EthNodeWSURLs,
			BSCNodeWSURLs:     s.BSCNodeWSURLs,
			EthAthenaContract: s.EthAthenaContract,
			BSCAthenaContract: s.BSCAthenaContract,
			EthEnabled:        s.EthEnabled,
			BSCEnabled:        s.BSCEnabled,
			NodeWSUseProxy:    s.NodeWSUseProxy,
		})
		if err := s.qualifier.Start(ctx); err != nil {
			_ = store.Close()
			_ = s.service.Stop()
			s.store = nil
			s.qualifier = nil
			return err
		}
	case ModeProjectDataCollector:
		store, err := s.startStore(ctx, mode)
		if err != nil {
			_ = s.service.Stop()
			return err
		}
		s.dataCollector = projectdatacollector.NewWorker(projectdatacollector.Options{
			Store:              store,
			EthNodeWSURLs:      s.EthNodeWSURLs,
			BSCNodeWSURLs:      s.BSCNodeWSURLs,
			EthAthenaContract:  s.EthAthenaContract,
			BSCAthenaContract:  s.BSCAthenaContract,
			EthEnabled:         s.EthEnabled,
			BSCEnabled:         s.BSCEnabled,
			NodeWSUseProxy:     s.NodeWSUseProxy,
			AveAPIKey:          s.AveAPIKey,
			AveAPIBaseURL:      s.AveAPIBaseURL,
			EthereumAPIAddress: s.EthereumAPIAddress,
		})
		if err := s.dataCollector.Start(ctx); err != nil {
			_ = store.Close()
			_ = s.service.Stop()
			s.store = nil
			s.dataCollector = nil
			return err
		}
	case ModeProjectReportEvaluator:
		store, err := s.startStore(ctx, mode)
		if err != nil {
			_ = s.service.Stop()
			return err
		}
		s.reportEvaluator = projectreportevaluator.NewWorker(projectreportevaluator.Options{Store: store})
		if err := s.reportEvaluator.Start(ctx); err != nil {
			_ = store.Close()
			_ = s.service.Stop()
			s.store = nil
			s.reportEvaluator = nil
			return err
		}
	}
	s.setHealthStatus(grpc_health_v1.HealthCheckResponse_SERVING)
	return nil
}

func (s *Server) startStore(ctx context.Context, mode string) (*tokenstore.SQLStore, error) {
	if s.store != nil {
		return s.store, nil
	}
	if s.StoreSrc == nil {
		return nil, fmt.Errorf("token store source is required for %s mode", mode)
	}
	store, err := s.StoreSrc(ctx)
	if err != nil {
		return nil, err
	}
	s.store = store
	return store, nil
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
	if s.qualifier != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		if err := s.qualifier.Stop(ctx); err != nil && result == nil {
			result = err
		}
		cancel()
		s.qualifier = nil
	}
	if s.dataCollector != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		if err := s.dataCollector.Stop(ctx); err != nil && result == nil {
			result = err
		}
		cancel()
		s.dataCollector = nil
	}
	if s.reportEvaluator != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		if err := s.reportEvaluator.Stop(ctx); err != nil && result == nil {
			result = err
		}
		cancel()
		s.reportEvaluator = nil
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
