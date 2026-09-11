package activity

import (
	tm "github.com/useryege/athena/internal/tradersync/types"
	"testing"
	"time"
)

func TestFormationCohortUsesOriginalPredecessorAndPhysicalClock(t *testing.T) {
	ptr := func(n int64) *int64 { return &n }
	base := tm.FormationEvidence{Current: tm.ArrivalEvidence{SourceID: 2, Epoch: 1, Sequence: 2, ElapsedNS: ptr(900000000), ChatID: ptr(123), BindingRevision: ptr(1)}, Previous: &tm.ArrivalEvidence{SourceID: 1, Epoch: 1, Sequence: 1, ElapsedNS: ptr(1), Formed: true, Mode: "ordinary", Confirmation: "confirmed", ChatID: ptr(123), BindingRevision: ptr(1), DeliveryStatus: "pending", DeliveryEligible: true}}
	for _, tt := range []struct {
		name   string
		change func(*tm.FormationEvidence)
		want   string
	}{
		{"dense first pending", func(*tm.FormationEvidence) {}, "ordinary_burst"},
		{"zero gap", func(e *tm.FormationEvidence) { e.Current.ElapsedNS = ptr(1) }, "ordinary_burst"},
		{"one second", func(e *tm.FormationEvidence) { e.Current.ElapsedNS = ptr(1000000001) }, "ordinary_default"},
		{"missing clock", func(e *tm.FormationEvidence) { e.Current.ElapsedNS = nil }, "ordinary_unclassified"},
		{"negative clock", func(e *tm.FormationEvidence) { e.Current.ElapsedNS = ptr(-1) }, "ordinary_unclassified"},
		{"reversed clock", func(e *tm.FormationEvidence) { e.Current.ElapsedNS = ptr(0) }, "ordinary_unclassified"},
		{"epoch changed", func(e *tm.FormationEvidence) { e.Current.Epoch = 2 }, "ordinary_unclassified"},
		{"sequence reversed", func(e *tm.FormationEvidence) { e.Current.Sequence = 1 }, "ordinary_unclassified"},
		{"latest not formed", func(e *tm.FormationEvidence) { e.Previous.Formed = false }, "ordinary_unclassified"},
		{"previous unverified", func(e *tm.FormationEvidence) { e.Previous.Confirmation = "unverified" }, "ordinary_unclassified"},
		{"first observed", func(e *tm.FormationEvidence) { e.Previous = nil }, "ordinary_default"},
		{"previous summary", func(e *tm.FormationEvidence) { e.Previous.Mode = "summary" }, "ordinary_default"},
		{"different chat", func(e *tm.FormationEvidence) { e.Previous.ChatID = ptr(234) }, "ordinary_default"},
		{"different revision", func(e *tm.FormationEvidence) { e.Previous.BindingRevision = ptr(2) }, "ordinary_default"},
		{"missing route", func(e *tm.FormationEvidence) { e.Previous.ChatID = nil }, "ordinary_unclassified"},
		{"first sending", func(e *tm.FormationEvidence) { e.Previous.DeliveryStatus = "sending"; e.Previous.DeliveryAttempts = 1 }, "ordinary_burst"},
		{"retry pending", func(e *tm.FormationEvidence) { e.Previous.DeliveryAttempts = 1 }, "ordinary_default"},
		{"retry sending", func(e *tm.FormationEvidence) { e.Previous.DeliveryStatus = "sending"; e.Previous.DeliveryAttempts = 2 }, "ordinary_default"},
		{"revoked", func(e *tm.FormationEvidence) { e.Previous.DeliveryEligible = false }, "ordinary_default"},
		{"sent", func(e *tm.FormationEvidence) { e.Previous.DeliveryStatus = "sent" }, "ordinary_default"},
		{"removed", func(e *tm.FormationEvidence) { e.Previous.Removed = true }, "ordinary_default"},
		{"other queue does not classify", func(e *tm.FormationEvidence) {
			e.Previous.DeliveryStatus = "sent"
			e.Queue.PartPending = 100
			e.Queue.ReplyPending = 100
		}, "ordinary_default"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			e := base
			p := *base.Previous
			e.Previous = &p
			tt.change(&e)
			got := ClassifyFormation("ordinary", e)
			if got.Cohort != tt.want || got.Rule != "arrival_and_owner_queue_v1" {
				t.Fatalf("cohort=%s reason=%s want %s", got.Cohort, got.Reason, tt.want)
			}
		})
	}
	for _, mode := range []string{"summary", "in_app_only"} {
		if got := ClassifyFormation(mode, base); got.Cohort != mode {
			t.Fatal(got)
		}
	}
	// Receipt UTC reversal cannot alter the physically measured interval.
	e := base
	e.Current.ReceivedAt = time.Unix(1, 0)
	e.Previous.ReceivedAt = time.Unix(2, 0)
	if got := ClassifyFormation("ordinary", e); got.Cohort != "ordinary_burst" {
		t.Fatal(got)
	}
}
