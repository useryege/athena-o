package walletsecret

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/useryege/athena/internal/moduleaccess"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	LoginSessionRequiredReason            = "WALLET_LOGIN_SESSION_REQUIRED"
	ReauthenticationRequiredReason        = "WALLET_REAUTH_REQUIRED"
	ReauthenticationUnavailableReason     = "WALLET_REAUTH_UNAVAILABLE"
	WormLoginSessionRequiredReason        = "WORM_TRADING_LOGIN_SESSION_REQUIRED"
	WormReauthenticationRequiredReason    = "WORM_TRADING_REAUTH_REQUIRED"
	WormReauthenticationUnavailableReason = "WORM_TRADING_REAUTH_UNAVAILABLE"
	WormConnectOutcomeUnknownReason       = "CONNECT_OUTCOME_UNKNOWN"
	ErrorDomain                           = "athena.wallet_secret"
	WormErrorDomain                       = "athena.worm_trading"
)

var (
	ErrLoginSessionRequired            = stableError(codes.Unauthenticated, LoginSessionRequiredReason, ErrorDomain)
	ErrReauthenticationRequired        = stableError(codes.Unauthenticated, ReauthenticationRequiredReason, ErrorDomain)
	ErrReauthenticationUnavailable     = stableError(codes.Unavailable, ReauthenticationUnavailableReason, ErrorDomain)
	ErrWormLoginSessionRequired        = stableError(codes.Unauthenticated, WormLoginSessionRequiredReason, WormErrorDomain)
	ErrWormReauthenticationRequired    = stableError(codes.Unauthenticated, WormReauthenticationRequiredReason, WormErrorDomain)
	ErrWormReauthenticationUnavailable = stableError(codes.Unavailable, WormReauthenticationUnavailableReason, WormErrorDomain)
	ErrWormConnectOutcomeUnknown       = stableError(codes.Aborted, WormConnectOutcomeUnknownReason, WormErrorDomain)
)

func stableError(code codes.Code, reason, domain string) error {
	errStatus := status.New(code, reason)
	withDetails, err := errStatus.WithDetails(&errdetails.ErrorInfo{Reason: reason, Domain: domain})
	if err != nil {
		return errStatus.Err()
	}
	return withDetails.Err()
}

// IsUnavailable distinguishes a Redis or provider outage from an absent,
// expired, or stale lease.
func IsUnavailable(err error) bool {
	if errors.Is(err, ErrReauthenticationUnavailable) || errors.Is(err, ErrWormReauthenticationUnavailable) {
		return true
	}
	errStatus, ok := status.FromError(err)
	return ok && errStatus.Code() == codes.Unavailable &&
		(errStatus.Message() == ReauthenticationUnavailableReason || errStatus.Message() == WormReauthenticationUnavailableReason)
}

// Reason returns a stable wallet-secret error reason when one is present.
func Reason(err error) string {
	if err == nil {
		return ""
	}
	errStatus, ok := status.FromError(err)
	if !ok {
		return ""
	}
	for _, detail := range errStatus.Details() {
		if info, ok := detail.(*errdetails.ErrorInfo); ok && (info.Domain == ErrorDomain || info.Domain == WormErrorDomain || info.Domain == moduleaccess.Domain) {
			return info.Reason
		}
	}
	switch errStatus.Message() {
	case LoginSessionRequiredReason, ReauthenticationRequiredReason, ReauthenticationUnavailableReason,
		WormLoginSessionRequiredReason, WormReauthenticationRequiredReason, WormReauthenticationUnavailableReason,
		WormConnectOutcomeUnknownReason:
		return errStatus.Message()
	default:
		return ""
	}
}

type errorEnvelope struct {
	Error errorBody `json:"error"`
}

type errorBody struct {
	ModuleKey string           `json:"module_key,omitempty"`
	Details   []map[string]any `json:"details,omitempty"`
	Code      int32            `json:"code"`
	Message   string           `json:"message"`
	Reason    string           `json:"reason,omitempty"`
}

// SetSecretResponseHeaders prevents private-key and reauthentication responses
// from entering shared or browser caches.
func SetSecretResponseHeaders(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-store, private")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Vary", "Cookie, Authorization")
	w.Header().Set("Referrer-Policy", "no-referrer")
}

// WriteError writes the same stable envelope used by native wallet-secret HTTP
// resources, including a header that callers may inspect without decoding a
// response body.
func WriteError(w http.ResponseWriter, err error) {
	SetSecretResponseHeaders(w)
	errStatus, ok := status.FromError(err)
	if !ok {
		errStatus = status.New(codes.Internal, "internal server error")
	}
	reason := Reason(err)
	var details []map[string]any
	var moduleKey string
	for _, detail := range errStatus.Details() {
		if info, ok := detail.(*errdetails.ErrorInfo); ok {
			details = append(details, map[string]any{"@type": "type.googleapis.com/google.rpc.ErrorInfo", "reason": info.Reason, "domain": info.Domain, "metadata": info.Metadata})
			if info.Domain == moduleaccess.Domain {
				moduleKey = info.Metadata["module_key"]
			}
		}
	}
	if reason != "" {
		w.Header().Set("X-Athena-Error-Reason", reason)
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(httpStatus(errStatus.Code()))
	_ = json.NewEncoder(w).Encode(errorEnvelope{Error: errorBody{
		Code:      int32(errStatus.Code()),
		Details:   details,
		ModuleKey: moduleKey,
		Message:   errStatus.Message(),
		Reason:    reason,
	}})
}

func httpStatus(code codes.Code) int {
	switch code {
	case codes.InvalidArgument:
		return http.StatusBadRequest
	case codes.Unauthenticated:
		return http.StatusUnauthorized
	case codes.PermissionDenied:
		return http.StatusForbidden
	case codes.NotFound:
		return http.StatusNotFound
	case codes.Aborted, codes.AlreadyExists, codes.FailedPrecondition:
		return http.StatusConflict
	case codes.ResourceExhausted:
		return http.StatusTooManyRequests
	case codes.Unavailable:
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}
