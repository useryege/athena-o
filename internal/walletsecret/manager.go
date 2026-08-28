package walletsecret

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/useryege/athena/internal/accountcredentials"
	"github.com/useryege/athena/internal/authregistration"
)

const (
	CookieName                   = "athena.wallet-secret.lease"
	WormCredentialCookieName     = "athena.worm-trading.lease"
	ScopePrivateKeyReveal        = "wallet.private_key.reveal"
	ScopeWormAPICredentialManage = "worm.api_credential.manage"
	LeaseTTL                     = 5 * time.Minute
	leaseKeyPrefix               = "wallet-secret-lease|"
	wormCredentialLeaseKeyPrefix = "worm-credential-lease|"
	leaseIssueAttempts           = 3
)

type lease struct {
	AccountID        string `json:"accountId"`
	SessionJTIDigest string `json:"sessionJtiDigest"`
	AccessRevision   uint64 `json:"accessRevision"`
	Scope            string `json:"scope"`
	IssuedAt         int64  `json:"issuedAt"`
	ExpiresAt        int64  `json:"expiresAt"`
}

// Manager owns short-lived, non-sliding authorization leases for revealing
// custodial wallet private keys.
type Manager struct {
	redis             *redis.Client
	cookieName        string
	cookiePath        string
	scope             string
	keyPrefix         string
	loginRequired     error
	reauthRequired    error
	reauthUnavailable error
	secureCookie      bool
}

func NewManager(redisClient *redis.Client, baseHRef string, secureCookie bool) (*Manager, error) {
	return newManager(
		redisClient,
		CookieName,
		LeaseCookiePath(baseHRef),
		ScopePrivateKeyReveal,
		leaseKeyPrefix,
		ErrLoginSessionRequired,
		ErrReauthenticationRequired,
		ErrReauthenticationUnavailable,
		secureCookie,
	)
}

// NewWormCredentialManager constructs the independent five-minute lease used
// only for creating, replacing, or revoking Worm API credentials.
func NewWormCredentialManager(redisClient *redis.Client, baseHRef string, secureCookie bool) (*Manager, error) {
	return newManager(
		redisClient,
		WormCredentialCookieName,
		WormCredentialLeaseCookiePath(baseHRef),
		ScopeWormAPICredentialManage,
		wormCredentialLeaseKeyPrefix,
		ErrWormLoginSessionRequired,
		ErrWormReauthenticationRequired,
		ErrWormReauthenticationUnavailable,
		secureCookie,
	)
}

func newManager(
	redisClient *redis.Client,
	cookieName, cookiePath, scope, keyPrefix string,
	loginRequired, reauthRequired, reauthUnavailable error,
	secureCookie bool,
) (*Manager, error) {
	if redisClient == nil {
		return nil, fmt.Errorf("sensitive-operation Redis client is required")
	}
	if cookieName == "" || cookiePath == "" || scope == "" || keyPrefix == "" || loginRequired == nil || reauthRequired == nil || reauthUnavailable == nil {
		return nil, fmt.Errorf("sensitive-operation lease configuration is incomplete")
	}
	return &Manager{
		redis:             redisClient,
		cookieName:        cookieName,
		cookiePath:        cookiePath,
		scope:             scope,
		keyPrefix:         keyPrefix,
		loginRequired:     loginRequired,
		reauthRequired:    reauthRequired,
		reauthUnavailable: reauthUnavailable,
		secureCookie:      secureCookie,
	}, nil
}

// LeaseCookiePath restricts the opaque lease to wallet API resources under the
// configured external base href.
func LeaseCookiePath(baseHRef string) string {
	base := strings.Trim(strings.TrimSpace(baseHRef), "/")
	if base == "" {
		return "/api/v1/wallets"
	}
	return "/" + base + "/api/v1/wallets"
}

// WormCredentialLeaseCookiePath restricts the independent Worm credential
// lease to Worm Trading native HTTP resources.
func WormCredentialLeaseCookiePath(baseHRef string) string {
	base := strings.Trim(strings.TrimSpace(baseHRef), "/")
	if base == "" {
		return "/api/v1/worm-trading"
	}
	return "/" + base + "/api/v1/worm-trading"
}

// SessionJTIDigest returns the non-reversible lease binding stored in Redis.
func SessionJTIDigest(jti string) string {
	digest := sha256.Sum256([]byte(jti))
	return hex.EncodeToString(digest[:])
}

