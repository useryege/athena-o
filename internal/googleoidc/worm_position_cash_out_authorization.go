package googleoidc

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/useryege/athena/internal/walletsecret"
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
	// WormPositionCashOutProofKindGoogle is the durable proof kind passed to
	// the operation authorizer after fresh OIDC verification.
	WormPositionCashOutProofKindGoogle = "GOOGLE"

	wormPositionCashOutStateCookieName = "athena.worm-position-cash-out.google.state"
	wormPositionCashOutDefaultReturnTo = "/worm-trading"

	WormPositionCashOutLoginSessionRequiredReason     = "WORM_POSITION_CASH_OUT_LOGIN_SESSION_REQUIRED"
	WormPositionCashOutAuthorizationRequiredReason    = "WORM_POSITION_CASH_OUT_AUTHORIZATION_REQUIRED"
	WormPositionCashOutAuthorizationUnavailableReason = "WORM_POSITION_CASH_OUT_AUTHORIZATION_UNAVAILABLE"
)

// WormPositionCashOutAuthenticator validates the Athena login cookie,
// attaches its typed credential, and enforces interactive Worm Trading
// READ_WRITE access. API keys must be rejected by the caller.
type WormPositionCashOutAuthenticator func(
	request *http.Request,
) (context.Context, accountcredentials.AuthenticatedCredential, error)

// WormPositionCashOutDescriptorRequest binds a fresh owner-scoped operation
// lookup to the current login Session and access revision.
type WormPositionCashOutDescriptorRequest struct {
	CashOutID              string
	CommandID              string
	ExpectedRevision       int64
	AccountID              string
	SessionJTIDigestSHA256 []byte
	AccessRevision         uint64
	ReturnTo               string
}

// WormPositionCashOutDescriptor contains the immutable authorization identity.
// IntentDigestSHA256 must be exactly 32 bytes.
type WormPositionCashOutDescriptor struct {
	CashOutID          string
	Revision           int64
	IntentDigestSHA256 []byte
}

// WormPositionCashOutDescriptorLoader performs the current owner/revision
// lookup and returns the immutable Cash Out intent digest.
type WormPositionCashOutDescriptorLoader func(
	context.Context,
	WormPositionCashOutDescriptorRequest,
) (WormPositionCashOutDescriptor, error)

// WormPositionCashOutAuthorizationRequest is emitted only after the current
// Google identity and immutable operation descriptor have both been verified.
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
	google         *Handler
	store          *wormPositionCashOutTransactionStore
	authenticate   WormPositionCashOutAuthenticator
	credentials    *accountcredentials.CredentialManager
	loadDescriptor WormPositionCashOutDescriptorLoader
	authorize      WormPositionCashOutAuthorizer
	writeError     WormPositionCashOutErrorWriter
}

