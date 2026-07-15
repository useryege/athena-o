package discovery

import (
	"time"

	"github.com/ethereum/go-ethereum/common"
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
	ValidationLockToken      string
	ValidationLockedAt       time.Time
	ValidationLeaseExpiresAt time.Time
	CreatedAt                time.Time
}

type CandidatePage struct {
	Items    []ProjectCandidate
	Total    int64
	Page     int32
	PageSize int32
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
