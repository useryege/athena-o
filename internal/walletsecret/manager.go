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
	CookieName            = "athena.wallet-secret.lease"
	ScopePrivateKeyReveal = "wallet.private_key.reveal"
	LeaseTTL              = 5 * time.Minute
	leaseKeyPrefix        = "wallet-secret-lease|"
	leaseIssueAttempts    = 3
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
	redis        *redis.Client
	cookiePath   string
	secureCookie bool
}

func NewManager(redisClient *redis.Client, baseHRef string, secureCookie bool) (*Manager, error) {
	if redisClient == nil {
		return nil, fmt.Errorf("wallet-secret Redis client is required")
	}
	return &Manager{
		redis:        redisClient,
		cookiePath:   LeaseCookiePath(baseHRef),
		secureCookie: secureCookie,
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
		return time.Time{}, ErrLoginSessionRequired
	}
	now := time.Now().UTC().Truncate(time.Second)
	expiresAt := now.Add(LeaseTTL)
	value := lease{
		AccountID:        credential.AccountID,
		SessionJTIDigest: SessionJTIDigest(credential.JTI),
		AccessRevision:   credential.AccessRevision,
		Scope:            ScopePrivateKeyReveal,
		IssuedAt:         now.Unix(),
		ExpiresAt:        expiresAt.Unix(),
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return time.Time{}, ErrReauthenticationUnavailable
	}
	for attempt := 0; attempt < leaseIssueAttempts; attempt++ {
		opaque, err := authregistration.RandomOpaqueValue()
		if err != nil {
			return time.Time{}, ErrReauthenticationUnavailable
		}
		created, err := m.redis.SetNX(ctx, leaseKey(opaque), encoded, LeaseTTL).Result()
		if err != nil {
			return time.Time{}, ErrReauthenticationUnavailable
		}
		if !created {
			continue
		}
		m.setCookie(w, opaque, expiresAt)
		return expiresAt, nil
	}
	return time.Time{}, ErrReauthenticationUnavailable
}

// Validate checks current account, session, access revision, scope, and fixed
// expiry without extending Redis TTL.
func (m *Manager) Validate(ctx context.Context, r *http.Request, credential accountcredentials.AuthenticatedCredential) error {
	if m == nil || m.redis == nil {
		return ErrReauthenticationUnavailable
	}
	if !credential.IsInteractiveLogin() || credential.AccountID == "" || credential.JTI == "" || credential.AccessRevision == 0 {
		return ErrLoginSessionRequired
	}
	cookie, err := r.Cookie(CookieName)
	if err != nil || !authregistration.ValidOpaqueValue(cookie.Value) {
		return ErrReauthenticationRequired
	}
	encoded, err := m.redis.Get(ctx, leaseKey(cookie.Value)).Bytes()
	if errors.Is(err, redis.Nil) {
		return ErrReauthenticationRequired
	}
	if err != nil {
		return ErrReauthenticationUnavailable
	}
	var stored lease
	if err := json.Unmarshal(encoded, &stored); err != nil {
		return ErrReauthenticationRequired
	}
	now := time.Now().UTC()
	if stored.AccountID == "" || stored.SessionJTIDigest == "" || stored.Scope != ScopePrivateKeyReveal || stored.AccessRevision == 0 ||
		stored.IssuedAt <= 0 || stored.ExpiresAt-stored.IssuedAt != int64(LeaseTTL/time.Second) || stored.IssuedAt > now.Add(time.Minute).Unix() || stored.ExpiresAt <= now.Unix() {
		return ErrReauthenticationRequired
	}
	if !authregistration.ConstantTimeEqual(stored.AccountID, credential.AccountID) ||
		!authregistration.ConstantTimeEqual(stored.SessionJTIDigest, SessionJTIDigest(credential.JTI)) ||
		stored.AccessRevision != credential.AccessRevision {
		return ErrReauthenticationRequired
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
		Name:     CookieName,
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
		Name:     CookieName,
		Value:    value,
		Path:     m.cookiePath,
		MaxAge:   int(LeaseTTL / time.Second),
		Expires:  expiresAt,
		HttpOnly: true,
		Secure:   m.secureCookie,
		SameSite: http.SameSiteStrictMode,
	})
}

func leaseKey(opaque string) string {
	digest := sha256.Sum256([]byte(opaque))
	return fmt.Sprintf("%s%x", leaseKeyPrefix, digest[:])
}
