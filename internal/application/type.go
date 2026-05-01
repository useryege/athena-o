package application

import (
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// CreationTxEvent carries minimal data for contract creation transactions.
type Project struct {
	BlockTime     uint64
	BlockNumber   uint64
	Tx            *types.Transaction
	TokenMetadata *TokenMetadata
	PerfTrace     *PerfTrace
}

type TokenMetadata struct {
	Name        string
	Symbol      string
	Decimals    uint8
	TotalSupply *big.Int
	Address     common.Address
}

type PerfTrace struct {
	BlockDiscoveredAt  time.Time
	TxDiscoveredAt     time.Time
	FilterStartedAt    time.Time
	FilterCompletedAt  time.Time
	ManagerStartedAt   time.Time
	ManagerCompletedAt time.Time
}
