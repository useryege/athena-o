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
	"github.com/useryege/athena/internal/walletsecret"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mr-tron/base58/base58"
	"github.com/redis/go-redis/v9"
	log "github.com/sirupsen/logrus"

	"github.com/useryege/athena/internal/accountcredentials"
	"github.com/useryege/athena/internal/authregistration"
	operationlogrecord "github.com/useryege/athena/internal/operationlog/record"
)

const (
	WormPositionCashOutBatchProofKindPhantom = "PHANTOM"

	wormPositionCashOutBatchChallengeCookieName  = "athena.worm-position-cash-out-batch.solana.challenge"
	wormPositionCashOutBatchChallengeCookiePath  = "/auth/worm-trading/position-cash-out-batches"
	wormPositionCashOutBatchRoutePrefix          = "/auth/worm-trading/position-cash-out-batches/"
	wormPositionCashOutBatchChallengeRouteSuffix = "/solana/challenge"
	wormPositionCashOutBatchVerifyRouteSuffix    = "/solana/verify"
	wormPositionCashOutBatchDefaultReturnTo      = "/worm-trading"

	WormPositionCashOutBatchLoginSessionRequiredReason     = "WORM_POSITION_CASH_OUT_BATCH_LOGIN_SESSION_REQUIRED"
	WormPositionCashOutBatchAuthorizationRequiredReason    = "WORM_POSITION_CASH_OUT_BATCH_AUTHORIZATION_REQUIRED"
	WormPositionCashOutBatchAuthorizationUnavailableReason = "WORM_POSITION_CASH_OUT_BATCH_AUTHORIZATION_UNAVAILABLE"
)

type wormPositionCashOutBatchChallengeRequest struct {
	CommandID        string `json:"commandId"`
	ExpectedRevision int64  `json:"expectedRevision"`
	ReturnTo         string `json:"returnTo"`
}

type wormPositionCashOutBatchChallengeResponse struct {
	Message   string `json:"message"`
	ExpiresAt int64  `json:"expiresAt"`
}

type wormPositionCashOutBatchVerifyRequest struct {
	Signature string `json:"signature"`
}

type WormPositionCashOutBatchAuthenticator func(
	request *http.Request,
) (context.Context, accountcredentials.AuthenticatedCredential, error)

type WormPositionCashOutBatchDescriptorRequest struct {
	BatchID                string
	CommandID              string
	ExpectedRevision       int64
	AccountID              string
	SessionJTIDigestSHA256 []byte
	AccessRevision         uint64
	ReturnTo               string
}

type WormPositionCashOutBatchDescriptor struct {
	BatchID            string
	Revision           int64
	IntentDigestSHA256 []byte
}

type WormPositionCashOutBatchDescriptorLoader func(
	context.Context,
	WormPositionCashOutBatchDescriptorRequest,
) (WormPositionCashOutBatchDescriptor, error)

type WormPositionCashOutBatchAuthorizationRequest struct {
	BatchID                string
	CommandID              string
	ExpectedRevision       int64
	AccountID              string
	SessionJTIDigestSHA256 []byte
	AccessRevision         uint64
	IntentDigestSHA256     []byte
	ReturnTo               string
	ProofKind              string
}

type WormPositionCashOutBatchAuthorizer func(
	context.Context,
	WormPositionCashOutBatchAuthorizationRequest,
) (any, error)

type WormPositionCashOutBatchErrorWriter func(http.ResponseWriter, error)

type wormPositionCashOutBatchAuthorization struct {
	phantom        *Handler
	store          *wormPositionCashOutBatchChallengeStore
	authenticate   WormPositionCashOutBatchAuthenticator
	admit          func(context.Context) error
	credentials    *accountcredentials.CredentialManager
	loadDescriptor WormPositionCashOutBatchDescriptorLoader
	authorize      WormPositionCashOutBatchAuthorizer
	writeError     WormPositionCashOutBatchErrorWriter
}

