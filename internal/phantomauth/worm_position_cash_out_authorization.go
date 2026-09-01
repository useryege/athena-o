package phantomauth

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mr-tron/base58/base58"
	"github.com/redis/go-redis/v9"
	log "github.com/sirupsen/logrus"

	"github.com/useryege/athena/internal/accountcredentials"
	"github.com/useryege/athena/internal/authregistration"
)

const (
	WormPositionCashOutProofKindPhantom = "PHANTOM"

	wormPositionCashOutChallengeCookieName  = "athena.worm-position-cash-out.solana.challenge"
	wormPositionCashOutChallengeCookiePath  = "/auth/worm-trading/position-cash-outs"
	wormPositionCashOutRoutePrefix          = "/auth/worm-trading/position-cash-outs/"
	wormPositionCashOutChallengeRouteSuffix = "/solana/challenge"
	wormPositionCashOutVerifyRouteSuffix    = "/solana/verify"
	wormPositionCashOutDefaultReturnTo      = "/worm-trading"

	WormPositionCashOutLoginSessionRequiredReason     = "WORM_POSITION_CASH_OUT_LOGIN_SESSION_REQUIRED"
	WormPositionCashOutAuthorizationRequiredReason    = "WORM_POSITION_CASH_OUT_AUTHORIZATION_REQUIRED"
	WormPositionCashOutAuthorizationUnavailableReason = "WORM_POSITION_CASH_OUT_AUTHORIZATION_UNAVAILABLE"
)

type wormPositionCashOutChallengeRequest struct {
	CommandID        string `json:"commandId"`
	ExpectedRevision int64  `json:"expectedRevision"`
	ReturnTo         string `json:"returnTo"`
}

type wormPositionCashOutChallengeResponse struct {
	Message   string `json:"message"`
	ExpiresAt int64  `json:"expiresAt"`
}

type wormPositionCashOutVerifyRequest struct {
	Signature string `json:"signature"`
}

type WormPositionCashOutAuthenticator func(
	request *http.Request,
) (context.Context, accountcredentials.AuthenticatedCredential, error)

type WormPositionCashOutDescriptorRequest struct {
	CashOutID              string
	CommandID              string
	ExpectedRevision       int64
	AccountID              string
	SessionJTIDigestSHA256 []byte
	AccessRevision         uint64
	ReturnTo               string
}

type WormPositionCashOutDescriptor struct {
	CashOutID          string
	Revision           int64
	IntentDigestSHA256 []byte
}

type WormPositionCashOutDescriptorLoader func(
	context.Context,
	WormPositionCashOutDescriptorRequest,
) (WormPositionCashOutDescriptor, error)

type WormPositionCashOutAuthorizationRequest struct {
	CashOutID              string
	CommandID              string
	ExpectedRevision       int64
	AccountID              string
	SessionJTIDigestSHA256 []byte
	AccessRevision         uint64
	IntentDigestSHA256     []byte
	ReturnTo               string
	ProofKind              string
}

type WormPositionCashOutAuthorizer func(
	context.Context,
	WormPositionCashOutAuthorizationRequest,
) (any, error)

type WormPositionCashOutErrorWriter func(http.ResponseWriter, error)

type wormPositionCashOutAuthorization struct {
	phantom        *Handler
	store          *wormPositionCashOutChallengeStore
	authenticate   WormPositionCashOutAuthenticator
	credentials    *accountcredentials.CredentialManager
	loadDescriptor WormPositionCashOutDescriptorLoader
	authorize      WormPositionCashOutAuthorizer
	writeError     WormPositionCashOutErrorWriter
}

