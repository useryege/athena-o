package store

import (
	"encoding/json"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

const (
	ChainIngestStatusRunning = "running"
	ChainIngestStatusStopped = "stopped"
)

const (
	ProjectCandidateStatusPending   = "pending"
	ProjectCandidateStatusQualified = "qualified"
	ProjectCandidateStatusRejected  = "rejected"
)

const (
	ProjectDataCollectionTypeAve        = "ave"
	ProjectDataCollectionTypeChainState = "chain_state"
)

const (
	ProjectDataCollectionStatusPending   = "pending"
	ProjectDataCollectionStatusSucceeded = "succeeded"
	ProjectDataCollectionStatusFailed    = "failed"
)

type ChainIngestCheckpoint struct {
	ChainID           int64
	ChainName         string
	Enabled           bool
	CursorBlockNumber uint64
	Status            string
	CreatedAt         time.Time
}

type ProjectCandidate struct {
	ID          int64
	ChainID     int64
	Contract    common.Address
	Creator     common.Address
	TxHash      common.Hash
	TxIndex     uint64
	BlockNumber uint64
	BlockTime   uint64
	Status      string
	CreatedAt   time.Time
}

type ContractCode struct {
	CodeHash            common.Hash
	SourceCode          string
	SourceCodeHash      common.Hash
	SourceCodeFetchedAt time.Time
	SourceCodeOrigin    string
	CreatedAt           time.Time
}

type Project struct {
	ID          int64
	ChainID     int64
	Contract    common.Address
	Creator     common.Address
	TxHash      common.Hash
	TxIndex     uint64
	BlockNumber uint64
	BlockTime   uint64
	CodeHash    common.Hash
	WethPair    common.Address
	UsdtPair    common.Address
	CreatedAt   time.Time
}

type ProjectAveData struct {
	ProjectID   int64
	AveResponse json.RawMessage
	FetchedAt   time.Time
	CreatedAt   time.Time
}

type ProjectChainStateData struct {
	ProjectID  int64
	ChainState json.RawMessage
	FetchedAt  time.Time
	CreatedAt  time.Time
}

type ProjectDataCollectionTask struct {
	ProjectID     int64
	DataType      string
	Status        string
	Attempts      int32
	NextAttemptAt time.Time
	LastError     string
	CreatedAt     time.Time
}

type ProjectDataCollectionTaskWithProject struct {
	Task    ProjectDataCollectionTask
	Project Project
}

type ProjectCandidatePage struct {
	Items    []ProjectCandidate
	Total    int64
	Page     int32
	PageSize int32
}

type ContractCodePage struct {
	Items    []ContractCode
	Total    int64
	Page     int32
	PageSize int32
}

type ProjectPage struct {
	Items    []Project
	Total    int64
	Page     int32
	PageSize int32
}