// EnableWormPositionCashOutBatchAuthorization attaches the independent,
// operation-bound Phantom identity proof.
func (h *Handler) EnableWormPositionCashOutBatchAuthorization(
	redisClient *redis.Client,
	authenticate WormPositionCashOutBatchAuthenticator,
	admit func(context.Context) error,
	credentials *accountcredentials.CredentialManager,
	loadDescriptor WormPositionCashOutBatchDescriptorLoader,
	authorize WormPositionCashOutBatchAuthorizer,
	writeError WormPositionCashOutBatchErrorWriter,
) error {
	if h == nil || authenticate == nil || admit == nil || credentials == nil || loadDescriptor == nil ||
		authorize == nil || writeError == nil {
		return fmt.Errorf("Solana Worm position Cash Out Batch authorization dependencies are required")
	}
	if h.wormPositionCashOutBatches != nil {
		return fmt.Errorf("Solana Worm position Cash Out Batch authorization is already enabled")
	}
	store, err := newWormPositionCashOutBatchChallengeStore(redisClient)
	if err != nil {
		return err
	}
	h.wormPositionCashOutBatches = &wormPositionCashOutBatchAuthorization{
		phantom:        h,
		store:          store,
		authenticate:   authenticate,
		admit:          admit,
		credentials:    credentials,
		loadDescriptor: loadDescriptor,
		authorize:      authorize,
		writeError:     writeError,
	}
	return nil
}

func (h *Handler) WormPositionCashOutBatchChallenge(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.wormPositionCashOutBatches == nil {
		setWormPositionCashOutBatchProofHeaders(w)
		writeWormPositionCashOutBatchProofError(
			w,
			http.StatusServiceUnavailable,
			WormPositionCashOutBatchAuthorizationUnavailableReason,
		)
		return
	}
	h.wormPositionCashOutBatches.challenge(w, r)
}

func (h *Handler) WormPositionCashOutBatchVerify(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.wormPositionCashOutBatches == nil {
		setWormPositionCashOutBatchProofHeaders(w)
		writeWormPositionCashOutBatchProofError(
			w,
			http.StatusServiceUnavailable,
			WormPositionCashOutBatchAuthorizationUnavailableReason,
		)
		return
	}
	h.wormPositionCashOutBatches.verify(w, r)
}

