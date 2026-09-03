package collection

import (
	"math/big"
	"time"

	"github.com/useryege/athena/internal/token/shared"
)

type AveResultV1 struct {
	ChainID int64       `json:"chainId"`
	Token   AveTokenV1  `json:"token"`
	Pairs   []AvePairV1 `json:"pairs"`
	AveRisk AveRiskV1   `json:"aveRisk"`
}

type AveTokenV1 struct {
	Address         shared.Address `json:"address"`
	Name            string         `json:"name"`
	Symbol          string         `json:"symbol"`
	LogoURL         string         `json:"logoUrl,omitempty"`
	Decimals        int            `json:"decimals"`
	TotalSupply     *Decimal       `json:"totalSupply,omitempty"`
	CurrentPriceUSD *Decimal       `json:"currentPriceUsd,omitempty"`
	CurrentPriceETH *Decimal       `json:"currentPriceEth,omitempty"`
	MarketCap       *Decimal       `json:"marketCap,omitempty"`
	FDV             *Decimal       `json:"fdv,omitempty"`
	TVL             *Decimal       `json:"tvl,omitempty"`
	MainPairTVL     *Decimal       `json:"mainPairTvl,omitempty"`
	Holders         int            `json:"holders"`
	LaunchAt        *time.Time     `json:"launchAt,omitempty"`
	UpdatedAt       *time.Time     `json:"updatedAt,omitempty"`
}

type AveRiskV1 struct {
	IsAudited        bool     `json:"isAudited"`
	RiskLevel        int      `json:"riskLevel"`
	RiskScore        *Decimal `json:"riskScore,omitempty"`
	RiskInfo         string   `json:"riskInfo"`
	IsMintable       *bool    `json:"isMintable,omitempty"`
	HasMintMethod    bool     `json:"hasMintMethod"`
	IsLPNotLocked    bool     `json:"isLpNotLocked"`
	HasNotRenounced  bool     `json:"hasNotRenounced"`
	HasNotAudited    bool     `json:"hasNotAudited"`
	HasNotOpenSource bool     `json:"hasNotOpenSource"`
	IsInBlacklist    bool     `json:"isInBlacklist"`
	IsHoneypot       bool     `json:"isHoneypot"`
}

type AvePairV1 struct {
	Pair          shared.Address `json:"pair"`
	ChainID       int64          `json:"chainId"`
	AMM           string         `json:"amm"`
	Token0Address shared.Address `json:"token0Address"`
	Token0Symbol  string         `json:"token0Symbol"`
	Token1Address shared.Address `json:"token1Address"`
	Token1Symbol  string         `json:"token1Symbol"`
	Reserve0      *Decimal       `json:"reserve0,omitempty"`
	Reserve1      *Decimal       `json:"reserve1,omitempty"`
	VolumeUSD     *Decimal       `json:"volumeUsd,omitempty"`
	MarketCap     *Decimal       `json:"marketCap,omitempty"`
	FDV           *Decimal       `json:"fdv,omitempty"`
	IsFake        bool           `json:"isFake"`
	CreatedAt     *time.Time     `json:"createdAt,omitempty"`
	UpdatedAt     *time.Time     `json:"updatedAt,omitempty"`
}

type ChainStateResultV1 struct {
	TokenContract  shared.Address `json:"tokenContract"`
	ChainTimestamp *big.Int       `json:"chainTimestamp"`
	Token          ChainTokenV1   `json:"token"`
	WethPair       ChainPairV1    `json:"wethPair"`
	UsdtPair       ChainPairV1    `json:"usdtPair"`
}

type ChainTokenV1 struct {
	IsValidERC20 bool           `json:"isValidErc20"`
	Name         string         `json:"name"`
	Symbol       string         `json:"symbol"`
	Decimals     uint8          `json:"decimals"`
	TotalSupply  *big.Int       `json:"totalSupply"`
	WethPair     shared.Address `json:"wethPair"`
	UsdtPair     shared.Address `json:"usdtPair"`
}

