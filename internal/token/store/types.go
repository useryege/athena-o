package store

import (
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

type ChainIngestCheckpoint struct {
	ChainID              int64
	ChainName            string
	Enabled              bool
	FinalizedBlockNumber uint64
	FinalizedBlockHash   common.Hash
	CursorBlockNumber    uint64
	CursorBlockHash      common.Hash
	Status               string
	CreatedAt            time.Time
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