func (h *wormPositionCashOutBatchAuthorization) challenge(w http.ResponseWriter, r *http.Request) {
	setWormPositionCashOutBatchProofHeaders(w)
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if !h.phantom.validOrigin(r) {
		h.fail(r.Context(), w, http.StatusForbidden, WormPositionCashOutBatchAuthorizationRequiredReason, "origin_verify", nil)
		return
	}
	if !isJSONRequest(r) {
		h.fail(r.Context(), w, http.StatusUnsupportedMediaType, WormPositionCashOutBatchAuthorizationRequiredReason, "content_type", nil)
		return
	}
	batchID, err := wormPositionCashOutBatchIDFromRequest(r, wormPositionCashOutBatchChallengeRouteSuffix)
	if err != nil {
		h.fail(r.Context(), w, http.StatusBadRequest, WormPositionCashOutBatchAuthorizationRequiredReason, "batch_id", err)
		return
	}
	var input wormPositionCashOutBatchChallengeRequest
	if err := decodeJSON(w, r, &input); err != nil {
		h.fail(r.Context(), w, http.StatusBadRequest, WormPositionCashOutBatchAuthorizationRequiredReason, "challenge_decode", err)
		return
	}
	commandID, err := canonicalWormPositionCashOutBatchID(input.CommandID, "commandId")
	if err != nil || input.ExpectedRevision <= 0 {
		h.fail(r.Context(), w, http.StatusBadRequest, WormPositionCashOutBatchAuthorizationRequiredReason, "command_binding", err)
		return
	}
	returnTo := validateWormPositionCashOutBatchReturnTo(input.ReturnTo)
	authCtx, credential, err := h.authenticate(r)
	if err != nil || credential.Capability != accountcredentials.CapabilityLogin || credential.JTI == "" ||
		credential.AccessRevision == 0 || credential.AccessRevision > math.MaxInt64 {
		h.fail(r.Context(), w, http.StatusUnauthorized, WormPositionCashOutBatchLoginSessionRequiredReason, "login_session", nil)
		return
	}
	if err := h.admit(authCtx); err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	accountID, err := accountcredentials.CanonicalAccountID(credential.AccountID)
	if err != nil || accountID != credential.AccountID {
		h.fail(r.Context(), w, http.StatusUnauthorized, WormPositionCashOutBatchLoginSessionRequiredReason, "account_binding", err)
		return
	}
	account, err := h.credentials.Get(accountID)
	if err != nil || account.IdentityProvider != accountcredentials.IdentityProviderSolanaWallet ||
		!account.HasExternalIdentity() {
		h.fail(r.Context(), w, http.StatusUnauthorized, WormPositionCashOutBatchAuthorizationRequiredReason, "identity_provider", err)
		return
	}
	address, err := accountcredentials.NormalizeIdentitySubject(
		accountcredentials.IdentityProviderSolanaWallet,
		account.IdentitySubject,
	)
	if err != nil || address != account.IdentitySubject {
		h.fail(r.Context(), w, http.StatusUnauthorized, WormPositionCashOutBatchAuthorizationRequiredReason, "identity_address", err)
		return
	}
	sessionDigest := sha256.Sum256([]byte(credential.JTI))
	descriptorRequest := WormPositionCashOutBatchDescriptorRequest{
		BatchID:                batchID,
		CommandID:              commandID,
		ExpectedRevision:       input.ExpectedRevision,
		AccountID:              accountID,
		SessionJTIDigestSHA256: append([]byte(nil), sessionDigest[:]...),
		AccessRevision:         credential.AccessRevision,
		ReturnTo:               returnTo,
	}
	descriptor, err := h.loadDescriptor(r.Context(), descriptorRequest)
	if err != nil {
		h.failInjected(r.Context(), w, "descriptor_load", err)
		return
	}
	intentDigest, err := validateWormPositionCashOutBatchDescriptor(descriptorRequest, descriptor)
	if err != nil {
		h.fail(r.Context(), w, http.StatusServiceUnavailable, WormPositionCashOutBatchAuthorizationUnavailableReason, "descriptor_validate", err)
		return
	}
	id, err := authregistration.RandomOpaqueValue()
	if err != nil {
		h.fail(r.Context(), w, http.StatusServiceUnavailable, WormPositionCashOutBatchAuthorizationUnavailableReason, "challenge_id_generation", err)
		return
	}
	nonce, err := randomNonce()
	if err != nil {
		h.fail(r.Context(), w, http.StatusServiceUnavailable, WormPositionCashOutBatchAuthorizationUnavailableReason, "nonce_generation", err)
		return
	}
	issuedAt := time.Now().UTC().Truncate(time.Second)
	expiresAt := issuedAt.Add(wormPositionCashOutBatchChallengeTTL)
	statement := wormPositionCashOutBatchSIWSStatement(batchID, intentDigest)
	message := h.phantom.siwsMessageWithStatement(address, statement, nonce, issuedAt, expiresAt)
	if err := h.store.create(r.Context(), id, wormPositionCashOutBatchChallenge{
		Address:                address,
		Message:                message,
		Nonce:                  nonce,
		ReturnTo:               returnTo,
		BatchID:                batchID,
		CommandID:              commandID,
		ExpectedRevision:       input.ExpectedRevision,
		AccountID:              accountID,
		SessionJTIDigestSHA256: hex.EncodeToString(sessionDigest[:]),
		AccessRevision:         credential.AccessRevision,
		IntentDigestSHA256:     hex.EncodeToString(intentDigest),
		CreatedAt:              issuedAt,
		ExpiresAt:              expiresAt,
	}); err != nil {
		h.fail(r.Context(), w, http.StatusServiceUnavailable, WormPositionCashOutBatchAuthorizationUnavailableReason, "challenge_create", err)
		return
	}
	h.setChallengeCookie(w, id, expiresAt)
	h.phantom.writeJSON(w, http.StatusOK, wormPositionCashOutBatchChallengeResponse{
		Message: message, ExpiresAt: expiresAt.Unix(),
	})
}

