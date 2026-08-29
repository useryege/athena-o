package wallet

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"strings"

	"github.com/useryege/athena/internal/server/version"
	"github.com/useryege/athena/internal/wallet/apiclient"
	walletstore "github.com/useryege/athena/internal/wallet/store"
	versionpkg "github.com/useryege/athena/pkg/apiclient/version"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type Server struct {
	ServerOpts
	service                          *Service
	healthService                    *health.Server
	internalAuthTokenHash            [sha256.Size]byte
	wormExecutionSignerAuthTokenHash [sha256.Size]byte
}

type ServerOpts struct {
	Store                        *walletstore.SQLStore
	EncryptionKey                []byte
	InternalAuthToken            string
	WormExecutionSignerAuthToken string
}

func NewServer(opts ServerOpts) (*Server, error) {
	internalAuthToken, err := apiclient.NormalizeInternalAuthToken(opts.InternalAuthToken)
	if err != nil {
		return nil, err
	}
	wormExecutionSignerAuthToken, err := apiclient.NormalizeWormExecutionSignerAuthToken(opts.WormExecutionSignerAuthToken)
	if err != nil {
		return nil, err
	}
	if subtle.ConstantTimeCompare([]byte(internalAuthToken), []byte(wormExecutionSignerAuthToken)) == 1 {
		return nil, status.Error(codes.FailedPrecondition, "wallet internal and Worm execution signer auth tokens must be different")
	}
	opts.InternalAuthToken = ""
	opts.WormExecutionSignerAuthToken = ""
	healthService := health.NewServer()
	healthService.SetServingStatus("", grpc_health_v1.HealthCheckResponse_NOT_SERVING)
	return &Server{
		ServerOpts:                       opts,
		service:                          NewService(opts.Store, opts.EncryptionKey),
		healthService:                    healthService,
		internalAuthTokenHash:            sha256.Sum256([]byte(internalAuthToken)),
		wormExecutionSignerAuthTokenHash: sha256.Sum256([]byte(wormExecutionSignerAuthToken)),
	}, nil
}

func (s *Server) CreateGRPC() *grpc.Server {
	server := grpc.NewServer(
		grpc.MaxRecvMsgSize(apiclient.MaxGRPCMessageSize),
		grpc.ChainUnaryInterceptor(s.authenticateUnaryRPC),
		grpc.ChainStreamInterceptor(s.authenticateStreamRPC),
	)
	versionService := version.NewServer(nil, func() (bool, error) {
		return true, nil
	})
	versionpkg.RegisterVersionServiceServer(server, versionService)
	apiclient.RegisterWalletServiceServer(server, s.service)
	apiclient.RegisterWormExecutionSignerServiceServer(server, s.service)
	grpc_health_v1.RegisterHealthServer(server, s.healthService)
	return server
}

func (s *Server) authenticateUnaryRPC(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	if !isWalletHealthMethod(info.FullMethod) {
		if err := s.authenticateInternalRPC(ctx, info.FullMethod); err != nil {
			return nil, err
		}
	}
	return handler(ctx, req)
}

func (s *Server) authenticateStreamRPC(srv any, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	if !isWalletHealthMethod(info.FullMethod) {
		if err := s.authenticateInternalRPC(stream.Context(), info.FullMethod); err != nil {
			return err
		}
	}
	return handler(srv, stream)
}

func (s *Server) authenticateInternalRPC(ctx context.Context, fullMethod string) error {
	if strings.HasPrefix(fullMethod, "/athena.internal.wallet.WormExecutionSignerService/") &&
		!isWormExecutionSignerMethod(fullMethod) {
		return status.Error(codes.PermissionDenied, "wallet signer RPC is not allowed for this capability")
	}
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
	expectedHash := s.internalAuthTokenHash
	if isWormExecutionSignerMethod(fullMethod) {
		expectedHash = s.wormExecutionSignerAuthTokenHash
	}
	matched := subtle.ConstantTimeCompare(providedHash[:], expectedHash[:])
	if validHeader&matched != 1 {
		return status.Error(codes.Unauthenticated, "wallet internal authentication failed")
	}
	return nil
}

func isWormExecutionSignerMethod(fullMethod string) bool {
	switch fullMethod {
	case "/athena.internal.wallet.WormExecutionSignerService/SignWormWebSignInMessage",
		"/athena.internal.wallet.WormExecutionSignerService/SignWormPositionRequestTransaction":
		return true
	default:
		return false
	}
}

func isWalletHealthMethod(fullMethod string) bool {
	return strings.HasPrefix(fullMethod, "/grpc.health.v1.Health/")
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
