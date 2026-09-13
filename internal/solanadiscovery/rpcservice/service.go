package rpcservice

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/useryege/athena/internal/accountaccess"
	"github.com/useryege/athena/internal/solanadiscovery"
	api "github.com/useryege/athena/pkg/apiclient/solana"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const AccountIDMetadataKey = "x-athena-account-id"
const InternalAuthTokenEnv = "ATHENA_SOLANA_DISCOVERY_INTERNAL_AUTH_TOKEN"

// ProjectReader is the independent Solana service's own durable read model.
type ProjectReader interface {
	ListProjects(context.Context, uint32, uint32, string) ([]solanadiscovery.Project, int64, error)
	GetDiscoveryStatus(context.Context) (solanadiscovery.DiscoveryStatus, error)
}

// AccountAccessReader reads the authoritative account matrix on every call.
type AccountAccessReader interface {
	GetAccountAccess(context.Context, string) (accountaccess.Access, error)
}

type Server struct {
	api.UnimplementedSolanaServiceServer
	projects ProjectReader
	accounts AccountAccessReader
	token    string
}

func NewServer(projects ProjectReader, accounts AccountAccessReader, internalToken string) (*Server, error) {
	if projects == nil || accounts == nil {
		return nil, fmt.Errorf("Solana project and account readers are required")
	}
	token, err := NormalizeInternalAuthToken(internalToken)
	if err != nil {
		return nil, err
	}
	return &Server{projects: projects, accounts: accounts, token: token}, nil
}

func NormalizeInternalAuthToken(value string) (string, error) {
	token := strings.TrimSpace(value)
	if len(token) < 32 || strings.IndexFunc(token, func(r rune) bool { return unicode.IsSpace(r) || unicode.IsControl(r) }) >= 0 {
		return "", fmt.Errorf("Solana internal auth token must contain at least 32 bytes and no whitespace")
	}
	return token, nil
}

func (s *Server) authorize(ctx context.Context) error {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return status.Error(codes.Unauthenticated, "Solana internal authentication is required")
	}
	bearer := md.Get("authorization")
	if len(bearer) != 1 || !strings.HasPrefix(bearer[0], "Bearer ") || subtle.ConstantTimeCompare([]byte(strings.TrimPrefix(bearer[0], "Bearer ")), []byte(s.token)) != 1 {
		return status.Error(codes.Unauthenticated, "Solana internal authentication is required")
	}
	ids := md.Get(AccountIDMetadataKey)
	if len(ids) != 1 {
		return status.Error(codes.Unauthenticated, "exactly one authenticated account is required")
	}
	parsed, err := uuid.Parse(ids[0])
	if err != nil || parsed == uuid.Nil || parsed.String() != ids[0] {
		return status.Error(codes.Unauthenticated, "authenticated account ID is invalid")
	}
	access, err := s.accounts.GetAccountAccess(ctx, ids[0])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) || status.Code(err) == codes.NotFound {
			return status.Error(codes.PermissionDenied, "Solana account access is required")
		}
		return status.Error(codes.Unavailable, "Solana account access is unavailable")
	}
	if err := access.Validate(); err != nil {
		return status.Error(codes.Internal, "Solana account access is invalid")
	}
	if !access.LoginEnabled || access.Administrator || access.Modules[accountaccess.ModuleSolana] != accountaccess.AccessLevelRead {
		return status.Error(codes.PermissionDenied, "Solana READ access is required")
	}
	return nil
}

func (s *Server) ListProjects(ctx context.Context, req *api.ListProjectsRequest) (*api.ListProjectsResponse, error) {
	if err := s.authorize(ctx); err != nil {
		return nil, err
	}
	if req == nil {
		req = &api.ListProjectsRequest{}
	}
	page, pageSize := req.GetPage(), req.GetPageSize()
	if page == 0 {
		page = 1
	}
	if pageSize == 0 {
		pageSize = 25
	}
	if pageSize > 100 || utf8.RuneCountInString(req.GetQuery()) > 128 || !utf8.ValidString(req.GetQuery()) {
		return nil, status.Error(codes.InvalidArgument, "Solana list parameters are invalid")
	}
	rows, total, err := s.projects.ListProjects(ctx, page, pageSize, req.GetQuery())
	if err != nil {
		return nil, status.Errorf(codes.Unavailable, "Solana projects are unavailable: %v", err)
	}
	items := make([]*api.Project, 0, len(rows))
	for _, row := range rows {
		items = append(items, &api.Project{Mint: row.Mint, TokenProgram: row.TokenProgram, Signature: row.Signature, FeePayer: row.FeePayer, MintAuthority: row.MintAuthority, FreezeAuthority: row.FreezeAuthority, Decimals: row.Decimals, Slot: row.Slot, BlockTime: row.BlockTime, DiscoveredAt: row.DiscoveredAt.Unix()})
	}
	return &api.ListProjectsResponse{Items: items, TotalSize: total}, nil
}

func (s *Server) GetDiscoveryStatus(ctx context.Context, _ *api.GetDiscoveryStatusRequest) (*api.GetDiscoveryStatusResponse, error) {
	if err := s.authorize(ctx); err != nil {
		return nil, err
	}
	value, err := s.projects.GetDiscoveryStatus(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unavailable, "Solana discovery status is unavailable: %v", err)
	}
	lastSuccess := int64(0)
	if !value.LastSuccessAt.IsZero() {
		lastSuccess = value.LastSuccessAt.Unix()
	}
	return &api.GetDiscoveryStatusResponse{Status: value.Status, StartSlot: value.StartSlot, LastProcessedSlot: value.LastProcessedSlot, LatestFinalizedSlot: value.LatestFinalizedSlot, LastSuccessAt: lastSuccess, TotalProjects: value.TotalProjects, LastError: value.LastError}, nil
}
