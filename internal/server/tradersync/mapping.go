package tradersync

import (
	log "github.com/sirupsen/logrus"
	trpc "github.com/useryege/athena/internal/tradersync/apiclient"
	api "github.com/useryege/athena/pkg/apiclient/tradersync"
	app "github.com/useryege/athena/pkg/apis/application/v1alpha1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// responseMapper copies the wire contract without deriving domain facts. A missing
// required object poisons the entire response, including malformed list entries.
type responseMapper struct{ err error }

func (m *responseMapper) missing() {
	if m.err == nil {
		log.WithField("reason", "response_contract_invalid").Warn("Trader Sync facade unavailable")
	}
	m.err = status.Error(codes.Unavailable, "Trader Sync response contract invalid")
}
func stringValue(v *trpc.StringValue) *string {
	if v == nil {
		return nil
	}
	s := v.Value
	return &s
}
func boolValue(v *trpc.BoolValue) *bool {
	if v == nil {
		return nil
	}
	b := v.Value
	return &b
}
func (m *responseMapper) mapFieldEvidence(v *trpc.FieldEvidence) *app.TraderSyncFieldEvidence {
	r := &app.TraderSyncFieldEvidence{}
	if v == nil {
		m.missing()
		return r
	}
	r.Availability = v.Availability
	r.ReasonCode = v.ReasonCode
	r.Source = v.Source
	r.QueriedAt = v.QueriedAt
	return r
}
func (m *responseMapper) mapStringField(v *trpc.StringField) *app.TraderSyncStringField {
	r := &app.TraderSyncStringField{}
	if v == nil {
		m.missing()
		return r
	}
	r.Evidence = *m.mapFieldEvidence(v.Evidence)
	r.Value = stringValue(v.Value)
	return r
}
func (m *responseMapper) mapDecimalField(v *trpc.DecimalField) *app.TraderSyncDecimalField {
	r := &app.TraderSyncDecimalField{}
	if v == nil {
		m.missing()
		return r
	}
	r.Evidence = *m.mapFieldEvidence(v.Evidence)
	r.Value = stringValue(v.Value)
	return r
}
func (m *responseMapper) mapBoolField(v *trpc.BoolField) *app.TraderSyncBoolField {
	r := &app.TraderSyncBoolField{}
	if v == nil {
		m.missing()
		return r
	}
	r.Evidence = *m.mapFieldEvidence(v.Evidence)
	r.Value = boolValue(v.Value)
	return r
}
func (m *responseMapper) mapTimeField(v *trpc.TimeField) *app.TraderSyncTimeField {
	r := &app.TraderSyncTimeField{}
	if v == nil {
		m.missing()
		return r
	}
	r.Evidence = *m.mapFieldEvidence(v.Evidence)
	r.Value = stringValue(v.Value)
	return r
}
func (m *responseMapper) mapCurvePoint(v *trpc.CurvePoint) *app.TraderSyncCurvePoint {
	r := &app.TraderSyncCurvePoint{}
	if v == nil {
		m.missing()
		return r
	}
	r.T = v.T
	r.P = v.P
	return r
}
func (m *responseMapper) mapCurve(v *trpc.Curve) *app.TraderSyncCurve {
	r := &app.TraderSyncCurve{}
	if v == nil {
		m.missing()
		return r
	}
	r.Evidence = *m.mapFieldEvidence(v.Evidence)
	if v.Points != nil {
		r.Points = make([]app.TraderSyncCurvePoint, 0, len(v.Points.Items))
		for _, item := range v.Points.Items {
			r.Points = append(r.Points, *m.mapCurvePoint(item))
		}
	}
	return r
}
func (m *responseMapper) mapPnLView(v *trpc.PnLView) *app.TraderSyncPnLView {
	r := &app.TraderSyncPnLView{}
	if v == nil {
		m.missing()
		return r
	}
	r.Period = v.Period
	r.Amount = *m.mapDecimalField(v.Amount)
	r.Curve = *m.mapCurve(v.Curve)
	r.Interval = v.Interval
	r.Fidelity = v.Fidelity
	r.ReferenceTime = *m.mapTimeField(v.ReferenceTime)
	r.Timezone = *m.mapStringField(v.Timezone)
	return r
}
func (m *responseMapper) mapResolvedTarget(v *trpc.ResolvedTarget) *app.TraderSyncResolvedTarget {
	r := &app.TraderSyncResolvedTarget{}
	if v == nil {
		m.missing()
		return r
	}
	r.Wallet = v.Wallet
	r.CanonicalProfileURL = v.CanonicalProfileUrl
	r.Avatar = *m.mapStringField(v.Avatar)
	r.DisplayName = *m.mapStringField(v.DisplayName)
	r.Verified = *m.mapBoolField(v.Verified)
	r.JoinedAt = *m.mapTimeField(v.JoinedAt)
	r.PositionValue = *m.mapDecimalField(v.PositionValue)
	r.LargestWin = *m.mapDecimalField(v.LargestWin)
	r.Predictions = *m.mapDecimalField(v.Predictions)
	if v.PnL != nil {
		r.PnL = make([]app.TraderSyncPnLView, 0, len(v.PnL))
		for _, item := range v.PnL {
			r.PnL = append(r.PnL, *m.mapPnLView(item))
		}
	}
	r.DefaultPeriod = v.DefaultPeriod
	r.ConfirmationToken = v.ConfirmationToken
	r.ExpiresAt = v.ExpiresAt
	r.UsageNotice = v.UsageNotice
	if v.SavedNote != nil {
		r.SavedNote = m.mapTargetNote(v.SavedNote)
	}
	if v.ExistingSubscription != nil {
		r.ExistingSubscription = m.mapExistingSubscription(v.ExistingSubscription)
	}
	r.Quota = *m.mapQuota(v.Quota)
	return r
}
func (m *responseMapper) mapTargetNote(v *trpc.TargetNote) *app.TraderSyncTargetNote {
	r := &app.TraderSyncTargetNote{}
	if v == nil {
		m.missing()
		return r
	}
	r.Wallet = v.Wallet
	r.Note = v.Note
	r.Revision = v.Revision
	return r
}
func (m *responseMapper) mapQuota(v *trpc.Quota) *app.TraderSyncQuota {
	r := &app.TraderSyncQuota{}
	if v == nil {
		m.missing()
		return r
	}
	r.Used = v.Used
	r.Limit = v.Limit
	return r
}
func (m *responseMapper) mapExistingSubscription(v *trpc.ExistingSubscription) *app.TraderSyncExistingSubscription {
	r := &app.TraderSyncExistingSubscription{}
	if v == nil {
		m.missing()
		return r
	}
	r.ID = v.Id
	r.Status = v.Status
	r.Revision = v.Revision
	return r
}
func (m *responseMapper) mapTargetDisplay(v *trpc.TargetDisplay) *app.TraderSyncTargetDisplay {
	r := &app.TraderSyncTargetDisplay{}
	if v == nil {
		m.missing()
		return r
	}
	r.DisplayName = *m.mapStringField(v.DisplayName)
	r.Avatar = *m.mapStringField(v.Avatar)
	r.ProfileURL = *m.mapStringField(v.ProfileUrl)
	return r
}
func (m *responseMapper) mapSubscription(v *trpc.Subscription) *app.TraderSyncSubscription {
	r := &app.TraderSyncSubscription{}
	if v == nil {
		m.missing()
		return r
	}
	r.ID = v.Id
	r.Wallet = v.Wallet
	r.Status = v.Status
	r.Revision = v.Revision
	r.Generation = v.Generation
	r.Note = v.Note
	r.NoteRevision = v.NoteRevision
	r.CreatedAt = v.CreatedAt
	r.UpdatedAt = v.UpdatedAt
	r.PausedAt = stringValue(v.PausedAt)
	r.CancelledAt = stringValue(v.CancelledAt)
	r.PermissionDisabledAt = stringValue(v.PermissionDisabledAt)
	if v.CurrentInterval != nil {
		r.CurrentInterval = m.mapInterval(v.CurrentInterval)
	}
	r.Observation = *m.mapObservation(v.Observation)
	r.BindingStatus = v.BindingStatus
	r.QueueNotice = v.QueueNotice
	r.QueueCounts = *m.mapStatusCounts(v.QueueCounts)
	r.TargetDisplay = *m.mapTargetDisplay(v.TargetDisplay)
	return r
}
func (m *responseMapper) mapInterval(v *trpc.Interval) *app.TraderSyncInterval {
	r := &app.TraderSyncInterval{}
	if v == nil {
		m.missing()
		return r
	}
	r.EffectiveAt = v.EffectiveAt
	r.EndedAt = stringValue(v.EndedAt)
	r.Generation = v.Generation
	r.Epoch = v.Epoch
	return r
}
func (m *responseMapper) mapObservation(v *trpc.Observation) *app.TraderSyncObservation {
	r := &app.TraderSyncObservation{}
	if v == nil {
		m.missing()
		return r
	}
	r.State = v.State
	r.Reason = v.Reason
	r.LastReliableAt = stringValue(v.LastReliableAt)
	if v.LatestInterruption != nil {
		r.LatestInterruption = m.mapInterruption(v.LatestInterruption)
	}
	r.InterruptionCount = v.InterruptionCount
	return r
}
func (m *responseMapper) mapInterruption(v *trpc.Interruption) *app.TraderSyncInterruption {
	r := &app.TraderSyncInterruption{}
	if v == nil {
		m.missing()
		return r
	}
	r.Start = stringValue(v.Start)
	r.End = stringValue(v.End)
	r.RecoveredAt = stringValue(v.RecoveredAt)
	r.Reason = v.Reason
	r.Uncertainty = v.Uncertainty
	r.PossibleMissing = v.PossibleMissing
	return r
}
func (m *responseMapper) mapHistoryEntry(v *trpc.HistoryEntry) *app.TraderSyncHistoryEntry {
	r := &app.TraderSyncHistoryEntry{}
	if v == nil {
		m.missing()
		return r
	}
	r.ID = v.Id
	r.Kind = v.Kind
	r.SortAt = v.SortAt
	if v.Interval != nil {
		r.Interval = m.mapInterval(v.Interval)
	}
	if v.Interruption != nil {
		r.Interruption = m.mapInterruption(v.Interruption)
	}
	return r
}
func (m *responseMapper) mapActivity(v *trpc.Activity) *app.TraderSyncActivity {
	r := &app.TraderSyncActivity{}
	if v == nil {
		m.missing()
		return r
	}
	r.ID = v.Id
	r.SubscriptionID = v.SubscriptionId
	r.SourceRecordID = v.SourceRecordId
	r.Wallet = v.Wallet
	r.Side = v.Side
	r.PositionID = v.PositionId
	r.CollateralRaw = v.CollateralRaw
	r.SharesRaw = v.SharesRaw
	r.FeeRaw = v.FeeRaw
	r.CollateralSymbol = v.CollateralSymbol
	r.CollateralDecimals = v.CollateralDecimals
	r.SharesDecimals = v.SharesDecimals
	r.PriceNumerator = v.PriceNumerator
	r.PriceDenominator = v.PriceDenominator
	r.PriceEvidence = *m.mapFieldEvidence(v.PriceEvidence)
	r.SourceVersion = v.SourceVersion
	r.SettledAt = v.SettledAt
	r.ReceivedAt = v.ReceivedAt
	r.RecordedAt = v.RecordedAt
	r.PublicTimeEvidence = *m.mapFieldEvidence(v.PublicTimeEvidence)
	r.Metadata = *m.mapTradeMetadata(v.Metadata)
	r.NoteSnapshot = v.NoteSnapshot
	r.NotificationMode = v.NotificationMode
	r.NotificationReason = v.NotificationReason
	if v.Delivery != nil {
		r.Delivery = m.mapDelivery(v.Delivery)
	}
	if v.SummaryProgress != nil {
		r.SummaryProgress = m.mapSummaryProgress(v.SummaryProgress)
	}
	r.TargetDisplaySnapshot = *m.mapTargetDisplay(v.TargetDisplaySnapshot)
	if v.FinalityAnomaly != nil {
		r.FinalityAnomaly = m.mapFinalityAnomaly(v.FinalityAnomaly)
	}
	r.SourceLocation = *m.mapSourceLocation(v.SourceLocation)
	return r
}
func (m *responseMapper) mapSourceLocation(v *trpc.SourceLocation) *app.TraderSyncSourceLocation {
	r := &app.TraderSyncSourceLocation{}
	if v == nil {
		m.missing()
		return r
	}
	r.ChainID = v.ChainId
	r.ExchangeAddress = v.ExchangeAddress
	r.TransactionHash = v.TransactionHash
	r.BlockHash = v.BlockHash
	r.BlockNumber = v.BlockNumber
	r.LogIndex = v.LogIndex
	return r
}
func (m *responseMapper) mapFinalityAnomaly(v *trpc.FinalityAnomaly) *app.TraderSyncFinalityAnomaly {
	r := &app.TraderSyncFinalityAnomaly{}
	if v == nil {
		m.missing()
		return r
	}
	r.Reason = v.Reason
	r.DetectedAt = v.DetectedAt
	r.PublishedBlockHash = v.PublishedBlockHash
	r.ConflictingBlockHash = stringValue(v.ConflictingBlockHash)
	return r
}
func (m *responseMapper) mapMarketRef(v *trpc.MarketRef) *app.TraderSyncMarketRef {
	r := &app.TraderSyncMarketRef{}
	if v == nil {
		m.missing()
		return r
	}
	r.Evidence = *m.mapFieldEvidence(v.Evidence)
	r.ID = v.Id
	r.Title = v.Title
	r.URL = v.Url
	r.ConditionID = v.ConditionId
	r.PositionID = v.PositionId
	r.Outcome = v.Outcome
	return r
}
func (m *responseMapper) mapComboLeg(v *trpc.ComboLeg) *app.TraderSyncComboLeg {
	r := &app.TraderSyncComboLeg{}
	if v == nil {
		m.missing()
		return r
	}
	r.PositionID = v.PositionId
	r.Market = *m.mapMarketRef(v.Market)
	return r
}
func (m *responseMapper) mapTradeMetadata(v *trpc.TradeMetadata) *app.TraderSyncTradeMetadata {
	r := &app.TraderSyncTradeMetadata{}
	if v == nil {
		m.missing()
		return r
	}
	r.Market = *m.mapMarketRef(v.Market)
	r.LegsEvidence = *m.mapFieldEvidence(v.LegsEvidence)
	if v.Legs != nil {
		r.Legs = make([]app.TraderSyncComboLeg, 0, len(v.Legs.Items))
		for _, item := range v.Legs.Items {
			r.Legs = append(r.Legs, *m.mapComboLeg(item))
		}
	}
	r.Relationship = v.Relationship
	return r
}
func (m *responseMapper) mapDelivery(v *trpc.Delivery) *app.TraderSyncDelivery {
	r := &app.TraderSyncDelivery{}
	if v == nil {
		m.missing()
		return r
	}
	r.ID = v.Id
	r.Status = v.Status
	r.Reason = v.Reason
	r.AuthorizedAt = stringValue(v.AuthorizedAt)
	r.StartedAt = stringValue(v.StartedAt)
	r.ResultAt = stringValue(v.ResultAt)
	r.MessageID = stringValue(v.MessageId)
	r.AttemptCount = v.AttemptCount
	if v.LatestAttempt != nil {
		r.LatestAttempt = m.mapAttempt(v.LatestAttempt)
	}
	return r
}
func (m *responseMapper) mapAttempt(v *trpc.Attempt) *app.TraderSyncAttempt {
	r := &app.TraderSyncAttempt{}
	if v == nil {
		m.missing()
		return r
	}
	r.Index = v.Index
	r.AuthorizedAt = v.AuthorizedAt
	r.StartedAt = stringValue(v.StartedAt)
	r.ResultAt = stringValue(v.ResultAt)
	r.Status = v.Status
	r.Reason = v.Reason
	return r
}
func (m *responseMapper) mapStatusCounts(v *trpc.StatusCounts) *app.TraderSyncStatusCounts {
	r := &app.TraderSyncStatusCounts{}
	if v == nil {
		m.missing()
		return r
	}
	r.Total = v.Total
	r.Pending = v.Pending
	r.Sending = v.Sending
	r.Sent = v.Sent
	r.Failed = v.Failed
	r.Unknown = v.Unknown
	r.Cancelled = v.Cancelled
	return r
}
func (m *responseMapper) mapSummaryProgress(v *trpc.SummaryProgress) *app.TraderSyncSummaryProgress {
	r := &app.TraderSyncSummaryProgress{}
	if v == nil {
		m.missing()
		return r
	}
	r.Phase = v.Phase
	r.Reason = v.Reason
	r.BatchID = stringValue(v.BatchId)
	r.RelatedPartCounts = *m.mapStatusCounts(v.RelatedPartCounts)
	r.BatchPartCounts = *m.mapStatusCounts(v.BatchPartCounts)
	r.OldestAt = v.OldestAt
	r.FirstStartedAt = stringValue(v.FirstStartedAt)
	return r
}
func (m *responseMapper) mapTargetCount(v *trpc.TargetCount) *app.TraderSyncTargetCount {
	r := &app.TraderSyncTargetCount{}
	if v == nil {
		m.missing()
		return r
	}
	r.Wallet = v.Wallet
	r.Count = v.Count
	return r
}
func (m *responseMapper) mapSummaryBatch(v *trpc.SummaryBatch) *app.TraderSyncSummaryBatch {
	r := &app.TraderSyncSummaryBatch{}
	if v == nil {
		m.missing()
		return r
	}
	r.ID = v.Id
	r.OldestAt = v.OldestAt
	r.SettledFrom = v.SettledFrom
	r.SettledTo = v.SettledTo
	r.RecordedFrom = v.RecordedFrom
	r.RecordedTo = v.RecordedTo
	r.FirstStartedAt = stringValue(v.FirstStartedAt)
	r.ActivityCount = v.ActivityCount
	if v.TargetCounts != nil {
		r.TargetCounts = make([]app.TraderSyncTargetCount, 0, len(v.TargetCounts))
		for _, item := range v.TargetCounts {
			r.TargetCounts = append(r.TargetCounts, *m.mapTargetCount(item))
		}
	}
	r.PartCounts = *m.mapStatusCounts(v.PartCounts)
	r.AsOf = v.AsOf
	return r
}
func (m *responseMapper) mapSummaryPart(v *trpc.SummaryPart) *app.TraderSyncSummaryPart {
	r := &app.TraderSyncSummaryPart{}
	if v == nil {
		m.missing()
		return r
	}
	r.ID = v.Id
	r.Index = v.Index
	r.Total = v.Total
	r.Delivery = *m.mapDelivery(v.Delivery)
	r.AssociatedActivityCount = v.AssociatedActivityCount
	return r
}
func (m *responseMapper) mapSubscriptionSummary(v *trpc.SubscriptionSummary) *app.TraderSyncSubscriptionSummary {
	r := &app.TraderSyncSubscriptionSummary{}
	if v == nil {
		m.missing()
		return r
	}
	r.SubscriptionID = v.SubscriptionId
	r.AccountID = v.AccountId
	r.Username = v.Username
	r.Email = v.Email
	r.Wallet = v.Wallet
	r.Status = v.Status
	r.CreatedAt = v.CreatedAt
	r.UpdatedAt = v.UpdatedAt
	r.PausedAt = stringValue(v.PausedAt)
	r.CancelledAt = stringValue(v.CancelledAt)
	r.PermissionDisabledAt = stringValue(v.PermissionDisabledAt)
	r.Observation = *m.mapObservation(v.Observation)
	r.ActivityCount = v.ActivityCount
	r.AssociatedDeliveryCounts = *m.mapStatusCounts(v.AssociatedDeliveryCounts)
	r.AsOf = v.AsOf
	return r
}
func (m *responseMapper) mapRuntimeMetric(v *trpc.RuntimeMetric) *app.TraderSyncRuntimeMetric {
	r := &app.TraderSyncRuntimeMetric{}
	if v == nil {
		m.missing()
		return r
	}
	r.Name = v.Name
	r.Value = v.Value
	r.Unit = v.Unit
	r.Kind = v.Kind
	r.WindowStart = stringValue(v.WindowStart)
	r.WindowEnd = stringValue(v.WindowEnd)
	r.ServiceEpoch = stringValue(v.ServiceEpoch)
	return r
}
func (m *responseMapper) mapRuntimeStatus(v *trpc.RuntimeStatus) *app.TraderSyncRuntimeStatus {
	r := &app.TraderSyncRuntimeStatus{}
	if v == nil {
		m.missing()
		return r
	}
	r.CollectorConnected = v.CollectorConnected
	r.CollectorEpoch = v.CollectorEpoch
	r.FilterRevision = v.FilterRevision
	if v.Metrics != nil {
		r.Metrics = make([]app.TraderSyncRuntimeMetric, 0, len(v.Metrics))
		for _, item := range v.Metrics {
			r.Metrics = append(r.Metrics, *m.mapRuntimeMetric(item))
		}
	}
	r.AsOf = v.AsOf
	return r
}
func (m *responseMapper) mapPageInfo(v *trpc.PageInfo) *api.PageInfo {
	r := &api.PageInfo{}
	if v == nil {
		m.missing()
		return r
	}
	r.NextCursor = v.NextCursor
	return r
}
func (m *responseMapper) mapActivityPageInfo(v *trpc.ActivityPageInfo) *api.ActivityPageInfo {
	r := &api.ActivityPageInfo{}
	if v == nil {
		m.missing()
		return r
	}
	r.NextCursor = v.NextCursor
	r.RefreshCursor = v.RefreshCursor
	r.Snapshot = v.Snapshot
	r.AsOf = v.AsOf
	r.HasNewer = v.HasNewer
	return r
}
func (m *responseMapper) mapResolveTargetResponse(v *trpc.ResolveTargetResponse) *api.ResolveTargetResponse {
	r := &api.ResolveTargetResponse{}
	if v == nil {
		m.missing()
		return r
	}
	r.Target = m.mapResolvedTarget(v.Target)
	return r
}
func (m *responseMapper) mapCreateSubscriptionResponse(v *trpc.CreateSubscriptionResponse) *api.CreateSubscriptionResponse {
	r := &api.CreateSubscriptionResponse{}
	if v == nil {
		m.missing()
		return r
	}
	r.Subscription = m.mapSubscription(v.Subscription)
	return r
}
func (m *responseMapper) mapListSubscriptionsResponse(v *trpc.ListSubscriptionsResponse) *api.ListSubscriptionsResponse {
	r := &api.ListSubscriptionsResponse{}
	if v == nil {
		m.missing()
		return r
	}
	if v.Subscriptions != nil {
		r.Subscriptions = make([]*app.TraderSyncSubscription, 0, len(v.Subscriptions))
		for _, item := range v.Subscriptions {
			r.Subscriptions = append(r.Subscriptions, m.mapSubscription(item))
		}
	}
	r.Page = m.mapPageInfo(v.Page)
	r.Quota = m.mapQuota(v.Quota)
	r.AsOf = v.AsOf
	return r
}
func (m *responseMapper) mapGetSubscriptionResponse(v *trpc.GetSubscriptionResponse) *api.GetSubscriptionResponse {
	r := &api.GetSubscriptionResponse{}
	if v == nil {
		m.missing()
		return r
	}
	r.Subscription = m.mapSubscription(v.Subscription)
	return r
}
func (m *responseMapper) mapPauseSubscriptionResponse(v *trpc.PauseSubscriptionResponse) *api.PauseSubscriptionResponse {
	r := &api.PauseSubscriptionResponse{}
	if v == nil {
		m.missing()
		return r
	}
	r.Subscription = m.mapSubscription(v.Subscription)
	return r
}
func (m *responseMapper) mapResumeSubscriptionResponse(v *trpc.ResumeSubscriptionResponse) *api.ResumeSubscriptionResponse {
	r := &api.ResumeSubscriptionResponse{}
	if v == nil {
		m.missing()
		return r
	}
	r.Subscription = m.mapSubscription(v.Subscription)
	return r
}
func (m *responseMapper) mapCancelSubscriptionResponse(v *trpc.CancelSubscriptionResponse) *api.CancelSubscriptionResponse {
	r := &api.CancelSubscriptionResponse{}
	if v == nil {
		m.missing()
		return r
	}
	r.Subscription = m.mapSubscription(v.Subscription)
	return r
}
func (m *responseMapper) mapUpdateTargetNoteResponse(v *trpc.UpdateTargetNoteResponse) *api.UpdateTargetNoteResponse {
	r := &api.UpdateTargetNoteResponse{}
	if v == nil {
		m.missing()
		return r
	}
	r.Note = m.mapTargetNote(v.Note)
	return r
}
func (m *responseMapper) mapListActivitiesResponse(v *trpc.ListActivitiesResponse) *api.ListActivitiesResponse {
	r := &api.ListActivitiesResponse{}
	if v == nil {
		m.missing()
		return r
	}
	if v.Activities != nil {
		r.Activities = make([]*app.TraderSyncActivity, 0, len(v.Activities))
		for _, item := range v.Activities {
			r.Activities = append(r.Activities, m.mapActivity(item))
		}
	}
	r.Page = m.mapActivityPageInfo(v.Page)
	return r
}
func (m *responseMapper) mapGetActivityResponse(v *trpc.GetActivityResponse) *api.GetActivityResponse {
	r := &api.GetActivityResponse{}
	if v == nil {
		m.missing()
		return r
	}
	r.Activity = m.mapActivity(v.Activity)
	return r
}
func (m *responseMapper) mapListSubscriptionHistoryResponse(v *trpc.ListSubscriptionHistoryResponse) *api.ListSubscriptionHistoryResponse {
	r := &api.ListSubscriptionHistoryResponse{}
	if v == nil {
		m.missing()
		return r
	}
	if v.Entries != nil {
		r.Entries = make([]*app.TraderSyncHistoryEntry, 0, len(v.Entries))
		for _, item := range v.Entries {
			r.Entries = append(r.Entries, m.mapHistoryEntry(item))
		}
	}
	r.Page = m.mapPageInfo(v.Page)
	r.AsOf = v.AsOf
	return r
}
func (m *responseMapper) mapGetSummaryBatchResponse(v *trpc.GetSummaryBatchResponse) *api.GetSummaryBatchResponse {
	r := &api.GetSummaryBatchResponse{}
	if v == nil {
		m.missing()
		return r
	}
	r.Batch = m.mapSummaryBatch(v.Batch)
	return r
}
func (m *responseMapper) mapListSummaryPartsResponse(v *trpc.ListSummaryPartsResponse) *api.ListSummaryPartsResponse {
	r := &api.ListSummaryPartsResponse{}
	if v == nil {
		m.missing()
		return r
	}
	if v.Parts != nil {
		r.Parts = make([]*app.TraderSyncSummaryPart, 0, len(v.Parts))
		for _, item := range v.Parts {
			r.Parts = append(r.Parts, m.mapSummaryPart(item))
		}
	}
	r.Page = m.mapPageInfo(v.Page)
	r.AsOf = v.AsOf
	return r
}
func (m *responseMapper) mapListSubscriptionSummariesResponse(v *trpc.ListSubscriptionSummariesResponse) *api.ListSubscriptionSummariesResponse {
	r := &api.ListSubscriptionSummariesResponse{}
	if v == nil {
		m.missing()
		return r
	}
	if v.Summaries != nil {
		r.Summaries = make([]*app.TraderSyncSubscriptionSummary, 0, len(v.Summaries))
		for _, item := range v.Summaries {
			r.Summaries = append(r.Summaries, m.mapSubscriptionSummary(item))
		}
	}
	r.Page = m.mapPageInfo(v.Page)
	r.AsOf = v.AsOf
	return r
}
func (m *responseMapper) mapGetSubscriptionSummaryResponse(v *trpc.GetSubscriptionSummaryResponse) *api.GetSubscriptionSummaryResponse {
	r := &api.GetSubscriptionSummaryResponse{}
	if v == nil {
		m.missing()
		return r
	}
	r.Summary = m.mapSubscriptionSummary(v.Summary)
	return r
}
func (m *responseMapper) mapGetTraderSyncRuntimeStatusResponse(v *trpc.GetTraderSyncRuntimeStatusResponse) *api.GetTraderSyncRuntimeStatusResponse {
	r := &api.GetTraderSyncRuntimeStatusResponse{}
	if v == nil {
		m.missing()
		return r
	}
	r.Status = m.mapRuntimeStatus(v.Status)
	return r
}
