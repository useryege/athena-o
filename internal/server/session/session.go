package session

import (
	"context"
	"fmt"

	"github.com/useryege/athena/util/settings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/useryege/athena/internal/server/rbacpolicy"
	"github.com/useryege/athena/pkg/apiclient/session"
	utilio "github.com/useryege/athena/util/io"
	"github.com/useryege/athena/util/rbac"
	sessionmgr "github.com/useryege/athena/util/session"
)

// Server provides a Session service
type Server struct {
	mgr                *sessionmgr.SessionManager
	settingsMgr        *settings.SettingsManager
	authenticator      Authenticator
	policyEnf          *rbacpolicy.RBACPolicyEnforcer
	limitLoginAttempts func() (utilio.Closer, error)
}

type Authenticator interface {
	Authenticate(ctx context.Context) (context.Context, error)
}

const (
	success = "success"
	failure = "failure"
)

var uiBootstrapPermissions = []*session.ResourcePermission{
	{Resource: rbac.ResourceTokenAPI, Action: rbac.ActionGet, Subresource: "options"},
	{Resource: rbac.ResourceTokenAPI, Action: rbac.ActionGet, Subresource: "node-statuses"},
	{Resource: rbac.ResourceTokenAPI, Action: rbac.ActionGet, Subresource: "projects"},
	{Resource: rbac.ResourceTokenAPI, Action: rbac.ActionGet, Subresource: "project-reports"},
	{Resource: rbac.ResourceTokenAPI, Action: rbac.ActionGet, Subresource: "contract-codes"},
	{Resource: rbac.ResourceTokenAPI, Action: rbac.ActionGet, Subresource: "bytecode-blacklists"},
	{Resource: rbac.ResourceTokenAPI, Action: rbac.ActionGet, Subresource: "wallet-blacklists"},
	{Resource: rbac.ResourceTokenAPI, Action: rbac.ActionGet, Subresource: "chain-checkpoints"},
	{Resource: rbac.ResourceTokenAPI, Action: rbac.ActionGet, Subresource: "collection-tasks"},
	{Resource: rbac.ResourceWallets, Action: rbac.ActionGet, Subresource: "*"},
	{Resource: rbac.ResourceWallets, Action: rbac.ActionUpdate, Subresource: "*"},
	{Resource: rbac.ResourceWallets, Action: rbac.ActionInvoke, Subresource: "*"},
	{Resource: rbac.ResourceWorm, Action: rbac.ActionGet, Subresource: "*"},
	{Resource: rbac.ResourceWormPoly, Action: rbac.ActionGet, Subresource: "*"},
	{Resource: rbac.ResourceWormPoly, Action: rbac.ActionUpdate, Subresource: "*"},
	{Resource: rbac.ResourcePolymarket, Action: rbac.ActionGet, Subresource: "*"},
	{Resource: rbac.ResourcePolymarket, Action: rbac.ActionUpdate, Subresource: "*"},
	{Resource: rbac.ResourcePolymarket, Action: rbac.ActionInvoke, Subresource: "*"},
	{Resource: rbac.ResourceNotifications, Action: rbac.ActionGet, Subresource: "*"},
	{Resource: rbac.ResourceServiceStatus, Action: rbac.ActionGet, Subresource: "*"},
	{Resource: rbac.ResourceServiceStatus, Action: rbac.ActionInvoke, Subresource: "*"},
}

// NewServer returns a new instance of the Session service
func NewServer(mgr *sessionmgr.SessionManager, settingsMgr *settings.SettingsManager, authenticator Authenticator, policyEnf *rbacpolicy.RBACPolicyEnforcer, rateLimiter func() (utilio.Closer, error)) *Server {
	return &Server{mgr, settingsMgr, authenticator, policyEnf, rateLimiter}
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
	err := s.mgr.VerifyLogin(ctx, q.Username, q.Password, clientIPFromContext(ctx))
	if err != nil {
		s.mgr.IncLoginRequestCounter(failure)
		return nil, err
	}
	// generate a unique id for the session
	uniqueId, err := uuid.NewRandom()
	if err != nil {
		s.mgr.IncLoginRequestCounter(failure)
		return nil, err
	}

	// get the athena settings from athena-cm and athena-secret
	athenaSettings, err := s.settingsMgr.GetSettings()
	if err != nil {
		s.mgr.IncLoginRequestCounter(failure)
		return nil, err
	}

	// create a JWT token for the session
	jwtToken, err := s.mgr.Create(
		fmt.Sprintf("%s:%s", q.Username, settings.AccountCapabilityLogin),
		int64(athenaSettings.UserSessionDuration.Seconds()),
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
func (s *Server) AuthFuncOverride(ctx context.Context, _ string) (context.Context, error) {
	// this authenticates the user, but ignores any error, so that we have claims populated
	ctx, _ = s.authenticator.Authenticate(ctx)
	return ctx, nil
}

func (s *Server) GetUserInfo(ctx context.Context, _ *session.GetUserInfoRequest) (*session.GetUserInfoResponse, error) {
	return &session.GetUserInfoResponse{
		LoggedIn:    sessionmgr.LoggedIn(ctx),
		Username:    sessionmgr.Username(ctx),
		Iss:         sessionmgr.Iss(ctx),
		Groups:      sessionmgr.Groups(ctx, s.policyEnf.GetScopes()),
		Permissions: s.userPermissions(ctx),
	}, nil
}

func (s *Server) userPermissions(ctx context.Context) []*session.ResourcePermission {
	if !sessionmgr.LoggedIn(ctx) {
		return nil
	}

	claims, ok := ctx.Value("claims").(jwt.Claims)
	if !ok {
		return nil
	}
	permissions := make([]*session.ResourcePermission, 0, len(uiBootstrapPermissions))
	for _, perm := range uiBootstrapPermissions {
		if s.policyEnf.EnforceClaims(claims, claims, perm.Resource, perm.Action, perm.Subresource) {
			permissions = append(permissions, &session.ResourcePermission{
				Resource:    perm.Resource,
				Action:      perm.Action,
				Subresource: perm.Subresource,
			})
		}
	}
	return permissions
}
