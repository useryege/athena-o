package operationlog

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/useryege/athena/internal/operationlog/access"
	ipb "github.com/useryege/athena/internal/operationlog/apiclient"
	"github.com/useryege/athena/internal/operationlog/query"
	pb "github.com/useryege/athena/pkg/apiclient/operationlog"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestPublicFacadeRequiresTrustedViewerResolver(t *testing.T) {
	s := NewServer(nil, nil)
	_, err := s.ListOperationLogs(context.Background(), &pb.ListOperationLogsRequest{})
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("code=%v", status.Code(err))
	}
}

func TestPersistentViewerDenialUsesStableAdminReason(t *testing.T) {
	viewer := func(context.Context) (query.Viewer, error) {
		return query.Viewer{AccountID: uuid.NewString(), Realm: "ADMIN", CredentialKind: "LOGIN_SESSION", SessionBinding: []byte("01234567890123456789012345678901"), AccessRevision: 1}, nil
	}
	s := NewServer(nil, viewer)
	s.Access = access.CheckerFunc(func(context.Context, string) (access.State, error) {
		return access.State{}, nil
	})
	_, err := s.ListOperationLogs(context.Background(), &pb.ListOperationLogsRequest{})
	if status.Code(err) != codes.PermissionDenied || errorInfoReason(err) != "ACCOUNT_ADMIN_REQUIRED" {
		t.Fatalf("err=%v reason=%q", err, errorInfoReason(err))
	}
}
func TestInternalFilterPreservesExplicitRange(t *testing.T) {
	r := &ipb.ListOperationLogsRequest{From: "2026-01-01T00:00:00Z", To: "2026-01-31T00:00:00Z"}
	f, err := internalFilter(r)
	if err != nil || f.From == nil || f.To == nil || f.From.Format(time.RFC3339) != "2026-01-01T00:00:00Z" {
		t.Fatalf("filter=%+v err=%v", f, err)
	}
}
func TestMapErrorUsesStableQueryReasons(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		code   codes.Code
		reason string
	}{
		{name: "not found", err: query.ErrNotFound, code: codes.NotFound, reason: "OPERATION_LOG_NOT_FOUND"},
		{name: "invalid id", err: query.ErrInvalidOperationID, code: codes.InvalidArgument, reason: "OPERATION_LOG_INVALID_OPERATION_ID"},
		{name: "filter", err: query.ErrInvalidFilter, code: codes.InvalidArgument, reason: "FILTER_INVALID"},
		{name: "cursor", err: query.ErrCursorInvalid, code: codes.InvalidArgument, reason: "CURSOR_INVALID"},
		{name: "snapshot", err: query.ErrSnapshotInvalid, code: codes.InvalidArgument, reason: "SNAPSHOT_INVALID"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := mapError(tt.err)
			if status.Code(err) != tt.code {
				t.Fatalf("code=%v want %v", status.Code(err), tt.code)
			}
			if got := errorInfoReason(err); got != tt.reason {
				t.Fatalf("reason=%q want %q", got, tt.reason)
			}
		})
	}
}

func errorInfoReason(err error) string {
	for _, detail := range status.Convert(err).Details() {
		if info, ok := detail.(*errdetails.ErrorInfo); ok {
			return info.Reason
		}
	}
	return ""
}
func TestCaptureAndActionsDoNotRequireLogAdapter(t *testing.T) {
	s := NewServer(nil, func(context.Context) (query.Viewer, error) {
		return query.Viewer{AccountID: uuid.NewString(), Realm: "ADMIN", CredentialKind: "LOGIN_SESSION", SessionBinding: []byte("01234567890123456789012345678901"), AccessRevision: 1}, nil
	})
	s.Capture = func(context.Context) (*pb.CaptureStatus, error) { return &pb.CaptureStatus{InstanceId: "api"}, nil }
	s.Actions = func(context.Context) (*pb.ListOperationLogActionsResponse, error) {
		return &pb.ListOperationLogActionsResponse{CatalogVersion: "v1"}, nil
	}
	if got, e := s.GetOperationLogCaptureStatus(context.Background(), &pb.GetOperationLogCaptureStatusRequest{}); e != nil || got.GetStatus().GetInstanceId() != "api" {
		t.Fatalf("capture=%v %v", got, e)
	}
	if got, e := s.ListOperationLogActions(context.Background(), &pb.ListOperationLogActionsRequest{}); e != nil || got.GetCatalogVersion() != "v1" {
		t.Fatalf("actions=%v %v", got, e)
	}
}
