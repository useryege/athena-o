package tradersync

import (
	"fmt"
	"math"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	tm "github.com/useryege/athena/internal/tradersync/types"
)

func ComputeBaseline(now time.Time, registeredHigh uint64, h *types.Header) (time.Time, error) {
	if h == nil || h.Number == nil || !h.Number.IsUint64() || h.Number.Uint64() < registeredHigh || h.Time > math.MaxInt64 {
		return time.Time{}, fmt.Errorf("head below registered observation or invalid")
	}
	settled := time.Unix(int64(h.Time), 0).UTC()
	if settled.Before(now.Add(-10*time.Second)) || settled.After(now.Add(2*time.Second)) {
		return time.Time{}, fmt.Errorf("head clock outside bounds")
	}
	at := now.UTC().Truncate(time.Second).Add(time.Second)
	if candidate := settled.Add(time.Second); candidate.After(at) {
		at = candidate
	}
	return at, nil
}

// Eligible keeps the original successful interval meaningful after connection
// loss. Only explicit user intent/generation changes revoke unprojected work.
func Eligible(c tm.Eligibility, s tm.Subscription) bool {
	return s.DesiredState == "enabled" && s.Generation == c.Generation && c.BaselineSucceeded && !c.SettledAt.Before(c.EffectiveAt) && (c.EndedAt == nil || c.SettledAt.Before(*c.EndedAt))
}
