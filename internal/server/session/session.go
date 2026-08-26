package session

import (
	"context"

	"github.com/useryege/athena/internal/accountaccess"
	"github.com/useryege/athena/internal/accountcenter"
	"github.com/useryege/athena/internal/accountcredentials"
	accountserver "github.com/useryege/athena/internal/server/account"
	"github.com/useryege/athena/pkg/apiclient/session"
	sessionmgr "github.com/useryege/athena/util/session"
)

// Server provides the optional-authentication session projection.
type Server struct {
	authenticator    Authenticator
	accessController *accountaccess.Controller
	accountCenter    *accountcenter.Manager
	credentials      *accountcredentials.CredentialManager
}

type Authenticator interface {
	Authenticate(ctx context.Context) (context.Context, error)
}

// NewServer returns a Session service that only exposes GetUserInfo.
func NewServer(authenticator Authenticator, accessController *accountaccess.Controller, accountCenter *accountcenter.Manager, credentials *accountcredentials.CredentialManager) *Server {
	return &Server{
		authenticator:    authenticator,
		accessController: accessController,
		accountCenter:    accountCenter,
		credentials:      credentials,
	}
}

// AuthFuncOverride populates optional authentication for GetUserInfo.
func (s *Server) AuthFuncOverride(ctx context.Context, _ string) (context.Context, error) {
	ctx, err := s.authenticator.Authenticate(ctx)
	if sessionmgr.IsAccountMaintenanceError(err) {
		return ctx, err
	}
	return ctx, nil
}

func (s *Server) GetUserInfo(ctx context.Context, _ *session.GetUserInfoRequest) (*session.GetUserInfoResponse, error) {
	return ProjectUserInfo(ctx, s.accessController, s.accountCenter, s.credentials)
}

// ProjectUserInfo builds the shared session and authorization projection for ctx.
func ProjectUserInfo(ctx context.Context, accessController *accountaccess.Controller, accountCenter *accountcenter.Manager, credentials *accountcredentials.CredentialManager) (*session.GetUserInfoResponse, error) {
	loggedIn := sessionmgr.LoggedIn(ctx)
	response := &session.GetUserInfoResponse{
		LoggedIn:  loggedIn,
		AccountId: sessionmgr.AccountID(ctx),
		Iss:       sessionmgr.Iss(ctx),
	}
	if !loggedIn {
		return response, nil
	}
	identity, err := credentials.Get(response.AccountId)
	if err != nil {
		return nil, err
	}
	response.Username = identity.Username
	response.Administrator = identity.Administrator
	access, err := accessController.Get(response.AccountId)
	if err != nil {
		return nil, err
	}
	response.Access = accountserver.ToAPIAccountAccess(access)
	profile, err := accountCenter.GetProfile(ctx, response.AccountId)
	if err != nil {
		return nil, err
	}
	preferences, err := accountCenter.GetPreferences(ctx, response.AccountId)
	if err != nil {
		return nil, err
	}
	response.Profile = accountserver.ToAPIAccountProfile(response.AccountId, profile)
	response.Preferences = accountserver.ToAPIAccountPreferences(preferences)
	response.Identity = accountserver.ToAPIAccountIdentity(identity)
	return response, nil
}
