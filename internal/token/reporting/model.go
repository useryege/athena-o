package reporting

import (
	"encoding/json"
	"math/big"
	"time"

	"github.com/useryege/athena/internal/token/shared"
)

const ReportSchemaVersionV1 = int32(1)

type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"
	TaskStatusRunning   TaskStatus = "running"
	TaskStatusSucceeded TaskStatus = "succeeded"
	TaskStatusFailed    TaskStatus = "failed"
)

type DataCollectionType string

const (
	DataCollectionTypeAve                DataCollectionType = "ave"
	DataCollectionTypeChainState         DataCollectionType = "chain_state"
	DataCollectionTypeWalletAssetState   DataCollectionType = "wallet_asset_state"
	DataCollectionTypeSimulationResult   DataCollectionType = "simulation_result"
	DataCollectionTypeContractCodeSource DataCollectionType = "contract_code_source"
)

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

type ObservationSnapshot struct {
	ID            int64
	ProjectID     int64
	DataType      DataCollectionType
	SchemaVersion int32
	ContentHash   shared.Hash
	Payload       json.RawMessage
	BlockNumber   *uint64
	ObservedAt    time.Time
	LastCheckedAt time.Time
}

type EvidenceReference struct {
	ObservationID int64              `json:"observationId"`
	DataType      DataCollectionType `json:"dataType"`
	SchemaVersion int32              `json:"schemaVersion"`
	ContentHash   string             `json:"contentHash"`
	BlockNumber   *uint64            `json:"blockNumber,omitempty"`
}

type ObservationFreshness struct {
	DataType      DataCollectionType `json:"dataType"`
	LastCheckedAt time.Time          `json:"lastCheckedAt"`
}

type ResearchObservationsV1 struct {
	Ave            json.RawMessage `json:"ave,omitempty"`
	ChainState     json.RawMessage `json:"chainState,omitempty"`
	WalletAssets   json.RawMessage `json:"walletAssets,omitempty"`
	Simulation     json.RawMessage `json:"simulation,omitempty"`
	ContractSource json.RawMessage `json:"contractSource,omitempty"`
}

type ReportRiskSummary struct {
	WethPairIsCreated         *bool    `json:"wethPairIsCreated,omitempty"`
	WethPairIsRemoveLiquidity *bool    `json:"wethPairIsRemoveLiquidity,omitempty"`
	WethPairIsMint            *bool    `json:"wethPairIsMint,omitempty"`
	WethPairQuoteUsdtValueInt *big.Int `json:"wethPairQuoteUsdtValueInt,omitempty"`
	WethPairLastSwapTimestamp *uint64  `json:"wethPairLastSwapTimestamp,omitempty"`
	UsdtPairIsCreated         *bool    `json:"usdtPairIsCreated,omitempty"`
	UsdtPairIsRemoveLiquidity *bool    `json:"usdtPairIsRemoveLiquidity,omitempty"`
	UsdtPairIsMint            *bool    `json:"usdtPairIsMint,omitempty"`
	UsdtPairQuoteUsdtValueInt *big.Int `json:"usdtPairQuoteUsdtValueInt,omitempty"`
	UsdtPairLastSwapTimestamp *uint64  `json:"usdtPairLastSwapTimestamp,omitempty"`
}

type ResearchReportV1 struct {
	SchemaVersion      int32                  `json:"schemaVersion"`
	ProjectID          int64                  `json:"projectId"`
	CompletenessStatus string                 `json:"completenessStatus"`
	Observations       ResearchObservationsV1 `json:"observations"`
	Freshness          []ObservationFreshness `json:"freshness"`
	RiskSummary        ReportRiskSummary      `json:"riskSummary"`
}

type ProjectReportRevision struct {
	ID                  int64
	ProjectID           int64
	ChainID             int64
	Contract            shared.Address
	Revision            int64
	SchemaVersion       int32
	ContentHash         shared.Hash
	CompletenessStatus  string
	Evidence            []EvidenceReference
	Report              ResearchReportV1
	ObservedBlockNumber *uint64
	BuiltAt             time.Time
	CreatedAt           time.Time
}

type ReportRevisionPage struct {
	Items    []ProjectReportRevision
	Total    int64
	Page     int32
	PageSize int32
}
