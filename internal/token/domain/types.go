package domain

import (
	"encoding/json"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/google/uuid"
)

const (
	ObservationSchemaVersionV1 = int32(1)
	ReportSchemaVersionV1      = int32(1)
)

type ChainIngestStatus string

const (
	ChainIngestStatusRunning ChainIngestStatus = "running"
	ChainIngestStatusStopped ChainIngestStatus = "stopped"
)

type ProjectCandidateStatus string

const (
	ProjectCandidateStatusPending   ProjectCandidateStatus = "pending"
	ProjectCandidateStatusValidated ProjectCandidateStatus = "validated"
	ProjectCandidateStatusRejected  ProjectCandidateStatus = "rejected"
)

type ProjectResearchStatus string

const (
	ProjectResearchStatusResearching ProjectResearchStatus = "researching"
	ProjectResearchStatusSelected    ProjectResearchStatus = "selected"
	ProjectResearchStatusRejected    ProjectResearchStatus = "rejected"
	ProjectResearchStatusExpired     ProjectResearchStatus = "expired"
)

type DataCollectionType string

const (
	DataCollectionTypeAve                DataCollectionType = "ave"
	DataCollectionTypeChainState         DataCollectionType = "chain_state"
	DataCollectionTypeWalletAssetState   DataCollectionType = "wallet_asset_state"
	DataCollectionTypeSimulationResult   DataCollectionType = "simulation_result"
	DataCollectionTypeContractCodeSource DataCollectionType = "contract_code_source"
)

type DataCollectionScheduleStatus string

const (
	DataCollectionScheduleStatusActive    DataCollectionScheduleStatus = "active"
	DataCollectionScheduleStatusCompleted DataCollectionScheduleStatus = "completed"
	DataCollectionScheduleStatusPaused    DataCollectionScheduleStatus = "paused"
)

type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"
	TaskStatusRunning   TaskStatus = "running"
	TaskStatusSucceeded TaskStatus = "succeeded"
	TaskStatusFailed    TaskStatus = "failed"
)

type SelectionOutcome string

const (
	SelectionOutcomeSelected SelectionOutcome = "selected"
	SelectionOutcomeRejected SelectionOutcome = "rejected"
	SelectionOutcomeDeferred SelectionOutcome = "deferred"
)

type RelatedWalletRole string

const (
	RelatedWalletRoleCreator          RelatedWalletRole = "creator"
	RelatedWalletRoleInitialRecipient RelatedWalletRole = "initial_recipient"
)

