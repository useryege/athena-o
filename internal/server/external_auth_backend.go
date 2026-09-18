package server

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/useryege/athena/internal/accountcredentials"
	"github.com/useryege/athena/internal/authregistration"
	util_session "github.com/useryege/athena/util/session"
)

// externalAuthBackend adapts the durable credential/session managers to the
// small provider-neutral browser authentication boundary.
type externalAuthBackend struct {
	credentials     *accountcredentials.CredentialManager
	sessions        *util_session.SessionManager
	sessionDuration time.Duration
}

func newExternalAuthBackend(credentials *accountcredentials.CredentialManager, sessions *util_session.SessionManager, sessionDuration time.Duration) (*externalAuthBackend, error) {
	if credentials == nil || sessions == nil {
		return nil, fmt.Errorf("external authentication credential and session managers are required")
	}
	if int64(sessionDuration.Seconds()) <= 0 {
		return nil, fmt.Errorf("external authentication session duration must be positive")
	}
	return &externalAuthBackend{credentials: credentials, sessions: sessions, sessionDuration: sessionDuration}, nil
}

func (backend *externalAuthBackend) GetByIdentity(ctx context.Context, identity authregistration.Identity) (authregistration.Account, bool, error) {
	if err := identity.Validate(); err != nil {
		return authregistration.Account{}, false, fmt.Errorf("%w: %v", authregistration.ErrIdentityNotAllowed, err)
	}
	account, found, err := backend.credentials.GetByIdentity(ctx, identity.Provider, identity.Subject, identity.Realm)
	if err != nil || !found {
		return authregistration.Account{}, found, err
	}
	return authregistration.Account{ID: account.ID}, true, nil
}

func (backend *externalAuthBackend) UsernameAvailable(ctx context.Context, username string, realm accountcredentials.ApplicationRealm) (bool, error) {
	administrator, err := realm.Administrator()
	if err != nil {
		return false, fmt.Errorf("%w: %v", authregistration.ErrIdentityNotAllowed, err)
	}
	return backend.credentials.UsernameAvailable(ctx, username, administrator)
}

func (backend *externalAuthBackend) RegisterExternalAccount(ctx context.Context, identity authregistration.Identity, username string) (authregistration.Account, error) {
	account, created, err := backend.credentials.RegisterExternalAccount(
		ctx,
		identity.Provider,
		identity.Subject,
		identity.VerifiedEmail,
		username,
		identity.Realm,
	)
	if err != nil {
		switch {
		case errors.Is(err, accountcredentials.ErrUsernameInvalid):
			return authregistration.Account{}, fmt.Errorf("%w: %v", authregistration.ErrUsernameInvalid, err)
		case errors.Is(err, accountcredentials.ErrUsernameUnavailable):
			return authregistration.Account{}, fmt.Errorf("%w: %v", authregistration.ErrUsernameUnavailable, err)
		case errors.Is(err, accountcredentials.ErrAdministratorIdentityConflict), status.Code(err) == codes.PermissionDenied:
			return authregistration.Account{}, fmt.Errorf("%w: %v", authregistration.ErrIdentityNotAllowed, err)
		default:
			return authregistration.Account{}, err
		}
	}
	return authregistration.Account{ID: account.ID, Created: created}, nil
}

func (backend *externalAuthBackend) RegisterCommittedAccess(ctx context.Context, accountID string) error {
	return backend.sessions.RegisterCommittedAccountAccess(ctx, accountID)
}

func (backend *externalAuthBackend) CreateExternalLogin(ctx context.Context, accountID string, identity authregistration.Identity) (string, error) {
	jti, err := uuid.NewRandom()
	if err != nil {
		return "", fmt.Errorf("generate login JTI: %w", err)
	}
	token, err := backend.sessions.CreateExternalLogin(
		ctx,
		accountID,
		identity.Realm,
		identity.Provider,
		identity.Subject,
		identity.VerifiedEmail,
		int64(backend.sessionDuration.Seconds()),
		jti.String(),
	)
	if err != nil {
		switch {
		case util_session.IsAccountMaintenanceError(err):
			return "", fmt.Errorf("%w: %v", authregistration.ErrMaintenance, err)
		case status.Code(err) == codes.PermissionDenied:
			return "", fmt.Errorf("%w: %v", authregistration.ErrIdentityNotAllowed, err)
		default:
			return "", err
		}
	}
	return token, nil
}

func (backend *externalAuthBackend) RecordLoginResult(result string) {
	backend.sessions.IncLoginRequestCounter(result)
}
