package operationlog

import (
	"context"
	"testing"

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
func TestCaptureAndActionsDoNotRequireLogAdapter(t *testing.T) {
	s := NewServer(nil, func(context.Context) (query.Viewer, error) { return query.Viewer{}, nil })
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
