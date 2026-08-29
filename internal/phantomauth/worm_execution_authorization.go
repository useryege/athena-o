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
	// WormExecutionProofKindPhantom is the durable proof-kind value passed to
	// the Run authorization callback after SIWS verification.
	WormExecutionProofKindPhantom = "PHANTOM"

	wormExecutionChallengeCookieName  = "athena.worm-execution.solana.challenge"
	wormExecutionChallengeCookiePath  = "/auth/worm-trading/executions"
	wormExecutionRoutePrefix          = "/auth/worm-trading/executions/"
	wormExecutionChallengeRouteSuffix = "/solana/challenge"
	wormExecutionVerifyRouteSuffix    = "/solana/verify"
	wormExecutionDefaultReturnTo      = "/worm-trading/executions"

	// WormExecutionLoginSessionRequiredReason identifies a missing or stale
	// interactive Athena session during Run proof.
	WormExecutionLoginSessionRequiredReason = "WORM_EXECUTION_LOGIN_SESSION_REQUIRED"
	// WormExecutionAuthorizationRequiredReason identifies a rejected or stale
	// wallet identity proof.
	WormExecutionAuthorizationRequiredReason = "WORM_EXECUTION_AUTHORIZATION_REQUIRED"
	// WormExecutionAuthorizationUnavailableReason identifies a temporary proof
	// dependency failure.
	WormExecutionAuthorizationUnavailableReason = "WORM_EXECUTION_AUTHORIZATION_UNAVAILABLE"
)

type wormExecutionChallengeRequest struct {
	CommandID        string `json:"commandId"`
	ExpectedRevision int64  `json:"expectedRevision"`
	ReturnTo         string `json:"returnTo"`
}

type wormExecutionChallengeResponse struct {
	Message   string `json:"message"`
	ExpiresAt int64  `json:"expiresAt"`
}

type wormExecutionVerifyRequest struct {
	Signature string `json:"signature"`
}

// WormExecutionAuthenticator validates the Athena login cookie, attaches its
// typed credential to the returned context, and enforces Worm Trading
// READ_WRITE. API keys must be rejected by the caller.
type WormExecutionAuthenticator func(request *http.Request) (context.Context, accountcredentials.AuthenticatedCredential, error)

// WormExecutionDescriptorRequest is the authenticated binding used for a
// fresh authoritative Run lookup. SessionJTIDigestSHA256 is always exactly 32
// bytes.
type WormExecutionDescriptorRequest struct {
	RunID                  string
	CommandID              string
	ExpectedRevision       int64
	AccountID              string
	SessionJTIDigestSHA256 []byte
	AccessRevision         uint64
	ReturnTo               string
}

// WormExecutionDescriptor contains only the immutable fields required in the
// wallet-visible proof statement. PlanDigestSHA256 must be exactly 32 bytes.
type WormExecutionDescriptor struct {
	RunID            string
	Revision         int64
	PlanDigestSHA256 []byte
}

// WormExecutionDescriptorLoader loads the current owner-scoped Run and must
// reject a stale expected revision before returning its immutable plan digest.
type WormExecutionDescriptorLoader func(context.Context, WormExecutionDescriptorRequest) (WormExecutionDescriptor, error)

// WormExecutionAuthorizationRequest is emitted only after the exact stored
// SIWS message has been verified against the account's login Phantom address.
// It contains no signature or raw wallet proof.
type WormExecutionAuthorizationRequest struct {
	RunID                  string
	CommandID              string
	ExpectedRevision       int64
	AccountID              string
	SessionJTIDigestSHA256 []byte
	AccessRevision         uint64
	PlanDigestSHA256       []byte
	ReturnTo               string
	ProofKind              string
}

// WormExecutionAuthorizer persists the verified Run authorization and returns
// the public projection that the verify endpoint writes as JSON.
type WormExecutionAuthorizer func(context.Context, WormExecutionAuthorizationRequest) (any, error)

// WormExecutionErrorWriter projects injected descriptor and authorization
// errors without coupling this package to API Server or Worm Trading types.
type WormExecutionErrorWriter func(http.ResponseWriter, error)

type wormExecutionAuthorization struct {
	phantom        *Handler
	store          *wormExecutionChallengeStore
	authenticate   WormExecutionAuthenticator
	credentials    *accountcredentials.CredentialManager
	loadDescriptor WormExecutionDescriptorLoader
	authorize      WormExecutionAuthorizer
	writeError     WormExecutionErrorWriter
}

