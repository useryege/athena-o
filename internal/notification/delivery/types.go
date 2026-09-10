// Package delivery defines immutable send permits and provider outcomes.
package delivery

import (
	"github.com/google/uuid"
	"time"
)

type WorkRef struct {
	Kind string
	ID   int64
}
type Permit struct {
	Payload           []byte
	ChatID            int64
	Group             bool
	Work              WorkRef
	AttemptID         uuid.UUID
	OwnerID           string
	SenderIncarnation uuid.UUID
	PayloadDigest     []byte
	AuthorizedAt      time.Time
}
type Outcome struct {
	Kind       string
	MessageID  string
	RetryAfter time.Duration
	Code       string
}

// NextState only retries a definite rejection while the original delivery remains eligible.
func NextState(o Outcome, attempts int, eligible bool) (string, time.Duration) {
	switch o.Kind {
	case "sent", "unknown":
		return o.Kind, 0
	case "failed":
		if !eligible {
			return "cancelled", 0
		}
		return "failed", 0
	case "retryable":
		if !eligible {
			return "cancelled", 0
		}
		if attempts >= 5 {
			return "failed", 0
		}
		wait := time.Second * time.Duration(1<<max(0, min(attempts-1, 3)))
		return "pending", max(wait, o.RetryAfter)
	default:
		return "unknown", 0
	}
}

// Candidate is a visible pending item; future deadline work remains visible to reserve capacity.
type Candidate struct {
	Ref       WorkRef
	OwnerID   string
	ChatID    int64
	Group     bool
	NotBefore time.Time
	Deadline  *time.Time
}
