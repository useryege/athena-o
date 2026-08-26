package googleoidc

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
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
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/useryege/athena/internal/accountcredentials"
	httputil "github.com/useryege/athena/util/http"
	sessionmgr "github.com/useryege/athena/util/session"
)

const (
	stateCookieName        = "athena.google.state"
	registrationCookieName = "athena.google.registration"
	maxReturnToBytes       = 2048
	loginSuccess           = "success"
	loginFailure           = "failure"
)

type googleClaims struct {
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
}

type registrationView struct {
	Provider      string `json:"provider"`
	VerifiedEmail string `json:"verifiedEmail"`
	Administrator bool   `json:"administrator"`
	ExpiresAt     int64  `json:"expiresAt"`
	CSRFToken     string `json:"csrfToken"`
}

type registrationSubmission struct {
	Username  string `json:"username"`
	CSRFToken string `json:"csrfToken"`
}

type registrationAvailability struct {
	Status string `json:"status"`
}

type registrationRedirect struct {
	RedirectTo string `json:"redirectTo"`
}

type registrationError struct {
	Reason string `json:"reason"`
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
	adminEmail      string
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
	// Google documents both the HTTPS and bare-host issuer values. The library's
	// issuer comparison accepts only one exact string, so verify the supported
	// pair explicitly after signature, audience, and time validation below.
	verifier := oidc.NewVerifier("https://accounts.google.com", keySet, &oidc.Config{
		ClientID:        config.ClientID,
		SkipIssuerCheck: true,
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
		adminEmail:      config.AdminEmail,
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
	if err := idToken.Claims(&claims); err != nil || !claims.EmailVerified || strings.TrimSpace(claims.Email) == "" {
		h.fail(w, r, "google_not_allowed", returnTo, "email_verify", err)
		return
	}
	account, found, err := h.credentials.GetByGoogleSubject(r.Context(), idToken.Subject)
	if err != nil {
		h.fail(w, r, "google_unavailable", returnTo, "identity_lookup", err)
		return
	}
	if !found {
		ticketID, ticketErr := randomOpaqueValue()
		if ticketErr != nil {
			h.fail(w, r, "google_unavailable", returnTo, "registration_ticket_generation", ticketErr)
			return
		}
		csrfSecret, csrfErr := randomOpaqueValue()
		if csrfErr != nil {
			h.fail(w, r, "google_unavailable", returnTo, "registration_csrf_generation", csrfErr)
			return
		}
		createdAt := time.Now().UTC()
		if ticketErr := h.store.CreateRegistration(r.Context(), ticketID, registrationTicket{
			Subject:                idToken.Subject,
			VerifiedEmail:          strings.TrimSpace(claims.Email),
			AdministratorCandidate: strings.EqualFold(strings.TrimSpace(claims.Email), h.adminEmail),
			ReturnTo:               returnTo,
			CSRFSecret:             csrfSecret,
			CreatedAt:              createdAt,
		}); ticketErr != nil {
			h.fail(w, r, "google_unavailable", returnTo, "registration_ticket_create", ticketErr)
			return
		}
		h.setRegistrationCookie(w, ticketID, int(registrationTTL.Seconds()))
		h.clearStateCookie(w)
		log.WithField("stage", "registration_required").Info("Verified Google identity requires Athena username registration")
		http.Redirect(w, r, "/register", http.StatusSeeOther)
		return
	}
	h.completeGoogleLogin(w, r, account, idToken.Subject, claims.Email, returnTo)
}

func (h *Handler) completeGoogleLogin(w http.ResponseWriter, r *http.Request, account accountcredentials.Account, subject, verifiedEmail, returnTo string) bool {
	jti, err := uuid.NewRandom()
	if err != nil {
		h.fail(w, r, "google_unavailable", returnTo, "jti_generation", err)
		return false
	}
	athenaToken, err := h.sessions.CreateGoogleLogin(
		r.Context(),
		account.ID,
		subject,
		verifiedEmail,
		int64(h.sessionDuration.Seconds()),
		jti.String(),
	)
	if err != nil {
		if sessionmgr.IsAccountMaintenanceError(err) {
			h.fail(w, r, "maintenance", returnTo, "account_maintenance", err)
			return false
		}
		reason := "google_unavailable"
		if status.Code(err) == codes.PermissionDenied {
			reason = "google_not_allowed"
		}
		h.fail(w, r, reason, returnTo, "session_issue", err)
		return false
	}
	if err := httputil.SetTokenCookie(athenaToken, h.baseHRef, h.secureCookie, w); err != nil {
		h.fail(w, r, "google_unavailable", returnTo, "cookie_issue", err)
		return false
	}
	h.sessions.IncLoginRequestCounter(loginSuccess)
	log.WithFields(log.Fields{"stage": "complete", "account_id": account.ID}).Info("Google OIDC login succeeded")
	http.Redirect(w, r, returnTo, http.StatusSeeOther)
	return true
}

// Registration serves the anonymous username setup resource. A valid,
// browser-bound registration cookie is required for every method.
func (h *Handler) Registration(w http.ResponseWriter, r *http.Request) {
	setOAuthResponseHeaders(w)
	switch r.Method {
	case http.MethodGet:
		h.getRegistration(w, r)
	case http.MethodPost:
		h.createRegistration(w, r)
	case http.MethodDelete:
		h.deleteRegistration(w, r)
	default:
		w.Header().Set("Allow", "GET, POST, DELETE")
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// UsernameAvailability returns only the coarse state needed by the setup UI.
func (h *Handler) UsernameAvailability(w http.ResponseWriter, r *http.Request) {
	setOAuthResponseHeaders(w)
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	_, ticket, ok := h.registrationFromRequest(w, r)
	if !ok {
		return
	}
	available, err := h.credentials.UsernameAvailable(r.Context(), r.URL.Query().Get("username"), ticket.AdministratorCandidate)
	if errors.Is(err, accountcredentials.ErrUsernameInvalid) {
		h.writeJSON(w, http.StatusOK, registrationAvailability{Status: "invalid"})
		return
	}
	if err != nil {
		h.writeRegistrationError(w, http.StatusServiceUnavailable, "registration_unavailable")
		return
	}
	statusValue := "unavailable"
	if available {
		statusValue = "available"
	}
	h.writeJSON(w, http.StatusOK, registrationAvailability{Status: statusValue})
}

func (h *Handler) getRegistration(w http.ResponseWriter, r *http.Request) {
	_, ticket, ok := h.registrationFromRequest(w, r)
	if !ok {
		return
	}
	h.writeJSON(w, http.StatusOK, registrationView{
		Provider:      accountcredentials.IdentityProviderGoogle,
		VerifiedEmail: ticket.VerifiedEmail,
		Administrator: ticket.AdministratorCandidate,
		ExpiresAt:     ticket.CreatedAt.Add(registrationTTL).Unix(),
		CSRFToken:     ticket.CSRFSecret,
	})
}

func (h *Handler) createRegistration(w http.ResponseWriter, r *http.Request) {
	ticketID, ticket, ok := h.registrationFromRequest(w, r)
	if !ok {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	var input registrationSubmission
	if err := decoder.Decode(&input); err != nil {
		h.registrationFailed(w, http.StatusBadRequest, "username_invalid", "registration_decode", err)
		return
	}
	if !constantTimeEqual(input.CSRFToken, ticket.CSRFSecret) {
		h.registrationFailed(w, http.StatusForbidden, "registration_expired", "registration_csrf", nil)
		return
	}
	if err := accountcredentials.ValidateUsername(input.Username, ticket.AdministratorCandidate); err != nil {
		h.registrationFailed(w, http.StatusBadRequest, "username_invalid", "username_validate", err)
		return
	}
	claimID, err := randomOpaqueValue()
	if err != nil {
		h.registrationFailed(w, http.StatusServiceUnavailable, "registration_unavailable", "registration_claim_generation", err)
		return
	}
	claimedTicket, err := h.store.ClaimRegistration(r.Context(), ticketID, claimID)
	if err != nil {
		reason := "registration_unavailable"
		statusCode := http.StatusServiceUnavailable
		if errors.Is(err, errRegistrationNotFound) {
			reason = "registration_expired"
			statusCode = http.StatusUnauthorized
			h.clearRegistrationCookie(w)
		} else if errors.Is(err, errRegistrationInProgress) {
			statusCode = http.StatusConflict
		}
		h.registrationFailed(w, statusCode, reason, "registration_claim", err)
		return
	}
	if !constantTimeEqual(input.CSRFToken, claimedTicket.CSRFSecret) {
		_ = h.store.ReleaseRegistrationClaim(r.Context(), ticketID, claimID)
		h.registrationFailed(w, http.StatusForbidden, "registration_expired", "registration_claim_binding", nil)
		return
	}
	ticket = claimedTicket
	claimComplete := false
	defer func() {
		if !claimComplete {
			_ = h.store.ReleaseRegistrationClaim(context.Background(), ticketID, claimID)
		}
	}()
	registered, _, err := h.credentials.RegisterGoogleAccount(
		r.Context(),
		ticket.Subject,
		ticket.VerifiedEmail,
		input.Username,
		ticket.AdministratorCandidate,
	)
	if err != nil {
		switch {
		case errors.Is(err, accountcredentials.ErrUsernameInvalid):
			h.registrationFailed(w, http.StatusBadRequest, "username_invalid", "username_validate", err)
		case errors.Is(err, accountcredentials.ErrUsernameUnavailable):
			h.registrationFailed(w, http.StatusConflict, "username_unavailable", "username_conflict", err)
		case errors.Is(err, accountcredentials.ErrAdministratorIdentityConflict):
			h.registrationFailed(w, http.StatusForbidden, "google_not_allowed", "administrator_conflict", err)
		default:
			h.registrationFailed(w, http.StatusServiceUnavailable, "registration_unavailable", "account_create", err)
		}
		return
	}
	if err := h.sessions.RegisterCommittedAccountAccess(r.Context(), registered.ID); err != nil {
		h.registrationFailed(w, http.StatusServiceUnavailable, "registration_unavailable", "access_publish", err)
		return
	}
	if err := h.store.CompleteRegistration(r.Context(), ticketID, claimID); err != nil {
		reason := "registration_unavailable"
		statusCode := http.StatusServiceUnavailable
		if errors.Is(err, errRegistrationNotFound) {
			reason = "registration_expired"
			statusCode = http.StatusUnauthorized
			h.clearRegistrationCookie(w)
		} else if errors.Is(err, errRegistrationInProgress) {
			statusCode = http.StatusConflict
		}
		h.registrationFailed(w, statusCode, reason, "registration_consume", err)
		return
	}
	claimComplete = true
	h.clearRegistrationCookie(w)
	jti, err := uuid.NewRandom()
	if err != nil {
		h.registrationFailed(w, http.StatusServiceUnavailable, "registration_unavailable", "jti_generation", err)
		return
	}
	athenaToken, err := h.sessions.CreateGoogleLogin(
		r.Context(),
		registered.ID,
		ticket.Subject,
		ticket.VerifiedEmail,
		int64(h.sessionDuration.Seconds()),
		jti.String(),
	)
	if err != nil {
		if sessionmgr.IsAccountMaintenanceError(err) {
			h.registrationFailed(w, http.StatusServiceUnavailable, "maintenance", "account_maintenance", err)
			return
		}
		reason := "registration_unavailable"
		statusCode := http.StatusServiceUnavailable
		if status.Code(err) == codes.PermissionDenied {
			reason = "google_not_allowed"
			statusCode = http.StatusForbidden
		}
		h.registrationFailed(w, statusCode, reason, "session_issue", err)
		return
	}
	if err := httputil.SetTokenCookie(athenaToken, h.baseHRef, h.secureCookie, w); err != nil {
		h.registrationFailed(w, http.StatusServiceUnavailable, "registration_unavailable", "cookie_issue", err)
		return
	}
	h.sessions.IncLoginRequestCounter(loginSuccess)
	log.WithFields(log.Fields{"stage": "registration_complete", "account_id": registered.ID}).Info("Google OIDC registration succeeded")
	h.writeJSON(w, http.StatusOK, registrationRedirect{RedirectTo: defaultReturnTo})
}

func (h *Handler) deleteRegistration(w http.ResponseWriter, r *http.Request) {
	ticketID, ticket, ok := h.registrationFromRequest(w, r)
	if !ok {
		h.clearRegistrationCookie(w)
		return
	}
	if !constantTimeEqual(r.Header.Get("X-Athena-CSRF-Token"), ticket.CSRFSecret) {
		h.writeRegistrationError(w, http.StatusForbidden, "registration_expired")
		return
	}
	if err := h.store.DeleteRegistration(r.Context(), ticketID); err != nil {
		h.writeRegistrationError(w, http.StatusServiceUnavailable, "registration_unavailable")
		return
	}
	h.clearRegistrationCookie(w)
	query := url.Values{"returnTo": []string{ValidateReturnTo(ticket.ReturnTo)}}
	h.writeJSON(w, http.StatusOK, registrationRedirect{RedirectTo: "/auth/google/login?" + query.Encode()})
}

func (h *Handler) registrationFromRequest(w http.ResponseWriter, r *http.Request) (string, registrationTicket, bool) {
	cookie, err := r.Cookie(registrationCookieName)
	if err != nil || !validOpaqueValue(cookie.Value) {
		h.clearRegistrationCookie(w)
		h.writeRegistrationError(w, http.StatusUnauthorized, "registration_expired")
		return "", registrationTicket{}, false
	}
	ticket, err := h.store.GetRegistration(r.Context(), cookie.Value)
	if err != nil {
		reason := "registration_unavailable"
		statusCode := http.StatusServiceUnavailable
		if errors.Is(err, errRegistrationNotFound) {
			reason = "registration_expired"
			statusCode = http.StatusUnauthorized
			h.clearRegistrationCookie(w)
		} else if !errors.Is(err, errRegistrationUnavailable) {
			// A malformed server-side ticket cannot be recovered by this
			// browser, but it remains a dependency failure rather than an
			// authentication-policy denial.
			h.clearRegistrationCookie(w)
		}
		h.writeRegistrationError(w, statusCode, reason)
		return "", registrationTicket{}, false
	}
	return cookie.Value, ticket, true
}

func (h *Handler) registrationFailed(w http.ResponseWriter, statusCode int, reason, stage string, err error) {
	fields := log.Fields{"stage": stage, "reason": reason}
	if err != nil {
		fields["error_type"] = fmt.Sprintf("%T", err)
	}
	log.WithFields(fields).Warn("Google OIDC registration failed")
	h.sessions.IncLoginRequestCounter(loginFailure)
	h.writeRegistrationError(w, statusCode, reason)
}

func (h *Handler) writeRegistrationError(w http.ResponseWriter, statusCode int, reason string) {
	h.writeJSON(w, statusCode, registrationError{Reason: reason})
}

func (h *Handler) writeJSON(w http.ResponseWriter, statusCode int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.WithError(err).Warn("Google OIDC response encoding failed")
	}
}

// ValidateReturnTo accepts only same-origin absolute paths and supplies the
// canonical pending-access fallback for malformed or looping values.
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
	if parsed.Path == "/login" || strings.HasPrefix(parsed.Path, "/login/") || parsed.Path == "/register" || strings.HasPrefix(parsed.Path, "/register/") {
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

func (h *Handler) setRegistrationCookie(w http.ResponseWriter, value string, maxAge int) {
	http.SetCookie(w, &http.Cookie{
		Name:     registrationCookieName,
		Value:    value,
		Path:     "/auth/google/registration",
		MaxAge:   maxAge,
		Expires:  time.Now().Add(registrationTTL),
		HttpOnly: true,
		Secure:   h.secureCookie,
		SameSite: http.SameSiteStrictMode,
	})
}

func (h *Handler) clearRegistrationCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     registrationCookieName,
		Value:    "",
		Path:     "/auth/google/registration",
		MaxAge:   -1,
		Expires:  time.Unix(1, 0),
		HttpOnly: true,
		Secure:   h.secureCookie,
		SameSite: http.SameSiteStrictMode,
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
