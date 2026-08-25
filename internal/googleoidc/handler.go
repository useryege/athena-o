package googleoidc

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	log "github.com/sirupsen/logrus"
	"golang.org/x/oauth2"

	"github.com/useryege/athena/internal/accountcredentials"
	httputil "github.com/useryege/athena/util/http"
	sessionmgr "github.com/useryege/athena/util/session"
)

const (
	stateCookieName  = "athena.google.state"
	maxReturnToBytes = 2048
	loginSuccess     = "success"
	loginFailure     = "failure"
)

type googleClaims struct {
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
}

// Handler implements the browser Google Authorization Code + PKCE flow.
type Handler struct {
	oauth2Config    oauth2.Config
	verifier        *oidc.IDTokenVerifier
	store           *TransactionStore
	credentials     *accountcredentials.CredentialManager
	sessions        *sessionmgr.SessionManager
	sessionDuration time.Duration
	baseHRef        string
	secureCookie    bool
}

// NewHandler constructs the flow without contacting Google. Remote JWKS are
// fetched and cached only when a callback first verifies an ID token.
func NewHandler(
	config Config,
	redisClient *redis.Client,
	credentials *accountcredentials.CredentialManager,
	sessions *sessionmgr.SessionManager,
	sessionDuration time.Duration,
	baseHRef string,
) (*Handler, error) {
	store, err := NewTransactionStore(redisClient)
	if err != nil {
		return nil, err
	}
	if credentials == nil || sessions == nil {
		return nil, fmt.Errorf("Google OIDC credential and session managers are required")
	}
	if sessionDuration <= 0 {
		return nil, fmt.Errorf("Google OIDC session duration must be positive")
	}
	keySetContext := oidc.ClientContext(context.Background(), &http.Client{Timeout: 15 * time.Second})
	keySet := oidc.NewRemoteKeySet(keySetContext, googleJWKS)
	verifier := oidc.NewVerifier("https://accounts.google.com", keySet, &oidc.Config{
		ClientID: config.ClientID,
	})
	return &Handler{
		oauth2Config: oauth2.Config{
			ClientID:     config.ClientID,
			ClientSecret: config.ClientSecret,
			RedirectURL:  config.RedirectURI,
			Endpoint: oauth2.Endpoint{
				AuthURL:  googleAuthorization,
				TokenURL: googleToken,
			},
			Scopes: []string{oidc.ScopeOpenID, "email"},
		},
		verifier:        verifier,
		store:           store,
		credentials:     credentials,
		sessions:        sessions,
		sessionDuration: sessionDuration,
		baseHRef:        baseHRef,
		secureCookie:    config.secureCookie,
	}, nil
}

// Login begins one Google authorization transaction.
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	setOAuthResponseHeaders(w)
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	returnTo := ValidateReturnTo(r.URL.Query().Get("returnTo"))
	state, err := randomOpaqueValue()
	if err != nil {
		h.fail(w, r, "google_unavailable", returnTo, "state_generation", err)
		return
	}
	nonce, err := randomOpaqueValue()
	if err != nil {
		h.fail(w, r, "google_unavailable", returnTo, "nonce_generation", err)
		return
	}
	verifier := oauth2.GenerateVerifier()
	if err := h.store.Create(r.Context(), state, loginRateIdentity(r), transaction{
		Nonce:     nonce,
		Verifier:  verifier,
		ReturnTo:  returnTo,
		CreatedAt: time.Now().UTC(),
	}); err != nil {
		h.fail(w, r, "google_unavailable", returnTo, "transaction_create", err)
		return
	}
	h.setStateCookie(w, state, int(transactionTTL.Seconds()))
	authorizationURL := h.oauth2Config.AuthCodeURL(
		state,
		oauth2.S256ChallengeOption(verifier),
		oauth2.SetAuthURLParam("nonce", nonce),
		oauth2.SetAuthURLParam("prompt", "select_account"),
	)
	http.Redirect(w, r, authorizationURL, http.StatusSeeOther)
}

