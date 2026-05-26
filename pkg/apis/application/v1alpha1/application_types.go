package v1alpha1

type ProjectView struct {
	Meta      ProjectMeta `protobuf:"bytes,1,opt,name=meta" json:"meta"`
	AveDetail AveDetail   `protobuf:"bytes,2,opt,name=aveDetail" json:"aveDetail"`
}

type ProjectListItem struct {
	Contract                string `protobuf:"bytes,1,opt,name=contract" json:"contract"`
	Name                    string `protobuf:"bytes,2,opt,name=name" json:"name"`
	Symbol                  string `protobuf:"bytes,3,opt,name=symbol" json:"symbol"`
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
	AveLogo                 string `protobuf:"bytes,16,opt,name=aveLogo" json:"aveLogo"`
	AveDetailAvailable      bool   `protobuf:"varint,17,opt,name=aveDetailAvailable" json:"aveDetailAvailable"`
	AveIsHoneypot           bool   `protobuf:"varint,18,opt,name=aveIsHoneypot" json:"aveIsHoneypot"`
	AveHasMintMethod        bool   `protobuf:"varint,19,opt,name=aveHasMintMethod" json:"aveHasMintMethod"`
	AveIsMintable           string `protobuf:"bytes,20,opt,name=aveIsMintable" json:"aveIsMintable"`
	AveHolders              int32  `protobuf:"varint,21,opt,name=aveHolders" json:"aveHolders"`
	AveMarketCap            string `protobuf:"bytes,22,opt,name=aveMarketCap" json:"aveMarketCap"`
}

type AveDetail struct {
	Status    int32          `protobuf:"varint,1,opt,name=status" json:"status"`
	Msg       string         `protobuf:"bytes,2,opt,name=msg" json:"msg"`
	DataType  int32          `protobuf:"varint,3,opt,name=dataType" json:"dataType"`
	IsAudited bool           `protobuf:"varint,4,opt,name=isAudited" json:"isAudited"`
	FetchedAt string         `protobuf:"bytes,5,opt,name=fetchedAt" json:"fetchedAt"`
	Token     AveTokenDetail `protobuf:"bytes,6,opt,name=token" json:"token"`
	Pairs     []AvePair      `protobuf:"bytes,7,rep,name=pairs" json:"pairs"`
}

