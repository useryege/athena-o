package authregistration

import (
	"context"
	"errors"
	"fmt"

	"github.com/useryege/athena/internal/accountcredentials"
)

const (
	// DefaultReturnTo is the safe fallback after authentication and registration.
	DefaultReturnTo = "/account/access"
	// AdministratorDefaultReturnTo is the safe landing for the administrator realm.
	AdministratorDefaultReturnTo = "/admin/accounts"

	LoginSuccess = "success"
	LoginFailure = "failure"
)

var (
	// ErrUsernameInvalid and ErrUsernameUnavailable are stable classifications
	// returned by Backend implementations to the anonymous registration handler.
	ErrUsernameInvalid     = errors.New("username is invalid")
	ErrUsernameUnavailable = errors.New("username is unavailable")
	// ErrIdentityNotAllowed prevents a verified identity from claiming the
	// account aggregate requested by its registration ticket.
	ErrIdentityNotAllowed = errors.New("external identity is not allowed")
	// ErrMaintenance reports that the resolved account currently has login
	// disabled. It is intentionally provider-neutral.
	ErrMaintenance = errors.New("account login is disabled")
)

// Identity is a server-verified external identity. Subject is never projected
// by the registration HTTP API; a Solana subject is separately exposed as its
// public wallet address where presentation requires it.
type Identity struct {
	Provider      accountcredentials.IdentityProvider `json:"provider"`
	Subject       string                              `json:"subject"`
	VerifiedEmail string                              `json:"verifiedEmail"`
	Realm         accountcredentials.ApplicationRealm `json:"realm"`
}

// Validate checks the identity invariant before it enters a Redis ticket or a
// durable account transaction.
func (identity Identity) Validate() error {
	subject, email, err := accountcredentials.NormalizeExternalIdentity(
		identity.Provider,
		identity.Subject,
		identity.VerifiedEmail,
		identity.Realm,
	)
	if err != nil {
		return fmt.Errorf("invalid external identity: %w", err)
	}
	if subject != identity.Subject || email != identity.VerifiedEmail {
		return fmt.Errorf("external identity must use canonical subject and email values")
	}
	return nil
}

// Account is the minimum durable projection needed after realm-aware identity
// resolution. Username never participates in login resolution.
type Account struct {
	ID string
}

// Backend is the provider-neutral durable identity and Athena session boundary.
// Implementations must publish a committed credential before returning from
// RegisterExternalAccount and publish access before registration is consumed.
type Backend interface {
	GetByIdentity(ctx context.Context, identity Identity) (Account, bool, error)
	UsernameAvailable(ctx context.Context, username string, realm accountcredentials.ApplicationRealm) (bool, error)
	RegisterExternalAccount(ctx context.Context, identity Identity, username string) (Account, error)
	RegisterCommittedAccess(ctx context.Context, accountID string) error
	CreateExternalLogin(ctx context.Context, accountID string, identity Identity) (string, error)
	RecordLoginResult(status string)
}
