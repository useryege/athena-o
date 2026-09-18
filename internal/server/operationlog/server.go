// Package operationlog exposes the administrator operation-log facade.
package operationlog

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/useryege/athena/internal/operationlog/access"
	ipb "github.com/useryege/athena/internal/operationlog/apiclient"
	"github.com/useryege/athena/internal/operationlog/query"
	pb "github.com/useryege/athena/pkg/apiclient/operationlog"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ViewerResolver func(context.Context) (query.Viewer, error)
type Server struct {
	pb.UnimplementedOperationLogServiceServer
	Adapter *query.Adapter
	Viewer  ViewerResolver
	Capture func(context.Context) (*pb.CaptureStatus, error)
	Actions func(context.Context) (*pb.ListOperationLogActionsResponse, error)
	Access  access.Checker
}

func NewServer(adapter *query.Adapter, viewer ViewerResolver) *Server {
	return &Server{Adapter: adapter, Viewer: viewer}
}
func (s *Server) resolve(ctx context.Context) (query.Viewer, error) {
	if s.Viewer == nil {
		return query.Viewer{}, rpcError(codes.PermissionDenied, "OPERATION_LOG_VIEWER_FORBIDDEN")
	}
	v, e := s.Viewer(ctx)
	if e != nil {
		return query.Viewer{}, rpcError(codes.PermissionDenied, "OPERATION_LOG_VIEWER_FORBIDDEN")
	}
	if s.Access != nil {
		if err := s.Access.Check(ctx, v.AccountID); err != nil {
			if errors.Is(err, access.ErrUnavailable) {
				return query.Viewer{}, rpcError(codes.Unavailable, "OPERATION_LOG_ACCOUNT_UNAVAILABLE")
			}
			return query.Viewer{}, rpcError(codes.PermissionDenied, "OPERATION_LOG_VIEWER_FORBIDDEN")
		}
	}
	return v, nil
}
func requestFilter(r *pb.ListOperationLogsRequest) (query.Filter, error) {
	f := query.Filter{ActorQuery: r.GetActorQuery(), ActorRole: r.GetActorRole(), CredentialKind: r.GetCredentialKind(), ModuleCode: r.GetModuleCode(), ActionCode: r.GetActionCode(), Outcome: r.GetOutcome(), TargetAccountID: r.GetTargetAccountId(), ResourceType: r.GetResourceType(), ResourceID: r.GetResourceId(), PageSize: int(r.GetPageSize()), Cursor: r.GetCursor()}
	var e error
	if r.GetFrom() != "" {
		t, x := time.Parse(time.RFC3339Nano, r.GetFrom())
		if x != nil {
			return f, x
		}
		f.From = &t
	}
	if r.GetTo() != "" {
		t, x := time.Parse(time.RFC3339Nano, r.GetTo())
		if x != nil {
			return f, x
		}
		f.To = &t
	}
	return f, e
}
func (s *Server) ListOperationLogs(ctx context.Context, r *pb.ListOperationLogsRequest) (*pb.ListOperationLogsResponse, error) {
	v, e := s.resolve(ctx)
	if e != nil {
		return nil, e
	}
	f, e := requestFilter(r)
	if e != nil {
		return nil, rpcError(codes.InvalidArgument, "OPERATION_LOG_INVALID_FILTER")
	}
	if s.Adapter == nil {
		return nil, rpcError(codes.Unavailable, "OPERATION_LOG_UNAVAILABLE")
	}
	p, e := s.Adapter.List(ctx, f, v)
	if e != nil {
		return nil, mapError(e)
	}
	out := &pb.ListOperationLogsResponse{Page: &pb.Page{NextCursor: p.NextCursor, SnapshotToken: p.SnapshotToken, SnapshotSequence: itoa(p.SnapshotSequence), SnapshotAt: p.SnapshotAt.Format(time.RFC3339Nano), PageSize: int32(p.PageSize)}, AppliedFilters: &pb.AppliedFilters{From: p.AppliedFilters.From.Format(time.RFC3339Nano), To: p.AppliedFilters.To.Format(time.RFC3339Nano), ActorQuery: p.AppliedFilters.ActorQuery, ActorRole: p.AppliedFilters.ActorRole, CredentialKind: p.AppliedFilters.CredentialKind, ModuleCode: p.AppliedFilters.ModuleCode, ActionCode: p.AppliedFilters.ActionCode, Outcome: p.AppliedFilters.Outcome, TargetAccountId: p.AppliedFilters.TargetAccountID, ResourceType: p.AppliedFilters.ResourceType, ResourceId: p.AppliedFilters.ResourceID}}
	for _, x := range p.Items {
		out.Items = append(out.Items, &pb.OperationLogSummary{OperationId: x.OperationID, StartedAt: x.StartedAt.Format(time.RFC3339Nano), ActorAccountId: x.ActorAccountID, ActorUsername: x.ActorUsername, ActorRole: x.ActorRole, Realm: x.Realm, CredentialKind: x.CredentialKind, IdentityVerified: x.IdentityVerified, IdentitySnapshotComplete: x.IdentitySnapshotComplete, ModuleCode: x.ModuleCode, ActionCode: x.ActionCode, PrimaryResourceType: x.PrimaryResourceType, PrimaryResourceId: x.PrimaryResourceID, Outcome: x.Outcome, Observation: x.Observation, DurationMs: formatPtr(x.DurationMS), BusinessState: x.BusinessState, ResourceCount: formatPtr(x.ResourceCount), ResourcesComplete: x.ResourcesComplete, ResponseWriteFailed: x.ResponseWriteFailed})
	}
	return out, nil
}
func (s *Server) GetOperationLog(ctx context.Context, r *pb.GetOperationLogRequest) (*pb.GetOperationLogResponse, error) {
	v, e := s.resolve(ctx)
	if e != nil {
		return nil, e
	}
	if strings.TrimSpace(r.GetOperationId()) == "" {
		return nil, rpcError(codes.InvalidArgument, "OPERATION_LOG_INVALID_OPERATION_ID")
	}
	if s.Adapter == nil {
		return nil, rpcError(codes.Unavailable, "OPERATION_LOG_UNAVAILABLE")
	}
	d, e := s.Adapter.Get(ctx, r.GetOperationId(), v, r.GetSnapshotToken())
	if e != nil {
		return nil, mapError(e)
	}
	return &pb.GetOperationLogResponse{Item: &pb.OperationLogDetail{Summary: &pb.OperationLogSummary{OperationId: d.OperationID, StartedAt: d.StartedAt.Format(time.RFC3339Nano), ActorAccountId: d.ActorAccountID, ActorUsername: d.ActorUsername, ActorRole: d.ActorRole, Realm: d.Realm, CredentialKind: d.CredentialKind, ModuleCode: d.ModuleCode, ActionCode: d.ActionCode, Outcome: d.Outcome, Observation: d.Observation, PrimaryResourceType: d.PrimaryResourceType, PrimaryResourceId: d.PrimaryResourceID, DurationMs: formatPtr(d.DurationMS)}, FinishedAt: formatTime(d.FinishedAt), RequestId: d.RequestID, ParentOperationId: d.ParentOperationID, BusinessRequestId: d.BusinessRequestID}}, nil
}
func (s *Server) GetOperationLogRuntimeStatus(_ context.Context, _ *pb.GetOperationLogRuntimeStatusRequest) (*pb.GetOperationLogRuntimeStatusResponse, error) {
	return nil, rpcError(codes.Unavailable, "OPERATION_LOG_UNAVAILABLE")
}
func (s *Server) GetOperationLogCaptureStatus(ctx context.Context, _ *pb.GetOperationLogCaptureStatusRequest) (*pb.GetOperationLogCaptureStatusResponse, error) {
	if s.Capture == nil {
		return nil, rpcError(codes.Unavailable, "OPERATION_LOG_CAPTURE_UNAVAILABLE")
	}
	v, e := s.Capture(ctx)
	if e != nil {
		return nil, e
	}
	return &pb.GetOperationLogCaptureStatusResponse{Status: v}, nil
}
func (s *Server) ListOperationLogActions(ctx context.Context, _ *pb.ListOperationLogActionsRequest) (*pb.ListOperationLogActionsResponse, error) {
	if s.Actions == nil {
		return &pb.ListOperationLogActionsResponse{}, nil
	}
	return s.Actions(ctx)
}
func fromInternalViewer(v *ipb.Viewer) query.Viewer {
	if v == nil {
		return query.Viewer{}
	}
	return query.Viewer{AccountID: v.AccountId, Realm: v.Realm, CredentialKind: v.CredentialKind, SessionBinding: v.SessionBinding, AccessRevision: v.AccessRevision}
}
func mapError(e error) error {
	if errors.Is(e, query.ErrCursorExpired) {
		return rpcError(codes.InvalidArgument, "OPERATION_LOG_CURSOR_EXPIRED")
	}
	if errors.Is(e, query.ErrCursorInvalid) {
		return rpcError(codes.InvalidArgument, "OPERATION_LOG_CURSOR_INVALID")
	}
	if errors.Is(e, query.ErrInvalidFilter) {
		return rpcError(codes.InvalidArgument, "OPERATION_LOG_INVALID_FILTER")
	}
	return rpcError(codes.Unavailable, "OPERATION_LOG_UNAVAILABLE")
}
func rpcError(c codes.Code, r string) error {
	s := status.New(c, r)
	x, e := s.WithDetails(&errdetails.ErrorInfo{Domain: "athena.operationlog", Reason: r})
	if e != nil {
		return s.Err()
	}
	return x.Err()
}
func itoa(v int64) string { return fmt.Sprintf("%d", v) }
func formatPtr(v *int64) string {
	if v == nil {
		return ""
	}
	return fmt.Sprintf("%d", *v)
}
func formatTime(v *time.Time) string {
	if v == nil {
		return ""
	}
	return v.Format(time.RFC3339Nano)
}

