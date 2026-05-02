package application

import (
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/google/uuid"
)

// CreationTxEvent carries minimal data for contract creation transactions.
type Project struct {
	ProjectID     uuid.UUID
	BlockTime     uint64
	BlockNumber   uint64
	Tx            *types.Transaction
	PerfTrace     *PerfTrace
	TokenMetadata *TokenMetadata
	Creator       *Wallet
}

type Wallet struct {
	Address common.Address
}

type StaticMetadataStatus string

const (
	StaticMetadataPending StaticMetadataStatus = "pending"
	StaticMetadataReady   StaticMetadataStatus = "ready"
	StaticMetadataFailed  StaticMetadataStatus = "failed"
)

type TokenMetadata struct {
	Static  TokenStaticMetadata
	Dynamic TokenDynamicState
}

type TokenStaticMetadata struct {
	Name          string
	Symbol        string
	Decimals      uint8
	Address       common.Address
	SourceCode    string
	SourceCodeABI string
	TotalSupply   *big.Int

	FetchedAt time.Time
	Status    StaticMetadataStatus
}

type TokenDynamicState struct {
	BlockNumber uint64
	StateHash   string
	UpdatedAt   time.Time
	CheckedAt   time.Time
}

type PerfTrace struct {
	BlockDiscoveredAt  time.Time
	TxDiscoveredAt     time.Time
	FilterStartedAt    time.Time
	FilterCompletedAt  time.Time
	ManagerStartedAt   time.Time
	ManagerCompletedAt time.Time
}
