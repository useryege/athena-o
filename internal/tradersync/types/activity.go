package types

import "time"
import et "github.com/ethereum/go-ethereum/core/types"
import "github.com/ethereum/go-ethereum/common"

type Activity struct {
	TargetDisplaySnapshot                              TargetDisplay
	ID                                                 int64
	OwnerID, SubscriptionID                            string
	SourceID                                           int64
	Trade                                              Trade
	Metadata                                           TradeMetadata
	NoteSnapshot, NotificationMode, NotificationReason string
	SettledAt, ReceivedAt, RecordedAt                  time.Time
	Generation                                         uint64
}
type Projection struct {
	Candidate    Candidate
	Trade        Trade
	Confirmation CanonicalEvidence
	Metadata     TradeMetadata
}

// ProjectionSource retains original receipt intake plus independently mutable evidence.
type ProjectionSource struct {
	ID               int64
	Raw              et.Log
	Wallet           common.Address
	Candidates       []Candidate
	Trade            *Trade
	Confirmation     CanonicalEvidence
	MetadataComplete bool
}
