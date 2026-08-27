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
	"github.com/useryege/athena/internal/walletsecret"
)

const (
	walletSecretChallengeCookieName = "athena.wallet-secret.solana.challenge"
	walletSecretChallengeCookiePath = "/auth/wallet-secrets/solana"
	walletSecretSIWSStatement       = "Authorize Athena to reveal custodial wallet private keys only. This request will not trigger a blockchain transaction or network fee."
)

type walletSecretChallengeRequest struct{}

type walletSecretChallengeResponse struct {
	Message   string `json:"message"`
	ExpiresAt int64  `json:"expiresAt"`
}

type walletSecretVerifyRequest struct {
	Signature string `json:"signature"`
}

type walletSecretVerifyResponse struct {
	ExpiresAt int64 `json:"expiresAt"`
}

// WalletSecretAuthenticator validates only Athena's login cookie, attaches its
// typed credential to the returned context, and enforces Wallet READ_WRITE.
type WalletSecretAuthenticator func(request *http.Request) (context.Context, accountcredentials.AuthenticatedCredential, error)

type walletSecretReauthentication struct {
	phantom      *Handler
	store        *walletSecretChallengeStore
	authenticate WalletSecretAuthenticator
	credentials  *accountcredentials.CredentialManager
	leases       *walletsecret.Manager
}

// EnableWalletSecretReauthentication adds the authenticated, address-free SIWS
// proof used only to authorize custodial private-key viewing.
func (h *Handler) EnableWalletSecretReauthentication(
	redisClient *redis.Client,
	authenticate WalletSecretAuthenticator,
	credentials *accountcredentials.CredentialManager,
	leases *walletsecret.Manager,
) error {
	if h == nil || authenticate == nil || credentials == nil || leases == nil {
		return fmt.Errorf("Solana wallet-secret reauthentication dependencies are required")
	}
	store, err := newWalletSecretChallengeStore(redisClient)
	if err != nil {
		return err
	}
	h.walletSecrets = &walletSecretReauthentication{
		phantom:      h,
		store:        store,
		authenticate: authenticate,
		credentials:  credentials,
		leases:       leases,
	}
	return nil
}

func (h *Handler) WalletSecretChallenge(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.walletSecrets == nil {
		walletsecret.WriteError(w, walletsecret.ErrReauthenticationUnavailable)
		return
	}
	h.walletSecrets.challenge(w, r)
}

func (h *Handler) WalletSecretVerify(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.walletSecrets == nil {
		walletsecret.WriteError(w, walletsecret.ErrReauthenticationUnavailable)
		return
	}
	h.walletSecrets.verify(w, r)
}

func (h *walletSecretReauthentication) challenge(w http.ResponseWriter, r *http.Request) {
	walletsecret.SetSecretResponseHeaders(w)
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if !h.phantom.validOrigin(r) {
		walletsecret.WriteError(w, walletsecret.ErrReauthenticationRequired)
		return
	}
	if !isJSONRequest(r) {
		walletsecret.WriteError(w, walletsecret.ErrReauthenticationRequired)
		return
	}
	var input walletSecretChallengeRequest
	if err := decodeJSON(w, r, &input); err != nil {
		walletsecret.WriteError(w, walletsecret.ErrReauthenticationRequired)
		return
	}
	_, credential, err := h.authenticate(r)
	if err != nil || credential.Capability != accountcredentials.CapabilityLogin {
		walletsecret.WriteError(w, walletsecret.ErrLoginSessionRequired)
		return
	}
	account, err := h.credentials.Get(credential.AccountID)
	if err != nil || account.IdentityProvider != accountcredentials.IdentityProviderSolanaWallet || !account.HasExternalIdentity() {
		walletsecret.WriteError(w, walletsecret.ErrReauthenticationRequired)
		return
	}
	id, err := authregistration.RandomOpaqueValue()
	if err != nil {
		walletsecret.WriteError(w, walletsecret.ErrReauthenticationUnavailable)
		return
	}
	nonce, err := randomNonce()
	if err != nil {
		walletsecret.WriteError(w, walletsecret.ErrReauthenticationUnavailable)
		return
	}
	issuedAt := time.Now().UTC().Truncate(time.Second)
	expiresAt := issuedAt.Add(challengeTTL)
	message := h.phantom.siwsMessageWithStatement(account.IdentitySubject, walletSecretSIWSStatement, nonce, issuedAt, expiresAt)
	if err := h.store.create(r.Context(), id, walletSecretChallenge{
		Address:          account.IdentitySubject,
		Message:          message,
		Nonce:            nonce,
		AccountID:        credential.AccountID,
		SessionJTIDigest: walletsecret.SessionJTIDigest(credential.JTI),
		AccessRevision:   credential.AccessRevision,
		CreatedAt:        issuedAt,
		ExpiresAt:        expiresAt,
	}); err != nil {
		walletsecret.WriteError(w, walletsecret.ErrReauthenticationUnavailable)
		return
	}
	h.setChallengeCookie(w, id, expiresAt)
	h.phantom.writeJSON(w, http.StatusOK, walletSecretChallengeResponse{Message: message, ExpiresAt: expiresAt.Unix()})
}

