package googleoidc

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/redis/go-redis/v9"
	log "github.com/sirupsen/logrus"
	"golang.org/x/oauth2"

	"github.com/useryege/athena/internal/accountcredentials"
	"github.com/useryege/athena/internal/authregistration"
	"github.com/useryege/athena/internal/walletsecret"
)

const (
	wormCredentialStateCookieName = "athena.worm-trading.google.state"
	wormCredentialDefaultReturnTo = "/worm-trading"
)

// WormCredentialAuthenticator validates only Athena's login cookie, attaches
// its typed credential to the returned context, and enforces Worm Trading
// READ_WRITE.
type WormCredentialAuthenticator func(request *http.Request) (context.Context, accountcredentials.AuthenticatedCredential, error)

type wormCredentialReauthentication struct {
	google       *Handler
	store        *wormCredentialTransactionStore
	authenticate WormCredentialAuthenticator
	admit        func(context.Context) error
	credentials  *accountcredentials.CredentialManager
	leases       *walletsecret.Manager
}

// EnableWormCredentialReauthentication attaches the independent Google proof
// used only to authorize Worm API credential management.
func (h *Handler) EnableWormCredentialReauthentication(
	redisClient *redis.Client,
	authenticate WormCredentialAuthenticator,
	admit func(context.Context) error,
	credentials *accountcredentials.CredentialManager,
	leases *walletsecret.Manager,
) error {
	if h == nil || authenticate == nil || admit == nil || credentials == nil || leases == nil {
		return fmt.Errorf("Google Worm credential reauthentication dependencies are required")
	}
	store, err := newWormCredentialTransactionStore(redisClient)
	if err != nil {
		return err
	}
	h.wormCredentials = &wormCredentialReauthentication{
		google:       h,
		store:        store,
		authenticate: authenticate,
		admit:        admit,
		credentials:  credentials,
		leases:       leases,
	}
	return nil
}

// WormCredentialReauthentication begins a fresh Google authorization flow
// bound to the current login session and authorization revision.
func (h *Handler) WormCredentialReauthentication(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.wormCredentials == nil {
		walletsecret.WriteError(w, walletsecret.ErrWormReauthenticationUnavailable)
		return
	}
	h.wormCredentials.begin(w, r)
}

func (h *wormCredentialReauthentication) begin(w http.ResponseWriter, r *http.Request) {
	walletsecret.SetSecretResponseHeaders(w)
	returnTo := validateWormCredentialReturnTo(r.URL.Query().Get("returnTo"))
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if err := bindMemberApplicationRealmFromQuery(r); err != nil {
		h.redirectFailure(w, r, returnTo, walletsecret.WormLoginSessionRequiredReason, "application_realm")
		return
	}
	authCtx, credential, err := h.authenticate(r)
	if err != nil || credential.Capability != accountcredentials.CapabilityLogin {
		h.redirectFailure(w, r, returnTo, walletsecret.WormLoginSessionRequiredReason, "login_session")
		return
	}
	if err := h.admit(authCtx); err != nil {
		reason := walletsecret.Reason(err)
		if reason == "" {
			reason = walletsecret.WormReauthenticationUnavailableReason
		}
		h.redirectFailure(w, r, returnTo, reason, "module_access")
		return
	}
	account, err := h.credentials.Get(credential.AccountID)
	if err != nil || account.IdentityProvider != accountcredentials.IdentityProviderGoogle || !account.HasExternalIdentity() {
		h.redirectFailure(w, r, returnTo, walletsecret.WormReauthenticationRequiredReason, "identity_provider")
		return
	}
	state, err := newWormCredentialState()
	if err != nil {
		h.redirectFailure(w, r, returnTo, walletsecret.WormReauthenticationUnavailableReason, "state_generation")
		return
	}
	nonce, err := authregistration.RandomOpaqueValue()
	if err != nil {
		h.redirectFailure(w, r, returnTo, walletsecret.WormReauthenticationUnavailableReason, "nonce_generation")
		return
	}
	verifier := oauth2.GenerateVerifier()
	createdAt := time.Now().UTC()
	if err := h.store.create(r.Context(), state, wormCredentialTransaction{
		Nonce:            nonce,
		Verifier:         verifier,
		ReturnTo:         returnTo,
		AccountID:        credential.AccountID,
		SessionJTIDigest: walletsecret.SessionJTIDigest(credential.JTI),
		AccessRevision:   credential.AccessRevision,
		CreatedAt:        createdAt,
	}); err != nil {
		h.redirectFailure(w, r, returnTo, walletsecret.WormReauthenticationUnavailableReason, "transaction_create")
		return
	}
	h.setStateCookie(w, state)
	authorizationURL := h.google.oauth2Config.AuthCodeURL(
		state,
		oauth2.S256ChallengeOption(verifier),
		oauth2.SetAuthURLParam("nonce", nonce),
		oauth2.SetAuthURLParam("prompt", "select_account"),
		oauth2.SetAuthURLParam("max_age", "0"),
	)
	http.Redirect(w, r, authorizationURL, http.StatusSeeOther)
}

