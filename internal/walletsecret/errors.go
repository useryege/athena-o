package walletsecret

import (
	"encoding/json"
	"errors"
	"net/http"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	LoginSessionRequiredReason        = "WALLET_LOGIN_SESSION_REQUIRED"
	ReauthenticationRequiredReason    = "WALLET_REAUTH_REQUIRED"
	ReauthenticationUnavailableReason = "WALLET_REAUTH_UNAVAILABLE"
	ErrorDomain                       = "athena.wallet_secret"
)

var (
	ErrLoginSessionRequired        = stableError(codes.Unauthenticated, LoginSessionRequiredReason)
	ErrReauthenticationRequired    = stableError(codes.Unauthenticated, ReauthenticationRequiredReason)
	ErrReauthenticationUnavailable = stableError(codes.Unavailable, ReauthenticationUnavailableReason)
)

func stableError(code codes.Code, reason string) error {
	errStatus := status.New(code, reason)
	withDetails, err := errStatus.WithDetails(&errdetails.ErrorInfo{Reason: reason, Domain: ErrorDomain})
	if err != nil {
		return errStatus.Err()
	}
	return withDetails.Err()
}

// IsUnavailable distinguishes a Redis or provider outage from an absent,
// expired, or stale lease.
func IsUnavailable(err error) bool {
	if errors.Is(err, ErrReauthenticationUnavailable) {
		return true
	}
	errStatus, ok := status.FromError(err)
	return ok && errStatus.Code() == codes.Unavailable && errStatus.Message() == ReauthenticationUnavailableReason
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
		if info, ok := detail.(*errdetails.ErrorInfo); ok && info.Domain == ErrorDomain {
			return info.Reason
		}
	}
	switch errStatus.Message() {
	case LoginSessionRequiredReason, ReauthenticationRequiredReason, ReauthenticationUnavailableReason:
		return errStatus.Message()
	default:
		return ""
	}
}

type errorEnvelope struct {
	Error errorBody `json:"error"`
}

type errorBody struct {
	Code    int32  `json:"code"`
	Message string `json:"message"`
	Reason  string `json:"reason,omitempty"`
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
	if reason != "" {
		w.Header().Set("X-Athena-Error-Reason", reason)
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(httpStatus(errStatus.Code()))
	_ = json.NewEncoder(w).Encode(errorEnvelope{Error: errorBody{
		Code:    int32(errStatus.Code()),
		Message: errStatus.Message(),
		Reason:  reason,
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
	case codes.Aborted, codes.AlreadyExists:
		return http.StatusConflict
	case codes.ResourceExhausted:
		return http.StatusTooManyRequests
	case codes.Unavailable:
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}
