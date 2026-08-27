package apiclient

import (
	"context"
	"fmt"
	"strings"
	"unicode"
)

const (
	InternalAuthTokenEnv      = "ATHENA_WALLET_INTERNAL_AUTH_TOKEN"
	minimumInternalTokenBytes = 32
)

// NormalizeInternalAuthToken validates the shared service credential used by
// the API Server to call the trusted Wallet gRPC boundary.
func NormalizeInternalAuthToken(value string) (string, error) {
	token := strings.TrimSpace(value)
	if len(token) < minimumInternalTokenBytes {
		return "", fmt.Errorf("wallet internal auth token must contain at least %d bytes", minimumInternalTokenBytes)
	}
	if strings.IndexFunc(token, func(r rune) bool {
		return unicode.IsSpace(r) || unicode.IsControl(r)
	}) >= 0 {
		return "", fmt.Errorf("wallet internal auth token must not contain whitespace or control characters")
	}
	return token, nil
}

type internalBearerCredentials struct {
	authorization string
}

func (c internalBearerCredentials) GetRequestMetadata(context.Context, ...string) (map[string]string, error) {
	return map[string]string{"authorization": c.authorization}, nil
}

// Wallet's internal channel is currently plaintext inside the local process
// graph or private Compose network. The bearer token authenticates the caller;
// transport encryption remains a deployment-network responsibility.
func (internalBearerCredentials) RequireTransportSecurity() bool { return false }
