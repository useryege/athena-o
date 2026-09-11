package delivery

import (
	"testing"
	"time"
)

func TestNextState(t *testing.T) {
	for _, tt := range []struct {
		name     string
		outcome  Outcome
		attempts int
		eligible bool
		state    string
		wait     time.Duration
	}{
		{"unknown terminal", Outcome{Kind: "unknown"}, 1, true, "unknown", 0},
		{"revoked permanent failure cancelled", Outcome{Kind: "failed"}, 1, false, "cancelled", 0},
		{"revoked retry cancelled", Outcome{Kind: "retryable", RetryAfter: 7 * time.Second}, 1, false, "cancelled", 0},
		{"receipt survives revocation", Outcome{Kind: "sent"}, 1, false, "sent", 0},
		{"first retry", Outcome{Kind: "retryable"}, 1, true, "pending", time.Second},
		{"fourth retry", Outcome{Kind: "retryable"}, 4, true, "pending", 8 * time.Second},
		{"provider minimum", Outcome{Kind: "retryable", RetryAfter: 7 * time.Second}, 1, true, "pending", 7 * time.Second},
		{"attempt ceiling", Outcome{Kind: "retryable"}, 5, true, "failed", 0},
		{"known failure", Outcome{Kind: "failed"}, 1, true, "failed", 0},
	} {
		t.Run(tt.name, func(t *testing.T) {
			state, wait := NextState(tt.outcome, tt.attempts, tt.eligible)
			if state != tt.state || wait != tt.wait {
				t.Fatalf("got %s %s want %s %s", state, wait, tt.state, tt.wait)
			}
		})
	}
}
