package operationlog

import (
	"context"
	"testing"

	ipb "github.com/useryege/athena/internal/operationlog/apiclient"
	pb "github.com/useryege/athena/pkg/apiclient/operationlog"
	"google.golang.org/grpc"
)

type forwardingClient struct {
	list    func(context.Context, *ipb.ListOperationLogsRequest) (*ipb.ListOperationLogsResponse, error)
	get     func(context.Context, *ipb.GetOperationLogRequest) (*ipb.GetOperationLogResponse, error)
	runtime func(context.Context, *ipb.GetOperationLogRuntimeStatusRequest) (*ipb.GetOperationLogRuntimeStatusResponse, error)
}

func (c forwardingClient) ListOperationLogs(ctx context.Context, req *ipb.ListOperationLogsRequest, _ ...grpc.CallOption) (*ipb.ListOperationLogsResponse, error) {
	return c.list(ctx, req)
}
func (c forwardingClient) GetOperationLog(ctx context.Context, req *ipb.GetOperationLogRequest, _ ...grpc.CallOption) (*ipb.GetOperationLogResponse, error) {
	return c.get(ctx, req)
}
func (c forwardingClient) GetOperationLogRuntimeStatus(ctx context.Context, req *ipb.GetOperationLogRuntimeStatusRequest, _ ...grpc.CallOption) (*ipb.GetOperationLogRuntimeStatusResponse, error) {
	return c.runtime(ctx, req)
}

func TestForwardingServerBridgesViewerAndResponses(t *testing.T) {
	viewerID := "00000000-0000-4000-8000-000000000001"
	client := forwardingClient{list: func(_ context.Context, req *ipb.ListOperationLogsRequest) (*ipb.ListOperationLogsResponse, error) {
		if req.GetViewer().GetAccountId() != viewerID || req.GetViewer().GetRealm() != "ADMIN" || req.GetActionCode() != "account.access.update" {
			t.Fatalf("request=%+v", req)
		}
		return &ipb.ListOperationLogsResponse{Page: &ipb.Page{NextCursor: "next"}, Items: []*ipb.OperationLogSummary{{OperationId: "00000000-0000-4000-8000-000000000002", ActionCode: "account.access.update"}}}, nil
	}}
	server := NewForwardingServer(client, func(context.Context) (*ipb.Viewer, error) {
		return &ipb.Viewer{AccountId: viewerID, Realm: "ADMIN", CredentialKind: "LOGIN_SESSION", SessionBinding: make([]byte, 32), AccessRevision: 1}, nil
	})
	response, err := server.ListOperationLogs(context.Background(), &pb.ListOperationLogsRequest{ActionCode: "account.access.update"})
	if err != nil || response.GetPage().GetNextCursor() != "next" || response.GetItems()[0].GetActionCode() != "account.access.update" {
		t.Fatalf("response=%+v err=%v", response, err)
	}
}
