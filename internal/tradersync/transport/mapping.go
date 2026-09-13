package transport

import (
	"math/big"
	"strconv"
	"time"

	trpc "github.com/useryege/athena/internal/tradersync/apiclient"
	tm "github.com/useryege/athena/internal/tradersync/types"
)

func utc(v time.Time) string {
	if v.IsZero() {
		return ""
	}
	return v.UTC().Format(time.RFC3339Nano)
}
func optionalTime(v *time.Time) *trpc.StringValue {
	if v == nil {
		return nil
	}
	s := utc(*v)
	return &trpc.StringValue{Value: s}
}
func number(v int64) string { return strconv.FormatInt(v, 10) }
func copyString(v *string) *trpc.StringValue {
	if v == nil {
		return nil
	}
	s := *v
	return &trpc.StringValue{Value: s}
}
func internalEvidence(v tm.Evidence) *trpc.FieldEvidence {
	if v.Availability == "" {
		v.Availability = "unavailable"
		if v.ReasonCode == "" {
			v.ReasonCode = "not_observed"
		}
	}
	return &trpc.FieldEvidence{Availability: v.Availability, ReasonCode: v.ReasonCode, Source: v.Source, QueriedAt: utc(v.QueriedAt)}
}
func internalString(v tm.Scalar) *trpc.StringField {
	r := &trpc.StringField{Evidence: internalEvidence(v.Evidence)}
	if v.Availability == "available" {
		r.Value = copyString(v.Value)
	}
	return r
}
func internalDecimal(v tm.Scalar) *trpc.DecimalField {
	r := internalString(v)
	return &trpc.DecimalField{Evidence: r.Evidence, Value: r.Value}
}
func internalBool(v tm.Scalar) *trpc.BoolField {
	r := &trpc.BoolField{Evidence: internalEvidence(v.Evidence)}
	if v.Availability == "available" && v.Value != nil {
		b, e := strconv.ParseBool(*v.Value)
		if e == nil {
			r.Value = &trpc.BoolValue{Value: b}
		} else {
			r.Evidence.Availability = "unavailable"
			r.Evidence.ReasonCode = "invalid_boolean"
		}
	}
	return r
}
func internalTime(v tm.Scalar) *trpc.TimeField {
	r := &trpc.TimeField{Evidence: internalEvidence(v.Evidence)}
	if v.Availability == "available" && v.Value != nil {
		t, e := time.Parse(time.RFC3339Nano, *v.Value)
		if e == nil {
			s := utc(t)
			r.Value = &trpc.StringValue{Value: s}
		} else {
			r.Evidence.Availability = "unavailable"
			r.Evidence.ReasonCode = "invalid_time"
		}
	}
	return r
}
func internalDisplay(v tm.TargetDisplay) *trpc.TargetDisplay {
	return &trpc.TargetDisplay{DisplayName: internalString(v.DisplayName), Avatar: internalString(v.Avatar), ProfileUrl: internalString(v.ProfileURL)}
}
func internalQuota(v tm.Quota) *trpc.Quota {
	return &trpc.Quota{Used: v.Used, Limit: v.Limit}
}
func internalNote(v tm.TargetNote) *trpc.TargetNote {
	return &trpc.TargetNote{Wallet: v.Wallet.Hex(), Note: v.Note, Revision: v.Revision}
}
func internalResolved(v tm.ResolvedTarget) *trpc.ResolvedTarget {
	c := v.Card
	r := &trpc.ResolvedTarget{Wallet: c.Identity.Wallet.Hex(), CanonicalProfileUrl: c.Identity.ProfileURL, Avatar: internalString(c.Avatar), DisplayName: internalString(c.DisplayName), Verified: internalBool(c.Verified), JoinedAt: internalTime(c.JoinedAt), PositionValue: internalDecimal(c.PositionValue), LargestWin: internalDecimal(c.LargestWin), Predictions: internalDecimal(c.Predictions), DefaultPeriod: "1Y", ConfirmationToken: v.Token, ExpiresAt: utc(v.ExpiresAt), UsageNotice: "优先选择低频交易者；高频监控不纳入性能保障", Quota: internalQuota(v.Context.Quota)}
	if v.Context.SavedNote != nil {
		r.SavedNote = internalNote(*v.Context.SavedNote)
	}
	if e := v.Context.Existing; e != nil {
		r.ExistingSubscription = &trpc.ExistingSubscription{Id: e.ID, Status: e.Status, Revision: e.Revision}
	}
	for _, period := range []string{"1D", "1W", "1M", "1Y", "YTD", "ALL"} {
		p := c.PnL[period]
		curve := &trpc.Curve{Evidence: internalEvidence(p.Curve.Evidence)}
		if p.Curve.Points != nil {
			curve.Points = &trpc.CurvePointList{Items: make([]*trpc.CurvePoint, 0, len(p.Curve.Points))}
			for _, point := range p.Curve.Points {
				curve.Points.Items = append(curve.Points.Items, &trpc.CurvePoint{T: number(point.T), P: point.P})
			}
		}
		reference := &trpc.TimeField{Evidence: &trpc.FieldEvidence{Availability: "unavailable", ReasonCode: "official_reference_unavailable", Source: p.Amount.Source, QueriedAt: utc(p.Amount.QueriedAt)}}
		if p.Reference != nil {
			reference.Value = optionalTime(p.Reference)
			reference.Evidence.Availability = "available"
			reference.Evidence.ReasonCode = ""
		}
		zone := &trpc.StringField{Evidence: &trpc.FieldEvidence{Availability: "unavailable", ReasonCode: "official_timezone_unavailable", Source: p.Amount.Source, QueriedAt: utc(p.Amount.QueriedAt)}}
		if p.Timezone != "" {
			zone.Value = copyString(&p.Timezone)
			zone.Evidence.Availability = "available"
			zone.Evidence.ReasonCode = ""
		}
		r.PnL = append(r.PnL, &trpc.PnLView{Period: period, Amount: internalDecimal(p.Amount), Curve: curve, Interval: p.Interval, Fidelity: p.Fidelity, ReferenceTime: reference, Timezone: zone})
	}
	return r
}
func internalCounts(v tm.StatusCounts) *trpc.StatusCounts {
	return &trpc.StatusCounts{Total: number(v.Total), Pending: number(v.Pending), Sending: number(v.Sending), Sent: number(v.Sent), Failed: number(v.Failed), Unknown: number(v.Unknown), Cancelled: number(v.Cancelled)}
}
func internalInterruption(v *tm.InterruptionDetails) *trpc.Interruption {
	if v == nil {
		return nil
	}
	return &trpc.Interruption{Start: optionalTime(v.Start), End: optionalTime(v.End), RecoveredAt: optionalTime(v.RecoveredAt), Reason: v.Reason, Uncertainty: v.Uncertainty, PossibleMissing: v.PossibleMissing}
}
func internalObservation(v tm.ObservationDetails) *trpc.Observation {
	return &trpc.Observation{State: v.State, Reason: v.Reason, LastReliableAt: optionalTime(v.LastReliableAt), LatestInterruption: internalInterruption(v.LatestInterruption), InterruptionCount: number(v.InterruptionCount)}
}
func internalInterval(v *tm.IntervalDetails) *trpc.Interval {
	if v == nil {
		return nil
	}
	return &trpc.Interval{EffectiveAt: utc(v.EffectiveAt), EndedAt: optionalTime(v.EndedAt), Generation: v.Generation, Epoch: v.Epoch}
}
func internalSubscription(v tm.SubscriptionDetails) *trpc.Subscription {
	state := v.DesiredState
	if state == "enabled" {
		state = v.ObservationState
	}
	return &trpc.Subscription{Id: v.ID, Wallet: v.Wallet.Hex(), Status: state, Revision: v.Revision, Generation: v.Generation, Note: v.Note, NoteRevision: v.NoteRevision, CreatedAt: utc(v.CreatedAt), UpdatedAt: utc(v.UpdatedAt), PausedAt: optionalTime(v.PausedAt), CancelledAt: optionalTime(v.CancelledAt), PermissionDisabledAt: optionalTime(v.PermissionDisabledAt), CurrentInterval: internalInterval(v.CurrentInterval), Observation: internalObservation(v.Observation), BindingStatus: v.BindingStatus, QueueNotice: v.QueueNotice, QueueCounts: internalCounts(v.QueueCounts), TargetDisplay: internalDisplay(v.TargetDisplay)}
}
func internalDelivery(v *tm.DeliveryDetails) *trpc.Delivery {
	if v == nil {
		return nil
	}
	r := &trpc.Delivery{Id: number(v.ID), Status: v.Status, Reason: v.Reason, AuthorizedAt: optionalTime(v.AuthorizedAt), StartedAt: optionalTime(v.StartedAt), ResultAt: optionalTime(v.ResultAt), MessageId: copyString(v.MessageID), AttemptCount: number(v.AttemptCount)}
	if a := v.LatestAttempt; a != nil {
		r.LatestAttempt = &trpc.Attempt{Index: number(a.Index), AuthorizedAt: utc(a.AuthorizedAt), StartedAt: optionalTime(a.StartedAt), ResultAt: optionalTime(a.ResultAt), Status: a.Status, Reason: a.Reason}
	}
	return r
}
func internalMarket(v tm.MarketRef) *trpc.MarketRef {
	return &trpc.MarketRef{Evidence: internalEvidence(v.Evidence), Id: v.ID, Title: v.Title, Url: v.URL, ConditionId: v.ConditionID, PositionId: v.PositionID, Outcome: v.Outcome}
}
func internalMetadata(v tm.TradeMetadata) *trpc.TradeMetadata {
	r := &trpc.TradeMetadata{Market: internalMarket(v.Market), LegsEvidence: internalEvidence(v.LegsEvidence), Relationship: v.Relationship}
	if v.Legs != nil {
		r.Legs = &trpc.ComboLegList{Items: make([]*trpc.ComboLeg, 0, len(v.Legs))}
		for _, leg := range v.Legs {
			r.Legs.Items = append(r.Legs.Items, &trpc.ComboLeg{PositionId: leg.PositionID, Market: internalMarket(leg.Market)})
		}
	}
	return r
}
func internalSummaryProgress(v *tm.SummaryDetails) *trpc.SummaryProgress {
	if v == nil {
		return nil
	}
	r := &trpc.SummaryProgress{Phase: v.Phase, Reason: v.Reason, RelatedPartCounts: internalCounts(v.RelatedPartCounts), BatchPartCounts: internalCounts(v.BatchPartCounts), OldestAt: utc(v.OldestAt), FirstStartedAt: optionalTime(v.FirstStartedAt)}
	if v.BatchID != nil {
		n := number(*v.BatchID)
		r.BatchId = &trpc.StringValue{Value: n}
	}
	return r
}
func internalActivity(v tm.ActivityDetails) *trpc.Activity {
	t := v.Trade
	loc := v.SourceLocation
	price := &trpc.FieldEvidence{Availability: "unavailable", ReasonCode: "price_unavailable", Source: "source_record"}
	n, nOK := new(big.Int).SetString(t.PriceNumerator, 10)
	d, dOK := new(big.Int).SetString(t.PriceDenominator, 10)
	if nOK && dOK && n.Sign() >= 0 && d.Sign() > 0 {
		price.Availability = "available"
		price.ReasonCode = ""
	}
	r := &trpc.Activity{Id: number(v.ID), SubscriptionId: v.SubscriptionID, SourceRecordId: number(v.SourceID), Wallet: t.Wallet.Hex(), Side: t.Side, PositionId: t.PositionID, CollateralRaw: t.CollateralRaw, SharesRaw: t.SharesRaw, FeeRaw: t.FeeRaw, CollateralSymbol: t.CollateralSymbol, CollateralDecimals: int32(t.CollateralDecimals), SharesDecimals: int32(t.SharesDecimals), PriceNumerator: t.PriceNumerator, PriceDenominator: t.PriceDenominator, PriceEvidence: price, SourceVersion: t.SourceVersion, SettledAt: utc(v.SettledAt), ReceivedAt: utc(v.ReceivedAt), RecordedAt: utc(v.RecordedAt), PublicTimeEvidence: &trpc.FieldEvidence{Availability: "unavailable", ReasonCode: "public_time_unobservable", Source: "source_record"}, Metadata: internalMetadata(v.Metadata), NoteSnapshot: v.NoteSnapshot, NotificationMode: v.NotificationMode, NotificationReason: v.NotificationReason, Delivery: internalDelivery(v.Delivery), SummaryProgress: internalSummaryProgress(v.SummaryProgress), TargetDisplaySnapshot: internalDisplay(v.TargetDisplaySnapshot), SourceLocation: &trpc.SourceLocation{ChainId: number(loc.ChainID), ExchangeAddress: loc.Exchange.Hex(), TransactionHash: loc.TransactionHash.Hex(), BlockHash: loc.BlockHash.Hex(), BlockNumber: number(loc.BlockNumber), LogIndex: number(loc.LogIndex)}}
	if a := v.FinalityAnomaly; a != nil {
		r.FinalityAnomaly = &trpc.FinalityAnomaly{Reason: a.Reason, DetectedAt: utc(a.DetectedAt), PublishedBlockHash: a.PublishedBlockHash.Hex()}
		if a.ConflictingBlockHash != nil {
			h := a.ConflictingBlockHash.Hex()
			r.FinalityAnomaly.ConflictingBlockHash = &trpc.StringValue{Value: h}
		}
	}
	return r
}
func internalHistory(v tm.HistoryEntry) *trpc.HistoryEntry {
	return &trpc.HistoryEntry{Id: v.ID, Kind: v.Kind, SortAt: utc(v.SortAt), Interval: internalInterval(v.Interval), Interruption: internalInterruption(v.Interruption)}
}
func internalBatch(v tm.SummaryBatchDetails) *trpc.SummaryBatch {
	r := &trpc.SummaryBatch{Id: number(v.ID), OldestAt: utc(v.OldestAt), SettledFrom: utc(v.SettledFrom), SettledTo: utc(v.SettledTo), RecordedFrom: utc(v.RecordedFrom), RecordedTo: utc(v.RecordedTo), FirstStartedAt: optionalTime(v.FirstStartedAt), ActivityCount: number(v.ActivityCount), PartCounts: internalCounts(v.PartCounts), AsOf: utc(v.AsOf)}
	for _, t := range v.TargetCounts {
		r.TargetCounts = append(r.TargetCounts, &trpc.TargetCount{Wallet: t.Wallet.Hex(), Count: number(t.Count)})
	}
	return r
}
func internalPart(v tm.SummaryPartDetails) *trpc.SummaryPart {
	return &trpc.SummaryPart{Id: number(v.ID), Index: v.Index, Total: v.Total, Delivery: internalDelivery(&v.Delivery), AssociatedActivityCount: number(v.AssociatedActivityCount)}
}
func internalAdminSubscription(v tm.SubscriptionSummary) *trpc.SubscriptionSummary {
	return &trpc.SubscriptionSummary{SubscriptionId: v.SubscriptionID, AccountId: v.AccountID, Username: v.Username, Email: v.Email, Wallet: v.Wallet.Hex(), Status: v.Status, CreatedAt: utc(v.CreatedAt), UpdatedAt: utc(v.UpdatedAt), PausedAt: optionalTime(v.PausedAt), CancelledAt: optionalTime(v.CancelledAt), PermissionDisabledAt: optionalTime(v.PermissionDisabledAt), Observation: internalObservation(v.Observation), ActivityCount: number(v.ActivityCount), AssociatedDeliveryCounts: internalCounts(v.AssociatedDeliveryCounts), AsOf: utc(v.AsOf)}
}
func internalRuntime(v tm.RuntimeStatus) *trpc.RuntimeStatus {
	r := &trpc.RuntimeStatus{CollectorConnected: v.CollectorConnected, CollectorEpoch: v.CollectorEpoch, FilterRevision: v.FilterRevision, AsOf: utc(v.AsOf)}
	for _, m := range v.Metrics {
		r.Metrics = append(r.Metrics, &trpc.RuntimeMetric{Name: m.Name, Value: m.Value, Unit: m.Unit, Kind: m.Kind, WindowStart: optionalTime(m.WindowStart), WindowEnd: optionalTime(m.WindowEnd), ServiceEpoch: copyString(m.ServiceEpoch)})
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
