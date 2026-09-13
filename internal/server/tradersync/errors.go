package tradersync

import (
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// MapInternalError distinguishes the service contract from public credentials.
func MapInternalError(err error) error {
	if err == nil {
		return nil
	}
	s := status.Convert(err)
	for _, detail := range s.Details() {
		info, ok := detail.(*errdetails.ErrorInfo)
		if !ok || info.Domain != "tradersync.internal.v1" {
			continue
		}
		switch info.Reason {
		case "SERVICE_AUTH_MISSING", "SERVICE_AUTH_INVALID", "ACTOR_INVALID":
			return status.Error(codes.Unavailable, "Trader Sync internal contract unavailable")
		}
	}
	switch s.Code() {
	case codes.Unknown, codes.Internal:
		return status.Error(codes.Internal, "Trader Sync request failed")
	}
	return err
}
