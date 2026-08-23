package session

import (
	"context"
	"errors"
	"math"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/useryege/athena/common"
	"github.com/useryege/athena/internal/accountaccess"
	"github.com/useryege/athena/internal/accountcredentials"
	"github.com/useryege/athena/util/env"
	httputil "github.com/useryege/athena/util/http"
	jwtutil "github.com/useryege/athena/util/jwt"
)

// SessionManager generates and validates JWT tokens for login sessions.
type SessionManager struct {
	credentials      *accountcredentials.CredentialManager
	jwtCodec         *accountcredentials.JWTCodec
	accessController *accountaccess.Controller
	// projectsLister                v1alpha1.AppProjectNamespaceLister
	storage                       UserStateStorage
	sleep                         func(d time.Duration)
	verificationDelayNoiseEnabled bool
	loginRateLimit                loginRateLimitConfig
	metricsRegistry               MetricsRegistry
}

type MetricsRegistry interface {
	IncLoginRequestCounter(status string)
}

const (
	AuthErrorCtxKey = "auth-error"

	// invalidLoginError, for security purposes, doesn't say whether the username or password was invalid.  This does not mitigate the potential for timing attacks to determine which is which.
	invalidLoginError         = "Invalid username or password"
	blankPasswordError        = "Blank passwords are not allowed"
	usernameTooLongError      = "Username is too long (%d bytes max)"
	userDoesNotHaveCapability = "Account %s does not have %s capability"
)

// AccountMaintenanceMessage is the stable client-visible message for disabled accounts.
const AccountMaintenanceMessage = "系统维护中"

const (
	// Maximum length of username, too keep the cache's memory signature low
	maxUsernameLength = 32
	// The password verification delay max
	verificationDelayNoiseMin = 500 * time.Millisecond
	// The password verification delay max
	verificationDelayNoiseMax = 1000 * time.Millisecond
)

const (
	defaultLoginUserMaxFailures    = 5
	defaultLoginIPMaxFailures      = 20
	defaultLoginUserIPMaxFailures  = 5
	defaultLoginGlobalMaxFailures  = 200
	defaultLoginFailureWindow      = 15 * time.Minute
	defaultLoginLockDuration       = 15 * time.Minute
	defaultLoginGlobalWindow       = 5 * time.Minute
	defaultLoginGlobalLockDuration = time.Minute
	envLoginUserMaxFailures        = "ATHENA_LOGIN_USER_MAX_FAILURES"
	envLoginIPMaxFailures          = "ATHENA_LOGIN_IP_MAX_FAILURES"
	envLoginUserIPMaxFailures      = "ATHENA_LOGIN_USER_IP_MAX_FAILURES"
	envLoginGlobalMaxFailures      = "ATHENA_LOGIN_GLOBAL_MAX_FAILURES"
	envLoginFailureWindow          = "ATHENA_LOGIN_FAILURE_WINDOW"
	envLoginLockDuration           = "ATHENA_LOGIN_LOCK_DURATION"
	envLoginGlobalWindow           = "ATHENA_LOGIN_GLOBAL_WINDOW"
	envLoginGlobalLockDuration     = "ATHENA_LOGIN_GLOBAL_LOCK_DURATION"
	loginRateLimitCounterPrefix    = "login-fail"
	loginRateLimitLockPrefix       = "login-lock"
	loginRateLimitUserDimension    = "user"
	loginRateLimitIPDimension      = "ip"
	loginRateLimitUserIPDimension  = "user-ip"
	loginRateLimitGlobalDimension  = "global"
)

var InvalidLoginErr = status.Errorf(codes.Unauthenticated, invalidLoginError)

// AccountMaintenanceErr is returned when valid credentials belong to a disabled account.
// Its gRPC Unavailable code maps to HTTP 503 through grpc-gateway.
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

var errLoginRateLimited = errors.New("login rate limited")

type loginRateLimitConfig struct {
	UserMaxFailures    int
	IPMaxFailures      int
	UserIPMaxFailures  int
	GlobalMaxFailures  int
	FailureWindow      time.Duration
	LockDuration       time.Duration
	GlobalWindow       time.Duration
	GlobalLockDuration time.Duration
}

