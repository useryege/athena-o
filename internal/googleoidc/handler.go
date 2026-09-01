package googleoidc

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/redis/go-redis/v9"
	log "github.com/sirupsen/logrus"
	"golang.org/x/oauth2"

	"github.com/useryege/athena/common"
	"github.com/useryege/athena/internal/accountcredentials"
	"github.com/useryege/athena/internal/authregistration"
)

const (
	stateCookieNamePrefix = "athena.google.state."
	entryCookieName       = "athena.google.entry"
)

type googleClaims struct {
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	AuthTime      int64  `json:"auth_time"`
}

type verifiedGoogleIdentity struct {
	identity authregistration.Identity
}

type googleVerificationError struct {
	reason string
	stage  string
	err    error
}

// Handler implements the browser Google Authorization Code + PKCE flow.
type Handler struct {
	oauth2Config         oauth2.Config
	verifier             *oidc.IDTokenVerifier
	store                *TransactionStore
	backend              authregistration.Backend
	registrations        *authregistration.Handler
	secureCookie         bool
	publicOrigin         string
	baseHRef             string
	adminEmail           string
	walletSecrets        *walletSecretReauthentication
	wormCredentials      *wormCredentialReauthentication
	wormExecutions       *wormExecutionAuthorization
	wormPositionCashOuts *wormPositionCashOutAuthorization
}

// NewHandler constructs the flow without contacting Google. Remote JWKS are
// fetched and cached only when a callback first verifies an ID token.
func NewHandler(
	config Config,
	redisClient *redis.Client,
	backend authregistration.Backend,
	registrations *authregistration.Handler,
	baseHRef string,
) (*Handler, error) {
	store, err := NewTransactionStore(redisClient)
	if err != nil {
		return nil, err
	}
	if backend == nil || registrations == nil {
		return nil, fmt.Errorf("Google OIDC authentication backend and registration handler are required")
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
		verifier:      verifier,
		store:         store,
		backend:       backend,
		registrations: registrations,
		secureCookie:  config.secureCookie,
		publicOrigin:  config.PublicOrigin(),
		baseHRef:      baseHRef,
		adminEmail:    config.AdminEmail,
	}, nil
}

// Login begins one Google authorization transaction.
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	authregistration.SetResponseHeaders(w)
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	realm, err := applicationRealmFromQuery(r)
	if err != nil {
		h.backend.RecordLoginResult(authregistration.LoginFailure)
		http.Error(w, "application realm is required", http.StatusBadRequest)
		return
	}
	returnTo := authregistration.ReturnToForRealm(r.URL.Query().Get("returnTo"), realm)
	state, err := authregistration.RandomOpaqueValue()
	if err != nil {
		h.fail(w, r, "google_unavailable", realm, returnTo, "state_generation", err)
		return
	}
	nonce, err := authregistration.RandomOpaqueValue()
	if err != nil {
		h.fail(w, r, "google_unavailable", realm, returnTo, "nonce_generation", err)
		return
	}
	verifier := oauth2.GenerateVerifier()
	if err := h.store.Create(r.Context(), state, authregistration.ClientRateIdentity(r), transaction{
		Nonce:     nonce,
		Verifier:  verifier,
		ReturnTo:  returnTo,
		Realm:     realm,
		CreatedAt: time.Now().UTC(),
	}); err != nil {
		h.fail(w, r, "google_unavailable", realm, returnTo, "transaction_create", err)
		return
	}
	h.setStateCookies(w, state, realm)
	authorizationURL := h.oauth2Config.AuthCodeURL(
		state,
		oauth2.S256ChallengeOption(verifier),
		oauth2.SetAuthURLParam("nonce", nonce),
		oauth2.SetAuthURLParam("prompt", "select_account"),
	)
	http.Redirect(w, r, authorizationURL, http.StatusSeeOther)
}

