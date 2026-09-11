package tradersync

import (
	"math"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	tm "github.com/useryege/athena/internal/tradersync/types"
)

func TestBaselineIsNextSecondAndNotBelowObservedHead(t *testing.T) {
	now := time.Unix(100, 800_000_000)
	h := &types.Header{Number: big.NewInt(10), Time: 100}
	at, err := ComputeBaseline(now, 10, h)
	if err != nil || !at.Equal(time.Unix(101, 0)) {
		t.Fatal(at, err)
	}
	if _, err = ComputeBaseline(now, 11, h); err == nil {
		t.Fatal("accepted older head")
	}
}
func TestBaselineClockBoundsAreInclusiveAndNeverOverflow(t *testing.T) {
	now := time.Unix(100, 0)
	for _, test := range []struct {
		name  string
		stamp uint64
		valid bool
		want  int64
	}{
		{"lower exact", 90, true, 101}, {"lower outside", 89, false, 0}, {"upper exact", 102, true, 103}, {"upper outside", 103, false, 0}, {"time overflow", math.MaxUint64, false, 0},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := ComputeBaseline(now, 10, &types.Header{Number: big.NewInt(10), Time: test.stamp})
			if (err == nil) != test.valid || (test.valid && !got.Equal(time.Unix(test.want, 0))) {
				t.Fatal(got, err)
			}
		})
	}
	for _, h := range []*types.Header{nil, {}, {Number: big.NewInt(-1)}, {Number: new(big.Int).Lsh(big.NewInt(1), 64)}} {
		if _, err := ComputeBaseline(now, 0, h); err == nil {
			t.Fatal("invalid header accepted", h)
		}
	}
	// Integer block seconds are compared to the actual database fractional clock.
	if _, err := ComputeBaseline(time.Unix(100, 1), 0, &types.Header{Number: big.NewInt(1), Time: 90}); err == nil {
		t.Fatal("rounded DB clock weakened lower bound")
	}
}
func TestEligibleUsesOriginalIntervalAndCurrentIntentGeneration(t *testing.T) {
	start, end := time.Unix(101, 0), time.Unix(102, 0)
	c := tm.Eligibility{Generation: 2, BaselineSucceeded: true, EffectiveAt: start, SettledAt: start, EndedAt: &end}
	s := tm.Subscription{DesiredState: "enabled", Generation: 2, Revision: 99, ObservationState: "interrupted"}
	if !Eligible(c, s) {
		t.Fatal("old successful interval rejected due to current observation")
	}
	c.SettledAt = end
	if Eligible(c, s) {
		t.Fatal("end must be exclusive")
	}
	c.SettledAt = start.Add(-time.Nanosecond)
	if Eligible(c, s) {
		t.Fatal("before baseline accepted")
	}
	c.SettledAt = start
	for _, state := range []string{"paused", "cancelled", "permission_disabled"} {
		s.DesiredState = state
		if Eligible(c, s) {
			t.Fatal(state)
		}
	}
	s.DesiredState = "enabled"
	s.Generation++
	if Eligible(c, s) {
		t.Fatal("old generation accepted")
	}
	s.Generation--
	c.BaselineSucceeded = false
	if Eligible(c, s) {
		t.Fatal("pending/failed accepted")
	}
	c.BaselineSucceeded = true
	emptyEnd := start.Add(-time.Second)
	c.EndedAt = &emptyEnd
	if Eligible(c, s) {
		t.Fatal("empty coverage accepted")
	}
	c.EndedAt = nil
	if !Eligible(c, s) {
		t.Fatal("open original interval rejected")
	}
}