var _ access.Checker

// InternalServer is the private RPC implementation. It is kept separate from
// the public facade so the two protobuf packages cannot accidentally be mixed.
type InternalServer struct {
	ipb.UnimplementedOperationLogInternalServiceServer
	Adapter *query.Adapter
	Access  access.Checker
}

func NewInternalServer(adapter *query.Adapter) *InternalServer {
	return &InternalServer{Adapter: adapter}
}
func (s *InternalServer) ListOperationLogs(ctx context.Context, r *ipb.ListOperationLogsRequest) (*ipb.ListOperationLogsResponse, error) {
	v := fromInternalViewer(r.GetViewer())
	if s.Access != nil {
		if err := s.Access.Check(ctx, v.AccountID); err != nil {
			if errors.Is(err, access.ErrUnavailable) {
				return nil, rpcError(codes.Unavailable, "OPERATION_LOG_ACCOUNT_UNAVAILABLE")
			}
			return nil, rpcError(codes.PermissionDenied, "OPERATION_LOG_VIEWER_FORBIDDEN")
		}
	}
	if s.Adapter == nil {
		return nil, rpcError(codes.Unavailable, "OPERATION_LOG_UNAVAILABLE")
	}
	f := query.Filter{ActorQuery: r.GetActorQuery(), ActorRole: r.GetActorRole(), CredentialKind: r.GetCredentialKind(), ModuleCode: r.GetModuleCode(), ActionCode: r.GetActionCode(), Outcome: r.GetOutcome(), TargetAccountID: r.GetTargetAccountId(), ResourceType: r.GetResourceType(), ResourceID: r.GetResourceId(), PageSize: int(r.GetPageSize()), Cursor: r.GetCursor()}
	p, e := s.Adapter.List(ctx, f, v)
	if e != nil {
		return nil, mapError(e)
	}
	out := &ipb.ListOperationLogsResponse{Page: &ipb.Page{NextCursor: p.NextCursor, SnapshotToken: p.SnapshotToken, SnapshotSequence: itoa(p.SnapshotSequence), SnapshotAt: p.SnapshotAt.Format(time.RFC3339Nano), PageSize: int32(p.PageSize)}}
	for _, x := range p.Items {
		out.Items = append(out.Items, &ipb.OperationLogSummary{OperationId: x.OperationID, StartedAt: x.StartedAt.Format(time.RFC3339Nano), ActorAccountId: x.ActorAccountID, ActorUsername: x.ActorUsername, ActorRole: x.ActorRole, Realm: x.Realm, CredentialKind: x.CredentialKind, ModuleCode: x.ModuleCode, ActionCode: x.ActionCode, Outcome: x.Outcome, Observation: x.Observation})
	}
	return out, nil
}
func (s *InternalServer) GetOperationLog(ctx context.Context, r *ipb.GetOperationLogRequest) (*ipb.GetOperationLogResponse, error) {
	v := fromInternalViewer(r.GetViewer())
	if s.Access != nil {
		if err := s.Access.Check(ctx, v.AccountID); err != nil {
			if errors.Is(err, access.ErrUnavailable) {
				return nil, rpcError(codes.Unavailable, "OPERATION_LOG_ACCOUNT_UNAVAILABLE")
			}
			return nil, rpcError(codes.PermissionDenied, "OPERATION_LOG_VIEWER_FORBIDDEN")
		}
	}
	if s.Adapter == nil {
		return nil, rpcError(codes.Unavailable, "OPERATION_LOG_UNAVAILABLE")
	}
	d, e := s.Adapter.Get(ctx, r.GetOperationId(), v, r.GetSnapshotToken())
	if e != nil {
		return nil, mapError(e)
	}
	return &ipb.GetOperationLogResponse{Item: &ipb.OperationLogDetail{Summary: &ipb.OperationLogSummary{OperationId: d.OperationID, StartedAt: d.StartedAt.Format(time.RFC3339Nano), Outcome: d.Outcome, Observation: d.Observation}}}, nil
}
func (s *InternalServer) GetOperationLogRuntimeStatus(_ context.Context, _ *ipb.GetOperationLogRuntimeStatusRequest) (*ipb.GetOperationLogRuntimeStatusResponse, error) {
	return nil, rpcError(codes.Unavailable, "OPERATION_LOG_UNAVAILABLE")
}
