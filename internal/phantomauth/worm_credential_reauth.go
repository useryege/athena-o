package phantomauth

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/mr-tron/base58/base58"
	"github.com/redis/go-redis/v9"
	log "github.com/sirupsen/logrus"

	"github.com/useryege/athena/internal/accountcredentials"
	"github.com/useryege/athena/internal/authregistration"
	operationlogrecord "github.com/useryege/athena/internal/operationlog/record"
	"github.com/useryege/athena/internal/walletsecret"
)

const (
	wormCredentialChallengeCookieName = "athena.worm-trading.solana.challenge"
	wormCredentialChallengeCookiePath = "/auth/worm-trading/solana"
	wormCredentialSIWSStatement       = "Authorize Athena to manage Worm API credentials only. This request will not trigger a blockchain transaction or network fee."
)

type wormCredentialChallengeRequest struct{}

type wormCredentialChallengeResponse struct {
	Message   string `json:"message"`
	ExpiresAt int64  `json:"expiresAt"`
}

type wormCredentialVerifyRequest struct {
	Signature string `json:"signature"`
}

type wormCredentialVerifyResponse struct {
	ExpiresAt int64 `json:"expiresAt"`
}

// WormCredentialAuthenticator validates only Athena's login cookie, attaches
// its typed credential to the returned context, and enforces Worm Trading
// READ_WRITE.
type WormCredentialAuthenticator func(request *http.Request) (context.Context, accountcredentials.AuthenticatedCredential, error)

type wormCredentialReauthentication struct {
	phantom      *Handler
	store        *wormCredentialChallengeStore
	authenticate WormCredentialAuthenticator
	admit        func(context.Context) error
	credentials  *accountcredentials.CredentialManager
	leases       *walletsecret.Manager
}

// EnableWormCredentialReauthentication adds the authenticated, address-free
// SIWS proof used only to authorize Worm API credential management.
func (h *Handler) EnableWormCredentialReauthentication(
	redisClient *redis.Client,
	authenticate WormCredentialAuthenticator,
	admit func(context.Context) error,
	credentials *accountcredentials.CredentialManager,
	leases *walletsecret.Manager,
) error {
	if h == nil || authenticate == nil || admit == nil || credentials == nil || leases == nil {
		return fmt.Errorf("Solana Worm credential reauthentication dependencies are required")
	}
	store, err := newWormCredentialChallengeStore(redisClient)
	if err != nil {
		return err
	}
	h.wormCredentials = &wormCredentialReauthentication{
		phantom:      h,
		store:        store,
		authenticate: authenticate,
		admit:        admit,
		credentials:  credentials,
		leases:       leases,
	}
	return nil
}

func (h *Handler) WormCredentialChallenge(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.wormCredentials == nil {
		walletsecret.WriteError(w, walletsecret.ErrWormReauthenticationUnavailable)
		return
	}
	h.wormCredentials.challenge(w, r)
}

func (h *Handler) WormCredentialVerify(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.wormCredentials == nil {
		walletsecret.WriteError(w, walletsecret.ErrWormReauthenticationUnavailable)
		return
	}
	h.wormCredentials.verify(w, r)
}

func (h *wormCredentialReauthentication) challenge(w http.ResponseWriter, r *http.Request) {
	observed := false
	defer func() {
		if !observed {
			observeAuthorizationFailure(r.Context(), "challenge", responseStatusCode(w), "AUTHORIZATION_FAILED")
		}
	}()
	walletsecret.SetSecretResponseHeaders(w)
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if !h.phantom.validOrigin(r) || !isJSONRequest(r) {
		walletsecret.WriteError(w, walletsecret.ErrWormReauthenticationRequired)
		return
	}
	var input wormCredentialChallengeRequest
	if err := decodeJSON(w, r, &input); err != nil {
		walletsecret.WriteError(w, walletsecret.ErrWormReauthenticationRequired)
		return
	}
	authCtx, credential, err := h.authenticate(r)
	if err != nil || credential.Capability != accountcredentials.CapabilityLogin {
		walletsecret.WriteError(w, walletsecret.ErrWormLoginSessionRequired)
		return
	}
	if err := h.admit(authCtx); err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	account, err := h.credentials.Get(credential.AccountID)
	if err != nil || account.IdentityProvider != accountcredentials.IdentityProviderSolanaWallet || !account.HasExternalIdentity() {
		walletsecret.WriteError(w, walletsecret.ErrWormReauthenticationRequired)
		return
	}
	id, err := authregistration.RandomOpaqueValue()
	if err != nil {
		walletsecret.WriteError(w, walletsecret.ErrWormReauthenticationUnavailable)
		return
	}
	nonce, err := randomNonce()
	if err != nil {
		walletsecret.WriteError(w, walletsecret.ErrWormReauthenticationUnavailable)
		return
	}
	issuedAt := time.Now().UTC().Truncate(time.Second)
	expiresAt := issuedAt.Add(challengeTTL)
	message := h.phantom.siwsMessageWithStatement(account.IdentitySubject, wormCredentialSIWSStatement, nonce, issuedAt, expiresAt)
	if err := h.store.create(r.Context(), id, wormCredentialChallenge{
		Address:          account.IdentitySubject,
		Message:          message,
		Nonce:            nonce,
		AccountID:        credential.AccountID,
		SessionJTIDigest: walletsecret.SessionJTIDigest(credential.JTI),
		AccessRevision:   credential.AccessRevision,
		CreatedAt:        issuedAt,
		ExpiresAt:        expiresAt,
	}); err != nil {
		walletsecret.WriteError(w, walletsecret.ErrWormReauthenticationUnavailable)
		return
	}
	h.setChallengeCookie(w, id, expiresAt)
	observed = true
	h.phantom.writeJSON(w, http.StatusOK, wormCredentialChallengeResponse{Message: message, ExpiresAt: expiresAt.Unix()})
}

