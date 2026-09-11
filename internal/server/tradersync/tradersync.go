package tradersync

import (
	"context"
	"github.com/ethereum/go-ethereum/common"
	"github.com/useryege/athena/util/session"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"math/big"
	"strconv"
	"strings"
	"time"

	ts "github.com/useryege/athena/internal/tradersync"
	tm "github.com/useryege/athena/internal/tradersync/types"
	api "github.com/useryege/athena/pkg/apiclient/tradersync"
	app "github.com/useryege/athena/pkg/apis/application/v1alpha1"
)

type Server struct {
	api.UnimplementedTraderSyncServiceServer
	service *ts.Service
}

func New(service *ts.Service) *Server { return &Server{service: service} }

func utc(v time.Time) string {
	if v.IsZero() {
		return ""
	}
	return v.UTC().Format(time.RFC3339Nano)
}
func optionalTime(v *time.Time) *string {
	if v == nil {
		return nil
	}
	s := utc(*v)
	return &s
}
func number(v int64) string { return strconv.FormatInt(v, 10) }
func copyString(v *string) *string {
	if v == nil {
		return nil
	}
	s := *v
	return &s
}
func publicEvidence(v tm.Evidence) app.TraderSyncFieldEvidence {
	if v.Availability == "" {
		v.Availability = "unavailable"
		if v.ReasonCode == "" {
			v.ReasonCode = "not_observed"
		}
	}
	return app.TraderSyncFieldEvidence{Availability: v.Availability, ReasonCode: v.ReasonCode, Source: v.Source, QueriedAt: utc(v.QueriedAt)}
}
func publicString(v tm.Scalar) app.TraderSyncStringField {
	r := app.TraderSyncStringField{Evidence: publicEvidence(v.Evidence)}
	if v.Availability == "available" {
		r.Value = copyString(v.Value)
	}
	return r
}
func publicDecimal(v tm.Scalar) app.TraderSyncDecimalField {
	r := publicString(v)
	return app.TraderSyncDecimalField{Evidence: r.Evidence, Value: r.Value}
}
func publicBool(v tm.Scalar) app.TraderSyncBoolField {
	r := app.TraderSyncBoolField{Evidence: publicEvidence(v.Evidence)}
	if v.Availability == "available" && v.Value != nil {
		b, e := strconv.ParseBool(*v.Value)
		if e == nil {
			r.Value = &b
		} else {
			r.Evidence.Availability = "unavailable"
			r.Evidence.ReasonCode = "invalid_boolean"
		}
	}
	return r
}
func publicTime(v tm.Scalar) app.TraderSyncTimeField {
	r := app.TraderSyncTimeField{Evidence: publicEvidence(v.Evidence)}
	if v.Availability == "available" && v.Value != nil {
		t, e := time.Parse(time.RFC3339Nano, *v.Value)
		if e == nil {
			s := utc(t)
			r.Value = &s
		} else {
			r.Evidence.Availability = "unavailable"
			r.Evidence.ReasonCode = "invalid_time"
		}
	}
	return r
}
func publicDisplay(v tm.TargetDisplay) app.TraderSyncTargetDisplay {
	return app.TraderSyncTargetDisplay{DisplayName: publicString(v.DisplayName), Avatar: publicString(v.Avatar), ProfileURL: publicString(v.ProfileURL)}
}
func publicQuota(v tm.Quota) *app.TraderSyncQuota {
	return &app.TraderSyncQuota{Used: v.Used, Limit: v.Limit}
}
func publicNote(v tm.TargetNote) *app.TraderSyncTargetNote {
	return &app.TraderSyncTargetNote{Wallet: v.Wallet.Hex(), Note: v.Note, Revision: v.Revision}
}
func publicResolved(v tm.ResolvedTarget) *app.TraderSyncResolvedTarget {
	c := v.Card
	r := &app.TraderSyncResolvedTarget{Wallet: c.Identity.Wallet.Hex(), CanonicalProfileURL: c.Identity.ProfileURL, Avatar: publicString(c.Avatar), DisplayName: publicString(c.DisplayName), Verified: publicBool(c.Verified), JoinedAt: publicTime(c.JoinedAt), PositionValue: publicDecimal(c.PositionValue), LargestWin: publicDecimal(c.LargestWin), Predictions: publicDecimal(c.Predictions), DefaultPeriod: "1Y", ConfirmationToken: v.Token, ExpiresAt: utc(v.ExpiresAt), UsageNotice: "优先选择低频交易者；高频监控不纳入性能保障", Quota: *publicQuota(v.Context.Quota)}
	if v.Context.SavedNote != nil {
		r.SavedNote = publicNote(*v.Context.SavedNote)
	}
	if e := v.Context.Existing; e != nil {
		r.ExistingSubscription = &app.TraderSyncExistingSubscription{ID: e.ID, Status: e.Status, Revision: e.Revision}
	}
	for _, period := range []string{"1D", "1W", "1M", "1Y", "YTD", "ALL"} {
		p := c.PnL[period]
		curve := app.TraderSyncCurve{Evidence: publicEvidence(p.Curve.Evidence)}
		if p.Curve.Points != nil {
			curve.Points = make([]app.TraderSyncCurvePoint, 0, len(p.Curve.Points))
			for _, point := range p.Curve.Points {
				curve.Points = append(curve.Points, app.TraderSyncCurvePoint{T: number(point.T), P: point.P})
			}
		}
		reference := app.TraderSyncTimeField{Evidence: app.TraderSyncFieldEvidence{Availability: "unavailable", ReasonCode: "official_reference_unavailable", Source: p.Amount.Source, QueriedAt: utc(p.Amount.QueriedAt)}}
		if p.Reference != nil {
			reference.Value = optionalTime(p.Reference)
			reference.Evidence.Availability = "available"
			reference.Evidence.ReasonCode = ""
		}
		zone := app.TraderSyncStringField{Evidence: app.TraderSyncFieldEvidence{Availability: "unavailable", ReasonCode: "official_timezone_unavailable", Source: p.Amount.Source, QueriedAt: utc(p.Amount.QueriedAt)}}
		if p.Timezone != "" {
			zone.Value = copyString(&p.Timezone)
			zone.Evidence.Availability = "available"
			zone.Evidence.ReasonCode = ""
		}
		r.PnL = append(r.PnL, app.TraderSyncPnLView{Period: period, Amount: publicDecimal(p.Amount), Curve: curve, Interval: p.Interval, Fidelity: p.Fidelity, ReferenceTime: reference, Timezone: zone})
	}
	return r
}
func publicCounts(v tm.StatusCounts) app.TraderSyncStatusCounts {
	return app.TraderSyncStatusCounts{Total: number(v.Total), Pending: number(v.Pending), Sending: number(v.Sending), Sent: number(v.Sent), Failed: number(v.Failed), Unknown: number(v.Unknown), Cancelled: number(v.Cancelled)}
}
func publicInterruption(v *tm.InterruptionDetails) *app.TraderSyncInterruption {
	if v == nil {
		return nil
	}
	return &app.TraderSyncInterruption{Start: optionalTime(v.Start), End: optionalTime(v.End), RecoveredAt: optionalTime(v.RecoveredAt), Reason: v.Reason, Uncertainty: v.Uncertainty, PossibleMissing: v.PossibleMissing}
}
func publicObservation(v tm.ObservationDetails) app.TraderSyncObservation {
	return app.TraderSyncObservation{State: v.State, Reason: v.Reason, LastReliableAt: optionalTime(v.LastReliableAt), LatestInterruption: publicInterruption(v.LatestInterruption), InterruptionCount: number(v.InterruptionCount)}
}
func publicInterval(v *tm.IntervalDetails) *app.TraderSyncInterval {
	if v == nil {
		return nil
	}
	return &app.TraderSyncInterval{EffectiveAt: utc(v.EffectiveAt), EndedAt: optionalTime(v.EndedAt), Generation: v.Generation, Epoch: v.Epoch}
}
func publicSubscription(v tm.SubscriptionDetails) *app.TraderSyncSubscription {
	state := v.DesiredState
	if state == "enabled" {
		state = v.ObservationState
	}
	return &app.TraderSyncSubscription{ID: v.ID, Wallet: v.Wallet.Hex(), Status: state, Revision: v.Revision, Generation: v.Generation, Note: v.Note, NoteRevision: v.NoteRevision, CreatedAt: utc(v.CreatedAt), UpdatedAt: utc(v.UpdatedAt), PausedAt: optionalTime(v.PausedAt), CancelledAt: optionalTime(v.CancelledAt), PermissionDisabledAt: optionalTime(v.PermissionDisabledAt), CurrentInterval: publicInterval(v.CurrentInterval), Observation: publicObservation(v.Observation), BindingStatus: v.BindingStatus, QueueNotice: v.QueueNotice, QueueCounts: publicCounts(v.QueueCounts), TargetDisplay: publicDisplay(v.TargetDisplay)}
}
func publicDelivery(v *tm.DeliveryDetails) *app.TraderSyncDelivery {
	if v == nil {
		return nil
	}
	r := &app.TraderSyncDelivery{ID: number(v.ID), Status: v.Status, Reason: v.Reason, AuthorizedAt: optionalTime(v.AuthorizedAt), StartedAt: optionalTime(v.StartedAt), ResultAt: optionalTime(v.ResultAt), MessageID: copyString(v.MessageID), AttemptCount: number(v.AttemptCount)}
	if a := v.LatestAttempt; a != nil {
		r.LatestAttempt = &app.TraderSyncAttempt{Index: number(a.Index), AuthorizedAt: utc(a.AuthorizedAt), StartedAt: optionalTime(a.StartedAt), ResultAt: optionalTime(a.ResultAt), Status: a.Status, Reason: a.Reason}
	}
	return r
}
func publicMarket(v tm.MarketRef) app.TraderSyncMarketRef {
	return app.TraderSyncMarketRef{Evidence: publicEvidence(v.Evidence), ID: v.ID, Title: v.Title, URL: v.URL, ConditionID: v.ConditionID, PositionID: v.PositionID, Outcome: v.Outcome}
}
func publicMetadata(v tm.TradeMetadata) app.TraderSyncTradeMetadata {
	r := app.TraderSyncTradeMetadata{Market: publicMarket(v.Market), LegsEvidence: publicEvidence(v.LegsEvidence), Relationship: v.Relationship}
	if v.Legs != nil {
		r.Legs = make([]app.TraderSyncComboLeg, 0, len(v.Legs))
		for _, leg := range v.Legs {
			r.Legs = append(r.Legs, app.TraderSyncComboLeg{PositionID: leg.PositionID, Market: publicMarket(leg.Market)})
		}
	}
	return r
}
func publicSummaryProgress(v *tm.SummaryDetails) *app.TraderSyncSummaryProgress {
	if v == nil {
		return nil
	}
	r := &app.TraderSyncSummaryProgress{Phase: v.Phase, Reason: v.Reason, RelatedPartCounts: publicCounts(v.RelatedPartCounts), BatchPartCounts: publicCounts(v.BatchPartCounts), OldestAt: utc(v.OldestAt), FirstStartedAt: optionalTime(v.FirstStartedAt)}
	if v.BatchID != nil {
		n := number(*v.BatchID)
		r.BatchID = &n
	}
	return r
}
func publicActivity(v tm.ActivityDetails) *app.TraderSyncActivity {
	t := v.Trade
	loc := v.SourceLocation
	price := app.TraderSyncFieldEvidence{Availability: "unavailable", ReasonCode: "price_unavailable", Source: "source_record"}
	n, nOK := new(big.Int).SetString(t.PriceNumerator, 10)
	d, dOK := new(big.Int).SetString(t.PriceDenominator, 10)
	if nOK && dOK && n.Sign() >= 0 && d.Sign() > 0 {
		price.Availability = "available"
		price.ReasonCode = ""
	}
	r := &app.TraderSyncActivity{ID: number(v.ID), SubscriptionID: v.SubscriptionID, SourceRecordID: number(v.SourceID), Wallet: t.Wallet.Hex(), Side: t.Side, PositionID: t.PositionID, CollateralRaw: t.CollateralRaw, SharesRaw: t.SharesRaw, FeeRaw: t.FeeRaw, CollateralSymbol: t.CollateralSymbol, CollateralDecimals: int32(t.CollateralDecimals), SharesDecimals: int32(t.SharesDecimals), PriceNumerator: t.PriceNumerator, PriceDenominator: t.PriceDenominator, PriceEvidence: price, SourceVersion: t.SourceVersion, SettledAt: utc(v.SettledAt), ReceivedAt: utc(v.ReceivedAt), RecordedAt: utc(v.RecordedAt), PublicTimeEvidence: app.TraderSyncFieldEvidence{Availability: "unavailable", ReasonCode: "public_time_unobservable", Source: "source_record"}, Metadata: publicMetadata(v.Metadata), NoteSnapshot: v.NoteSnapshot, NotificationMode: v.NotificationMode, NotificationReason: v.NotificationReason, Delivery: publicDelivery(v.Delivery), SummaryProgress: publicSummaryProgress(v.SummaryProgress), TargetDisplaySnapshot: publicDisplay(v.TargetDisplaySnapshot), SourceLocation: app.TraderSyncSourceLocation{ChainID: number(loc.ChainID), ExchangeAddress: loc.Exchange.Hex(), TransactionHash: loc.TransactionHash.Hex(), BlockHash: loc.BlockHash.Hex(), BlockNumber: number(loc.BlockNumber), LogIndex: number(loc.LogIndex)}}
	if a := v.FinalityAnomaly; a != nil {
		r.FinalityAnomaly = &app.TraderSyncFinalityAnomaly{Reason: a.Reason, DetectedAt: utc(a.DetectedAt), PublishedBlockHash: a.PublishedBlockHash.Hex()}
		if a.ConflictingBlockHash != nil {
			h := a.ConflictingBlockHash.Hex()
			r.FinalityAnomaly.ConflictingBlockHash = &h
		}
	}
	return r
}
func publicHistory(v tm.HistoryEntry) *app.TraderSyncHistoryEntry {
	return &app.TraderSyncHistoryEntry{ID: v.ID, Kind: v.Kind, SortAt: utc(v.SortAt), Interval: publicInterval(v.Interval), Interruption: publicInterruption(v.Interruption)}
}
func publicBatch(v tm.SummaryBatchDetails) *app.TraderSyncSummaryBatch {
	r := &app.TraderSyncSummaryBatch{ID: number(v.ID), OldestAt: utc(v.OldestAt), SettledFrom: utc(v.SettledFrom), SettledTo: utc(v.SettledTo), RecordedFrom: utc(v.RecordedFrom), RecordedTo: utc(v.RecordedTo), FirstStartedAt: optionalTime(v.FirstStartedAt), ActivityCount: number(v.ActivityCount), PartCounts: publicCounts(v.PartCounts), AsOf: utc(v.AsOf)}
	for _, t := range v.TargetCounts {
		r.TargetCounts = append(r.TargetCounts, app.TraderSyncTargetCount{Wallet: t.Wallet.Hex(), Count: number(t.Count)})
	}
	return r
}
func publicPart(v tm.SummaryPartDetails) *app.TraderSyncSummaryPart {
	return &app.TraderSyncSummaryPart{ID: number(v.ID), Index: v.Index, Total: v.Total, Delivery: *publicDelivery(&v.Delivery), AssociatedActivityCount: number(v.AssociatedActivityCount)}
}
func publicAdminSubscription(v tm.SubscriptionSummary) *app.TraderSyncSubscriptionSummary {
	return &app.TraderSyncSubscriptionSummary{SubscriptionID: v.SubscriptionID, AccountID: v.AccountID, Username: v.Username, Email: v.Email, Wallet: v.Wallet.Hex(), Status: v.Status, CreatedAt: utc(v.CreatedAt), UpdatedAt: utc(v.UpdatedAt), PausedAt: optionalTime(v.PausedAt), CancelledAt: optionalTime(v.CancelledAt), PermissionDisabledAt: optionalTime(v.PermissionDisabledAt), Observation: publicObservation(v.Observation), ActivityCount: number(v.ActivityCount), AssociatedDeliveryCounts: publicCounts(v.AssociatedDeliveryCounts), AsOf: utc(v.AsOf)}
}
func publicRuntime(v tm.RuntimeStatus) *app.TraderSyncRuntimeStatus {
	r := &app.TraderSyncRuntimeStatus{CollectorConnected: v.CollectorConnected, CollectorEpoch: v.CollectorEpoch, FilterRevision: v.FilterRevision, AsOf: utc(v.AsOf)}
	for _, m := range v.Metrics {
		r.Metrics = append(r.Metrics, app.TraderSyncRuntimeMetric{Name: m.Name, Value: m.Value, Unit: m.Unit, Kind: m.Kind, WindowStart: optionalTime(m.WindowStart), WindowEnd: optionalTime(m.WindowEnd), ServiceEpoch: copyString(m.ServiceEpoch)})
	}
	return r
}
func mapItems[A, B any](values []A, fn func(A) *B) []*B {
	out := make([]*B, 0, len(values))
	for _, v := range values {
		out = append(out, fn(v))
	}
	return out
}
func principal(ctx context.Context) (string, error) {
	id := session.AccountID(ctx)
	if id == "" {
		return "", status.Error(codes.Unauthenticated, "account identity required")
	}
	return id, nil
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

func (s *Server) ResolveTarget(ctx context.Context, r *api.ResolveTargetRequest) (*api.ResolveTargetResponse, error) {
	owner, e := principal(ctx)
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
	return &api.ResolveTargetResponse{Target: publicResolved(v)}, nil
}
func (s *Server) CreateSubscription(ctx context.Context, r *api.CreateSubscriptionRequest) (*api.CreateSubscriptionResponse, error) {
	owner, e := principal(ctx)
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
	return &api.CreateSubscriptionResponse{Subscription: publicSubscription(v)}, nil
}
func (s *Server) GetSubscription(ctx context.Context, r *api.GetSubscriptionRequest) (*api.GetSubscriptionResponse, error) {
	owner, e := principal(ctx)
	if e != nil {
		return nil, e
	}
	v, e := s.service.GetSubscription(ctx, owner, r.GetSubscriptionId())
	if e != nil {
		return nil, e
	}
	return &api.GetSubscriptionResponse{Subscription: publicSubscription(v)}, nil
}
func (s *Server) ListSubscriptions(ctx context.Context, r *api.ListSubscriptionsRequest) (*api.ListSubscriptionsResponse, error) {
	owner, e := principal(ctx)
	if e != nil {
		return nil, e
	}
	v, e := s.service.ListSubscriptions(ctx, owner, r.GetPage().GetPageSize(), r.GetPage().GetCursor(), tm.SubscriptionFilter{View: r.GetView(), State: r.GetState()})
	if e != nil {
		return nil, e
	}
	return &api.ListSubscriptionsResponse{Subscriptions: mapItems(v.Subscriptions, publicSubscription), Page: &api.PageInfo{NextCursor: v.NextCursor}, Quota: publicQuota(v.Quota), AsOf: utc(v.AsOf)}, nil
}
func (s *Server) PauseSubscription(ctx context.Context, r *api.PauseSubscriptionRequest) (*api.PauseSubscriptionResponse, error) {
	owner, e := principal(ctx)
	if e != nil {
		return nil, e
	}
	v, e := s.service.ChangeSubscription(ctx, owner, "pause", tm.ChangeInput{SubscriptionID: r.GetSubscriptionId(), ExpectedRevision: r.GetExpectedRevision(), RequestID: r.GetRequestId()})
	if e != nil {
		return nil, e
	}
	return &api.PauseSubscriptionResponse{Subscription: publicSubscription(v)}, nil
}
func (s *Server) ResumeSubscription(ctx context.Context, r *api.ResumeSubscriptionRequest) (*api.ResumeSubscriptionResponse, error) {
	owner, e := principal(ctx)
	if e != nil {
		return nil, e
	}
	v, e := s.service.ChangeSubscription(ctx, owner, "resume", tm.ChangeInput{SubscriptionID: r.GetSubscriptionId(), ExpectedRevision: r.GetExpectedRevision(), RequestID: r.GetRequestId()})
	if e != nil {
		return nil, e
	}
	return &api.ResumeSubscriptionResponse{Subscription: publicSubscription(v)}, nil
}
func (s *Server) CancelSubscription(ctx context.Context, r *api.CancelSubscriptionRequest) (*api.CancelSubscriptionResponse, error) {
	owner, e := principal(ctx)
	if e != nil {
		return nil, e
	}
	v, e := s.service.ChangeSubscription(ctx, owner, "cancel", tm.ChangeInput{SubscriptionID: r.GetSubscriptionId(), ExpectedRevision: r.GetExpectedRevision(), RequestID: r.GetRequestId()})
	if e != nil {
		return nil, e
	}
	return &api.CancelSubscriptionResponse{Subscription: publicSubscription(v)}, nil
}
func (s *Server) UpdateTargetNote(ctx context.Context, r *api.UpdateTargetNoteRequest) (*api.UpdateTargetNoteResponse, error) {
	owner, e := principal(ctx)
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
	return &api.UpdateTargetNoteResponse{Note: publicNote(v)}, nil
}
func (s *Server) GetActivity(ctx context.Context, r *api.GetActivityRequest) (*api.GetActivityResponse, error) {
	owner, e := principal(ctx)
	if e != nil {
		return nil, e
	}
	v, e := s.service.GetActivity(ctx, owner, r.GetActivityId())
	if e != nil {
		return nil, e
	}
	return &api.GetActivityResponse{Activity: publicActivity(v)}, nil
}
func (s *Server) ListActivities(ctx context.Context, r *api.ListActivitiesRequest) (*api.ListActivitiesResponse, error) {
	owner, e := principal(ctx)
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
	return &api.ListActivitiesResponse{Activities: mapItems(v.Activities, publicActivity), Page: &api.ActivityPageInfo{NextCursor: v.NextCursor, RefreshCursor: v.RefreshCursor, Snapshot: v.SnapshotToken, AsOf: utc(v.AsOf), HasNewer: v.HasNewer}}, nil
}
func (s *Server) ListSubscriptionHistory(ctx context.Context, r *api.ListSubscriptionHistoryRequest) (*api.ListSubscriptionHistoryResponse, error) {
	owner, e := principal(ctx)
	if e != nil {
		return nil, e
	}
	v, e := s.service.ListHistory(ctx, owner, r.GetSubscriptionId(), r.GetPage().GetPageSize(), r.GetPage().GetCursor())
	if e != nil {
		return nil, e
	}
	return &api.ListSubscriptionHistoryResponse{Entries: mapItems(v.Entries, publicHistory), Page: &api.PageInfo{NextCursor: v.NextCursor}, AsOf: utc(v.AsOf)}, nil
}
func (s *Server) GetSummaryBatch(ctx context.Context, r *api.GetSummaryBatchRequest) (*api.GetSummaryBatchResponse, error) {
	owner, e := principal(ctx)
	if e != nil {
		return nil, e
	}
	v, e := s.service.GetSummaryBatch(ctx, owner, r.GetBatchId())
	if e != nil {
		return nil, e
	}
	return &api.GetSummaryBatchResponse{Batch: publicBatch(v)}, nil
}
func (s *Server) ListSummaryParts(ctx context.Context, r *api.ListSummaryPartsRequest) (*api.ListSummaryPartsResponse, error) {
	owner, e := principal(ctx)
	if e != nil {
		return nil, e
	}
	v, e := s.service.ListSummaryParts(ctx, owner, r.GetBatchId(), r.GetActivityId(), r.GetPage().GetPageSize(), r.GetPage().GetCursor())
	if e != nil {
		return nil, e
	}
	return &api.ListSummaryPartsResponse{Parts: mapItems(v.Parts, publicPart), Page: &api.PageInfo{NextCursor: v.NextCursor}, AsOf: utc(v.AsOf)}, nil
}
func (s *Server) GetSubscriptionSummary(ctx context.Context, r *api.GetSubscriptionSummaryRequest) (*api.GetSubscriptionSummaryResponse, error) {
	owner, e := principal(ctx)
	if e != nil {
		return nil, e
	}
	v, e := s.service.GetSubscriptionSummary(ctx, owner, r.GetSubscriptionId())
	if e != nil {
		return nil, e
	}
	return &api.GetSubscriptionSummaryResponse{Summary: publicAdminSubscription(v)}, nil
}
func (s *Server) ListSubscriptionSummaries(ctx context.Context, r *api.ListSubscriptionSummariesRequest) (*api.ListSubscriptionSummariesResponse, error) {
	owner, e := principal(ctx)
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
	return &api.ListSubscriptionSummariesResponse{Summaries: mapItems(v.Summaries, publicAdminSubscription), Page: &api.PageInfo{NextCursor: v.NextCursor}, AsOf: utc(v.AsOf)}, nil
}
func (s *Server) GetTraderSyncRuntimeStatus(ctx context.Context, _ *api.GetTraderSyncRuntimeStatusRequest) (*api.GetTraderSyncRuntimeStatusResponse, error) {
	owner, e := principal(ctx)
	if e != nil {
		return nil, e
	}
	v, e := s.service.GetRuntimeStatus(ctx, owner)
	if e != nil {
		return nil, e
	}
	return &api.GetTraderSyncRuntimeStatusResponse{Status: publicRuntime(v)}, nil
}
