package transport

import (
	"context"
	"github.com/ethereum/go-ethereum/common"
	ts "github.com/useryege/athena/internal/tradersync"
	trpc "github.com/useryege/athena/internal/tradersync/apiclient"
	tm "github.com/useryege/athena/internal/tradersync/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"strconv"
	"strings"
	"time"
)

type Server struct {
	trpc.UnimplementedTraderSyncServiceServer
	service *ts.Service
}

func NewServer(service *ts.Service) *Server { return &Server{service: service} }
func principal(actor *trpc.Actor) (string, error) {
	if err := ValidateActor(actor); err != nil {
		return "", err
	}
	return actor.AccountId, nil
}

func parseTimeFilter(v string) (*time.Time, error) {
	if v == "" {
		return nil, nil
	}
	at, e := time.Parse(time.RFC3339Nano, v)
	if e != nil {
		return nil, status.Error(codes.InvalidArgument, "UTC time filter required")
	}
	at = at.UTC()
	return &at, nil
}
func parseWallet(v string) (common.Address, error) {
	if !common.IsHexAddress(v) {
		return common.Address{}, status.Error(codes.InvalidArgument, "wallet address required")
	}
	wallet := common.HexToAddress(v)
	if wallet == (common.Address{}) {
		return wallet, status.Error(codes.InvalidArgument, "nonzero wallet required")
	}
	return wallet, nil
}

