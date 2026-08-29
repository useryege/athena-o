package wormtrading

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"strings"
	"time"

	"github.com/useryege/athena/internal/server/version"
	walletapiclient "github.com/useryege/athena/internal/wallet/apiclient"
	wormmarketsapiclient "github.com/useryege/athena/internal/wormmarkets/apiclient"
	"github.com/useryege/athena/internal/wormtrading/apiclient"
	wormstore "github.com/useryege/athena/internal/wormtrading/store"
	versionpkg "github.com/useryege/athena/pkg/apiclient/version"
	utilworm "github.com/useryege/athena/util/worm"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type Server struct {
	ServerOpts
	service               *Service
	healthService         *health.Server
	internalAuthTokenHash [sha256.Size]byte
}

type ServerOpts struct {
	BalanceAdapter          *SolanaBalanceAdapter
	CredentialStore         wormstore.Store
	CredentialEncryptionKey []byte
	WormAPIAttemptTimeout   time.Duration
	WormPositionBudget      time.Duration
	WormPositionConcurrency int
	WormMarketsClientset    wormmarketsapiclient.Clientset
	WormWebClient           utilworm.WebClient
	WalletSignerClientset   walletapiclient.WormExecutionSignerClientset
	InternalAuthToken       string
}

func NewServer(opts ServerOpts) (*Server, error) {
	internalAuthToken, err := apiclient.NormalizeInternalAuthToken(opts.InternalAuthToken)
	if err != nil {
		return nil, err
	}
	opts.InternalAuthToken = ""
	healthService := health.NewServer()
	healthService.SetServingStatus("", grpc_health_v1.HealthCheckResponse_NOT_SERVING)
	server := &Server{
		ServerOpts:            opts,
		healthService:         healthService,
		internalAuthTokenHash: sha256.Sum256([]byte(internalAuthToken)),
	}
	server.service, err = NewServiceWithOptions(ServiceOptions{
		BalanceAdapter:          opts.BalanceAdapter,
		CredentialStore:         opts.CredentialStore,
		CredentialEncryptionKey: opts.CredentialEncryptionKey,
		WormAPIAttemptTimeout:   opts.WormAPIAttemptTimeout,
		WormPositionBudget:      opts.WormPositionBudget,
		WormPositionConcurrency: opts.WormPositionConcurrency,
		WormMarketsClientset:    opts.WormMarketsClientset,
		WormWebClient:           opts.WormWebClient,
		WalletSignerClientset:   opts.WalletSignerClientset,
		SetHealthStatus:         server.setHealthStatus,
	})
	if err != nil {
		return nil, err
	}
	return server, nil
}

func (s *Server) CreateGRPC() *grpc.Server {
	server := grpc.NewServer(
		grpc.MaxRecvMsgSize(apiclient.MaxGRPCMessageSize),
		grpc.ChainUnaryInterceptor(s.authenticateUnaryRPC),
		grpc.ChainStreamInterceptor(s.authenticateStreamRPC),
	)
	versionpkg.RegisterVersionServiceServer(server, version.NewServer(nil, func() (bool, error) { return true, nil }))
	apiclient.RegisterWormTradingServiceServer(server, s.service)
	grpc_health_v1.RegisterHealthServer(server, s.healthService)
	return server
}

func (s *Server) Start() error {
	return s.service.Start()
}

func (s *Server) Stop() error {
	s.setHealthStatus(grpc_health_v1.HealthCheckResponse_NOT_SERVING)
	return s.service.Stop()
}

func (s *Server) authenticateUnaryRPC(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	if !isWormTradingHealthMethod(info.FullMethod) {
		if err := s.authenticateInternalRPC(ctx); err != nil {
			return nil, err
		}
	}
	return handler(ctx, req)
}

func (s *Server) authenticateStreamRPC(srv any, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	if !isWormTradingHealthMethod(info.FullMethod) {
		if err := s.authenticateInternalRPC(stream.Context()); err != nil {
			return err
		}
	}
	return handler(srv, stream)
}

func (s *Server) authenticateInternalRPC(ctx context.Context) error {
	providedToken := ""
	validHeader := 0
	if incoming, ok := metadata.FromIncomingContext(ctx); ok {
		values := incoming.Get("authorization")
		if len(values) == 1 && strings.HasPrefix(values[0], "Bearer ") {
			providedToken = strings.TrimPrefix(values[0], "Bearer ")
			if providedToken != "" {
				validHeader = 1
			}
		}
	}
	providedHash := sha256.Sum256([]byte(providedToken))
	matched := subtle.ConstantTimeCompare(providedHash[:], s.internalAuthTokenHash[:])
	if validHeader&matched != 1 {
		return status.Error(codes.Unauthenticated, "Worm Trading internal authentication failed")
	}
	return nil
}

func isWormTradingHealthMethod(fullMethod string) bool {
	return strings.HasPrefix(fullMethod, "/grpc.health.v1.Health/")
}

func (s *Server) setHealthStatus(servingStatus grpc_health_v1.HealthCheckResponse_ServingStatus) {
	if s.healthService != nil {
		s.healthService.SetServingStatus("", servingStatus)
	}
}
