package tradersync

import (
	"testing"
	"time"
)

func TestUnknownPublicTimeIsNotReceivedTime(t *testing.T) {
	r := EvaluateTiming(TimingSample{ReceivedAt: time.Unix(100, 0), RecordedAt: time.Unix(102, 0)})
	if r.PublicAssessable {
		t.Fatal("fabricated public timestamp")
	}
	if r.ReceivedToRecorded != 2*time.Second {
		t.Fatalf("received→recorded=%s want 2s", r.ReceivedToRecorded)
	}
	if r.Reason != "public_time_or_clock_unverified" {
		t.Fatal(r)
	}
}
func TestTimingBoundsAndClockFailures(t *testing.T) {
	at := func(s int64) *time.Time { v := time.Unix(s, 0); return &v }
	base := TimingSample{PublicEarliest: at(90), PublicLatest: at(92), ReceivedAt: *at(100), RecordedAt: *at(102), ClockSource: "independent_observer", ClockUncertainty: time.Second}
	r := EvaluateTiming(base)
	if !r.PublicAssessable || r.Lower != 9*time.Second || r.Upper != 13*time.Second {
		t.Fatalf("bounds: %+v", r)
	}
	for _, tt := range []struct {
		name   string
		modify func(*TimingSample)
		reason string
	}{
		{"unknown clock", func(s *TimingSample) { s.ClockSource = "" }, "public_time_or_clock_unverified"},
		{"untrusted clock", func(s *TimingSample) { s.ClockSource = "untrusted" }, "public_time_or_clock_unverified"},
		{"reversed public interval", func(s *TimingSample) { s.PublicEarliest = at(94) }, "public_interval_invalid"},
		{"negative uncertainty", func(s *TimingSample) { s.ClockUncertainty = -time.Second }, "clock_uncertainty_invalid"},
		{"negative local segment", func(s *TimingSample) { s.RecordedAt = *at(99) }, "received_recorded_clock_anomaly"},
		{"missing recorded", func(s *TimingSample) { s.RecordedAt = time.Time{} }, "local_time_missing"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			s := base
			tt.modify(&s)
			r := EvaluateTiming(s)
			if r.PublicAssessable || r.Reason != tt.reason {
				t.Fatalf("%+v", r)
			}
		})
	}
}
