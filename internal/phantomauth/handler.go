package phantomauth

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/mr-tron/base58/base58"
	"github.com/redis/go-redis/v9"
	log "github.com/sirupsen/logrus"

	"github.com/useryege/athena/common"
	"github.com/useryege/athena/internal/accountcredentials"
	"github.com/useryege/athena/internal/authregistration"
)

const (
	challengeCookieName = "athena.phantom.challenge"
	challengeCookiePath = "/auth/phantom"
	siwsStatement       = "Sign in to Athena only. This request will not trigger a blockchain transaction or network fee."
	siwsVersion         = "1"
	siwsChainID         = "solana:mainnet"
)

type challengeRequest struct {
	Address  string `json:"address"`
	ReturnTo string `json:"returnTo"`
}

type challengeResponse struct {
	Message   string `json:"message"`
	ExpiresAt int64  `json:"expiresAt"`
}

type verifyRequest struct {
	Signature string `json:"signature"`
}

type redirectResponse struct {
	RedirectTo string `json:"redirectTo"`
}

type errorResponse struct {
	Reason string `json:"reason"`
}

// Handler implements wallet-standard Sign-In With Solana verification. The
// proof authenticates a Solana public key; it does not attest a wallet brand.
type Handler struct {
	store                *challengeStore
	backend              authregistration.Backend
	registrations        *authregistration.Handler
	publicOrigin         string
	domain               string
	secureCookie         bool
	baseHRef             string
	walletSecrets        *walletSecretReauthentication
	wormCredentials      *wormCredentialReauthentication
	wormExecutions       *wormExecutionAuthorization
	wormPositionCashOuts *wormPositionCashOutAuthorization
}

func NewHandler(redisClient *redis.Client, backend authregistration.Backend, registrations *authregistration.Handler, publicOrigin, baseHRef string) (*Handler, error) {
	store, err := newChallengeStore(redisClient)
	if err != nil {
		return nil, err
	}
	if backend == nil || registrations == nil {
		return nil, fmt.Errorf("Phantom authentication backend and registration handler are required")
	}
	origin, err := validatePublicOrigin(publicOrigin)
	if err != nil {
		return nil, err
	}
	return &Handler{
		store:         store,
		backend:       backend,
		registrations: registrations,
		publicOrigin:  origin.Scheme + "://" + origin.Host,
		domain:        origin.Host,
		secureCookie:  origin.Scheme == "https",
		baseHRef:      baseHRef,
	}, nil
}

// Challenge creates the exact message that the browser asks its wallet to
// sign. Only message and expiry cross the public response boundary.
func (h *Handler) Challenge(w http.ResponseWriter, r *http.Request) {
	authregistration.SetResponseHeaders(w)
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if err := requireMemberApplicationRealm(r); err != nil {
		h.fail(w, http.StatusBadRequest, "phantom_state_invalid", "application_realm", err)
		return
	}
	if !h.validOrigin(r) {
		h.fail(w, http.StatusForbidden, "phantom_state_invalid", "origin_verify", nil)
		return
	}
	if !isJSONRequest(r) {
		h.fail(w, http.StatusUnsupportedMediaType, "phantom_signature_invalid", "content_type", nil)
		return
	}
	var input challengeRequest
	if err := decodeJSON(w, r, &input); err != nil {
		h.fail(w, http.StatusBadRequest, "phantom_signature_invalid", "challenge_decode", err)
		return
	}
	address, err := accountcredentials.NormalizeIdentitySubject(accountcredentials.IdentityProviderSolanaWallet, input.Address)
	if err != nil {
		h.fail(w, http.StatusBadRequest, "phantom_signature_invalid", "address_validate", err)
		return
	}
	id, err := authregistration.RandomOpaqueValue()
	if err != nil {
		h.fail(w, http.StatusServiceUnavailable, "phantom_unavailable", "challenge_id_generation", err)
		return
	}
	nonce, err := randomNonce()
	if err != nil {
		h.fail(w, http.StatusServiceUnavailable, "phantom_unavailable", "nonce_generation", err)
		return
	}
	issuedAt := time.Now().UTC().Truncate(time.Second)
	expiresAt := issuedAt.Add(challengeTTL)
	message := h.siwsMessage(address, nonce, issuedAt, expiresAt)
	if err := h.store.Create(r.Context(), id, authregistration.ClientRateIdentity(r), challenge{
		Address:   address,
		Message:   message,
		Nonce:     nonce,
		ReturnTo:  authregistration.ReturnToForRealm(input.ReturnTo, accountcredentials.ApplicationRealmMember),
		CreatedAt: issuedAt,
		ExpiresAt: expiresAt,
	}); err != nil {
		statusCode := http.StatusServiceUnavailable
		stage := "challenge_create"
		if errors.Is(err, errChallengeRateLimited) {
			statusCode = http.StatusTooManyRequests
			stage = "challenge_rate_limited"
		}
		h.fail(w, statusCode, "phantom_unavailable", stage, err)
		return
	}
	h.setChallengeCookie(w, id)
	h.writeJSON(w, http.StatusOK, challengeResponse{Message: message, ExpiresAt: expiresAt.Unix()})
}

