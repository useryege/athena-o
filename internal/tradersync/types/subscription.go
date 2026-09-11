package types

import (
	"github.com/ethereum/go-ethereum/common"
	"time"
)

type TargetDisplay struct{ DisplayName, Avatar, ProfileURL Scalar }

type Subscription struct {
	TargetDisplay                          TargetDisplay
	ID, OwnerID                            string
	Wallet                                 common.Address
	DesiredState, ObservationState, Reason string
	Revision, Generation, NoteRevision     uint64
	EffectiveAt, EndedAt                   *time.Time
	CreatedAt, UpdatedAt                   time.Time
	Note, BindingStatus, QueueNotice       string
}
type CreateInput struct {
	Token, RequestID string
	Note             *string
}
type ChangeInput struct {
	SubscriptionID, RequestID string
	ExpectedRevision          uint64
}
type NoteInput struct {
	Wallet           common.Address
	RequestID, Note  string
	ExpectedRevision uint64
}
