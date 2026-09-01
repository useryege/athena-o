package googleoidc

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	log "github.com/sirupsen/logrus"
	"golang.org/x/oauth2"

	"github.com/useryege/athena/internal/accountcredentials"
	"github.com/useryege/athena/internal/authregistration"
)

const (
	// WormPositionCashOutBatchProofKindGoogle is the durable proof kind passed to
	// the operation authorizer after fresh OIDC verification.
	WormPositionCashOutBatchProofKindGoogle = "GOOGLE"

	wormPositionCashOutBatchStateCookieName = "athena.worm-position-cash-out-batch.google.state"
	wormPositionCashOutBatchDefaultReturnTo = "/worm-trading"

	WormPositionCashOutBatchLoginSessionRequiredReason     = "WORM_POSITION_CASH_OUT_BATCH_LOGIN_SESSION_REQUIRED"
	WormPositionCashOutBatchAuthorizationRequiredReason    = "WORM_POSITION_CASH_OUT_BATCH_AUTHORIZATION_REQUIRED"
	WormPositionCashOutBatchAuthorizationUnavailableReason = "WORM_POSITION_CASH_OUT_BATCH_AUTHORIZATION_UNAVAILABLE"
)

// WormPositionCashOutBatchAuthenticator validates the Athena login cookie,
// attaches its typed credential, and enforces interactive Worm Trading
// READ_WRITE access. API keys must be rejected by the caller.
type WormPositionCashOutBatchAuthenticator func(
	request *http.Request,
) (context.Context, accountcredentials.AuthenticatedCredential, error)

// WormPositionCashOutBatchDescriptorRequest binds a fresh owner-scoped operation
// lookup to the current login Session and access revision.
type WormPositionCashOutBatchDescriptorRequest struct {
	BatchID                string
	CommandID              string
	ExpectedRevision       int64
	AccountID              string
	SessionJTIDigestSHA256 []byte
	AccessRevision         uint64
	ReturnTo               string
}

// WormPositionCashOutBatchDescriptor contains the immutable authorization identity.
// IntentDigestSHA256 must be exactly 32 bytes.
type WormPositionCashOutBatchDescriptor struct {
	BatchID            string
	Revision           int64
	IntentDigestSHA256 []byte
}

// WormPositionCashOutBatchDescriptorLoader performs the current owner/revision
// lookup and returns the immutable Cash Out intent digest.
type WormPositionCashOutBatchDescriptorLoader func(
	context.Context,
	WormPositionCashOutBatchDescriptorRequest,
) (WormPositionCashOutBatchDescriptor, error)

// WormPositionCashOutBatchAuthorizationRequest is emitted only after the current
// Google identity and immutable operation descriptor have both been verified.
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
	google         *Handler
	store          *wormPositionCashOutBatchTransactionStore
	authenticate   WormPositionCashOutBatchAuthenticator
	credentials    *accountcredentials.CredentialManager
	loadDescriptor WormPositionCashOutBatchDescriptorLoader
	authorize      WormPositionCashOutBatchAuthorizer
	writeError     WormPositionCashOutBatchErrorWriter
}

// EnableWormPositionCashOutBatchAuthorization attaches the independent, single-use
// fresh-Google proof for one immutable Cash Out operation.
func (h *Handler) EnableWormPositionCashOutBatchAuthorization(
	redisClient *redis.Client,
	authenticate WormPositionCashOutBatchAuthenticator,
	credentials *accountcredentials.CredentialManager,
	loadDescriptor WormPositionCashOutBatchDescriptorLoader,
	authorize WormPositionCashOutBatchAuthorizer,
	writeError WormPositionCashOutBatchErrorWriter,
) error {
	if h == nil || authenticate == nil || credentials == nil || loadDescriptor == nil ||
		authorize == nil || writeError == nil {
		return fmt.Errorf("Google Worm position Cash Out Batch authorization dependencies are required")
	}
	if h.wormPositionCashOutBatches != nil {
		return fmt.Errorf("Google Worm position Cash Out Batch authorization is already enabled")
	}
	store, err := newWormPositionCashOutBatchTransactionStore(redisClient)
	if err != nil {
		return err
	}
	h.wormPositionCashOutBatches = &wormPositionCashOutBatchAuthorization{
		google:         h,
		store:          store,
		authenticate:   authenticate,
		credentials:    credentials,
		loadDescriptor: loadDescriptor,
		authorize:      authorize,
		writeError:     writeError,
	}
	return nil
}

