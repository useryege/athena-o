package session

import (
	"context"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/useryege/athena/common"
	"github.com/useryege/athena/internal/accountaccess"
	"github.com/useryege/athena/internal/accountcenter"
	accountserver "github.com/useryege/athena/internal/server/account"
	"github.com/useryege/athena/pkg/apiclient/session"
	utilio "github.com/useryege/athena/util/io"
	sessionmgr "github.com/useryege/athena/util/session"
)

// Server provides a Session service
type Server struct {
	mgr                 *sessionmgr.SessionManager
	userSessionDuration time.Duration
	authenticator       Authenticator
	accessController    *accountaccess.Controller
	accountCenter       *accountcenter.Manager
	limitLoginAttempts  func() (utilio.Closer, error)
}

type Authenticator interface {
	Authenticate(ctx context.Context) (context.Context, error)
}

const (
	success = "success"
	failure = "failure"
)

// NewServer returns a new instance of the Session service
func NewServer(mgr *sessionmgr.SessionManager, userSessionDuration time.Duration, authenticator Authenticator, accessController *accountaccess.Controller, accountCenter *accountcenter.Manager, rateLimiter func() (utilio.Closer, error)) *Server {
	return &Server{mgr, userSessionDuration, authenticator, accessController, accountCenter, rateLimiter}
}

// Create generates a JWT token signed by Athena intended for web/CLI logins of the admin user
// using username/password
func (s *Server) Create(ctx context.Context, q *session.SessionCreateRequest) (*session.SessionResponse, error) {
	// try to get a rate limiter if it is set
	if s.limitLoginAttempts != nil {
		closer, err := s.limitLoginAttempts()
		if err != nil {
			// if failed to get a rate limiter, increment the login request counter and return the error
			s.mgr.IncLoginRequestCounter(failure)
			return nil, err
		}
		// defer the closer to release the rate limiter
		defer utilio.Close(closer)
	}

	// if use a token to create a session, return an error
	if q.Token != "" {
		s.mgr.IncLoginRequestCounter(failure)
		return nil, status.Errorf(codes.Unauthenticated, "token-based session creation no longer supported. please upgrade athena cli to v0.7+")
	}

	// if no username or password is provided, increment the login request counter and return an error
	if q.Username == "" || q.Password == "" {
		s.mgr.IncLoginRequestCounter(failure)
		return nil, sessionmgr.InvalidLoginErr
	}

	if isHTTPGatewayRequest(ctx) {
		if err := s.mgr.VerifyCaptcha(ctx, q.CaptchaId, q.CaptchaAnswer); err != nil {
			s.mgr.IncLoginRequestCounter(failure)
			return nil, err
		}
	}

	// verify the username and password
	verification, err := s.mgr.VerifyLogin(ctx, q.Username, q.Password, clientIPFromContext(ctx))
	if err != nil {
		if !sessionmgr.IsAccountMaintenanceError(err) {
			s.mgr.IncLoginRequestCounter(failure)
		}
		return nil, err
	}
	// generate a unique id for the session
	uniqueId, err := uuid.NewRandom()
	if err != nil {
		s.mgr.IncLoginRequestCounter(failure)
		return nil, err
	}

	// create a JWT token for the session
	jwtToken, err := s.mgr.CreateVerifiedLogin(
		verification,
		int64(s.userSessionDuration.Seconds()),
		uniqueId.String())
	if err != nil {
		s.mgr.IncLoginRequestCounter(failure)
		return nil, err
	}
	// increment the login request counter for success
	s.mgr.IncLoginRequestCounter(success)
	// return the JWT token for the session
	return &session.SessionResponse{Token: jwtToken}, nil
}

func (s *Server) GetCaptcha(ctx context.Context, _ *session.CaptchaRequest) (*session.CaptchaResponse, error) {
	challenge, err := s.mgr.NewCaptcha(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create captcha")
	}
	return &session.CaptchaResponse{
		CaptchaId:    challenge.ID,
		ImageDataUrl: challenge.ImageDataURL,
		ExpiresIn:    challenge.ExpiresIn,
	}, nil
}

// Delete an authentication cookie from the client.  This makes sense only for the Web client.
func (s *Server) Delete(_ context.Context, _ *session.SessionDeleteRequest) (*session.SessionResponse, error) {
	return &session.SessionResponse{Token: ""}, nil
}

// AuthFuncOverride overrides the authentication function and let us not require auth to receive auth.
// Without this function here, AthenaServer.authenticate would be invoked and credentials checked.
// Since this service is generally invoked when the user has _no_ credentials, that would create a
// chicken-and-egg situation if we didn't place this here to allow traffic to pass through.
func (s *Server) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error) {
	// This authenticates the user so claims are populated. Authentication errors are
	// normally ignored because login, logout, and captcha endpoints must stay public.
	ctx, err := s.authenticator.Authenticate(ctx)
	if fullMethodName == "/session.SessionService/GetUserInfo" && sessionmgr.IsAccountMaintenanceError(err) {
		return ctx, err
	}
	return ctx, nil
}

func (s *Server) GetUserInfo(ctx context.Context, _ *session.GetUserInfoRequest) (*session.GetUserInfoResponse, error) {
	return ProjectUserInfo(ctx, s.accessController, s.accountCenter)
}

// ProjectUserInfo builds the shared session and authorization projection for ctx.
func ProjectUserInfo(ctx context.Context, accessController *accountaccess.Controller, accountCenter *accountcenter.Manager) (*session.GetUserInfoResponse, error) {
	loggedIn := sessionmgr.LoggedIn(ctx)
	response := &session.GetUserInfoResponse{
		LoggedIn: loggedIn,
		Username: sessionmgr.Username(ctx),
		Iss:      sessionmgr.Iss(ctx),
	}
	if !loggedIn {
		return response, nil
	}
	access, err := accessController.Get(response.Username)
	if err != nil {
		return nil, err
	}
	response.Administrator = response.Username == common.AthenaAdminUsername
	response.Access = accountserver.ToAPIAccountAccess(access)
	profile, err := accountCenter.GetProfile(ctx, response.Username)
	if err != nil {
		return nil, err
	}
	preferences, err := accountCenter.GetPreferences(ctx, response.Username)
	if err != nil {
		return nil, err
	}
	response.Profile = accountserver.ToAPIAccountProfile(response.Username, profile)
	response.Preferences = accountserver.ToAPIAccountPreferences(preferences)
	return response, nil
}
