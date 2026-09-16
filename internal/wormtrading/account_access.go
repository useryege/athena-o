package wormtrading

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/useryege/athena/internal/accountaccess"
	"github.com/useryege/athena/internal/wormtrading/apiclient"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const AccountIDMetadataKey = "x-athena-account-id"

// AccountAccessReader reads current durable permissions without borrowing API runtime state.
type AccountAccessReader interface {
	GetAccountAccess(context.Context, string) (accountaccess.Access, error)
}

func (s *Service) authorizeCatalogAccount(ctx context.Context, owner string, required accountaccess.AccessLevel) (string, error) {
	md, _ := metadata.FromIncomingContext(ctx)
	ids := md.Get(AccountIDMetadataKey)
	if len(ids) != 1 {
		return "", status.Error(codes.Unauthenticated, "exactly one account identity is required")
	}
	id, err := uuid.Parse(ids[0])
	if err != nil || id == uuid.Nil || id.String() != ids[0] {
		return "", status.Error(codes.Unauthenticated, "canonical account identity is required")
	}
	if owner != "" && owner != ids[0] {
		return "", status.Error(codes.PermissionDenied, "account owner does not match identity")
	}
	if required != accountaccess.AccessLevelRead && required != accountaccess.AccessLevelReadWrite {
		return "", status.Error(codes.Internal, "invalid required account access")
	}
	if err := ctx.Err(); err != nil {
		return "", status.FromContextError(err).Err()
	}
	if s.accountAccessReader == nil {
		return "", status.Error(codes.Unavailable, "account access reader is unavailable")
	}
	access, err := s.accountAccessReader.GetAccountAccess(ctx, ids[0])
	if err != nil {
		switch {
		case errors.Is(err, context.Canceled):
			return "", status.Error(codes.Canceled, "account access read canceled")
		case errors.Is(err, context.DeadlineExceeded):
			return "", status.Error(codes.DeadlineExceeded, "account access read timed out")
		case errors.Is(err, pgx.ErrNoRows), status.Code(err) == codes.NotFound:
			return "", status.Error(codes.PermissionDenied, "account access denied")
		case status.Code(err) == codes.Canceled, status.Code(err) == codes.DeadlineExceeded:
			return "", err
		// The SQL reader identifies corruption separately from query failures.
		case errors.Is(err, accountaccess.ErrInvalidPersistedAccess), status.Code(err) == codes.InvalidArgument:
			return "", status.Error(codes.Internal, "invalid persisted account access")
		default:
			return "", status.Error(codes.Unavailable, "account access store is unavailable")
		}
	}
	if err := ctx.Err(); err != nil {
		return "", status.FromContextError(err).Err()
	}
	if err := access.Validate(); err != nil {
		return "", status.Error(codes.Internal, "invalid persisted account access")
	}
	level := access.Modules[accountaccess.ModuleWormTrading]
	if !access.LoginEnabled || access.Administrator || (level != accountaccess.AccessLevelReadWrite && (required != accountaccess.AccessLevelRead || level != accountaccess.AccessLevelRead)) {
		return "", status.Error(codes.PermissionDenied, "Worm Trading account access denied")
	}
	return ids[0], nil
}

func (s *Service) GetOrderEventCatalog(ctx context.Context, req *apiclient.GetOrderEventCatalogRequest) (*apiclient.GetOrderEventCatalogResponse, error) {
	// Bound authorization, catalog work and any downstream wait from RPC entry.
	ctx, cancel := context.WithTimeout(ctx, s.wormCatalogBudget)
	defer cancel()
	if _, err := s.authorizeCatalogAccount(ctx, "", accountaccess.AccessLevelRead); err != nil {
		return nil, err
	}
	return s.catalogReader.GetOrderEventCatalog(ctx, req)
}