// EnableWormExecutionAuthorization attaches the independent Run-bound SIWS
// proof flow. The Redis challenge is short-lived and single-use; it is not the
// Run authorization and does not create or reuse the Worm credential lease.
func (h *Handler) EnableWormExecutionAuthorization(
	redisClient *redis.Client,
	authenticate WormExecutionAuthenticator,
	credentials *accountcredentials.CredentialManager,
	loadDescriptor WormExecutionDescriptorLoader,
	authorize WormExecutionAuthorizer,
	writeError WormExecutionErrorWriter,
) error {
	if h == nil || authenticate == nil || credentials == nil || loadDescriptor == nil || authorize == nil || writeError == nil {
		return fmt.Errorf("Solana Worm execution authorization dependencies are required")
	}
	if h.wormExecutions != nil {
		return fmt.Errorf("Solana Worm execution authorization is already enabled")
	}
	store, err := newWormExecutionChallengeStore(redisClient)
	if err != nil {
		return err
	}
	h.wormExecutions = &wormExecutionAuthorization{
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

// WormExecutionChallenge creates a dynamic, Run-bound SIWS message. The Run
// ID is read from the route path, never from JSON submitted by the browser.
func (h *Handler) WormExecutionChallenge(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.wormExecutions == nil {
		setWormExecutionProofHeaders(w)
		writeWormExecutionProofError(w, http.StatusServiceUnavailable, WormExecutionAuthorizationUnavailableReason)
		return
	}
	h.wormExecutions.challenge(w, r)
}

// WormExecutionVerify consumes and verifies one Run-bound challenge before
// returning the projection produced by the injected authorizer.
func (h *Handler) WormExecutionVerify(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.wormExecutions == nil {
		setWormExecutionProofHeaders(w)
		writeWormExecutionProofError(w, http.StatusServiceUnavailable, WormExecutionAuthorizationUnavailableReason)
		return
	}
	h.wormExecutions.verify(w, r)
}

func (h *wormExecutionAuthorization) challenge(w http.ResponseWriter, r *http.Request) {
	setWormExecutionProofHeaders(w)
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if !h.phantom.validOrigin(r) {
		h.fail(w, http.StatusForbidden, WormExecutionAuthorizationRequiredReason, "origin_verify", nil)
		return
	}
	if !isJSONRequest(r) {
		h.fail(w, http.StatusUnsupportedMediaType, WormExecutionAuthorizationRequiredReason, "content_type", nil)
		return
	}
	runID, err := wormExecutionRunIDFromRequest(r, wormExecutionChallengeRouteSuffix)
	if err != nil {
		h.fail(w, http.StatusBadRequest, WormExecutionAuthorizationRequiredReason, "run_id", nil)
		return
	}
	var input wormExecutionChallengeRequest
	if err := decodeJSON(w, r, &input); err != nil {
		h.fail(w, http.StatusBadRequest, WormExecutionAuthorizationRequiredReason, "challenge_decode", err)
		return
	}
	commandID, err := canonicalWormExecutionID(input.CommandID, "commandId")
	if err != nil || input.ExpectedRevision <= 0 {
		h.fail(w, http.StatusBadRequest, WormExecutionAuthorizationRequiredReason, "command_binding", err)
		return
	}
	returnTo := validateWormExecutionReturnTo(input.ReturnTo, runID)
	_, credential, err := h.authenticate(r)
	if err != nil || credential.Capability != accountcredentials.CapabilityLogin || credential.JTI == "" ||
		credential.AccessRevision == 0 || credential.AccessRevision > math.MaxInt64 {
		h.fail(w, http.StatusUnauthorized, WormExecutionLoginSessionRequiredReason, "login_session", nil)
		return
	}
	accountID, err := accountcredentials.CanonicalAccountID(credential.AccountID)
	if err != nil || accountID != credential.AccountID {
		h.fail(w, http.StatusUnauthorized, WormExecutionLoginSessionRequiredReason, "account_binding", err)
		return
	}
	account, err := h.credentials.Get(accountID)
	if err != nil || account.IdentityProvider != accountcredentials.IdentityProviderSolanaWallet || !account.HasExternalIdentity() {
		h.fail(w, http.StatusUnauthorized, WormExecutionAuthorizationRequiredReason, "identity_provider", err)
		return
	}
	address, err := accountcredentials.NormalizeIdentitySubject(accountcredentials.IdentityProviderSolanaWallet, account.IdentitySubject)
	if err != nil || address != account.IdentitySubject {
		h.fail(w, http.StatusUnauthorized, WormExecutionAuthorizationRequiredReason, "identity_address", err)
		return
	}
	sessionDigest := sha256.Sum256([]byte(credential.JTI))
	descriptorRequest := WormExecutionDescriptorRequest{
		RunID:                  runID,
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
	planDigest, err := validateWormExecutionDescriptor(descriptorRequest, descriptor)
	if err != nil {
		h.fail(w, http.StatusServiceUnavailable, WormExecutionAuthorizationUnavailableReason, "descriptor_validate", err)
		return
	}
	id, err := authregistration.RandomOpaqueValue()
	if err != nil {
		h.fail(w, http.StatusServiceUnavailable, WormExecutionAuthorizationUnavailableReason, "challenge_id_generation", err)
		return
	}
	nonce, err := randomNonce()
	if err != nil {
		h.fail(w, http.StatusServiceUnavailable, WormExecutionAuthorizationUnavailableReason, "nonce_generation", err)
		return
	}
	issuedAt := time.Now().UTC().Truncate(time.Second)
	expiresAt := issuedAt.Add(wormExecutionChallengeTTL)
	statement := wormExecutionSIWSStatement(runID, planDigest)
	message := h.phantom.siwsMessageWithStatement(address, statement, nonce, issuedAt, expiresAt)
	if err := h.store.create(r.Context(), id, wormExecutionChallenge{
		Address:                address,
		Message:                message,
		Nonce:                  nonce,
		ReturnTo:               returnTo,
		RunID:                  runID,
		CommandID:              commandID,
		ExpectedRevision:       input.ExpectedRevision,
		AccountID:              accountID,
		SessionJTIDigestSHA256: hex.EncodeToString(sessionDigest[:]),
		AccessRevision:         credential.AccessRevision,
		PlanDigestSHA256:       hex.EncodeToString(planDigest),
		CreatedAt:              issuedAt,
		ExpiresAt:              expiresAt,
	}); err != nil {
		h.fail(w, http.StatusServiceUnavailable, WormExecutionAuthorizationUnavailableReason, "challenge_create", err)
		return
	}
	h.setChallengeCookie(w, id, expiresAt)
	h.phantom.writeJSON(w, http.StatusOK, wormExecutionChallengeResponse{Message: message, ExpiresAt: expiresAt.Unix()})
}

func (h *wormExecutionAuthorization) verify(w http.ResponseWriter, r *http.Request) {
	setWormExecutionProofHeaders(w)
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if !h.phantom.validOrigin(r) {
		h.fail(w, http.StatusForbidden, WormExecutionAuthorizationRequiredReason, "origin_verify", nil)
		return
	}
	runID, err := wormExecutionRunIDFromRequest(r, wormExecutionVerifyRouteSuffix)
	if err != nil {
		h.fail(w, http.StatusBadRequest, WormExecutionAuthorizationRequiredReason, "run_id", nil)
		return
	}
	cookie, cookieErr := r.Cookie(wormExecutionChallengeCookieName)
	h.clearChallengeCookie(w)
	if cookieErr != nil || !authregistration.ValidOpaqueValue(cookie.Value) {
		h.fail(w, http.StatusUnauthorized, WormExecutionAuthorizationRequiredReason, "challenge_cookie", cookieErr)
		return
	}
	stored, err := h.store.consume(r.Context(), cookie.Value)
	if err != nil {
		if errors.Is(err, errWormExecutionChallengeUnavailable) {
			h.fail(w, http.StatusServiceUnavailable, WormExecutionAuthorizationUnavailableReason, "challenge_consume", err)
			return
		}
		h.fail(w, http.StatusUnauthorized, WormExecutionAuthorizationRequiredReason, "challenge_consume", err)
		return
	}
	if !authregistration.ConstantTimeEqual(stored.RunID, runID) {
		h.fail(w, http.StatusUnauthorized, WormExecutionAuthorizationRequiredReason, "run_binding", nil)
		return
	}
	_, credential, err := h.authenticate(r)
	if err != nil || credential.Capability != accountcredentials.CapabilityLogin || credential.JTI == "" ||
		credential.AccessRevision == 0 || credential.AccessRevision > math.MaxInt64 {
		h.fail(w, http.StatusUnauthorized, WormExecutionLoginSessionRequiredReason, "login_session", nil)
		return
	}
	currentSessionDigest := sha256.Sum256([]byte(credential.JTI))
	storedSessionDigest, digestErr := canonicalWormExecutionDigest(stored.SessionJTIDigestSHA256)
	if digestErr != nil || !authregistration.ConstantTimeEqual(stored.AccountID, credential.AccountID) ||
		subtle.ConstantTimeCompare(storedSessionDigest, currentSessionDigest[:]) != 1 ||
		stored.AccessRevision != credential.AccessRevision {
		h.fail(w, http.StatusUnauthorized, WormExecutionAuthorizationRequiredReason, "session_binding", digestErr)
		return
	}
	account, err := h.credentials.Get(credential.AccountID)
	if err != nil || account.IdentityProvider != accountcredentials.IdentityProviderSolanaWallet ||
		!authregistration.ConstantTimeEqual(account.IdentitySubject, stored.Address) {
		h.fail(w, http.StatusUnauthorized, WormExecutionAuthorizationRequiredReason, "identity_binding", err)
		return
	}
	planDigest, err := canonicalWormExecutionDigest(stored.PlanDigestSHA256)
	if err != nil {
		h.fail(w, http.StatusUnauthorized, WormExecutionAuthorizationRequiredReason, "plan_binding", err)
		return
	}
	descriptorRequest := WormExecutionDescriptorRequest{
		RunID:                  stored.RunID,
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
	currentPlanDigest, err := validateWormExecutionDescriptor(descriptorRequest, descriptor)
	if err != nil || subtle.ConstantTimeCompare(planDigest, currentPlanDigest) != 1 {
		h.fail(w, http.StatusUnauthorized, WormExecutionAuthorizationRequiredReason, "descriptor_binding", err)
		return
	}
	expectedStatement := wormExecutionSIWSStatement(stored.RunID, planDigest)
	expectedMessage := h.phantom.siwsMessageWithStatement(stored.Address, expectedStatement, stored.Nonce, stored.CreatedAt, stored.ExpiresAt)
	if !authregistration.ConstantTimeEqual(expectedMessage, stored.Message) {
		h.fail(w, http.StatusUnauthorized, WormExecutionAuthorizationRequiredReason, "challenge_binding", nil)
		return
	}
	if !isJSONRequest(r) {
		h.fail(w, http.StatusUnsupportedMediaType, WormExecutionAuthorizationRequiredReason, "content_type", nil)
		return
	}
	var input wormExecutionVerifyRequest
	if err := decodeJSON(w, r, &input); err != nil {
		h.fail(w, http.StatusBadRequest, WormExecutionAuthorizationRequiredReason, "signature_decode", err)
		return
	}
	signature, err := base64.RawURLEncoding.DecodeString(input.Signature)
	if err != nil || len(signature) != ed25519.SignatureSize || base64.RawURLEncoding.EncodeToString(signature) != input.Signature {
		h.fail(w, http.StatusForbidden, WormExecutionAuthorizationRequiredReason, "signature_encoding", err)
		return
	}
	publicKey, err := base58.Decode(stored.Address)
	if err != nil || len(publicKey) != ed25519.PublicKeySize ||
		!ed25519.Verify(ed25519.PublicKey(publicKey), []byte(stored.Message), signature) {
		h.fail(w, http.StatusForbidden, WormExecutionAuthorizationRequiredReason, "signature_verify", err)
		return
	}
	projection, err := h.authorize(r.Context(), WormExecutionAuthorizationRequest{
		RunID:                  stored.RunID,
		CommandID:              stored.CommandID,
		ExpectedRevision:       stored.ExpectedRevision,
		AccountID:              stored.AccountID,
		SessionJTIDigestSHA256: append([]byte(nil), currentSessionDigest[:]...),
		AccessRevision:         stored.AccessRevision,
		PlanDigestSHA256:       append([]byte(nil), planDigest...),
		ReturnTo:               stored.ReturnTo,
		ProofKind:              WormExecutionProofKindPhantom,
	})
	if err != nil {
		h.failInjected(w, "authorize", err)
		return
	}
	log.WithFields(log.Fields{"stage": "complete", "provider": accountcredentials.IdentityProviderSolanaWallet}).Info("Worm execution Solana authorization succeeded")
	h.phantom.writeJSON(w, http.StatusOK, projection)
}

func validateWormExecutionDescriptor(request WormExecutionDescriptorRequest, descriptor WormExecutionDescriptor) ([]byte, error) {
	runID, err := canonicalWormExecutionID(descriptor.RunID, "descriptor runId")
	if err != nil || !authregistration.ConstantTimeEqual(runID, request.RunID) || descriptor.Revision != request.ExpectedRevision {
		return nil, fmt.Errorf("Worm execution descriptor binding is invalid")
	}
	if len(descriptor.PlanDigestSHA256) != sha256.Size {
		return nil, fmt.Errorf("Worm execution descriptor plan digest is invalid")
	}
	return append([]byte(nil), descriptor.PlanDigestSHA256...), nil
}

func wormExecutionSIWSStatement(runID string, planDigest []byte) string {
	return fmt.Sprintf(
		"Authorize Athena to execute Worm Run ID %s with plan SHA-256 %x. Athena will sign the original transactions returned by Worm for this Run. The 10 USDC per-order limit is only Athena's requested funds maximum sent to Worm; it is not a cryptographic limit on the transaction's on-chain spending.",
		runID,
		planDigest,
	)
}

func (h *wormExecutionAuthorization) setChallengeCookie(w http.ResponseWriter, value string, expiresAt time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     wormExecutionChallengeCookieName,
		Value:    value,
		Path:     wormExecutionChallengeCookiePath,
		MaxAge:   int(wormExecutionChallengeTTL / time.Second),
		Expires:  expiresAt,
		HttpOnly: true,
		Secure:   h.phantom.secureCookie,
		SameSite: http.SameSiteStrictMode,
	})
}

func (h *wormExecutionAuthorization) clearChallengeCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     wormExecutionChallengeCookieName,
		Value:    "",
		Path:     wormExecutionChallengeCookiePath,
		MaxAge:   -1,
		Expires:  time.Unix(1, 0),
		HttpOnly: true,
		Secure:   h.phantom.secureCookie,
		SameSite: http.SameSiteStrictMode,
	})
}

