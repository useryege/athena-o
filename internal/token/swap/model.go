package swap

import (
	"math/big"
	"time"

	"github.com/useryege/athena/internal/token/shared"
)

const (
	TargetSwapBlockCount       = uint16(100)
	FirstSwapTimeout           = 24 * time.Hour
	IdleSwapTimeout            = 24 * time.Hour
	AbsoluteObservationTimeout = 7 * 24 * time.Hour
)

type ProcessingStatus string

const (
	ProcessingStatusRunning ProcessingStatus = "running"
	ProcessingStatusStopped ProcessingStatus = "stopped"
)

type PairKind string

const (
	PairKindWETH PairKind = "weth"
	PairKindUSDT PairKind = "usdt"
)

type PairStatus string

const (
	PairStatusCollecting PairStatus = "collecting"
	PairStatusCompleted  PairStatus = "completed"
	PairStatusExpired    PairStatus = "expired"
)

type ExpiredReason string

const (
	ExpiredReasonNoSwap      ExpiredReason = "no_swap"
	ExpiredReasonInactive    ExpiredReason = "inactive"
	ExpiredReasonMaxDuration ExpiredReason = "max_duration"
)

type ProcessingCheckpoint struct {
	ChainID           int64
	CursorBlockNumber uint64
	Initialized       bool
	Status            ProcessingStatus
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type Pair struct {
	ID                      int64
	ProjectID               int64
	ChainID                 int64
	Kind                    PairKind
	Address                 shared.Address
	StartBlockNumber        uint64
	StartBlockTime          uint64
	SwapBlockCount          uint16
	Status                  PairStatus
	FirstSwapBlockNumber    *uint64
	FirstSwapBlockTime      *uint64
	LastSwapBlockNumber     *uint64
	LastSwapBlockTime       *uint64
	AbsoluteExpiryBlockTime uint64
	NextExpiryBlockTime     *uint64
	CompletedBlockNumber    *uint64
	CompletedBlockTime      *uint64
	ExpiredBlockNumber      *uint64
	ExpiredBlockTime        *uint64
	ExpiredReason           ExpiredReason
	CreatedAt               time.Time
	UpdatedAt               time.Time
}

type RawLog struct {
	Address          shared.Address
	BlockNumber      uint64
	TransactionHash  shared.Hash
	TransactionIndex uint64
	LogIndex         uint64
	Topics           []shared.Hash
	Data             []byte
	Removed          bool
}

type DecodedEvent struct {
	PairAddress      shared.Address
	TransactionHash  shared.Hash
	TransactionIndex uint64
	LogIndex         uint64
	TxFrom           shared.Address
	Sender           shared.Address
	ToAddress        shared.Address
	Amount0In        *big.Int
	Amount1In        *big.Int
	Amount0Out       *big.Int
	Amount1Out       *big.Int
}

type BlockObservation struct {
	BlockTime      uint64
	Events         []DecodedEvent
	RPCDuration    time.Duration
	DecodeDuration time.Duration
}

type Event struct {
	TransactionHash  shared.Hash
	TransactionIndex uint64
	LogIndex         uint64
	TxFrom           shared.Address
	Sender           shared.Address
	ToAddress        shared.Address
	Amount0In        *big.Int
	Amount1In        *big.Int
	Amount0Out       *big.Int
	Amount1Out       *big.Int
}

type PairBlockEvents struct {
	PairID int64
	Events []Event
}
