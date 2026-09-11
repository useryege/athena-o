package tradersync

import (
	"bytes"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/useryege/athena/internal/notification/delivery"
	tm "github.com/useryege/athena/internal/tradersync/types"
	"strings"
	"testing"
	"time"
	"unicode/utf16"
	"unicode/utf8"
)

func TestSummaryWindowHasBothBounds(t *testing.T) {
	p := time.Unix(100, 0)
	for _, tc := range []struct {
		old             int64
		previous        *time.Time
		start, deadline int64
	}{
		{110, &p, 160, 170}, {170, &p, 170, 230}, {110, nil, 110, 170},
	} {
		start, end := SummaryWindow(time.Unix(tc.old, 0), tc.previous)
		if !start.Equal(time.Unix(tc.start, 0)) || !end.Equal(time.Unix(tc.deadline, 0)) {
			t.Fatalf("window=%v,%v want seconds=%d,%d", start, end, tc.start, tc.deadline)
		}
	}
}

func summaryItem(id int64) tm.Activity {
	return tm.Activity{ID: id, NoteSnapshot: fmt.Sprintf("备注%d <>&🙂", id%2), RecordedAt: time.Unix(100+id, 0), SettledAt: time.Unix(90+id, 0), Trade: tm.Trade{Wallet: common.HexToAddress("0x1111111111111111111111111111111111111111"), Side: "BUY", PositionID: fmt.Sprint(id), CollateralRaw: "10", SharesRaw: "20", FeeRaw: "0", PriceNumerator: "1", PriceDenominator: "2", CollateralSymbol: "USDC"}, Metadata: tm.TradeMetadata{Market: tm.MarketRef{Evidence: tm.Evidence{Availability: "available"}, ID: fmt.Sprint(id), Title: fmt.Sprintf("完整市场%d", id), Outcome: "YES", URL: fmt.Sprintf("https://polymarket.com/event/market-%d", id)}}}
}

func TestRenderSummaryCompletePartsAndCrossPartCombo(t *testing.T) {
	var items []tm.Activity
	for i := int64(1); i <= 20; i++ {
		items = append(items, summaryItem(i))
	}
	combo := &items[0]
	combo.Metadata.Relationship = "NOT AND"
	combo.Metadata.Market.Outcome = "NO"
	combo.Metadata.LegsEvidence.Availability = "available"
	for i := 0; i < 60; i++ {
		combo.Metadata.Legs = append(combo.Metadata.Legs, tm.ComboLeg{PositionID: fmt.Sprint(100 + i), Market: tm.MarketRef{Evidence: tm.Evidence{Availability: "available"}, ConditionID: fmt.Sprintf("condition-%02d", i), Title: strings.Repeat("🙂", 90) + fmt.Sprint(i), Outcome: "YES", URL: fmt.Sprintf("https://polymarket.com/event/leg-%02d", i)}})
	}
	parts, err := RenderSummary(items, "37", "https://athena.test/deployment/")
	if err != nil {
		t.Fatal(err)
	}
	if len(parts) < 3 {
		t.Fatalf("complete summary needs multiple parts; got %d", len(parts))
	}
	all := ""
	seen := map[int64]int{}
	for i, p := range parts {
		if p.Index != i+1 || p.Total != len(parts) || !strings.Contains(p.Text, fmt.Sprintf("%d/%d", i+1, len(parts))) {
			t.Fatalf("unstable pagination: %+v", p)
		}
		if utf8.RuneCountInString(p.Text) > 4096 || len(utf16.Encode([]rune(p.Text))) > 4096 {
			t.Fatalf("oversized part %d", i+1)
		}
		payload, e := delivery.EncodePayload(delivery.Payload{Format: "plain", Text: p.Text})
		if e != nil || !bytes.Equal(delivery.PayloadDigest(payload), p.PayloadDigest) {
			t.Fatal("digest differs from frozen plain payload", e)
		}
		if !strings.Contains(p.Text, "https://athena.test/deployment/trader-sync/summaries/37") {
			t.Fatal("missing batch link")
		}
		if len(p.ActivityIDs) == 0 {
			t.Fatal("part without activity mapping")
		}
		for _, id := range p.ActivityIDs {
			seen[id]++
		}
		all += p.Text + "\n"
	}
	for _, a := range items {
		for _, fact := range []string{a.NoteSnapshot, a.Trade.Wallet.Hex(), a.Metadata.Market.Title, a.Metadata.Market.URL, "https://athena.test/deployment/trader-sync/activities/" + fmt.Sprint(a.ID)} {
			if !strings.Contains(all, fact) {
				t.Fatalf("missing fact %q", fact)
			}
		}
		if seen[a.ID] == 0 {
			t.Fatalf("missing activity %d", a.ID)
		}
	}
	for _, leg := range combo.Metadata.Legs {
		for _, fact := range []string{leg.Market.ConditionID, leg.Market.URL, leg.PositionID} {
			if !strings.Contains(all, fact) {
				t.Fatalf("missing combo fact %q", fact)
			}
		}
	}
	if seen[1] < 2 || !strings.Contains(all, "NOT AND") || !strings.Contains(all, "延续") || strings.Contains(all, "&lt;") {
		t.Fatal("cross-part combo or plain content lost")
	}
}

func TestRenderSummaryLongRowAndNotesAreNotTruncatedOrMerged(t *testing.T) {
	a, b := summaryItem(1), summaryItem(2)
	b.Trade = a.Trade
	b.Metadata = a.Metadata
	a.NoteSnapshot = strings.Repeat("😀", 2300) + "末尾事实"
	parts, e := RenderSummary([]tm.Activity{a, b}, "9", "https://athena.test/base")
	if e != nil {
		t.Fatal(e)
	}
	var all string
	for _, p := range parts {
		all += p.Text
		if len(utf16.Encode([]rune(p.Text))) > 4096 {
			t.Fatal("long row overflow")
		}
	}
	if !strings.Contains(all, "末尾事实") || !strings.Contains(all, b.NoteSnapshot) || !strings.Contains(all, "延续") {
		t.Fatal("long row/independent note lost")
	}
}

func TestRenderSummaryPageNumberGrowthKeepsBothLimits(t *testing.T) {
	for length := 3750; length < 4100; length++ {
		var items []tm.Activity
		for i := int64(1); i <= 12; i++ {
			a := summaryItem(i)
			a.NoteSnapshot = strings.Repeat("x", length)
			items = append(items, a)
		}
		parts, e := RenderSummary(items, "1", "https://athena.test/base")
		if e != nil {
			t.Fatal(e)
		}
		for _, p := range parts {
			if units := len(utf16.Encode([]rune(p.Text))); units > 4096 {
				t.Fatalf("length=%d page=%d/%d units=%d", length, p.Index, p.Total, units)
			}
		}
	}
}