// Callback consumes the transaction, verifies Google identity, and either
// begins shared username registration or issues an Athena-only browser session.
func (h *Handler) Callback(w http.ResponseWriter, r *http.Request) {
	if h.wormPositionCashOuts != nil && h.wormPositionCashOuts.ownsCallback(r) {
		h.wormPositionCashOuts.callback(w, r)
		return
	}
	if h.wormExecutions != nil && h.wormExecutions.ownsCallback(r) {
		h.wormExecutions.callback(w, r)
		return
	}
	if h.wormCredentials != nil && h.wormCredentials.ownsCallback(r) {
		h.wormCredentials.callback(w, r)
		return
	}
	if h.walletSecrets != nil && h.walletSecrets.ownsCallback(r) {
		h.walletSecrets.callback(w, r)
		return
	}
	authregistration.SetResponseHeaders(w)
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	state := r.URL.Query().Get("state")
	fallbackRealm, stateCookieMatched := h.primaryCallbackFallbackRealm(r, state)
	if !authregistration.ValidOpaqueValue(state) {
		h.fail(w, r, "google_state_invalid", fallbackRealm, "", "state_missing", nil)
		return
	}
	transaction, err := h.store.Consume(r.Context(), state)
	if err != nil {
		if stateCookieMatched {
			h.clearPrimaryStateCookie(w, fallbackRealm)
		}
		reason := "google_state_invalid"
		if errors.Is(err, errTransactionUnavailable) {
			reason = "google_unavailable"
		}
		h.fail(w, r, reason, fallbackRealm, "", "transaction_consume", err)
		return
	}
	realm := transaction.Realm
	returnTo := authregistration.ReturnToForRealm(transaction.ReturnTo, realm)
	cookie, cookieErr := r.Cookie(primaryStateCookieName(realm))
	h.clearPrimaryStateCookies(w, realm)
	if cookieErr != nil || !authregistration.ConstantTimeEqual(cookie.Value, state) || !transactionFresh(transaction.CreatedAt) {
		h.fail(w, r, "google_state_invalid", realm, returnTo, "state_binding", cookieErr)
		return
	}
	if r.URL.Query().Get("error") == "access_denied" {
		h.fail(w, r, "google_cancelled", realm, returnTo, "authorization_cancelled", nil)
		return
	}
	if r.URL.Query().Get("error") != "" || r.URL.Query().Get("code") == "" {
		h.fail(w, r, "google_unavailable", realm, returnTo, "authorization_response", nil)
		return
	}

	verified, verificationErr := h.exchangeAndVerify(r.Context(), r.URL.Query().Get("code"), transaction.Verifier, transaction.Nonce, time.Time{})
	if verificationErr != nil {
		h.fail(w, r, verificationErr.reason, realm, returnTo, verificationErr.stage, verificationErr.err)
		return
	}
	identity := verified.identity
	identity.Realm = realm
	account, found, err := h.backend.GetByIdentity(r.Context(), identity)
	if err != nil {
		h.fail(w, r, "google_unavailable", realm, returnTo, "identity_lookup", err)
		return
	}
	if !found {
		if realm == accountcredentials.ApplicationRealmAdmin && !strings.EqualFold(identity.VerifiedEmail, h.adminEmail) {
			h.fail(w, r, "google_not_allowed", realm, returnTo, "administrator_identity_rejected", nil)
			return
		}
		if err := h.registrations.Begin(r.Context(), w, identity, returnTo); err != nil {
			h.fail(w, r, "google_unavailable", realm, returnTo, "registration_ticket_create", err)
			return
		}
		log.WithField("stage", "registration_required").Info("Verified Google identity requires Athena username registration")
		registrationQuery := url.Values{common.ApplicationRealmQueryParameter: []string{string(realm)}}
		http.Redirect(w, r, h.deploymentPath("/register?"+registrationQuery.Encode()), http.StatusSeeOther)
		return
	}
	h.completeLogin(w, r, account, identity, returnTo)
}