type AveTokenDetail struct {
	Total               string `protobuf:"bytes,1,opt,name=total" json:"total"`
	LaunchPrice         string `protobuf:"bytes,2,opt,name=launchPrice" json:"launchPrice"`
	CurrentPriceETH     string `protobuf:"bytes,3,opt,name=currentPriceEth" json:"currentPriceEth"`
	CurrentPriceUSD     string `protobuf:"bytes,4,opt,name=currentPriceUsd" json:"currentPriceUsd"`
	PriceChange1D       string `protobuf:"bytes,5,opt,name=priceChange1d" json:"priceChange1d"`
	PriceChange24H      string `protobuf:"bytes,6,opt,name=priceChange24h" json:"priceChange24h"`
	PriceChange1H       string `protobuf:"bytes,7,opt,name=priceChange1h" json:"priceChange1h"`
	LockAmount          string `protobuf:"bytes,8,opt,name=lockAmount" json:"lockAmount"`
	BurnAmount          string `protobuf:"bytes,9,opt,name=burnAmount" json:"burnAmount"`
	OtherAmount         string `protobuf:"bytes,10,opt,name=otherAmount" json:"otherAmount"`
	TxAmount24H         string `protobuf:"bytes,11,opt,name=txAmount24h" json:"txAmount24h"`
	TxVolumeU24H        string `protobuf:"bytes,12,opt,name=txVolumeU24h" json:"txVolumeU24h"`
	LockedPercent       string `protobuf:"bytes,13,opt,name=lockedPercent" json:"lockedPercent"`
	MarketCap           string `protobuf:"bytes,14,opt,name=marketCap" json:"marketCap"`
	FDV                 string `protobuf:"bytes,15,opt,name=fdv" json:"fdv"`
	TVL                 string `protobuf:"bytes,16,opt,name=tvl" json:"tvl"`
	MainPairTVL         string `protobuf:"bytes,17,opt,name=mainPairTvl" json:"mainPairTvl"`
	TokenPriceChange5M  string `protobuf:"bytes,18,opt,name=tokenPriceChange5m" json:"tokenPriceChange5m"`
	TokenPriceChange1H  string `protobuf:"bytes,19,opt,name=tokenPriceChange1h" json:"tokenPriceChange1h"`
	TokenPriceChange4H  string `protobuf:"bytes,20,opt,name=tokenPriceChange4h" json:"tokenPriceChange4h"`
	TokenPriceChange24H string `protobuf:"bytes,21,opt,name=tokenPriceChange24h" json:"tokenPriceChange24h"`
	TokenTxVolumeUSD5M  string `protobuf:"bytes,22,opt,name=tokenTxVolumeUsd5m" json:"tokenTxVolumeUsd5m"`
	TokenTxVolumeUSD1H  string `protobuf:"bytes,23,opt,name=tokenTxVolumeUsd1h" json:"tokenTxVolumeUsd1h"`
	TokenTxVolumeUSD4H  string `protobuf:"bytes,24,opt,name=tokenTxVolumeUsd4h" json:"tokenTxVolumeUsd4h"`
	TokenTxVolumeUSD24H string `protobuf:"bytes,25,opt,name=tokenTxVolumeUsd24h" json:"tokenTxVolumeUsd24h"`
	TokenBuyVolumeU5M   string `protobuf:"bytes,26,opt,name=tokenBuyVolumeU5m" json:"tokenBuyVolumeU5m"`
	TokenSellVolumeU5M  string `protobuf:"bytes,27,opt,name=tokenSellVolumeU5m" json:"tokenSellVolumeU5m"`
	Token               string `protobuf:"bytes,28,opt,name=token" json:"token"`
	Chain               string `protobuf:"bytes,29,opt,name=chain" json:"chain"`
	Decimal             int32  `protobuf:"varint,30,opt,name=decimal" json:"decimal"`
	Name                string `protobuf:"bytes,31,opt,name=name" json:"name"`
	Symbol              string `protobuf:"bytes,32,opt,name=symbol" json:"symbol"`
	Holders             int32  `protobuf:"varint,33,opt,name=holders" json:"holders"`
	Appendix            string `protobuf:"bytes,34,opt,name=appendix" json:"appendix"`
	RiskLevel           int32  `protobuf:"varint,35,opt,name=riskLevel" json:"riskLevel"`
	LogoURL             string `protobuf:"bytes,36,opt,name=logoUrl" json:"logoUrl"`
	RiskInfo            string `protobuf:"bytes,37,opt,name=riskInfo" json:"riskInfo"`
	RiskScore           string `protobuf:"bytes,38,opt,name=riskScore" json:"riskScore"`
	LaunchAt            int64  `protobuf:"varint,39,opt,name=launchAt" json:"launchAt"`
	CreatedAt           int64  `protobuf:"varint,40,opt,name=createdAt" json:"createdAt"`
	TxCount24H          int32  `protobuf:"varint,41,opt,name=txCount24h" json:"txCount24h"`
	LockPlatform        string `protobuf:"bytes,42,opt,name=lockPlatform" json:"lockPlatform"`
	IsMintable          string `protobuf:"bytes,43,opt,name=isMintable" json:"isMintable"`
	UpdatedAt           int64  `protobuf:"varint,44,opt,name=updatedAt" json:"updatedAt"`
	MainPair            string `protobuf:"bytes,45,opt,name=mainPair" json:"mainPair"`
	HasMintMethod       bool   `protobuf:"varint,46,opt,name=hasMintMethod" json:"hasMintMethod"`
	IsLPNotLocked       bool   `protobuf:"varint,47,opt,name=isLpNotLocked" json:"isLpNotLocked"`
	HasNotRenounced     bool   `protobuf:"varint,48,opt,name=hasNotRenounced" json:"hasNotRenounced"`
	HasNotAudited       bool   `protobuf:"varint,49,opt,name=hasNotAudited" json:"hasNotAudited"`
	HasNotOpenSource    bool   `protobuf:"varint,50,opt,name=hasNotOpenSource" json:"hasNotOpenSource"`
	IsInBlacklist       bool   `protobuf:"varint,51,opt,name=isInBlacklist" json:"isInBlacklist"`
	IsHoneypot          bool   `protobuf:"varint,52,opt,name=isHoneypot" json:"isHoneypot"`
	AveRiskLevel        int32  `protobuf:"varint,53,opt,name=aveRiskLevel" json:"aveRiskLevel"`
}