func (h *walletSecretReauthentication) verify(w http.ResponseWriter, r *http.Request) {
	walletsecret.SetSecretResponseHeaders(w)
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if !h.phantom.validOrigin(r) {
		walletsecret.WriteError(w, walletsecret.ErrReauthenticationRequired)
		return
	}
	cookie, err := r.Cookie(walletSecretChallengeCookieName)
	h.clearChallengeCookie(w)
	if err != nil || !authregistration.ValidOpaqueValue(cookie.Value) {
		walletsecret.WriteError(w, walletsecret.ErrReauthenticationRequired)
		return
	}
	stored, err := h.store.consume(r.Context(), cookie.Value)
	if err != nil {
		if errors.Is(err, errWalletSecretChallengeUnavailable) {
			walletsecret.WriteError(w, walletsecret.ErrReauthenticationUnavailable)
			return
		}
		walletsecret.WriteError(w, walletsecret.ErrReauthenticationRequired)
		return
	}
	_, credential, err := h.authenticate(r)
	if err != nil || credential.Capability != accountcredentials.CapabilityLogin {
		walletsecret.WriteError(w, walletsecret.ErrLoginSessionRequired)
		return
	}
	if !authregistration.ConstantTimeEqual(stored.AccountID, credential.AccountID) ||
		!authregistration.ConstantTimeEqual(stored.SessionJTIDigest, walletsecret.SessionJTIDigest(credential.JTI)) ||
		stored.AccessRevision != credential.AccessRevision {
		walletsecret.WriteError(w, walletsecret.ErrReauthenticationRequired)
		return
	}
	account, err := h.credentials.Get(credential.AccountID)
	if err != nil || account.IdentityProvider != accountcredentials.IdentityProviderSolanaWallet ||
		!authregistration.ConstantTimeEqual(account.IdentitySubject, stored.Address) {
		walletsecret.WriteError(w, walletsecret.ErrReauthenticationRequired)
		return
	}
	expectedMessage := h.phantom.siwsMessageWithStatement(stored.Address, walletSecretSIWSStatement, stored.Nonce, stored.CreatedAt, stored.ExpiresAt)
	if !authregistration.ConstantTimeEqual(expectedMessage, stored.Message) || !isJSONRequest(r) {
		walletsecret.WriteError(w, walletsecret.ErrReauthenticationRequired)
		return
	}
	var input walletSecretVerifyRequest
	if err := decodeJSON(w, r, &input); err != nil {
		walletsecret.WriteError(w, walletsecret.ErrReauthenticationRequired)
		return
	}
	signature, err := base64.RawURLEncoding.DecodeString(input.Signature)
	if err != nil || len(signature) != ed25519.SignatureSize || base64.RawURLEncoding.EncodeToString(signature) != input.Signature {
		walletsecret.WriteError(w, walletsecret.ErrReauthenticationRequired)
		return
	}
	publicKey, err := base58.Decode(stored.Address)
	if err != nil || len(publicKey) != ed25519.PublicKeySize || !ed25519.Verify(ed25519.PublicKey(publicKey), []byte(stored.Message), signature) {
		walletsecret.WriteError(w, walletsecret.ErrReauthenticationRequired)
		return
	}
	expiresAt, err := h.leases.Issue(r.Context(), w, credential)
	if err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	log.WithFields(log.Fields{"stage": "complete", "provider": accountcredentials.IdentityProviderSolanaWallet, "account_id": credential.AccountID}).Info("Wallet-secret Solana reauthentication succeeded")
	h.phantom.writeJSON(w, http.StatusOK, walletSecretVerifyResponse{ExpiresAt: expiresAt.Unix()})
}

func (h *walletSecretReauthentication) setChallengeCookie(w http.ResponseWriter, value string, expiresAt time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     walletSecretChallengeCookieName,
		Value:    value,
		Path:     walletSecretChallengeCookiePath,
		MaxAge:   int(challengeTTL / time.Second),
		Expires:  expiresAt,
		HttpOnly: true,
		Secure:   h.phantom.secureCookie,
		SameSite: http.SameSiteStrictMode,
	})
}

func (h *walletSecretReauthentication) clearChallengeCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     walletSecretChallengeCookieName,
		Value:    "",
		Path:     walletSecretChallengeCookiePath,
		MaxAge:   -1,
		Expires:  time.Unix(1, 0),
		HttpOnly: true,
		Secure:   h.phantom.secureCookie,
		SameSite: http.SameSiteStrictMode,
	})
}