type ChainIngestCheckpoint struct {
	ChainID           int64
	ChainName         string
	Enabled           bool
	CursorBlockNumber uint64
	Status            ChainIngestStatus
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type Chain struct {
	ID        int64
	Name      string
	Enabled   bool
	CreatedAt time.Time
}

type ProjectCandidate struct {
	ID                       int64
	ChainID                  int64
	Contract                 common.Address
	TxSender                 common.Address
	TxHash                   common.Hash
	TxIndex                  uint64
	BlockNumber              uint64
	BlockTime                uint64
	Status                   ProjectCandidateStatus
	ValidationLockToken      uuid.UUID
	ValidationLockedAt       time.Time
	ValidationLeaseExpiresAt time.Time
	CreatedAt                time.Time
}

type ContractCode struct {
	CodeHash            common.Hash
	SourceCode          string
	SourceCodeFetchedAt time.Time
	DeploymentCount     int64
	CreatedAt           time.Time
}

type Project struct {
	ID          int64
	ChainID     int64
	Contract    common.Address
	TxSender    common.Address
	TxHash      common.Hash
	TxIndex     uint64
	BlockNumber uint64
	BlockTime   uint64
	CodeHash    common.Hash
	Name        string
	Symbol      string
	Decimals    uint8
	TotalSupply *big.Int
	WethPair    common.Address
	UsdtPair    common.Address
	CreatedAt   time.Time
}

type ProjectTokenMetadata struct {
	Name        string
	Symbol      string
	Decimals    uint8
	TotalSupply *big.Int
}

type ProjectRelatedWallet struct {
	ProjectID int64
	Wallet    common.Address
	Role      RelatedWalletRole
	CreatedAt time.Time
}

type ProjectInitialRecipient struct {
	ID                int64
	ProjectID         int64
	Wallet            common.Address
	RatioBPS          int64
	RankIndex         int32
	SourceTxHash      common.Hash
	SourceBlockNumber uint64
	CreatedAt         time.Time
}

type ContractCodeBlocklistEntry struct {
	CodeHash       common.Hash
	Note           string
	SourceChainID  int64
	SourceContract common.Address
	CreatedAt      time.Time
}

type WalletBlocklistEntry struct {
	Wallet    common.Address
	Note      string
	CreatedAt time.Time
}

type ProjectDataCollectionSchedule struct {
	ProjectID           int64
	DataType            DataCollectionType
	Status              DataCollectionScheduleStatus
	RefreshInterval     time.Duration
	NextRunAt           time.Time
	LatestTaskRevision  int64
	ConsecutiveFailures int32
	LastError           string
	LastCheckedAt       time.Time
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

type ProjectDataCollectionTask struct {
	ID             int64
	ProjectID      int64
	DataType       DataCollectionType
	Status         TaskStatus
	Revision       int64
	Attempts       int32
	AvailableAt    time.Time
	LockedAt       time.Time
	LeaseExpiresAt time.Time
	LastError      string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type ProjectDataCollectionTaskWithProject struct {
	Task    ProjectDataCollectionTask
	Project Project
}

type ProjectObservation struct {
	ID            int64
	ProjectID     int64
	DataType      DataCollectionType
	SchemaVersion int32
	ContentHash   common.Hash
	Payload       json.RawMessage
	BlockNumber   *uint64
	ObservedAt    time.Time
	LastCheckedAt time.Time
	CreatedAt     time.Time
}

type ProjectResearchState struct {
	ProjectID                   int64
	ChainID                     int64
	Contract                    common.Address
	Status                      ProjectResearchStatus
	EvidenceRevision            int64
	CurrentReportRevision       int64
	CurrentSelectionID          int64
	CurrentSelectionOutcome     SelectionOutcome
	LastEvaluatedReportRevision int64
	LastEvaluatedAt             time.Time
	ExpiresAt                   time.Time
	CreatedAt                   time.Time
	UpdatedAt                   time.Time
}

type ProjectReportBuildTask struct {
	ID               int64
	ProjectID        int64
	EvidenceRevision int64
	Status           TaskStatus
	Attempts         int32
	AvailableAt      time.Time
	LeaseExpiresAt   time.Time
	LastError        string
}

type ProjectSelectionEvaluationTask struct {
	ID             int64
	ProjectID      int64
	ReportRevision int64
	Status         TaskStatus
	Attempts       int32
	AvailableAt    time.Time
	LeaseExpiresAt time.Time
	LastError      string
}

type ProjectReportRevision struct {
	ID                  int64
	ProjectID           int64
	ChainID             int64
	Contract            common.Address
	Revision            int64
	SchemaVersion       int32
	ContentHash         common.Hash
	CompletenessStatus  string
	Evidence            []EvidenceReference
	Report              ResearchReportV1
	ObservedBlockNumber *uint64
	BuiltAt             time.Time
	CreatedAt           time.Time
}

type ProjectSelection struct {
	ID              int64
	ProjectID       int64
	ChainID         int64
	Contract        common.Address
	Outcome         SelectionOutcome
	StrategyKey     string
	StrategyVersion string
	ReportRevision  int64
	ReasonCodes     []string
	ReasonDetail    string
	DecidedAt       time.Time
	CreatedAt       time.Time
}

type StrategyInput struct {
	ProjectID      int64
	ReportRevision int64
	Report         ResearchReportV1
}

type SelectionDecision struct {
	Outcome      SelectionOutcome
	ReasonCodes  []string
	ReasonDetail string
}
