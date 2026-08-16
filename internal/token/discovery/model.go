package discovery

import (
	"time"

	"github.com/useryege/athena/internal/token/shared"
)

type ChainProcessingStatus string

type ChainBlockProcessingAttemptStatus string

type ChainBlockProcessingStage string

const (
	ChainProcessingStatusRunning ChainProcessingStatus = "running"
	ChainProcessingStatusStopped ChainProcessingStatus = "stopped"

	ChainBlockProcessingAttemptStatusRunning     ChainBlockProcessingAttemptStatus = "running"
	ChainBlockProcessingAttemptStatusSucceeded   ChainBlockProcessingAttemptStatus = "succeeded"
	ChainBlockProcessingAttemptStatusFailed      ChainBlockProcessingAttemptStatus = "failed"
	ChainBlockProcessingAttemptStatusCancelled   ChainBlockProcessingAttemptStatus = "cancelled"
	ChainBlockProcessingAttemptStatusInterrupted ChainBlockProcessingAttemptStatus = "interrupted"

	ChainBlockProcessingStageCheckpointRead      ChainBlockProcessingStage = "checkpoint_read"
	ChainBlockProcessingStageCandidateDiscovery  ChainBlockProcessingStage = "candidate_discovery"
	ChainBlockProcessingStageCandidateValidation ChainBlockProcessingStage = "candidate_validation"
	ChainBlockProcessingStagePersistence         ChainBlockProcessingStage = "persistence"
)

type ChainProcessingCheckpoint struct {
	ChainID           int64
	ChainName         string
	Enabled           bool
	CursorBlockNumber uint64
	Status            ChainProcessingStatus
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type ChainBlockProcessingAttempt struct {
	ID                        int64
	ChainID                   int64
	BlockNumber               uint64
	AttemptNumber             int32
	BlockTime                 uint64
	Status                    ChainBlockProcessingAttemptStatus
	TerminalStage             ChainBlockProcessingStage
	ErrorMessage              string
	CheckpointReadDurationUS  int64
	DiscoveryDurationUS       int64
	ValidationDurationUS      int64
	PersistenceDurationUS     int64
	TotalDurationUS           int64
	CandidateCount            int32
	ValidatedCount            int32
	RejectedCount             int32
	ExpiredResearchStateCount int64
	TimingComplete            bool
	StartedAt                 time.Time
	CompletedAt               time.Time
	CreatedAt                 time.Time
	UpdatedAt                 time.Time
}

type ChainBlockProcessingAttemptCompletion struct {
	AttemptID                 int64
	BlockTime                 uint64
	Status                    ChainBlockProcessingAttemptStatus
	TerminalStage             ChainBlockProcessingStage
	ErrorMessage              string
	CheckpointReadDuration    *time.Duration
	DiscoveryDuration         *time.Duration
	ValidationDuration        *time.Duration
	PersistenceDuration       *time.Duration
	CandidateCount            *int32
	ValidatedCount            *int32
	RejectedCount             *int32
	ExpiredResearchStateCount *int64
	TimingComplete            bool
}

type ChainBlockProcessingFilter struct {
	ChainID       int64
	BlockNumber   uint64
	WindowSeconds int64
	Status        ChainBlockProcessingAttemptStatus
}

type ChainBlockProcessingAttemptPage struct {
	Items    []ChainBlockProcessingAttempt
	Total    int64
	Page     int32
	PageSize int32
}

type ChainBlockProcessingSummary struct {
	ChainID                         int64
	RangeStartBlockTime             uint64
	RangeEndBlockTime               uint64
	AttemptCount                    int64
	RunningCount                    int64
	SucceededCount                  int64
	FailedCount                     int64
	CancelledCount                  int64
	InterruptedCount                int64
	IncompleteSucceededCount        int64
	MeasuredSucceededCount          int64
	FailureRateBPS                  int64
	AverageDurationUS               int64
	AverageCheckpointReadDurationUS int64
	AverageDiscoveryDurationUS      int64
	AverageValidationDurationUS     int64
	AveragePersistenceDurationUS    int64
	FastestBlockNumber              uint64
	FastestDurationUS               int64
	SlowestBlockNumber              uint64
	SlowestDurationUS               int64
}

type Chain struct {
	ID        int64
	Name      string
	Enabled   bool
	CreatedAt time.Time
}

type BlockHeader struct {
	Number    uint64
	Timestamp uint64
}

type ProjectCandidateBlock struct {
	Header     BlockHeader
	Candidates []ProjectCandidate
}

type ProjectCandidate struct {
	ChainID         int64
	Contract        shared.Address
	TxSender        shared.Address
	TxHash          shared.Hash
	TxIndex         uint64
	DeploymentNonce uint64
	BlockNumber     uint64
	BlockTime       uint64
}

type NodeStatus struct {
	ChainID              int64
	ChainName            string
	Endpoint             string
	Available            bool
	Latency              time.Duration
	ReportedChainID      int64
	LatestBlockNumber    uint64
	CheckedAt            time.Time
	Error                string
	ReferenceBlockNumber uint64
	BlockLag             uint64
	LatestBlockTime      time.Time
	Syncing              bool
}