func (h *wormPositionCashOutBatchAuthorization) verify(w http.ResponseWriter, r *http.Request) {
	setWormPositionCashOutBatchProofHeaders(w)
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if !h.phantom.validOrigin(r) {
		h.fail(r.Context(), w, http.StatusForbidden, WormPositionCashOutBatchAuthorizationRequiredReason, "origin_verify", nil)
		return
	}
	batchID, err := wormPositionCashOutBatchIDFromRequest(r, wormPositionCashOutBatchVerifyRouteSuffix)
	if err != nil {
		h.fail(r.Context(), w, http.StatusBadRequest, WormPositionCashOutBatchAuthorizationRequiredReason, "batch_id", err)
		return
	}
	cookie, cookieErr := r.Cookie(wormPositionCashOutBatchChallengeCookieName)
	h.clearChallengeCookie(w)
	if cookieErr != nil || !authregistration.ValidOpaqueValue(cookie.Value) {
		h.fail(r.Context(), w, http.StatusUnauthorized, WormPositionCashOutBatchAuthorizationRequiredReason, "challenge_cookie", cookieErr)
		return
	}
	stored, err := h.store.consume(r.Context(), cookie.Value)
	if err != nil {
		if errors.Is(err, errWormPositionCashOutBatchChallengeUnavailable) {
			h.fail(r.Context(), w, http.StatusServiceUnavailable, WormPositionCashOutBatchAuthorizationUnavailableReason, "challenge_consume", err)
			return
		}
		h.fail(r.Context(), w, http.StatusUnauthorized, WormPositionCashOutBatchAuthorizationRequiredReason, "challenge_consume", err)
		return
	}
	if !authregistration.ConstantTimeEqual(stored.BatchID, batchID) {
		h.fail(r.Context(), w, http.StatusUnauthorized, WormPositionCashOutBatchAuthorizationRequiredReason, "cash_out_binding", nil)
		return
	}
	authCtx, credential, err := h.authenticate(r)
	if err != nil || credential.Capability != accountcredentials.CapabilityLogin || credential.JTI == "" ||
		credential.AccessRevision == 0 || credential.AccessRevision > math.MaxInt64 {
		h.fail(r.Context(), w, http.StatusUnauthorized, WormPositionCashOutBatchLoginSessionRequiredReason, "login_session", nil)
		return
	}
	currentSessionDigest := sha256.Sum256([]byte(credential.JTI))
	storedSessionDigest, digestErr := canonicalWormPositionCashOutBatchDigest(stored.SessionJTIDigestSHA256)
	if digestErr != nil || !authregistration.ConstantTimeEqual(stored.AccountID, credential.AccountID) ||
		subtle.ConstantTimeCompare(storedSessionDigest, currentSessionDigest[:]) != 1 ||
		stored.AccessRevision != credential.AccessRevision {
		h.fail(r.Context(), w, http.StatusUnauthorized, WormPositionCashOutBatchAuthorizationRequiredReason, "session_binding", digestErr)
		return
	}
	if err := h.admit(authCtx); err != nil {
		walletsecret.WriteError(w, err)
		return
	}
	account, err := h.credentials.Get(credential.AccountID)
	if err != nil || account.IdentityProvider != accountcredentials.IdentityProviderSolanaWallet ||
		!authregistration.ConstantTimeEqual(account.IdentitySubject, stored.Address) {
		h.fail(r.Context(), w, http.StatusUnauthorized, WormPositionCashOutBatchAuthorizationRequiredReason, "identity_binding", err)
		return
	}
	intentDigest, err := canonicalWormPositionCashOutBatchDigest(stored.IntentDigestSHA256)
	if err != nil {
		h.fail(r.Context(), w, http.StatusUnauthorized, WormPositionCashOutBatchAuthorizationRequiredReason, "intent_binding", err)
		return
	}
	descriptorRequest := WormPositionCashOutBatchDescriptorRequest{
		BatchID:                stored.BatchID,
		CommandID:              stored.CommandID,
		ExpectedRevision:       stored.ExpectedRevision,
		AccountID:              stored.AccountID,
		SessionJTIDigestSHA256: append([]byte(nil), currentSessionDigest[:]...),
		AccessRevision:         stored.AccessRevision,
		ReturnTo:               stored.ReturnTo,
	}
	descriptor, err := h.loadDescriptor(r.Context(), descriptorRequest)
	if err != nil {
		h.failInjected(r.Context(), w, "descriptor_reload", err)
		return
	}
	currentIntentDigest, err := validateWormPositionCashOutBatchDescriptor(descriptorRequest, descriptor)
	if err != nil || subtle.ConstantTimeCompare(intentDigest, currentIntentDigest) != 1 {
		h.fail(r.Context(), w, http.StatusUnauthorized, WormPositionCashOutBatchAuthorizationRequiredReason, "descriptor_binding", err)
		return
	}
	expectedStatement := wormPositionCashOutBatchSIWSStatement(stored.BatchID, intentDigest)
	expectedMessage := h.phantom.siwsMessageWithStatement(
		stored.Address,
		expectedStatement,
		stored.Nonce,
		stored.CreatedAt,
		stored.ExpiresAt,
	)
	if !authregistration.ConstantTimeEqual(expectedMessage, stored.Message) {
		h.fail(r.Context(), w, http.StatusUnauthorized, WormPositionCashOutBatchAuthorizationRequiredReason, "challenge_binding", nil)
		return
	}
	if !isJSONRequest(r) {
		h.fail(r.Context(), w, http.StatusUnsupportedMediaType, WormPositionCashOutBatchAuthorizationRequiredReason, "content_type", nil)
		return
	}
	var input wormPositionCashOutBatchVerifyRequest
	if err := decodeJSON(w, r, &input); err != nil {
		h.fail(r.Context(), w, http.StatusBadRequest, WormPositionCashOutBatchAuthorizationRequiredReason, "signature_decode", err)
		return
	}
	signature, err := base64.RawURLEncoding.DecodeString(input.Signature)
	if err != nil || len(signature) != ed25519.SignatureSize ||
		base64.RawURLEncoding.EncodeToString(signature) != input.Signature {
		h.fail(r.Context(), w, http.StatusForbidden, WormPositionCashOutBatchAuthorizationRequiredReason, "signature_encoding", err)
		return
	}
	publicKey, err := base58.Decode(stored.Address)
	if err != nil || len(publicKey) != ed25519.PublicKeySize ||
		!ed25519.Verify(ed25519.PublicKey(publicKey), []byte(stored.Message), signature) {
		h.fail(r.Context(), w, http.StatusForbidden, WormPositionCashOutBatchAuthorizationRequiredReason, "signature_verify", err)
		return
	}
	projection, err := h.authorize(r.Context(), WormPositionCashOutBatchAuthorizationRequest{
		BatchID:                stored.BatchID,
		CommandID:              stored.CommandID,
		ExpectedRevision:       stored.ExpectedRevision,
		AccountID:              stored.AccountID,
		SessionJTIDigestSHA256: append([]byte(nil), currentSessionDigest[:]...),
		AccessRevision:         stored.AccessRevision,
		IntentDigestSHA256:     append([]byte(nil), intentDigest...),
		ReturnTo:               stored.ReturnTo,
		ProofKind:              WormPositionCashOutBatchProofKindPhantom,
	})
	if err != nil {
		h.failInjected(r.Context(), w, "authorize", err)
		return
	}
	log.WithFields(log.Fields{
		"stage": "complete", "provider": accountcredentials.IdentityProviderSolanaWallet,
	}).Info("Worm position Cash Out Batch Solana authorization succeeded")
	operationlogrecord.CaptureResource(r.Context(), "cash_out_batch", stored.BatchID)
	operationlogrecord.BindVerifiedAccount(r.Context(), credential.AccountID, "solana_wallet")
	operationlogrecord.CaptureString(r.Context(), "provider", "solana_wallet")
	operationlogrecord.CaptureString(r.Context(), "proofKind", "PHANTOM")
	operationlogrecord.CaptureString(r.Context(), "expectedRevision", strconv.FormatInt(stored.ExpectedRevision, 10))
	operationlogrecord.CaptureString(r.Context(), "confirmedRevision", strconv.FormatInt(descriptor.Revision, 10))
	operationlogrecord.CaptureString(r.Context(), "stage", "authorization_verified")
	operationlogrecord.Commit(r.Context(), "WORM_CASH_OUT_BATCH_AUTHORIZE")
	h.phantom.writeJSON(w, http.StatusOK, projection)
}

