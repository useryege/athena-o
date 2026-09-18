package phantomauth

import (
	"context"
	"net/http"

	"github.com/useryege/athena/internal/operationlog/event"
	operationlogrecord "github.com/useryege/athena/internal/operationlog/record"
)

func responseStatusCode(w http.ResponseWriter) int {
	if statusWriter, ok := w.(interface{ StatusCode() int }); ok {
		if statusCode := statusWriter.StatusCode(); statusCode != 0 {
			return statusCode
		}
	}
	return http.StatusInternalServerError
}

// observeAuthorizationFailure keeps failure-only Phantom positions explicit
// even when the HTTP transport returns a 5xx that cannot establish a business
// outcome by itself.
func observeAuthorizationFailure(ctx context.Context, stage string, statusCode int, reason string) {
	operationlogrecord.CaptureString(ctx, "provider", "solana_wallet")
	operationlogrecord.CaptureString(ctx, "stage", stage)
	if statusCode == 401 || statusCode == 403 {
		operationlogrecord.Deny(ctx, reason)
		return
	} else if statusCode >= http.StatusInternalServerError {
		if recorder := operationlogrecord.FromContext(ctx); recorder != nil {
			recorder.Result(event.Unknown, reason)
		}
		return
	}
	operationlogrecord.Fail(ctx, reason)
}
