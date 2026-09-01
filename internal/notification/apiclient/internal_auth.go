package apiclient

import (
	"context"
	"fmt"
	"strings"
	"unicode"

	"google.golang.org/grpc/credentials"
)

const (
	InternalAuthTokenEnv      = "ATHENA_NOTIFICATION_INTERNAL_AUTH_TOKEN"
	minimumInternalTokenBytes = 32
	grpcHealthMethodPrefix    = "/grpc.health.v1.Health/"
	grpcHealthServiceSuffix   = "/grpc.health.v1.Health"
)

func NormalizeInternalAuthToken(value string) (string, error) {
	if strings.IndexFunc(value, func(r rune) bool {
		return unicode.IsSpace(r) || unicode.IsControl(r)
	}) >= 0 {
		return "", fmt.Errorf("notification internal auth token must not contain whitespace or control characters")
	}
	if len(value) < minimumInternalTokenBytes {
		return "", fmt.Errorf("notification internal auth token must contain at least %d bytes", minimumInternalTokenBytes)
	}
	return value, nil
}

type internalBearerCredentials struct {
	authorization string
}

func (c internalBearerCredentials) GetRequestMetadata(ctx context.Context, uri ...string) (map[string]string, error) {
	if requestInfo, ok := credentials.RequestInfoFromContext(ctx); ok && strings.HasPrefix(requestInfo.Method, grpcHealthMethodPrefix) {
		return nil, nil
	}
	for _, value := range uri {
		if strings.HasSuffix(strings.TrimRight(value, "/"), grpcHealthServiceSuffix) {
			return nil, nil
		}
	}
	return map[string]string{"authorization": c.authorization}, nil
}

func (internalBearerCredentials) RequireTransportSecurity() bool { return false }
