package activity

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"
	"unicode/utf16"

	"github.com/useryege/athena/internal/notification/delivery"
	tm "github.com/useryege/athena/internal/tradersync/types"
)

type summaryLine struct {
	text string
	ids  []int64
	link bool
}

// utf16Length is the stricter of the two Telegram limits for valid UTF-8.
func utf16Length(s string) int { return len(utf16.Encode([]rune(s))) }

// RenderSummary freezes complete display facts, independent of notification I/O.
// Rows share a group only when wallet, note, market, outcome and side agree.
func RenderSummary(input []tm.Activity, batchID, siteURL string) ([]tm.RenderedPart, error) {
	u, e := url.Parse(siteURL)
	if e != nil || u.Host == "" || (u.Scheme != "https" && u.Scheme != "http") || u.RawQuery != "" || u.Fragment != "" || batchID == "" || strings.ContainsAny(batchID, "/?#") || len(input) == 0 {
		return nil, fmt.Errorf("invalid summary identity or site URL")
	}
	items := append([]tm.Activity(nil), input...)
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].RecordedAt.Equal(items[j].RecordedAt) {
			return items[i].ID < items[j].ID
		}
		return items[i].RecordedAt.Before(items[j].RecordedAt)
	})
	type groupKey struct{ wallet, note, market, position, outcome, side string }
	var keys []groupKey
	groups := map[groupKey][]tm.Activity{}
	targets := map[string]int{}
	ids := []int64{}
	seen := map[int64]bool{}
	for _, a := range items {
		if a.ID <= 0 || seen[a.ID] || a.RecordedAt.IsZero() {
			return nil, fmt.Errorf("invalid or duplicate summary activity")
		}
		seen[a.ID] = true
		// The ordinary validator checks exact numeric facts without using its truncated text.
		check := a
		check.NoteSnapshot = ""
		check.TargetDisplaySnapshot = tm.TargetDisplay{}
		check.Metadata.Market.Title = ""
		check.Metadata.Market.URL = ""
		if _, e := Render(check, siteURL); e != nil {
			return nil, e
		}
		k := groupKey{a.Trade.Wallet.Hex(), a.NoteSnapshot, a.Metadata.Market.ID, a.Trade.PositionID, a.Metadata.Market.Outcome, a.Trade.Side}
		if _, ok := groups[k]; !ok {
			keys = append(keys, k)
		}
		groups[k] = append(groups[k], a)
		targets[k.wallet]++
		ids = append(ids, a.ID)
	}
	base := strings.TrimRight(siteURL, "/")
	batchLink := base + "/trader-sync/summaries/" + batchID
	var lines []summaryLine
	add := func(text string, ids []int64) { lines = append(lines, summaryLine{text: text, ids: ids}) }
	link := func(text string, ids []int64) { lines = append(lines, summaryLine{text: text, ids: ids, link: true}) }
	zone := time.FixedZone("UTC+8", 8*3600)
	stamp := func(t time.Time) string { return t.In(zone).Format("2006-01-02 15:04:05.000") + " UTC+8" }
	add(fmt.Sprintf("覆盖形成时间: %s — %s\n活动总数: %d", stamp(items[0].RecordedAt), stamp(items[len(items)-1].RecordedAt), len(items)), ids)
	wallets := make([]string, 0, len(targets))
	for wallet := range targets {
		wallets = append(wallets, wallet)
	}
	sort.Strings(wallets)
	for _, wallet := range wallets {
		var related []int64
		for _, a := range items {
			if a.Trade.Wallet.Hex() == wallet {
				related = append(related, a.ID)
			}
		}
		add(fmt.Sprintf("目标 %s · %d 条", wallet, targets[wallet]), related)
	}
	for _, k := range keys {
		group := groups[k]
		related := make([]int64, len(group))
		for i, a := range group {
			related[i] = a.ID
		}
		first := group[0]
		name := k.note
		if name == "" && first.TargetDisplaySnapshot.DisplayName.Availability == "available" && first.TargetDisplaySnapshot.DisplayName.Value != nil {
			name = *first.TargetDisplaySnapshot.DisplayName.Value
		}
		if name == "" {
			name = k.wallet
		}
		add(fmt.Sprintf("目标/备注: %s\n钱包: %s\n市场: %s\nPositionID: %s · Outcome: %s · %s · %d 条", name, k.wallet, first.Metadata.Market.Title, k.position, k.outcome, k.side, len(group)), related)
		if !first.TargetDisplaySnapshot.DisplayName.QueriedAt.IsZero() {
			add("确认时资料（可能陈旧）: "+stamp(first.TargetDisplaySnapshot.DisplayName.QueriedAt), related)
		}
		// Preserve each member's frozen metadata, including independent unavailable legs.
		for _, a := range group {
			member := []int64{a.ID}
			m := a.Metadata.Market
			add(fmt.Sprintf("活动 %d · 市场 %s · Outcome %s · %s", a.ID, m.Title, m.Outcome, a.Trade.Side), member)
			if m.Availability != "available" {
				add("Market unavailable: "+m.ReasonCode, member)
			}
			if m.URL != "" {
				link(m.URL, member)
			}
			if a.Metadata.Relationship != "" {
				add(fmt.Sprintf("Combo 活动 %d: %s · Outcome %s（同一组合成交，腿不是独立交易）", a.ID, a.Metadata.Relationship, m.Outcome), member)
				for i, leg := range a.Metadata.Legs {
					add(fmt.Sprintf("Combo 活动 %d 延续 · 腿 %d/%d · PositionID %s · ConditionID %s\n%s · Outcome %s · %s %s", a.ID, i+1, len(a.Metadata.Legs), leg.PositionID, leg.Market.ConditionID, leg.Market.Title, leg.Market.Outcome, leg.Market.Availability, leg.Market.ReasonCode), member)
					if leg.Market.URL != "" {
						link(leg.Market.URL, member)
					}
				}
				if a.Metadata.LegsEvidence.ReasonCode != "" {
					add("Combo legs: "+a.Metadata.LegsEvidence.ReasonCode, member)
				}
			}
			amount, _ := rawAmount(a.Trade.CollateralRaw, a.Trade.CollateralDecimals)
			shares, _ := rawAmount(a.Trade.SharesRaw, a.Trade.SharesDecimals)
			fee, _ := rawAmount(a.Trade.FeeRaw, a.Trade.CollateralDecimals)
			add(fmt.Sprintf("活动 %d · Price %s/%s · Shares %s · Amount %s %s · Fee %s %s\n结算时间: %s", a.ID, a.Trade.PriceNumerator, a.Trade.PriceDenominator, shares, amount, a.Trade.CollateralSymbol, fee, a.Trade.CollateralSymbol, stamp(a.SettledAt)), member)
			link(base+fmt.Sprintf("/trader-sync/activities/%d", a.ID), member)
		}
	}
	// Pagination depends on the final denominator; repeat until that denominator
	// no longer changes the packing. Increasing its digit count only reduces space.
	total := 1
	for {
		parts, e := paginateSummary(lines, batchID, batchLink, total)
		if e != nil {
			return nil, e
		}
		if len(parts) == total {
			for i := range parts {
				parts[i].Total = total
				payload, e := delivery.EncodePayload(delivery.Payload{Format: "plain", Text: parts[i].Text})
				if e != nil {
					return nil, e
				}
				parts[i].PayloadDigest = delivery.PayloadDigest(payload)
			}
			return parts, nil
		}
		total = len(parts)
	}
}

