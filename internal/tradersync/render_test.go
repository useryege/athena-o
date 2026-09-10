package tradersync

import (
	"github.com/ethereum/go-ethereum/common"
	tm "github.com/useryege/athena/internal/tradersync/types"
	"strings"
	"testing"
	"time"
)

func TestClassificationRetainsFirstTenOrdinary(t *testing.T) {
	for i := int64(1); i <= 12; i++ {
		want := "ordinary"
		if i > 10 {
			want = "summary"
		}
		if got := ClassifyActivity(i); got != want {
			t.Fatalf("activity %d: got %q want %q", i, got, want)
		}
	}
}

func TestRenderActivityPreservesExactFactsAndPlainCharacters(t *testing.T) {
	a := tm.Activity{ID: 17, NoteSnapshot: "<鲸>&🙂", Trade: tm.Trade{Wallet: common.HexToAddress("0x1111111111111111111111111111111111111111"), Side: "SELL", PositionID: "900719925474099399999", CollateralRaw: "9007199254740993", CollateralDecimals: 6, SharesRaw: "1", SharesDecimals: 6, FeeRaw: "2", CollateralSymbol: "USDC", PriceNumerator: "1", PriceDenominator: "3"}, SettledAt: time.Date(2026, 9, 10, 1, 2, 3, 0, time.UTC), Metadata: tm.TradeMetadata{Market: tm.MarketRef{Evidence: tm.Evidence{Availability: "unavailable", ReasonCode: "market_not_found"}}}}
	text, err := RenderActivity(a, "https://athena.test")
	if err != nil {
		t.Fatal(err)
	}
	for _, value := range []string{"<鲸>&🙂", a.Trade.Wallet.Hex(), "SELL", "900719925474099399999", "9007199254.740993", "0.000001", "0.000002", "1/3", "USDC", "结算时间", "2026-09-10", "https://athena.test/trader-sync/activities/17", "market_not_found"} {
		if !strings.Contains(text, value) {
			t.Fatalf("missing exact fact %q in %s", value, text)
		}
	}
	if strings.Contains(text, "&lt;") {
		t.Fatal("plain content escaped as HTML")
	}
	a.Metadata.Relationship = "NOT AND"
	a.Metadata.Market.Outcome = "NO"
	a.Metadata.LegsEvidence.Availability = "available"
	a.Metadata.Legs = []tm.ComboLeg{{PositionID: "1"}, {PositionID: "2"}}
	text, err = RenderActivity(a, "https://athena.test")
	if err != nil || !strings.Contains(text, "NOT AND") || !strings.Contains(text, "NO") || !strings.Contains(text, "0/2") {
		t.Fatalf("combo logic missing: %s %v", text, err)
	}
}

func TestRenderConfirmedNameDisclosesSnapshotQueryTime(t *testing.T) {
	name := "公开名称"
	at := time.Date(2026, 9, 10, 1, 2, 3, 0, time.UTC)
	a := tm.Activity{ID: 1, SettledAt: at, Trade: tm.Trade{Wallet: common.HexToAddress("0x123"), Side: "BUY", PositionID: "1", CollateralRaw: "1", SharesRaw: "1", FeeRaw: "0", PriceNumerator: "1", PriceDenominator: "1"}, TargetDisplaySnapshot: tm.TargetDisplay{DisplayName: tm.Scalar{Value: &name, Evidence: tm.Evidence{Availability: "available", QueriedAt: at}}}}
	text, e := RenderActivity(a, "https://athena.test")
	if e != nil {
		t.Fatal(e)
	}
	if !strings.HasPrefix(text, name+"\n") || !strings.Contains(text, "确认时资料") || !strings.Contains(text, "可能陈旧") || !strings.Contains(text, "2026-09-10 09:02:03") {
		t.Fatal("snapshot name lacks time/staleness disclosure", text)
	}
	a.NoteSnapshot = "备注"
	text, e = RenderActivity(a, "https://athena.test")
	if e != nil || !strings.HasPrefix(text, "备注\n") || !strings.Contains(text, a.Trade.Wallet.Hex()) {
		t.Fatal(text, e)
	}
}
