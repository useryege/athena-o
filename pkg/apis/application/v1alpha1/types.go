package v1alpha1

type ProjectView struct {
	Meta       ProjectMeta       `protobuf:"bytes,1,opt,name=meta" json:"meta"`
	ChainState ProjectChainState `protobuf:"bytes,2,opt,name=chainState" json:"chainState"`
}

type ProjectListItem struct {
	Contract                string `protobuf:"bytes,1,opt,name=contract" json:"contract"`
	Name                    string `protobuf:"bytes,2,opt,name=name" json:"name"`
	Symbol                  string `protobuf:"bytes,3,opt,name=symbol" json:"symbol"`
	IsArchived              bool   `protobuf:"varint,4,opt,name=isArchived" json:"isArchived"`
	HasMintRisk             bool   `protobuf:"varint,6,opt,name=hasMintRisk" json:"hasMintRisk"`
	IsOpenSource            bool   `protobuf:"varint,7,opt,name=isOpenSource" json:"isOpenSource"`
	WethPairQuoteUsdtValue  string `protobuf:"bytes,8,opt,name=wethPairQuoteUsdtValue" json:"wethPairQuoteUsdtValue"`
	WethPairRemoveLiquidity bool   `protobuf:"varint,9,opt,name=wethPairRemoveLiquidity" json:"wethPairRemoveLiquidity"`
	UsdtPairQuoteUsdtValue  string `protobuf:"bytes,10,opt,name=usdtPairQuoteUsdtValue" json:"usdtPairQuoteUsdtValue"`
	UsdtPairRemoveLiquidity bool   `protobuf:"varint,11,opt,name=usdtPairRemoveLiquidity" json:"usdtPairRemoveLiquidity"`
	CreatorAssetUsdtValue   string `protobuf:"bytes,12,opt,name=creatorAssetUsdtValue" json:"creatorAssetUsdtValue"`
	BlockTime               uint64 `protobuf:"varint,13,opt,name=blockTime" json:"blockTime"`
	BlockNumber             uint64 `protobuf:"varint,14,opt,name=blockNumber" json:"blockNumber"`
	TxIndex                 uint64 `protobuf:"varint,15,opt,name=txIndex" json:"txIndex"`
}

type ProjectMeta struct {
	BlockTime                    uint64               `protobuf:"varint,2,opt,name=blockTime" json:"blockTime"`
	BlockNumber                  uint64               `protobuf:"varint,3,opt,name=blockNumber" json:"blockNumber"`
	Contract                     string               `protobuf:"bytes,4,opt,name=contract" json:"contract"`
	Creator                      string               `protobuf:"bytes,5,opt,name=creator" json:"creator"`
	TxHash                       string               `protobuf:"bytes,6,opt,name=txHash" json:"txHash"`
	TxIndex                      uint64               `protobuf:"varint,7,opt,name=txIndex" json:"txIndex"`
	SourceCode                   string               `protobuf:"bytes,8,opt,name=sourceCode" json:"sourceCode"`
	CreatorResult                SimulateResult       `protobuf:"bytes,9,opt,name=creatorResult" json:"creatorResult"`
	IsArchived                   bool                 `protobuf:"varint,11,opt,name=isArchived" json:"isArchived"`
	GenesisWallets               []GenesisWalletState `protobuf:"bytes,12,rep,name=genesisWallets" json:"genesisWallets"`
	CreatorOtherProjectContracts []string             `protobuf:"bytes,13,rep,name=creatorOtherProjectContracts" json:"creatorOtherProjectContracts"`
	SourceQualityReport          string               `protobuf:"bytes,14,opt,name=sourceQualityReport" json:"sourceQualityReport"`
	SourceQualityReportedAt      string               `protobuf:"bytes,15,opt,name=sourceQualityReportedAt" json:"sourceQualityReportedAt"`
	IsOpenSource                 bool                 `protobuf:"varint,16,opt,name=isOpenSource" json:"isOpenSource"`
	SourceCodeHash               string               `protobuf:"bytes,17,opt,name=sourceCodeHash" json:"sourceCodeHash"`
	CodeBinHash                  string               `protobuf:"bytes,18,opt,name=codeBinHash" json:"codeBinHash"`
}

type GenesisWalletState struct {
	Wallet    string `protobuf:"bytes,1,opt,name=wallet" json:"wallet"`
	NetAmount string `protobuf:"bytes,2,opt,name=netAmount" json:"netAmount"`
	RatioBps  int64  `protobuf:"varint,3,opt,name=ratioBps" json:"ratioBps"`
	Rank      int32  `protobuf:"varint,4,opt,name=rank" json:"rank"`
}

type ProjectChainState struct {
	Token                    TokenState                `protobuf:"bytes,1,opt,name=token" json:"token"`
	WethPair                 PairV2State               `protobuf:"bytes,2,opt,name=wethPair" json:"wethPair"`
	UsdtPair                 PairV2State               `protobuf:"bytes,3,opt,name=usdtPair" json:"usdtPair"`
	AssetState               AssetState                `protobuf:"bytes,4,opt,name=assetState" json:"assetState"`
	GenesisWalletAssetStates []GenesisWalletAssetState `protobuf:"bytes,5,rep,name=genesisWalletAssetStates" json:"genesisWalletAssetStates"`
}

