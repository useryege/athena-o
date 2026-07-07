package etherscangateway

import (
	"context"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const authorizationMetadataKey = "authorization"

func bearerAuthUnaryInterceptor(authToken string) grpc.UnaryServerInterceptor {
	expected := "Bearer " + strings.TrimSpace(authToken)
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		if info != nil && strings.HasPrefix(info.FullMethod, "/grpc.health.v1.Health/") {
			return handler(ctx, req)
		}
		if strings.TrimSpace(authToken) == "" {
			return nil, status.Error(codes.FailedPrecondition, "gateway auth token is not configured")
		}
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "authorization metadata is required")
		}
		for _, value := range md.Get(authorizationMetadataKey) {
			if strings.TrimSpace(value) == expected {
				return handler(ctx, req)
			}
		}
		return nil, status.Error(codes.Unauthenticated, "valid bearer authorization is required")
	}
}
