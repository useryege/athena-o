package tradersync

import (
	"context"
	trpc "github.com/useryege/athena/internal/tradersync/apiclient"
	api "github.com/useryege/athena/pkg/apiclient/tradersync"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ActorResolver func(context.Context) (*trpc.Actor, error)
type Server struct {
	api.UnimplementedTraderSyncServiceServer
	client trpc.TraderSyncServiceClient
	actors ActorResolver
}

func New(client trpc.TraderSyncServiceClient, actors ActorResolver) *Server {
	if client == nil {
		client = trpc.NewUnavailableClient("dependency_unavailable")
	}
	return &Server{client: client, actors: actors}
}
func (s *Server) actor(ctx context.Context) (*trpc.Actor, error) {
	if s.actors == nil {
		return nil, status.Error(codes.Unavailable, "Trader Sync actor resolver unavailable")
	}
	a, e := s.actors(ctx)
	if e != nil {
		return nil, e
	}
	if a == nil {
		return nil, status.Error(codes.Unavailable, "Trader Sync actor contract invalid")
	}
	return a, nil
}
func internalPage(v *api.PageInput) *trpc.PageInput {
	if v == nil {
		return nil
	}
	return &trpc.PageInput{PageSize: v.PageSize, Cursor: v.Cursor}
}
func internalNote(v *api.TraderSyncNoteInput) *trpc.NoteInput {
	if v == nil {
		return nil
	}
	return &trpc.NoteInput{Value: v.Value}
}
func (s *Server) ResolveTarget(ctx context.Context, r *api.ResolveTargetRequest) (*api.ResolveTargetResponse, error) {
	actor, err := s.actor(ctx)
	if err != nil {
		return nil, err
	}
	v, err := s.client.ResolveTarget(ctx, &trpc.ResolveTargetRequest{Actor: actor, Input: r.GetInput()})
	if err != nil {
		return nil, MapInternalError(err)
	}
	m := &responseMapper{}
	out := m.mapResolveTargetResponse(v)
	if m.err != nil {
		return nil, m.err
	}
	return out, nil
}
func (s *Server) CreateSubscription(ctx context.Context, r *api.CreateSubscriptionRequest) (*api.CreateSubscriptionResponse, error) {
	actor, err := s.actor(ctx)
	if err != nil {
		return nil, err
	}
	v, err := s.client.CreateSubscription(ctx, &trpc.CreateSubscriptionRequest{Actor: actor, ConfirmationToken: r.GetConfirmationToken(), RequestId: r.GetRequestId(), Note: internalNote(r.GetNote())})
	if err != nil {
		return nil, MapInternalError(err)
	}
	m := &responseMapper{}
	out := m.mapCreateSubscriptionResponse(v)
	if m.err != nil {
		return nil, m.err
	}
	return out, nil
}
func (s *Server) ListSubscriptions(ctx context.Context, r *api.ListSubscriptionsRequest) (*api.ListSubscriptionsResponse, error) {
	actor, err := s.actor(ctx)
	if err != nil {
		return nil, err
	}
	v, err := s.client.ListSubscriptions(ctx, &trpc.ListSubscriptionsRequest{Actor: actor, Page: internalPage(r.GetPage()), View: r.GetView(), State: r.GetState()})
	if err != nil {
		return nil, MapInternalError(err)
	}
	m := &responseMapper{}
	out := m.mapListSubscriptionsResponse(v)
	if m.err != nil {
		return nil, m.err
	}
	return out, nil
}
func (s *Server) GetSubscription(ctx context.Context, r *api.GetSubscriptionRequest) (*api.GetSubscriptionResponse, error) {
	actor, err := s.actor(ctx)
	if err != nil {
		return nil, err
	}
	v, err := s.client.GetSubscription(ctx, &trpc.GetSubscriptionRequest{Actor: actor, SubscriptionId: r.GetSubscriptionId()})
	if err != nil {
		return nil, MapInternalError(err)
	}
	m := &responseMapper{}
	out := m.mapGetSubscriptionResponse(v)
	if m.err != nil {
		return nil, m.err
	}
	return out, nil
}
func (s *Server) PauseSubscription(ctx context.Context, r *api.PauseSubscriptionRequest) (*api.PauseSubscriptionResponse, error) {
	actor, err := s.actor(ctx)
	if err != nil {
		return nil, err
	}
	v, err := s.client.PauseSubscription(ctx, &trpc.PauseSubscriptionRequest{Actor: actor, SubscriptionId: r.GetSubscriptionId(), ExpectedRevision: r.GetExpectedRevision(), RequestId: r.GetRequestId()})
	if err != nil {
		return nil, MapInternalError(err)
	}
	m := &responseMapper{}
	out := m.mapPauseSubscriptionResponse(v)
	if m.err != nil {
		return nil, m.err
	}
	return out, nil
}
func (s *Server) ResumeSubscription(ctx context.Context, r *api.ResumeSubscriptionRequest) (*api.ResumeSubscriptionResponse, error) {
	actor, err := s.actor(ctx)
	if err != nil {
		return nil, err
	}
	v, err := s.client.ResumeSubscription(ctx, &trpc.ResumeSubscriptionRequest{Actor: actor, SubscriptionId: r.GetSubscriptionId(), ExpectedRevision: r.GetExpectedRevision(), RequestId: r.GetRequestId()})
	if err != nil {
		return nil, MapInternalError(err)
	}
	m := &responseMapper{}
	out := m.mapResumeSubscriptionResponse(v)
	if m.err != nil {
		return nil, m.err
	}
	return out, nil
}
func (s *Server) CancelSubscription(ctx context.Context, r *api.CancelSubscriptionRequest) (*api.CancelSubscriptionResponse, error) {
	actor, err := s.actor(ctx)
	if err != nil {
		return nil, err
	}
	v, err := s.client.CancelSubscription(ctx, &trpc.CancelSubscriptionRequest{Actor: actor, SubscriptionId: r.GetSubscriptionId(), ExpectedRevision: r.GetExpectedRevision(), RequestId: r.GetRequestId()})
	if err != nil {
		return nil, MapInternalError(err)
	}
	m := &responseMapper{}
	out := m.mapCancelSubscriptionResponse(v)
	if m.err != nil {
		return nil, m.err
	}
	return out, nil
}
func (s *Server) UpdateTargetNote(ctx context.Context, r *api.UpdateTargetNoteRequest) (*api.UpdateTargetNoteResponse, error) {
	actor, err := s.actor(ctx)
	if err != nil {
		return nil, err
	}
	v, err := s.client.UpdateTargetNote(ctx, &trpc.UpdateTargetNoteRequest{Actor: actor, Wallet: r.GetWallet(), Note: r.GetNote(), ExpectedRevision: r.GetExpectedRevision(), RequestId: r.GetRequestId()})
	if err != nil {
		return nil, MapInternalError(err)
	}
	m := &responseMapper{}
	out := m.mapUpdateTargetNoteResponse(v)
	if m.err != nil {
		return nil, m.err
	}
	return out, nil
}
func (s *Server) ListActivities(ctx context.Context, r *api.ListActivitiesRequest) (*api.ListActivitiesResponse, error) {
	actor, err := s.actor(ctx)
	if err != nil {
		return nil, err
	}
	v, err := s.client.ListActivities(ctx, &trpc.ListActivitiesRequest{Actor: actor, Page: internalPage(r.GetPage()), SubscriptionId: r.GetSubscriptionId(), From: r.GetFrom(), To: r.GetTo(), SummaryBatchId: r.GetSummaryBatchId(), RefreshCursor: r.GetRefreshCursor()})
	if err != nil {
		return nil, MapInternalError(err)
	}
	m := &responseMapper{}
	out := m.mapListActivitiesResponse(v)
	if m.err != nil {
		return nil, m.err
	}
	return out, nil
}
func (s *Server) GetActivity(ctx context.Context, r *api.GetActivityRequest) (*api.GetActivityResponse, error) {
	actor, err := s.actor(ctx)
	if err != nil {
		return nil, err
	}
	v, err := s.client.GetActivity(ctx, &trpc.GetActivityRequest{Actor: actor, ActivityId: r.GetActivityId()})
	if err != nil {
		return nil, MapInternalError(err)
	}
	m := &responseMapper{}
	out := m.mapGetActivityResponse(v)
	if m.err != nil {
		return nil, m.err
	}
	return out, nil
}
func (s *Server) ListSubscriptionHistory(ctx context.Context, r *api.ListSubscriptionHistoryRequest) (*api.ListSubscriptionHistoryResponse, error) {
	actor, err := s.actor(ctx)
	if err != nil {
		return nil, err
	}
	v, err := s.client.ListSubscriptionHistory(ctx, &trpc.ListSubscriptionHistoryRequest{Actor: actor, SubscriptionId: r.GetSubscriptionId(), Page: internalPage(r.GetPage())})
	if err != nil {
		return nil, MapInternalError(err)
	}
	m := &responseMapper{}
	out := m.mapListSubscriptionHistoryResponse(v)
	if m.err != nil {
		return nil, m.err
	}
	return out, nil
}
func (s *Server) GetSummaryBatch(ctx context.Context, r *api.GetSummaryBatchRequest) (*api.GetSummaryBatchResponse, error) {
	actor, err := s.actor(ctx)
	if err != nil {
		return nil, err
	}
	v, err := s.client.GetSummaryBatch(ctx, &trpc.GetSummaryBatchRequest{Actor: actor, BatchId: r.GetBatchId()})
	if err != nil {
		return nil, MapInternalError(err)
	}
	m := &responseMapper{}
	out := m.mapGetSummaryBatchResponse(v)
	if m.err != nil {
		return nil, m.err
	}
	return out, nil
}
func (s *Server) ListSummaryParts(ctx context.Context, r *api.ListSummaryPartsRequest) (*api.ListSummaryPartsResponse, error) {
	actor, err := s.actor(ctx)
	if err != nil {
		return nil, err
	}
	v, err := s.client.ListSummaryParts(ctx, &trpc.ListSummaryPartsRequest{Actor: actor, BatchId: r.GetBatchId(), ActivityId: r.GetActivityId(), Page: internalPage(r.GetPage())})
	if err != nil {
		return nil, MapInternalError(err)
	}
	m := &responseMapper{}
	out := m.mapListSummaryPartsResponse(v)
	if m.err != nil {
		return nil, m.err
	}
	return out, nil
}
func (s *Server) ListSubscriptionSummaries(ctx context.Context, r *api.ListSubscriptionSummariesRequest) (*api.ListSubscriptionSummariesResponse, error) {
	actor, err := s.actor(ctx)
	if err != nil {
		return nil, err
	}
	v, err := s.client.ListSubscriptionSummaries(ctx, &trpc.ListSubscriptionSummariesRequest{Actor: actor, Page: internalPage(r.GetPage()), AccountId: r.GetAccountId(), State: r.GetState(), Wallet: r.GetWallet(), IncludeCancelled: r.GetIncludeCancelled()})
	if err != nil {
		return nil, MapInternalError(err)
	}
	m := &responseMapper{}
	out := m.mapListSubscriptionSummariesResponse(v)
	if m.err != nil {
		return nil, m.err
	}
	return out, nil
}
func (s *Server) GetSubscriptionSummary(ctx context.Context, r *api.GetSubscriptionSummaryRequest) (*api.GetSubscriptionSummaryResponse, error) {
	actor, err := s.actor(ctx)
	if err != nil {
		return nil, err
	}
	v, err := s.client.GetSubscriptionSummary(ctx, &trpc.GetSubscriptionSummaryRequest{Actor: actor, SubscriptionId: r.GetSubscriptionId()})
	if err != nil {
		return nil, MapInternalError(err)
	}
	m := &responseMapper{}
	out := m.mapGetSubscriptionSummaryResponse(v)
	if m.err != nil {
		return nil, m.err
	}
	return out, nil
}
func (s *Server) GetTraderSyncRuntimeStatus(ctx context.Context, r *api.GetTraderSyncRuntimeStatusRequest) (*api.GetTraderSyncRuntimeStatusResponse, error) {
	actor, err := s.actor(ctx)
	if err != nil {
		return nil, err
	}
	v, err := s.client.GetTraderSyncRuntimeStatus(ctx, &trpc.GetTraderSyncRuntimeStatusRequest{Actor: actor})
	if err != nil {
		return nil, MapInternalError(err)
	}
	m := &responseMapper{}
	out := m.mapGetTraderSyncRuntimeStatusResponse(v)
	if m.err != nil {
		return nil, m.err
	}
	return out, nil
}
