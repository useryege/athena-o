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
	// WormExecutionProofKindGoogle is the durable proof-kind value passed to
	// the Run authorization callback after fresh OIDC verification.
	WormExecutionProofKindGoogle = "GOOGLE"

	wormExecutionStateCookieName = "athena.worm-execution.google.state"
	wormExecutionDefaultReturnTo = "/worm-trading/executions"

	// WormExecutionLoginSessionRequiredReason identifies a missing or stale
	// interactive Athena session during Run proof.
	WormExecutionLoginSessionRequiredReason = "WORM_EXECUTION_LOGIN_SESSION_REQUIRED"
	// WormExecutionAuthorizationRequiredReason identifies a rejected or stale
	// identity proof.
	WormExecutionAuthorizationRequiredReason = "WORM_EXECUTION_AUTHORIZATION_REQUIRED"
	// WormExecutionAuthorizationUnavailableReason identifies a temporary proof
	// dependency failure.
	WormExecutionAuthorizationUnavailableReason = "WORM_EXECUTION_AUTHORIZATION_UNAVAILABLE"
)

// WormExecutionAuthenticator validates the Athena login cookie, attaches its
// typed credential to the returned context, and enforces Worm Trading
// READ_WRITE. API keys must be rejected by the caller.
type WormExecutionAuthenticator func(request *http.Request) (context.Context, accountcredentials.AuthenticatedCredential, error)

// WormExecutionAuthorizationRequest is the verified, non-secret proof binding
// passed to the API Server. SessionJTIDigestSHA256 is always exactly 32 bytes.
// No Google token, authorization code, or raw identity assertion crosses this
// callback boundary.
type WormExecutionAuthorizationRequest struct {
	RunID                  string
	CommandID              string
	ExpectedRevision       int64
	AccountID              string
	SessionJTIDigestSHA256 []byte
	AccessRevision         uint64
	ReturnTo               string
	ProofKind              string
}

// WormExecutionAuthorizer persists a verified Run-bound authorization and may
// return the current public Run projection. Google browser callbacks redirect
// after completion and therefore intentionally do not encode the projection.
type WormExecutionAuthorizer func(context.Context, WormExecutionAuthorizationRequest) (any, error)

// WormExecutionErrorWriter projects injected domain errors without coupling
// this package to the API Server or Worm Trading implementation.
type WormExecutionErrorWriter func(http.ResponseWriter, error)

type wormExecutionAuthorization struct {
	google       *Handler
	store        *wormExecutionTransactionStore
	authenticate WormExecutionAuthenticator
	admit        func(context.Context) error
	credentials  *accountcredentials.CredentialManager
	authorize    WormExecutionAuthorizer
	writeError   WormExecutionErrorWriter
}

// EnableWormExecutionAuthorization attaches the independent fresh-Google proof
// flow used only to authorize one immutable Worm execution Run. The Redis
// transaction is short-lived and single-use; it is not the Run authorization
// and does not create or reuse the Worm credential-management lease.
func (h *Handler) EnableWormExecutionAuthorization(
	redisClient *redis.Client,
	authenticate WormExecutionAuthenticator,
	admit func(context.Context) error,
	credentials *accountcredentials.CredentialManager,
	authorize WormExecutionAuthorizer,
	writeError WormExecutionErrorWriter,
) error {
	if h == nil || authenticate == nil || admit == nil || credentials == nil || authorize == nil || writeError == nil {
		return fmt.Errorf("Google Worm execution authorization dependencies are required")
	}
	if h.wormExecutions != nil {
		return fmt.Errorf("Google Worm execution authorization is already enabled")
	}
	store, err := newWormExecutionTransactionStore(redisClient)
	if err != nil {
		return err
	}
	h.wormExecutions = &wormExecutionAuthorization{
		google:       h,
		store:        store,
		authenticate: authenticate,
		admit:        admit,
		credentials:  credentials,
		authorize:    authorize,
		writeError:   writeError,
	}
	return nil
}

