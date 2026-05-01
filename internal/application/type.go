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

type TokenMetadata struct {
	Name        string
	Symbol      string
	Decimals    uint8
	TotalSupply *big.Int
	Address     common.Address

	SourceCode    string
	SourceCodeABI string
}

type PerfTrace struct {
	BlockDiscoveredAt  time.Time
	TxDiscoveredAt     time.Time
	FilterStartedAt    time.Time
	FilterCompletedAt  time.Time
	ManagerStartedAt   time.Time
	ManagerCompletedAt time.Time
}