func paginateSummary(lines []summaryLine, batchID, batchLink string, total int) ([]tm.RenderedPart, error) {
	var parts []tm.RenderedPart
	var body []string
	members := map[int64]bool{}
	used := 0
	header := func() string { return fmt.Sprintf("摘要 %s · %d/%d\n%s\n", batchID, len(parts)+1, total, batchLink) }
	flush := func() {
		if len(body) == 0 {
			return
		}
		ids := make([]int64, 0, len(members))
		for id := range members {
			ids = append(ids, id)
		}
		sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
		parts = append(parts, tm.RenderedPart{Index: len(parts) + 1, Text: header() + strings.Join(body, "\n"), ActivityIDs: ids})
		body = nil
		members = map[int64]bool{}
		used = 0
	}
	for _, line := range lines {
		// Split only oversized display text, never URLs. Continuation labels retain
		// every original code point and the full member relation on every fragment.
		rest := line.text
		continuation := false
		for rest != "" {
			capacity := 4096 - utf16Length(header())
			if capacity < 64 {
				return nil, fmt.Errorf("summary link/header exceeds limit")
			}
			prefix := ""
			if continuation {
				prefix = "延续: "
			}
			size := utf16Length(prefix + rest)
			if size <= capacity {
				if used+size+1 > capacity && len(body) > 0 {
					flush()
					continue // the next page number may require another digit
				}
				body = append(body, prefix+rest)
				used += size + 1
				for _, id := range line.ids {
					members[id] = true
				}
				break
			}
			if line.link {
				return nil, fmt.Errorf("complete URL exceeds Telegram limit")
			}
			flush()
			capacity = 4096 - utf16Length(header()) - utf16Length(prefix) - 1
			cut, units := 0, 0
			for offset, r := range rest {
				n := 1
				if r > 0xffff {
					n = 2
				}
				if units+n > capacity {
					break
				}
				units += n
				cut = offset + len(string(r))
			}
			if cut == 0 {
				return nil, fmt.Errorf("summary row cannot fit")
			}
			body = append(body, prefix+rest[:cut])
			for _, id := range line.ids {
				members[id] = true
			}
			flush()
			rest = rest[cut:]
			continuation = true
		}
	}
	flush()
	return parts, nil
}
