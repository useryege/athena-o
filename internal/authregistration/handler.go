package authregistration

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode"

	log "github.com/sirupsen/logrus"

	"github.com/useryege/athena/common"
	"github.com/useryege/athena/internal/accountcredentials"
	operationlogrecord "github.com/useryege/athena/internal/operationlog/record"
	httputil "github.com/useryege/athena/util/http"
)

const (
	CookieName       = "athena.registration"
	CookiePath       = "/auth/registration"
	maxReturnToBytes = 2048
)

type registrationView struct {
	Provider      accountcredentials.IdentityProvider `json:"provider"`
	VerifiedEmail string                              `json:"verifiedEmail"`
	SolanaAddress string                              `json:"solanaAddress"`
	Administrator bool                                `json:"administrator"`
	ExpiresAt     int64                               `json:"expiresAt"`
	CSRFToken     string                              `json:"csrfToken"`
}

type registrationSubmission struct {
	Username  string `json:"username"`
	CSRFToken string `json:"csrfToken"`
}

type availabilityResponse struct {
	Status string `json:"status"`
}

type redirectResponse struct {
	RedirectTo string `json:"redirectTo"`
}

type errorResponse struct {
	Reason string `json:"reason"`
}

// Handler owns provider-neutral anonymous username registration after an
// external identity has already been cryptographically verified.
type Handler struct {
	store        *Store
	backend      Backend
	baseHRef     string
	secureCookie bool
}

func NewHandler(redisStore *Store, backend Backend, baseHRef string, secureCookie bool) (*Handler, error) {
	if redisStore == nil || backend == nil {
		return nil, fmt.Errorf("registration store and backend are required")
	}
	return &Handler{
		store:        redisStore,
		backend:      backend,
		baseHRef:     baseHRef,
		secureCookie: secureCookie,
	}, nil
}

// Begin creates a browser-bound ticket without creating a durable Athena
// account. The provider handler remains responsible for redirecting/responding.
func (h *Handler) Begin(ctx context.Context, w http.ResponseWriter, identity Identity, returnTo string) error {
	if err := identity.Validate(); err != nil {
		return err
	}
	ticketID, err := RandomOpaqueValue()
	if err != nil {
		return fmt.Errorf("generate registration ticket: %w", err)
	}
	csrfSecret, err := RandomOpaqueValue()
	if err != nil {
		return fmt.Errorf("generate registration CSRF secret: %w", err)
	}
	if err := h.store.Create(ctx, ticketID, Ticket{
		Identity:   identity,
		ReturnTo:   ReturnToForRealm(returnTo, identity.Realm),
		CSRFSecret: csrfSecret,
		CreatedAt:  time.Now().UTC(),
	}); err != nil {
		return err
	}
	h.setCookie(w, ticketID)
	return nil
}

