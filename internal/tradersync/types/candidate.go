package types

import (
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"time"
)

type Candidate struct {
	SourceID                int64
	SubscriptionID, OwnerID string
	Generation              uint64
	AttemptID               string
	ReceivedAt              time.Time
}
type Eligibility struct {
	Generation             uint64
	BaselineSucceeded      bool
	SettledAt, EffectiveAt time.Time
	EndedAt                *time.Time
}

// ReceivedLog records read-loop observation before any persistence queue wait.
type ReceivedLog struct {
	Raw               ethtypes.Log
	ReceivedAt        time.Time
	Sequence          uint64
	ReceivedElapsedNS *int64 `json:"receivedElapsedNs,omitempty"`
}
type WalletObservation struct{ High, Sequence uint64 }
