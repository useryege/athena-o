package research

import (
	"math/big"
	"time"

	"github.com/useryege/athena/internal/token/shared"
)

type AveObservationV1 struct {
	ChainID   int64       `json:"chainId"`
	Token     AveTokenV1  `json:"token"`
	Pairs     []AvePairV1 `json:"pairs"`
	IsAudited bool        `json:"isAudited"`
}

type AveTokenV1 struct {
	Address          shared.Address `json:"address"`
	Name             string         `json:"name"`
	Symbol           string         `json:"symbol"`
	Decimals         int            `json:"decimals"`
	TotalSupply      *Decimal       `json:"totalSupply,omitempty"`
	CurrentPriceUSD  *Decimal       `json:"currentPriceUsd,omitempty"`
	CurrentPriceETH  *Decimal       `json:"currentPriceEth,omitempty"`
	MarketCap        *Decimal       `json:"marketCap,omitempty"`
	FDV              *Decimal       `json:"fdv,omitempty"`
	TVL              *Decimal       `json:"tvl,omitempty"`
	MainPairTVL      *Decimal       `json:"mainPairTvl,omitempty"`
	Holders          int            `json:"holders"`
	RiskLevel        int            `json:"riskLevel"`
	RiskScore        *Decimal       `json:"riskScore,omitempty"`
	RiskInfo         string         `json:"riskInfo"`
	IsMintable       *bool          `json:"isMintable,omitempty"`
	HasMintMethod    bool           `json:"hasMintMethod"`
	IsLPNotLocked    bool           `json:"isLpNotLocked"`
	HasNotRenounced  bool           `json:"hasNotRenounced"`
	HasNotAudited    bool           `json:"hasNotAudited"`
	HasNotOpenSource bool           `json:"hasNotOpenSource"`
	IsInBlacklist    bool           `json:"isInBlacklist"`
	IsHoneypot       bool           `json:"isHoneypot"`
	LaunchAt         *time.Time     `json:"launchAt,omitempty"`
	UpdatedAt        *time.Time     `json:"updatedAt,omitempty"`
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

type ChainStateObservationV1 struct {
	TokenContract shared.Address     `json:"tokenContract"`
	UpdatedAt     *big.Int           `json:"updatedAt"`
	Token         ChainTokenV1       `json:"token"`
	TokenReport   ChainTokenReportV1 `json:"tokenReport"`
	WethPair      ChainPairV1        `json:"wethPair"`
	WethReport    ChainPairReportV1  `json:"wethReport"`
	UsdtPair      ChainPairV1        `json:"usdtPair"`
	UsdtReport    ChainPairReportV1  `json:"usdtReport"`
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

type ChainTokenReportV1 struct {
	IsValidERC20 bool `json:"isValidErc20"`
}

type ChainPairV1 struct {
	PairContract      shared.Address            `json:"pairContract"`
	IsCreated         bool                      `json:"isCreated"`
	LiquidityState    ChainPairLiquidityStateV1 `json:"liquidityState"`
	BaseBalance       *big.Int                  `json:"baseBalance"`
	QuoteBalance      *big.Int                  `json:"quoteBalance"`
	QuoteUsdtValue    *big.Int                  `json:"quoteUsdtValue"`
	QuoteUsdtValueInt *big.Int                  `json:"quoteUsdtValueInt"`
	LastSwapTimestamp uint32                    `json:"lastSwapTimestamp"`
}

type ChainPairLiquidityStateV1 struct {
	TotalSupply                    *big.Int `json:"totalSupply"`
	LockedLiquidity                *big.Int `json:"lockedLiquidity"`
	FeeAddressHoldLiquidityBalance *big.Int `json:"feeAddressHoldLiquidityBalance"`
	FeeAddressHoldLiquidityRatio   *big.Int `json:"feeAddressHoldLiquidityRatio"`
}

type ChainPairReportV1 struct {
	IsRemoveLiquidity bool `json:"isRemoveLiquidity"`
	IsMint            bool `json:"isMint"`
}

type WalletAssetObservationV1 struct {
	Items []WalletAssetStateV1 `json:"items"`
}

type WalletAssetStateV1 struct {
	ChainID             int64          `json:"chainId"`
	Wallet              shared.Address `json:"wallet"`
	WethBalance         *big.Int       `json:"wethBalance"`
	UsdtBalance         *big.Int       `json:"usdtBalance"`
	NativeBalance       *big.Int       `json:"nativeBalance"`
	TotalAssetUsdtValue *big.Int       `json:"totalAssetUsdtValue"`
}

type SimulationObservationV1 struct {
	Items []SimulationResultV1 `json:"items"`
}

type SimulationResultV1 struct {
	ProjectID                          int64          `json:"projectId"`
	Wallet                             shared.Address `json:"wallet"`
	CanMintFromDeadViaTransferFrom     bool           `json:"canMintFromDeadViaTransferFrom"`
	CanMintFromZeroViaTransferFrom     bool           `json:"canMintFromZeroViaTransferFrom"`
	CanMintFromWethPairViaTransferFrom bool           `json:"canMintFromWethPairViaTransferFrom"`
	CanMintFromUsdtPairViaTransferFrom bool           `json:"canMintFromUsdtPairViaTransferFrom"`
	CanMintViaTransferToWethPair       bool           `json:"canMintViaTransferToWethPair"`
	CanMintViaTransferToUsdtPair       bool           `json:"canMintViaTransferToUsdtPair"`
}

type ContractSourceObservationV1 struct {
	CodeHash        shared.Hash `json:"codeHash"`
	SourceAvailable bool        `json:"sourceAvailable"`
}