func (h *wormCredentialReauthentication) ownsCallback(r *http.Request) bool {
	return validWormCredentialState(r.URL.Query().Get("state"))
}

func (h *wormCredentialReauthentication) callback(w http.ResponseWriter, r *http.Request) {
	walletsecret.SetSecretResponseHeaders(w)
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	bindMemberApplicationRealm(r)
	state := r.URL.Query().Get("state")
	cookie, cookieErr := r.Cookie(wormCredentialStateCookieName)
	h.clearStateCookie(w)
	if !validWormCredentialState(state) {
		h.redirectFailure(w, r, wormCredentialDefaultReturnTo, walletsecret.WormReauthenticationRequiredReason, "state_missing")
		return
	}
	transaction, err := h.store.consume(r.Context(), state)
	if err != nil {
		reason := walletsecret.WormReauthenticationRequiredReason
		if errors.Is(err, errWormCredentialTransactionUnavailable) {
			reason = walletsecret.WormReauthenticationUnavailableReason
		}
		h.redirectFailure(w, r, wormCredentialDefaultReturnTo, reason, "transaction_consume")
		return
	}
	returnTo := validateWormCredentialReturnTo(transaction.ReturnTo)
	if cookieErr != nil || !authregistration.ConstantTimeEqual(cookie.Value, state) {
		h.redirectFailure(w, r, returnTo, walletsecret.WormReauthenticationRequiredReason, "state_binding")
		return
	}
	authCtx, credential, err := h.authenticate(r)
	if err != nil || credential.Capability != accountcredentials.CapabilityLogin {
		h.redirectFailure(w, r, returnTo, walletsecret.WormLoginSessionRequiredReason, "login_session")
		return
	}
	if !authregistration.ConstantTimeEqual(transaction.AccountID, credential.AccountID) ||
		!authregistration.ConstantTimeEqual(transaction.SessionJTIDigest, walletsecret.SessionJTIDigest(credential.JTI)) ||
		transaction.AccessRevision != credential.AccessRevision {
		h.redirectFailure(w, r, returnTo, walletsecret.WormReauthenticationRequiredReason, "session_binding")
		return
	}
	if err := h.admit(authCtx); err != nil {
		reason := walletsecret.Reason(err)
		if reason == "" {
			reason = walletsecret.WormReauthenticationUnavailableReason
		}
		h.redirectFailure(w, r, returnTo, reason, "module_access")
		return
	}
	if r.URL.Query().Get("error") != "" || r.URL.Query().Get("code") == "" {
		h.redirectFailure(w, r, returnTo, walletsecret.WormReauthenticationRequiredReason, "authorization_response")
		return
	}
	verified, verificationErr := h.google.exchangeAndVerify(r.Context(), r.URL.Query().Get("code"), transaction.Verifier, transaction.Nonce, transaction.CreatedAt)
	if verificationErr != nil {
		reason := walletsecret.WormReauthenticationUnavailableReason
		if verificationErr.reason == "google_not_allowed" {
			reason = walletsecret.WormReauthenticationRequiredReason
		}
		h.redirectFailure(w, r, returnTo, reason, verificationErr.stage)
		return
	}
	account, err := h.credentials.Get(credential.AccountID)
	if err != nil || account.IdentityProvider != accountcredentials.IdentityProviderGoogle ||
		!authregistration.ConstantTimeEqual(account.IdentitySubject, verified.identity.Subject) {
		h.redirectFailure(w, r, returnTo, walletsecret.WormReauthenticationRequiredReason, "identity_binding")
		return
	}
	if _, err := h.leases.Issue(r.Context(), w, credential); err != nil {
		reason := walletsecret.WormReauthenticationRequiredReason
		if walletsecret.IsUnavailable(err) {
			reason = walletsecret.WormReauthenticationUnavailableReason
		}
		h.redirectFailure(w, r, returnTo, reason, "lease_issue")
		return
	}
	log.WithFields(log.Fields{"stage": "complete", "provider": accountcredentials.IdentityProviderGoogle}).Info("Worm credential Google reauthentication succeeded")
	observeGoogleAuthorization(r.Context(), "google", "authorization_verified", "GOOGLE", "account", credential.AccountID, "WORM_CONNECTION_AUTHORIZE", int64(transaction.AccessRevision), int64(credential.AccessRevision), false)
	http.Redirect(w, r, h.google.deploymentPath(returnTo), http.StatusSeeOther)
}

