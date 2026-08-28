package apiclient

import (
	"context"
	"fmt"
	"strings"
	"unicode"
)

const (
	InternalAuthTokenEnv      = "ATHENA_WORM_TRADING_INTERNAL_AUTH_TOKEN"
	minimumInternalTokenBytes = 32
)

// NormalizeInternalAuthToken validates the service credential used to call the
// trusted Worm Trading boundary.
func NormalizeInternalAuthToken(value string) (string, error) {
	token := strings.TrimSpace(value)
	if len(token) < minimumInternalTokenBytes {
		return "", fmt.Errorf("Worm Trading internal auth token must contain at least %d bytes", minimumInternalTokenBytes)
	}
	if strings.IndexFunc(token, func(r rune) bool {
		return unicode.IsSpace(r) || unicode.IsControl(r)
	}) >= 0 {
		return "", fmt.Errorf("Worm Trading internal auth token must not contain whitespace or control characters")
	}
	return token, nil
}

type internalBearerCredentials struct {
	authorization string
}

func (c internalBearerCredentials) GetRequestMetadata(context.Context, ...string) (map[string]string, error) {
	return map[string]string{"authorization": c.authorization}, nil
}

// The internal channel is plaintext inside the local process graph or private
// deployment network. The independent Bearer authenticates its caller.
func (internalBearerCredentials) RequireTransportSecurity() bool { return false }