func (h *wormExecutionAuthorization) fail(w http.ResponseWriter, statusCode int, reason, stage string, err error) {
	fields := log.Fields{"stage": stage, "reason": reason, "provider": accountcredentials.IdentityProviderSolanaWallet}
	if err != nil {
		fields["error_type"] = fmt.Sprintf("%T", err)
	}
	log.WithFields(fields).Warn("Worm execution Solana authorization failed")
	writeWormExecutionProofError(w, statusCode, reason)
}

func (h *wormExecutionAuthorization) failInjected(w http.ResponseWriter, stage string, err error) {
	log.WithFields(log.Fields{
		"stage":      stage,
		"provider":   accountcredentials.IdentityProviderSolanaWallet,
		"error_type": fmt.Sprintf("%T", err),
	}).Warn("Worm execution Solana authorization dependency failed")
	h.writeError(w, err)
}

func canonicalWormExecutionID(raw, label string) (string, error) {
	if raw == "" || strings.TrimSpace(raw) != raw {
		return "", fmt.Errorf("%s must be a canonical non-zero UUID", label)
	}
	parsed, err := uuid.Parse(raw)
	if err != nil || parsed == uuid.Nil || parsed.String() != raw {
		return "", fmt.Errorf("%s must be a canonical non-zero UUID", label)
	}
	return raw, nil
}