func (h *wormCredentialReauthentication) setStateCookie(w http.ResponseWriter, value string) {
	http.SetCookie(w, &http.Cookie{
		Name:     wormCredentialStateCookieName,
		Value:    value,
		Path:     h.google.deploymentPath("/auth/google"),
		MaxAge:   int(wormCredentialTransactionTTL / time.Second),
		Expires:  time.Now().Add(wormCredentialTransactionTTL),
		HttpOnly: true,
		Secure:   h.google.secureCookie,
		SameSite: http.SameSiteLaxMode,
	})
}

func (h *wormCredentialReauthentication) clearStateCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     wormCredentialStateCookieName,
		Value:    "",
		Path:     h.google.deploymentPath("/auth/google"),
		MaxAge:   -1,
		Expires:  time.Unix(1, 0),
		HttpOnly: true,
		Secure:   h.google.secureCookie,
		SameSite: http.SameSiteLaxMode,
	})
}

func (h *wormCredentialReauthentication) redirectFailure(w http.ResponseWriter, r *http.Request, returnTo, reason, stage string) {
	failGoogleAuthorization(r.Context(), reason, stage)
	log.WithFields(log.Fields{"stage": stage, "reason": reason, "provider": accountcredentials.IdentityProviderGoogle}).Warn("Worm credential Google reauthentication failed")
	http.Redirect(w, r, h.google.deploymentPath(wormCredentialFailureURL(returnTo, reason)), http.StatusSeeOther)
}

func validateWormCredentialReturnTo(raw string) string {
	validated := authregistration.ReturnToForRealm(raw, accountcredentials.ApplicationRealmMember)
	if raw == "" || validated == authregistration.DefaultReturnTo {
		return wormCredentialDefaultReturnTo
	}
	return validated
}

func wormCredentialFailureURL(returnTo, reason string) string {
	parsed, err := url.Parse(validateWormCredentialReturnTo(returnTo))
	if err != nil {
		return wormCredentialDefaultReturnTo + "?wormTradingReason=" + url.QueryEscape(reason)
	}
	query := parsed.Query()
	query.Set("wormTradingReason", reason)
	parsed.RawQuery = query.Encode()
	return parsed.String()
}