// EnableWormPositionCashOutAuthorization attaches the independent,
// operation-bound Phantom identity proof.
func (h *Handler) EnableWormPositionCashOutAuthorization(
	redisClient *redis.Client,
	authenticate WormPositionCashOutAuthenticator,
	credentials *accountcredentials.CredentialManager,
	loadDescriptor WormPositionCashOutDescriptorLoader,
	authorize WormPositionCashOutAuthorizer,
	writeError WormPositionCashOutErrorWriter,
) error {
	if h == nil || authenticate == nil || credentials == nil || loadDescriptor == nil ||
		authorize == nil || writeError == nil {
		return fmt.Errorf("Solana Worm position Cash Out authorization dependencies are required")
	}
	if h.wormPositionCashOuts != nil {
		return fmt.Errorf("Solana Worm position Cash Out authorization is already enabled")
	}
	store, err := newWormPositionCashOutChallengeStore(redisClient)
	if err != nil {
		return err
	}
	h.wormPositionCashOuts = &wormPositionCashOutAuthorization{
		phantom:        h,
		store:          store,
		authenticate:   authenticate,
		credentials:    credentials,
		loadDescriptor: loadDescriptor,
		authorize:      authorize,
		writeError:     writeError,
	}
	return nil
}

func (h *Handler) WormPositionCashOutChallenge(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.wormPositionCashOuts == nil {
		setWormPositionCashOutProofHeaders(w)
		writeWormPositionCashOutProofError(
			w,
			http.StatusServiceUnavailable,
			WormPositionCashOutAuthorizationUnavailableReason,
		)
		return
	}
	h.wormPositionCashOuts.challenge(w, r)
}

func (h *Handler) WormPositionCashOutVerify(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.wormPositionCashOuts == nil {
		setWormPositionCashOutProofHeaders(w)
		writeWormPositionCashOutProofError(
			w,
			http.StatusServiceUnavailable,
			WormPositionCashOutAuthorizationUnavailableReason,
		)
		return
	}
	h.wormPositionCashOuts.verify(w, r)
}