type loginRateLimitRule struct {
	Name        string
	CounterKey  string
	LockKey     string
	MaxFailures int
	Window      time.Duration
	LockFor     time.Duration
}

func newLoginRateLimitConfig() loginRateLimitConfig {
	return loginRateLimitConfig{
		UserMaxFailures:    env.ParseNumFromEnv(envLoginUserMaxFailures, defaultLoginUserMaxFailures, 0, math.MaxInt32),
		IPMaxFailures:      env.ParseNumFromEnv(envLoginIPMaxFailures, defaultLoginIPMaxFailures, 0, math.MaxInt32),
		UserIPMaxFailures:  env.ParseNumFromEnv(envLoginUserIPMaxFailures, defaultLoginUserIPMaxFailures, 0, math.MaxInt32),
		GlobalMaxFailures:  env.ParseNumFromEnv(envLoginGlobalMaxFailures, defaultLoginGlobalMaxFailures, 0, math.MaxInt32),
		FailureWindow:      env.ParseDurationFromEnv(envLoginFailureWindow, defaultLoginFailureWindow, 0, time.Duration(math.MaxInt64)),
		LockDuration:       env.ParseDurationFromEnv(envLoginLockDuration, defaultLoginLockDuration, 0, time.Duration(math.MaxInt64)),
		GlobalWindow:       env.ParseDurationFromEnv(envLoginGlobalWindow, defaultLoginGlobalWindow, 0, time.Duration(math.MaxInt64)),
		GlobalLockDuration: env.ParseDurationFromEnv(envLoginGlobalLockDuration, defaultLoginGlobalLockDuration, 0, time.Duration(math.MaxInt64)),
	}
}

// NewSessionManager creates a session manager from independent credential,
// signing, access, and user-state dependencies.
func NewSessionManager(credentials *accountcredentials.CredentialManager, jwtCodec *accountcredentials.JWTCodec, storage UserStateStorage, accessController *accountaccess.Controller) *SessionManager {
	return &SessionManager{
		credentials:      credentials,
		jwtCodec:         jwtCodec,
		accessController: accessController,
		storage:          storage,
		sleep:            time.Sleep,
		// projectsLister:                projectsLister,
		verificationDelayNoiseEnabled: true,
		loginRateLimit:                newLoginRateLimitConfig(),
	}
}

func (mgr *SessionManager) CollectMetrics(registry MetricsRegistry) {
	mgr.metricsRegistry = registry
	if mgr.metricsRegistry == nil {
		log.Warn("Metrics registry is not set, metrics will not be collected")
		return
	}
}

func (mgr *SessionManager) IncLoginRequestCounter(status string) {
	if mgr.metricsRegistry != nil {
		mgr.metricsRegistry.IncLoginRequestCounter(status)
	}
}

// Parse tries to parse the provided string and returns the token claims for local login.
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

	if err := mgr.credentials.ValidateCredential(parsed.Account, parsed.Capability, parsed.ID, parsed.CredentialEpoch); err != nil {
		return nil, "", err
	}
	if parsed.ID == "" || mgr.storage.IsTokenRevoked(parsed.ID) {
		return nil, "", errors.New("token is revoked, please re-login")
	}
	return parsed.Claims, "", nil
}

func (mgr *SessionManager) loginRateLimitRules(username, clientIP string) []loginRateLimitRule {
	cfg := mgr.loginRateLimit
	rules := []loginRateLimitRule{
		mgr.loginRateLimitRule(loginRateLimitUserDimension, username, cfg.UserMaxFailures, cfg.FailureWindow, cfg.LockDuration),
		mgr.loginRateLimitRule(loginRateLimitGlobalDimension, "", cfg.GlobalMaxFailures, cfg.GlobalWindow, cfg.GlobalLockDuration),
	}
	if clientIP != "" {
		rules = append(rules,
			mgr.loginRateLimitRule(loginRateLimitIPDimension, clientIP, cfg.IPMaxFailures, cfg.FailureWindow, cfg.LockDuration),
			mgr.loginRateLimitRule(loginRateLimitUserIPDimension, username+"|"+clientIP, cfg.UserIPMaxFailures, cfg.FailureWindow, cfg.LockDuration),
		)
	}
	return rules
}

