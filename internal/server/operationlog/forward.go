package operationlog

import (
	"context"
	"time"

	golangproto "github.com/golang/protobuf/proto" //nolint:staticcheck
	ipb "github.com/useryege/athena/internal/operationlog/apiclient"
	pb "github.com/useryege/athena/pkg/apiclient/operationlog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ForwardingServer keeps the public API server free of operation-log query
// storage. Query and runtime requests cross the authenticated internal client
// boundary with a viewer snapshot supplied by the API auth context.
type ForwardingServer struct {
	pb.UnimplementedOperationLogServiceServer
	Client  ipb.OperationLogInternalServiceClient
	Viewer  func(context.Context) (*ipb.Viewer, error)
	Capture func(context.Context) (*pb.CaptureStatus, error)
	Actions func(context.Context) (*pb.ListOperationLogActionsResponse, error)
}

func NewForwardingServer(client ipb.OperationLogInternalServiceClient, viewer func(context.Context) (*ipb.Viewer, error)) *ForwardingServer {
	return &ForwardingServer{Client: client, Viewer: viewer}
}

func (s *ForwardingServer) viewer(ctx context.Context) (*ipb.Viewer, error) {
	if s == nil || s.Viewer == nil {
		return nil, status.Error(codes.Unauthenticated, "operation log viewer unavailable")
	}
	return s.Viewer(ctx)
}

func bridgeOperationLogMessage(src, dst golangproto.Message) error {
	data, err := golangproto.Marshal(src)
	if err != nil {
		return err
	}
	return golangproto.Unmarshal(data, dst)
}

func (s *ForwardingServer) ListOperationLogs(ctx context.Context, req *pb.ListOperationLogsRequest) (*pb.ListOperationLogsResponse, error) {
	if s.Client == nil {
		return nil, unavailable()
	}
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	in := &ipb.ListOperationLogsRequest{}
	if err := bridgeOperationLogMessage(req, in); err != nil {
		return nil, err
	}
	viewer, err := s.viewer(ctx)
	if err != nil {
		return nil, err
	}
	in.Viewer = viewer
	out, err := s.Client.ListOperationLogs(ctx, in)
	if err != nil {
		return nil, err
	}
	result := &pb.ListOperationLogsResponse{}
	if err := bridgeOperationLogMessage(out, result); err != nil {
		return nil, err
	}
	return result, nil
}

func (s *ForwardingServer) GetOperationLog(ctx context.Context, req *pb.GetOperationLogRequest) (*pb.GetOperationLogResponse, error) {
	if s.Client == nil {
		return nil, unavailable()
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	in := &ipb.GetOperationLogRequest{}
	if err := bridgeOperationLogMessage(req, in); err != nil {
		return nil, err
	}
	viewer, err := s.viewer(ctx)
	if err != nil {
		return nil, err
	}
	in.Viewer = viewer
	out, err := s.Client.GetOperationLog(ctx, in)
	if err != nil {
		return nil, err
	}
	result := &pb.GetOperationLogResponse{}
	if err := bridgeOperationLogMessage(out, result); err != nil {
		return nil, err
	}
	return result, nil
}

func (s *ForwardingServer) GetOperationLogRuntimeStatus(ctx context.Context, _ *pb.GetOperationLogRuntimeStatusRequest) (*pb.GetOperationLogRuntimeStatusResponse, error) {
	if s.Client == nil {
		return nil, unavailable()
	}
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	viewer, err := s.viewer(ctx)
	if err != nil {
		return nil, err
	}
	out, err := s.Client.GetOperationLogRuntimeStatus(ctx, &ipb.GetOperationLogRuntimeStatusRequest{Viewer: viewer})
	if err != nil {
		return nil, err
	}
	result := &pb.GetOperationLogRuntimeStatusResponse{}
	if err := bridgeOperationLogMessage(out, result); err != nil {
		return nil, err
	}
	return result, nil
}

func (s *ForwardingServer) GetOperationLogCaptureStatus(ctx context.Context, _ *pb.GetOperationLogCaptureStatusRequest) (*pb.GetOperationLogCaptureStatusResponse, error) {
	if _, err := s.viewer(ctx); err != nil {
		return nil, err
	}
	if s.Capture == nil {
		return nil, unavailable()
	}
	status, err := s.Capture(ctx)
	if err != nil {
		return nil, err
	}
	return &pb.GetOperationLogCaptureStatusResponse{Status: status}, nil
}

func (s *ForwardingServer) ListOperationLogActions(ctx context.Context, _ *pb.ListOperationLogActionsRequest) (*pb.ListOperationLogActionsResponse, error) {
	if _, err := s.viewer(ctx); err != nil {
		return nil, err
	}
	if s.Actions == nil {
		return &pb.ListOperationLogActionsResponse{}, nil
	}
	return s.Actions(ctx)
}

func unavailable() error { return status.Error(codes.Unavailable, "operation log service unavailable") }
