package googleoidc

import (
	"context"
	"strconv"

	operationlogrecord "github.com/useryege/athena/internal/operationlog/record"
)

func observeGoogleAuthorization(ctx context.Context, provider, stage, proofKind, resourceType, resourceID, effect string, expectedRevision, confirmedRevision int64, accepted bool) {
	operationlogrecord.CaptureString(ctx, "provider", provider)
	operationlogrecord.CaptureString(ctx, "stage", stage)
	if proofKind != "" {
		operationlogrecord.CaptureString(ctx, "proofKind", proofKind)
	}
	if expectedRevision >= 0 {
		operationlogrecord.CaptureString(ctx, "expectedRevision", strconv.FormatInt(expectedRevision, 10))
	}
	if confirmedRevision >= 0 {
		operationlogrecord.CaptureString(ctx, "confirmedRevision", strconv.FormatInt(confirmedRevision, 10))
	}
	if resourceID != "" {
		operationlogrecord.CaptureResource(ctx, resourceType, resourceID)
		if resourceType == "account" {
			operationlogrecord.BindVerifiedAccount(ctx, resourceID, provider)
		}
	}
	if accepted {
		operationlogrecord.Accept(ctx, effect)
	} else {
		operationlogrecord.Commit(ctx, effect)
	}
}


func failGoogleAuthorization(ctx context.Context, reason, stage string) {
	operationlogrecord.CaptureString(ctx, "provider", "google")
	operationlogrecord.CaptureString(ctx, "stage", stage)
	operationlogrecord.Fail(ctx, reason)
}
