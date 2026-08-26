package session

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/useryege/athena/common"
	"github.com/useryege/athena/internal/accountaccess"
	"github.com/useryege/athena/internal/accountcredentials"
	httputil "github.com/useryege/athena/util/http"
	jwtutil "github.com/useryege/athena/util/jwt"
)

// SessionManager issues and validates Athena JWT v3 credentials.
type SessionManager struct {
	credentials      *accountcredentials.CredentialManager
	jwtCodec         *accountcredentials.JWTCodec
	accessController *accountaccess.Controller
	storage          UserStateStorage
	metricsRegistry  MetricsRegistry
}

type MetricsRegistry interface {
	IncLoginRequestCounter(status string)
}

const AuthErrorCtxKey = "auth-error"

// AccountMaintenanceMessage is the stable client-visible message for disabled accounts.
const AccountMaintenanceMessage = "系统维护中"

// AccountMaintenanceErr is returned when a credential belongs to a disabled account.
var AccountMaintenanceErr = status.Error(codes.Unavailable, AccountMaintenanceMessage)

// IsAccountMaintenanceError reports whether err is the stable disabled-account error.
func IsAccountMaintenanceError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, AccountMaintenanceErr) {
		return true
	}
	errStatus, ok := status.FromError(err)
	return ok && errStatus.Code() == codes.Unavailable && errStatus.Message() == AccountMaintenanceMessage
}

// NewSessionManager composes credential, signing, access, and revocation state.
func NewSessionManager(credentials *accountcredentials.CredentialManager, jwtCodec *accountcredentials.JWTCodec, storage UserStateStorage, accessController *accountaccess.Controller) *SessionManager {
	return &SessionManager{
		credentials:      credentials,
		jwtCodec:         jwtCodec,
		accessController: accessController,
		storage:          storage,
	}
}

func (mgr *SessionManager) CollectMetrics(registry MetricsRegistry) {
	mgr.metricsRegistry = registry
	if mgr.metricsRegistry == nil {
		log.Warn("Metrics registry is not set, metrics will not be collected")
	}
}

func (mgr *SessionManager) IncLoginRequestCounter(status string) {
	if mgr.metricsRegistry != nil {
		mgr.metricsRegistry.IncLoginRequestCounter(status)
	}
}

// RegisterCommittedAccountAccess publishes the access half of a newly
// committed account aggregate before its one-time registration ticket is
// consumed. CredentialManager has already published the identity at this
// point; keeping this explicit boundary ahead of cookie issuance prevents a
// durable account from remaining absent from the process authorization
// snapshot when Redis completion fails.
func (mgr *SessionManager) RegisterCommittedAccountAccess(ctx context.Context, accountID string) error {
	if mgr == nil || mgr.accessController == nil {
		return fmt.Errorf("account access controller is not configured")
	}
	_, err := mgr.accessController.Register(ctx, accountID)
	return err
}

// CreateExternalLogin validates current account state and issues a session
// bound to the provider identity that completed external authentication.
func (mgr *SessionManager) CreateExternalLogin(ctx context.Context, accountID string, verifiedProvider accountcredentials.IdentityProvider, verifiedSubject, verifiedEmail string, secondsBeforeExpiry int64, id string) (string, error) {
	account, err := mgr.credentials.Get(accountID)
	if err != nil {
		return "", err
	}
	subject, email, err := accountcredentials.NormalizeExternalIdentity(verifiedProvider, verifiedSubject, verifiedEmail, account.Administrator)
	if err != nil || !account.HasExternalIdentity() || account.IdentityProvider != verifiedProvider || account.IdentitySubject != subject {
		return "", status.Error(codes.PermissionDenied, "external identity is not authorized for login")
	}
	access, err := mgr.accessController.Register(ctx, accountID)
	if err != nil {
		return "", err
	}
	if !access.LoginEnabled {
		return "", AccountMaintenanceErr
	}
	token, err := mgr.credentials.IssueLoginSession(accountID, verifiedProvider, subject, id, secondsBeforeExpiry)
	if err != nil {
		return "", err
	}
	if _, err := mgr.credentials.RecordLogin(ctx, accountID, verifiedProvider, subject, email); err != nil {
		if errors.Is(err, accountcredentials.ErrLoginDisabled) {
			return "", AccountMaintenanceErr
		}
		return "", err
	}
	return token, nil
}