// WormPositionCashOutBatchAuthorization begins a fresh Google OIDC transaction
// from an exact-origin POST. Query values are batchId, commandId,
// expectedRevision, and returnTo.
func (h *Handler) WormPositionCashOutBatchAuthorization(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.wormPositionCashOutBatches == nil {
		setWormPositionCashOutBatchProofHeaders(w)
		writeWormPositionCashOutBatchProofError(
			w,
			http.StatusServiceUnavailable,
			WormPositionCashOutBatchAuthorizationUnavailableReason,
		)
		return
	}
	h.wormPositionCashOutBatches.begin(w, r)
}

func (h *wormPositionCashOutBatchAuthorization) begin(w http.ResponseWriter, r *http.Request) {
	setWormPositionCashOutBatchProofHeaders(w)
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	origin := strings.TrimSpace(r.Header.Get("Origin"))
	if origin == "" || h.google.publicOrigin == "" ||
		subtle.ConstantTimeCompare([]byte(origin), []byte(h.google.publicOrigin)) != 1 {
		h.redirectFailure(w, r, wormPositionCashOutBatchDefaultReturnTo, WormPositionCashOutBatchAuthorizationRequiredReason, "origin_verify")
		return
	}
	batchID, err := canonicalWormPositionCashOutBatchID(r.URL.Query().Get("batchId"), "batchId")
	if err != nil {
		h.redirectFailure(w, r, wormPositionCashOutBatchDefaultReturnTo, WormPositionCashOutBatchAuthorizationRequiredReason, "batch_id")
		return
	}
	returnTo := validateWormPositionCashOutBatchReturnTo(r.URL.Query().Get("returnTo"))
	commandID, err := canonicalWormPositionCashOutBatchID(r.URL.Query().Get("commandId"), "commandId")
	if err != nil {
		h.redirectFailure(w, r, returnTo, WormPositionCashOutBatchAuthorizationRequiredReason, "command_id")
		return
	}
	expectedRevision, err := parseWormPositionCashOutBatchExpectedRevision(r.URL.Query().Get("expectedRevision"))
	if err != nil {
		h.redirectFailure(w, r, returnTo, WormPositionCashOutBatchAuthorizationRequiredReason, "expected_revision")
		return
	}
	if err := bindMemberApplicationRealmFromQuery(r); err != nil {
		h.redirectFailure(w, r, returnTo, WormPositionCashOutBatchLoginSessionRequiredReason, "application_realm")
		return
	}
	_, credential, err := h.authenticate(r)
	if err != nil || credential.Capability != accountcredentials.CapabilityLogin || credential.JTI == "" ||
		credential.AccessRevision == 0 || credential.AccessRevision > math.MaxInt64 {
		h.redirectFailure(w, r, returnTo, WormPositionCashOutBatchLoginSessionRequiredReason, "login_session")
		return
	}
	accountID, err := accountcredentials.CanonicalAccountID(credential.AccountID)
	if err != nil || accountID != credential.AccountID {
		h.redirectFailure(w, r, returnTo, WormPositionCashOutBatchLoginSessionRequiredReason, "account_binding")
		return
	}
	account, err := h.credentials.Get(accountID)
	if err != nil || account.IdentityProvider != accountcredentials.IdentityProviderGoogle || !account.HasExternalIdentity() {
		h.redirectFailure(w, r, returnTo, WormPositionCashOutBatchAuthorizationRequiredReason, "identity_provider")
		return
	}
	sessionDigest := sha256.Sum256([]byte(credential.JTI))
	descriptorRequest := WormPositionCashOutBatchDescriptorRequest{
		BatchID:                batchID,
		CommandID:              commandID,
		ExpectedRevision:       expectedRevision,
		AccountID:              accountID,
		SessionJTIDigestSHA256: append([]byte(nil), sessionDigest[:]...),
		AccessRevision:         credential.AccessRevision,
		ReturnTo:               returnTo,
	}
	descriptor, err := h.loadDescriptor(r.Context(), descriptorRequest)
	if err != nil {
		h.redirectInjectedFailure(w, r, returnTo, "descriptor_load", err)
		return
	}
	intentDigest, err := validateWormPositionCashOutBatchDescriptor(descriptorRequest, descriptor)
	if err != nil {
		h.redirectFailure(w, r, returnTo, WormPositionCashOutBatchAuthorizationUnavailableReason, "descriptor_validate")
		return
	}
	state, err := newWormPositionCashOutBatchState()
	if err != nil {
		h.redirectFailure(w, r, returnTo, WormPositionCashOutBatchAuthorizationUnavailableReason, "state_generation")
		return
	}
	nonce, err := authregistration.RandomOpaqueValue()
	if err != nil {
		h.redirectFailure(w, r, returnTo, WormPositionCashOutBatchAuthorizationUnavailableReason, "nonce_generation")
		return
	}
	verifier := oauth2.GenerateVerifier()
	createdAt := time.Now().UTC()
	if err := h.store.create(r.Context(), state, wormPositionCashOutBatchTransaction{
		Nonce:                  nonce,
		Verifier:               verifier,
		ReturnTo:               returnTo,
		BatchID:                batchID,
		CommandID:              commandID,
		ExpectedRevision:       expectedRevision,
		AccountID:              accountID,
		SessionJTIDigestSHA256: hex.EncodeToString(sessionDigest[:]),
		AccessRevision:         credential.AccessRevision,
		IntentDigestSHA256:     hex.EncodeToString(intentDigest),
		CreatedAt:              createdAt,
	}); err != nil {
		h.redirectFailure(w, r, returnTo, WormPositionCashOutBatchAuthorizationUnavailableReason, "transaction_create")
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

func (h *wormPositionCashOutBatchAuthorization) ownsCallback(r *http.Request) bool {
	return r != nil && strings.HasPrefix(r.URL.Query().Get("state"), wormPositionCashOutBatchStatePrefix)
}

func (h *wormPositionCashOutBatchAuthorization) callback(w http.ResponseWriter, r *http.Request) {
	setWormPositionCashOutBatchProofHeaders(w)
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	bindMemberApplicationRealm(r)
	state := r.URL.Query().Get("state")
	cookie, cookieErr := r.Cookie(wormPositionCashOutBatchStateCookieName)
	h.clearStateCookie(w)
	if !validWormPositionCashOutBatchState(state) {
		h.redirectFailure(w, r, wormPositionCashOutBatchDefaultReturnTo, WormPositionCashOutBatchAuthorizationRequiredReason, "state_missing")
		return
	}
	transaction, err := h.store.consume(r.Context(), state)
	if err != nil {
		reason := WormPositionCashOutBatchAuthorizationRequiredReason
		if errors.Is(err, errWormPositionCashOutBatchTransactionUnavailable) {
			reason = WormPositionCashOutBatchAuthorizationUnavailableReason
		}
		h.redirectFailure(w, r, wormPositionCashOutBatchDefaultReturnTo, reason, "transaction_consume")
		return
	}
	returnTo := validateWormPositionCashOutBatchReturnTo(transaction.ReturnTo)
	if cookieErr != nil || !authregistration.ConstantTimeEqual(cookie.Value, state) {
		h.redirectFailure(w, r, returnTo, WormPositionCashOutBatchAuthorizationRequiredReason, "state_binding")
		return
	}
	_, credential, err := h.authenticate(r)
	if err != nil || credential.Capability != accountcredentials.CapabilityLogin || credential.JTI == "" ||
		credential.AccessRevision == 0 || credential.AccessRevision > math.MaxInt64 {
		h.redirectFailure(w, r, returnTo, WormPositionCashOutBatchLoginSessionRequiredReason, "login_session")
		return
	}
	currentSessionDigest := sha256.Sum256([]byte(credential.JTI))
	storedSessionDigest, digestErr := canonicalWormPositionCashOutBatchDigest(transaction.SessionJTIDigestSHA256)
	if digestErr != nil || !authregistration.ConstantTimeEqual(transaction.AccountID, credential.AccountID) ||
		subtle.ConstantTimeCompare(storedSessionDigest, currentSessionDigest[:]) != 1 ||
		transaction.AccessRevision != credential.AccessRevision {
		h.redirectFailure(w, r, returnTo, WormPositionCashOutBatchAuthorizationRequiredReason, "session_binding")
		return
	}
	if r.URL.Query().Get("error") != "" || r.URL.Query().Get("code") == "" {
		h.redirectFailure(w, r, returnTo, WormPositionCashOutBatchAuthorizationRequiredReason, "authorization_response")
		return
	}
	verified, verificationErr := h.google.exchangeAndVerify(
		r.Context(),
		r.URL.Query().Get("code"),
		transaction.Verifier,
		transaction.Nonce,
		transaction.CreatedAt,
	)
	if verificationErr != nil {
		reason := WormPositionCashOutBatchAuthorizationUnavailableReason
		if verificationErr.reason == "google_not_allowed" {
			reason = WormPositionCashOutBatchAuthorizationRequiredReason
		}
		h.redirectFailure(w, r, returnTo, reason, verificationErr.stage)
		return
	}
	account, err := h.credentials.Get(credential.AccountID)
	if err != nil || account.IdentityProvider != accountcredentials.IdentityProviderGoogle ||
		!authregistration.ConstantTimeEqual(account.IdentitySubject, verified.identity.Subject) {
		h.redirectFailure(w, r, returnTo, WormPositionCashOutBatchAuthorizationRequiredReason, "identity_binding")
		return
	}
	storedIntentDigest, err := canonicalWormPositionCashOutBatchDigest(transaction.IntentDigestSHA256)
	if err != nil {
		h.redirectFailure(w, r, returnTo, WormPositionCashOutBatchAuthorizationRequiredReason, "intent_binding")
		return
	}
	descriptorRequest := WormPositionCashOutBatchDescriptorRequest{
		BatchID:                transaction.BatchID,
		CommandID:              transaction.CommandID,
		ExpectedRevision:       transaction.ExpectedRevision,
		AccountID:              transaction.AccountID,
		SessionJTIDigestSHA256: append([]byte(nil), currentSessionDigest[:]...),
		AccessRevision:         transaction.AccessRevision,
		ReturnTo:               returnTo,
	}
	descriptor, err := h.loadDescriptor(r.Context(), descriptorRequest)
	if err != nil {
		h.redirectInjectedFailure(w, r, returnTo, "descriptor_reload", err)
		return
	}
	currentIntentDigest, err := validateWormPositionCashOutBatchDescriptor(descriptorRequest, descriptor)
	if err != nil || subtle.ConstantTimeCompare(storedIntentDigest, currentIntentDigest) != 1 {
		h.redirectFailure(w, r, returnTo, WormPositionCashOutBatchAuthorizationRequiredReason, "descriptor_binding")
		return
	}
	request := WormPositionCashOutBatchAuthorizationRequest{
		BatchID:                transaction.BatchID,
		CommandID:              transaction.CommandID,
		ExpectedRevision:       transaction.ExpectedRevision,
		AccountID:              transaction.AccountID,
		SessionJTIDigestSHA256: append([]byte(nil), currentSessionDigest[:]...),
		AccessRevision:         transaction.AccessRevision,
		IntentDigestSHA256:     append([]byte(nil), storedIntentDigest...),
		ReturnTo:               returnTo,
		ProofKind:              WormPositionCashOutBatchProofKindGoogle,
	}
	if _, err := h.authorize(r.Context(), request); err != nil {
		h.redirectInjectedFailure(w, r, returnTo, "authorize", err)
		return
	}
	log.WithFields(log.Fields{
		"stage": "complete", "provider": accountcredentials.IdentityProviderGoogle,
	}).Info("Worm position Cash Out Batch Google authorization succeeded")
	http.Redirect(w, r, h.google.deploymentPath(returnTo), http.StatusSeeOther)
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

func (h *wormPositionCashOutBatchAuthorization) setStateCookie(w http.ResponseWriter, value string) {
	http.SetCookie(w, &http.Cookie{
		Name:     wormPositionCashOutBatchStateCookieName,
		Value:    value,
		Path:     h.google.deploymentPath("/auth/google"),
		MaxAge:   int(wormPositionCashOutBatchTransactionTTL / time.Second),
		Expires:  time.Now().Add(wormPositionCashOutBatchTransactionTTL),
		HttpOnly: true,
		Secure:   h.google.secureCookie,
		SameSite: http.SameSiteLaxMode,
	})
}

func (h *wormPositionCashOutBatchAuthorization) clearStateCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     wormPositionCashOutBatchStateCookieName,
		Value:    "",
		Path:     h.google.deploymentPath("/auth/google"),
		MaxAge:   -1,
		Expires:  time.Unix(1, 0),
		HttpOnly: true,
		Secure:   h.google.secureCookie,
		SameSite: http.SameSiteLaxMode,
	})
}