func (mgr *SessionManager) loginRateLimitClearRules(username, clientIP string) []loginRateLimitRule {
	cfg := mgr.loginRateLimit
	rules := []loginRateLimitRule{
		mgr.loginRateLimitRule(loginRateLimitUserDimension, username, cfg.UserMaxFailures, cfg.FailureWindow, cfg.LockDuration),
	}
	if clientIP != "" {
		rules = append(rules, mgr.loginRateLimitRule(loginRateLimitUserIPDimension, username+"|"+clientIP, cfg.UserIPMaxFailures, cfg.FailureWindow, cfg.LockDuration))
	}
	return rules
}

func (mgr *SessionManager) loginRateLimitRule(name, value string, maxFailures int, window, lockFor time.Duration) loginRateLimitRule {
	keyParts := []string{loginRateLimitCounterPrefix, name}
	lockKeyParts := []string{loginRateLimitLockPrefix, name}
	if value != "" {
		keyValue := url.QueryEscape(value)
		keyParts = append(keyParts, keyValue)
		lockKeyParts = append(lockKeyParts, keyValue)
	}
	return loginRateLimitRule{
		Name:        name,
		CounterKey:  strings.Join(keyParts, "|"),
		LockKey:     strings.Join(lockKeyParts, "|"),
		MaxFailures: maxFailures,
		Window:      window,
		LockFor:     lockFor,
	}
}

func (mgr *SessionManager) withVerificationDelay(start time.Time) {
	if !mgr.verificationDelayNoiseEnabled {
		return
	}
	delayNanoseconds := verificationDelayNoiseMin.Nanoseconds() +
		int64(rand.Intn(int(verificationDelayNoiseMax.Nanoseconds()-verificationDelayNoiseMin.Nanoseconds())))
	delayNanoseconds -= time.Since(start).Nanoseconds()
	if delayNanoseconds > 0 {
		mgr.sleep(time.Duration(delayNanoseconds))
	}
}

// VerifyLogin verifies login credentials and applies Redis-backed brute-force protection.
func (mgr *SessionManager) VerifyLogin(ctx context.Context, username string, password string, clientIP string) (accountcredentials.PasswordVerification, error) {
	start := time.Now()
	defer mgr.withVerificationDelay(start)

	if password == "" {
		return accountcredentials.PasswordVerification{}, status.Errorf(codes.Unauthenticated, blankPasswordError)
	}
	if len(username) > maxUsernameLength {
		return accountcredentials.PasswordVerification{}, status.Errorf(codes.InvalidArgument, usernameTooLongError, maxUsernameLength)
	}

	rules := mgr.loginRateLimitRules(username, clientIP)
	if err := mgr.storage.CheckLoginRateLimit(ctx, rules); err != nil {
		if !errors.Is(err, errLoginRateLimited) {
			log.Warnf("failed to check login rate limit: %v", err)
		}
		return accountcredentials.PasswordVerification{}, InvalidLoginErr
	}

	verification, err := mgr.verifyUsernamePassword(username, password)
	if err != nil {
		if IsAccountMaintenanceError(err) {
			return accountcredentials.PasswordVerification{}, err
		}
		if recordErr := mgr.storage.RecordLoginFailure(ctx, rules); recordErr != nil {
			log.Warnf("failed to record login failure: %v", recordErr)
		}
		return accountcredentials.PasswordVerification{}, InvalidLoginErr
	}

	if err := mgr.storage.ClearLoginFailures(ctx, mgr.loginRateLimitClearRules(username, clientIP)); err != nil {
		log.Warnf("failed to clear login failures: %v", err)
		return accountcredentials.PasswordVerification{}, InvalidLoginErr
	}
	return verification, nil
}