// EnableWormPositionCashOutAuthorization attaches the independent, single-use
// fresh-Google proof for one immutable Cash Out operation.
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
		return fmt.Errorf("Google Worm position Cash Out authorization dependencies are required")
	}
	if h.wormPositionCashOuts != nil {
		return fmt.Errorf("Google Worm position Cash Out authorization is already enabled")
	}
	store, err := newWormPositionCashOutTransactionStore(redisClient)
	if err != nil {
		return err
	}
	h.wormPositionCashOuts = &wormPositionCashOutAuthorization{
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

// WormPositionCashOutAuthorization begins a fresh Google OIDC transaction
// from an exact-origin POST. Query values are cashOutId, commandId,
// expectedRevision, and returnTo.
func (h *Handler) WormPositionCashOutAuthorization(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.wormPositionCashOuts == nil {
		setWormPositionCashOutProofHeaders(w)
		writeWormPositionCashOutProofError(
			w,
			http.StatusServiceUnavailable,
			WormPositionCashOutAuthorizationUnavailableReason,
		)
		return
	}
	h.wormPositionCashOuts.begin(w, r)
}

func (h *wormPositionCashOutAuthorization) begin(w http.ResponseWriter, r *http.Request) {
	setWormPositionCashOutProofHeaders(w)
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	origin := strings.TrimSpace(r.Header.Get("Origin"))
	if origin == "" || h.google.publicOrigin == "" ||
		subtle.ConstantTimeCompare([]byte(origin), []byte(h.google.publicOrigin)) != 1 {
		h.redirectFailure(w, r, wormPositionCashOutDefaultReturnTo, WormPositionCashOutAuthorizationRequiredReason, "origin_verify")
		return
	}
	cashOutID, err := canonicalWormPositionCashOutID(r.URL.Query().Get("cashOutId"), "cashOutId")
	if err != nil {
		h.redirectFailure(w, r, wormPositionCashOutDefaultReturnTo, WormPositionCashOutAuthorizationRequiredReason, "cash_out_id")
		return
	}
	returnTo := validateWormPositionCashOutReturnTo(r.URL.Query().Get("returnTo"))
	commandID, err := canonicalWormPositionCashOutID(r.URL.Query().Get("commandId"), "commandId")
	if err != nil {
		h.redirectFailure(w, r, returnTo, WormPositionCashOutAuthorizationRequiredReason, "command_id")
		return
	}
	expectedRevision, err := parseWormPositionCashOutExpectedRevision(r.URL.Query().Get("expectedRevision"))
	if err != nil {
		h.redirectFailure(w, r, returnTo, WormPositionCashOutAuthorizationRequiredReason, "expected_revision")
		return
	}
	if err := bindMemberApplicationRealmFromQuery(r); err != nil {
		h.redirectFailure(w, r, returnTo, WormPositionCashOutLoginSessionRequiredReason, "application_realm")
		return
	}
	_, credential, err := h.authenticate(r)
	if reason := walletsecret.Reason(err); reason == "MODULE_ACCESS_CLOSED" || reason == "MODULE_ACCESS_UNAVAILABLE" {
		h.redirectFailure(w, r, returnTo, reason, "module_access")
		return
	}
	if err != nil || credential.Capability != accountcredentials.CapabilityLogin || credential.JTI == "" ||
		credential.AccessRevision == 0 || credential.AccessRevision > math.MaxInt64 {
		h.redirectFailure(w, r, returnTo, WormPositionCashOutLoginSessionRequiredReason, "login_session")
		return
	}
	accountID, err := accountcredentials.CanonicalAccountID(credential.AccountID)
	if err != nil || accountID != credential.AccountID {
		h.redirectFailure(w, r, returnTo, WormPositionCashOutLoginSessionRequiredReason, "account_binding")
		return
	}
	account, err := h.credentials.Get(accountID)
	if err != nil || account.IdentityProvider != accountcredentials.IdentityProviderGoogle || !account.HasExternalIdentity() {
		h.redirectFailure(w, r, returnTo, WormPositionCashOutAuthorizationRequiredReason, "identity_provider")
		return
	}
	sessionDigest := sha256.Sum256([]byte(credential.JTI))
	descriptorRequest := WormPositionCashOutDescriptorRequest{
		CashOutID:              cashOutID,
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
	intentDigest, err := validateWormPositionCashOutDescriptor(descriptorRequest, descriptor)
	if err != nil {
		h.redirectFailure(w, r, returnTo, WormPositionCashOutAuthorizationUnavailableReason, "descriptor_validate")
		return
	}
	state, err := newWormPositionCashOutState()
	if err != nil {
		h.redirectFailure(w, r, returnTo, WormPositionCashOutAuthorizationUnavailableReason, "state_generation")
		return
	}
	nonce, err := authregistration.RandomOpaqueValue()
	if err != nil {
		h.redirectFailure(w, r, returnTo, WormPositionCashOutAuthorizationUnavailableReason, "nonce_generation")
		return
	}
	verifier := oauth2.GenerateVerifier()
	createdAt := time.Now().UTC()
	if err := h.store.create(r.Context(), state, wormPositionCashOutTransaction{
		Nonce:                  nonce,
		Verifier:               verifier,
		ReturnTo:               returnTo,
		CashOutID:              cashOutID,
		CommandID:              commandID,
		ExpectedRevision:       expectedRevision,
		AccountID:              accountID,
		SessionJTIDigestSHA256: hex.EncodeToString(sessionDigest[:]),
		AccessRevision:         credential.AccessRevision,
		IntentDigestSHA256:     hex.EncodeToString(intentDigest),
		CreatedAt:              createdAt,
	}); err != nil {
		h.redirectFailure(w, r, returnTo, WormPositionCashOutAuthorizationUnavailableReason, "transaction_create")
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

func (h *wormPositionCashOutAuthorization) ownsCallback(r *http.Request) bool {
	return r != nil && strings.HasPrefix(r.URL.Query().Get("state"), wormPositionCashOutStatePrefix)
}

func (h *wormPositionCashOutAuthorization) callback(w http.ResponseWriter, r *http.Request) {
	setWormPositionCashOutProofHeaders(w)
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	bindMemberApplicationRealm(r)
	state := r.URL.Query().Get("state")
	cookie, cookieErr := r.Cookie(wormPositionCashOutStateCookieName)
	h.clearStateCookie(w)
	if !validWormPositionCashOutState(state) {
		h.redirectFailure(w, r, wormPositionCashOutDefaultReturnTo, WormPositionCashOutAuthorizationRequiredReason, "state_missing")
		return
	}
	transaction, err := h.store.consume(r.Context(), state)
	if err != nil {
		reason := WormPositionCashOutAuthorizationRequiredReason
		if errors.Is(err, errWormPositionCashOutTransactionUnavailable) {
			reason = WormPositionCashOutAuthorizationUnavailableReason
		}
		h.redirectFailure(w, r, wormPositionCashOutDefaultReturnTo, reason, "transaction_consume")
		return
	}
	returnTo := validateWormPositionCashOutReturnTo(transaction.ReturnTo)
	if cookieErr != nil || !authregistration.ConstantTimeEqual(cookie.Value, state) {
		h.redirectFailure(w, r, returnTo, WormPositionCashOutAuthorizationRequiredReason, "state_binding")
		return
	}
	_, credential, err := h.authenticate(r)
	if reason := walletsecret.Reason(err); reason == "MODULE_ACCESS_CLOSED" || reason == "MODULE_ACCESS_UNAVAILABLE" {
		h.redirectFailure(w, r, returnTo, reason, "module_access")
		return
	}
	if err != nil || credential.Capability != accountcredentials.CapabilityLogin || credential.JTI == "" ||
		credential.AccessRevision == 0 || credential.AccessRevision > math.MaxInt64 {
		h.redirectFailure(w, r, returnTo, WormPositionCashOutLoginSessionRequiredReason, "login_session")
		return
	}
	currentSessionDigest := sha256.Sum256([]byte(credential.JTI))
	storedSessionDigest, digestErr := canonicalWormPositionCashOutDigest(transaction.SessionJTIDigestSHA256)
	if digestErr != nil || !authregistration.ConstantTimeEqual(transaction.AccountID, credential.AccountID) ||
		subtle.ConstantTimeCompare(storedSessionDigest, currentSessionDigest[:]) != 1 ||
		transaction.AccessRevision != credential.AccessRevision {
		h.redirectFailure(w, r, returnTo, WormPositionCashOutAuthorizationRequiredReason, "session_binding")
		return
	}
	if r.URL.Query().Get("error") != "" || r.URL.Query().Get("code") == "" {
		h.redirectFailure(w, r, returnTo, WormPositionCashOutAuthorizationRequiredReason, "authorization_response")
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
		reason := WormPositionCashOutAuthorizationUnavailableReason
		if verificationErr.reason == "google_not_allowed" {
			reason = WormPositionCashOutAuthorizationRequiredReason
		}
		h.redirectFailure(w, r, returnTo, reason, verificationErr.stage)
		return
	}
	account, err := h.credentials.Get(credential.AccountID)
	if err != nil || account.IdentityProvider != accountcredentials.IdentityProviderGoogle ||
		!authregistration.ConstantTimeEqual(account.IdentitySubject, verified.identity.Subject) {
		h.redirectFailure(w, r, returnTo, WormPositionCashOutAuthorizationRequiredReason, "identity_binding")
		return
	}
	storedIntentDigest, err := canonicalWormPositionCashOutDigest(transaction.IntentDigestSHA256)
	if err != nil {
		h.redirectFailure(w, r, returnTo, WormPositionCashOutAuthorizationRequiredReason, "intent_binding")
		return
	}
	descriptorRequest := WormPositionCashOutDescriptorRequest{
		CashOutID:              transaction.CashOutID,
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
	currentIntentDigest, err := validateWormPositionCashOutDescriptor(descriptorRequest, descriptor)
	if err != nil || subtle.ConstantTimeCompare(storedIntentDigest, currentIntentDigest) != 1 {
		h.redirectFailure(w, r, returnTo, WormPositionCashOutAuthorizationRequiredReason, "descriptor_binding")
		return
	}
	request := WormPositionCashOutAuthorizationRequest{
		CashOutID:              transaction.CashOutID,
		CommandID:              transaction.CommandID,
		ExpectedRevision:       transaction.ExpectedRevision,
		AccountID:              transaction.AccountID,
		SessionJTIDigestSHA256: append([]byte(nil), currentSessionDigest[:]...),
		AccessRevision:         transaction.AccessRevision,
		IntentDigestSHA256:     append([]byte(nil), storedIntentDigest...),
		ReturnTo:               returnTo,
		ProofKind:              WormPositionCashOutProofKindGoogle,
	}
	if _, err := h.authorize(r.Context(), request); err != nil {
		h.redirectInjectedFailure(w, r, returnTo, "authorize", err)
		return
	}
	log.WithFields(log.Fields{
		"stage": "complete", "provider": accountcredentials.IdentityProviderGoogle,
	}).Info("Worm position Cash Out Google authorization succeeded")
	http.Redirect(w, r, h.google.deploymentPath(returnTo), http.StatusSeeOther)
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

func (h *wormPositionCashOutAuthorization) setStateCookie(w http.ResponseWriter, value string) {
	http.SetCookie(w, &http.Cookie{
		Name:     wormPositionCashOutStateCookieName,
		Value:    value,
		Path:     h.google.deploymentPath("/auth/google"),
		MaxAge:   int(wormPositionCashOutTransactionTTL / time.Second),
		Expires:  time.Now().Add(wormPositionCashOutTransactionTTL),
		HttpOnly: true,
		Secure:   h.google.secureCookie,
		SameSite: http.SameSiteLaxMode,
	})
}

func (h *wormPositionCashOutAuthorization) clearStateCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     wormPositionCashOutStateCookieName,
		Value:    "",
		Path:     h.google.deploymentPath("/auth/google"),
		MaxAge:   -1,
		Expires:  time.Unix(1, 0),
		HttpOnly: true,
		Secure:   h.google.secureCookie,
		SameSite: http.SameSiteLaxMode,
	})
}

func (h *wormPositionCashOutAuthorization) redirectFailure(
	w http.ResponseWriter,
	r *http.Request,
	returnTo string,
	reason string,
	stage string,
) {
	log.WithFields(log.Fields{
		"stage": stage, "reason": reason, "provider": accountcredentials.IdentityProviderGoogle,
	}).Warn("Worm position Cash Out Google authorization failed")
	http.Redirect(
		w,
		r,
		h.google.deploymentPath(wormPositionCashOutFailureURL(returnTo, reason)),
		http.StatusSeeOther,
	)
}

func (h *wormPositionCashOutAuthorization) redirectInjectedFailure(
	w http.ResponseWriter,
	r *http.Request,
	returnTo string,
	stage string,
	err error,
) {
	recorder := &wormPositionCashOutErrorRecorder{header: make(http.Header)}
	h.writeError(recorder, err)
	reason := strings.TrimSpace(recorder.header.Get("X-Athena-Error-Reason"))
	if reason == "" {
		reason = WormPositionCashOutAuthorizationUnavailableReason
	}
	h.redirectFailure(w, r, returnTo, reason, stage)
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

func parseWormPositionCashOutExpectedRevision(raw string) (int64, error) {
	if raw == "" || strings.TrimSpace(raw) != raw {
		return 0, fmt.Errorf("expectedRevision is required")
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || value <= 0 {
		return 0, fmt.Errorf("expectedRevision must be a positive integer")
	}
	return value, nil
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

func wormPositionCashOutFailureURL(returnTo string, reason string) string {
	parsed, err := url.Parse(validateWormPositionCashOutReturnTo(returnTo))
	if err != nil {
		return wormPositionCashOutDefaultReturnTo + "?wormPositionCashOutReason=" + url.QueryEscape(reason)
	}
	query := parsed.Query()
	query.Set("wormPositionCashOutReason", reason)
	parsed.RawQuery = query.Encode()
	return parsed.String()
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
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]string{"message": reason, "reason": reason},
	})
}

type wormPositionCashOutErrorRecorder struct {
	header http.Header
	status int
}

func (r *wormPositionCashOutErrorRecorder) Header() http.Header { return r.header }

func (r *wormPositionCashOutErrorRecorder) WriteHeader(statusCode int) {
	if r.status == 0 {
		r.status = statusCode
	}
}

func (r *wormPositionCashOutErrorRecorder) Write(value []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	return len(value), nil
}

var _ http.ResponseWriter = (*wormPositionCashOutErrorRecorder)(nil)
