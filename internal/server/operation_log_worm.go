package server

import (
	"context"
	"net/http"

	"github.com/useryege/athena/internal/operationlog/event"
	"github.com/useryege/athena/internal/operationlog/record"
)

// observeWormHTTPMutation records facts that are available only after the
// downstream Worm command has returned a durable projection. The transport
// middleware supplies the recorder; this helper deliberately remains nil-safe
// so logging can never change the business path.
func observeWormHTTPMutation(ctx context.Context, resourceType, resourceID, effect, state string, accepted bool, details map[string]string) {
	if record.FromContext(ctx) == nil {
		return
	}
	if resourceID != "" {
		record.CaptureResource(ctx, resourceType, resourceID)
	}
	for key, value := range details {
		record.CaptureString(ctx, key, value)
	}
	if state != "" {
		if r := record.FromContext(ctx); r != nil {
			r.BusinessState(state)
		}
	}
	if accepted {
		record.Accept(ctx, effect)
	} else {
		record.Commit(ctx, effect)
	}
}

func observeWormHTTPMutationStrings(ctx context.Context, resourceType, resourceID, effect, state string, accepted bool, strings map[string][]string, details map[string]string) {
	if record.FromContext(ctx) == nil {
		return
	}
	if resourceID != "" {
		record.CaptureResource(ctx, resourceType, resourceID)
	}
	for key, values := range strings {
		record.CaptureStrings(ctx, key, values)
	}
	for key, value := range details {
		record.CaptureString(ctx, key, value)
	}
	if state != "" {
		if r := record.FromContext(ctx); r != nil {
			r.BusinessState(state)
		}
	}
	if accepted {
		record.Accept(ctx, effect)
	} else {
		record.Commit(ctx, effect)
	}
}

// observeWormAuthorization records the explicit proof result for the
// loopback-only development authorization positions. These positions expose
// no durable mutation, so the authorization itself is the business effect.
func observeWormAuthorization(ctx context.Context, resourceType, resourceID, effect, proofKind, stage string, success bool, statusCode int) {
	if record.FromContext(ctx) == nil {
		return
	}
	if resourceID != "" {
		record.CaptureResource(ctx, resourceType, resourceID)
	}
	record.CaptureString(ctx, "proofKind", proofKind)
	record.CaptureString(ctx, "stage", stage)
	if success {
		record.Commit(ctx, effect)
	} else if statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden {
		record.Deny(ctx, "AUTHORIZATION_FAILED")
	} else if statusCode >= http.StatusInternalServerError {
		if recorder := record.FromContext(ctx); recorder != nil {
			recorder.Result(event.Unknown, "AUTHORIZATION_FAILED")
		}
	} else {
		record.Fail(ctx, "AUTHORIZATION_FAILED")
	}
}
