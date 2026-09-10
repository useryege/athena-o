// Package types contains the shared Trader Sync contracts, independent of service and storage.
package types

import (
	"time"

	"github.com/ethereum/go-ethereum/common"
)

type Identity struct {
	Wallet                             common.Address
	ProfileURL, DisplayName, AvatarURL string
	Digest                             [32]byte
}
type Evidence struct {
	Availability, ReasonCode, Source string
	QueriedAt                        time.Time
}
type Scalar struct {
	Evidence
	Value *string
}
type PnLPoint struct {
	T int64  `json:"t"`
	P string `json:"p"`
}
type Curve struct {
	Evidence
	Points []PnLPoint
}
type PnLView struct {
	Amount             Scalar
	Curve              Curve
	Interval, Fidelity string
	Reference          *time.Time
	Timezone           string
}
type ConfirmationCard struct {
	Identity                                                                        Identity
	Avatar, DisplayName, Verified, JoinedAt, PositionValue, LargestWin, Predictions Scalar
	PnL                                                                             map[string]PnLView
	DefaultPeriod, UsageNotice                                                      string
}
type TargetNote struct {
	Wallet   common.Address
	Note     string
	Revision uint64
}
type ExistingSubscription struct {
	ID, Status string
	Revision   uint64
}
type Quota struct{ Used, Limit int32 }
type ResolutionContext struct {
	SavedNote *TargetNote
	Existing  *ExistingSubscription
	Quota     Quota
}
type ResolvedTarget struct {
	Card      ConfirmationCard
	Context   ResolutionContext
	Token     string
	ExpiresAt time.Time
}