// WormExecutionAuthorization begins a fresh Google OIDC transaction. The
// request must provide canonical runId and commandId UUIDs, a positive
// expectedRevision, and an optional same-origin returnTo path.
func (h *Handler) WormExecutionAuthorization(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.wormExecutions == nil {
		setWormExecutionProofHeaders(w)
		writeWormExecutionProofError(w, http.StatusServiceUnavailable, WormExecutionAuthorizationUnavailableReason)
		return
	}
	h.wormExecutions.begin(w, r)
}

func (h *wormExecutionAuthorization) begin(w http.ResponseWriter, r *http.Request) {
	setWormExecutionProofHeaders(w)
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	runID, err := canonicalWormExecutionID(r.URL.Query().Get("runId"), "runId")
	if err != nil {
		h.redirectFailure(w, r, wormExecutionDefaultReturnTo, WormExecutionAuthorizationRequiredReason, "run_id")
		return
	}
	returnTo := validateWormExecutionReturnTo(r.URL.Query().Get("returnTo"), runID)
	commandID, err := canonicalWormExecutionID(r.URL.Query().Get("commandId"), "commandId")
	if err != nil {
		h.redirectFailure(w, r, returnTo, WormExecutionAuthorizationRequiredReason, "command_id")
		return
	}
	expectedRevision, err := parseWormExecutionExpectedRevision(r.URL.Query().Get("expectedRevision"))
	if err != nil {
		h.redirectFailure(w, r, returnTo, WormExecutionAuthorizationRequiredReason, "expected_revision")
		return
	}
	if err := bindMemberApplicationRealmFromQuery(r); err != nil {
		h.redirectFailure(w, r, returnTo, WormExecutionLoginSessionRequiredReason, "application_realm")
		return
	}
	authCtx, credential, err := h.authenticate(r)
	if err != nil || credential.Capability != accountcredentials.CapabilityLogin || credential.JTI == "" ||
		credential.AccessRevision == 0 || credential.AccessRevision > math.MaxInt64 {
		h.redirectFailure(w, r, returnTo, WormExecutionLoginSessionRequiredReason, "login_session")
		return
	}
	if err := h.admit(authCtx); err != nil {
		reason := walletsecret.Reason(err)
		if reason == "" {
			reason = WormExecutionAuthorizationUnavailableReason
		}
		h.redirectFailure(w, r, returnTo, reason, "module_access")
		return
	}
	accountID, err := accountcredentials.CanonicalAccountID(credential.AccountID)
	if err != nil || accountID != credential.AccountID {
		h.redirectFailure(w, r, returnTo, WormExecutionLoginSessionRequiredReason, "account_binding")
		return
	}
	account, err := h.credentials.Get(accountID)
	if err != nil || account.IdentityProvider != accountcredentials.IdentityProviderGoogle || !account.HasExternalIdentity() {
		h.redirectFailure(w, r, returnTo, WormExecutionAuthorizationRequiredReason, "identity_provider")
		return
	}
	state, err := newWormExecutionState()
	if err != nil {
		h.redirectFailure(w, r, returnTo, WormExecutionAuthorizationUnavailableReason, "state_generation")
		return
	}
	nonce, err := authregistration.RandomOpaqueValue()
	if err != nil {
		h.redirectFailure(w, r, returnTo, WormExecutionAuthorizationUnavailableReason, "nonce_generation")
		return
	}
	verifier := oauth2.GenerateVerifier()
	createdAt := time.Now().UTC()
	sessionDigest := sha256.Sum256([]byte(credential.JTI))
	if err := h.store.create(r.Context(), state, wormExecutionTransaction{
		Nonce:                  nonce,
		Verifier:               verifier,
		ReturnTo:               returnTo,
		RunID:                  runID,
		CommandID:              commandID,
		ExpectedRevision:       expectedRevision,
		AccountID:              accountID,
		SessionJTIDigestSHA256: hex.EncodeToString(sessionDigest[:]),
		AccessRevision:         credential.AccessRevision,
		CreatedAt:              createdAt,
	}); err != nil {
		h.redirectFailure(w, r, returnTo, WormExecutionAuthorizationUnavailableReason, "transaction_create")
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

func (h *wormExecutionAuthorization) ownsCallback(r *http.Request) bool {
	return r != nil && strings.HasPrefix(r.URL.Query().Get("state"), wormExecutionStatePrefix)
}

func (h *wormExecutionAuthorization) callback(w http.ResponseWriter, r *http.Request) {
	setWormExecutionProofHeaders(w)
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	bindMemberApplicationRealm(r)
	state := r.URL.Query().Get("state")
	cookie, cookieErr := r.Cookie(wormExecutionStateCookieName)
	h.clearStateCookie(w)
	if !validWormExecutionState(state) {
		h.redirectFailure(w, r, wormExecutionDefaultReturnTo, WormExecutionAuthorizationRequiredReason, "state_missing")
		return
	}
	transaction, err := h.store.consume(r.Context(), state)
	if err != nil {
		reason := WormExecutionAuthorizationRequiredReason
		if errors.Is(err, errWormExecutionTransactionUnavailable) {
			reason = WormExecutionAuthorizationUnavailableReason
		}
		h.redirectFailure(w, r, wormExecutionDefaultReturnTo, reason, "transaction_consume")
		return
	}
	returnTo := validateWormExecutionReturnTo(transaction.ReturnTo, transaction.RunID)
	if cookieErr != nil || !authregistration.ConstantTimeEqual(cookie.Value, state) {
		h.redirectFailure(w, r, returnTo, WormExecutionAuthorizationRequiredReason, "state_binding")
		return
	}
	authCtx, credential, err := h.authenticate(r)
	if err != nil || credential.Capability != accountcredentials.CapabilityLogin || credential.JTI == "" ||
		credential.AccessRevision == 0 || credential.AccessRevision > math.MaxInt64 {
		h.redirectFailure(w, r, returnTo, WormExecutionLoginSessionRequiredReason, "login_session")
		return
	}
	currentSessionDigest := sha256.Sum256([]byte(credential.JTI))
	storedSessionDigest, digestErr := hex.DecodeString(transaction.SessionJTIDigestSHA256)
	if digestErr != nil || len(storedSessionDigest) != sha256.Size ||
		!authregistration.ConstantTimeEqual(transaction.AccountID, credential.AccountID) ||
		subtle.ConstantTimeCompare(storedSessionDigest, currentSessionDigest[:]) != 1 ||
		transaction.AccessRevision != credential.AccessRevision {
		h.redirectFailure(w, r, returnTo, WormExecutionAuthorizationRequiredReason, "session_binding")
		return
	}
	if err := h.admit(authCtx); err != nil {
		reason := walletsecret.Reason(err)
		if reason == "" {
			reason = WormExecutionAuthorizationUnavailableReason
		}
		h.redirectFailure(w, r, returnTo, reason, "module_access")
		return
	}
	if r.URL.Query().Get("error") != "" || r.URL.Query().Get("code") == "" {
		h.redirectFailure(w, r, returnTo, WormExecutionAuthorizationRequiredReason, "authorization_response")
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
		reason := WormExecutionAuthorizationUnavailableReason
		if verificationErr.reason == "google_not_allowed" {
			reason = WormExecutionAuthorizationRequiredReason
		}
		h.redirectFailure(w, r, returnTo, reason, verificationErr.stage)
		return
	}
	account, err := h.credentials.Get(credential.AccountID)
	if err != nil || account.IdentityProvider != accountcredentials.IdentityProviderGoogle ||
		!authregistration.ConstantTimeEqual(account.IdentitySubject, verified.identity.Subject) {
		h.redirectFailure(w, r, returnTo, WormExecutionAuthorizationRequiredReason, "identity_binding")
		return
	}
	request := WormExecutionAuthorizationRequest{
		RunID:                  transaction.RunID,
		CommandID:              transaction.CommandID,
		ExpectedRevision:       transaction.ExpectedRevision,
		AccountID:              transaction.AccountID,
		SessionJTIDigestSHA256: append([]byte(nil), currentSessionDigest[:]...),
		AccessRevision:         transaction.AccessRevision,
		ReturnTo:               returnTo,
		ProofKind:              WormExecutionProofKindGoogle,
	}
	if _, err := h.authorize(r.Context(), request); err != nil {
		// The injected writer owns the API error envelope. Google callbacks are
		// navigations, so a small in-memory recorder obtains only the stable
		// reason header before redirecting back to the execution page.
		recorder := &wormExecutionErrorRecorder{header: make(http.Header)}
		h.writeError(recorder, err)
		reason := strings.TrimSpace(recorder.header.Get("X-Athena-Error-Reason"))
		if reason == "" {
			reason = WormExecutionAuthorizationUnavailableReason
		}
		h.redirectFailure(w, r, returnTo, reason, "authorize")
		return
	}
	log.WithFields(log.Fields{"stage": "complete", "provider": accountcredentials.IdentityProviderGoogle}).Info("Worm execution Google authorization succeeded")
	observeGoogleAuthorization(r.Context(), "google", "authorization_verified", "GOOGLE", "execution", transaction.RunID, "WORM_EXECUTION_AUTHORIZE", transaction.ExpectedRevision, transaction.ExpectedRevision, false)
	http.Redirect(w, r, h.google.deploymentPath(returnTo), http.StatusSeeOther)
}

func (h *wormExecutionAuthorization) setStateCookie(w http.ResponseWriter, value string) {
	http.SetCookie(w, &http.Cookie{
		Name:     wormExecutionStateCookieName,
		Value:    value,
		Path:     h.google.deploymentPath("/auth/google"),
		MaxAge:   int(wormExecutionTransactionTTL / time.Second),
		Expires:  time.Now().Add(wormExecutionTransactionTTL),
		HttpOnly: true,
		Secure:   h.google.secureCookie,
		SameSite: http.SameSiteLaxMode,
	})
}

func (h *wormExecutionAuthorization) clearStateCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     wormExecutionStateCookieName,
		Value:    "",
		Path:     h.google.deploymentPath("/auth/google"),
		MaxAge:   -1,
		Expires:  time.Unix(1, 0),
		HttpOnly: true,
		Secure:   h.google.secureCookie,
		SameSite: http.SameSiteLaxMode,
	})
}

