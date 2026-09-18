package operationlog

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	ipb "github.com/useryege/athena/internal/operationlog/apiclient"
	"github.com/useryege/athena/internal/operationlog/query"
	pb "github.com/useryege/athena/pkg/apiclient/operationlog"
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
func TestInternalFilterPreservesExplicitRange(t *testing.T) {
	r := &ipb.ListOperationLogsRequest{From: "2026-01-01T00:00:00Z", To: "2026-01-31T00:00:00Z"}
	f, err := internalFilter(r)
	if err != nil || f.From == nil || f.To == nil || f.From.Format(time.RFC3339) != "2026-01-01T00:00:00Z" {
		t.Fatalf("filter=%+v err=%v", f, err)
	}
}
func TestMapErrorUsesStableQueryReasons(t *testing.T) {
	if status.Code(mapError(query.ErrNotFound)) != codes.NotFound {
		t.Fatal("not found must map to NotFound")
	}
	if status.Code(mapError(query.ErrInvalidOperationID)) != codes.InvalidArgument {
		t.Fatal("invalid id must map to InvalidArgument")
	}
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
