package projectview

import (
	"math/big"
	"time"

	"github.com/useryege/athena/internal/token/catalog"
	"github.com/useryege/athena/internal/token/collection"
	"github.com/useryege/athena/internal/token/profile"
	"github.com/useryege/athena/internal/token/shared"
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
