package transport

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"strings"

	"github.com/google/uuid"
	trpc "github.com/useryege/athena/internal/tradersync/apiclient"
	"github.com/useryege/athena/internal/tradersync/rpcconfig"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func ValidateActor(actor *trpc.Actor) error {
	invalid := func() error {
		return contractError(codes.InvalidArgument, "ACTOR_INVALID", "valid Trader Sync actor required")
	}
	if actor == nil {
		return invalid()
	}
	id, err := uuid.Parse(actor.AccountId)
	if err != nil || id == uuid.Nil || id.String() != actor.AccountId {
		return invalid()
	}
	switch actor.Realm {
	case trpc.ApplicationRealm_APPLICATION_REALM_MEMBER, trpc.ApplicationRealm_APPLICATION_REALM_ADMIN:
		return nil
	default:
		return invalid()
	}
}

func NewUnaryInterceptor(token string, ready func() bool) grpc.UnaryServerInterceptor {
	expected := sha256.Sum256([]byte(token))
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if info.FullMethod == "/grpc.health.v1.Health/Check" || info.FullMethod == "/grpc.health.v1.Health/List" {
			return handler(ctx, req)
		}
		budget, err := rpcconfig.MethodBudget(info.FullMethod)
		if err != nil {
			return nil, status.Error(codes.Unimplemented, "unknown Trader Sync method")
		}
		values := metadata.ValueFromIncomingContext(ctx, "authorization")
		if len(values) == 0 || (len(values) == 1 && values[0] == "") {
			return nil, contractError(codes.Unauthenticated, "SERVICE_AUTH_MISSING", "internal service authentication required")
		}
		if len(values) != 1 || !strings.HasPrefix(values[0], "Bearer ") {
			return nil, contractError(codes.Unauthenticated, "SERVICE_AUTH_INVALID", "invalid internal service authentication")
		}
		actual := sha256.Sum256([]byte(strings.TrimPrefix(values[0], "Bearer ")))
		if token == "" || subtle.ConstantTimeCompare(actual[:], expected[:]) != 1 {
			return nil, contractError(codes.Unauthenticated, "SERVICE_AUTH_INVALID", "invalid internal service authentication")
		}
		request, ok := req.(interface{ GetActor() *trpc.Actor })
		if !ok {
			return nil, contractError(codes.InvalidArgument, "ACTOR_INVALID", "valid Trader Sync actor required")
		}
		actor := request.GetActor()
		if err := ValidateActor(actor); err != nil {
			return nil, err
		}
		realm := trpc.ApplicationRealm_APPLICATION_REALM_MEMBER
		switch info.FullMethod {
		case "/tradersync.internal.v1.TraderSyncService/GetSubscriptionSummary", "/tradersync.internal.v1.TraderSyncService/ListSubscriptionSummaries", "/tradersync.internal.v1.TraderSyncService/GetTraderSyncRuntimeStatus":
			realm = trpc.ApplicationRealm_APPLICATION_REALM_ADMIN
		}
		if actor.Realm != realm {
			return nil, status.Error(codes.PermissionDenied, "actor realm does not permit this operation")
		}
		if ready == nil || !ready() {
			return nil, status.Error(codes.Unavailable, "Trader Sync is not ready")
		}
		bounded, cancel := context.WithTimeout(ctx, budget)
		defer cancel()
		return handler(bounded, req)
	}
}