// Registration serves GET, POST, and DELETE for the shared anonymous setup
// resource. Every operation requires the HttpOnly browser ticket cookie.
func (h *Handler) Registration(w http.ResponseWriter, r *http.Request) {
	SetResponseHeaders(w)
	switch r.Method {
	case http.MethodGet:
		h.get(w, r)
	case http.MethodPost:
		h.create(w, r)
	case http.MethodDelete:
		h.delete(w, r)
	default:
		w.Header().Set("Allow", "GET, POST, DELETE")
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// UsernameAvailability exposes only a coarse advisory state. PostgreSQL's
// case-insensitive unique constraint remains the final registration decision.
func (h *Handler) UsernameAvailability(w http.ResponseWriter, r *http.Request) {
	SetResponseHeaders(w)
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	_, ticket, ok := h.ticketFromRequest(w, r)
	if !ok {
		return
	}
	administrator, err := ticket.Identity.Realm.Administrator()
	if err != nil {
		h.writeError(w, http.StatusUnauthorized, "registration_expired")
		return
	}
	if err := accountcredentials.ValidateUsername(r.URL.Query().Get("username"), administrator); err != nil {
		h.writeJSON(w, http.StatusOK, availabilityResponse{Status: "invalid"})
		return
	}
	available, err := h.backend.UsernameAvailable(r.Context(), r.URL.Query().Get("username"), ticket.Identity.Realm)
	if err != nil {
		h.writeError(w, http.StatusServiceUnavailable, "registration_unavailable")
		return
	}
	statusValue := "unavailable"
	if available {
		statusValue = "available"
	}
	h.writeJSON(w, http.StatusOK, availabilityResponse{Status: statusValue})
}

// SetAthenaSessionCookie is shared by provider handlers and the final
// registration step, so all browser sessions use identical cookie policy.
func (h *Handler) SetAthenaSessionCookie(w http.ResponseWriter, realm accountcredentials.ApplicationRealm, token string) error {
	return httputil.SetTokenCookie(token, realm, h.baseHRef, h.secureCookie, w)
}

// ClearCookie removes a stale or consumed shared registration binding.
func (h *Handler) ClearCookie(w http.ResponseWriter) {
	h.clearCookie(w)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	_, ticket, ok := h.ticketFromRequest(w, r)
	if !ok {
		return
	}
	solanaAddress := ""
	if ticket.Identity.Provider == accountcredentials.IdentityProviderSolanaWallet {
		solanaAddress = ticket.Identity.Subject
	}
	h.writeJSON(w, http.StatusOK, registrationView{
		Provider:      ticket.Identity.Provider,
		VerifiedEmail: ticket.Identity.VerifiedEmail,
		SolanaAddress: solanaAddress,
		Administrator: ticket.Identity.Realm == accountcredentials.ApplicationRealmAdmin,
		ExpiresAt:     ticket.CreatedAt.Add(registrationTTL).Unix(),
		CSRFToken:     ticket.CSRFSecret,
	})
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	ticketID, ticket, ok := h.ticketFromRequest(w, r)
	if !ok {
		return
	}
	var input registrationSubmission
	if err := decodeJSON(w, r, &input); err != nil {
		h.failed(w, http.StatusBadRequest, "username_invalid", "registration_decode", ticket.Identity.Provider, err)
		return
	}
	if !ConstantTimeEqual(input.CSRFToken, ticket.CSRFSecret) {
		h.failed(w, http.StatusForbidden, "registration_expired", "registration_csrf", ticket.Identity.Provider, nil)
		return
	}
	administrator, realmErr := ticket.Identity.Realm.Administrator()
	if realmErr != nil {
		h.failed(w, http.StatusUnauthorized, "registration_expired", "registration_realm", ticket.Identity.Provider, realmErr)
		return
	}
	if err := accountcredentials.ValidateUsername(input.Username, administrator); err != nil {
		h.failed(w, http.StatusBadRequest, "username_invalid", "username_validate", ticket.Identity.Provider, err)
		return
	}
	claimID, err := RandomOpaqueValue()
	if err != nil {
		h.failed(w, http.StatusServiceUnavailable, "registration_unavailable", "claim_generation", ticket.Identity.Provider, err)
		return
	}
	claimedTicket, err := h.store.Claim(r.Context(), ticketID, claimID)
	if err != nil {
		h.claimFailed(w, err, "registration_claim", ticket.Identity.Provider)
		return
	}
	if !ConstantTimeEqual(input.CSRFToken, claimedTicket.CSRFSecret) {
		_ = h.store.ReleaseClaim(r.Context(), ticketID, claimID)
		h.failed(w, http.StatusForbidden, "registration_expired", "registration_claim_binding", claimedTicket.Identity.Provider, nil)
		return
	}
	ticket = claimedTicket
	claimComplete := false
	defer func() {
		if !claimComplete {
			_ = h.store.ReleaseClaim(context.Background(), ticketID, claimID)
		}
	}()

	registered, err := h.backend.RegisterExternalAccount(r.Context(), ticket.Identity, input.Username)
	if err != nil {
		switch {
		case errors.Is(err, ErrUsernameInvalid):
			h.failed(w, http.StatusBadRequest, "username_invalid", "username_validate", ticket.Identity.Provider, err)
		case errors.Is(err, ErrUsernameUnavailable):
			h.failed(w, http.StatusConflict, "username_unavailable", "username_conflict", ticket.Identity.Provider, err)
		case errors.Is(err, ErrIdentityNotAllowed) && ticket.Identity.Provider == accountcredentials.IdentityProviderGoogle:
			h.failed(w, http.StatusForbidden, "google_not_allowed", "administrator_conflict", ticket.Identity.Provider, err)
		default:
			h.failed(w, http.StatusServiceUnavailable, "registration_unavailable", "account_create", ticket.Identity.Provider, err)
		}
		return
	}
	operationlogrecord.CaptureResource(r.Context(), "account", registered.ID)
	operationlogrecord.BindVerifiedAccount(r.Context(), registered.ID, string(ticket.Identity.Provider), "UNAUTHENTICATED")
	operationlogrecord.CaptureBool(r.Context(), "accountCreated", registered.Created)
	operationlogrecord.CaptureBool(r.Context(), "accountResolved", !registered.Created)
	if registered.Created {
		operationlogrecord.CaptureString(r.Context(), "stage", "account_created")
		if recorder := operationlogrecord.FromContext(r.Context()); recorder != nil {
			recorder.Effect("IDENTITY_ACCOUNT_CREATED")
		}
	} else {
		operationlogrecord.CaptureString(r.Context(), "stage", "account_resolved")
		if recorder := operationlogrecord.FromContext(r.Context()); recorder != nil {
			recorder.Effect("IDENTITY_ACCOUNT_RESOLVED")
		}
	}
	if err := h.backend.RegisterCommittedAccess(r.Context(), registered.ID); err != nil {
		operationlogrecord.Partial(r.Context(), "access_publish")
		h.failed(w, http.StatusServiceUnavailable, "registration_unavailable", "access_publish", ticket.Identity.Provider, err)
		return
	}
	if err := h.store.Complete(r.Context(), ticketID, claimID); err != nil {
		operationlogrecord.Partial(r.Context(), "registration_consume")
		h.claimFailed(w, err, "registration_consume", ticket.Identity.Provider)
		return
	}
	claimComplete = true
	h.clearCookie(w)
	loginRecorder := operationlogrecord.Begin(r.Context(), "identity.login")
	loginCtx := r.Context()
	if loginRecorder != nil {
		loginRecorder.Start()
		loginRecorder.Dispatched()
		loginCtx = loginRecorder.Context()
		operationlogrecord.CaptureString(loginCtx, "provider", string(ticket.Identity.Provider))
		operationlogrecord.CaptureString(loginCtx, "stage", "session_issue")
	}
	token, err := h.backend.CreateExternalLogin(loginCtx, registered.ID, ticket.Identity)
	if err != nil {
		operationlogrecord.Partial(r.Context(), "session_issue")
		if loginRecorder != nil {
			loginRecorder.ObserveError(err)
			loginRecorder.Finish()
		}
		if errors.Is(err, ErrMaintenance) {
			h.failed(w, http.StatusServiceUnavailable, "maintenance", "account_maintenance", ticket.Identity.Provider, err)
			return
		}
		reason := "registration_unavailable"
		statusCode := http.StatusServiceUnavailable
		if errors.Is(err, ErrIdentityNotAllowed) && ticket.Identity.Provider == accountcredentials.IdentityProviderGoogle {
			reason = "google_not_allowed"
			statusCode = http.StatusForbidden
		}
		h.failed(w, statusCode, reason, "session_issue", ticket.Identity.Provider, err)
		return
	}
	if err := h.SetAthenaSessionCookie(w, ticket.Identity.Realm, token); err != nil {
		operationlogrecord.Partial(r.Context(), "cookie_issue")
		if loginRecorder != nil {
			loginRecorder.ObserveError(err)
			loginRecorder.Finish()
		}
		h.failed(w, http.StatusServiceUnavailable, "registration_unavailable", "cookie_issue", ticket.Identity.Provider, err)
		return
	}
	if loginRecorder != nil {
		operationlogrecord.CaptureString(loginCtx, "stage", "session_issued")
		operationlogrecord.Commit(loginCtx, "IDENTITY_LOGIN")
		loginRecorder.Finish()
	}
	operationlogrecord.CaptureString(r.Context(), "stage", "session_issued")
	operationlogrecord.Commit(r.Context(), "IDENTITY_REGISTRATION_SUBMIT")
	h.backend.RecordLoginResult(LoginSuccess)
	log.WithFields(log.Fields{
		"stage":      "registration_complete",
		"provider":   ticket.Identity.Provider,
		"realm":      ticket.Identity.Realm,
		"account_id": registered.ID,
	}).Info("External identity registration succeeded")
	redirectTo := ReturnToForRealm(ticket.ReturnTo, ticket.Identity.Realm)
	h.writeJSON(w, http.StatusOK, redirectResponse{RedirectTo: redirectTo})
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	ticketID, ticket, ok := h.ticketFromRequest(w, r)
	if !ok {
		return
	}
	if !ConstantTimeEqual(r.Header.Get("X-Athena-CSRF-Token"), ticket.CSRFSecret) {
		h.writeError(w, http.StatusForbidden, "registration_expired")
		return
	}
	if err := h.store.Delete(r.Context(), ticketID); err != nil {
		statusCode := http.StatusServiceUnavailable
		reason := "registration_unavailable"
		if errors.Is(err, ErrRegistrationInProgress) {
			statusCode = http.StatusConflict
		} else if errors.Is(err, ErrRegistrationNotFound) {
			statusCode = http.StatusUnauthorized
			reason = "registration_expired"
			h.clearCookie(w)
		}
		h.writeError(w, statusCode, reason)
		return
	}
	operationlogrecord.CaptureString(r.Context(), "stage", "cancelled")
	operationlogrecord.Commit(r.Context(), "IDENTITY_REGISTRATION_CANCEL")
	h.clearCookie(w)
	query := url.Values{
		"returnTo":                            []string{ReturnToForRealm(ticket.ReturnTo, ticket.Identity.Realm)},
		common.ApplicationRealmQueryParameter: []string{string(ticket.Identity.Realm)},
	}
	redirectTo := "/login?" + query.Encode()
	if ticket.Identity.Provider == accountcredentials.IdentityProviderGoogle {
		redirectTo = "/auth/google/login?" + query.Encode()
	}
	h.writeJSON(w, http.StatusOK, redirectResponse{RedirectTo: redirectTo})
}

func (h *Handler) ticketFromRequest(w http.ResponseWriter, r *http.Request) (string, Ticket, bool) {
	cookie, err := r.Cookie(CookieName)
	if err != nil || !ValidOpaqueValue(cookie.Value) {
		h.clearCookie(w)
		h.writeError(w, http.StatusUnauthorized, "registration_expired")
		return "", Ticket{}, false
	}
	ticket, err := h.store.Get(r.Context(), cookie.Value)
	if err != nil {
		reason := "registration_unavailable"
		statusCode := http.StatusServiceUnavailable
		if errors.Is(err, ErrRegistrationNotFound) {
			reason = "registration_expired"
			statusCode = http.StatusUnauthorized
			h.clearCookie(w)
		} else if !errors.Is(err, ErrRegistrationUnavailable) {
			h.clearCookie(w)
		}
		h.writeError(w, statusCode, reason)
		return "", Ticket{}, false
	}
	return cookie.Value, ticket, true
}

func (h *Handler) claimFailed(w http.ResponseWriter, err error, stage string, provider accountcredentials.IdentityProvider) {
	reason := "registration_unavailable"
	statusCode := http.StatusServiceUnavailable
	if errors.Is(err, ErrRegistrationNotFound) {
		reason = "registration_expired"
		statusCode = http.StatusUnauthorized
		h.clearCookie(w)
	} else if errors.Is(err, ErrRegistrationInProgress) {
		statusCode = http.StatusConflict
	}
	h.failed(w, statusCode, reason, stage, provider, err)
}

func (h *Handler) failed(w http.ResponseWriter, statusCode int, reason, stage string, provider accountcredentials.IdentityProvider, err error) {
	fields := log.Fields{"stage": stage, "reason": reason, "provider": provider}
	if err != nil {
		fields["error_type"] = fmt.Sprintf("%T", err)
	}
	log.WithFields(fields).Warn("External identity registration failed")
	h.backend.RecordLoginResult(LoginFailure)
	h.writeError(w, statusCode, reason)
}

func (h *Handler) writeError(w http.ResponseWriter, statusCode int, reason string) {
	h.writeJSON(w, statusCode, errorResponse{Reason: reason})
}

func (h *Handler) writeJSON(w http.ResponseWriter, statusCode int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.WithError(err).Warn("Registration response encoding failed")
	}
}

func (h *Handler) setCookie(w http.ResponseWriter, value string) {
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    value,
		Path:     DeploymentPath(h.baseHRef, CookiePath),
		MaxAge:   int(registrationTTL.Seconds()),
		Expires:  time.Now().Add(registrationTTL),
		HttpOnly: true,
		Secure:   h.secureCookie,
		SameSite: http.SameSiteStrictMode,
	})
}

