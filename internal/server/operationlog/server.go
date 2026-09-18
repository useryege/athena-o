// Package operationlog exposes the administrator operation-log facade.
package operationlog

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
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
			return query.Viewer{}, rpcError(codes.PermissionDenied, "ACCOUNT_ADMIN_REQUIRED")
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
func parseRange(from, to string) (*time.Time, *time.Time, error) {
	if (from == "") != (to == "") {
		return nil, nil, fmt.Errorf("from and to must be supplied together")
	}
	if from == "" {
		return nil, nil, nil
	}
	f, err := time.Parse(time.RFC3339Nano, from)
	if err != nil {
		return nil, nil, err
	}
	t, err := time.Parse(time.RFC3339Nano, to)
	if err != nil {
		return nil, nil, err
	}
	return &f, &t, nil
}
func internalFilter(r *ipb.ListOperationLogsRequest) (query.Filter, error) {
	f := query.Filter{ActorQuery: r.GetActorQuery(), ActorRole: r.GetActorRole(), CredentialKind: r.GetCredentialKind(), ModuleCode: r.GetModuleCode(), ActionCode: r.GetActionCode(), Outcome: r.GetOutcome(), TargetAccountID: r.GetTargetAccountId(), ResourceType: r.GetResourceType(), ResourceID: r.GetResourceId(), PageSize: int(r.GetPageSize()), Cursor: r.GetCursor()}
	from, to, err := parseRange(r.GetFrom(), r.GetTo())
	if err != nil {
		return f, err
	}
	f.From, f.To = from, to
	return f, nil
}
func (s *Server) ListOperationLogs(ctx context.Context, r *pb.ListOperationLogsRequest) (*pb.ListOperationLogsResponse, error) {
	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	v, e := s.resolve(ctx)
	if e != nil {
		return nil, e
	}
	f, e := requestFilter(r)
	if e != nil {
		return nil, rpcError(codes.InvalidArgument, "FILTER_INVALID")
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
		out.Items = append(out.Items, summaryPublic(x))
	}
	return out, nil
}
func (s *Server) GetOperationLog(ctx context.Context, r *pb.GetOperationLogRequest) (*pb.GetOperationLogResponse, error) {
	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
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
	sm := summaryPublic(d.Summary)
	det := &pb.OperationLogDetail{Summary: sm, FinishedAt: optionalTimePB(d.FinishedAt), RequestId: optionalValuePB(d.RequestID), ParentOperationId: optionalValuePB(d.ParentOperationID), BusinessRequestId: optionalValuePB(d.BusinessRequestID), Provider: optionalValuePB(d.Provider), Effect: append([]string(nil), d.Effect...), CountFacts: countsPublic(d.Counts)}
	for _, r := range d.Resources {
		det.ResourceFacts = append(det.ResourceFacts, &pb.OperationLogResource{Type: r.Type, Id: r.ID, Relation: r.Relation, ReferenceVerified: nullableBoolPB(r.ReferenceVerified)})
	}
	for _, c := range d.Changes {
		det.ChangeFacts = append(det.ChangeFacts, changePublic(c))
	}
	det.Protocol = &pb.OperationLogProtocolResult{GrpcCode: optionalValuePB(d.Protocol.GRPCCode), HttpStatus: optionalValuePB(d.Protocol.HTTPStatus), ReasonCode: optionalValuePB(d.Protocol.ReasonCode), ResponseWriteFailed: nullableBoolPB(d.Protocol.ResponseWriteFailed)}
	det.Source = sourcePublic(d.Source)
	return &pb.GetOperationLogResponse{Item: det}, nil
}
func (s *Server) GetOperationLogRuntimeStatus(ctx context.Context, _ *pb.GetOperationLogRuntimeStatusRequest) (*pb.GetOperationLogRuntimeStatusResponse, error) {
	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(ctx, time.Second)
	defer cancel()
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
	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()
	if _, err := s.resolve(ctx); err != nil {
		return nil, err
	}
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
	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()
	if _, err := s.resolve(ctx); err != nil {
		return nil, err
	}
	if s.Actions == nil {
		return &pb.ListOperationLogActionsResponse{}, nil
	}
	return s.Actions(ctx)
}
func appliedInternal(f query.NormalizedFilter) *ipb.AppliedFilters {
	return &ipb.AppliedFilters{From: f.From.Format(time.RFC3339Nano), To: f.To.Format(time.RFC3339Nano), ActorQuery: f.ActorQuery, ActorRole: f.ActorRole, CredentialKind: f.CredentialKind, ModuleCode: f.ModuleCode, ActionCode: f.ActionCode, Outcome: f.Outcome, TargetAccountId: f.TargetAccountID, ResourceType: f.ResourceType, ResourceId: f.ResourceID}
}
func summaryPublic(x query.Summary) *pb.OperationLogSummary {
	return &pb.OperationLogSummary{OperationId: x.OperationID, StartedAt: nullableStringPB(x.StartedAt.Format(time.RFC3339Nano)), ActorAccountId: optionalValuePB(x.ActorAccountID), ActorUsername: optionalValuePB(x.ActorUsername), ActorRole: x.ActorRole, Realm: x.Realm, CredentialKind: x.CredentialKind, IdentityVerified: nullableBoolPB(x.IdentityVerified), IdentitySnapshotComplete: nullableBoolPB(x.IdentitySnapshotComplete), ModuleCode: x.ModuleCode, ActionCode: x.ActionCode, PrimaryResourceType: optionalValuePB(x.PrimaryResourceType), PrimaryResourceId: optionalValuePB(x.PrimaryResourceID), TargetAccountId: optionalValuePB(x.TargetAccountID), Outcome: x.Outcome, Observation: x.Observation, ReasonCode: optionalValuePB(x.ReasonCode), DurationMs: optionalIntPB(x.DurationMS), BusinessState: optionalValuePB(x.BusinessState), ResourceCount: optionalIntPB(x.ResourceCount), ResourcesComplete: nullableBoolPB(x.ResourcesComplete), ResponseWriteFailed: nullableBoolPB(x.ResponseWriteFailed), Provider: optionalValuePB(x.Provider)}
}
func countsPublic(c query.CountsFact) *pb.OperationLogCounts {
	return &pb.OperationLogCounts{Requested: optionalIntStringPB(c.Requested), Confirmed: optionalIntStringPB(c.Confirmed), Failed: optionalIntStringPB(c.Failed), Unknown: optionalIntStringPB(c.Unknown)}
}
func changePublic(c query.ChangeFact) *pb.OperationLogChange {
	return &pb.OperationLogChange{FieldCode: c.FieldCode, BeforeValue: optionalStringPB(c.BeforeValue), AfterValue: optionalStringPB(c.AfterValue), BeforeAvailable: nullableBoolPB(c.BeforeAvailable), ValueKind: c.ValueKind}
}
func changeInternal(c query.ChangeFact) *ipb.OperationLogChange {
	return &ipb.OperationLogChange{FieldCode: c.FieldCode, BeforeValue: optionalStringInternal(c.BeforeValue), AfterValue: optionalStringInternal(c.AfterValue), BeforeAvailable: nullableBoolInternal(c.BeforeAvailable), ValueKind: c.ValueKind}
}
func sourcePublic(s query.SourceFacts) *pb.OperationLogSourceFacts {
	out := &pb.OperationLogSourceFacts{ProducerId: optionalValuePB(s.ProducerID), PhasesReceived: s.PhasesReceived, SnapshotSequence: nullableStringPB(itoa(s.SnapshotSequence))}
	if !s.FirstReceivedAt.IsZero() {
		out.FirstReceivedAt = nullableStringPB(s.FirstReceivedAt.Format(time.RFC3339Nano))
	}
	if !s.LastReceivedAt.IsZero() {
		out.LastReceivedAt = nullableStringPB(s.LastReceivedAt.Format(time.RFC3339Nano))
	}
	return out
}
func nullableStringPB(v string) *pb.NullableString { return &pb.NullableString{Value: v} }
func optionalValuePB(v string) *pb.NullableString {
	if v == "" {
		return nil
	}
	return nullableStringPB(v)
}
func nullableBoolPB(v bool) *pb.NullableBool { return &pb.NullableBool{Value: v} }
func optionalStringPB(v *string) *pb.NullableString {
	if v == nil {
		return nil
	}
	return nullableStringPB(*v)
}
func optionalTimePB(v *time.Time) *pb.NullableString {
	if v == nil || v.IsZero() {
		return nil
	}
	return nullableStringPB(v.Format(time.RFC3339Nano))
}
func optionalIntPB(v *int64) *pb.NullableString {
	if v == nil {
		return nil
	}
	return nullableStringPB(fmt.Sprintf("%d", *v))
}
func optionalIntStringPB(v *int64) *pb.NullableString { return optionalIntPB(v) }
func optionalBoolPB(v *bool) *pb.NullableBool {
	if v == nil {
		return nil
	}
	return nullableBoolPB(*v)
}
func nullableStringInternal(v string) *ipb.NullableString { return &ipb.NullableString{Value: v} }
func optionalValueInternal(v string) *ipb.NullableString {
	if v == "" {
		return nil
	}
	return nullableStringInternal(v)
}
func nullableBoolInternal(v bool) *ipb.NullableBool { return &ipb.NullableBool{Value: v} }
func optionalStringInternal(v *string) *ipb.NullableString {
	if v == nil {
		return nil
	}
	return nullableStringInternal(*v)
}
func optionalTimeInternal(v *time.Time) *ipb.NullableString {
	if v == nil || v.IsZero() {
		return nil
	}
	return nullableStringInternal(v.Format(time.RFC3339Nano))
}
func optionalIntInternal(v *int64) *ipb.NullableString {
	if v == nil {
		return nil
	}
	return nullableStringInternal(fmt.Sprintf("%d", *v))
}
func optionalBoolInternal(v *bool) *ipb.NullableBool {
	if v == nil {
		return nil
	}
	return nullableBoolInternal(*v)
}
func sourceInternal(s query.SourceFacts) *ipb.OperationLogSourceFacts {
	out := &ipb.OperationLogSourceFacts{ProducerId: optionalValueInternal(s.ProducerID), PhasesReceived: s.PhasesReceived, SnapshotSequence: nullableStringInternal(itoa(s.SnapshotSequence))}
	if !s.FirstReceivedAt.IsZero() {
		out.FirstReceivedAt = nullableStringInternal(s.FirstReceivedAt.Format(time.RFC3339Nano))
	}
	if !s.LastReceivedAt.IsZero() {
		out.LastReceivedAt = nullableStringInternal(s.LastReceivedAt.Format(time.RFC3339Nano))
	}
	return out
}
func summaryInternal(x query.Summary) *ipb.OperationLogSummary {
	return &ipb.OperationLogSummary{OperationId: x.OperationID, StartedAt: nullableStringInternal(x.StartedAt.Format(time.RFC3339Nano)), ActorAccountId: optionalValueInternal(x.ActorAccountID), ActorUsername: optionalValueInternal(x.ActorUsername), ActorRole: x.ActorRole, Realm: x.Realm, CredentialKind: x.CredentialKind, IdentityVerified: nullableBoolInternal(x.IdentityVerified), IdentitySnapshotComplete: nullableBoolInternal(x.IdentitySnapshotComplete), ModuleCode: x.ModuleCode, ActionCode: x.ActionCode, PrimaryResourceType: optionalValueInternal(x.PrimaryResourceType), PrimaryResourceId: optionalValueInternal(x.PrimaryResourceID), TargetAccountId: optionalValueInternal(x.TargetAccountID), Outcome: x.Outcome, Observation: x.Observation, ReasonCode: optionalValueInternal(x.ReasonCode), DurationMs: optionalIntInternal(x.DurationMS), BusinessState: optionalValueInternal(x.BusinessState), ResourceCount: optionalIntInternal(x.ResourceCount), ResourcesComplete: nullableBoolInternal(x.ResourcesComplete), ResponseWriteFailed: nullableBoolInternal(x.ResponseWriteFailed), Provider: optionalValueInternal(x.Provider)}
}
func detailInternal(d query.Detail) *ipb.OperationLogDetail {
	out := &ipb.OperationLogDetail{Summary: summaryInternal(d.Summary), FinishedAt: optionalTimeInternal(d.FinishedAt), RequestId: optionalValueInternal(d.RequestID), ParentOperationId: optionalValueInternal(d.ParentOperationID), BusinessRequestId: optionalValueInternal(d.BusinessRequestID), Provider: optionalValueInternal(d.Provider), Effect: append([]string(nil), d.Effect...), Protocol: &ipb.OperationLogProtocolResult{GrpcCode: optionalValueInternal(d.Protocol.GRPCCode), HttpStatus: optionalValueInternal(d.Protocol.HTTPStatus), ReasonCode: optionalValueInternal(d.Protocol.ReasonCode), ResponseWriteFailed: nullableBoolInternal(d.Protocol.ResponseWriteFailed)}, Source: sourceInternal(d.Source), CountFacts: &ipb.OperationLogCounts{Requested: optionalIntInternal(d.Counts.Requested), Confirmed: optionalIntInternal(d.Counts.Confirmed), Failed: optionalIntInternal(d.Counts.Failed), Unknown: optionalIntInternal(d.Counts.Unknown)}}
	for _, r := range d.Resources {
		out.ResourceFacts = append(out.ResourceFacts, &ipb.OperationLogResource{Type: r.Type, Id: r.ID, Relation: r.Relation, ReferenceVerified: nullableBoolInternal(r.ReferenceVerified)})
	}
	for _, c := range d.Changes {
		out.ChangeFacts = append(out.ChangeFacts, changeInternal(c))
	}
	return out
}

