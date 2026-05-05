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

type OnceValue[T any] struct {
	Value      T
	ResolvedAt time.Time
}

func (v *OnceValue[T]) Get() T {
	return v.Value
}

func (v *OnceValue[T]) Set(value T) {
	v.Value = value
	v.ResolvedAt = time.Now()
}

func (v *OnceValue[T]) IsReady() bool {
	return !v.ResolvedAt.IsZero()
}

type ProjectDelayedState struct {
	SourceCode    RetryUntilReadyValue[string]
	SourceCodeABI RetryUntilReadyValue[string]
}

type RetryUntilReadyValue[T any] struct {
	Value  T
	Status StaticValueStatus

	AttemptCount  int
	LastAttemptAt time.Time
	NextAttemptAt time.Time
	ResolvedAt    time.Time

	LastError string
}

func (v *RetryUntilReadyValue[T]) IsReady() bool {
	return v.Status == StaticValueReady
}

func (v *RetryUntilReadyValue[T]) Get() T {
	return v.Value
}

func (v *RetryUntilReadyValue[T]) Set(value T) {
	v.Value = value
	v.ResolvedAt = time.Now()
}

type ProjectDynamicState struct {
	HolderCount DynamicValue[uint64]
}

type DynamicValue[T comparable] struct {
	Value T

	Initialized bool

	LastCheckedAt time.Time
	LastChangedAt time.Time
	UpdatedAt     time.Time

	LastError string
}

func (v *DynamicValue[T]) Update(newValue T, now time.Time) bool {
	v.LastCheckedAt = now

	if !v.Initialized {
		v.Value = newValue
		v.Initialized = true
		v.UpdatedAt = now
		v.LastChangedAt = now
		v.LastError = ""
		return true
	}

	if v.Value == newValue {
		v.UpdatedAt = now
		v.LastError = ""
		return false
	}

	v.Value = newValue
	v.UpdatedAt = now
	v.LastChangedAt = now
	v.LastError = ""
	return true
}
