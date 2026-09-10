package activity

import (
	"fmt"
	tm "github.com/useryege/athena/internal/tradersync/types"
	"math/big"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"
)

func rawAmount(raw string, decimals uint8) (string, error) {
	n, ok := new(big.Int).SetString(raw, 10)
	if !ok || n.Sign() < 0 || n.String() != raw {
		return "", fmt.Errorf("invalid raw amount")
	}
	if decimals == 0 {
		return raw, nil
	}
	digits := int(decimals)
	s := raw
	if len(s) <= digits {
		s = strings.Repeat("0", digits+1-len(s)) + s
	}
	return s[:len(s)-digits] + "." + s[len(s)-digits:], nil
}
func displayTitle(s string) string {
	r := []rune(s)
	if len(r) > 180 {
		return string(r[:180]) + "…"
	}
	return s
}
func Render(a tm.Activity, siteURL string) (string, error) {
	u, err := url.Parse(siteURL)
	if err != nil || u.Host == "" || (u.Scheme != "https" && u.Scheme != "http") || u.RawQuery != "" || u.Fragment != "" {
		return "", fmt.Errorf("absolute activity site URL required")
	}
	if a.ID <= 0 || a.SettledAt.IsZero() || (a.Trade.Side != "BUY" && a.Trade.Side != "SELL") {
		return "", fmt.Errorf("incomplete activity")
	}
	amount, err := rawAmount(a.Trade.CollateralRaw, a.Trade.CollateralDecimals)
	if err != nil {
		return "", err
	}
	shares, err := rawAmount(a.Trade.SharesRaw, a.Trade.SharesDecimals)
	if err != nil {
		return "", err
	}
	fee, err := rawAmount(a.Trade.FeeRaw, a.Trade.CollateralDecimals)
	if err != nil {
		return "", err
	}
	numerator, nok := new(big.Int).SetString(a.Trade.PriceNumerator, 10)
	denominator, dok := new(big.Int).SetString(a.Trade.PriceDenominator, 10)
	if !nok || !dok || numerator.Sign() < 0 || denominator.Sign() <= 0 {
		return "", fmt.Errorf("invalid fill price")
	}
	var lines []string
	if a.NoteSnapshot != "" {
		lines = append(lines, a.NoteSnapshot)
	} else if a.TargetDisplaySnapshot.DisplayName.Availability == "available" && a.TargetDisplaySnapshot.DisplayName.Value != nil {
		lines = append(lines, displayTitle(*a.TargetDisplaySnapshot.DisplayName.Value))
	}
	if !a.TargetDisplaySnapshot.DisplayName.QueriedAt.IsZero() {
		lines = append(lines, "确认时资料（可能陈旧）: "+a.TargetDisplaySnapshot.DisplayName.QueriedAt.In(time.FixedZone("UTC+8", 8*3600)).Format("2006-01-02 15:04:05")+" UTC+8")
	}
	lines = append(lines, a.Trade.Wallet.Hex(), "TRADE · "+a.Trade.Side, "PositionID: "+a.Trade.PositionID)
	m := a.Metadata.Market
	if m.Title != "" {
		lines = append(lines, displayTitle(m.Title))
	}
	if m.Outcome != "" {
		lines = append(lines, "Outcome: "+m.Outcome)
	}
	if m.Availability != "available" {
		reason := m.ReasonCode
		if reason == "" {
			reason = "metadata_unavailable"
		}
		lines = append(lines, "Market unavailable: "+reason)
	}
	if a.Metadata.Relationship != "" {
		known := 0
		for _, leg := range a.Metadata.Legs {
			if leg.Market.Availability == "available" {
				known++
			}
		}
		lines = append(lines, "Combo: "+a.Metadata.Relationship, fmt.Sprintf("Leg metadata: %d/%d", known, len(a.Metadata.Legs)))
	}
	if a.Metadata.LegsEvidence.ReasonCode != "" && a.Metadata.LegsEvidence.ReasonCode != "not_combo" {
		lines = append(lines, "Legs: "+a.Metadata.LegsEvidence.ReasonCode)
	}
	lines = append(lines, "Price: "+a.Trade.PriceNumerator+"/"+a.Trade.PriceDenominator+" "+a.Trade.CollateralSymbol, "Shares: "+shares, "Amount: "+amount+" "+a.Trade.CollateralSymbol, "Fee: "+fee+" "+a.Trade.CollateralSymbol, "结算时间: "+a.SettledAt.In(time.FixedZone("UTC+8", 8*3600)).Format("2006-01-02 15:04:05")+" UTC+8")
	if m.URL != "" {
		lines = append(lines, m.URL)
	}
	lines = append(lines, strings.TrimRight(siteURL, "/")+fmt.Sprintf("/trader-sync/activities/%d", a.ID))
	text := strings.Join(lines, "\n")
	if utf8.RuneCountInString(text) > 4096 {
		return "", fmt.Errorf("activity notification exceeds Telegram limit")
	}
	return text, nil
}