// exchangeAndVerify is the shared Google code-exchange and OIDC verification
// primitive used by both primary login and wallet-secret reauthentication.
// A non-zero reauthenticatedAfter additionally requires a fresh auth_time.
func (h *Handler) exchangeAndVerify(ctx context.Context, code, verifier, nonce string, reauthenticatedAfter time.Time) (verifiedGoogleIdentity, *googleVerificationError) {
	networkContext, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	googleToken, err := h.oauth2Config.Exchange(networkContext, code, oauth2.VerifierOption(verifier))
	if err != nil {
		return verifiedGoogleIdentity{}, &googleVerificationError{reason: "google_unavailable", stage: "code_exchange", err: err}
	}
	rawIDToken, ok := googleToken.Extra("id_token").(string)
	if !ok || rawIDToken == "" {
		return verifiedGoogleIdentity{}, &googleVerificationError{reason: "google_unavailable", stage: "id_token_missing"}
	}
	idToken, err := h.verifier.Verify(networkContext, rawIDToken)
	if err != nil {
		return verifiedGoogleIdentity{}, &googleVerificationError{reason: "google_unavailable", stage: "id_token_verify", err: err}
	}
	if idToken.Issuer != "https://accounts.google.com" && idToken.Issuer != "accounts.google.com" {
		return verifiedGoogleIdentity{}, &googleVerificationError{reason: "google_not_allowed", stage: "issuer_verify"}
	}
	now := time.Now().UTC()
	if idToken.IssuedAt.IsZero() || idToken.IssuedAt.After(now.Add(time.Minute)) || idToken.Expiry.IsZero() {
		return verifiedGoogleIdentity{}, &googleVerificationError{reason: "google_not_allowed", stage: "token_time_verify"}
	}
	if idToken.Subject == "" || !authregistration.ConstantTimeEqual(idToken.Nonce, nonce) {
		return verifiedGoogleIdentity{}, &googleVerificationError{reason: "google_not_allowed", stage: "identity_claims"}
	}
	var claims googleClaims
	if err := idToken.Claims(&claims); err != nil || !claims.EmailVerified || strings.TrimSpace(claims.Email) == "" {
		return verifiedGoogleIdentity{}, &googleVerificationError{reason: "google_not_allowed", stage: "email_verify", err: err}
	}
	authTime := time.Time{}
	if !reauthenticatedAfter.IsZero() {
		if claims.AuthTime <= 0 {
			return verifiedGoogleIdentity{}, &googleVerificationError{reason: "google_not_allowed", stage: "auth_time_missing"}
		}
		authTime = time.Unix(claims.AuthTime, 0).UTC()
		if authTime.After(now.Add(time.Minute)) || authTime.Before(reauthenticatedAfter.Add(-time.Minute)) {
			return verifiedGoogleIdentity{}, &googleVerificationError{reason: "google_not_allowed", stage: "auth_time_stale"}
		}
	}
	return verifiedGoogleIdentity{
		identity: authregistration.Identity{
			Provider:      accountcredentials.IdentityProviderGoogle,
			Subject:       idToken.Subject,
			VerifiedEmail: strings.TrimSpace(claims.Email),
		},
	}, nil
}

func (h *Handler) completeLogin(w http.ResponseWriter, r *http.Request, account authregistration.Account, identity authregistration.Identity, returnTo string) {
	token, err := h.backend.CreateExternalLogin(r.Context(), account.ID, identity)
	if err != nil {
		if errors.Is(err, authregistration.ErrMaintenance) {
			h.fail(w, r, "maintenance", identity.Realm, returnTo, "account_maintenance", err)
			return
		}
		reason := "google_unavailable"
		if errors.Is(err, authregistration.ErrIdentityNotAllowed) {
			reason = "google_not_allowed"
		}
		h.fail(w, r, reason, identity.Realm, returnTo, "session_issue", err)
		return
	}
	h.registrations.ClearCookie(w)
	if err := h.registrations.SetAthenaSessionCookie(w, identity.Realm, token); err != nil {
		h.fail(w, r, "google_unavailable", identity.Realm, returnTo, "cookie_issue", err)
		return
	}
	h.backend.RecordLoginResult(authregistration.LoginSuccess)
	log.WithFields(log.Fields{"stage": "complete", "provider": accountcredentials.IdentityProviderGoogle, "realm": identity.Realm, "account_id": account.ID}).Info("Google OIDC login succeeded")
	http.Redirect(w, r, h.deploymentPath(authregistration.ReturnToForRealm(returnTo, identity.Realm)), http.StatusSeeOther)
}

func transactionFresh(createdAt time.Time) bool {
	now := time.Now().UTC()
	return !createdAt.After(now.Add(time.Minute)) && now.Sub(createdAt) <= transactionTTL
}

func primaryStateCookieName(realm accountcredentials.ApplicationRealm) string {
	return stateCookieNamePrefix + string(realm)
}

func (h *Handler) primaryCallbackFallbackRealm(r *http.Request, state string) (accountcredentials.ApplicationRealm, bool) {
	if authregistration.ValidOpaqueValue(state) {
		matchedRealm := accountcredentials.ApplicationRealm("")
		for _, realm := range []accountcredentials.ApplicationRealm{accountcredentials.ApplicationRealmMember, accountcredentials.ApplicationRealmAdmin} {
			cookie, err := r.Cookie(primaryStateCookieName(realm))
			if err != nil || !authregistration.ConstantTimeEqual(cookie.Value, state) {
				continue
			}
			if matchedRealm != "" {
				matchedRealm = ""
				break
			}
			matchedRealm = realm
		}
		if matchedRealm != "" {
			return matchedRealm, true
		}
	}
	entryCookie, _ := r.Cookie(entryCookieName)
	if entryCookie != nil {
		if realm, err := accountcredentials.ParseApplicationRealm(entryCookie.Value); err == nil {
			return realm, false
		}
	}
	return accountcredentials.ApplicationRealmMember, false
}

