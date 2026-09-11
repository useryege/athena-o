package activity

import (
	tm "github.com/useryege/athena/internal/tradersync/types"
	"testing"
	"time"
)

func TestFinalityFirstSequenceAndRetryWait(t *testing.T) {
	start := time.Now()
	first := tm.FinalityRoundObservation{ClockID: "one", StartedAt: start, ReturnedAt: start.Add(time.Millisecond), StartedNS: 100, ReturnedNS: 200, Reliable: true, SourceAfterCutoff: true}
	waiting := MergeFinalityObservation(tm.FinalityTiming{}, first)
	if waiting.State != "waiting" || waiting.FirstStartedNS == nil || *waiting.FirstStartedNS != 100 || waiting.FirstRoundNS == nil || *waiting.FirstRoundNS != 100 {
		t.Fatalf("first actual attempt missing: %+v", waiting)
	}
	next := first
	next.StartedNS = 800
	next.ReturnedNS = 820
	next.Confirmed = true
	next.StartedAt = start.Add(time.Second)
	next.ReturnedAt = next.StartedAt.Add(time.Millisecond)
	completed := MergeFinalityObservation(waiting, next)
	if completed.State != "completed" || *completed.FirstStartedNS != 100 || *completed.FirstRoundNS != 100 || completed.FirstConfirmedNS == nil || *completed.FirstConfirmedNS != 820 {
		t.Fatalf("retry replaced first interval: %+v", completed)
	}
	if wait := *completed.FirstConfirmedNS - *completed.FirstStartedNS; wait != 720 {
		t.Fatal(wait)
	}
	if after := *completed.FirstConfirmedNS - *completed.FirstStartedNS - *completed.FirstRoundNS; after != 620 {
		t.Fatal(after)
	}
	next.ReturnedNS = 1000
	if final := MergeFinalityObservation(completed, next); *final.FirstConfirmedNS != 820 {
		t.Fatal("reconfirmed changed first completion")
	}
	first.Confirmed = true
	immediate := MergeFinalityObservation(tm.FinalityTiming{}, first)
	if immediate.State != "completed" || *immediate.FirstConfirmedNS-*immediate.FirstStartedNS-*immediate.FirstRoundNS != 0 {
		t.Fatalf("first success must retain legal zero: %+v", immediate)
	}
}
func TestFinalityMissingAndCrossClockNeverAcquireLaterFirst(t *testing.T) {
	o := tm.FinalityRoundObservation{ClockID: "one", StartedAt: time.Now(), ReturnedAt: time.Now(), StartedNS: 100, ReturnedNS: 200, Reliable: true, SourceAfterCutoff: true}
	waiting := MergeFinalityObservation(tm.FinalityTiming{}, o)
	for _, tt := range []struct {
		name     string
		previous tm.FinalityTiming
		edit     func(*tm.FinalityRoundObservation)
		reason   string
	}{
		{"old source", tm.FinalityTiming{}, func(o *tm.FinalityRoundObservation) { o.SourceAfterCutoff = false }, "first_observation_missing"},
		{"cross clock", waiting, func(o *tm.FinalityRoundObservation) { o.ClockID = "two" }, "clock_changed"},
		{"lost confirmed", waiting, func(o *tm.FinalityRoundObservation) { o.Reliable = false }, "observation_gap"},
		{"invalid mono", waiting, func(o *tm.FinalityRoundObservation) { o.ReturnedNS = 99 }, "clock_invalid"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			in := o
			tt.edit(&in)
			in.Confirmed = true
			unavailable := MergeFinalityObservation(tt.previous, in)
			if unavailable.State != "unavailable" || unavailable.Reason != tt.reason {
				t.Fatalf("%+v", unavailable)
			}
			in = o
			in.Confirmed = true
			in.StartedNS = 300
			in.ReturnedNS = 400
			if v := MergeFinalityObservation(unavailable, in); v.State != "unavailable" || v.FirstConfirmedNS != nil {
				t.Fatalf("later confirmation invented first: %+v", v)
			}
		})
	}
}