func (h *wormPositionCashOutAuthorization) challenge(w http.ResponseWriter, r *http.Request) {
	setWormPositionCashOutProofHeaders(w)
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if !h.phantom.validOrigin(r) {
		h.fail(w, http.StatusForbidden, WormPositionCashOutAuthorizationRequiredReason, "origin_verify", nil)
		return
	}
	if !isJSONRequest(r) {
		h.fail(w, http.StatusUnsupportedMediaType, WormPositionCashOutAuthorizationRequiredReason, "content_type", nil)
		return
	}
	cashOutID, err := wormPositionCashOutIDFromRequest(r, wormPositionCashOutChallengeRouteSuffix)
	if err != nil {
		h.fail(w, http.StatusBadRequest, WormPositionCashOutAuthorizationRequiredReason, "cash_out_id", err)
		return
	}
	var input wormPositionCashOutChallengeRequest
	if err := decodeJSON(w, r, &input); err != nil {
		h.fail(w, http.StatusBadRequest, WormPositionCashOutAuthorizationRequiredReason, "challenge_decode", err)
		return
	}
	commandID, err := canonicalWormPositionCashOutID(input.CommandID, "commandId")
	if err != nil || input.ExpectedRevision <= 0 {
		h.fail(w, http.StatusBadRequest, WormPositionCashOutAuthorizationRequiredReason, "command_binding", err)
		return
	}
	returnTo := validateWormPositionCashOutReturnTo(input.ReturnTo)
	_, credential, err := h.authenticate(r)
	if err != nil || credential.Capability != accountcredentials.CapabilityLogin || credential.JTI == "" ||
		credential.AccessRevision == 0 || credential.AccessRevision > math.MaxInt64 {
		h.fail(w, http.StatusUnauthorized, WormPositionCashOutLoginSessionRequiredReason, "login_session", nil)
		return
	}
	accountID, err := accountcredentials.CanonicalAccountID(credential.AccountID)
	if err != nil || accountID != credential.AccountID {
		h.fail(w, http.StatusUnauthorized, WormPositionCashOutLoginSessionRequiredReason, "account_binding", err)
		return
	}
	account, err := h.credentials.Get(accountID)
	if err != nil || account.IdentityProvider != accountcredentials.IdentityProviderSolanaWallet ||
		!account.HasExternalIdentity() {
		h.fail(w, http.StatusUnauthorized, WormPositionCashOutAuthorizationRequiredReason, "identity_provider", err)
		return
	}
	address, err := accountcredentials.NormalizeIdentitySubject(
		accountcredentials.IdentityProviderSolanaWallet,
		account.IdentitySubject,
	)
	if err != nil || address != account.IdentitySubject {
		h.fail(w, http.StatusUnauthorized, WormPositionCashOutAuthorizationRequiredReason, "identity_address", err)
		return
	}
	sessionDigest := sha256.Sum256([]byte(credential.JTI))
	descriptorRequest := WormPositionCashOutDescriptorRequest{
		CashOutID:              cashOutID,
		CommandID:              commandID,
		ExpectedRevision:       input.ExpectedRevision,
		AccountID:              accountID,
		SessionJTIDigestSHA256: append([]byte(nil), sessionDigest[:]...),
		AccessRevision:         credential.AccessRevision,
		ReturnTo:               returnTo,
	}
	descriptor, err := h.loadDescriptor(r.Context(), descriptorRequest)
	if err != nil {
		h.failInjected(w, "descriptor_load", err)
		return
	}
	intentDigest, err := validateWormPositionCashOutDescriptor(descriptorRequest, descriptor)
	if err != nil {
		h.fail(w, http.StatusServiceUnavailable, WormPositionCashOutAuthorizationUnavailableReason, "descriptor_validate", err)
		return
	}
	id, err := authregistration.RandomOpaqueValue()
	if err != nil {
		h.fail(w, http.StatusServiceUnavailable, WormPositionCashOutAuthorizationUnavailableReason, "challenge_id_generation", err)
		return
	}
	nonce, err := randomNonce()
	if err != nil {
		h.fail(w, http.StatusServiceUnavailable, WormPositionCashOutAuthorizationUnavailableReason, "nonce_generation", err)
		return
	}
	issuedAt := time.Now().UTC().Truncate(time.Second)
	expiresAt := issuedAt.Add(wormPositionCashOutChallengeTTL)
	statement := wormPositionCashOutSIWSStatement(cashOutID, intentDigest)
	message := h.phantom.siwsMessageWithStatement(address, statement, nonce, issuedAt, expiresAt)
	if err := h.store.create(r.Context(), id, wormPositionCashOutChallenge{
		Address:                address,
		Message:                message,
		Nonce:                  nonce,
		ReturnTo:               returnTo,
		CashOutID:              cashOutID,
		CommandID:              commandID,
		ExpectedRevision:       input.ExpectedRevision,
		AccountID:              accountID,
		SessionJTIDigestSHA256: hex.EncodeToString(sessionDigest[:]),
		AccessRevision:         credential.AccessRevision,
		IntentDigestSHA256:     hex.EncodeToString(intentDigest),
		CreatedAt:              issuedAt,
		ExpiresAt:              expiresAt,
	}); err != nil {
		h.fail(w, http.StatusServiceUnavailable, WormPositionCashOutAuthorizationUnavailableReason, "challenge_create", err)
		return
	}
	h.setChallengeCookie(w, id, expiresAt)
	h.phantom.writeJSON(w, http.StatusOK, wormPositionCashOutChallengeResponse{
		Message: message, ExpiresAt: expiresAt.Unix(),
	})
}

