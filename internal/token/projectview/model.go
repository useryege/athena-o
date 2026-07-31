package projectview

import (
	"time"

	"github.com/useryege/athena/internal/token/catalog"
	"github.com/useryege/athena/internal/token/reporting"
	"github.com/useryege/athena/internal/token/research"
	"github.com/useryege/athena/internal/token/selection"
	"github.com/useryege/athena/internal/token/shared"
	"github.com/useryege/athena/internal/token/swap"
)

type WalletTransactionCount struct {
	Wallet           shared.Address
	TransactionCount int64
}

type Detail struct {
	Project                 catalog.Project
	ResearchState           *research.ProjectResearchState
	CurrentReport           *reporting.ProjectReportRevision
	CurrentSelection        *selection.ProjectSelection
	CurrentObservations     []research.ProjectObservation
	RelatedWallets          []catalog.ProjectRelatedWallet
	InitialRecipients       []catalog.ProjectInitialRecipient
	CollectionSchedules     []research.ProjectDataCollectionSchedule
	WalletTransactionCounts []WalletTransactionCount
	TransactionCount        int64
}

type ObservationPage struct {
	Items    []research.ProjectObservation
	Total    int64
	Page     int32
	PageSize int32
}

type WalletNormalTransactionPage struct {
	Items    []research.WalletNormalTransaction
	Total    int64
	Page     int32
	PageSize int32
}

type TrendPoint struct {
	ObservedAt time.Time
	Value      string
}

type TrendSeries struct {
	Key      string
	Label    string
	Unit     string
	DataType research.DataCollectionType
	Points   []TrendPoint
}

type TrendResult struct {
	Range        string
	ObservedFrom time.Time
	GeneratedAt  time.Time
	Series       []TrendSeries
}

type SwapDirection string

const (
	SwapDirectionBuy     SwapDirection = "buy"
	SwapDirectionSell    SwapDirection = "sell"
	SwapDirectionComplex SwapDirection = "complex"
)

type SwapAsset struct {
	Address  shared.Address
	Symbol   string
	Decimals uint8
}

type SwapFlow struct {
	BaseIn   string
	BaseOut  string
	QuoteIn  string
	QuoteOut string
}

type SwapPriceSummary struct {
	Open  *string
	High  *string
	Low   *string
	Close *string
	VWAP  *string
}

type SwapActivityTotals struct {
	EventCount             int64
	TransactionCount       int64
	TransactionOriginCount int64
	BuyEventCount          int64
	SellEventCount         int64
	ComplexEventCount      int64
	Flow                   SwapFlow
	BuyQuoteVolume         string
	SellQuoteVolume        string
}

type SwapBlockActivity struct {
	SampleIndex            uint16
	BlockNumber            uint64
	BlockTime              uint64
	PreviousBlockGap       *uint64
	PreviousTimeGapSeconds *uint64
	Totals                 SwapActivityTotals
	Price                  SwapPriceSummary
}

type SwapPairActivity struct {
	ID                      int64
	Kind                    swap.PairKind
	Address                 shared.Address
	Status                  swap.PairStatus
	SwapBlockCount          uint16
	TargetSwapBlockCount    uint16
	BaseAsset               SwapAsset
	QuoteAsset              SwapAsset
	BaseTokenIndex          uint8
	StartBlockNumber        uint64
	StartBlockTime          uint64
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
	ExpiredReason           swap.ExpiredReason
	Totals                  SwapActivityTotals
	Blocks                  []SwapBlockActivity
}

type SwapActivity struct {
	ProjectID   int64
	ChainID     int64
	GeneratedAt time.Time
	Pairs       []SwapPairActivity
}

type SwapEvent struct {
	ID               int64
	TransactionHash  shared.Hash
	TransactionIndex uint64
	LogIndex         uint64
	TxFrom           shared.Address
	Sender           shared.Address
	ToAddress        shared.Address
	Amount0In        string
	Amount1In        string
	Amount0Out       string
	Amount1Out       string
	Flow             SwapFlow
	Direction        SwapDirection
	EffectivePrice   *string
}

type SwapEventPage struct {
	Items    []SwapEvent
	Total    int64
	Page     int32
	PageSize int32
}
