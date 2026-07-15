package reporting

import (
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/useryege/athena/internal/token/research"
)

const ReportSchemaVersionV1 = int32(1)

type ProjectReportBuildTask struct {
	ID               int64
	ProjectID        int64
	EvidenceRevision int64
	Status           research.TaskStatus
	Attempts         int32
	AvailableAt      time.Time
	LeaseExpiresAt   time.Time
	LastError        string
}

type EvidenceReference struct {
	ObservationID int64                       `json:"observationId"`
	DataType      research.DataCollectionType `json:"dataType"`
	SchemaVersion int32                       `json:"schemaVersion"`
	ContentHash   string                      `json:"contentHash"`
	BlockNumber   *uint64                     `json:"blockNumber,omitempty"`
}

type ObservationFreshness struct {
	DataType      research.DataCollectionType `json:"dataType"`
	LastCheckedAt time.Time                   `json:"lastCheckedAt"`
}

type ResearchObservationsV1 struct {
	Ave            *research.AveObservationV1            `json:"ave,omitempty"`
	ChainState     *research.ChainStateObservationV1     `json:"chainState,omitempty"`
	WalletAssets   *research.WalletAssetObservationV1    `json:"walletAssets,omitempty"`
	Simulation     *research.SimulationObservationV1     `json:"simulation,omitempty"`
	ContractSource *research.ContractSourceObservationV1 `json:"contractSource,omitempty"`
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

type ProjectReportReadModel struct {
	Report         ProjectReportRevision
	ChainID        int64
	Name           string
	Symbol         string
	Contract       common.Address
	BuildStatus    string
	BuildAttempts  int32
	BuildLastError string
	BuildUpdatedAt time.Time
}

type ProjectReportPage struct {
	Items    []ProjectReportReadModel
	Total    int64
	Page     int32
	PageSize int32
}

type ReportRevisionPage struct {
	Items    []ProjectReportRevision
	Total    int64
	Page     int32
	PageSize int32
}