func (h *wormPositionCashOutAuthorization) verify(w http.ResponseWriter, r *http.Request) {
	setWormPositionCashOutProofHeaders(w)
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if !h.phantom.validOrigin(r) {
		h.fail(w, http.StatusForbidden, WormPositionCashOutAuthorizationRequiredReason, "origin_verify", nil)
		return
	}
	cashOutID, err := wormPositionCashOutIDFromRequest(r, wormPositionCashOutVerifyRouteSuffix)
	if err != nil {
		h.fail(w, http.StatusBadRequest, WormPositionCashOutAuthorizationRequiredReason, "cash_out_id", err)
		return
	}
	cookie, cookieErr := r.Cookie(wormPositionCashOutChallengeCookieName)
	h.clearChallengeCookie(w)
	if cookieErr != nil || !authregistration.ValidOpaqueValue(cookie.Value) {
		h.fail(w, http.StatusUnauthorized, WormPositionCashOutAuthorizationRequiredReason, "challenge_cookie", cookieErr)
		return
	}
	stored, err := h.store.consume(r.Context(), cookie.Value)
	if err != nil {
		if errors.Is(err, errWormPositionCashOutChallengeUnavailable) {
			h.fail(w, http.StatusServiceUnavailable, WormPositionCashOutAuthorizationUnavailableReason, "challenge_consume", err)
			return
		}
		h.fail(w, http.StatusUnauthorized, WormPositionCashOutAuthorizationRequiredReason, "challenge_consume", err)
		return
	}
	if !authregistration.ConstantTimeEqual(stored.CashOutID, cashOutID) {
		h.fail(w, http.StatusUnauthorized, WormPositionCashOutAuthorizationRequiredReason, "cash_out_binding", nil)
		return
	}
	_, credential, err := h.authenticate(r)
	if err != nil || credential.Capability != accountcredentials.CapabilityLogin || credential.JTI == "" ||
		credential.AccessRevision == 0 || credential.AccessRevision > math.MaxInt64 {
		h.fail(w, http.StatusUnauthorized, WormPositionCashOutLoginSessionRequiredReason, "login_session", nil)
		return
	}
	currentSessionDigest := sha256.Sum256([]byte(credential.JTI))
	storedSessionDigest, digestErr := canonicalWormPositionCashOutDigest(stored.SessionJTIDigestSHA256)
	if digestErr != nil || !authregistration.ConstantTimeEqual(stored.AccountID, credential.AccountID) ||
		subtle.ConstantTimeCompare(storedSessionDigest, currentSessionDigest[:]) != 1 ||
		stored.AccessRevision != credential.AccessRevision {
		h.fail(w, http.StatusUnauthorized, WormPositionCashOutAuthorizationRequiredReason, "session_binding", digestErr)
		return
	}
	account, err := h.credentials.Get(credential.AccountID)
	if err != nil || account.IdentityProvider != accountcredentials.IdentityProviderSolanaWallet ||
		!authregistration.ConstantTimeEqual(account.IdentitySubject, stored.Address) {
		h.fail(w, http.StatusUnauthorized, WormPositionCashOutAuthorizationRequiredReason, "identity_binding", err)
		return
	}
	intentDigest, err := canonicalWormPositionCashOutDigest(stored.IntentDigestSHA256)
	if err != nil {
		h.fail(w, http.StatusUnauthorized, WormPositionCashOutAuthorizationRequiredReason, "intent_binding", err)
		return
	}
	descriptorRequest := WormPositionCashOutDescriptorRequest{
		CashOutID:              stored.CashOutID,
		CommandID:              stored.CommandID,
		ExpectedRevision:       stored.ExpectedRevision,
		AccountID:              stored.AccountID,
		SessionJTIDigestSHA256: append([]byte(nil), currentSessionDigest[:]...),
		AccessRevision:         stored.AccessRevision,
		ReturnTo:               stored.ReturnTo,
	}
	descriptor, err := h.loadDescriptor(r.Context(), descriptorRequest)
	if err != nil {
		h.failInjected(w, "descriptor_reload", err)
		return
	}
	currentIntentDigest, err := validateWormPositionCashOutDescriptor(descriptorRequest, descriptor)
	if err != nil || subtle.ConstantTimeCompare(intentDigest, currentIntentDigest) != 1 {
		h.fail(w, http.StatusUnauthorized, WormPositionCashOutAuthorizationRequiredReason, "descriptor_binding", err)
		return
	}
	expectedStatement := wormPositionCashOutSIWSStatement(stored.CashOutID, intentDigest)
	expectedMessage := h.phantom.siwsMessageWithStatement(
		stored.Address,
		expectedStatement,
		stored.Nonce,
		stored.CreatedAt,
		stored.ExpiresAt,
	)
	if !authregistration.ConstantTimeEqual(expectedMessage, stored.Message) {
		h.fail(w, http.StatusUnauthorized, WormPositionCashOutAuthorizationRequiredReason, "challenge_binding", nil)
		return
	}
	if !isJSONRequest(r) {
		h.fail(w, http.StatusUnsupportedMediaType, WormPositionCashOutAuthorizationRequiredReason, "content_type", nil)
		return
	}
	var input wormPositionCashOutVerifyRequest
	if err := decodeJSON(w, r, &input); err != nil {
		h.fail(w, http.StatusBadRequest, WormPositionCashOutAuthorizationRequiredReason, "signature_decode", err)
		return
	}
	signature, err := base64.RawURLEncoding.DecodeString(input.Signature)
	if err != nil || len(signature) != ed25519.SignatureSize ||
		base64.RawURLEncoding.EncodeToString(signature) != input.Signature {
		h.fail(w, http.StatusForbidden, WormPositionCashOutAuthorizationRequiredReason, "signature_encoding", err)
		return
	}
	publicKey, err := base58.Decode(stored.Address)
	if err != nil || len(publicKey) != ed25519.PublicKeySize ||
		!ed25519.Verify(ed25519.PublicKey(publicKey), []byte(stored.Message), signature) {
		h.fail(w, http.StatusForbidden, WormPositionCashOutAuthorizationRequiredReason, "signature_verify", err)
		return
	}
	projection, err := h.authorize(r.Context(), WormPositionCashOutAuthorizationRequest{
		CashOutID:              stored.CashOutID,
		CommandID:              stored.CommandID,
		ExpectedRevision:       stored.ExpectedRevision,
		AccountID:              stored.AccountID,
		SessionJTIDigestSHA256: append([]byte(nil), currentSessionDigest[:]...),
		AccessRevision:         stored.AccessRevision,
		IntentDigestSHA256:     append([]byte(nil), intentDigest...),
		ReturnTo:               stored.ReturnTo,
		ProofKind:              WormPositionCashOutProofKindPhantom,
	})
	if err != nil {
		h.failInjected(w, "authorize", err)
		return
	}
	log.WithFields(log.Fields{
		"stage": "complete", "provider": accountcredentials.IdentityProviderSolanaWallet,
	}).Info("Worm position Cash Out Solana authorization succeeded")
	h.phantom.writeJSON(w, http.StatusOK, projection)
}

