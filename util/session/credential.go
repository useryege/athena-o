package session

import (
	"context"

	"github.com/useryege/athena/internal/accountcredentials"
)

type authenticatedCredentialContextKey struct{}

// WithAuthenticatedCredential attaches the validated server-side credential
// projection to a request context. Public JWT claims remain the compatibility
// boundary for existing handlers; sensitive handlers must use this typed value.
func WithAuthenticatedCredential(ctx context.Context, credential accountcredentials.AuthenticatedCredential) context.Context {
	return context.WithValue(ctx, authenticatedCredentialContextKey{}, credential)
}

// AuthenticatedCredentialFromContext returns the credential that passed the
// current request's complete signature, account-access, membership, and
// revocation checks.
func AuthenticatedCredentialFromContext(ctx context.Context) (accountcredentials.AuthenticatedCredential, bool) {
	credential, ok := ctx.Value(authenticatedCredentialContextKey{}).(accountcredentials.AuthenticatedCredential)
	return credential, ok && credential.AccountID != "" && credential.JTI != ""
}
