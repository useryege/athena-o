package activity

import tm "github.com/useryege/athena/internal/tradersync/types"

// MergeFinalityObservation preserves the first continuous provable sequence.
// A completed interval is historical evidence; unavailable sequences never
// acquire a later first endpoint. UTC is only retained provenance.
func MergeFinalityObservation(previous tm.FinalityTiming, o tm.FinalityRoundObservation) tm.FinalityTiming {
	if previous.State == "completed" || previous.State == "unavailable" {
		return previous
	}
	fail := func(reason string) tm.FinalityTiming {
		previous.State = "unavailable"
		previous.Reason = reason
		return previous
	}
	if !o.Reliable {
		return fail("observation_gap")
	}
	if o.ClockID == "" || o.StartedNS < 0 || o.ReturnedNS < o.StartedNS || o.StartedAt.IsZero() || o.ReturnedAt.IsZero() {
		return fail("clock_invalid")
	}
	if previous.ClockID != "" && previous.ClockID != o.ClockID {
		return fail("clock_changed")
	}
	if previous.State == "" {
		if !o.SourceAfterCutoff {
			return fail("first_observation_missing")
		}
		start := o.StartedAt.UTC()
		elapsed := o.StartedNS
		round := o.ReturnedNS - o.StartedNS
		previous = tm.FinalityTiming{State: "waiting", ClockID: o.ClockID, FirstStartedAt: &start, FirstStartedNS: &elapsed, FirstRoundNS: &round}
	}
	if previous.State != "waiting" || previous.FirstStartedNS == nil || previous.FirstRoundNS == nil || o.StartedNS < *previous.FirstStartedNS {
		return fail("clock_invalid")
	}
	if o.Confirmed {
		end := o.ReturnedAt.UTC()
		elapsed := o.ReturnedNS
		previous.State = "completed"
		previous.FirstConfirmedAt = &end
		previous.FirstConfirmedNS = &elapsed
	}
	return previous
}