// Issue creates a new five-minute lease and writes its opaque bearer cookie.
// The TTL never changes after this operation.
func (m *Manager) Issue(ctx context.Context, w http.ResponseWriter, credential accountcredentials.AuthenticatedCredential) (time.Time, error) {
	if m == nil || m.redis == nil {
		return time.Time{}, ErrReauthenticationUnavailable
	}
	if !credential.IsInteractiveLogin() || credential.AccountID == "" || credential.JTI == "" || credential.AccessRevision == 0 {
		return time.Time{}, m.loginRequired
	}
	now := time.Now().UTC().Truncate(time.Second)
	expiresAt := now.Add(LeaseTTL)
	value := lease{
		AccountID:        credential.AccountID,
		SessionJTIDigest: SessionJTIDigest(credential.JTI),
		AccessRevision:   credential.AccessRevision,
		Scope:            m.scope,
		IssuedAt:         now.Unix(),
		ExpiresAt:        expiresAt.Unix(),
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return time.Time{}, m.reauthUnavailable
	}
	for attempt := 0; attempt < leaseIssueAttempts; attempt++ {
		opaque, err := authregistration.RandomOpaqueValue()
		if err != nil {
			return time.Time{}, m.reauthUnavailable
		}
		created, err := m.redis.SetNX(ctx, m.leaseKey(opaque), encoded, LeaseTTL).Result()
		if err != nil {
			return time.Time{}, m.reauthUnavailable
		}
		if !created {
			continue
		}
		m.setCookie(w, opaque, expiresAt)
		return expiresAt, nil
	}
	return time.Time{}, m.reauthUnavailable
}

// Validate checks current account, session, access revision, scope, and fixed
// expiry without extending Redis TTL.
func (m *Manager) Validate(ctx context.Context, r *http.Request, credential accountcredentials.AuthenticatedCredential) error {
	if m == nil || m.redis == nil {
		return ErrReauthenticationUnavailable
	}
	if !credential.IsInteractiveLogin() || credential.AccountID == "" || credential.JTI == "" || credential.AccessRevision == 0 {
		return m.loginRequired
	}
	cookie, err := r.Cookie(m.cookieName)
	if err != nil || !authregistration.ValidOpaqueValue(cookie.Value) {
		return m.reauthRequired
	}
	encoded, err := m.redis.Get(ctx, m.leaseKey(cookie.Value)).Bytes()
	if errors.Is(err, redis.Nil) {
		return m.reauthRequired
	}
	if err != nil {
		return m.reauthUnavailable
	}
	var stored lease
	if err := json.Unmarshal(encoded, &stored); err != nil {
		return m.reauthRequired
	}
	now := time.Now().UTC()
	if stored.AccountID == "" || stored.SessionJTIDigest == "" || stored.Scope != m.scope || stored.AccessRevision == 0 ||
		stored.IssuedAt <= 0 || stored.ExpiresAt-stored.IssuedAt != int64(LeaseTTL/time.Second) || stored.IssuedAt > now.Add(time.Minute).Unix() || stored.ExpiresAt <= now.Unix() {
		return m.reauthRequired
	}
	if !authregistration.ConstantTimeEqual(stored.AccountID, credential.AccountID) ||
		!authregistration.ConstantTimeEqual(stored.SessionJTIDigest, SessionJTIDigest(credential.JTI)) ||
		stored.AccessRevision != credential.AccessRevision {
		return m.reauthRequired
	}
	return nil
}

// ClearCookie removes the browser lease. The server record expires naturally;
// a revoked login session cannot satisfy validation even if a stale cookie is
// replayed.
func (m *Manager) ClearCookie(w http.ResponseWriter) {
	if m == nil {
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     m.cookieName,
		Value:    "",
		Path:     m.cookiePath,
		MaxAge:   -1,
		Expires:  time.Unix(1, 0),
		HttpOnly: true,
		Secure:   m.secureCookie,
		SameSite: http.SameSiteStrictMode,
	})
}

func (m *Manager) setCookie(w http.ResponseWriter, value string, expiresAt time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     m.cookieName,
		Value:    value,
		Path:     m.cookiePath,
		MaxAge:   int(LeaseTTL / time.Second),
		Expires:  expiresAt,
		HttpOnly: true,
		Secure:   m.secureCookie,
		SameSite: http.SameSiteStrictMode,
	})
}

func (m *Manager) leaseKey(opaque string) string {
	digest := sha256.Sum256([]byte(opaque))
	return fmt.Sprintf("%s%x", m.keyPrefix, digest[:])
}