type TokenState struct {
	Name         string `protobuf:"bytes,1,opt,name=name" json:"name"`
	Symbol       string `protobuf:"bytes,2,opt,name=symbol" json:"symbol"`
	Decimals     uint32 `protobuf:"varint,3,opt,name=decimals" json:"decimals"`
	TotalSupply  string `protobuf:"bytes,4,opt,name=totalSupply" json:"totalSupply"`
	IsValidERC20 bool   `protobuf:"varint,5,opt,name=isValidERC20" json:"isValidERC20"`
}

type PairV2State struct {
	IsCreated                      bool   `protobuf:"varint,1,opt,name=isCreated" json:"isCreated"`
	Contract                       string `protobuf:"bytes,2,opt,name=contract" json:"contract"`
	Token0                         string `protobuf:"bytes,3,opt,name=token0" json:"token0"`
	Token1                         string `protobuf:"bytes,4,opt,name=token1" json:"token1"`
	TotalSupply                    string `protobuf:"bytes,5,opt,name=totalSupply" json:"totalSupply"`
	Reserve0                       string `protobuf:"bytes,6,opt,name=reserve0" json:"reserve0"`
	Reserve1                       string `protobuf:"bytes,7,opt,name=reserve1" json:"reserve1"`
	BlockTimestampLast             uint32 `protobuf:"varint,8,opt,name=blockTimestampLast" json:"blockTimestampLast"`
	BaseBalance                    string `protobuf:"bytes,9,opt,name=baseBalance" json:"baseBalance"`
	QuoteBalance                   string `protobuf:"bytes,10,opt,name=quoteBalance" json:"quoteBalance"`
	QuoteUsdtValue                 string `protobuf:"bytes,11,opt,name=quoteUsdtValue" json:"quoteUsdtValue"`
	LockedLiquidity                string `protobuf:"bytes,12,opt,name=lockedLiquidity" json:"lockedLiquidity"`
	FeeAddressHoldLiquidityBalance string `protobuf:"bytes,13,opt,name=feeAddressHoldLiquidityBalance" json:"feeAddressHoldLiquidityBalance"`
	IsRemoveLiquidity              bool   `protobuf:"varint,14,opt,name=isRemoveLiquidity" json:"isRemoveLiquidity"`
	FeeAddressHoldLiquidityRatio   string `protobuf:"bytes,15,opt,name=feeAddressHoldLiquidityRatio" json:"feeAddressHoldLiquidityRatio"`
}

type AssetState struct {
	TokenBalance  string `protobuf:"bytes,1,opt,name=tokenBalance" json:"tokenBalance"`
	WethBalance   string `protobuf:"bytes,2,opt,name=wethBalance" json:"wethBalance"`
	UsdtBalance   string `protobuf:"bytes,3,opt,name=usdtBalance" json:"usdtBalance"`
	NativeBalance string `protobuf:"bytes,4,opt,name=nativeBalance" json:"nativeBalance"`
	UsdtValue     string `protobuf:"bytes,5,opt,name=usdtValue" json:"usdtValue"`
}

type GenesisWalletAssetState struct {
	Wallet     string     `protobuf:"bytes,1,opt,name=wallet" json:"wallet"`
	AssetState AssetState `protobuf:"bytes,2,opt,name=assetState" json:"assetState"`
}

type SimulateResult struct {
	CanMintFromDeadViaTransferFrom     bool `protobuf:"varint,1,opt,name=canMintFromDeadViaTransferFrom" json:"canMintFromDeadViaTransferFrom"`
	CanMintFromZeroViaTransferFrom     bool `protobuf:"varint,2,opt,name=canMintFromZeroViaTransferFrom" json:"canMintFromZeroViaTransferFrom"`
	CanMintFromWethPairViaTransferFrom bool `protobuf:"varint,3,opt,name=canMintFromWethPairViaTransferFrom" json:"canMintFromWethPairViaTransferFrom"`
	CanMintViaTransfer                 bool `protobuf:"varint,4,opt,name=canMintViaTransfer" json:"canMintViaTransfer"`
	CanMintFromUsdtPairViaTransferFrom bool `protobuf:"varint,5,opt,name=canMintFromUsdtPairViaTransferFrom" json:"canMintFromUsdtPairViaTransferFrom"`
	CanMintViaTransferToWethPair       bool `protobuf:"varint,6,opt,name=canMintViaTransferToWethPair" json:"canMintViaTransferToWethPair"`
	CanMintViaTransferToUsdtPair       bool `protobuf:"varint,7,opt,name=canMintViaTransferToUsdtPair" json:"canMintViaTransferToUsdtPair"`
}

type ProjectOption struct {
	FactoryContract string `protobuf:"bytes,1,opt,name=factoryContract" json:"factoryContract"`
	WethContract    string `protobuf:"bytes,2,opt,name=wethContract" json:"wethContract"`
	UsdtContract    string `protobuf:"bytes,3,opt,name=usdtContract" json:"usdtContract"`
	WethDecimals    uint32 `protobuf:"varint,4,opt,name=wethDecimals" json:"wethDecimals"`
	UsdtDecimals    uint32 `protobuf:"varint,5,opt,name=usdtDecimals" json:"usdtDecimals"`
}

type ProjectScope int32

const (
	ProjectScopeUnspecified ProjectScope = 0
	ProjectScopeActive      ProjectScope = 1
	ProjectScopeArchived    ProjectScope = 2
	ProjectScopeAll         ProjectScope = 3
)
