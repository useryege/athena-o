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
	walletSecretStateCookieName = "athena.wallet-secret.google.state"
	walletSecretDefaultReturnTo = "/wallet"
)

// WalletSecretAuthenticator validates only Athena's login cookie, attaches its
// typed credential to the returned context, and enforces Wallet READ_WRITE.
type WalletSecretAuthenticator func(request *http.Request) (context.Context, accountcredentials.AuthenticatedCredential, error)

type walletSecretReauthentication struct {
	google       *Handler
	store        *walletSecretTransactionStore
	authenticate WalletSecretAuthenticator
	credentials  *accountcredentials.CredentialManager
	leases       *walletsecret.Manager
}

// EnableWalletSecretReauthentication attaches the independent Google
// reauthentication state machine while retaining the existing callback URI.
func (h *Handler) EnableWalletSecretReauthentication(
	redisClient *redis.Client,
	authenticate WalletSecretAuthenticator,
	credentials *accountcredentials.CredentialManager,
	leases *walletsecret.Manager,
) error {
	if h == nil || authenticate == nil || credentials == nil || leases == nil {
		return fmt.Errorf("Google wallet-secret reauthentication dependencies are required")
	}
	store, err := newWalletSecretTransactionStore(redisClient)
	if err != nil {
		return err
	}
	h.walletSecrets = &walletSecretReauthentication{
		google:       h,
		store:        store,
		authenticate: authenticate,
		credentials:  credentials,
		leases:       leases,
	}
	return nil
}

// WalletSecretReauthentication begins a fresh Google authorization code flow
// bound to the current Athena login session and access revision.
func (h *Handler) WalletSecretReauthentication(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.walletSecrets == nil {
		walletsecret.WriteError(w, walletsecret.ErrReauthenticationUnavailable)
		return
	}
	h.walletSecrets.begin(w, r)
}