func (h *Handler) clearCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    "",
		Path:     DeploymentPath(h.baseHRef, CookiePath),
		MaxAge:   -1,
		Expires:  time.Unix(1, 0),
		HttpOnly: true,
		Secure:   h.secureCookie,
		SameSite: http.SameSiteStrictMode,
	})
}

// ValidateReturnTo accepts only same-origin absolute paths and rejects login
// and registration loops.
func ValidateReturnTo(raw string) string {
	if raw == "" || len(raw) > maxReturnToBytes || !strings.HasPrefix(raw, "/") || strings.HasPrefix(raw, "//") || strings.Contains(raw, "\\") {
		return DefaultReturnTo
	}
	unescapedRaw, err := url.PathUnescape(raw)
	if err != nil || strings.Contains(unescapedRaw, "\\") || strings.HasPrefix(unescapedRaw, "//") {
		return DefaultReturnTo
	}
	for _, value := range unescapedRaw {
		if unicode.IsControl(value) {
			return DefaultReturnTo
		}
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.IsAbs() || parsed.Host != "" || parsed.Path == "" || !strings.HasPrefix(parsed.Path, "/") || strings.HasPrefix(parsed.Path, "//") || strings.Contains(parsed.Path, "\\") {
		return DefaultReturnTo
	}
	for _, segment := range strings.Split(parsed.Path, "/") {
		if segment == "." || segment == ".." {
			return DefaultReturnTo
		}
	}
	for _, value := range parsed.Path + parsed.Fragment {
		if unicode.IsControl(value) {
			return DefaultReturnTo
		}
	}
	if strings.Contains(parsed.Fragment, "\\") {
		return DefaultReturnTo
	}
	if parsed.Path == "/login" || strings.HasPrefix(parsed.Path, "/login/") || parsed.Path == "/admin/login" || strings.HasPrefix(parsed.Path, "/admin/login/") || parsed.Path == "/register" || strings.HasPrefix(parsed.Path, "/register/") {
		return DefaultReturnTo
	}
	return parsed.String()
}

func administratorReturnTo(raw string) string {
	validated := ValidateReturnTo(raw)
	parsed, err := url.Parse(validated)
	if err != nil || (parsed.Path != "/admin" && !strings.HasPrefix(parsed.Path, "/admin/")) {
		return AdministratorDefaultReturnTo
	}
	return validated
}

// ReturnToForRealm keeps provider and registration redirects inside the
// application that initiated authentication. The realm is server-held state;
// a return target cannot switch it after identity verification.
func ReturnToForRealm(raw string, realm accountcredentials.ApplicationRealm) string {
	validated := ValidateReturnTo(raw)
	switch realm {
	case accountcredentials.ApplicationRealmAdmin:
		return administratorReturnTo(validated)
	case accountcredentials.ApplicationRealmMember:
		parsed, err := url.Parse(validated)
		if err != nil || parsed.Path == "/admin" || strings.HasPrefix(parsed.Path, "/admin/") {
			return DefaultReturnTo
		}
		return validated
	default:
		return DefaultReturnTo
	}
}

// DeploymentPath resolves one validated, deployment-root-relative Athena path
// beneath the configured external base href. Query strings and fragments on
// logicalPath are retained verbatim; callers remain responsible for validating
// values originating from a browser before passing them here.
func DeploymentPath(baseHRef, logicalPath string) string {
	logicalURL, logicalErr := url.Parse(logicalPath)
	if logicalErr != nil || logicalURL.IsAbs() || logicalURL.Host != "" || logicalURL.Path == "" ||
		!strings.HasPrefix(logicalURL.Path, "/") || strings.HasPrefix(logicalURL.Path, "//") || strings.Contains(logicalURL.Path, "\\") {
		logicalPath = "/"
	}
	baseURL, baseErr := url.Parse(strings.TrimSpace(baseHRef))
	if baseErr != nil || baseURL.IsAbs() || baseURL.Host != "" || baseURL.ForceQuery || baseURL.RawQuery != "" || baseURL.Fragment != "" || strings.Contains(baseURL.Path, "\\") {
		return logicalPath
	}
	base := strings.Trim(baseURL.Path, "/")
	if base == "" {
		return logicalPath
	}
	return "/" + base + logicalPath
}

func RandomOpaqueValue() (string, error) {
	value := make([]byte, 32)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(value), nil
}

func ValidOpaqueValue(value string) bool {
	decoded, err := base64.RawURLEncoding.DecodeString(value)
	return err == nil && len(decoded) == 32
}

func ConstantTimeEqual(left, right string) bool {
	if len(left) != len(right) || left == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(left), []byte(right)) == 1
}

func SetResponseHeaders(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Referrer-Policy", "no-referrer")
}

// ClientRateIdentity derives a non-reversible, fixed-width key without exposing
// the request IP to Redis key names, logs, or metrics.
func ClientRateIdentity(r *http.Request) string {
	identity := strings.TrimSpace(r.Header.Get("X-Real-IP"))
	if net.ParseIP(identity) == nil {
		identity = strings.TrimSpace(strings.Split(r.Header.Get("X-Forwarded-For"), ",")[0])
	}
	if net.ParseIP(identity) == nil {
		if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
			identity = host
		} else {
			identity = r.RemoteAddr
		}
	}
	if parsed := net.ParseIP(identity); parsed != nil {
		identity = parsed.String()
	} else {
		identity = "unknown"
	}
	digest := sha256.Sum256([]byte(identity))
	return hex.EncodeToString(digest[:])
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