func (s *Server) ResolveTarget(ctx context.Context, r *trpc.ResolveTargetRequest) (*trpc.ResolveTargetResponse, error) {
	owner, e := principal(r.GetActor())
	if e != nil {
		return nil, e
	}
	if strings.TrimSpace(r.GetInput()) == "" {
		return nil, status.Error(codes.InvalidArgument, "target input required")
	}
	v, e := s.service.ResolveTarget(ctx, owner, r.GetInput())
	if e != nil {
		return nil, e
	}
	return &trpc.ResolveTargetResponse{Target: internalResolved(v)}, nil
}
func (s *Server) CreateSubscription(ctx context.Context, r *trpc.CreateSubscriptionRequest) (*trpc.CreateSubscriptionResponse, error) {
	owner, e := principal(r.GetActor())
	if e != nil {
		return nil, e
	}
	var note *string
	if wrapper := r.GetNote(); wrapper != nil {
		value := wrapper.Value
		note = &value
	}
	v, e := s.service.CreateSubscription(ctx, owner, tm.CreateInput{Token: r.GetConfirmationToken(), RequestID: r.GetRequestId(), Note: note})
	if e != nil {
		return nil, e
	}
	return &trpc.CreateSubscriptionResponse{Subscription: internalSubscription(v)}, nil
}
func (s *Server) GetSubscription(ctx context.Context, r *trpc.GetSubscriptionRequest) (*trpc.GetSubscriptionResponse, error) {
	owner, e := principal(r.GetActor())
	if e != nil {
		return nil, e
	}
	v, e := s.service.GetSubscription(ctx, owner, r.GetSubscriptionId())
	if e != nil {
		return nil, e
	}
	return &trpc.GetSubscriptionResponse{Subscription: internalSubscription(v)}, nil
}
func (s *Server) ListSubscriptions(ctx context.Context, r *trpc.ListSubscriptionsRequest) (*trpc.ListSubscriptionsResponse, error) {
	owner, e := principal(r.GetActor())
	if e != nil {
		return nil, e
	}
	v, e := s.service.ListSubscriptions(ctx, owner, r.GetPage().GetPageSize(), r.GetPage().GetCursor(), tm.SubscriptionFilter{View: r.GetView(), State: r.GetState()})
	if e != nil {
		return nil, e
	}
	return &trpc.ListSubscriptionsResponse{Subscriptions: mapItems(v.Subscriptions, internalSubscription), Page: &trpc.PageInfo{NextCursor: v.NextCursor}, Quota: internalQuota(v.Quota), AsOf: utc(v.AsOf)}, nil
}
func (s *Server) PauseSubscription(ctx context.Context, r *trpc.PauseSubscriptionRequest) (*trpc.PauseSubscriptionResponse, error) {
	owner, e := principal(r.GetActor())
	if e != nil {
		return nil, e
	}
	v, e := s.service.ChangeSubscription(ctx, owner, "pause", tm.ChangeInput{SubscriptionID: r.GetSubscriptionId(), ExpectedRevision: r.GetExpectedRevision(), RequestID: r.GetRequestId()})
	if e != nil {
		return nil, e
	}
	return &trpc.PauseSubscriptionResponse{Subscription: internalSubscription(v)}, nil
}
func (s *Server) ResumeSubscription(ctx context.Context, r *trpc.ResumeSubscriptionRequest) (*trpc.ResumeSubscriptionResponse, error) {
	owner, e := principal(r.GetActor())
	if e != nil {
		return nil, e
	}
	v, e := s.service.ChangeSubscription(ctx, owner, "resume", tm.ChangeInput{SubscriptionID: r.GetSubscriptionId(), ExpectedRevision: r.GetExpectedRevision(), RequestID: r.GetRequestId()})
	if e != nil {
		return nil, e
	}
	return &trpc.ResumeSubscriptionResponse{Subscription: internalSubscription(v)}, nil
}
func (s *Server) CancelSubscription(ctx context.Context, r *trpc.CancelSubscriptionRequest) (*trpc.CancelSubscriptionResponse, error) {
	owner, e := principal(r.GetActor())
	if e != nil {
		return nil, e
	}
	v, e := s.service.ChangeSubscription(ctx, owner, "cancel", tm.ChangeInput{SubscriptionID: r.GetSubscriptionId(), ExpectedRevision: r.GetExpectedRevision(), RequestID: r.GetRequestId()})
	if e != nil {
		return nil, e
	}
	return &trpc.CancelSubscriptionResponse{Subscription: internalSubscription(v)}, nil
}
func (s *Server) UpdateTargetNote(ctx context.Context, r *trpc.UpdateTargetNoteRequest) (*trpc.UpdateTargetNoteResponse, error) {
	owner, e := principal(r.GetActor())
	if e != nil {
		return nil, e
	}
	wallet, e := parseWallet(r.GetWallet())
	if e != nil {
		return nil, e
	}
	v, e := s.service.UpdateTargetNote(ctx, owner, tm.NoteInput{Wallet: wallet, Note: r.GetNote(), ExpectedRevision: r.GetExpectedRevision(), RequestID: r.GetRequestId()})
	if e != nil {
		return nil, e
	}
	return &trpc.UpdateTargetNoteResponse{Note: internalNote(v)}, nil
}
func (s *Server) GetActivity(ctx context.Context, r *trpc.GetActivityRequest) (*trpc.GetActivityResponse, error) {
	owner, e := principal(r.GetActor())
	if e != nil {
		return nil, e
	}
	v, e := s.service.GetActivity(ctx, owner, r.GetActivityId())
	if e != nil {
		return nil, e
	}
	return &trpc.GetActivityResponse{Activity: internalActivity(v)}, nil
}
func (s *Server) ListActivities(ctx context.Context, r *trpc.ListActivitiesRequest) (*trpc.ListActivitiesResponse, error) {
	owner, e := principal(r.GetActor())
	if e != nil {
		return nil, e
	}
	filter := tm.ActivityFilter{SubscriptionID: r.GetSubscriptionId()}
	filter.From, e = parseTimeFilter(r.GetFrom())
	if e != nil {
		return nil, e
	}
	filter.To, e = parseTimeFilter(r.GetTo())
	if e != nil {
		return nil, e
	}
	if batch := r.GetSummaryBatchId(); batch != "" {
		filter.BatchID, e = strconv.ParseInt(batch, 10, 64)
		if e != nil || filter.BatchID <= 0 || number(filter.BatchID) != batch {
			return nil, status.Error(codes.InvalidArgument, "positive batch ID required")
		}
	}
	v, e := s.service.ListActivities(ctx, owner, r.GetPage().GetPageSize(), r.GetPage().GetCursor(), r.GetRefreshCursor(), filter)
	if e != nil {
		return nil, e
	}
	return &trpc.ListActivitiesResponse{Activities: mapItems(v.Activities, internalActivity), Page: &trpc.ActivityPageInfo{NextCursor: v.NextCursor, RefreshCursor: v.RefreshCursor, Snapshot: v.SnapshotToken, AsOf: utc(v.AsOf), HasNewer: v.HasNewer}}, nil
}
func (s *Server) ListSubscriptionHistory(ctx context.Context, r *trpc.ListSubscriptionHistoryRequest) (*trpc.ListSubscriptionHistoryResponse, error) {
	owner, e := principal(r.GetActor())
	if e != nil {
		return nil, e
	}
	v, e := s.service.ListHistory(ctx, owner, r.GetSubscriptionId(), r.GetPage().GetPageSize(), r.GetPage().GetCursor())
	if e != nil {
		return nil, e
	}
	return &trpc.ListSubscriptionHistoryResponse{Entries: mapItems(v.Entries, internalHistory), Page: &trpc.PageInfo{NextCursor: v.NextCursor}, AsOf: utc(v.AsOf)}, nil
}
func (s *Server) GetSummaryBatch(ctx context.Context, r *trpc.GetSummaryBatchRequest) (*trpc.GetSummaryBatchResponse, error) {
	owner, e := principal(r.GetActor())
	if e != nil {
		return nil, e
	}
	v, e := s.service.GetSummaryBatch(ctx, owner, r.GetBatchId())
	if e != nil {
		return nil, e
	}
	return &trpc.GetSummaryBatchResponse{Batch: internalBatch(v)}, nil
}
func (s *Server) ListSummaryParts(ctx context.Context, r *trpc.ListSummaryPartsRequest) (*trpc.ListSummaryPartsResponse, error) {
	owner, e := principal(r.GetActor())
	if e != nil {
		return nil, e
	}
	v, e := s.service.ListSummaryParts(ctx, owner, r.GetBatchId(), r.GetActivityId(), r.GetPage().GetPageSize(), r.GetPage().GetCursor())
	if e != nil {
		return nil, e
	}
	return &trpc.ListSummaryPartsResponse{Parts: mapItems(v.Parts, internalPart), Page: &trpc.PageInfo{NextCursor: v.NextCursor}, AsOf: utc(v.AsOf)}, nil
}
func (s *Server) GetSubscriptionSummary(ctx context.Context, r *trpc.GetSubscriptionSummaryRequest) (*trpc.GetSubscriptionSummaryResponse, error) {
	owner, e := principal(r.GetActor())
	if e != nil {
		return nil, e
	}
	v, e := s.service.GetSubscriptionSummary(ctx, owner, r.GetSubscriptionId())
	if e != nil {
		return nil, e
	}
	return &trpc.GetSubscriptionSummaryResponse{Summary: internalAdminSubscription(v)}, nil
}
func (s *Server) ListSubscriptionSummaries(ctx context.Context, r *trpc.ListSubscriptionSummariesRequest) (*trpc.ListSubscriptionSummariesResponse, error) {
	owner, e := principal(r.GetActor())
	if e != nil {
		return nil, e
	}
	filter := tm.AdminSubscriptionFilter{AccountID: r.GetAccountId(), State: r.GetState(), IncludeCancelled: r.GetIncludeCancelled()}
	if r.GetWallet() != "" {
		wallet, e := parseWallet(r.GetWallet())
		if e != nil {
			return nil, e
		}
		filter.Wallet = &wallet
	}
	v, e := s.service.ListSubscriptionSummaries(ctx, owner, r.GetPage().GetPageSize(), r.GetPage().GetCursor(), filter)
	if e != nil {
		return nil, e
	}
	return &trpc.ListSubscriptionSummariesResponse{Summaries: mapItems(v.Summaries, internalAdminSubscription), Page: &trpc.PageInfo{NextCursor: v.NextCursor}, AsOf: utc(v.AsOf)}, nil
}
func (s *Server) GetTraderSyncRuntimeStatus(ctx context.Context, r *trpc.GetTraderSyncRuntimeStatusRequest) (*trpc.GetTraderSyncRuntimeStatusResponse, error) {
	owner, e := principal(r.GetActor())
	if e != nil {
		return nil, e
	}
	v, e := s.service.GetRuntimeStatus(ctx, owner)
	if e != nil {
		return nil, e
	}
	return &trpc.GetTraderSyncRuntimeStatusResponse{Status: internalRuntime(v)}, nil
}
