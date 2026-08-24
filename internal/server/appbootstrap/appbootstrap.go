package appbootstrap

import (
	"context"

	"github.com/useryege/athena/internal/accountaccess"
	"github.com/useryege/athena/internal/accountcenter"
	sessionserver "github.com/useryege/athena/internal/server/session"
	settingsserver "github.com/useryege/athena/internal/server/settings"
	appbootstrappkg "github.com/useryege/athena/pkg/apiclient/appbootstrap"
	sessionmgr "github.com/useryege/athena/util/session"
)

// Server provides the application bootstrap service.
type Server struct {
	settingsProjector *settingsserver.Projector
	accessController  *accountaccess.Controller
	accountCenter     *accountcenter.Manager
	authenticator     Authenticator
}

type Authenticator interface {
	Authenticate(ctx context.Context) (context.Context, error)
}

// NewServer creates the application bootstrap service.
func NewServer(settingsProjector *settingsserver.Projector, accessController *accountaccess.Controller, accountCenter *accountcenter.Manager, authenticator Authenticator) *Server {
	return &Server{
		settingsProjector: settingsProjector,
		accessController:  accessController,
		accountCenter:     accountCenter,
		authenticator:     authenticator,
	}
}

// GetAppBootstrap returns settings and the initial browser session state.
func (s *Server) GetAppBootstrap(ctx context.Context, _ *appbootstrappkg.GetAppBootstrapRequest) (*appbootstrappkg.GetAppBootstrapResponse, error) {
	projectedSettings, err := s.settingsProjector.Project(ctx)
	if err != nil {
		return nil, err
	}

	projectedSession := &appbootstrappkg.AppBootstrapSession{
		Status: appbootstrappkg.AppBootstrapSessionStatus_APP_BOOTSTRAP_SESSION_STATUS_ANONYMOUS,
	}
	authErr, _ := ctx.Value(sessionmgr.AuthErrorCtxKey).(error)
	if sessionmgr.IsAccountMaintenanceError(authErr) {
		projectedSession.Status = appbootstrappkg.AppBootstrapSessionStatus_APP_BOOTSTRAP_SESSION_STATUS_ACCOUNT_MAINTENANCE
	} else if sessionmgr.LoggedIn(ctx) {
		userInfo, err := sessionserver.ProjectUserInfo(ctx, s.accessController, s.accountCenter)
		if err != nil {
			return nil, err
		}
		projectedSession.Status = appbootstrappkg.AppBootstrapSessionStatus_APP_BOOTSTRAP_SESSION_STATUS_AUTHENTICATED
		projectedSession.UserInfo = userInfo
	}

	return &appbootstrappkg.GetAppBootstrapResponse{
		Settings: projectedSettings,
		Session:  projectedSession,
	}, nil
}

// AuthFuncOverride makes bootstrap public while preserving valid claims or a
// maintenance outcome projected by the shared authenticator.
func (s *Server) AuthFuncOverride(ctx context.Context, _ string) (context.Context, error) {
	if s.authenticator == nil {
		return ctx, nil
	}
	ctx, _ = s.authenticator.Authenticate(ctx)
	return ctx, nil
}