func runtimePublic(st query.RuntimeStatus) *pb.RuntimeStatus {
	r := &pb.RuntimeStatus{ServiceEpoch: st.ServiceEpoch, CheckedAt: nullableStringPB(st.CheckedAt.Format(time.RFC3339Nano)), QueryReady: nullableBoolPB(st.QueryReady), ProjectionState: st.ProjectionState, PublicationSequence: itoa(st.PublicationSequence), QuarantinedEvents: itoa(st.QuarantinedEvents), TotalObserved: optionalIntPB(&st.TotalObserved), ProducersComplete: nullableBoolPB(st.ProducersComplete)}
	if st.PendingEvents >= 0 {
		r.PendingEvents = optionalIntPB(&st.PendingEvents)
	}
	if st.LastPublishedAt != nil {
		r.LastPublishedAt = nullableStringPB(st.LastPublishedAt.Format(time.RFC3339Nano))
	}
	if st.OldestPendingReceivedAt != nil {
		r.OldestPendingReceivedAt = nullableStringPB(st.OldestPendingReceivedAt.Format(time.RFC3339Nano))
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
	return &pb.OperationLogProducerStatus{ProducerId: optionalValuePB(p.ProducerID), StartedAt: optionalTimePB(p.StartedAt), LastSeenAt: optionalTimePB(p.LastSeenAt), StoppedAt: optionalTimePB(p.StoppedAt), AttemptedEvents: optionalIntPB(&p.AttemptedEvents), ConfirmedEvents: optionalIntPB(&p.ConfirmedEvents), UnconfirmedEvents: optionalIntPB(&p.UnconfirmedEvents), InvalidEvents: optionalIntPB(&p.InvalidEvents), CapacityRejectedEvents: optionalIntPB(&p.CapacityRejectedEvents), LastFailureAt: optionalTimePB(p.LastFailureAt), LastFailureCode: optionalStringPB(p.LastFailureCode), LastRecoveredAt: optionalTimePB(p.LastRecoveredAt), PersistenceReachable: optionalBoolPB(p.PersistenceReachable)}
}
func runtimeInternal(st query.RuntimeStatus) *ipb.RuntimeStatus {
	r := &ipb.RuntimeStatus{ServiceEpoch: st.ServiceEpoch, CheckedAt: nullableStringInternal(st.CheckedAt.Format(time.RFC3339Nano)), QueryReady: nullableBoolInternal(st.QueryReady), ProjectionState: st.ProjectionState, PublicationSequence: itoa(st.PublicationSequence), QuarantinedEvents: itoa(st.QuarantinedEvents), TotalObserved: optionalIntInternal(&st.TotalObserved), ProducersComplete: nullableBoolInternal(st.ProducersComplete)}
	if st.PendingEvents >= 0 {
		r.PendingEvents = optionalIntInternal(&st.PendingEvents)
	}
	if st.LastPublishedAt != nil {
		r.LastPublishedAt = nullableStringInternal(st.LastPublishedAt.Format(time.RFC3339Nano))
	}
	if st.OldestPendingReceivedAt != nil {
		r.OldestPendingReceivedAt = nullableStringInternal(st.OldestPendingReceivedAt.Format(time.RFC3339Nano))
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
	return &ipb.OperationLogProducerStatus{ProducerId: optionalValueInternal(p.ProducerID), StartedAt: optionalTimeInternal(p.StartedAt), LastSeenAt: optionalTimeInternal(p.LastSeenAt), StoppedAt: optionalTimeInternal(p.StoppedAt), AttemptedEvents: optionalIntInternal(&p.AttemptedEvents), ConfirmedEvents: optionalIntInternal(&p.ConfirmedEvents), UnconfirmedEvents: optionalIntInternal(&p.UnconfirmedEvents), InvalidEvents: optionalIntInternal(&p.InvalidEvents), CapacityRejectedEvents: optionalIntInternal(&p.CapacityRejectedEvents), LastFailureAt: optionalTimeInternal(p.LastFailureAt), LastFailureCode: optionalStringInternal(p.LastFailureCode), LastRecoveredAt: optionalTimeInternal(p.LastRecoveredAt), PersistenceReachable: optionalBoolInternal(p.PersistenceReachable)}
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
	if errors.Is(e, query.ErrInvalidOperationID) {
		return rpcError(codes.InvalidArgument, "OPERATION_LOG_INVALID_OPERATION_ID")
	}
	if errors.Is(e, context.DeadlineExceeded) {
		return rpcError(codes.DeadlineExceeded, "OPERATION_LOG_QUERY_TIMEOUT")
	}
	var pgErr *pgconn.PgError
	if errors.As(e, &pgErr) && pgErr.Code == "57014" {
		return rpcError(codes.DeadlineExceeded, "OPERATION_LOG_QUERY_TIMEOUT")
	}
	if errors.Is(e, pgx.ErrNoRows) {
		return rpcError(codes.NotFound, "OPERATION_LOG_NOT_FOUND")
	}
	if errors.Is(e, query.ErrCursorExpired) {
		return rpcError(codes.FailedPrecondition, "CURSOR_EXPIRED")
	}
	if errors.Is(e, query.ErrSnapshotExpired) {
		return rpcError(codes.FailedPrecondition, "SNAPSHOT_EXPIRED")
	}
	if errors.Is(e, query.ErrSnapshotInvalid) {
		return rpcError(codes.InvalidArgument, "SNAPSHOT_INVALID")
	}
	if errors.Is(e, query.ErrNotFound) {
		return rpcError(codes.NotFound, "OPERATION_LOG_NOT_FOUND")
	}
	if errors.Is(e, query.ErrCursorInvalid) {
		return rpcError(codes.InvalidArgument, "CURSOR_INVALID")
	}
	if errors.Is(e, query.ErrInvalidFilter) {
		return rpcError(codes.InvalidArgument, "FILTER_INVALID")
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
	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	v := fromInternalViewer(r.GetViewer())
	if query.ValidateViewer(v) != nil {
		return nil, rpcError(codes.PermissionDenied, "OPERATION_LOG_VIEWER_FORBIDDEN")
	}
	if s.Access != nil {
		if err := s.Access.Check(ctx, v.AccountID); err != nil {
			if errors.Is(err, access.ErrUnavailable) {
				return nil, rpcError(codes.Unavailable, "OPERATION_LOG_ACCOUNT_UNAVAILABLE")
			}
			return nil, rpcError(codes.PermissionDenied, "ACCOUNT_ADMIN_REQUIRED")
		}
	}
	if s.Adapter == nil {
		return nil, rpcError(codes.Unavailable, "OPERATION_LOG_UNAVAILABLE")
	}
	f, e := internalFilter(r)
	if e != nil {
		return nil, rpcError(codes.InvalidArgument, "FILTER_INVALID")
	}
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
	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	v := fromInternalViewer(r.GetViewer())
	if query.ValidateViewer(v) != nil {
		return nil, rpcError(codes.PermissionDenied, "OPERATION_LOG_VIEWER_FORBIDDEN")
	}
	if s.Access != nil {
		if err := s.Access.Check(ctx, v.AccountID); err != nil {
			if errors.Is(err, access.ErrUnavailable) {
				return nil, rpcError(codes.Unavailable, "OPERATION_LOG_ACCOUNT_UNAVAILABLE")
			}
			return nil, rpcError(codes.PermissionDenied, "ACCOUNT_ADMIN_REQUIRED")
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
	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(ctx, time.Second)
	defer cancel()
	v := fromInternalViewer(r.GetViewer())
	if query.ValidateViewer(v) != nil {
		return nil, rpcError(codes.PermissionDenied, "OPERATION_LOG_VIEWER_FORBIDDEN")
	}
	if s.Access != nil {
		if err := s.Access.Check(ctx, v.AccountID); err != nil {
			if errors.Is(err, access.ErrUnavailable) {
				return nil, rpcError(codes.Unavailable, "OPERATION_LOG_ACCOUNT_UNAVAILABLE")
			}
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