func (h *wormPositionCashOutBatchAuthorization) redirectFailure(
	w http.ResponseWriter,
	r *http.Request,
	returnTo string,
	reason string,
	stage string,
) {
	log.WithFields(log.Fields{
		"stage": stage, "reason": reason, "provider": accountcredentials.IdentityProviderGoogle,
	}).Warn("Worm position Cash Out Batch Google authorization failed")
	http.Redirect(
		w,
		r,
		h.google.deploymentPath(wormPositionCashOutBatchFailureURL(returnTo, reason)),
		http.StatusSeeOther,
	)
}

func (h *wormPositionCashOutBatchAuthorization) redirectInjectedFailure(
	w http.ResponseWriter,
	r *http.Request,
	returnTo string,
	stage string,
	err error,
) {
	recorder := &wormPositionCashOutBatchErrorRecorder{header: make(http.Header)}
	h.writeError(recorder, err)
	reason := strings.TrimSpace(recorder.header.Get("X-Athena-Error-Reason"))
	if reason == "" {
		reason = WormPositionCashOutBatchAuthorizationUnavailableReason
	}
	h.redirectFailure(w, r, returnTo, reason, stage)
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

func parseWormPositionCashOutBatchExpectedRevision(raw string) (int64, error) {
	if raw == "" || strings.TrimSpace(raw) != raw {
		return 0, fmt.Errorf("expectedRevision is required")
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || value <= 0 {
		return 0, fmt.Errorf("expectedRevision must be a positive integer")
	}
	return value, nil
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

func wormPositionCashOutBatchFailureURL(returnTo string, reason string) string {
	parsed, err := url.Parse(validateWormPositionCashOutBatchReturnTo(returnTo))
	if err != nil {
		return wormPositionCashOutBatchDefaultReturnTo + "?wormPositionCashOutBatchReason=" + url.QueryEscape(reason)
	}
	query := parsed.Query()
	query.Set("wormPositionCashOutBatchReason", reason)
	parsed.RawQuery = query.Encode()
	return parsed.String()
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
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]string{"message": reason, "reason": reason},
	})
}

type wormPositionCashOutBatchErrorRecorder struct {
	header http.Header
	status int
}

func (r *wormPositionCashOutBatchErrorRecorder) Header() http.Header { return r.header }

func (r *wormPositionCashOutBatchErrorRecorder) WriteHeader(statusCode int) {
	if r.status == 0 {
		r.status = statusCode
	}
}

func (r *wormPositionCashOutBatchErrorRecorder) Write(value []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	return len(value), nil
}

var _ http.ResponseWriter = (*wormPositionCashOutBatchErrorRecorder)(nil)
