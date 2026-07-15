package domain

import (
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

type AveObservationV1 struct {
	Status     int         `json:"status"`
	Message    string      `json:"message"`
	SourceType int         `json:"sourceType"`
	Token      AveTokenV1  `json:"token"`
	Pairs      []AvePairV1 `json:"pairs"`
	IsAudited  bool        `json:"isAudited"`
}

type AveTokenV1 struct {
	Address          string `json:"address"`
	Chain            string `json:"chain"`
	Name             string `json:"name"`
	Symbol           string `json:"symbol"`
	Decimals         int    `json:"decimals"`
	TotalSupply      string `json:"totalSupply"`
	CurrentPriceUSD  string `json:"currentPriceUsd"`
	CurrentPriceETH  string `json:"currentPriceEth"`
	MarketCap        string `json:"marketCap"`
	FDV              string `json:"fdv"`
	TVL              string `json:"tvl"`
	MainPairTVL      string `json:"mainPairTvl"`
	Holders          int    `json:"holders"`
	RiskLevel        int    `json:"riskLevel"`
	RiskScore        string `json:"riskScore"`
	RiskInfo         string `json:"riskInfo"`
	IsMintable       string `json:"isMintable"`
	HasMintMethod    bool   `json:"hasMintMethod"`
	IsLPNotLocked    bool   `json:"isLpNotLocked"`
	HasNotRenounced  bool   `json:"hasNotRenounced"`
	HasNotAudited    bool   `json:"hasNotAudited"`
	HasNotOpenSource bool   `json:"hasNotOpenSource"`
	IsInBlacklist    bool   `json:"isInBlacklist"`
	IsHoneypot       bool   `json:"isHoneypot"`
	LaunchAt         int64  `json:"launchAt"`
	UpdatedAt        int64  `json:"updatedAt"`
}

type AvePairV1 struct {
	Pair          string `json:"pair"`
	Chain         string `json:"chain"`
	AMM           string `json:"amm"`
	Token0Address string `json:"token0Address"`
	Token0Symbol  string `json:"token0Symbol"`
	Token1Address string `json:"token1Address"`
	Token1Symbol  string `json:"token1Symbol"`
	Reserve0      string `json:"reserve0"`
	Reserve1      string `json:"reserve1"`
	VolumeUSD     string `json:"volumeUsd"`
	MarketCap     string `json:"marketCap"`
	FDV           string `json:"fdv"`
	IsFake        bool   `json:"isFake"`
	CreatedAt     int64  `json:"createdAt"`
	UpdatedAt     int64  `json:"updatedAt"`
}

type ChainStateObservationV1 struct {
	TokenContract common.Address     `json:"tokenContract"`
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
	WethPair     common.Address `json:"wethPair"`
	UsdtPair     common.Address `json:"usdtPair"`
}

type ChainTokenReportV1 struct {
	IsValidERC20 bool `json:"isValidErc20"`
}

type ChainPairV1 struct {
	PairContract      common.Address            `json:"pairContract"`
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
	ChainID       int64          `json:"chainId"`
	Wallet        common.Address `json:"wallet"`
	WethBalance   *big.Int       `json:"wethBalance"`
	UsdtBalance   *big.Int       `json:"usdtBalance"`
	NativeBalance *big.Int       `json:"nativeBalance"`
	UsdtValue     *big.Int       `json:"usdtValue"`
}

type SimulationObservationV1 struct {
	Items []SimulationResultV1 `json:"items"`
}

type SimulationResultV1 struct {
	ProjectID                          int64          `json:"projectId"`
	Wallet                             common.Address `json:"wallet"`
	CanMintFromDeadViaTransferFrom     bool           `json:"canMintFromDeadViaTransferFrom"`
	CanMintFromZeroViaTransferFrom     bool           `json:"canMintFromZeroViaTransferFrom"`
	CanMintFromWethPairViaTransferFrom bool           `json:"canMintFromWethPairViaTransferFrom"`
	CanMintFromUsdtPairViaTransferFrom bool           `json:"canMintFromUsdtPairViaTransferFrom"`
	CanMintViaTransferToWethPair       bool           `json:"canMintViaTransferToWethPair"`
	CanMintViaTransferToUsdtPair       bool           `json:"canMintViaTransferToUsdtPair"`
}

type ContractSourceObservationV1 struct {
	CodeHash        common.Hash `json:"codeHash"`
	SourceAvailable bool        `json:"sourceAvailable"`
}

type EvidenceReference struct {
	ObservationID int64              `json:"observationId"`
	DataType      DataCollectionType `json:"dataType"`
	SchemaVersion int32              `json:"schemaVersion"`
	ContentHash   string             `json:"contentHash"`
	BlockNumber   *uint64            `json:"blockNumber,omitempty"`
}

type ObservationFreshness struct {
	DataType      DataCollectionType `json:"dataType"`
	LastCheckedAt time.Time          `json:"lastCheckedAt"`
}

type ResearchObservationsV1 struct {
	Ave            *AveObservationV1            `json:"ave,omitempty"`
	ChainState     *ChainStateObservationV1     `json:"chainState,omitempty"`
	WalletAssets   *WalletAssetObservationV1    `json:"walletAssets,omitempty"`
	Simulation     *SimulationObservationV1     `json:"simulation,omitempty"`
	ContractSource *ContractSourceObservationV1 `json:"contractSource,omitempty"`
}

type ReportRiskSummary struct {
	WethPairIsCreated         *bool    `json:"wethPairIsCreated,omitempty"`
	WethPairIsRemoveLiquidity *bool    `json:"wethPairIsRemoveLiquidity,omitempty"`
	WethPairIsMint            *bool    `json:"wethPairIsMint,omitempty"`
	WethPairQuoteUsdtValueInt *big.Int `json:"wethPairQuoteUsdtValueInt,omitempty"`
	WethPairLastSwapTimestamp *uint64  `json:"wethPairLastSwapTimestamp,omitempty"`
	UsdtPairIsCreated         *bool    `json:"usdtPairIsCreated,omitempty"`
	UsdtPairIsRemoveLiquidity *bool    `json:"usdtPairIsRemoveLiquidity,omitempty"`
	UsdtPairIsMint            *bool    `json:"usdtPairIsMint,omitempty"`
	UsdtPairQuoteUsdtValueInt *big.Int `json:"usdtPairQuoteUsdtValueInt,omitempty"`
	UsdtPairLastSwapTimestamp *uint64  `json:"usdtPairLastSwapTimestamp,omitempty"`
}

type ResearchReportV1 struct {
	SchemaVersion      int32                  `json:"schemaVersion"`
	ProjectID          int64                  `json:"projectId"`
	CompletenessStatus string                 `json:"completenessStatus"`
	Observations       ResearchObservationsV1 `json:"observations"`
	Freshness          []ObservationFreshness `json:"freshness"`
	RiskSummary        ReportRiskSummary      `json:"riskSummary"`
}