func validateWormPositionCashOutBatchDescriptor(
	request WormPositionCashOutBatchDescriptorRequest,
	descriptor WormPositionCashOutBatchDescriptor,
) ([]byte, error) {
	batchID, err := canonicalWormPositionCashOutBatchID(descriptor.BatchID, "descriptor batchId")
	if err != nil || !authregistration.ConstantTimeEqual(batchID, request.BatchID) ||
		descriptor.Revision != request.ExpectedRevision || len(descriptor.IntentDigestSHA256) != sha256.Size {
		return nil, fmt.Errorf("Worm position Cash Out Batch descriptor binding is invalid")
	}
	return append([]byte(nil), descriptor.IntentDigestSHA256...), nil
}

func wormPositionCashOutBatchSIWSStatement(batchID string, intentDigest []byte) string {
	return fmt.Sprintf(
		"Authorize Athena to serially cash out the frozen Worm position batch ID %s with intent SHA-256 %x. This is an identity confirmation only; it does not sign a blockchain transaction or authorize a network fee.",
		batchID,
		intentDigest,
	)
}

func (h *wormPositionCashOutBatchAuthorization) setChallengeCookie(
	w http.ResponseWriter,
	value string,
	expiresAt time.Time,
) {
	http.SetCookie(w, &http.Cookie{
		Name:     wormPositionCashOutBatchChallengeCookieName,
		Value:    value,
		Path:     h.phantom.deploymentPath(wormPositionCashOutBatchChallengeCookiePath),
		MaxAge:   int(wormPositionCashOutBatchChallengeTTL / time.Second),
		Expires:  expiresAt,
		HttpOnly: true,
		Secure:   h.phantom.secureCookie,
		SameSite: http.SameSiteStrictMode,
	})
}