// Verify consumes the server-side challenge first, then verifies a raw
// base64url Ed25519 signature over the exact stored UTF-8 SIWS message.
func (h *Handler) Verify(w http.ResponseWriter, r *http.Request) {
	authregistration.SetResponseHeaders(w)
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if err := requireMemberApplicationRealm(r); err != nil {
		h.fail(w, http.StatusBadRequest, "phantom_state_invalid", "application_realm", err)
		return
	}
	if !h.validOrigin(r) {
		h.fail(w, http.StatusForbidden, "phantom_state_invalid", "origin_verify", nil)
		return
	}
	cookie, err := r.Cookie(challengeCookieName)
	h.clearChallengeCookie(w)
	if err != nil || !authregistration.ValidOpaqueValue(cookie.Value) {
		h.fail(w, http.StatusUnauthorized, "phantom_state_invalid", "challenge_cookie", err)
		return
	}
	stored, err := h.store.Consume(r.Context(), cookie.Value)
	if err != nil {
		reason := "phantom_state_invalid"
		statusCode := http.StatusUnauthorized
		if errors.Is(err, errChallengeUnavailable) {
			reason = "phantom_unavailable"
			statusCode = http.StatusServiceUnavailable
		}
		h.fail(w, statusCode, reason, "challenge_consume", err)
		return
	}
	if !isJSONRequest(r) {
		h.fail(w, http.StatusUnsupportedMediaType, "phantom_signature_invalid", "content_type", nil)
		return
	}
	canonicalAddress, canonicalErr := accountcredentials.NormalizeIdentitySubject(accountcredentials.IdentityProviderSolanaWallet, stored.Address)
	expectedMessage := h.siwsMessage(canonicalAddress, stored.Nonce, stored.CreatedAt, stored.ExpiresAt)
	if canonicalErr != nil || canonicalAddress != stored.Address || !authregistration.ConstantTimeEqual(expectedMessage, stored.Message) {
		h.fail(w, http.StatusUnauthorized, "phantom_state_invalid", "challenge_binding", canonicalErr)
		return
	}
	var input verifyRequest
	if err := decodeJSON(w, r, &input); err != nil {
		h.fail(w, http.StatusBadRequest, "phantom_signature_invalid", "signature_decode", err)
		return
	}
	signature, err := base64.RawURLEncoding.DecodeString(input.Signature)
	if err != nil || len(signature) != ed25519.SignatureSize || base64.RawURLEncoding.EncodeToString(signature) != input.Signature {
		h.fail(w, http.StatusForbidden, "phantom_signature_invalid", "signature_encoding", err)
		return
	}
	publicKey, err := base58.Decode(stored.Address)
	if err != nil || len(publicKey) != ed25519.PublicKeySize || !ed25519.Verify(ed25519.PublicKey(publicKey), []byte(stored.Message), signature) {
		h.fail(w, http.StatusForbidden, "phantom_signature_invalid", "signature_verify", err)
		return
	}
	identity := authregistration.Identity{
		Provider: accountcredentials.IdentityProviderSolanaWallet,
		Subject:  stored.Address,
		Realm:    accountcredentials.ApplicationRealmMember,
	}
	account, found, err := h.backend.GetByIdentity(r.Context(), identity)
	if err != nil {
		h.fail(w, http.StatusServiceUnavailable, "phantom_unavailable", "identity_lookup", err)
		return
	}
	if !found {
		if err := h.registrations.Begin(r.Context(), w, identity, stored.ReturnTo); err != nil {
			h.fail(w, http.StatusServiceUnavailable, "phantom_unavailable", "registration_ticket_create", err)
			return
		}
		log.WithField("stage", "registration_required").Info("Verified Solana wallet requires Athena username registration")
		registrationQuery := url.Values{common.ApplicationRealmQueryParameter: []string{string(accountcredentials.ApplicationRealmMember)}}
		h.writeJSON(w, http.StatusOK, redirectResponse{RedirectTo: "/register?" + registrationQuery.Encode()})
		return
	}
	token, err := h.backend.CreateExternalLogin(r.Context(), account.ID, identity)
	if err != nil {
		if errors.Is(err, authregistration.ErrMaintenance) {
			h.fail(w, http.StatusServiceUnavailable, "maintenance", "account_maintenance", err)
			return
		}
		reason := "phantom_unavailable"
		statusCode := http.StatusServiceUnavailable
		if errors.Is(err, authregistration.ErrIdentityNotAllowed) {
			reason = "phantom_signature_invalid"
			statusCode = http.StatusForbidden
		}
		h.fail(w, statusCode, reason, "session_issue", err)
		return
	}
	h.registrations.ClearCookie(w)
	if err := h.registrations.SetAthenaSessionCookie(w, accountcredentials.ApplicationRealmMember, token); err != nil {
		h.fail(w, http.StatusServiceUnavailable, "phantom_unavailable", "cookie_issue", err)
		return
	}
	h.backend.RecordLoginResult(authregistration.LoginSuccess)
	log.WithFields(log.Fields{"stage": "complete", "provider": accountcredentials.IdentityProviderSolanaWallet, "account_id": account.ID}).Info("Solana wallet login succeeded")
	h.writeJSON(w, http.StatusOK, redirectResponse{RedirectTo: authregistration.ReturnToForRealm(stored.ReturnTo, accountcredentials.ApplicationRealmMember)})
}

