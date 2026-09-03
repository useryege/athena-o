package projectview

import (
	"math/big"
	"time"

	"github.com/useryege/athena/internal/token/catalog"
	"github.com/useryege/athena/internal/token/collection"
	"github.com/useryege/athena/internal/token/profile"
	"github.com/useryege/athena/internal/token/shared"
	"github.com/useryege/athena/internal/token/swap"
)

type WalletTransactionCount struct {
	Wallet           shared.Address
	TransactionCount int64
}

type ProjectListFilter struct {
	CodeHash         shared.Hash
	Contract         shared.Address
	CollectionStatus ProjectCollectionStatus
	ProfileState     ProjectProfileState
	Pair             ProjectListPairFilter
}

type ProjectListPairFilter struct {
	PairTokenBalanceExceedsTotalSupplyStates []string
	LPMinimumSupplyOnlyStates                []string
	FixedFeeAddressLPShareGte90PercentStates []string
	QuoteUSDTMin                             *big.Int
	QuoteUSDTMax                             *big.Int
}

type ProjectCollectionStatus string

const (
	ProjectCollectionStatusQueued         ProjectCollectionStatus = "queued"
	ProjectCollectionStatusCollecting     ProjectCollectionStatus = "collecting"
	ProjectCollectionStatusComplete       ProjectCollectionStatus = "complete"
	ProjectCollectionStatusNeedsAttention ProjectCollectionStatus = "needs_attention"
)

type ProjectProfileState string

const (
	ProjectProfileStatePending    ProjectProfileState = "pending"
	ProjectProfileStateComplete   ProjectProfileState = "complete"
	ProjectProfileStateIncomplete ProjectProfileState = "incomplete"
	ProjectProfileStateFailed     ProjectProfileState = "failed"
)

type ProjectMarketSummary struct {
	LogoURL         string
	CurrentPriceUSD *collection.Decimal
	MarketCapUSD    *collection.Decimal
	FDVUSD          *collection.Decimal
	TVLUSD          *collection.Decimal
	Holders         *int64
}

type ProjectPairProfileSummary struct {
	Kind                               profile.PairKind
	Address                            shared.Address
	IsCreated                          bool
	QuoteUSDTValueInt                  *big.Int
	ReserveUpdatedAt                   uint64
	PairTokenBalanceExceedsTotalSupply bool
	LPMinimumSupplyOnly                bool
	FixedFeeAddressLPShareGte90Percent bool
}

type ProjectListItem struct {
	ProjectID                int64
	ChainID                  int64
	Name                     string
	Symbol                   string
	Contract                 shared.Address
	CodeHash                 shared.Hash
	BlockNumber              uint64
	BlockTime                uint64
	TxHash                   shared.Hash
	CreatedAt                time.Time
	CollectionStatus         ProjectCollectionStatus
	CollectionSucceededCount int32
	CollectionTerminalCount  int32
	CollectionTotalCount     int32
	CollectionFailedCount    int32
	ProfileState             ProjectProfileState
	CompletenessStatus       profile.CompletenessStatus
	ProfileBuiltAt           time.Time
	Market                   *ProjectMarketSummary
	WrappedNativePair        *ProjectPairProfileSummary
	USDTPair                 *ProjectPairProfileSummary
}

type ProjectListPage struct {
	Items    []ProjectListItem
	Total    int64
	Page     int32
	PageSize int32
}

type Detail struct {
	Project                 catalog.Project
	Profile                 *profile.ProjectProfile
	CollectionStatus        ProjectCollectionStatus
	ProfileState            ProjectProfileState
	CollectionTasks         []collection.TaskDetail
	RelatedWallets          []catalog.ProjectRelatedWallet
	InitialRecipients       []catalog.ProjectInitialRecipient
	WalletTransactionCounts []WalletTransactionCount
	TransactionCount        int64
	GeneratedAt             time.Time
}

type WalletNormalTransactionPage struct {
	Items    []collection.WalletNormalTransaction
	Total    int64
	Page     int32
	PageSize int32
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