func (h *wormPositionCashOutBatchAuthorization) clearChallengeCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     wormPositionCashOutBatchChallengeCookieName,
		Value:    "",
		Path:     h.phantom.deploymentPath(wormPositionCashOutBatchChallengeCookiePath),
		MaxAge:   -1,
		Expires:  time.Unix(1, 0),
		HttpOnly: true,
		Secure:   h.phantom.secureCookie,
		SameSite: http.SameSiteStrictMode,
	})
}

func (h *wormPositionCashOutBatchAuthorization) fail(
	ctx context.Context,
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
	log.WithFields(fields).Warn("Worm position Cash Out Batch Solana authorization failed")
	observeAuthorizationFailure(ctx, stage, statusCode, reason)
	writeWormPositionCashOutBatchProofError(w, statusCode, reason)
}

func (h *wormPositionCashOutBatchAuthorization) failInjected(
	ctx context.Context,
	w http.ResponseWriter,
	stage string,
	err error,
) {
	log.WithFields(log.Fields{
		"stage":      stage,
		"provider":   accountcredentials.IdentityProviderSolanaWallet,
		"error_type": fmt.Sprintf("%T", err),
	}).Warn("Worm position Cash Out Batch Solana authorization dependency failed")
	observeAuthorizationFailure(ctx, stage, http.StatusInternalServerError, "AUTHORIZATION_UNAVAILABLE")
	h.writeError(w, err)
}

func canonicalWormPositionCashOutBatchID(raw string, label string) (string, error) {
	if raw == "" || strings.TrimSpace(raw) != raw {
		return "", fmt.Errorf("%s must be a canonical non-zero UUID", label)
	}
	parsed, err := uuid.Parse(raw)
	if err != nil || parsed == uuid.Nil || parsed.String() != raw {
		return "", fmt.Errorf("%s must be a canonical non-zero UUID", label)
	}
	return raw, nil
}

func canonicalWormPositionCashOutBatchDigest(raw string) ([]byte, error) {
	decoded, err := hex.DecodeString(raw)
	if err != nil || len(decoded) != sha256.Size || hex.EncodeToString(decoded) != raw {
		return nil, fmt.Errorf("digest must be canonical lowercase SHA-256 hex")
	}
	return decoded, nil
}

func wormPositionCashOutBatchIDFromRequest(r *http.Request, suffix string) (string, error) {
	if r == nil {
		return "", fmt.Errorf("request is required")
	}
	batchID := r.PathValue("batchId")
	if batchID == "" {
		path := r.URL.Path
		if !strings.HasPrefix(path, wormPositionCashOutBatchRoutePrefix) || !strings.HasSuffix(path, suffix) {
			return "", fmt.Errorf("Worm position Cash Out Batch route is invalid")
		}
		batchID = strings.TrimSuffix(strings.TrimPrefix(path, wormPositionCashOutBatchRoutePrefix), suffix)
		if strings.Contains(batchID, "/") {
			return "", fmt.Errorf("Worm position Cash Out Batch route is invalid")
		}
	}
	return canonicalWormPositionCashOutBatchID(batchID, "batchId")
}

func validateWormPositionCashOutBatchReturnTo(raw string) string {
	if raw == "" {
		return wormPositionCashOutBatchDefaultReturnTo
	}
	validated := authregistration.ReturnToForRealm(raw, accountcredentials.ApplicationRealmMember)
	if validated == authregistration.DefaultReturnTo && raw != authregistration.DefaultReturnTo {
		return wormPositionCashOutBatchDefaultReturnTo
	}
	return validated
}

func setWormPositionCashOutBatchProofHeaders(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-store, private")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Vary", "Cookie, Authorization")
	w.Header().Set("Referrer-Policy", "no-referrer")
}

func writeWormPositionCashOutBatchProofError(w http.ResponseWriter, statusCode int, reason string) {
	w.Header().Set("X-Athena-Error-Reason", reason)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(errorResponse{Reason: reason})
}