func canonicalWormExecutionDigest(raw string) ([]byte, error) {
	decoded, err := hex.DecodeString(raw)
	if err != nil || len(decoded) != sha256.Size || hex.EncodeToString(decoded) != raw {
		return nil, fmt.Errorf("digest must be canonical lowercase SHA-256 hex")
	}
	return decoded, nil
}

func wormExecutionRunIDFromRequest(r *http.Request, suffix string) (string, error) {
	if r == nil {
		return "", fmt.Errorf("request is required")
	}
	runID := r.PathValue("runId")
	if runID == "" {
		path := r.URL.Path
		if !strings.HasPrefix(path, wormExecutionRoutePrefix) || !strings.HasSuffix(path, suffix) {
			return "", fmt.Errorf("Worm execution route is invalid")
		}
		runID = strings.TrimSuffix(strings.TrimPrefix(path, wormExecutionRoutePrefix), suffix)
		if strings.Contains(runID, "/") {
			return "", fmt.Errorf("Worm execution route is invalid")
		}
	}
	return canonicalWormExecutionID(runID, "runId")
}

func validateWormExecutionReturnTo(raw, runID string) string {
	fallback := wormExecutionDefaultReturnTo
	if canonical, err := canonicalWormExecutionID(runID, "runId"); err == nil {
		fallback += "/" + canonical
	}
	if raw == "" {
		return fallback
	}
	validated := authregistration.ValidateReturnTo(raw)
	if validated == authregistration.DefaultReturnTo && raw != authregistration.DefaultReturnTo {
		return fallback
	}
	return validated
}

func setWormExecutionProofHeaders(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-store, private")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Vary", "Cookie, Authorization")
	w.Header().Set("Referrer-Policy", "no-referrer")
}

func writeWormExecutionProofError(w http.ResponseWriter, statusCode int, reason string) {
	w.Header().Set("X-Athena-Error-Reason", reason)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(errorResponse{Reason: reason})
}
