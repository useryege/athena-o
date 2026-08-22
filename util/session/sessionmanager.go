package session

import (
	"context"
	"errors"
	"fmt"
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
	"github.com/useryege/athena/util/env"
	httputil "github.com/useryege/athena/util/http"
	jwtutil "github.com/useryege/athena/util/jwt"
	passwordutil "github.com/useryege/athena/util/password"
	"github.com/useryege/athena/util/settings"
)

// SessionManager generates and validates JWT tokens for login sessions.
type SessionManager struct {
	settingsMgr *settings.SettingsManager
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
	// SessionManagerClaimsIssuer fills the "iss" field of the token.
	SessionManagerClaimsIssuer = "athena"
	AuthErrorCtxKey            = "auth-error"

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

// NewSessionManager creates a new session manager from Athena settings.
func NewSessionManager(settingsMgr *settings.SettingsManager, storage UserStateStorage) *SessionManager {
	return &SessionManager{
		settingsMgr: settingsMgr,
		storage:     storage,
		sleep:       time.Sleep,
		// projectsLister:                projectsLister,
		verificationDelayNoiseEnabled: true,
		loginRateLimit:                newLoginRateLimitConfig(),
	}
}

// Create creates a new token for a given subject (user) and returns it as a string.
// Passing a value of `0` for secondsBeforeExpiry creates a token that never expires.
// The id parameter holds an optional unique JWT token identifier and stored as a standard claim "jti" in the JWT token.
func (mgr *SessionManager) Create(subject string, secondsBeforeExpiry int64, id string) (string, error) {
	now := time.Now().UTC()
	claims := jwt.RegisteredClaims{
		IssuedAt:  jwt.NewNumericDate(now),
		Issuer:    SessionManagerClaimsIssuer,
		NotBefore: jwt.NewNumericDate(now),
		Subject:   subject,
		ID:        id,
	}
	if secondsBeforeExpiry > 0 {
		expires := now.Add(time.Duration(secondsBeforeExpiry) * time.Second)
		claims.ExpiresAt = jwt.NewNumericDate(expires)
	}

	return mgr.signClaims(claims)
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

func (mgr *SessionManager) signClaims(claims jwt.Claims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	settings, err := mgr.settingsMgr.GetSettings()
	if err != nil {
		return "", err
	}
	return token.SignedString(settings.ServerSignature)
}

// GetSubjectAccountAndCapability analyzes Athena account token subject and extract account name
// and the capability it was generated for (default capability is API Key).
func GetSubjectAccountAndCapability(subject string) (string, settings.AccountCapability) {
	capability := settings.AccountCapabilityApiKey
	if parts := strings.Split(subject, ":"); len(parts) > 1 {
		subject = parts[0]
		switch parts[1] {
		case string(settings.AccountCapabilityLogin):
			capability = settings.AccountCapabilityLogin
		case string(settings.AccountCapabilityApiKey):
			capability = settings.AccountCapabilityApiKey
		}
	}
	return subject, capability
}

// Parse tries to parse the provided string and returns the token claims for local login.
func (mgr *SessionManager) Parse(tokenString string) (jwt.Claims, string, error) {
	// Parse takes the token string and a function for looking up the key. The latter is especially
	// useful if you use multiple keys for your application.  The standard is to use 'kid' in the
	// head of the token to identify which key to use, but the parsed token (head and claims) is provided
	// to the callback, providing flexibility.
	var claims jwt.MapClaims
	athenaSettings, err := mgr.settingsMgr.GetSettings()
	if err != nil {
		return nil, "", err
	}
	token, err := jwt.ParseWithClaims(tokenString, &claims, func(token *jwt.Token) (any, error) {
		// Don't forget to validate the alg is what you expect:
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return athenaSettings.ServerSignature, nil
	})
	if err != nil {
		return nil, "", err
	}

	issuedAt, err := jwtutil.IssuedAtTime(claims)
	if err != nil {
		return nil, "", err
	}

	subject := jwtutil.GetUserIdentifier(claims)
	id := jwtutil.StringField(claims, "jti")

	// if projName, role, ok := rbacpolicy.GetProjectRoleFromSubject(subject); ok {
	// 	proj, err := mgr.projectsLister.Get(projName)
	// 	if err != nil {
	// 		return nil, "", err
	// 	}
	// 	_, _, err = proj.GetJWTToken(role, issuedAt.Unix(), id)
	// 	if err != nil {
	// 		return nil, "", err
	// 	}

	// 	return token.Claims, "", nil
	// }

	subject, capability := GetSubjectAccountAndCapability(subject)
	claims["sub"] = subject

	account, err := mgr.settingsMgr.GetAccount(subject)
	if err != nil {
		return nil, "", err
	}

	if !account.Enabled {
		return nil, "", AccountMaintenanceErr
	}

	if !account.HasCapability(capability) {
		return nil, "", fmt.Errorf("account %s does not have '%s' capability", subject, capability)
	}

	if id == "" || mgr.storage.IsTokenRevoked(id) {
		return nil, "", errors.New("token is revoked, please re-login")
	} else if capability == settings.AccountCapabilityApiKey && account.TokenIndex(id) == -1 {
		return nil, "", fmt.Errorf("account %s does not have token with id %s", subject, id)
	}

	if account.PasswordMtime != nil && issuedAt.Before(*account.PasswordMtime) {
		return nil, "", errors.New("account password has changed since token issued")
	}

	return token.Claims, "", nil
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
func (mgr *SessionManager) VerifyLogin(ctx context.Context, username string, password string, clientIP string) error {
	start := time.Now()
	defer mgr.withVerificationDelay(start)

	if password == "" {
		return status.Errorf(codes.Unauthenticated, blankPasswordError)
	}
	if len(username) > maxUsernameLength {
		return status.Errorf(codes.InvalidArgument, usernameTooLongError, maxUsernameLength)
	}

	rules := mgr.loginRateLimitRules(username, clientIP)
	if err := mgr.storage.CheckLoginRateLimit(ctx, rules); err != nil {
		if !errors.Is(err, errLoginRateLimited) {
			log.Warnf("failed to check login rate limit: %v", err)
		}
		return InvalidLoginErr
	}

	if err := mgr.verifyUsernamePassword(username, password); err != nil {
		if IsAccountMaintenanceError(err) {
			return err
		}
		if recordErr := mgr.storage.RecordLoginFailure(ctx, rules); recordErr != nil {
			log.Warnf("failed to record login failure: %v", recordErr)
		}
		return InvalidLoginErr
	}

	if err := mgr.storage.ClearLoginFailures(ctx, mgr.loginRateLimitClearRules(username, clientIP)); err != nil {
		log.Warnf("failed to clear login failures: %v", err)
		return InvalidLoginErr
	}
	return nil
}

// VerifyUsernamePassword verifies if a username/password combo is correct.
func (mgr *SessionManager) VerifyUsernamePassword(username string, password string) error {
	start := time.Now()
	defer mgr.withVerificationDelay(start)
	return mgr.verifyUsernamePassword(username, password)
}

func (mgr *SessionManager) verifyUsernamePassword(username string, password string) error {
	if password == "" {
		return status.Errorf(codes.Unauthenticated, blankPasswordError)
	}
	if len(username) > maxUsernameLength {
		return status.Errorf(codes.InvalidArgument, usernameTooLongError, maxUsernameLength)
	}

	account, err := mgr.settingsMgr.GetAccount(username)
	if err != nil {
		if errStatus, ok := status.FromError(err); ok && errStatus.Code() == codes.NotFound {
			err = InvalidLoginErr
		}
		// to prevent time-based user enumeration, we must perform a password
		// hash cycle to keep response time consistent (if the function were
		// to continue and not return here)
		_, _ = passwordutil.HashPassword("for_consistent_response_time")
		return err
	}

	valid, _ := passwordutil.VerifyPassword(password, account.PasswordHash)
	if !valid {
		return InvalidLoginErr
	}

	if !account.Enabled {
		return AccountMaintenanceErr
	}

	if !account.HasCapability(settings.AccountCapabilityLogin) {
		return status.Errorf(codes.Unauthenticated, userDoesNotHaveCapability, username, settings.AccountCapabilityLogin)
	}
	return nil
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

		// Add claims to the context to inspect for RBAC
		//nolint:staticcheck
		ctx = context.WithValue(ctx, "claims", claims)
		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
	})
}

// VerifyToken verifies Athena-issued session and API tokens.
func (mgr *SessionManager) VerifyToken(_ context.Context, tokenString string) (jwt.Claims, string, error) {
	parser := jwt.NewParser(jwt.WithoutClaimsValidation())
	claims := jwt.MapClaims{}
	_, _, err := parser.ParseUnverified(tokenString, &claims)
	if err != nil {
		return nil, "", err
	}
	issuer, _ := claims["iss"].(string)
	if issuer != SessionManagerClaimsIssuer {
		return nil, "", common.ErrTokenVerification
	}
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

func Groups(ctx context.Context, scopes []string) []string {
	mapClaims, ok := mapClaims(ctx)
	if !ok {
		return nil
	}
	return jwtutil.GetGroups(mapClaims, scopes)
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