// Parse validates a local JWT and current credential, access, and revocation state.
func (mgr *SessionManager) Parse(tokenString string) (jwt.Claims, string, error) {
	parsed, err := mgr.jwtCodec.Parse(tokenString)
	if err != nil {
		return nil, "", err
	}
	access, err := mgr.accessController.Get(parsed.Account)
	if err != nil {
		return nil, "", err
	}
	if !access.LoginEnabled {
		return nil, "", AccountMaintenanceErr
	}
	if parsed.Capability == accountcredentials.CapabilityAPIKey && !access.APIKeyEnabled {
		return nil, "", status.Error(codes.PermissionDenied, "API Key access is disabled")
	}
	if err := mgr.credentials.ValidateCredential(parsed.Account, parsed.Capability, parsed.JTI, parsed.IdentityBinding); err != nil {
		return nil, "", err
	}
	if parsed.JTI == "" || mgr.storage == nil || mgr.storage.IsTokenRevoked(parsed.JTI) {
		return nil, "", errors.New("token is revoked, please re-login")
	}
	return parsed.Claims, "", nil
}

// AuthMiddlewareFunc returns authentication middleware for direct HTTP handlers.
func (mgr *SessionManager) AuthMiddlewareFunc(disabled bool) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return WithAuthMiddleware(disabled, mgr, h)
	}
}

// TokenVerifier defines token verification used by HTTP middleware.
type TokenVerifier interface {
	VerifyToken(ctx context.Context, token string) (jwt.Claims, string, error)
}

// WithAuthMiddleware authenticates an Athena cookie before invoking next.
func WithAuthMiddleware(disabled bool, authn TokenVerifier, next http.Handler) http.Handler {
	if disabled {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { next.ServeHTTP(w, r) })
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenString, err := httputil.JoinCookies(common.AuthCookieName, r.Cookies())
		if err != nil {
			http.Error(w, "Auth cookie not found", http.StatusBadRequest)
			return
		}
		claims, _, err := authn.VerifyToken(r.Context(), tokenString)
		if err != nil {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}
		//nolint:staticcheck // existing context claim boundary shared with gRPC.
		ctx := context.WithValue(r.Context(), "claims", claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// VerifyToken verifies Athena-issued session and API credentials.
func (mgr *SessionManager) VerifyToken(_ context.Context, tokenString string) (jwt.Claims, string, error) {
	return mgr.Parse(tokenString)
}

// ParseLoginForRevocation validates the signed shape of a current login token
// without consulting mutable account or revocation state. This deliberately
// narrow boundary lets logout revoke a session after its account is disabled or
// its external identity binding changes.
func (mgr *SessionManager) ParseLoginForRevocation(tokenString string) (jwt.Claims, error) {
	parsed, err := mgr.jwtCodec.Parse(tokenString)
	if err != nil {
		return nil, err
	}
	if parsed.Capability != accountcredentials.CapabilityLogin {
		return nil, fmt.Errorf("logout requires a login token")
	}
	return parsed.Claims, nil
}

func (mgr *SessionManager) RevokeToken(ctx context.Context, id string, expiringAt time.Duration) error {
	if mgr.storage == nil {
		return fmt.Errorf("session revocation storage is not configured")
	}
	return mgr.storage.RevokeToken(ctx, id, expiringAt)
}

func LoggedIn(ctx context.Context) bool {
	return GetUserIdentifier(ctx) != "" && ctx.Value(AuthErrorCtxKey) == nil
}

// AccountID extracts the stable Athena account UUID from a context.
func AccountID(ctx context.Context) string {
	mapClaims, ok := mapClaims(ctx)
	if !ok {
		return ""
	}
	return jwtutil.GetUserIdentifier(mapClaims)
}

func Iss(ctx context.Context) string {
	mapClaims, ok := mapClaims(ctx)
	if !ok {
		return ""
	}
	return jwtutil.StringField(mapClaims, "iss")
}

func Iat(ctx context.Context) (time.Time, error) {
	mapClaims, ok := mapClaims(ctx)
	if !ok {
		return time.Time{}, errors.New("unable to extract token claims")
	}
	return jwtutil.IssuedAtTime(mapClaims)
}

// GetUserIdentifier returns the stable Athena account UUID from context.
func GetUserIdentifier(ctx context.Context) string {
	mapClaims, ok := mapClaims(ctx)
	if !ok {
		return ""
	}
	return jwtutil.GetUserIdentifier(mapClaims)
}

func mapClaims(ctx context.Context) (jwt.MapClaims, bool) {
	claims, ok := ctx.Value("claims").(jwt.Claims)
	if !ok {
		return nil, false
	}
	mapClaims, err := jwtutil.MapClaims(claims)
	if err != nil {
		return nil, false
	}
	return mapClaims, true
}
