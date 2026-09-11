package types

import (
	"github.com/ethereum/go-ethereum/common"
	"time"
)

// Read models contain persisted facts, never transport cursors or frozen messages.
// Administrator projections are deliberately separate from member resources.
type ActivityFilter struct {
	SubscriptionID string
	BatchID        int64
	From, To       *time.Time
}
type ActivityReadInput struct {
	Filter                        ActivityFilter
	Limit                         int32
	Snapshot, After, Lower, Upper *int64
	Refresh, Empty                bool
	ActivityID                    int64
}
type ActivityReadPage struct {
	Activities []ActivityDetails
	Snapshot   int64
	HasNewer   bool
	HasOlder   bool
	AsOf       time.Time
}
type StatusCounts struct{ Total, Pending, Sending, Sent, Failed, Unknown, Cancelled int64 }
type AttemptDetails struct {
	Index               int64
	AuthorizedAt        time.Time
	StartedAt, ResultAt *time.Time
	Status, Reason      string
}
type DeliveryDetails struct {
	ID                                int64
	Status, Reason                    string
	AuthorizedAt, StartedAt, ResultAt *time.Time
	MessageID                         *string
	AttemptCount                      int64
	LatestAttempt                     *AttemptDetails
}
type SummaryDetails struct {
	Phase, Reason                      string
	BatchID                            *int64
	RelatedPartCounts, BatchPartCounts StatusCounts
	OldestAt                           time.Time
	FirstStartedAt                     *time.Time
}
type SourceLocation struct {
	ChainID, BlockNumber, LogIndex int64
	Exchange                       common.Address
	TransactionHash, BlockHash     common.Hash
}
type FinalityAnomaly struct {
	Reason               string
	DetectedAt           time.Time
	PublishedBlockHash   common.Hash
	ConflictingBlockHash *common.Hash
}
type ActivityDetails struct {
	Activity
	SourceLocation  SourceLocation
	FinalityAnomaly *FinalityAnomaly
	Delivery        *DeliveryDetails
	SummaryProgress *SummaryDetails
}
type IntervalDetails struct {
	ID                string
	EffectiveAt       time.Time
	EndedAt           *time.Time
	Generation, Epoch uint64
}
type InterruptionDetails struct {
	ID                      string
	Start, End, RecoveredAt *time.Time
	Reason, Uncertainty     string
	PossibleMissing         bool
}
type ObservationDetails struct {
	State, Reason      string
	LastReliableAt     *time.Time
	LatestInterruption *InterruptionDetails
	InterruptionCount  int64
}
type SubscriptionDetails struct {
	Subscription
	PausedAt, CancelledAt, PermissionDisabledAt *time.Time
	CurrentInterval                             *IntervalDetails
	Observation                                 ObservationDetails
	QueueCounts                                 StatusCounts
}
type ReadPage struct {
	Limit      int32
	AfterTime  *time.Time
	AfterID    string
	AfterIndex int32
}
type SubscriptionFilter struct{ View, State, ID string }
type SubscriptionReadPage struct {
	Subscriptions []SubscriptionDetails
	Quota         Quota
	AsOf          time.Time
}
type HistoryEntry struct {
	ID, Kind     string
	SortAt       time.Time
	Interval     *IntervalDetails
	Interruption *InterruptionDetails
}
type HistoryReadPage struct {
	Entries []HistoryEntry
	AsOf    time.Time
}
type TargetCount struct {
	Wallet common.Address
	Count  int64
}
type SummaryBatchDetails struct {
	ID                                                         int64
	OldestAt, SettledFrom, SettledTo, RecordedFrom, RecordedTo time.Time
	FirstStartedAt                                             *time.Time
	ActivityCount                                              int64
	TargetCounts                                               []TargetCount
	PartCounts                                                 StatusCounts
	AsOf                                                       time.Time
}
type SummaryPartDetails struct {
	ID                      int64
	Index, Total            int32
	Delivery                DeliveryDetails
	AssociatedActivityCount int64
}
type SummaryPartReadPage struct {
	Parts []SummaryPartDetails
	AsOf  time.Time
}
type AdminSubscriptionFilter struct {
	AccountID, State, ID string
	Wallet               *common.Address
	IncludeCancelled     bool
}
type SubscriptionSummary struct {
	SubscriptionID, AccountID, Username, Email  string
	Wallet                                      common.Address
	Status                                      string
	CreatedAt, UpdatedAt                        time.Time
	PausedAt, CancelledAt, PermissionDisabledAt *time.Time
	Observation                                 ObservationDetails
	ActivityCount                               int64
	AssociatedDeliveryCounts                    StatusCounts
	AsOf                                        time.Time
}
type AdminSubscriptionReadPage struct {
	Summaries []SubscriptionSummary
	AsOf      time.Time
}
type RuntimeMetric struct {
	Name, Value, Unit, Kind string
	WindowStart, WindowEnd  *time.Time
	ServiceEpoch            *string
}
type RuntimeStatus struct {
	CollectorConnected             bool
	CollectorEpoch, FilterRevision string
	Metrics                        []RuntimeMetric
	AsOf                           time.Time
}

// Checkpoint evidence retains process-monotonic timestamps until the exact
// observed UTC value is written. It is never reconstructed from raw log time.
type ObservationCheckpoint struct {
	Token, Epoch, FilterRevision uint64
	At, ExpiresAt, CoveredAt     time.Time
	Wallets                      []common.Address
}
type CheckpointInterval struct {
	ID, OwnerID, SubscriptionID       string
	Wallet                            common.Address
	Generation, Epoch, FilterRevision uint64
	EffectiveAt                       time.Time
	LastReliableAt                    *time.Time
}