func (h *walletSecretReauthentication) begin(w http.ResponseWriter, r *http.Request) {
	walletsecret.SetSecretResponseHeaders(w)
	returnTo := validateWalletSecretReturnTo(r.URL.Query().Get("returnTo"))
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	_, credential, err := h.authenticate(r)
	if err != nil || credential.Capability != accountcredentials.CapabilityLogin {
		h.redirectFailure(w, r, returnTo, walletsecret.LoginSessionRequiredReason, "login_session")
		return
	}
	account, err := h.credentials.Get(credential.AccountID)
	if err != nil || account.IdentityProvider != accountcredentials.IdentityProviderGoogle || !account.HasExternalIdentity() {
		h.redirectFailure(w, r, returnTo, walletsecret.ReauthenticationRequiredReason, "identity_provider")
		return
	}
	state, err := newWalletSecretState()
	if err != nil {
		h.redirectFailure(w, r, returnTo, walletsecret.ReauthenticationUnavailableReason, "state_generation")
		return
	}
	nonce, err := authregistration.RandomOpaqueValue()
	if err != nil {
		h.redirectFailure(w, r, returnTo, walletsecret.ReauthenticationUnavailableReason, "nonce_generation")
		return
	}
	verifier := oauth2.GenerateVerifier()
	createdAt := time.Now().UTC()
	if err := h.store.create(r.Context(), state, walletSecretTransaction{
		Nonce:            nonce,
		Verifier:         verifier,
		ReturnTo:         returnTo,
		AccountID:        credential.AccountID,
		SessionJTIDigest: walletsecret.SessionJTIDigest(credential.JTI),
		AccessRevision:   credential.AccessRevision,
		CreatedAt:        createdAt,
	}); err != nil {
		h.redirectFailure(w, r, returnTo, walletsecret.ReauthenticationUnavailableReason, "transaction_create")
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

func (h *walletSecretReauthentication) ownsCallback(r *http.Request) bool {
	return validWalletSecretState(r.URL.Query().Get("state"))
}

func (h *walletSecretReauthentication) callback(w http.ResponseWriter, r *http.Request) {
	walletsecret.SetSecretResponseHeaders(w)
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	state := r.URL.Query().Get("state")
	cookie, cookieErr := r.Cookie(walletSecretStateCookieName)
	h.clearStateCookie(w)
	if !validWalletSecretState(state) {
		h.redirectFailure(w, r, walletSecretDefaultReturnTo, walletsecret.ReauthenticationRequiredReason, "state_missing")
		return
	}
	transaction, err := h.store.consume(r.Context(), state)
	if err != nil {
		reason := walletsecret.ReauthenticationRequiredReason
		if errors.Is(err, errWalletSecretTransactionUnavailable) {
			reason = walletsecret.ReauthenticationUnavailableReason
		}
		h.redirectFailure(w, r, walletSecretDefaultReturnTo, reason, "transaction_consume")
		return
	}
	returnTo := validateWalletSecretReturnTo(transaction.ReturnTo)
	if cookieErr != nil || !authregistration.ConstantTimeEqual(cookie.Value, state) {
		h.redirectFailure(w, r, returnTo, walletsecret.ReauthenticationRequiredReason, "state_binding")
		return
	}
	_, credential, err := h.authenticate(r)
	if err != nil || credential.Capability != accountcredentials.CapabilityLogin {
		h.redirectFailure(w, r, returnTo, walletsecret.LoginSessionRequiredReason, "login_session")
		return
	}
	if !authregistration.ConstantTimeEqual(transaction.AccountID, credential.AccountID) ||
		!authregistration.ConstantTimeEqual(transaction.SessionJTIDigest, walletsecret.SessionJTIDigest(credential.JTI)) ||
		transaction.AccessRevision != credential.AccessRevision {
		h.redirectFailure(w, r, returnTo, walletsecret.ReauthenticationRequiredReason, "session_binding")
		return
	}
	if r.URL.Query().Get("error") != "" || r.URL.Query().Get("code") == "" {
		h.redirectFailure(w, r, returnTo, walletsecret.ReauthenticationRequiredReason, "authorization_response")
		return
	}
	verified, verificationErr := h.google.exchangeAndVerify(r.Context(), r.URL.Query().Get("code"), transaction.Verifier, transaction.Nonce, transaction.CreatedAt)
	if verificationErr != nil {
		reason := walletsecret.ReauthenticationUnavailableReason
		if verificationErr.reason == "google_not_allowed" {
			reason = walletsecret.ReauthenticationRequiredReason
		}
		h.redirectFailure(w, r, returnTo, reason, verificationErr.stage)
		return
	}
	account, err := h.credentials.Get(credential.AccountID)
	if err != nil || account.IdentityProvider != accountcredentials.IdentityProviderGoogle ||
		!authregistration.ConstantTimeEqual(account.IdentitySubject, verified.identity.Subject) {
		h.redirectFailure(w, r, returnTo, walletsecret.ReauthenticationRequiredReason, "identity_binding")
		return
	}
	if _, err := h.leases.Issue(r.Context(), w, credential); err != nil {
		reason := walletsecret.ReauthenticationRequiredReason
		if walletsecret.IsUnavailable(err) {
			reason = walletsecret.ReauthenticationUnavailableReason
		}
		h.redirectFailure(w, r, returnTo, reason, "lease_issue")
		return
	}
	log.WithFields(log.Fields{"stage": "complete", "provider": accountcredentials.IdentityProviderGoogle, "account_id": credential.AccountID}).Info("Wallet-secret Google reauthentication succeeded")
	http.Redirect(w, r, returnTo, http.StatusSeeOther)
}

func (h *walletSecretReauthentication) setStateCookie(w http.ResponseWriter, value string) {
	http.SetCookie(w, &http.Cookie{
		Name:     walletSecretStateCookieName,
		Value:    value,
		Path:     "/auth/google",
		MaxAge:   int(walletSecretTransactionTTL / time.Second),
		Expires:  time.Now().Add(walletSecretTransactionTTL),
		HttpOnly: true,
		Secure:   h.google.secureCookie,
		SameSite: http.SameSiteLaxMode,
	})
}

func (h *walletSecretReauthentication) clearStateCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     walletSecretStateCookieName,
		Value:    "",
		Path:     "/auth/google",
		MaxAge:   -1,
		Expires:  time.Unix(1, 0),
		HttpOnly: true,
		Secure:   h.google.secureCookie,
		SameSite: http.SameSiteLaxMode,
	})
}

func (h *walletSecretReauthentication) redirectFailure(w http.ResponseWriter, r *http.Request, returnTo, reason, stage string) {
	log.WithFields(log.Fields{"stage": stage, "reason": reason, "provider": accountcredentials.IdentityProviderGoogle}).Warn("Wallet-secret Google reauthentication failed")
	http.Redirect(w, r, walletSecretFailureURL(returnTo, reason), http.StatusSeeOther)
}

func validateWalletSecretReturnTo(raw string) string {
	validated := authregistration.ValidateReturnTo(raw)
	if raw == "" || validated == authregistration.DefaultReturnTo {
		return walletSecretDefaultReturnTo
	}
	return validated
}

func walletSecretFailureURL(returnTo, reason string) string {
	parsed, err := url.Parse(validateWalletSecretReturnTo(returnTo))
	if err != nil {
		return walletSecretDefaultReturnTo + "?walletSecretReason=" + url.QueryEscape(reason)
	}
	query := parsed.Query()
	query.Set("walletSecretReason", reason)
	parsed.RawQuery = query.Encode()
	return parsed.String()
}
