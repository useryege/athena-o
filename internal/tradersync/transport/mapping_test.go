package transport

import (
	"math"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gogo/protobuf/proto"
	"github.com/stretchr/testify/require"
	trpc "github.com/useryege/athena/internal/tradersync/apiclient"
	tm "github.com/useryege/athena/internal/tradersync/types"
)

func TestResolvedMappingPreservesPresenceAndEvidenceThroughWire(t *testing.T) {
	zero, no, empty, bad, timeValue := "0", "false", "", "garbage", "2026-09-13T10:03:04.000000005+08:00"
	evidence := tm.Evidence{Availability: "available", Source: "fixture"}
	input := tm.ResolvedTarget{Card: tm.ConfirmationCard{Identity: tm.Identity{Wallet: common.HexToAddress("0x123")}, Verified: tm.Scalar{Evidence: evidence, Value: &no}, PositionValue: tm.Scalar{Evidence: evidence, Value: &zero}, DisplayName: tm.Scalar{Evidence: evidence, Value: &empty}, JoinedAt: tm.Scalar{Evidence: evidence, Value: &timeValue}, LargestWin: tm.Scalar{Evidence: tm.Evidence{Availability: "unavailable", ReasonCode: "query_failed"}, Value: &zero}}}
	mapped := internalResolved(input)
	raw, err := proto.Marshal(mapped)
	require.NoError(t, err)
	got := new(trpc.ResolvedTarget)
	require.NoError(t, proto.Unmarshal(raw, got))
	require.NotNil(t, got.Verified.Value)
	require.False(t, got.Verified.Value.Value)
	require.NotNil(t, got.DisplayName.Value)
	require.Equal(t, "", got.DisplayName.Value.Value)
	require.Equal(t, "0", got.PositionValue.Value.Value)
	require.Nil(t, got.LargestWin.Value)
	require.Equal(t, "2026-09-13T02:03:04.000000005Z", got.JoinedAt.Value.Value)
	require.Nil(t, got.SavedNote)
	require.Nil(t, got.ExistingSubscription)
	require.NotNil(t, got.Quota)
	require.Equal(t, "1Y", got.DefaultPeriod)
	var periods []string
	for _, p := range got.PnL {
		periods = append(periods, p.Period)
		require.NotNil(t, p.Amount.Evidence)
		require.NotNil(t, p.Curve.Evidence)
		require.Nil(t, p.Curve.Points)
		require.Nil(t, p.ReferenceTime.Value)
		require.Equal(t, "official_reference_unavailable", p.ReferenceTime.Evidence.ReasonCode)
		require.Nil(t, p.Timezone.Value)
	}
	require.Equal(t, []string{"1D", "1W", "1M", "1Y", "YTD", "ALL"}, periods)
	input.Card.Verified.Value = &bad
	input.Card.JoinedAt.Value = &bad
	invalid := internalResolved(input)
	require.Nil(t, invalid.Verified.Value)
	require.Equal(t, "invalid_boolean", invalid.Verified.Evidence.ReasonCode)
	require.Nil(t, invalid.JoinedAt.Value)
	require.Equal(t, "invalid_time", invalid.JoinedAt.Evidence.ReasonCode)
}
func TestCurveAndLegListsPreserveUnknownEmptyAndOrder(t *testing.T) {
	input := tm.ResolvedTarget{}
	input.Card.PnL = map[string]tm.PnLView{}
	input.Card.PnL["1D"] = tm.PnLView{Curve: tm.Curve{Points: []tm.PnLPoint{}}}
	input.Card.PnL["1W"] = tm.PnLView{Curve: tm.Curve{Points: []tm.PnLPoint{{T: 1726000000, P: "0"}, {T: 1726000000, P: "2.50"}}}}
	mapped := internalResolved(input)
	raw, err := proto.Marshal(mapped)
	require.NoError(t, err)
	got := new(trpc.ResolvedTarget)
	require.NoError(t, proto.Unmarshal(raw, got))
	require.NotNil(t, got.PnL[0].Curve.Points)
	require.Empty(t, got.PnL[0].Curve.Points.Items)
	require.Len(t, got.PnL[1].Curve.Points.Items, 2)
	require.Equal(t, "1726000000", got.PnL[1].Curve.Points.Items[0].T)
	require.Equal(t, "1726000000", got.PnL[1].Curve.Points.Items[1].T)
	require.Equal(t, "2.50", got.PnL[1].Curve.Points.Items[1].P)
	require.Nil(t, got.PnL[2].Curve.Points)
	require.Nil(t, internalMetadata(tm.TradeMetadata{}).Legs)
	require.NotNil(t, internalMetadata(tm.TradeMetadata{Legs: []tm.ComboLeg{}}).Legs)
	legs := internalMetadata(tm.TradeMetadata{Legs: []tm.ComboLeg{{PositionID: "2"}, {PositionID: "1"}, {PositionID: "2"}}}).Legs.Items
	require.Equal(t, []string{"2", "1", "2"}, []string{legs[0].PositionId, legs[1].PositionId, legs[2].PositionId})
}
func TestActivityAndSubscriptionMappingKeepExactFacts(t *testing.T) {
	at := time.Date(2026, 9, 13, 1, 2, 3, 4, time.FixedZone("offset", 3600))
	empty := ""
	input := tm.ActivityDetails{Activity: tm.Activity{ID: 9007199254740993, SourceID: 9007199254740995, Trade: tm.Trade{PositionID: "90071992547409931234567890", CollateralRaw: "123456789012345678901234567890", SharesRaw: "0", FeeRaw: "0", PriceNumerator: "0", PriceDenominator: "10000000000000000000"}, RecordedAt: at}, Delivery: &tm.DeliveryDetails{ID: 9007199254740997, MessageID: &empty, AttemptCount: 2}, SourceLocation: tm.SourceLocation{ChainID: 137, LogIndex: 0}, SummaryProgress: &tm.SummaryDetails{BatchID: new(int64)}}
	raw, err := proto.Marshal(internalActivity(input))
	require.NoError(t, err)
	got := new(trpc.Activity)
	require.NoError(t, proto.Unmarshal(raw, got))
	require.Equal(t, "9007199254740993", got.Id)
	require.Equal(t, "9007199254740995", got.SourceRecordId)
	require.Equal(t, "90071992547409931234567890", got.PositionId)
	require.Equal(t, "123456789012345678901234567890", got.CollateralRaw)
	require.Equal(t, "0", got.PriceNumerator)
	require.Equal(t, "available", got.PriceEvidence.Availability)
	require.Equal(t, "2026-09-13T00:02:03.000000004Z", got.RecordedAt)
	require.Equal(t, "", got.SettledAt)
	require.Nil(t, got.Delivery.AuthorizedAt)
	require.Nil(t, got.Delivery.StartedAt)
	require.Nil(t, got.Delivery.LatestAttempt)
	require.NotNil(t, got.Delivery.MessageId)
	require.Equal(t, "", got.Delivery.MessageId.Value)
	require.Equal(t, "0", got.SourceLocation.LogIndex)
	require.Equal(t, "0", got.SummaryProgress.BatchId.Value)
	require.Nil(t, got.FinalityAnomaly)
	require.Equal(t, "public_time_unobservable", got.PublicTimeEvidence.ReasonCode)
	for _, price := range []struct{ n, d string }{{"-1", "1"}, {"1", "0"}, {"garbage", "1"}} {
		input.Trade.PriceNumerator = price.n
		input.Trade.PriceDenominator = price.d
		require.Equal(t, "unavailable", internalActivity(input).PriceEvidence.Availability)
	}
	sub := internalSubscription(tm.SubscriptionDetails{Subscription: tm.Subscription{Revision: math.MaxUint64, Generation: math.MaxUint64, DesiredState: "enabled", ObservationState: "interrupted"}})
	require.Equal(t, "interrupted", sub.Status)
	require.Equal(t, uint64(math.MaxUint64), sub.Revision)
	require.Equal(t, uint64(math.MaxUint64), sub.Generation)
	require.Nil(t, sub.PausedAt)
	require.Nil(t, sub.CurrentInterval)
	require.NotNil(t, sub.Observation)
	require.NotNil(t, sub.TargetDisplay)
	sub = internalSubscription(tm.SubscriptionDetails{Subscription: tm.Subscription{DesiredState: "paused", ObservationState: "healthy"}})
	require.Equal(t, "paused", sub.Status)
}
func TestRemainingViewsPreserveOptionalAndRequiredObjects(t *testing.T) {
	empty := ""
	at := time.Date(2026, 9, 13, 0, 0, 0, 0, time.UTC)
	history := internalHistory(tm.HistoryEntry{ID: "entry", Kind: "interruption", Interruption: &tm.InterruptionDetails{End: &at, PossibleMissing: true}})
	require.Nil(t, history.Interval)
	require.Nil(t, history.Interruption.Start)
	require.Equal(t, "2026-09-13T00:00:00Z", history.Interruption.End.Value)
	require.True(t, history.Interruption.PossibleMissing)
	batch := internalBatch(tm.SummaryBatchDetails{ID: 9007199254740993, TargetCounts: []tm.TargetCount{{Count: 0}, {Count: 2}}})
	require.Equal(t, "9007199254740993", batch.Id)
	require.Len(t, batch.TargetCounts, 2)
	require.Equal(t, "0", batch.TargetCounts[0].Count)
	require.NotNil(t, batch.PartCounts)
	require.Nil(t, batch.FirstStartedAt)
	part := internalPart(tm.SummaryPartDetails{ID: 2, Index: 1, Total: 3, Delivery: tm.DeliveryDetails{LatestAttempt: &tm.AttemptDetails{Index: 2, AuthorizedAt: at}}})
	require.Equal(t, int32(1), part.Index)
	require.NotNil(t, part.Delivery.LatestAttempt)
	require.Equal(t, "2", part.Delivery.LatestAttempt.Index)
	summary := internalAdminSubscription(tm.SubscriptionSummary{AccountID: memberID, Observation: tm.ObservationDetails{State: "healthy"}})
	require.Equal(t, memberID, summary.AccountId)
	require.NotNil(t, summary.AssociatedDeliveryCounts)
	runtime := internalRuntime(tm.RuntimeStatus{Metrics: []tm.RuntimeMetric{{Name: "one", Value: "9007199254740993", ServiceEpoch: &empty}, {Name: "two", Value: "0"}}})
	require.Len(t, runtime.Metrics, 2)
	require.Equal(t, "9007199254740993", runtime.Metrics[0].Value)
	require.NotNil(t, runtime.Metrics[0].ServiceEpoch)
	require.Nil(t, runtime.Metrics[1].ServiceEpoch)
}
