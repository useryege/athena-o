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
	if e != nil || query.ValidateViewer(v) != nil {
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
	sm := &pb.OperationLogSummary{OperationId: d.OperationID, StartedAt: d.StartedAt.Format(time.RFC3339Nano), ActorAccountId: d.ActorAccountID, ActorUsername: d.ActorUsername, ActorRole: d.ActorRole, Realm: d.Realm, CredentialKind: d.CredentialKind, IdentityVerified: d.IdentityVerified, IdentitySnapshotComplete: d.IdentitySnapshotComplete, ModuleCode: d.ModuleCode, ActionCode: d.ActionCode, Outcome: d.Outcome, Observation: d.Observation, ReasonCode: d.ReasonCode, PrimaryResourceType: d.PrimaryResourceType, PrimaryResourceId: d.PrimaryResourceID, DurationMs: formatPtr(d.DurationMS), BusinessState: d.BusinessState, ResourceCount: formatPtr(d.ResourceCount), ResourcesComplete: d.ResourcesComplete, ResponseWriteFailed: d.ResponseWriteFailed, Provider: d.Provider}
	det := &pb.OperationLogDetail{Summary: sm, FinishedAt: formatTime(d.FinishedAt), RequestId: d.RequestID, ParentOperationId: d.ParentOperationID, BusinessRequestId: d.BusinessRequestID, Effect: strings.Join(d.Effect, ",")}
	for _, r := range d.Resources {
		det.ResourceFacts = append(det.ResourceFacts, &pb.OperationLogResource{Type: r.Type, Id: r.ID, Relation: r.Relation, ReferenceVerified: r.ReferenceVerified})
	}
	for _, c := range d.Changes {
		det.ChangeFacts = append(det.ChangeFacts, &pb.OperationLogChange{FieldCode: c.FieldCode, BeforeValue: stringOrEmpty(c.BeforeValue), AfterValue: stringOrEmpty(c.AfterValue), BeforeAvailable: c.BeforeAvailable, ValueKind: c.ValueKind})
	}
	det.Protocol = &pb.OperationLogProtocolResult{GrpcCode: d.Protocol.GRPCCode, HttpStatus: d.Protocol.HTTPStatus, ReasonCode: d.Protocol.ReasonCode, ResponseWriteFailed: d.Protocol.ResponseWriteFailed}
	det.Source = &pb.OperationLogSourceFacts{ProducerId: d.Source.ProducerID, FirstReceivedAt: d.Source.FirstReceivedAt.Format(time.RFC3339Nano), LastReceivedAt: d.Source.LastReceivedAt.Format(time.RFC3339Nano), PhasesReceived: d.Source.PhasesReceived, SnapshotSequence: itoa(d.Source.SnapshotSequence)}
	return &pb.GetOperationLogResponse{Item: det}, nil
}
func (s *Server) GetOperationLogRuntimeStatus(ctx context.Context, _ *pb.GetOperationLogRuntimeStatusRequest) (*pb.GetOperationLogRuntimeStatusResponse, error) {
	if _, err := s.resolve(ctx); err != nil {
		return nil, err
	}
	if s.Adapter == nil {
		return nil, rpcError(codes.Unavailable, "OPERATION_LOG_UNAVAILABLE")
	}
	st, err := s.Adapter.Runtime(ctx)
	if err != nil {
		return nil, mapError(err)
	}
	return &pb.GetOperationLogRuntimeStatusResponse{Status: runtimePublic(st)}, nil
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
func appliedInternal(f query.NormalizedFilter) *ipb.AppliedFilters {
	return &ipb.AppliedFilters{From: f.From.Format(time.RFC3339Nano), To: f.To.Format(time.RFC3339Nano), ActorQuery: f.ActorQuery, ActorRole: f.ActorRole, CredentialKind: f.CredentialKind, ModuleCode: f.ModuleCode, ActionCode: f.ActionCode, Outcome: f.Outcome, TargetAccountId: f.TargetAccountID, ResourceType: f.ResourceType, ResourceId: f.ResourceID}
}
func summaryInternal(x query.Summary) *ipb.OperationLogSummary {
	return &ipb.OperationLogSummary{OperationId: x.OperationID, StartedAt: x.StartedAt.Format(time.RFC3339Nano), ActorAccountId: x.ActorAccountID, ActorUsername: x.ActorUsername, ActorRole: x.ActorRole, Realm: x.Realm, CredentialKind: x.CredentialKind, IdentityVerified: x.IdentityVerified, IdentitySnapshotComplete: x.IdentitySnapshotComplete, ModuleCode: x.ModuleCode, ActionCode: x.ActionCode, PrimaryResourceType: x.PrimaryResourceType, PrimaryResourceId: x.PrimaryResourceID, Outcome: x.Outcome, Observation: x.Observation, ReasonCode: x.ReasonCode, DurationMs: formatPtr(x.DurationMS), BusinessState: x.BusinessState, ResourceCount: formatPtr(x.ResourceCount), ResourcesComplete: x.ResourcesComplete, ResponseWriteFailed: x.ResponseWriteFailed, Provider: x.Provider}
}
func detailInternal(d query.Detail) *ipb.OperationLogDetail {
	out := &ipb.OperationLogDetail{Summary: summaryInternal(d.Summary), FinishedAt: formatTime(d.FinishedAt), RequestId: d.RequestID, ParentOperationId: d.ParentOperationID, BusinessRequestId: d.BusinessRequestID, Provider: d.Provider, Effect: strings.Join(d.Effect, ","), Protocol: &ipb.OperationLogProtocolResult{GrpcCode: d.Protocol.GRPCCode, HttpStatus: d.Protocol.HTTPStatus, ReasonCode: d.Protocol.ReasonCode, ResponseWriteFailed: d.Protocol.ResponseWriteFailed}, Source: &ipb.OperationLogSourceFacts{ProducerId: d.Source.ProducerID, FirstReceivedAt: timeString(&d.Source.FirstReceivedAt), LastReceivedAt: timeString(&d.Source.LastReceivedAt), PhasesReceived: d.Source.PhasesReceived, SnapshotSequence: itoa(d.Source.SnapshotSequence)}, CountFacts: &ipb.OperationLogCounts{Requested: formatPtr(d.Counts.Requested), Confirmed: formatPtr(d.Counts.Confirmed), Failed: formatPtr(d.Counts.Failed), Unknown: formatPtr(d.Counts.Unknown)}}
	for _, r := range d.Resources {
		out.ResourceFacts = append(out.ResourceFacts, &ipb.OperationLogResource{Type: r.Type, Id: r.ID, Relation: r.Relation, ReferenceVerified: r.ReferenceVerified})
	}
	for _, c := range d.Changes {
		out.ChangeFacts = append(out.ChangeFacts, &ipb.OperationLogChange{FieldCode: c.FieldCode, BeforeValue: stringOrEmpty(c.BeforeValue), AfterValue: stringOrEmpty(c.AfterValue), BeforeAvailable: c.BeforeAvailable, ValueKind: c.ValueKind})
	}
	return out
}

func runtimePublic(st query.RuntimeStatus) *pb.RuntimeStatus {
	r := &pb.RuntimeStatus{ServiceEpoch: st.ServiceEpoch, CheckedAt: st.CheckedAt.Format(time.RFC3339Nano), QueryReady: st.QueryReady, ProjectionState: st.ProjectionState, PublicationSequence: itoa(st.PublicationSequence), PendingEvents: itoa(st.PendingEvents), QuarantinedEvents: itoa(st.QuarantinedEvents), TotalObserved: itoa(st.TotalObserved), ProducersComplete: st.ProducersComplete}
	if st.LastPublishedAt != nil {
		r.LastPublishedAt = st.LastPublishedAt.Format(time.RFC3339Nano)
	}
	if st.OldestPendingReceivedAt != nil {
		r.OldestPendingReceivedAt = st.OldestPendingReceivedAt.Format(time.RFC3339Nano)
	}
	if st.LastProcessingErrorCode != nil {
		r.LastProcessingErrorCode = *st.LastProcessingErrorCode
	}
	for _, p := range st.Producers {
		r.ObservedProducers = append(r.ObservedProducers, producerPublic(p))
	}
	return r
}
func producerPublic(p query.ProducerStatus) *pb.OperationLogProducerStatus {
	return &pb.OperationLogProducerStatus{ProducerId: p.ProducerID, StartedAt: timeString(p.StartedAt), LastSeenAt: timeString(p.LastSeenAt), StoppedAt: timeString(p.StoppedAt), AttemptedEvents: itoa(p.AttemptedEvents), ConfirmedEvents: itoa(p.ConfirmedEvents), UnconfirmedEvents: itoa(p.UnconfirmedEvents), InvalidEvents: itoa(p.InvalidEvents), CapacityRejectedEvents: itoa(p.CapacityRejectedEvents), LastFailureAt: timeString(p.LastFailureAt), LastFailureCode: stringPtr(p.LastFailureCode), LastRecoveredAt: timeString(p.LastRecoveredAt), PersistenceReachable: boolPtr(p.PersistenceReachable)}
}
func runtimeInternal(st query.RuntimeStatus) *ipb.RuntimeStatus {
	r := &ipb.RuntimeStatus{ServiceEpoch: st.ServiceEpoch, CheckedAt: st.CheckedAt.Format(time.RFC3339Nano), QueryReady: st.QueryReady, ProjectionState: st.ProjectionState, PublicationSequence: itoa(st.PublicationSequence), PendingEvents: itoa(st.PendingEvents), QuarantinedEvents: itoa(st.QuarantinedEvents), TotalObserved: itoa(st.TotalObserved), ProducersComplete: st.ProducersComplete}
	if st.LastPublishedAt != nil {
		r.LastPublishedAt = st.LastPublishedAt.Format(time.RFC3339Nano)
	}
	if st.OldestPendingReceivedAt != nil {
		r.OldestPendingReceivedAt = st.OldestPendingReceivedAt.Format(time.RFC3339Nano)
	}
	if st.LastProcessingErrorCode != nil {
		r.LastProcessingErrorCode = *st.LastProcessingErrorCode
	}
	for _, p := range st.Producers {
		r.ObservedProducers = append(r.ObservedProducers, producerInternal(p))
	}
	return r
}
func producerInternal(p query.ProducerStatus) *ipb.OperationLogProducerStatus {
	return &ipb.OperationLogProducerStatus{ProducerId: p.ProducerID, StartedAt: timeString(p.StartedAt), LastSeenAt: timeString(p.LastSeenAt), StoppedAt: timeString(p.StoppedAt), AttemptedEvents: itoa(p.AttemptedEvents), ConfirmedEvents: itoa(p.ConfirmedEvents), UnconfirmedEvents: itoa(p.UnconfirmedEvents), InvalidEvents: itoa(p.InvalidEvents), CapacityRejectedEvents: itoa(p.CapacityRejectedEvents), LastFailureAt: timeString(p.LastFailureAt), LastFailureCode: stringPtr(p.LastFailureCode), LastRecoveredAt: timeString(p.LastRecoveredAt), PersistenceReachable: boolPtr(p.PersistenceReachable)}
}
func timeString(v *time.Time) string {
	if v == nil {
		return ""
	}
	return v.Format(time.RFC3339Nano)
}
func stringOrEmpty(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}
func stringPtr(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}
func boolPtr(v *bool) bool {
	if v == nil {
		return false
	}
	return *v
}

func fromInternalViewer(v *ipb.Viewer) query.Viewer {
	if v == nil {
		return query.Viewer{}
	}
	return query.Viewer{AccountID: v.AccountId, Realm: v.Realm, CredentialKind: v.CredentialKind, SessionBinding: v.SessionBinding, AccessRevision: v.AccessRevision}
}
func mapError(e error) error {
	if errors.Is(e, query.ErrCursorExpired) {
		return rpcError(codes.FailedPrecondition, "CURSOR_EXPIRED")
	}
	if errors.Is(e, query.ErrSnapshotExpired) {
		return rpcError(codes.FailedPrecondition, "SNAPSHOT_EXPIRED")
	}
	if errors.Is(e, query.ErrNotFound) {
		return rpcError(codes.NotFound, "OPERATION_LOG_NOT_FOUND")
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
	if query.ValidateViewer(v) != nil {
		return nil, rpcError(codes.PermissionDenied, "OPERATION_LOG_VIEWER_FORBIDDEN")
	}
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
	out := &ipb.ListOperationLogsResponse{Page: &ipb.Page{NextCursor: p.NextCursor, SnapshotToken: p.SnapshotToken, SnapshotSequence: itoa(p.SnapshotSequence), SnapshotAt: p.SnapshotAt.Format(time.RFC3339Nano), PageSize: int32(p.PageSize)}, AppliedFilters: appliedInternal(p.AppliedFilters)}
	for _, x := range p.Items {
		out.Items = append(out.Items, summaryInternal(x))
	}
	return out, nil
}
func (s *InternalServer) GetOperationLog(ctx context.Context, r *ipb.GetOperationLogRequest) (*ipb.GetOperationLogResponse, error) {
	v := fromInternalViewer(r.GetViewer())
	if query.ValidateViewer(v) != nil {
		return nil, rpcError(codes.PermissionDenied, "OPERATION_LOG_VIEWER_FORBIDDEN")
	}
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
	return &ipb.GetOperationLogResponse{Item: detailInternal(d)}, nil
}
func (s *InternalServer) GetOperationLogRuntimeStatus(ctx context.Context, r *ipb.GetOperationLogRuntimeStatusRequest) (*ipb.GetOperationLogRuntimeStatusResponse, error) {
	v := fromInternalViewer(r.GetViewer())
	if query.ValidateViewer(v) != nil {
		return nil, rpcError(codes.PermissionDenied, "OPERATION_LOG_VIEWER_FORBIDDEN")
	}
	if s.Access != nil {
		if err := s.Access.Check(ctx, v.AccountID); err != nil {
			return nil, rpcError(codes.PermissionDenied, "ACCOUNT_ADMIN_REQUIRED")
		}
	}
	if s.Adapter == nil {
		return nil, rpcError(codes.Unavailable, "OPERATION_LOG_UNAVAILABLE")
	}
	st, err := s.Adapter.Runtime(ctx)
	if err != nil {
		return nil, mapError(err)
	}
	return &ipb.GetOperationLogRuntimeStatusResponse{Status: runtimeInternal(st)}, nil
}
