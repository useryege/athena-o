package activity

import "strings"
import tm "github.com/useryege/athena/internal/tradersync/types"

func Classify(windowCount int64) string {
	if windowCount <= 10 {
		return "ordinary"
	}
	return "summary"
}
func Eligible(c tm.Eligibility, s tm.Subscription) bool {
	return s.DesiredState == "enabled" && s.Generation == c.Generation && c.BaselineSucceeded && !c.SettledAt.Before(c.EffectiveAt) && (c.EndedAt == nil || c.SettledAt.Before(*c.EndedAt))
}

// CloneMetadata publishes an immutable value even while later legs are resolved.
func CloneMetadata(m tm.TradeMetadata) tm.TradeMetadata {
	m.Legs = append([]tm.ComboLeg(nil), m.Legs...)
	return m
}
func MetadataKey(trade tm.Trade, hash string) string {
	key := trade.SourceVersion + ":" + trade.PositionID
	if strings.HasPrefix(trade.SourceVersion, "polygon137-combo-") {
		key += ":" + hash
	}
	return key
}
func CompleteMetadata(m tm.TradeMetadata, combo bool) bool {
	if m.Market.Availability != "available" {
		return false
	}
	if !combo {
		return true
	}
	if m.LegsEvidence.Availability != "available" {
		return false
	}
	for _, l := range m.Legs {
		if l.Market.Availability != "available" {
			return false
		}
	}
	return true
}
func conflict(e tm.Evidence) bool {
	return strings.Contains(e.ReasonCode, "conflict") || strings.Contains(e.ReasonCode, "mismatch") || strings.Contains(e.ReasonCode, "ambiguous")
}
func mergeMarket(old, next tm.MarketRef) tm.MarketRef {
	if conflict(old.Evidence) && !conflict(next.Evidence) {
		return old
	}
	if conflict(next.Evidence) {
		return next
	}
	if next.Availability != "available" {
		if old.Availability == "available" || conflict(old.Evidence) {
			return old
		}
		return next
	}
	if old.Availability == "available" && ((old.PositionID != "" && next.PositionID != old.PositionID) || (old.ID != "" && next.ID != "" && old.ID != next.ID) || (old.ConditionID != "" && next.ConditionID != "" && old.ConditionID != next.ConditionID) || (old.Outcome != "" && next.Outcome != "" && old.Outcome != next.Outcome) || (old.Source != "" && next.Source != "" && old.Source != next.Source)) {
		next.Availability = "unavailable"
		next.ReasonCode = "metadata_conflict"
	}
	return next
}

// MergeMetadata only accepts values already bound to the same evidence key.
// Transient failures preserve known fields; contradictory evidence stays visible.
func MergeMetadata(old, next tm.TradeMetadata) tm.TradeMetadata {
	result := CloneMetadata(next)
	if conflict(old.LegsEvidence) && !conflict(next.LegsEvidence) {
		result.LegsEvidence = old.LegsEvidence
	}
	result.Market = mergeMarket(old.Market, next.Market)
	if result.Relationship == "" {
		result.Relationship = old.Relationship
	} else if old.Relationship != "" && old.Relationship != result.Relationship {
		result.LegsEvidence.Availability = "unavailable"
		result.LegsEvidence.ReasonCode = "relationship_conflict"
		return result
	}
	if len(next.Legs) == 0 && !conflict(next.LegsEvidence) && len(old.Legs) > 0 {
		result.Legs = append([]tm.ComboLeg(nil), old.Legs...)
		result.LegsEvidence = old.LegsEvidence
		return result
	}
	if len(old.Legs) > 0 && len(next.Legs) > 0 {
		same := len(old.Legs) == len(next.Legs)
		if same {
			for i := range old.Legs {
				if old.Legs[i].PositionID != next.Legs[i].PositionID {
					same = false
					break
				}
			}
		}
		if !same {
			result.LegsEvidence.Availability = "unavailable"
			result.LegsEvidence.ReasonCode = "leg_position_conflict"
			for i := range result.Legs {
				result.Legs[i].Market.Availability = "unavailable"
				result.Legs[i].Market.ReasonCode = "leg_position_conflict"
			}
			return result
		}
		for i := range result.Legs {
			result.Legs[i].Market = mergeMarket(old.Legs[i].Market, result.Legs[i].Market)
		}
	}
	return result
}