// Scoped Google callbacks are full-page provider redirects and therefore
// cannot preserve the request header that began the member-only proof flow.
// Their one-time server transaction already binds the account and Session, so
// restore the fixed member realm before the normal cookie authenticator runs.
func bindMemberApplicationRealm(r *http.Request) {
	r.Header.Set(common.ApplicationRealmHeader, string(accountcredentials.ApplicationRealmMember))
}

func bindMemberApplicationRealmFromQuery(r *http.Request) error {
	realm, err := applicationRealmFromQuery(r)
	if err != nil || realm != accountcredentials.ApplicationRealmMember {
		return fmt.Errorf("member application realm is invalid")
	}
	r.Header.Set(common.ApplicationRealmHeader, string(realm))
	query := r.URL.Query()
	query.Del(common.ApplicationRealmQueryParameter)
	adaptedURL := *r.URL
	adaptedURL.RawQuery = query.Encode()
	r.URL = &adaptedURL
	return nil
}

func applicationRealmFromQuery(r *http.Request) (accountcredentials.ApplicationRealm, error) {
	values := r.URL.Query()[common.ApplicationRealmQueryParameter]
	if len(values) != 1 {
		return "", fmt.Errorf("application realm query is required exactly once")
	}
	realm, err := accountcredentials.ParseApplicationRealm(values[0])
	if err != nil {
		return "", fmt.Errorf("application realm query is invalid")
	}
	if headerValues := r.Header.Values(common.ApplicationRealmHeader); len(headerValues) > 0 {
		if len(headerValues) != 1 || headerValues[0] != string(realm) {
			return "", fmt.Errorf("application realm transports do not match")
		}
	}
	return realm, nil
}

func (h *Handler) setStateCookies(w http.ResponseWriter, value string, realm accountcredentials.ApplicationRealm) {
	http.SetCookie(w, &http.Cookie{
		Name:     primaryStateCookieName(realm),
		Value:    value,
		Path:     h.deploymentPath("/auth/google"),
		MaxAge:   int(transactionTTL.Seconds()),
		Expires:  time.Now().Add(transactionTTL),
		HttpOnly: true,
		Secure:   h.secureCookie,
		SameSite: http.SameSiteLaxMode,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     entryCookieName,
		Value:    string(realm),
		Path:     h.deploymentPath("/auth/google"),
		MaxAge:   int(transactionTTL.Seconds()),
		Expires:  time.Now().Add(transactionTTL),
		HttpOnly: true,
		Secure:   h.secureCookie,
		SameSite: http.SameSiteLaxMode,
	})
}

func (h *Handler) clearPrimaryStateCookies(w http.ResponseWriter, realm accountcredentials.ApplicationRealm) {
	h.clearPrimaryStateCookie(w, realm)
	h.clearEntryCookie(w)
}

func (h *Handler) clearPrimaryStateCookie(w http.ResponseWriter, realm accountcredentials.ApplicationRealm) {
	http.SetCookie(w, &http.Cookie{
		Name:     primaryStateCookieName(realm),
		Value:    "",
		Path:     h.deploymentPath("/auth/google"),
		MaxAge:   -1,
		Expires:  time.Unix(1, 0),
		HttpOnly: true,
		Secure:   h.secureCookie,
		SameSite: http.SameSiteLaxMode,
	})
}

func (h *Handler) clearEntryCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     entryCookieName,
		Value:    "",
		Path:     h.deploymentPath("/auth/google"),
		MaxAge:   -1,
		Expires:  time.Unix(1, 0),
		HttpOnly: true,
		Secure:   h.secureCookie,
		SameSite: http.SameSiteLaxMode,
	})
}

func (h *Handler) fail(w http.ResponseWriter, r *http.Request, reason string, realm accountcredentials.ApplicationRealm, returnTo, stage string, err error) {
	fields := log.Fields{"stage": stage, "reason": reason, "provider": accountcredentials.IdentityProviderGoogle, "realm": realm}
	if err != nil {
		fields["error_type"] = fmt.Sprintf("%T", err)
	}
	log.WithFields(fields).Warn("Google OIDC login failed")
	h.backend.RecordLoginResult(authregistration.LoginFailure)
	query := url.Values{"reason": []string{reason}}
	if returnTo != "" {
		query.Set("returnTo", authregistration.ReturnToForRealm(returnTo, realm))
	}
	loginPath := "/login"
	if realm == accountcredentials.ApplicationRealmAdmin {
		loginPath = "/admin/login"
	}
	http.Redirect(w, r, h.deploymentPath(loginPath)+"?"+query.Encode(), http.StatusSeeOther)
}

func (h *Handler) deploymentPath(logicalPath string) string {
	return authregistration.DeploymentPath(h.baseHRef, logicalPath)
}
