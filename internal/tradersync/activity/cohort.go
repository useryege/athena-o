package activity

import tm "github.com/useryege/athena/internal/tradersync/types"

// ClassifyFormation consumes the frozen original arrival and committed owner
// snapshot. Queue size or a later outcome can never change a sample's cohort.
func ClassifyFormation(mode string, e tm.FormationEvidence) tm.FormationEvidence {
	e.Rule = "arrival_and_owner_queue_v1"
	e.ArrivalIntervalNS = nil
	if mode != "ordinary" {
		e.Cohort = mode
		e.Reason = "notification_mode"
		return e
	}
	e.Cohort = "ordinary_default"
	p := e.Previous
	if p == nil {
		e.Reason = "first_observed_in_epoch"
		return e
	}
	unknown := func(reason string) tm.FormationEvidence {
		e.Cohort = "ordinary_unclassified"
		e.Reason = reason
		return e
	}
	c := e.Current
	if c.Epoch == 0 || c.Epoch != p.Epoch || p.Sequence >= c.Sequence || p.Sequence == 0 {
		return unknown("arrival_order_unverified")
	}
	if c.ElapsedNS == nil || p.ElapsedNS == nil || *c.ElapsedNS < 0 || *p.ElapsedNS < 0 || *c.ElapsedNS < *p.ElapsedNS {
		return unknown("arrival_clock_unverified")
	}
	delta := *c.ElapsedNS - *p.ElapsedNS
	e.ArrivalIntervalNS = &delta
	if p.Removed || p.Confirmation == "invalid" {
		e.Reason = "previous_invalid"
		return e
	}
	if !p.Formed || p.Confirmation != "confirmed" {
		return unknown("previous_not_confirmed_activity")
	}
	if p.Mode != "ordinary" {
		e.Reason = "previous_not_ordinary"
		return e
	}
	if c.ChatID == nil || p.ChatID == nil || c.BindingRevision == nil || p.BindingRevision == nil {
		return unknown("route_unverified")
	}
	if *c.ChatID <= 0 || *p.ChatID <= 0 || *c.BindingRevision <= 0 || *p.BindingRevision <= 0 {
		return unknown("route_unverified")
	}
	if *c.ChatID != *p.ChatID || *c.BindingRevision != *p.BindingRevision {
		e.Reason = "different_route"
		return e
	}
	if delta >= 1000000000 {
		e.Reason = "arrival_interval_not_dense"
		return e
	}
	competing := p.DeliveryEligible && ((p.DeliveryStatus == "pending" && p.DeliveryAttempts == 0 && !p.EverStarted) || (p.DeliveryStatus == "sending" && p.DeliveryAttempts == 1))
	if !competing {
		e.Reason = "previous_not_first_attempt_competition"
		return e
	}
	e.Cohort = "ordinary_burst"
	e.Reason = "dense_arrival_and_first_attempt_competition"
	return e
}