func requireMemberApplicationRealm(r *http.Request) error {
	values := r.Header.Values(common.ApplicationRealmHeader)
	if len(values) != 1 || values[0] != string(accountcredentials.ApplicationRealmMember) {
		return fmt.Errorf("member application realm header is required exactly once")
	}
	if _, present := r.URL.Query()[common.ApplicationRealmQueryParameter]; present {
		return fmt.Errorf("application realm query is not accepted")
	}
	return nil
}

func (h *Handler) siwsMessage(address, nonce string, issuedAt, expiresAt time.Time) string {
	return h.siwsMessageWithStatement(address, siwsStatement, nonce, issuedAt, expiresAt)
}

func (h *Handler) siwsMessageWithStatement(address, statement, nonce string, issuedAt, expiresAt time.Time) string {
	return fmt.Sprintf(
		"%s wants you to sign in with your Solana account:\n%s\n\n%s\n\nURI: %s\nVersion: %s\nChain ID: %s\nNonce: %s\nIssued At: %s\nExpiration Time: %s",
		h.domain,
		address,
		statement,
		h.publicOrigin,
		siwsVersion,
		siwsChainID,
		nonce,
		issuedAt.Format(time.RFC3339),
		expiresAt.Format(time.RFC3339),
	)
}

func (h *Handler) validOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	return origin != "" && authregistration.ConstantTimeEqual(origin, h.publicOrigin)
}

func (h *Handler) setChallengeCookie(w http.ResponseWriter, value string) {
	http.SetCookie(w, &http.Cookie{
		Name:     challengeCookieName,
		Value:    value,
		Path:     h.deploymentPath(challengeCookiePath),
		MaxAge:   int(challengeTTL.Seconds()),
		Expires:  time.Now().Add(challengeTTL),
		HttpOnly: true,
		Secure:   h.secureCookie,
		SameSite: http.SameSiteStrictMode,
	})
}

func (h *Handler) clearChallengeCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     challengeCookieName,
		Value:    "",
		Path:     h.deploymentPath(challengeCookiePath),
		MaxAge:   -1,
		Expires:  time.Unix(1, 0),
		HttpOnly: true,
		Secure:   h.secureCookie,
		SameSite: http.SameSiteStrictMode,
	})
}

func (h *Handler) fail(w http.ResponseWriter, statusCode int, reason, stage string, err error) {
	fields := log.Fields{"stage": stage, "reason": reason, "provider": accountcredentials.IdentityProviderSolanaWallet}
	if err != nil {
		fields["error_type"] = fmt.Sprintf("%T", err)
	}
	log.WithFields(fields).Warn("Solana wallet authentication failed")
	h.backend.RecordLoginResult(authregistration.LoginFailure)
	h.writeJSON(w, statusCode, errorResponse{Reason: reason})
}

func (h *Handler) deploymentPath(logicalPath string) string {
	return authregistration.DeploymentPath(h.baseHRef, logicalPath)
}

func (h *Handler) writeJSON(w http.ResponseWriter, statusCode int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.WithError(err).Warn("Solana wallet authentication response encoding failed")
	}
}

func validatePublicOrigin(raw string) (*url.URL, error) {
	parsed, err := url.Parse(raw)
	if err != nil || !parsed.IsAbs() || parsed.Host == "" || parsed.User != nil || parsed.Opaque != "" {
		return nil, fmt.Errorf("public origin must be an absolute HTTP origin")
	}
	if parsed.Path != "" || parsed.RawPath != "" || parsed.ForceQuery || parsed.RawQuery != "" || parsed.Fragment != "" {
		return nil, fmt.Errorf("public origin must not contain a path, query, or fragment")
	}
	if parsed.Scheme != "https" && (parsed.Scheme != "http" || parsed.Hostname() != "localhost") {
		return nil, fmt.Errorf("public origin must use HTTPS outside localhost")
	}
	if (parsed.Scheme == "https" && parsed.Port() == "443") || (parsed.Scheme == "http" && parsed.Port() == "80") {
		return nil, fmt.Errorf("public origin must omit its default port")
	}
	canonical := parsed.Scheme + "://" + parsed.Host
	if raw != canonical || strings.ToLower(parsed.Scheme) != parsed.Scheme || strings.ToLower(parsed.Hostname()) != parsed.Hostname() {
		return nil, fmt.Errorf("public origin must be canonical")
	}
	return parsed, nil
}

func randomNonce() (string, error) {
	value := make([]byte, 16)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return hex.EncodeToString(value), nil
}

func isJSONRequest(r *http.Request) bool {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	return err == nil && mediaType == "application/json"
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if err == nil {
			return fmt.Errorf("request body contains multiple JSON values")
		}
		return err
	}
	return nil
}