func validateWormPositionCashOutDescriptor(
	request WormPositionCashOutDescriptorRequest,
	descriptor WormPositionCashOutDescriptor,
) ([]byte, error) {
	cashOutID, err := canonicalWormPositionCashOutID(descriptor.CashOutID, "descriptor cashOutId")
	if err != nil || !authregistration.ConstantTimeEqual(cashOutID, request.CashOutID) ||
		descriptor.Revision != request.ExpectedRevision || len(descriptor.IntentDigestSHA256) != sha256.Size {
		return nil, fmt.Errorf("Worm position Cash Out descriptor binding is invalid")
	}
	return append([]byte(nil), descriptor.IntentDigestSHA256...), nil
}

func wormPositionCashOutSIWSStatement(cashOutID string, intentDigest []byte) string {
	return fmt.Sprintf(
		"Authorize Athena to cash out Worm position operation ID %s with intent SHA-256 %x. This is an identity confirmation only; it does not sign a blockchain transaction or authorize a network fee.",
		cashOutID,
		intentDigest,
	)
}

func (h *wormPositionCashOutAuthorization) setChallengeCookie(
	w http.ResponseWriter,
	value string,
	expiresAt time.Time,
) {
	http.SetCookie(w, &http.Cookie{
		Name:     wormPositionCashOutChallengeCookieName,
		Value:    value,
		Path:     h.phantom.deploymentPath(wormPositionCashOutChallengeCookiePath),
		MaxAge:   int(wormPositionCashOutChallengeTTL / time.Second),
		Expires:  expiresAt,
		HttpOnly: true,
		Secure:   h.phantom.secureCookie,
		SameSite: http.SameSiteStrictMode,
	})
}

