package profile

import (
	"errors"
	"math/big"
	"time"

	"github.com/useryege/athena/internal/token/collection"
	"github.com/useryege/athena/internal/token/shared"
)

const SchemaVersionV1 = int32(1)

var (
	// ErrBuildTaskClaimLost is returned by repositories when a worker attempts
	// to mutate a task after its lease or claim generation has changed.
	ErrBuildTaskClaimLost = errors.New("project profile build task claim lost")
	// ErrContentConflict is returned when the unique project profile already
	// exists with a different canonical content hash.
	ErrContentConflict = errors.New("project profile content conflict")
)

type BuildTaskStatus string

const (
	BuildTaskStatusPending   BuildTaskStatus = "pending"
	BuildTaskStatusRunning   BuildTaskStatus = "running"
	BuildTaskStatusSucceeded BuildTaskStatus = "succeeded"
	BuildTaskStatusFailed    BuildTaskStatus = "failed"
)

func (status BuildTaskStatus) IsTerminal() bool {
	return status == BuildTaskStatusSucceeded || status == BuildTaskStatusFailed
}

type CompletenessStatus string

const (
	CompletenessStatusComplete   CompletenessStatus = "complete"
	CompletenessStatusIncomplete CompletenessStatus = "incomplete"
)

type PairKind string

const (
	PairKindWrappedNative PairKind = "wrapped_native"
	PairKindUSDT          PairKind = "usdt"
)

type SourceVerificationStatus string

const (
	SourceVerificationStatusVerified   SourceVerificationStatus = "verified"
	SourceVerificationStatusUnverified SourceVerificationStatus = "unverified"
)