// Callback consumes the transaction, verifies Google identity, and issues an
// Athena-only browser session cookie.
func (h *Handler) Callback(w http.ResponseWriter, r *http.Request) {
	setOAuthResponseHeaders(w)
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	state := r.URL.Query().Get("state")
	cookie, cookieErr := r.Cookie(stateCookieName)
	h.clearStateCookie(w)
	if !validOpaqueValue(state) {
		h.fail(w, r, "google_state_invalid", "", "state_missing", nil)
		return
	}
	transaction, err := h.store.Consume(r.Context(), state)
	if err != nil {
		reason := "google_state_invalid"
		if errors.Is(err, errTransactionUnavailable) {
			reason = "google_unavailable"
		}
		h.fail(w, r, reason, "", "transaction_consume", err)
		return
	}
	returnTo := ValidateReturnTo(transaction.ReturnTo)
	if cookieErr != nil || !constantTimeEqual(cookie.Value, state) || !transactionFresh(transaction.CreatedAt) {
		h.fail(w, r, "google_state_invalid", returnTo, "state_binding", cookieErr)
		return
	}
	if r.URL.Query().Get("error") == "access_denied" {
		h.fail(w, r, "google_cancelled", returnTo, "authorization_cancelled", nil)
		return
	}
	if r.URL.Query().Get("error") != "" || r.URL.Query().Get("code") == "" {
		h.fail(w, r, "google_unavailable", returnTo, "authorization_response", nil)
		return
	}

	networkContext, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	googleToken, err := h.oauth2Config.Exchange(networkContext, r.URL.Query().Get("code"), oauth2.VerifierOption(transaction.Verifier))
	if err != nil {
		h.fail(w, r, "google_unavailable", returnTo, "code_exchange", err)
		return
	}
	rawIDToken, ok := googleToken.Extra("id_token").(string)
	if !ok || rawIDToken == "" {
		h.fail(w, r, "google_unavailable", returnTo, "id_token_missing", nil)
		return
	}
	idToken, err := h.verifier.Verify(networkContext, rawIDToken)
	if err != nil {
		h.fail(w, r, "google_unavailable", returnTo, "id_token_verify", err)
		return
	}
	if idToken.Issuer != "https://accounts.google.com" && idToken.Issuer != "accounts.google.com" {
		h.fail(w, r, "google_not_allowed", returnTo, "issuer_verify", nil)
		return
	}
	if idToken.IssuedAt.IsZero() || idToken.IssuedAt.After(time.Now().Add(time.Minute)) || idToken.Expiry.IsZero() {
		h.fail(w, r, "google_not_allowed", returnTo, "token_time_verify", nil)
		return
	}
	if idToken.Subject == "" || !constantTimeEqual(idToken.Nonce, transaction.Nonce) {
		h.fail(w, r, "google_not_allowed", returnTo, "identity_claims", nil)
		return
	}
	var claims googleClaims
	if err := idToken.Claims(&claims); err != nil || !claims.EmailVerified {
		h.fail(w, r, "google_not_allowed", returnTo, "email_verify", err)
		return
	}
	accountName, err := h.credentials.ResolveGoogleSubject(idToken.Subject)
	if err != nil {
		log.WithFields(log.Fields{
			"stage":        "identity_map",
			"google_email": claims.Email,
			"google_sub":   idToken.Subject,
		}).Warn("Verified Google identity is not mapped to an Athena account")
		h.sessions.IncLoginRequestCounter(loginFailure)
		h.redirectFailure(w, r, "google_not_allowed", returnTo)
		return
	}
	jti, err := uuid.NewRandom()
	if err != nil {
		h.fail(w, r, "google_unavailable", returnTo, "jti_generation", err)
		return
	}
	athenaToken, err := h.sessions.CreateGoogleLogin(
		accountName,
		idToken.Subject,
		int64(h.sessionDuration.Seconds()),
		jti.String(),
	)
	if err != nil {
		if sessionmgr.IsAccountMaintenanceError(err) {
			h.fail(w, r, "maintenance", returnTo, "account_maintenance", err)
			return
		}
		h.fail(w, r, "google_not_allowed", returnTo, "session_issue", err)
		return
	}
	if err := httputil.SetTokenCookie(athenaToken, h.baseHRef, h.secureCookie, w); err != nil {
		h.fail(w, r, "google_unavailable", returnTo, "cookie_issue", err)
		return
	}
	h.sessions.IncLoginRequestCounter(loginSuccess)
	log.WithFields(log.Fields{"stage": "complete", "account": accountName}).Info("Google OIDC login succeeded")
	http.Redirect(w, r, returnTo, http.StatusSeeOther)
}

