package notification

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"strings"

	"github.com/useryege/athena/internal/notification/apiclient"
	notificationstore "github.com/useryege/athena/internal/notification/store"
	"github.com/useryege/athena/internal/server/version"
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
	service               *Service
	healthService         *health.Server
	internalAuthTokenHash [sha256.Size]byte
}

type ServerOpts struct {
	Store             *notificationstore.SQLStore
	Sender            Sender
	ProfileSyncer     ProfileSyncer
	Poller            *TelegramPoller
	WorkerConfig      WorkerConfig
	InternalAuthToken string
}

func NewServer(opts ServerOpts) (*Server, error) {
	token, err := apiclient.NormalizeInternalAuthToken(opts.InternalAuthToken)
	if err != nil {
		return nil, err
	}
	opts.InternalAuthToken = ""
	healthService := health.NewServer()
	healthService.SetServingStatus("", grpc_health_v1.HealthCheckResponse_NOT_SERVING)
	return &Server{
		ServerOpts: opts,
		service: NewServiceWithWorkerConfig(
			opts.Store, opts.Sender, opts.ProfileSyncer, opts.Poller, opts.WorkerConfig,
		),
		healthService:         healthService,
		internalAuthTokenHash: sha256.Sum256([]byte(token)),
	}, nil
}

func (s *Server) CreateGRPC() *grpc.Server {
	server := grpc.NewServer(
		grpc.MaxRecvMsgSize(apiclient.MaxGRPCMessageSize),
		grpc.ChainUnaryInterceptor(s.authenticateUnaryRPC),
		grpc.ChainStreamInterceptor(s.authenticateStreamRPC),
	)
	versionService := version.NewServer(nil, func() (bool, error) { return true, nil })
	versionpkg.RegisterVersionServiceServer(server, versionService)
	apiclient.RegisterSystemNotificationServiceServer(server, s.service)
	apiclient.RegisterAccountNotificationServiceServer(server, s.service)
	apiclient.RegisterNotificationRuntimeServiceServer(server, s.service)
	grpc_health_v1.RegisterHealthServer(server, s.healthService)
	return server
}

func (s *Server) authenticateUnaryRPC(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	if !isNotificationHealthMethod(info.FullMethod) {
		if err := s.authenticateInternalRPC(ctx); err != nil {
			return nil, err
		}
	}
	return handler(ctx, req)
}

func (s *Server) authenticateStreamRPC(srv any, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	if !isNotificationHealthMethod(info.FullMethod) {
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
		return status.Error(codes.Unauthenticated, "notification internal authentication failed")
	}
	return nil
}

func isNotificationHealthMethod(fullMethod string) bool {
	return strings.HasPrefix(fullMethod, "/grpc.health.v1.Health/")
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

func (s *Server) setHealthStatus(statusValue grpc_health_v1.HealthCheckResponse_ServingStatus) {
	if s.healthService != nil {
		s.healthService.SetServingStatus("", statusValue)
	}
}
