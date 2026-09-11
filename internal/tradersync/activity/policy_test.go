package activity

import (
	tm "github.com/useryege/athena/internal/tradersync/types"
	"testing"
)

func TestExplicitConflictSurvivesOlderAvailableAndTransientCompletion(t *testing.T) {
	available := tm.TradeMetadata{Market: tm.MarketRef{PositionID: "1", ID: "one", Evidence: tm.Evidence{Availability: "available"}}, LegsEvidence: tm.Evidence{Availability: "available"}, Legs: []tm.ComboLeg{{PositionID: "2", Market: tm.MarketRef{PositionID: "2", ID: "two", Evidence: tm.Evidence{Availability: "available"}}}}}
	conflict := CloneMetadata(available)
	conflict.Market.Availability = "unavailable"
	conflict.Market.ReasonCode = "condition_conflict"
	conflict.Legs[0].Market.Availability = "unavailable"
	conflict.Legs[0].Market.ReasonCode = "position_conflict"
	got := MergeMetadata(conflict, available)
	if got.Market.Availability != "unavailable" || got.Legs[0].Market.Availability != "unavailable" {
		t.Fatalf("late available concealed explicit conflict: %+v", got)
	}
	got = MergeMetadata(got, tm.TradeMetadata{})
	if got.Market.ReasonCode != "condition_conflict" || got.Legs[0].Market.ReasonCode != "position_conflict" {
		t.Fatal(got)
	}
}