type AvePair struct {
	Reserve0       string `protobuf:"bytes,1,opt,name=reserve0" json:"reserve0"`
	Reserve1       string `protobuf:"bytes,2,opt,name=reserve1" json:"reserve1"`
	Token0PriceETH string `protobuf:"bytes,3,opt,name=token0PriceEth" json:"token0PriceEth"`
	Token0PriceUSD string `protobuf:"bytes,4,opt,name=token0PriceUsd" json:"token0PriceUsd"`
	Token1PriceETH string `protobuf:"bytes,5,opt,name=token1PriceEth" json:"token1PriceEth"`
	Token1PriceUSD string `protobuf:"bytes,6,opt,name=token1PriceUsd" json:"token1PriceUsd"`
	PriceChange    string `protobuf:"bytes,7,opt,name=priceChange" json:"priceChange"`
	PriceChange24H string `protobuf:"bytes,8,opt,name=priceChange24h" json:"priceChange24h"`
	PriceChange1H  string `protobuf:"bytes,9,opt,name=priceChange1h" json:"priceChange1h"`
	VolumeU        string `protobuf:"bytes,10,opt,name=volumeU" json:"volumeU"`
	LowU           string `protobuf:"bytes,11,opt,name=lowU" json:"lowU"`
	HighU          string `protobuf:"bytes,12,opt,name=highU" json:"highU"`
	Fee            string `protobuf:"bytes,13,opt,name=fee" json:"fee"`
	TotalSupply    string `protobuf:"bytes,14,opt,name=totalSupply" json:"totalSupply"`
	TxAmount       string `protobuf:"bytes,15,opt,name=txAmount" json:"txAmount"`
	Pair           string `protobuf:"bytes,16,opt,name=pair" json:"pair"`
	Chain          string `protobuf:"bytes,17,opt,name=chain" json:"chain"`
	AMM            string `protobuf:"bytes,18,opt,name=amm" json:"amm"`
	Token0Address  string `protobuf:"bytes,19,opt,name=token0Address" json:"token0Address"`
	Token0Symbol   string `protobuf:"bytes,20,opt,name=token0Symbol" json:"token0Symbol"`
	Token0Decimal  int32  `protobuf:"varint,21,opt,name=token0Decimal" json:"token0Decimal"`
	Token1Address  string `protobuf:"bytes,22,opt,name=token1Address" json:"token1Address"`
	Token1Symbol   string `protobuf:"bytes,23,opt,name=token1Symbol" json:"token1Symbol"`
	Token1Decimal  int32  `protobuf:"varint,24,opt,name=token1Decimal" json:"token1Decimal"`
	TargetToken    string `protobuf:"bytes,25,opt,name=targetToken" json:"targetToken"`
	PriceChange1D  string `protobuf:"bytes,26,opt,name=priceChange1d" json:"priceChange1d"`
	CreatedAt      int64  `protobuf:"varint,27,opt,name=createdAt" json:"createdAt"`
	TxCount        int32  `protobuf:"varint,28,opt,name=txCount" json:"txCount"`
	UpdatedAt      int64  `protobuf:"varint,29,opt,name=updatedAt" json:"updatedAt"`
	MarketCap      string `protobuf:"bytes,30,opt,name=marketCap" json:"marketCap"`
	FDV            string `protobuf:"bytes,31,opt,name=fdv" json:"fdv"`
	IsFake         bool   `protobuf:"varint,32,opt,name=isFake" json:"isFake"`
}