func (h *wormExecutionAuthorization) redirectFailure(w http.ResponseWriter, r *http.Request, returnTo, reason, stage string) {
	failGoogleAuthorization(r.Context(), reason, stage)
	log.WithFields(log.Fields{"stage": stage, "reason": reason, "provider": accountcredentials.IdentityProviderGoogle}).Warn("Worm execution Google authorization failed")
	http.Redirect(w, r, h.google.deploymentPath(wormExecutionFailureURL(returnTo, reason)), http.StatusSeeOther)
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

func parseWormExecutionExpectedRevision(raw string) (int64, error) {
	if raw == "" || strings.TrimSpace(raw) != raw {
		return 0, fmt.Errorf("expectedRevision is required")
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || value <= 0 {
		return 0, fmt.Errorf("expectedRevision must be a positive integer")
	}
	return value, nil
}

func validateWormExecutionReturnTo(raw, runID string) string {
	fallback := wormExecutionDefaultReturnTo
	if canonical, err := canonicalWormExecutionID(runID, "runId"); err == nil {
		fallback += "/" + canonical
	}
	if raw == "" {
		return fallback
	}
	validated := authregistration.ReturnToForRealm(raw, accountcredentials.ApplicationRealmMember)
	if validated == authregistration.DefaultReturnTo && raw != authregistration.DefaultReturnTo {
		return fallback
	}
	return validated
}

func wormExecutionFailureURL(returnTo, reason string) string {
	parsed, err := url.Parse(validateWormExecutionReturnTo(returnTo, ""))
	if err != nil {
		return wormExecutionDefaultReturnTo + "?wormExecutionReason=" + url.QueryEscape(reason)
	}
	query := parsed.Query()
	query.Set("wormExecutionReason", reason)
	parsed.RawQuery = query.Encode()
	return parsed.String()
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
	_ = json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]string{"message": reason, "reason": reason},
	})
}

type wormExecutionErrorRecorder struct {
	header http.Header
	status int
}

func (r *wormExecutionErrorRecorder) Header() http.Header {
	return r.header
}

func (r *wormExecutionErrorRecorder) WriteHeader(statusCode int) {
	if r.status == 0 {
		r.status = statusCode
	}
}

func (r *wormExecutionErrorRecorder) Write(value []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	return len(value), nil
}

var _ http.ResponseWriter = (*wormExecutionErrorRecorder)(nil)