// ValidateReturnTo accepts only same-origin absolute paths and supplies the
// canonical Account Center fallback for malformed or looping values.
func ValidateReturnTo(raw string) string {
	if raw == "" || len(raw) > maxReturnToBytes || !strings.HasPrefix(raw, "/") || strings.HasPrefix(raw, "//") || strings.Contains(raw, "\\") {
		return defaultReturnTo
	}
	unescapedRaw, err := url.PathUnescape(raw)
	if err != nil || strings.Contains(unescapedRaw, "\\") || strings.HasPrefix(unescapedRaw, "//") {
		return defaultReturnTo
	}
	for _, value := range unescapedRaw {
		if unicode.IsControl(value) {
			return defaultReturnTo
		}
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.IsAbs() || parsed.Host != "" || parsed.Path == "" || !strings.HasPrefix(parsed.Path, "/") || strings.HasPrefix(parsed.Path, "//") || strings.Contains(parsed.Path, "\\") {
		return defaultReturnTo
	}
	for _, value := range parsed.Path + parsed.Fragment {
		if unicode.IsControl(value) {
			return defaultReturnTo
		}
	}
	if strings.Contains(parsed.Fragment, "\\") {
		return defaultReturnTo
	}
	if parsed.Path == "/login" || strings.HasPrefix(parsed.Path, "/login/") {
		return defaultReturnTo
	}
	return parsed.String()
}

func setOAuthResponseHeaders(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Referrer-Policy", "no-referrer")
}

func loginRateIdentity(r *http.Request) string {
	identity := strings.TrimSpace(r.Header.Get("X-Real-IP"))
	if net.ParseIP(identity) == nil {
		identity = strings.TrimSpace(strings.Split(r.Header.Get("X-Forwarded-For"), ",")[0])
	}
	if net.ParseIP(identity) == nil {
		if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
			identity = host
		} else {
			identity = r.RemoteAddr
		}
	}
	if parsed := net.ParseIP(identity); parsed != nil {
		identity = parsed.String()
	} else {
		identity = "unknown"
	}
	digest := sha256.Sum256([]byte(identity))
	return hex.EncodeToString(digest[:])
}

func randomOpaqueValue() (string, error) {
	value := make([]byte, 32)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(value), nil
}

func validOpaqueValue(value string) bool {
	decoded, err := base64.RawURLEncoding.DecodeString(value)
	return err == nil && len(decoded) == 32
}

func constantTimeEqual(left, right string) bool {
	if len(left) != len(right) || left == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(left), []byte(right)) == 1
}

func transactionFresh(createdAt time.Time) bool {
	now := time.Now().UTC()
	return !createdAt.After(now.Add(time.Minute)) && now.Sub(createdAt) <= transactionTTL
}

func (h *Handler) setStateCookie(w http.ResponseWriter, value string, maxAge int) {
	http.SetCookie(w, &http.Cookie{
		Name:     stateCookieName,
		Value:    value,
		Path:     "/auth/google",
		MaxAge:   maxAge,
		Expires:  time.Now().Add(transactionTTL),
		HttpOnly: true,
		Secure:   h.secureCookie,
		SameSite: http.SameSiteLaxMode,
	})
}

func (h *Handler) clearStateCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     stateCookieName,
		Value:    "",
		Path:     "/auth/google",
		MaxAge:   -1,
		Expires:  time.Unix(1, 0),
		HttpOnly: true,
		Secure:   h.secureCookie,
		SameSite: http.SameSiteLaxMode,
	})
}

func (h *Handler) fail(w http.ResponseWriter, r *http.Request, reason, returnTo, stage string, err error) {
	fields := log.Fields{"stage": stage, "reason": reason}
	if err != nil {
		fields["error_type"] = fmt.Sprintf("%T", err)
	}
	log.WithFields(fields).Warn("Google OIDC login failed")
	h.sessions.IncLoginRequestCounter(loginFailure)
	h.redirectFailure(w, r, reason, returnTo)
}

func (h *Handler) redirectFailure(w http.ResponseWriter, r *http.Request, reason, returnTo string) {
	query := url.Values{"reason": []string{reason}}
	if returnTo != "" {
		query.Set("returnTo", ValidateReturnTo(returnTo))
	}
	http.Redirect(w, r, "/login?"+query.Encode(), http.StatusSeeOther)
}