type ChainPairV1 struct {
	PairContract                       shared.Address            `json:"pairContract"`
	IsCreated                          bool                      `json:"isCreated"`
	LiquidityState                     ChainPairLiquidityStateV1 `json:"liquidityState"`
	BaseBalance                        *big.Int                  `json:"baseBalance"`
	QuoteBalance                       *big.Int                  `json:"quoteBalance"`
	QuoteUsdtValue                     *big.Int                  `json:"quoteUsdtValue"`
	QuoteUsdtValueInt                  *big.Int                  `json:"quoteUsdtValueInt"`
	ReserveUpdatedAt                   uint32                    `json:"reserveUpdatedAt"`
	PairTokenBalanceExceedsTotalSupply bool                      `json:"pairTokenBalanceExceedsTotalSupply"`
	LPMinimumSupplyOnly                bool                      `json:"lpMinimumSupplyOnly"`
	FixedFeeAddressLPShareGte90Percent bool                      `json:"fixedFeeAddressLpShareGte90Percent"`
}

type ChainPairLiquidityStateV1 struct {
	TotalSupply                    *big.Int `json:"totalSupply"`
	LockedLiquidity                *big.Int `json:"lockedLiquidity"`
	FeeAddressHoldLiquidityBalance *big.Int `json:"feeAddressHoldLiquidityBalance"`
	FeeAddressHoldLiquidityRatio   *big.Int `json:"feeAddressHoldLiquidityRatio"`
}

type WalletAssetResultV1 struct {
	Items []WalletAssetStateV1 `json:"items"`
}

type WalletAssetStateV1 struct {
	ChainID               int64          `json:"chainId"`
	Wallet                shared.Address `json:"wallet"`
	WethBalance           *big.Int       `json:"wethBalance"`
	UsdtBalance           *big.Int       `json:"usdtBalance"`
	NativeBalance         *big.Int       `json:"nativeBalance"`
	TrackedAssetUsdtValue *big.Int       `json:"trackedAssetUsdtValue"`
}

type SimulationResultV1 struct {
	Items []WalletSimulationResultV1 `json:"items"`
}

type WalletSimulationResultV1 struct {
	ProjectID                                 int64          `json:"projectId"`
	Wallet                                    shared.Address `json:"wallet"`
	TransferFromDeadToWalletCallSucceeded     bool           `json:"transferFromDeadToWalletCallSucceeded"`
	TransferFromZeroToWalletCallSucceeded     bool           `json:"transferFromZeroToWalletCallSucceeded"`
	TransferFromWethPairToWalletCallSucceeded bool           `json:"transferFromWethPairToWalletCallSucceeded"`
	TransferFromUsdtPairToWalletCallSucceeded bool           `json:"transferFromUsdtPairToWalletCallSucceeded"`
	TransferFromWalletToWethPairCallSucceeded bool           `json:"transferFromWalletToWethPairCallSucceeded"`
	TransferFromWalletToUsdtPairCallSucceeded bool           `json:"transferFromWalletToUsdtPairCallSucceeded"`
}

type SourceVerificationStatus string

const (
	SourceVerificationStatusVerified   SourceVerificationStatus = "verified"
	SourceVerificationStatusUnverified SourceVerificationStatus = "unverified"
)

type ContractSourceResultV1 struct {
	CodeHash           shared.Hash              `json:"codeHash"`
	VerificationStatus SourceVerificationStatus `json:"verificationStatus"`
	ArtifactReference  string                   `json:"artifactReference,omitempty"`
}

type WalletNormalTransactionsResultV1 struct {
	WalletCount                 int                               `json:"walletCount"`
	TransactionAssociationCount int                               `json:"transactionAssociationCount"`
	UniqueTransactionCount      int                               `json:"uniqueTransactionCount"`
	SucceededTransactionCount   int                               `json:"succeededTransactionCount"`
	FailedTransactionCount      int                               `json:"failedTransactionCount"`
	TotalInflowNativeValue      *big.Int                          `json:"totalInflowNativeValue"`
	TotalOutflowNativeValue     *big.Int                          `json:"totalOutflowNativeValue"`
	TopMethods                  []NormalTransactionMethodV1       `json:"topMethods"`
	TopCounterparties           []NormalTransactionCounterpartyV1 `json:"topCounterparties"`
	CappedWallets               []shared.Address                  `json:"cappedWallets"`
}

type NormalTransactionMethodV1 struct {
	MethodID     string `json:"methodId"`
	FunctionName string `json:"functionName"`
	Count        int    `json:"count"`
}

type NormalTransactionCounterpartyV1 struct {
	Address shared.Address `json:"address"`
	Count   int            `json:"count"`
}
