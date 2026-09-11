package tradersync

import (
	"github.com/ethereum/go-ethereum/common"
	tm "github.com/useryege/athena/internal/tradersync/types"
	"testing"
	"time"
)

func TestPublicActivityPreservesExactFactsAndMissingEvidence(t *testing.T) {
	at := time.Date(2026, 9, 11, 1, 2, 3, 400, time.UTC)
	name := "Name at confirmation"
	v := tm.ActivityDetails{Activity: tm.Activity{ID: 9007199254740993, SourceID: 123, SubscriptionID: "sub", Trade: tm.Trade{Wallet: common.HexToAddress("0x123"), Side: "BUY", PositionID: "90071992547409931234567890", CollateralRaw: "0", SharesRaw: "0", FeeRaw: "0", CollateralSymbol: "USDC", SourceVersion: "version"}, RecordedAt: at, NotificationMode: "ordinary", TargetDisplaySnapshot: tm.TargetDisplay{DisplayName: tm.Scalar{Evidence: tm.Evidence{Availability: "available", Source: "confirmed", QueriedAt: at}, Value: &name}}}, Delivery: &tm.DeliveryDetails{ID: 9, Status: "sent", AttemptCount: 1, ResultAt: &at}, SourceLocation: tm.SourceLocation{ChainID: 137, LogIndex: 0, Exchange: common.HexToAddress("0x456"), TransactionHash: common.HexToHash("0xabc"), BlockHash: common.HexToHash("0xdef")}}
	got := publicActivity(v)
	if got.ID != "9007199254740993" || got.PositionID != v.Trade.PositionID || got.CollateralRaw != "0" {
		t.Fatalf("exact source facts lost: %+v", got)
	}
	if got.Delivery == nil || got.Delivery.Status != "sent" || got.Delivery.StartedAt != nil || got.Delivery.AuthorizedAt != nil || got.Delivery.LatestAttempt != nil {
		t.Fatal("delivery missing evidence fabricated", got.Delivery)
	}
	if got.PublicTimeEvidence.Availability != "unavailable" || got.PublicTimeEvidence.ReasonCode != "public_time_unobservable" || got.SourceLocation.LogIndex != "0" || got.FinalityAnomaly != nil {
		t.Fatal("source evidence missing or invented", got)
	}
	if got.TargetDisplaySnapshot.DisplayName.Value == nil || *got.TargetDisplaySnapshot.DisplayName.Value != name {
		t.Fatal("display snapshot lost")
	}
}
func TestPublicResolvedPreservesZeroFalseAndUnavailable(t *testing.T) {
	zero, no, empty := "0", "false", ""
	at := time.Now().UTC()
	available := tm.Evidence{Availability: "available", Source: "fixture", QueriedAt: at}
	v := tm.ResolvedTarget{Card: tm.ConfirmationCard{Identity: tm.Identity{Wallet: common.HexToAddress("0x123")}, Verified: tm.Scalar{Evidence: available, Value: &no}, PositionValue: tm.Scalar{Evidence: available, Value: &zero}, DisplayName: tm.Scalar{Evidence: available, Value: &empty}, LargestWin: tm.Scalar{Evidence: tm.Evidence{Availability: "unavailable", ReasonCode: "query_failed"}}}}
	got := publicResolved(v)
	if got.Verified.Value == nil || *got.Verified.Value || got.PositionValue.Value == nil || *got.PositionValue.Value != "0" || got.DisplayName.Value == nil || *got.DisplayName.Value != "" {
		t.Fatal("available zero/false/empty lost", got)
	}
	if got.LargestWin.Value != nil || got.SavedNote != nil || len(got.PnL) != 6 || got.DefaultPeriod != "1Y" {
		t.Fatal("missing or six-period semantics changed", got)
	}
}