func (h *wormCredentialReauthentication) verify(w http.ResponseWriter, r *http.Request) {
	observed := false
	defer func() {
		if !observed {
			observeAuthorizationFailure(r.Context(), "verify", responseStatusCode(w), "AUTHORIZATION_FAILED")
		}
	}()
	walletsecret.SetSecretResponseHeaders(w)
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if !h.phantom.validOrigin(r) {
		walletsecret.WriteError(w, walletsecret.ErrWormReauthenticationRequired)
		return
	}
	cookie, err := r.Cookie(wormCredentialChallengeCookieName)
	h.clearChallengeCookie(w)
	if err != nil || !authregistration.ValidOpaqueValue(cookie.Value) {
		walletsecret.WriteError(w, walletsecret.ErrWormReauthenticationRequired)
		return
	}
	stored, err := h.store.consume(r.Context(), cookie.Value)
	if err != nil {
		if errors.Is(err, errWormCredentialChallengeUnavailable) {
			walletsecret.WriteError(w, walletsecret.ErrWormReauthenticationUnavailable)
			return
		}
		walletsecret.WriteError(w, walletsecret.ErrWormReauthenticationRequired)
		return
	}
	authCtx, credential, err := h.authenticate(r)
	if err != nil || credential.Capability != accountcredentials.CapabilityLogin {
		walletsecret.WriteError(w, walletsecret.ErrWormLoginSessionRequired)
		return
	}
	if !authregistration.ConstantTimeEqual(stored.AccountID, credential.AccountID) ||
		!authregistration.ConstantTimeEqual(stored.SessionJTIDigest, walletsecret.SessionJTIDigest(credential.JTI)) ||
		stored.AccessRevision != credential.AccessRevision {
		walletsecret.WriteError(w, walletsecret.ErrWormReauthenticationRequired)
		return
	}
	if err := h.admit(authCtx); err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	account, err := h.credentials.Get(credential.AccountID)
	if err != nil || account.IdentityProvider != accountcredentials.IdentityProviderSolanaWallet ||
		!authregistration.ConstantTimeEqual(account.IdentitySubject, stored.Address) {
		walletsecret.WriteError(w, walletsecret.ErrWormReauthenticationRequired)
		return
	}
	expectedMessage := h.phantom.siwsMessageWithStatement(stored.Address, wormCredentialSIWSStatement, stored.Nonce, stored.CreatedAt, stored.ExpiresAt)
	if !authregistration.ConstantTimeEqual(expectedMessage, stored.Message) || !isJSONRequest(r) {
		walletsecret.WriteError(w, walletsecret.ErrWormReauthenticationRequired)
		return
	}
	var input wormCredentialVerifyRequest
	if err := decodeJSON(w, r, &input); err != nil {
		walletsecret.WriteError(w, walletsecret.ErrWormReauthenticationRequired)
		return
	}
	signature, err := base64.RawURLEncoding.DecodeString(input.Signature)
	if err != nil || len(signature) != ed25519.SignatureSize || base64.RawURLEncoding.EncodeToString(signature) != input.Signature {
		walletsecret.WriteError(w, walletsecret.ErrWormReauthenticationRequired)
		return
	}
	publicKey, err := base58.Decode(stored.Address)
	if err != nil || len(publicKey) != ed25519.PublicKeySize || !ed25519.Verify(ed25519.PublicKey(publicKey), []byte(stored.Message), signature) {
		walletsecret.WriteError(w, walletsecret.ErrWormReauthenticationRequired)
		return
	}
	expiresAt, err := h.leases.Issue(r.Context(), w, credential)
	if err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	log.WithFields(log.Fields{"stage": "complete", "provider": accountcredentials.IdentityProviderSolanaWallet}).Info("Worm credential Solana reauthentication succeeded")
	operationlogrecord.CaptureResource(r.Context(), "account", credential.AccountID)
	operationlogrecord.BindVerifiedAccount(r.Context(), credential.AccountID, "solana_wallet")
	operationlogrecord.CaptureString(r.Context(), "provider", "solana_wallet")
	operationlogrecord.CaptureString(r.Context(), "proofKind", "PHANTOM")
	operationlogrecord.CaptureString(r.Context(), "stage", "authorization_verified")
	observed = true
	operationlogrecord.Commit(r.Context(), "WORM_CONNECTION_AUTHORIZE")
	h.phantom.writeJSON(w, http.StatusOK, wormCredentialVerifyResponse{ExpiresAt: expiresAt.Unix()})
}

func (h *wormCredentialReauthentication) setChallengeCookie(w http.ResponseWriter, value string, expiresAt time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     wormCredentialChallengeCookieName,
		Value:    value,
		Path:     h.phantom.deploymentPath(wormCredentialChallengeCookiePath),
		MaxAge:   int(challengeTTL / time.Second),
		Expires:  expiresAt,
		HttpOnly: true,
		Secure:   h.phantom.secureCookie,
		SameSite: http.SameSiteStrictMode,
	})
}

func (h *wormCredentialReauthentication) clearChallengeCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     wormCredentialChallengeCookieName,
		Value:    "",
		Path:     h.phantom.deploymentPath(wormCredentialChallengeCookiePath),
		MaxAge:   -1,
		Expires:  time.Unix(1, 0),
		HttpOnly: true,
		Secure:   h.phantom.secureCookie,
		SameSite: http.SameSiteStrictMode,
	})
}
