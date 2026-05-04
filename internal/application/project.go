package application

import (
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/google/uuid"
)

// CreationTxEvent carries minimal data for contract creation transactions.
type Project struct {
	ProjectID   uuid.UUID
	BlockTime   uint64
	BlockNumber uint64
	Contract    common.Address
	Tx          *types.Transaction
	StaticState ProjectStaticState
	PerfTrace   *PerfTrace
}

type PerfTrace struct {
	BlockDiscoveredAt  time.Time
	TxDiscoveredAt     time.Time
	FilterStartedAt    time.Time
	FilterCompletedAt  time.Time
	ManagerStartedAt   time.Time
	ManagerCompletedAt time.Time
}

type ProjectStaticState struct {
	Name          StaticValue[string]
	Symbol        StaticValue[string]
	Decimals      StaticValue[uint8]
	SourceCode    StaticValue[string]
	SourceCodeABI StaticValue[string]
}