type BuildTask struct {
	ProjectID       int64
	Status          BuildTaskStatus
	FailureCount    int32
	AvailableAt     time.Time
	ClaimGeneration int64
	LockedAt        time.Time
	LeaseExpiresAt  time.Time
	LastError       string
	FinishedAt      time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// ProjectSnapshot is the immutable project context required to assemble a
// profile. Canonical project identity remains owned by the project table and is
// deliberately not duplicated in ProjectProfileV1.
type ProjectSnapshot struct {
	ID                    int64
	ChainID               int64
	Contract              shared.Address
	CodeHash              shared.Hash
	WrappedNativePair     shared.Address
	USDTPair              shared.Address
	DeploymentBlockNumber uint64
	RelatedWallets        []RelatedWallet
	InitialRecipients     []InitialRecipient
}

type RelatedWallet struct {
	Address shared.Address
	Role    string
}

type InitialRecipient struct {
	Address  shared.Address
	Rank     int32
	RatioBPS uint64
}

type BuildInput struct {
	Project ProjectSnapshot
	Tasks   []collection.Task
	Results []collection.Result
}

type ProjectProfile struct {
	ProjectID          int64
	SchemaVersion      int32
	CompletenessStatus CompletenessStatus
	FailedDataTypes    []collection.DataType
	ContentHash        shared.Hash
	Profile            ProjectProfileV1
	Projection         Projection
	BuiltAt            time.Time
	CreatedAt          time.Time
}

// ProjectProfileV1 is the canonical, immutable profile content. BuiltAt records
// when the successful build assembled this content.
type ProjectProfileV1 struct {
	SchemaVersion      int32                    `json:"schemaVersion"`
	ProjectID          int64                    `json:"projectId"`
	BuiltAt            time.Time                `json:"builtAt"`
	CompletenessStatus CompletenessStatus       `json:"completenessStatus"`
	FailedDataTypes    []collection.DataType    `json:"failedDataTypes"`
	Market             *MarketProfileV1         `json:"market,omitempty"`
	ContractSource     *ContractSourceProfileV1 `json:"contractSource,omitempty"`
	Pairs              PairProfilesV1           `json:"pairs"`
	WalletSummary      WalletSummaryV1          `json:"walletSummary"`
	Wallets            []WalletProfileV1        `json:"wallets"`
	Transactions       *TransactionSummaryV1    `json:"transactions,omitempty"`
	Evidence           []CollectionEvidenceV1   `json:"evidence"`
}

type MarketProfileV1 struct {
	LogoURL           string              `json:"logoUrl,omitempty"`
	CurrentPriceUSD   *collection.Decimal `json:"currentPriceUsd,omitempty"`
	CurrentPriceETH   *collection.Decimal `json:"currentPriceEth,omitempty"`
	MarketCapUSD      *collection.Decimal `json:"marketCapUsd,omitempty"`
	FDVUSD            *collection.Decimal `json:"fdvUsd,omitempty"`
	TVLUSD            *collection.Decimal `json:"tvlUsd,omitempty"`
	MainPairTVLUSD    *collection.Decimal `json:"mainPairTvlUsd,omitempty"`
	Holders           int64               `json:"holders"`
	LaunchAt          *time.Time          `json:"launchAt,omitempty"`
	ProviderUpdatedAt *time.Time          `json:"providerUpdatedAt,omitempty"`
	AveRisk           AveRiskProfileV1    `json:"aveRisk"`
}

type AveRiskProfileV1 struct {
	RiskLevel             int                 `json:"riskLevel"`
	RiskScore             *collection.Decimal `json:"riskScore,omitempty"`
	RiskInfo              string              `json:"riskInfo,omitempty"`
	Audited               bool                `json:"audited"`
	Mintable              *bool               `json:"mintable,omitempty"`
	HasMintMethod         bool                `json:"hasMintMethod"`
	LiquidityPoolUnlocked bool                `json:"liquidityPoolUnlocked"`
	OwnershipNotRenounced bool                `json:"ownershipNotRenounced"`
	NotAudited            bool                `json:"notAudited"`
	NotOpenSource         bool                `json:"notOpenSource"`
	InBlacklist           bool                `json:"inBlacklist"`
	Honeypot              bool                `json:"honeypot"`
}

type ContractSourceProfileV1 struct {
	CodeHash           shared.Hash              `json:"codeHash"`
	VerificationStatus SourceVerificationStatus `json:"verificationStatus"`
	ArtifactReference  string                   `json:"artifactReference,omitempty"`
}

type PairProfilesV1 struct {
	WrappedNative *PairProfileV1 `json:"wrappedNative,omitempty"`
	USDT          *PairProfileV1 `json:"usdt,omitempty"`
}

type PairProfileV1 struct {
	Kind       PairKind                 `json:"kind"`
	Address    shared.Address           `json:"address"`
	ChainState *PairChainStateProfileV1 `json:"chainState,omitempty"`
	Market     *PairMarketProfileV1     `json:"market,omitempty"`
}

type PairChainStateProfileV1 struct {
	IsCreated         bool            `json:"isCreated"`
	BaseBalance       *big.Int        `json:"baseBalance,omitempty"`
	QuoteBalance      *big.Int        `json:"quoteBalance,omitempty"`
	QuoteUsdtValue    *big.Int        `json:"quoteUsdtValue,omitempty"`
	QuoteUsdtValueInt *big.Int        `json:"quoteUsdtValueInt,omitempty"`
	ReserveUpdatedAt  uint64          `json:"reserveUpdatedAt"`
	Liquidity         PairLiquidityV1 `json:"liquidity"`
	Signals           PairSignalsV1   `json:"signals"`
}

type PairLiquidityV1 struct {
	TotalSupply            *big.Int `json:"totalSupply,omitempty"`
	LockedLiquidity        *big.Int `json:"lockedLiquidity,omitempty"`
	FixedFeeAddressBalance *big.Int `json:"fixedFeeAddressBalance,omitempty"`
	FixedFeeAddressShare   *big.Int `json:"fixedFeeAddressShare,omitempty"`
}

type PairSignalsV1 struct {
	PairTokenBalanceExceedsTotalSupply bool `json:"pairTokenBalanceExceedsTotalSupply"`
	LPMinimumSupplyOnly                bool `json:"lpMinimumSupplyOnly"`
	FixedFeeAddressLPShareGte90Percent bool `json:"fixedFeeAddressLpShareGte90Percent"`
}

type PairMarketProfileV1 struct {
	AMM           string              `json:"amm,omitempty"`
	Token0Address shared.Address      `json:"token0Address"`
	Token0Symbol  string              `json:"token0Symbol,omitempty"`
	Token1Address shared.Address      `json:"token1Address"`
	Token1Symbol  string              `json:"token1Symbol,omitempty"`
	Reserve0      *collection.Decimal `json:"reserve0,omitempty"`
	Reserve1      *collection.Decimal `json:"reserve1,omitempty"`
	VolumeUSD     *collection.Decimal `json:"volumeUsd,omitempty"`
	MarketCapUSD  *collection.Decimal `json:"marketCapUsd,omitempty"`
	FDVUSD        *collection.Decimal `json:"fdvUsd,omitempty"`
	IsFake        bool                `json:"isFake"`
	CreatedAt     *time.Time          `json:"createdAt,omitempty"`
	UpdatedAt     *time.Time          `json:"updatedAt,omitempty"`
}

type WalletSummaryV1 struct {
	WalletCount                   int32               `json:"walletCount"`
	RoleCounts                    []WalletRoleCountV1 `json:"roleCounts"`
	NativeBalanceTotal            *big.Int            `json:"nativeBalanceTotal,omitempty"`
	WrappedNativeBalanceTotal     *big.Int            `json:"wrappedNativeBalanceTotal,omitempty"`
	USDTBalanceTotal              *big.Int            `json:"usdtBalanceTotal,omitempty"`
	TrackedAssetUsdtValueTotal    *big.Int            `json:"trackedAssetUsdtValueTotal,omitempty"`
	InitialRecipientCount         int32               `json:"initialRecipientCount"`
	InitialRecipientAllocationBPS uint64              `json:"initialRecipientAllocationBps"`
	WalletsWithSimulationSignals  int32               `json:"walletsWithSimulationSignals"`
}

type WalletRoleCountV1 struct {
	Role  string `json:"role"`
	Count int32  `json:"count"`
}

type WalletProfileV1 struct {
	Address                 shared.Address             `json:"address"`
	Roles                   []string                   `json:"roles"`
	InitialRecipient        *InitialRecipientProfileV1 `json:"initialRecipient,omitempty"`
	Assets                  *WalletAssetProfileV1      `json:"assets,omitempty"`
	Simulation              *WalletSimulationProfileV1 `json:"simulation,omitempty"`
	TransactionSampleCapped bool                       `json:"transactionSampleCapped"`
}

type InitialRecipientProfileV1 struct {
	Rank     int32  `json:"rank"`
	RatioBPS uint64 `json:"ratioBps"`
}

type WalletAssetProfileV1 struct {
	NativeBalance         *big.Int `json:"nativeBalance,omitempty"`
	WrappedNativeBalance  *big.Int `json:"wrappedNativeBalance,omitempty"`
	USDTBalance           *big.Int `json:"usdtBalance,omitempty"`
	TrackedAssetUsdtValue *big.Int `json:"trackedAssetUsdtValue,omitempty"`
}

type WalletSimulationProfileV1 struct {
	TransferFromDeadToWalletCallSucceeded     bool `json:"transferFromDeadToWalletCallSucceeded"`
	TransferFromZeroToWalletCallSucceeded     bool `json:"transferFromZeroToWalletCallSucceeded"`
	TransferFromWethPairToWalletCallSucceeded bool `json:"transferFromWethPairToWalletCallSucceeded"`
	TransferFromUsdtPairToWalletCallSucceeded bool `json:"transferFromUsdtPairToWalletCallSucceeded"`
	TransferFromWalletToWethPairCallSucceeded bool `json:"transferFromWalletToWethPairCallSucceeded"`
	TransferFromWalletToUsdtPairCallSucceeded bool `json:"transferFromWalletToUsdtPairCallSucceeded"`
}

func (value WalletSimulationProfileV1) HasSignal() bool {
	return value.TransferFromDeadToWalletCallSucceeded ||
		value.TransferFromZeroToWalletCallSucceeded ||
		value.TransferFromWethPairToWalletCallSucceeded ||
		value.TransferFromUsdtPairToWalletCallSucceeded ||
		value.TransferFromWalletToWethPairCallSucceeded ||
		value.TransferFromWalletToUsdtPairCallSucceeded
}

type TransactionSummaryV1 struct {
	WalletCount                 int32                       `json:"walletCount"`
	TransactionAssociationCount int64                       `json:"transactionAssociationCount"`
	UniqueTransactionCount      int64                       `json:"uniqueTransactionCount"`
	SucceededTransactionCount   int64                       `json:"succeededTransactionCount"`
	FailedTransactionCount      int64                       `json:"failedTransactionCount"`
	TotalInflowNativeValue      *big.Int                    `json:"totalInflowNativeValue,omitempty"`
	TotalOutflowNativeValue     *big.Int                    `json:"totalOutflowNativeValue,omitempty"`
	TopMethods                  []TransactionMethodCountV1  `json:"topMethods"`
	TopCounterparties           []TransactionCounterpartyV1 `json:"topCounterparties"`
	CappedWallets               []shared.Address            `json:"cappedWallets"`
}

type TransactionMethodCountV1 struct {
	MethodID     string `json:"methodId,omitempty"`
	FunctionName string `json:"functionName,omitempty"`
	Count        int64  `json:"count"`
}

type TransactionCounterpartyV1 struct {
	Address shared.Address `json:"address"`
	Count   int64          `json:"count"`
}

type CollectionEvidenceV1 struct {
	TaskID              int64                 `json:"taskId"`
	DataType            collection.DataType   `json:"dataType"`
	Status              collection.TaskStatus `json:"status"`
	FailureCount        int32                 `json:"failureCount"`
	LastError           string                `json:"lastError,omitempty"`
	ResultSchemaVersion int32                 `json:"resultSchemaVersion,omitempty"`
	ResultContentHash   *shared.Hash          `json:"resultContentHash,omitempty"`
	BlockNumber         *uint64               `json:"blockNumber,omitempty"`
	CollectedAt         *time.Time            `json:"collectedAt,omitempty"`
}

// Projection contains only list/filter fields duplicated beside the canonical
// JSON profile. It is derived by Build and must never be supplied independently.
type Projection struct {
	LogoURL              string
	CurrentPriceUSD      *collection.Decimal
	MarketCapUSD         *collection.Decimal
	FDVUSD               *collection.Decimal
	TVLUSD               *collection.Decimal
	Holders              *int64
	ContractSourceStatus *SourceVerificationStatus
	WrappedNativePair    *PairProjection
	USDTPair             *PairProjection
}

type PairProjection struct {
	IsCreated                          bool
	PairTokenBalanceExceedsTotalSupply bool
	LPMinimumSupplyOnly                bool
	FixedFeeAddressLPShareGte90Percent bool
	QuoteUsdtValueInt                  *big.Int
	ReserveUpdatedAt                   uint64
}