// VerifyUsernamePassword verifies if a username/password combo is correct.
func (mgr *SessionManager) VerifyUsernamePassword(username string, password string) error {
	start := time.Now()
	defer mgr.withVerificationDelay(start)
	_, err := mgr.verifyUsernamePassword(username, password)
	return err
}

func (mgr *SessionManager) verifyUsernamePassword(username string, password string) (accountcredentials.PasswordVerification, error) {
	if password == "" {
		return accountcredentials.PasswordVerification{}, status.Errorf(codes.Unauthenticated, blankPasswordError)
	}
	if len(username) > maxUsernameLength {
		return accountcredentials.PasswordVerification{}, status.Errorf(codes.InvalidArgument, usernameTooLongError, maxUsernameLength)
	}

	verification, err := mgr.credentials.VerifyPassword(username, password)
	if err != nil {
		if errors.Is(err, accountcredentials.ErrInvalidCredentials) {
			return accountcredentials.PasswordVerification{}, InvalidLoginErr
		}
		return accountcredentials.PasswordVerification{}, err
	}
	account, err := mgr.credentials.Get(username)
	if err != nil {
		return accountcredentials.PasswordVerification{}, InvalidLoginErr
	}

	access, err := mgr.accessController.Get(username)
	if err != nil {
		return accountcredentials.PasswordVerification{}, err
	}
	if !access.LoginEnabled {
		return accountcredentials.PasswordVerification{}, AccountMaintenanceErr
	}

	if !account.HasCapability(accountcredentials.CapabilityLogin) {
		return accountcredentials.PasswordVerification{}, status.Errorf(codes.Unauthenticated, userDoesNotHaveCapability, username, accountcredentials.CapabilityLogin)
	}
	return verification, nil
}

// CreateVerifiedLogin signs a login session only if the password version used
// by VerifyLogin is still current.
func (mgr *SessionManager) CreateVerifiedLogin(verification accountcredentials.PasswordVerification, secondsBeforeExpiry int64, id string) (string, error) {
	token, err := mgr.credentials.IssueLoginSession(verification, id, secondsBeforeExpiry)
	if errors.Is(err, accountcredentials.ErrInvalidCredentials) {
		return "", InvalidLoginErr
	}
	return token, err
}

// AuthMiddlewareFunc returns a function that can be used as an
// authentication middleware for HTTP requests.
func (mgr *SessionManager) AuthMiddlewareFunc(disabled bool) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return WithAuthMiddleware(disabled, mgr, h)
	}
}

// TokenVerifier defines the contract to invoke token
// verification logic
type TokenVerifier interface {
	VerifyToken(ctx context.Context, token string) (jwt.Claims, string, error)
}

// WithAuthMiddleware is an HTTP middleware used to ensure incoming
// requests are authenticated before invoking the target handler. If
// disabled is true, it will just invoke the next handler in the chain.
func WithAuthMiddleware(disabled bool, authn TokenVerifier, next http.Handler) http.Handler {
	if disabled {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)
		})
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookies := r.Cookies()
		ctx := r.Context()
		tokenString, err := httputil.JoinCookies(common.AuthCookieName, cookies)
		if err != nil {
			http.Error(w, "Auth cookie not found", http.StatusBadRequest)
			return
		}
		claims, _, err := authn.VerifyToken(ctx, tokenString)
		if err != nil {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		// Add claims to the request context for account authorization.
		//nolint:staticcheck
		ctx = context.WithValue(ctx, "claims", claims)
		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	})
}

// VerifyToken verifies Athena-issued session and API tokens.
func (mgr *SessionManager) VerifyToken(_ context.Context, tokenString string) (jwt.Claims, string, error) {
	return mgr.Parse(tokenString)
}

func (mgr *SessionManager) RevokeToken(ctx context.Context, id string, expiringAt time.Duration) error {
	return mgr.storage.RevokeToken(ctx, id, expiringAt)
}

func LoggedIn(ctx context.Context) bool {
	return GetUserIdentifier(ctx) != "" && ctx.Value(AuthErrorCtxKey) == nil
}

// Username is a helper to extract a human readable username from a context
func Username(ctx context.Context) string {
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

// GetUserIdentifier returns the user identifier from context.
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