type ProjectMeta struct {
	BlockTime                          uint64                    `protobuf:"varint,2,opt,name=blockTime" json:"blockTime"`
	BlockNumber                        uint64                    `protobuf:"varint,3,opt,name=blockNumber" json:"blockNumber"`
	Contract                           string                    `protobuf:"bytes,4,opt,name=contract" json:"contract"`
	Creator                            string                    `protobuf:"bytes,5,opt,name=creator" json:"creator"`
	TxHash                             string                    `protobuf:"bytes,6,opt,name=txHash" json:"txHash"`
	TxIndex                            uint64                    `protobuf:"varint,7,opt,name=txIndex" json:"txIndex"`
	SourceCode                         string                    `protobuf:"bytes,8,opt,name=sourceCode" json:"sourceCode"`
	CreatorResult                      SimulateResult            `protobuf:"bytes,9,opt,name=creatorResult" json:"creatorResult"`
	GenesisWallets                     []GenesisWalletState      `protobuf:"bytes,12,rep,name=genesisWallets" json:"genesisWallets"`
	CreatorHistoricalProjects          []string                  `protobuf:"bytes,13,rep,name=creatorHistoricalProjects" json:"creatorHistoricalProjects"`
	SourceQualityReport                string                    `protobuf:"bytes,14,opt,name=sourceQualityReport" json:"sourceQualityReport"`
	SourceQualityReportFetchedAt       string                    `protobuf:"bytes,15,opt,name=sourceQualityReportFetchedAt" json:"sourceQualityReportFetchedAt"`
	IsOpenSource                       bool                      `protobuf:"varint,16,opt,name=isOpenSource" json:"isOpenSource"`
	SourceCodeHash                     string                    `protobuf:"bytes,17,opt,name=sourceCodeHash" json:"sourceCodeHash"`
	CodeBinHash                        string                    `protobuf:"bytes,18,opt,name=codeBinHash" json:"codeBinHash"`
	SourceCodeFetchedAt                string                    `protobuf:"bytes,19,opt,name=sourceCodeFetchedAt" json:"sourceCodeFetchedAt"`
	CodeBinHashFetchedAt               string                    `protobuf:"bytes,20,opt,name=codeBinHashFetchedAt" json:"codeBinHashFetchedAt"`
	GenesisWalletsFetchedAt            string                    `protobuf:"bytes,21,opt,name=genesisWalletsFetchedAt" json:"genesisWalletsFetchedAt"`
	CreatorHistoricalProjectsFetchedAt string                    `protobuf:"bytes,22,opt,name=creatorHistoricalProjectsFetchedAt" json:"creatorHistoricalProjectsFetchedAt"`
	FetchAt                            string                    `protobuf:"bytes,23,opt,name=fetchAt" json:"fetchAt"`
	Token                              TokenState                `protobuf:"bytes,24,opt,name=token" json:"token"`
	WethPair                           PairV2State               `protobuf:"bytes,25,opt,name=wethPair" json:"wethPair"`
	UsdtPair                           PairV2State               `protobuf:"bytes,26,opt,name=usdtPair" json:"usdtPair"`
	AssetState                         AssetState                `protobuf:"bytes,27,opt,name=assetState" json:"assetState"`
	GenesisWalletAssetStates           []GenesisWalletAssetState `protobuf:"bytes,28,rep,name=genesisWalletAssetStates" json:"genesisWalletAssetStates"`
	SourceCodeOrigin                   string                    `protobuf:"bytes,29,opt,name=sourceCodeOrigin" json:"sourceCodeOrigin"`
	SourceQualityReportOrigin          string                    `protobuf:"bytes,30,opt,name=sourceQualityReportOrigin" json:"sourceQualityReportOrigin"`
}

type GenesisWalletState struct {
	Wallet    string `protobuf:"bytes,1,opt,name=wallet" json:"wallet"`
	NetAmount string `protobuf:"bytes,2,opt,name=netAmount" json:"netAmount"`
	RatioBps  int64  `protobuf:"varint,3,opt,name=ratioBps" json:"ratioBps"`
	Rank      int32  `protobuf:"varint,4,opt,name=rank" json:"rank"`
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
