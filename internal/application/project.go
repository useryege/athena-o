package application

import (
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/google/uuid"
)

type Project struct {
	Meta      ProjectMeta
	PerfTrace PerfTrace

	InitState    ProjectInitState
	DelayedState ProjectDelayedState
	DynamicState ProjectDynamicState
}

type ProjectMeta struct {
	ProjectID   uuid.UUID
	BlockTime   uint64
	BlockNumber uint64
	Contract    common.Address
	Creator     common.Address
	Tx          *types.Transaction
}

type PerfTrace struct {
	BlockDiscoveredAt time.Time
	TxDiscoveredAt    time.Time
	FilterCompletedAt time.Time
}

type ProjectInitState struct {
	Name        OnceValue[string]
	Symbol      OnceValue[string]
	Decimals    OnceValue[uint8]
	TotalSupply OnceValue[*big.Int]
}

func (s *ProjectInitState) IsReady() bool {
	return s.Name.IsReady() && s.Symbol.IsReady() && s.Decimals.IsReady() && s.TotalSupply.IsReady()
}

type ProjectDelayedState struct {
	SourceCode    RetryUntilReadyValue[string]
	SourceCodeABI RetryUntilReadyValue[string]
}

type ProjectDynamicState struct {
	HolderCount DynamicValue[uint64]
}
