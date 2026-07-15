package research

import (
	"encoding/json"
	"time"

	"github.com/useryege/athena/internal/token/shared"
)

const ObservationSchemaVersionV1 = int32(1)

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

func ParseDataCollectionType(value string) (DataCollectionType, bool) {
	dataType := DataCollectionType(value)
	switch dataType {
	case DataCollectionTypeAve, DataCollectionTypeChainState, DataCollectionTypeWalletAssetState, DataCollectionTypeSimulationResult, DataCollectionTypeContractCodeSource:
		return dataType, true
	default:
		return "", false
	}
}

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

type ProjectCollectionContext struct {
	ID              int64
	ChainID         int64
	Contract        shared.Address
	CodeHash        shared.Hash
	WethPair        shared.Address
	UsdtPair        shared.Address
	RelatedWallets  []shared.Address
	RefreshInterval time.Duration
}

type ProjectDataCollectionTaskWithProject struct {
	Task    ProjectDataCollectionTask
	Project ProjectCollectionContext
}

type ProjectObservation struct {
	ID            int64
	ProjectID     int64
	DataType      DataCollectionType
	SchemaVersion int32
	ContentHash   shared.Hash
	Payload       json.RawMessage
	BlockNumber   *uint64
	ObservedAt    time.Time
	LastCheckedAt time.Time
	CreatedAt     time.Time
}

type ProjectResearchState struct {
	ProjectID                   int64
	ChainID                     int64
	Contract                    shared.Address
	Status                      ProjectResearchStatus
	EvidenceRevision            int64
	CurrentReportRevision       int64
	CurrentSelectionID          int64
	CurrentSelectionOutcome     string
	LastEvaluatedReportRevision int64
	LastEvaluatedAt             time.Time
	ExpiresAt                   time.Time
	CreatedAt                   time.Time
	UpdatedAt                   time.Time
}

type CollectionTaskPage struct {
	Items    []ProjectDataCollectionTask
	Total    int64
	Page     int32
	PageSize int32
}

type ResearchStatePage struct {
	Items    []ProjectResearchState
	Total    int64
	Page     int32
	PageSize int32
}