func (h *wormPositionCashOutAuthorization) clearChallengeCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     wormPositionCashOutChallengeCookieName,
		Value:    "",
		Path:     h.phantom.deploymentPath(wormPositionCashOutChallengeCookiePath),
		MaxAge:   -1,
		Expires:  time.Unix(1, 0),
		HttpOnly: true,
		Secure:   h.phantom.secureCookie,
		SameSite: http.SameSiteStrictMode,
	})
}

func (h *wormPositionCashOutAuthorization) fail(
	w http.ResponseWriter,
	statusCode int,
	reason string,
	stage string,
	err error,
) {
	fields := log.Fields{
		"stage": stage, "reason": reason, "provider": accountcredentials.IdentityProviderSolanaWallet,
	}
	if err != nil {
		fields["error_type"] = fmt.Sprintf("%T", err)
	}
	log.WithFields(fields).Warn("Worm position Cash Out Solana authorization failed")
	writeWormPositionCashOutProofError(w, statusCode, reason)
}

func (h *wormPositionCashOutAuthorization) failInjected(
	w http.ResponseWriter,
	stage string,
	err error,
) {
	log.WithFields(log.Fields{
		"stage":      stage,
		"provider":   accountcredentials.IdentityProviderSolanaWallet,
		"error_type": fmt.Sprintf("%T", err),
	}).Warn("Worm position Cash Out Solana authorization dependency failed")
	h.writeError(w, err)
}

func canonicalWormPositionCashOutID(raw string, label string) (string, error) {
	if raw == "" || strings.TrimSpace(raw) != raw {
		return "", fmt.Errorf("%s must be a canonical non-zero UUID", label)
	}
	parsed, err := uuid.Parse(raw)
	if err != nil || parsed == uuid.Nil || parsed.String() != raw {
		return "", fmt.Errorf("%s must be a canonical non-zero UUID", label)
	}
	return raw, nil
}

func canonicalWormPositionCashOutDigest(raw string) ([]byte, error) {
	decoded, err := hex.DecodeString(raw)
	if err != nil || len(decoded) != sha256.Size || hex.EncodeToString(decoded) != raw {
		return nil, fmt.Errorf("digest must be canonical lowercase SHA-256 hex")
	}
	return decoded, nil
}

func wormPositionCashOutIDFromRequest(r *http.Request, suffix string) (string, error) {
	if r == nil {
		return "", fmt.Errorf("request is required")
	}
	cashOutID := r.PathValue("cashOutId")
	if cashOutID == "" {
		path := r.URL.Path
		if !strings.HasPrefix(path, wormPositionCashOutRoutePrefix) || !strings.HasSuffix(path, suffix) {
			return "", fmt.Errorf("Worm position Cash Out route is invalid")
		}
		cashOutID = strings.TrimSuffix(strings.TrimPrefix(path, wormPositionCashOutRoutePrefix), suffix)
		if strings.Contains(cashOutID, "/") {
			return "", fmt.Errorf("Worm position Cash Out route is invalid")
		}
	}
	return canonicalWormPositionCashOutID(cashOutID, "cashOutId")
}

func validateWormPositionCashOutReturnTo(raw string) string {
	if raw == "" {
		return wormPositionCashOutDefaultReturnTo
	}
	validated := authregistration.ReturnToForRealm(raw, accountcredentials.ApplicationRealmMember)
	if validated == authregistration.DefaultReturnTo && raw != authregistration.DefaultReturnTo {
		return wormPositionCashOutDefaultReturnTo
	}
	return validated
}

func setWormPositionCashOutProofHeaders(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-store, private")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Vary", "Cookie, Authorization")
	w.Header().Set("Referrer-Policy", "no-referrer")
}

func writeWormPositionCashOutProofError(w http.ResponseWriter, statusCode int, reason string) {
	w.Header().Set("X-Athena-Error-Reason", reason)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(errorResponse{Reason: reason})
}
