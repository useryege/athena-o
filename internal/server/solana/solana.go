package solana

import (
	"context"
	"time"

	"github.com/google/uuid"
	api "github.com/useryege/athena/pkg/apiclient/solana"
	utilsession "github.com/useryege/athena/util/session"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const accountIDMetadataKey = "x-athena-account-id"

type Server struct {
	api.UnimplementedSolanaServiceServer
	client api.SolanaServiceClient
}

func NewServer(client api.SolanaServiceClient) *Server { return &Server{client: client} }

func (s *Server) downstreamContext(ctx context.Context) (context.Context, context.CancelFunc, error) {
	credential, ok := utilsession.AuthenticatedCredentialFromContext(ctx)
	if !ok {
		return nil, nil, status.Error(codes.Unauthenticated, "authenticated account is required")
	}
	parsed, err := uuid.Parse(credential.AccountID)
	if err != nil || parsed == uuid.Nil || parsed.String() != credential.AccountID {
		return nil, nil, status.Error(codes.Unauthenticated, "authenticated account ID is invalid")
	}
	bounded, cancel := context.WithTimeout(ctx, 10*time.Second)
	return metadata.NewOutgoingContext(bounded, metadata.Pairs(accountIDMetadataKey, credential.AccountID)), cancel, nil
}

func (s *Server) ListProjects(ctx context.Context, req *api.ListProjectsRequest) (*api.ListProjectsResponse, error) {
	outgoing, cancel, err := s.downstreamContext(ctx)
	if err != nil {
		return nil, err
	}
	defer cancel()
	if s.client == nil {
		return nil, status.Error(codes.Unavailable, "Solana discovery service is unavailable")
	}
	response, err := s.client.ListProjects(outgoing, req)
	if status.Code(err) == codes.Unavailable {
		return nil, status.Error(codes.Unavailable, "Solana discovery service is unavailable")
	}
	return response, err
}

func (s *Server) GetDiscoveryStatus(ctx context.Context, req *api.GetDiscoveryStatusRequest) (*api.GetDiscoveryStatusResponse, error) {
	outgoing, cancel, err := s.downstreamContext(ctx)
	if err != nil {
		return nil, err
	}
	defer cancel()
	if s.client == nil {
		return nil, status.Error(codes.Unavailable, "Solana discovery service is unavailable")
	}
	response, err := s.client.GetDiscoveryStatus(outgoing, req)
	if status.Code(err) == codes.Unavailable {
		return nil, status.Error(codes.Unavailable, "Solana discovery service is unavailable")
	}
	return response, err
}
