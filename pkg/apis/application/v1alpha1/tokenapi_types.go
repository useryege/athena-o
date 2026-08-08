package v1alpha1

type TokenContractCodeBlocklistEntry struct {
	CodeHash       string `protobuf:"bytes,1,opt,name=codeHash" json:"codeHash"`
	Note           string `protobuf:"bytes,2,opt,name=note" json:"note"`
	SourceChainID  int64  `protobuf:"varint,3,opt,name=sourceChainId" json:"sourceChainId"`
	SourceContract string `protobuf:"bytes,4,opt,name=sourceContract" json:"sourceContract"`
	CreatedAt      string `protobuf:"bytes,5,opt,name=createdAt" json:"createdAt"`
}

type TokenWalletBlocklistEntry struct {
	Wallet    string `protobuf:"bytes,1,opt,name=wallet" json:"wallet"`
	Note      string `protobuf:"bytes,2,opt,name=note" json:"note"`
	CreatedAt string `protobuf:"bytes,3,opt,name=createdAt" json:"createdAt"`
}

type TokenChainCheckpoint struct {
	ChainID           int64  `protobuf:"varint,1,opt,name=chainId" json:"chainId"`
	ChainName         string `protobuf:"bytes,2,opt,name=chainName" json:"chainName"`
	Enabled           bool   `protobuf:"varint,3,opt,name=enabled" json:"enabled"`
	CursorBlockNumber uint64 `protobuf:"varint,4,opt,name=cursorBlockNumber" json:"cursorBlockNumber"`
	Status            string `protobuf:"bytes,5,opt,name=status" json:"status"`
	CreatedAt         string `protobuf:"bytes,6,opt,name=createdAt" json:"createdAt"`
}

type TokenChain struct {
	ChainID   int64  `protobuf:"varint,1,opt,name=chainId" json:"chainId"`
	ChainName string `protobuf:"bytes,2,opt,name=chainName" json:"chainName"`
}

type TokenRuntimeConfiguration struct {
	Chains []TokenChain `protobuf:"bytes,1,rep,name=chains" json:"chains"`
}

type TokenNodeStatus struct {
	ChainID              int64  `protobuf:"varint,1,opt,name=chainId" json:"chainId"`
	ChainName            string `protobuf:"bytes,2,opt,name=chainName" json:"chainName"`
	Endpoint             string `protobuf:"bytes,3,opt,name=endpoint" json:"endpoint"`
	Available            bool   `protobuf:"varint,4,opt,name=available" json:"available"`
	LatencyMS            int64  `protobuf:"varint,5,opt,name=latencyMs" json:"latencyMs"`
	ReportedChainID      int64  `protobuf:"varint,6,opt,name=reportedChainId" json:"reportedChainId"`
	LatestBlockNumber    uint64 `protobuf:"varint,7,opt,name=latestBlockNumber" json:"latestBlockNumber"`
	CheckedAt            string `protobuf:"bytes,8,opt,name=checkedAt" json:"checkedAt"`
	Error                string `protobuf:"bytes,9,opt,name=error" json:"error"`
	ReferenceBlockNumber uint64 `protobuf:"varint,10,opt,name=referenceBlockNumber" json:"referenceBlockNumber"`
	BlockLag             uint64 `protobuf:"varint,11,opt,name=blockLag" json:"blockLag"`
	LatestBlockTime      string `protobuf:"bytes,12,opt,name=latestBlockTime" json:"latestBlockTime"`
	Syncing              bool   `protobuf:"varint,13,opt,name=syncing" json:"syncing"`
}

type TokenContractCode struct {
	CodeHash            string `protobuf:"bytes,1,opt,name=codeHash" json:"codeHash"`
	SourceCode          string `protobuf:"bytes,2,opt,name=sourceCode" json:"sourceCode"`
	SourceCodeFetchedAt string `protobuf:"bytes,3,opt,name=sourceCodeFetchedAt" json:"sourceCodeFetchedAt"`
	CreatedAt           string `protobuf:"bytes,4,opt,name=createdAt" json:"createdAt"`
	DeploymentCount     int64  `protobuf:"varint,5,opt,name=deploymentCount" json:"deploymentCount"`
}

type TokenProject struct {
	ProjectID   int64  `protobuf:"varint,1,opt,name=projectId" json:"projectId"`
	ChainID     int64  `protobuf:"varint,2,opt,name=chainId" json:"chainId"`
	Name        string `protobuf:"bytes,3,opt,name=name" json:"name"`
	Symbol      string `protobuf:"bytes,4,opt,name=symbol" json:"symbol"`
	Contract    string `protobuf:"bytes,5,opt,name=contract" json:"contract"`
	TxSender    string `protobuf:"bytes,6,opt,name=txSender" json:"txSender"`
	TxHash      string `protobuf:"bytes,7,opt,name=txHash" json:"txHash"`
	TxIndex     uint64 `protobuf:"varint,8,opt,name=txIndex" json:"txIndex"`
	BlockNumber uint64 `protobuf:"varint,9,opt,name=blockNumber" json:"blockNumber"`
	BlockTime   uint64 `protobuf:"varint,10,opt,name=blockTime" json:"blockTime"`
	CodeHash    string `protobuf:"bytes,11,opt,name=codeHash" json:"codeHash"`
	CreatedAt   string `protobuf:"bytes,12,opt,name=createdAt" json:"createdAt"`
	Decimals    int32  `protobuf:"varint,13,opt,name=decimals" json:"decimals"`
	TotalSupply string `protobuf:"bytes,14,opt,name=totalSupply" json:"totalSupply"`
	WethPair    string `protobuf:"bytes,15,opt,name=wethPair" json:"wethPair"`
	UsdtPair    string `protobuf:"bytes,16,opt,name=usdtPair" json:"usdtPair"`
}

type TokenProjectListItem struct {
	Project        *TokenProject              `protobuf:"bytes,1,opt,name=project" json:"project"`
	ResearchStatus string                     `protobuf:"bytes,2,opt,name=researchStatus" json:"researchStatus"`
	CurrentReport  *TokenProjectReportSummary `protobuf:"bytes,3,opt,name=currentReport" json:"currentReport"`
}

type TokenProjectReportSummary struct {
	Revision           int64                                `protobuf:"varint,1,opt,name=revision" json:"revision"`
	CompletenessStatus string                               `protobuf:"bytes,2,opt,name=completenessStatus" json:"completenessStatus"`
	BuiltAt            string                               `protobuf:"bytes,3,opt,name=builtAt" json:"builtAt"`
	RiskSummary        *TokenProjectReportRiskSummary       `protobuf:"bytes,4,opt,name=riskSummary" json:"riskSummary"`
	Evaluation         *TokenProjectReportEvaluationSummary `protobuf:"bytes,5,opt,name=evaluation" json:"evaluation"`
}

type TokenProjectReportRiskSummary struct {
	WethPair *TokenProjectPairRiskSummary `protobuf:"bytes,1,opt,name=wethPair" json:"wethPair"`
	UsdtPair *TokenProjectPairRiskSummary `protobuf:"bytes,2,opt,name=usdtPair" json:"usdtPair"`
}

type TokenProjectPairRiskSummary struct {
	IsCreated         bool   `protobuf:"varint,1,opt,name=isCreated" json:"isCreated"`
	IsRemoveLiquidity bool   `protobuf:"varint,2,opt,name=isRemoveLiquidity" json:"isRemoveLiquidity"`
	IsMint            bool   `protobuf:"varint,3,opt,name=isMint" json:"isMint"`
	QuoteUsdtValueInt string `protobuf:"bytes,4,opt,name=quoteUsdtValueInt" json:"quoteUsdtValueInt"`
	LastSwapAt        string `protobuf:"bytes,5,opt,name=lastSwapAt" json:"lastSwapAt"`
}

type TokenProjectReportEvaluationSummary struct {
	Status         string `protobuf:"bytes,1,opt,name=status" json:"status"`
	FailedAttempts int32  `protobuf:"varint,2,opt,name=failedAttempts" json:"failedAttempts"`
	LastError      string `protobuf:"bytes,3,opt,name=lastError" json:"lastError"`
	UpdatedAt      string `protobuf:"bytes,4,opt,name=updatedAt" json:"updatedAt"`
	Outcome        string `protobuf:"bytes,5,opt,name=outcome" json:"outcome"`
	EvaluatedAt    string `protobuf:"bytes,6,opt,name=evaluatedAt" json:"evaluatedAt"`
}

type TokenCollectionTask struct {
	TaskID         int64  `protobuf:"varint,1,opt,name=taskId" json:"taskId"`
	ProjectID      int64  `protobuf:"varint,2,opt,name=projectId" json:"projectId"`
	DataType       string `protobuf:"bytes,3,opt,name=dataType" json:"dataType"`
	Status         string `protobuf:"bytes,4,opt,name=status" json:"status"`
	Revision       int64  `protobuf:"varint,5,opt,name=revision" json:"revision"`
	Attempts       int32  `protobuf:"varint,6,opt,name=attempts" json:"attempts"`
	AvailableAt    string `protobuf:"bytes,7,opt,name=availableAt" json:"availableAt"`
	LeaseExpiresAt string `protobuf:"bytes,8,opt,name=leaseExpiresAt" json:"leaseExpiresAt"`
	LastError      string `protobuf:"bytes,9,opt,name=lastError" json:"lastError"`
	CreatedAt      string `protobuf:"bytes,10,opt,name=createdAt" json:"createdAt"`
	UpdatedAt      string `protobuf:"bytes,11,opt,name=updatedAt" json:"updatedAt"`
}

type TokenResearchState struct {
	ProjectID               int64  `protobuf:"varint,1,opt,name=projectId" json:"projectId"`
	ChainID                 int64  `protobuf:"varint,2,opt,name=chainId" json:"chainId"`
	Contract                string `protobuf:"bytes,3,opt,name=contract" json:"contract"`
	Status                  string `protobuf:"bytes,4,opt,name=status" json:"status"`
	EvidenceRevision        int64  `protobuf:"varint,5,opt,name=evidenceRevision" json:"evidenceRevision"`
	CurrentReportRevision   int64  `protobuf:"varint,6,opt,name=currentReportRevision" json:"currentReportRevision"`
	CurrentSelectionOutcome string `protobuf:"bytes,7,opt,name=currentSelectionOutcome" json:"currentSelectionOutcome"`
	LastEvaluatedRevision   int64  `protobuf:"varint,8,opt,name=lastEvaluatedRevision" json:"lastEvaluatedRevision"`
	LastEvaluatedAt         string `protobuf:"bytes,9,opt,name=lastEvaluatedAt" json:"lastEvaluatedAt"`
	ExpiresAt               string `protobuf:"bytes,10,opt,name=expiresAt" json:"expiresAt"`
	CreatedAt               string `protobuf:"bytes,11,opt,name=createdAt" json:"createdAt"`
	UpdatedAt               string `protobuf:"bytes,12,opt,name=updatedAt" json:"updatedAt"`
}

type TokenReportRevision struct {
	ReportRevisionID    int64                          `protobuf:"varint,1,opt,name=reportRevisionId" json:"reportRevisionId"`
	ProjectID           int64                          `protobuf:"varint,2,opt,name=projectId" json:"projectId"`
	ChainID             int64                          `protobuf:"varint,3,opt,name=chainId" json:"chainId"`
	Contract            string                         `protobuf:"bytes,4,opt,name=contract" json:"contract"`
	Revision            int64                          `protobuf:"varint,5,opt,name=revision" json:"revision"`
	ContentHash         string                         `protobuf:"bytes,6,opt,name=contentHash" json:"contentHash"`
	CompletenessStatus  string                         `protobuf:"bytes,7,opt,name=completenessStatus" json:"completenessStatus"`
	EvidenceJSON        string                         `protobuf:"bytes,8,opt,name=evidenceJson" json:"evidenceJson"`
	ReportJSON          string                         `protobuf:"bytes,9,opt,name=reportJson" json:"reportJson"`
	ObservedBlockNumber uint64                         `protobuf:"varint,10,opt,name=observedBlockNumber" json:"observedBlockNumber"`
	RiskSummary         *TokenProjectReportRiskSummary `protobuf:"bytes,11,opt,name=riskSummary" json:"riskSummary"`
	BuiltAt             string                         `protobuf:"bytes,21,opt,name=builtAt" json:"builtAt"`
	CreatedAt           string                         `protobuf:"bytes,22,opt,name=createdAt" json:"createdAt"`
}

type TokenSelection struct {
	SelectionID     int64    `protobuf:"varint,1,opt,name=selectionId" json:"selectionId"`
	ProjectID       int64    `protobuf:"varint,2,opt,name=projectId" json:"projectId"`
	ChainID         int64    `protobuf:"varint,3,opt,name=chainId" json:"chainId"`
	Contract        string   `protobuf:"bytes,4,opt,name=contract" json:"contract"`
	Outcome         string   `protobuf:"bytes,5,opt,name=outcome" json:"outcome"`
	StrategyKey     string   `protobuf:"bytes,6,opt,name=strategyKey" json:"strategyKey"`
	StrategyVersion string   `protobuf:"bytes,7,opt,name=strategyVersion" json:"strategyVersion"`
	ReportRevision  int64    `protobuf:"varint,8,opt,name=reportRevision" json:"reportRevision"`
	ReasonCodes     []string `protobuf:"bytes,9,rep,name=reasonCodes" json:"reasonCodes"`
	ReasonDetail    string   `protobuf:"bytes,10,opt,name=reasonDetail" json:"reasonDetail"`
	DecidedAt       string   `protobuf:"bytes,11,opt,name=decidedAt" json:"decidedAt"`
	CreatedAt       string   `protobuf:"bytes,12,opt,name=createdAt" json:"createdAt"`
}

type TokenProjectObservation struct {
	ObservationID int64  `protobuf:"varint,1,opt,name=observationId" json:"observationId"`
	ProjectID     int64  `protobuf:"varint,2,opt,name=projectId" json:"projectId"`
	DataType      string `protobuf:"bytes,3,opt,name=dataType" json:"dataType"`
	SchemaVersion int32  `protobuf:"varint,4,opt,name=schemaVersion" json:"schemaVersion"`
	ContentHash   string `protobuf:"bytes,5,opt,name=contentHash" json:"contentHash"`
	PayloadJSON   string `protobuf:"bytes,6,opt,name=payloadJson" json:"payloadJson"`
	BlockNumber   uint64 `protobuf:"varint,7,opt,name=blockNumber" json:"blockNumber"`
	ObservedAt    string `protobuf:"bytes,8,opt,name=observedAt" json:"observedAt"`
	LastCheckedAt string `protobuf:"bytes,9,opt,name=lastCheckedAt" json:"lastCheckedAt"`
	CreatedAt     string `protobuf:"bytes,10,opt,name=createdAt" json:"createdAt"`
}

type TokenAveToken struct {
	Address          string `protobuf:"bytes,1,opt,name=address" json:"address"`
	Name             string `protobuf:"bytes,2,opt,name=name" json:"name"`
	Symbol           string `protobuf:"bytes,3,opt,name=symbol" json:"symbol"`
	Decimals         int32  `protobuf:"varint,4,opt,name=decimals" json:"decimals"`
	TotalSupply      string `protobuf:"bytes,5,opt,name=totalSupply" json:"totalSupply"`
	CurrentPriceUSD  string `protobuf:"bytes,6,opt,name=currentPriceUsd" json:"currentPriceUsd"`
	CurrentPriceETH  string `protobuf:"bytes,7,opt,name=currentPriceEth" json:"currentPriceEth"`
	MarketCap        string `protobuf:"bytes,8,opt,name=marketCap" json:"marketCap"`
	FDV              string `protobuf:"bytes,9,opt,name=fdv" json:"fdv"`
	TVL              string `protobuf:"bytes,10,opt,name=tvl" json:"tvl"`
	MainPairTVL      string `protobuf:"bytes,11,opt,name=mainPairTvl" json:"mainPairTvl"`
	Holders          int32  `protobuf:"varint,12,opt,name=holders" json:"holders"`
	RiskLevel        int32  `protobuf:"varint,13,opt,name=riskLevel" json:"riskLevel"`
	RiskScore        string `protobuf:"bytes,14,opt,name=riskScore" json:"riskScore"`
	RiskInfo         string `protobuf:"bytes,15,opt,name=riskInfo" json:"riskInfo"`
	IsMintableKnown  bool   `protobuf:"varint,16,opt,name=isMintableKnown" json:"isMintableKnown"`
	IsMintable       bool   `protobuf:"varint,17,opt,name=isMintable" json:"isMintable"`
	HasMintMethod    bool   `protobuf:"varint,18,opt,name=hasMintMethod" json:"hasMintMethod"`
	IsLPNotLocked    bool   `protobuf:"varint,19,opt,name=isLpNotLocked" json:"isLpNotLocked"`
	HasNotRenounced  bool   `protobuf:"varint,20,opt,name=hasNotRenounced" json:"hasNotRenounced"`
	HasNotAudited    bool   `protobuf:"varint,21,opt,name=hasNotAudited" json:"hasNotAudited"`
	HasNotOpenSource bool   `protobuf:"varint,22,opt,name=hasNotOpenSource" json:"hasNotOpenSource"`
	IsInBlacklist    bool   `protobuf:"varint,23,opt,name=isInBlacklist" json:"isInBlacklist"`
	IsHoneypot       bool   `protobuf:"varint,24,opt,name=isHoneypot" json:"isHoneypot"`
	LaunchAt         string `protobuf:"bytes,25,opt,name=launchAt" json:"launchAt"`
	UpdatedAt        string `protobuf:"bytes,26,opt,name=updatedAt" json:"updatedAt"`
}

type TokenAvePair struct {
	Pair          string `protobuf:"bytes,1,opt,name=pair" json:"pair"`
	ChainID       int64  `protobuf:"varint,2,opt,name=chainId" json:"chainId"`
	AMM           string `protobuf:"bytes,3,opt,name=amm" json:"amm"`
	Token0Address string `protobuf:"bytes,4,opt,name=token0Address" json:"token0Address"`
	Token0Symbol  string `protobuf:"bytes,5,opt,name=token0Symbol" json:"token0Symbol"`
	Token1Address string `protobuf:"bytes,6,opt,name=token1Address" json:"token1Address"`
	Token1Symbol  string `protobuf:"bytes,7,opt,name=token1Symbol" json:"token1Symbol"`
	Reserve0      string `protobuf:"bytes,8,opt,name=reserve0" json:"reserve0"`
	Reserve1      string `protobuf:"bytes,9,opt,name=reserve1" json:"reserve1"`
	VolumeUSD     string `protobuf:"bytes,10,opt,name=volumeUsd" json:"volumeUsd"`
	MarketCap     string `protobuf:"bytes,11,opt,name=marketCap" json:"marketCap"`
	FDV           string `protobuf:"bytes,12,opt,name=fdv" json:"fdv"`
	IsFake        bool   `protobuf:"varint,13,opt,name=isFake" json:"isFake"`
	CreatedAt     string `protobuf:"bytes,14,opt,name=createdAt" json:"createdAt"`
	UpdatedAt     string `protobuf:"bytes,15,opt,name=updatedAt" json:"updatedAt"`
}

type TokenAveObservation struct {
	ChainID   int64           `protobuf:"varint,1,opt,name=chainId" json:"chainId"`
	Token     *TokenAveToken  `protobuf:"bytes,2,opt,name=token" json:"token"`
	Pairs     []*TokenAvePair `protobuf:"bytes,3,rep,name=pairs" json:"pairs"`
	IsAudited bool            `protobuf:"varint,4,opt,name=isAudited" json:"isAudited"`
}

type TokenChainToken struct {
	IsValidERC20 bool   `protobuf:"varint,1,opt,name=isValidErc20" json:"isValidErc20"`
	Name         string `protobuf:"bytes,2,opt,name=name" json:"name"`
	Symbol       string `protobuf:"bytes,3,opt,name=symbol" json:"symbol"`
	Decimals     int32  `protobuf:"varint,4,opt,name=decimals" json:"decimals"`
	TotalSupply  string `protobuf:"bytes,5,opt,name=totalSupply" json:"totalSupply"`
	WethPair     string `protobuf:"bytes,6,opt,name=wethPair" json:"wethPair"`
	UsdtPair     string `protobuf:"bytes,7,opt,name=usdtPair" json:"usdtPair"`
}

type TokenChainPairLiquidity struct {
	TotalSupply                    string `protobuf:"bytes,1,opt,name=totalSupply" json:"totalSupply"`
	LockedLiquidity                string `protobuf:"bytes,2,opt,name=lockedLiquidity" json:"lockedLiquidity"`
	FeeAddressHoldLiquidityBalance string `protobuf:"bytes,3,opt,name=feeAddressHoldLiquidityBalance" json:"feeAddressHoldLiquidityBalance"`
	FeeAddressHoldLiquidityRatio   string `protobuf:"bytes,4,opt,name=feeAddressHoldLiquidityRatio" json:"feeAddressHoldLiquidityRatio"`
}

type TokenChainPair struct {
	PairContract      string                   `protobuf:"bytes,1,opt,name=pairContract" json:"pairContract"`
	IsCreated         bool                     `protobuf:"varint,2,opt,name=isCreated" json:"isCreated"`
	Liquidity         *TokenChainPairLiquidity `protobuf:"bytes,3,opt,name=liquidity" json:"liquidity"`
	BaseBalance       string                   `protobuf:"bytes,4,opt,name=baseBalance" json:"baseBalance"`
	QuoteBalance      string                   `protobuf:"bytes,5,opt,name=quoteBalance" json:"quoteBalance"`
	QuoteUsdtValue    string                   `protobuf:"bytes,6,opt,name=quoteUsdtValue" json:"quoteUsdtValue"`
	QuoteUsdtValueInt string                   `protobuf:"bytes,7,opt,name=quoteUsdtValueInt" json:"quoteUsdtValueInt"`
	LastSwapAt        string                   `protobuf:"bytes,8,opt,name=lastSwapAt" json:"lastSwapAt"`
	IsRemoveLiquidity bool                     `protobuf:"varint,9,opt,name=isRemoveLiquidity" json:"isRemoveLiquidity"`
	IsMint            bool                     `protobuf:"varint,10,opt,name=isMint" json:"isMint"`
}

type TokenChainStateObservation struct {
	TokenContract string           `protobuf:"bytes,1,opt,name=tokenContract" json:"tokenContract"`
	UpdatedAt     string           `protobuf:"bytes,2,opt,name=updatedAt" json:"updatedAt"`
	Token         *TokenChainToken `protobuf:"bytes,3,opt,name=token" json:"token"`
	IsValidERC20  bool             `protobuf:"varint,4,opt,name=isValidErc20" json:"isValidErc20"`
	WethPair      *TokenChainPair  `protobuf:"bytes,5,opt,name=wethPair" json:"wethPair"`
	UsdtPair      *TokenChainPair  `protobuf:"bytes,6,opt,name=usdtPair" json:"usdtPair"`
}

type TokenWalletAssetState struct {
	ChainID             int64  `protobuf:"varint,1,opt,name=chainId" json:"chainId"`
	Wallet              string `protobuf:"bytes,2,opt,name=wallet" json:"wallet"`
	WethBalance         string `protobuf:"bytes,3,opt,name=wethBalance" json:"wethBalance"`
	UsdtBalance         string `protobuf:"bytes,4,opt,name=usdtBalance" json:"usdtBalance"`
	NativeBalance       string `protobuf:"bytes,5,opt,name=nativeBalance" json:"nativeBalance"`
	TotalAssetUsdtValue string `protobuf:"bytes,6,opt,name=totalAssetUsdtValue" json:"totalAssetUsdtValue"`
}

type TokenSimulationResult struct {
	ProjectID                          int64  `protobuf:"varint,1,opt,name=projectId" json:"projectId"`
	Wallet                             string `protobuf:"bytes,2,opt,name=wallet" json:"wallet"`
	CanMintFromDeadViaTransferFrom     bool   `protobuf:"varint,3,opt,name=canMintFromDeadViaTransferFrom" json:"canMintFromDeadViaTransferFrom"`
	CanMintFromZeroViaTransferFrom     bool   `protobuf:"varint,4,opt,name=canMintFromZeroViaTransferFrom" json:"canMintFromZeroViaTransferFrom"`
	CanMintFromWethPairViaTransferFrom bool   `protobuf:"varint,5,opt,name=canMintFromWethPairViaTransferFrom" json:"canMintFromWethPairViaTransferFrom"`
	CanMintFromUsdtPairViaTransferFrom bool   `protobuf:"varint,6,opt,name=canMintFromUsdtPairViaTransferFrom" json:"canMintFromUsdtPairViaTransferFrom"`
	CanMintViaTransferToWethPair       bool   `protobuf:"varint,7,opt,name=canMintViaTransferToWethPair" json:"canMintViaTransferToWethPair"`
	CanMintViaTransferToUsdtPair       bool   `protobuf:"varint,8,opt,name=canMintViaTransferToUsdtPair" json:"canMintViaTransferToUsdtPair"`
}

type TokenContractSourceObservation struct {
	CodeHash        string `protobuf:"bytes,1,opt,name=codeHash" json:"codeHash"`
	SourceAvailable bool   `protobuf:"varint,2,opt,name=sourceAvailable" json:"sourceAvailable"`
}

type TokenProjectRelatedWallet struct {
	ProjectID int64  `protobuf:"varint,1,opt,name=projectId" json:"projectId"`
	Wallet    string `protobuf:"bytes,2,opt,name=wallet" json:"wallet"`
	Role      string `protobuf:"bytes,3,opt,name=role" json:"role"`
	CreatedAt string `protobuf:"bytes,4,opt,name=createdAt" json:"createdAt"`
}

type TokenProjectInitialRecipient struct {
	RecipientID       int64  `protobuf:"varint,1,opt,name=recipientId" json:"recipientId"`
	ProjectID         int64  `protobuf:"varint,2,opt,name=projectId" json:"projectId"`
	Wallet            string `protobuf:"bytes,3,opt,name=wallet" json:"wallet"`
	RatioBPS          int64  `protobuf:"varint,4,opt,name=ratioBps" json:"ratioBps"`
	RankIndex         int32  `protobuf:"varint,5,opt,name=rankIndex" json:"rankIndex"`
	SourceTxHash      string `protobuf:"bytes,6,opt,name=sourceTxHash" json:"sourceTxHash"`
	SourceBlockNumber uint64 `protobuf:"varint,7,opt,name=sourceBlockNumber" json:"sourceBlockNumber"`
	CreatedAt         string `protobuf:"bytes,8,opt,name=createdAt" json:"createdAt"`
}

type TokenCollectionSchedule struct {
	ProjectID           int64  `protobuf:"varint,1,opt,name=projectId" json:"projectId"`
	DataType            string `protobuf:"bytes,2,opt,name=dataType" json:"dataType"`
	Status              string `protobuf:"bytes,3,opt,name=status" json:"status"`
	RetryIntervalSecs   int64  `protobuf:"varint,4,opt,name=retryIntervalSecs" json:"retryIntervalSecs"`
	NextRunAt           string `protobuf:"bytes,5,opt,name=nextRunAt" json:"nextRunAt"`
	LatestTaskRevision  int64  `protobuf:"varint,6,opt,name=latestTaskRevision" json:"latestTaskRevision"`
	ConsecutiveFailures int32  `protobuf:"varint,7,opt,name=consecutiveFailures" json:"consecutiveFailures"`
	LastError           string `protobuf:"bytes,8,opt,name=lastError" json:"lastError"`
	LastCheckedAt       string `protobuf:"bytes,9,opt,name=lastCheckedAt" json:"lastCheckedAt"`
	CreatedAt           string `protobuf:"bytes,10,opt,name=createdAt" json:"createdAt"`
	UpdatedAt           string `protobuf:"bytes,11,opt,name=updatedAt" json:"updatedAt"`
}

type TokenWalletTransactionCount struct {
	Wallet           string `protobuf:"bytes,1,opt,name=wallet" json:"wallet"`
	TransactionCount int64  `protobuf:"varint,2,opt,name=transactionCount" json:"transactionCount"`
}

type TokenProjectDetail struct {
	Project                 *TokenProject                        `protobuf:"bytes,1,opt,name=project" json:"project"`
	ResearchState           *TokenResearchState                  `protobuf:"bytes,2,opt,name=researchState" json:"researchState"`
	CurrentReport           *TokenReportRevision                 `protobuf:"bytes,3,opt,name=currentReport" json:"currentReport"`
	CurrentSelection        *TokenSelection                      `protobuf:"bytes,4,opt,name=currentSelection" json:"currentSelection"`
	Ave                     *TokenAveObservation                 `protobuf:"bytes,5,opt,name=ave" json:"ave"`
	ChainState              *TokenChainStateObservation          `protobuf:"bytes,6,opt,name=chainState" json:"chainState"`
	WalletAssets            []*TokenWalletAssetState             `protobuf:"bytes,7,rep,name=walletAssets" json:"walletAssets"`
	Simulations             []*TokenSimulationResult             `protobuf:"bytes,8,rep,name=simulations" json:"simulations"`
	ContractSource          *TokenContractSourceObservation      `protobuf:"bytes,9,opt,name=contractSource" json:"contractSource"`
	RelatedWallets          []*TokenProjectRelatedWallet         `protobuf:"bytes,10,rep,name=relatedWallets" json:"relatedWallets"`
	InitialRecipients       []*TokenProjectInitialRecipient      `protobuf:"bytes,11,rep,name=initialRecipients" json:"initialRecipients"`
	CollectionSchedules     []*TokenCollectionSchedule           `protobuf:"bytes,12,rep,name=collectionSchedules" json:"collectionSchedules"`
	WalletTransactionCounts []*TokenWalletTransactionCount       `protobuf:"bytes,13,rep,name=walletTransactionCounts" json:"walletTransactionCounts"`
	TransactionCount        int64                                `protobuf:"varint,14,opt,name=transactionCount" json:"transactionCount"`
	CurrentObservations     []*TokenProjectObservation           `protobuf:"bytes,15,rep,name=currentObservations" json:"currentObservations"`
	GeneratedAt             string                               `protobuf:"bytes,16,opt,name=generatedAt" json:"generatedAt"`
	CurrentReportEvaluation *TokenProjectReportEvaluationSummary `protobuf:"bytes,17,opt,name=currentReportEvaluation" json:"currentReportEvaluation"`
}

type TokenProjectSwapAsset struct {
	Address    string `protobuf:"bytes,1,opt,name=address" json:"address"`
	Symbol     string `protobuf:"bytes,2,opt,name=symbol" json:"symbol"`
	Decimals   int32  `protobuf:"varint,3,opt,name=decimals" json:"decimals"`
	TokenIndex int32  `protobuf:"varint,4,opt,name=tokenIndex" json:"tokenIndex"`
}

type TokenProjectSwapBlock struct {
	SampleIndex            int32  `protobuf:"varint,1,opt,name=sampleIndex" json:"sampleIndex"`
	BlockNumber            string `protobuf:"bytes,2,opt,name=blockNumber" json:"blockNumber"`
	BlockTime              string `protobuf:"bytes,3,opt,name=blockTime" json:"blockTime"`
	EventCount             int64  `protobuf:"varint,4,opt,name=eventCount" json:"eventCount"`
	TransactionCount       int64  `protobuf:"varint,5,opt,name=transactionCount" json:"transactionCount"`
	TransactionOriginCount int64  `protobuf:"varint,6,opt,name=transactionOriginCount" json:"transactionOriginCount"`
	BuyEventCount          int64  `protobuf:"varint,7,opt,name=buyEventCount" json:"buyEventCount"`
	SellEventCount         int64  `protobuf:"varint,8,opt,name=sellEventCount" json:"sellEventCount"`
	ComplexEventCount      int64  `protobuf:"varint,9,opt,name=complexEventCount" json:"complexEventCount"`
	PreviousBlockGap       string `protobuf:"bytes,10,opt,name=previousBlockGap" json:"previousBlockGap"`
	PreviousTimeGapSeconds string `protobuf:"bytes,11,opt,name=previousTimeGapSeconds" json:"previousTimeGapSeconds"`
	BaseAmountInRaw        string `protobuf:"bytes,12,opt,name=baseAmountInRaw" json:"baseAmountInRaw"`
	BaseAmountOutRaw       string `protobuf:"bytes,13,opt,name=baseAmountOutRaw" json:"baseAmountOutRaw"`
	QuoteAmountInRaw       string `protobuf:"bytes,14,opt,name=quoteAmountInRaw" json:"quoteAmountInRaw"`
	QuoteAmountOutRaw      string `protobuf:"bytes,15,opt,name=quoteAmountOutRaw" json:"quoteAmountOutRaw"`
	BuyQuoteAmountRaw      string `protobuf:"bytes,16,opt,name=buyQuoteAmountRaw" json:"buyQuoteAmountRaw"`
	SellQuoteAmountRaw     string `protobuf:"bytes,17,opt,name=sellQuoteAmountRaw" json:"sellQuoteAmountRaw"`
	OpenPrice              string `protobuf:"bytes,18,opt,name=openPrice" json:"openPrice"`
	HighPrice              string `protobuf:"bytes,19,opt,name=highPrice" json:"highPrice"`
	LowPrice               string `protobuf:"bytes,20,opt,name=lowPrice" json:"lowPrice"`
	ClosePrice             string `protobuf:"bytes,21,opt,name=closePrice" json:"closePrice"`
	VWAP                   string `protobuf:"bytes,22,opt,name=vwap" json:"vwap"`
}

type TokenProjectSwapPairActivity struct {
	PairKind                string                   `protobuf:"bytes,1,opt,name=pairKind" json:"pairKind"`
	PairAddress             string                   `protobuf:"bytes,2,opt,name=pairAddress" json:"pairAddress"`
	Status                  string                   `protobuf:"bytes,3,opt,name=status" json:"status"`
	SwapBlockCount          int32                    `protobuf:"varint,4,opt,name=swapBlockCount" json:"swapBlockCount"`
	TargetSwapBlockCount    int32                    `protobuf:"varint,5,opt,name=targetSwapBlockCount" json:"targetSwapBlockCount"`
	StartBlockNumber        string                   `protobuf:"bytes,6,opt,name=startBlockNumber" json:"startBlockNumber"`
	StartBlockTime          string                   `protobuf:"bytes,7,opt,name=startBlockTime" json:"startBlockTime"`
	FirstSwapBlockNumber    string                   `protobuf:"bytes,8,opt,name=firstSwapBlockNumber" json:"firstSwapBlockNumber"`
	FirstSwapBlockTime      string                   `protobuf:"bytes,9,opt,name=firstSwapBlockTime" json:"firstSwapBlockTime"`
	LastSwapBlockNumber     string                   `protobuf:"bytes,10,opt,name=lastSwapBlockNumber" json:"lastSwapBlockNumber"`
	LastSwapBlockTime       string                   `protobuf:"bytes,11,opt,name=lastSwapBlockTime" json:"lastSwapBlockTime"`
	AbsoluteExpiryBlockTime string                   `protobuf:"bytes,12,opt,name=absoluteExpiryBlockTime" json:"absoluteExpiryBlockTime"`
	NextExpiryBlockTime     string                   `protobuf:"bytes,13,opt,name=nextExpiryBlockTime" json:"nextExpiryBlockTime"`
	CompletedBlockNumber    string                   `protobuf:"bytes,14,opt,name=completedBlockNumber" json:"completedBlockNumber"`
	CompletedBlockTime      string                   `protobuf:"bytes,15,opt,name=completedBlockTime" json:"completedBlockTime"`
	ExpiredBlockNumber      string                   `protobuf:"bytes,16,opt,name=expiredBlockNumber" json:"expiredBlockNumber"`
	ExpiredBlockTime        string                   `protobuf:"bytes,17,opt,name=expiredBlockTime" json:"expiredBlockTime"`
	ExpiredReason           string                   `protobuf:"bytes,18,opt,name=expiredReason" json:"expiredReason"`
	BaseAsset               *TokenProjectSwapAsset   `protobuf:"bytes,19,opt,name=baseAsset" json:"baseAsset"`
	QuoteAsset              *TokenProjectSwapAsset   `protobuf:"bytes,20,opt,name=quoteAsset" json:"quoteAsset"`
	EventCount              int64                    `protobuf:"varint,21,opt,name=eventCount" json:"eventCount"`
	TransactionCount        int64                    `protobuf:"varint,22,opt,name=transactionCount" json:"transactionCount"`
	TransactionOriginCount  int64                    `protobuf:"varint,23,opt,name=transactionOriginCount" json:"transactionOriginCount"`
	BuyEventCount           int64                    `protobuf:"varint,24,opt,name=buyEventCount" json:"buyEventCount"`
	SellEventCount          int64                    `protobuf:"varint,25,opt,name=sellEventCount" json:"sellEventCount"`
	ComplexEventCount       int64                    `protobuf:"varint,26,opt,name=complexEventCount" json:"complexEventCount"`
	BaseAmountInRaw         string                   `protobuf:"bytes,27,opt,name=baseAmountInRaw" json:"baseAmountInRaw"`
	BaseAmountOutRaw        string                   `protobuf:"bytes,28,opt,name=baseAmountOutRaw" json:"baseAmountOutRaw"`
	QuoteAmountInRaw        string                   `protobuf:"bytes,29,opt,name=quoteAmountInRaw" json:"quoteAmountInRaw"`
	QuoteAmountOutRaw       string                   `protobuf:"bytes,30,opt,name=quoteAmountOutRaw" json:"quoteAmountOutRaw"`
	BuyQuoteAmountRaw       string                   `protobuf:"bytes,31,opt,name=buyQuoteAmountRaw" json:"buyQuoteAmountRaw"`
	SellQuoteAmountRaw      string                   `protobuf:"bytes,32,opt,name=sellQuoteAmountRaw" json:"sellQuoteAmountRaw"`
	Blocks                  []*TokenProjectSwapBlock `protobuf:"bytes,33,rep,name=blocks" json:"blocks"`
}

type TokenProjectSwapActivity struct {
	ProjectID   int64                           `protobuf:"varint,1,opt,name=projectId" json:"projectId"`
	ChainID     int64                           `protobuf:"varint,2,opt,name=chainId" json:"chainId"`
	GeneratedAt string                          `protobuf:"bytes,3,opt,name=generatedAt" json:"generatedAt"`
	Pairs       []*TokenProjectSwapPairActivity `protobuf:"bytes,4,rep,name=pairs" json:"pairs"`
}

type TokenProjectSwapEvent struct {
	TransactionHash   string `protobuf:"bytes,1,opt,name=transactionHash" json:"transactionHash"`
	TransactionIndex  string `protobuf:"bytes,2,opt,name=transactionIndex" json:"transactionIndex"`
	LogIndex          string `protobuf:"bytes,3,opt,name=logIndex" json:"logIndex"`
	TxFrom            string `protobuf:"bytes,4,opt,name=txFrom" json:"txFrom"`
	Sender            string `protobuf:"bytes,5,opt,name=sender" json:"sender"`
	ToAddress         string `protobuf:"bytes,6,opt,name=toAddress" json:"toAddress"`
	Amount0In         string `protobuf:"bytes,7,opt,name=amount0In" json:"amount0In"`
	Amount1In         string `protobuf:"bytes,8,opt,name=amount1In" json:"amount1In"`
	Amount0Out        string `protobuf:"bytes,9,opt,name=amount0Out" json:"amount0Out"`
	Amount1Out        string `protobuf:"bytes,10,opt,name=amount1Out" json:"amount1Out"`
	BaseAmountInRaw   string `protobuf:"bytes,11,opt,name=baseAmountInRaw" json:"baseAmountInRaw"`
	BaseAmountOutRaw  string `protobuf:"bytes,12,opt,name=baseAmountOutRaw" json:"baseAmountOutRaw"`
	QuoteAmountInRaw  string `protobuf:"bytes,13,opt,name=quoteAmountInRaw" json:"quoteAmountInRaw"`
	QuoteAmountOutRaw string `protobuf:"bytes,14,opt,name=quoteAmountOutRaw" json:"quoteAmountOutRaw"`
	Direction         string `protobuf:"bytes,15,opt,name=direction" json:"direction"`
	EffectivePrice    string `protobuf:"bytes,16,opt,name=effectivePrice" json:"effectivePrice"`
}

type TokenProjectTrendPoint struct {
	ObservedAt string `protobuf:"bytes,1,opt,name=observedAt" json:"observedAt"`
	Value      string `protobuf:"bytes,2,opt,name=value" json:"value"`
}

type TokenProjectTrendSeries struct {
	Key      string                    `protobuf:"bytes,1,opt,name=key" json:"key"`
	Label    string                    `protobuf:"bytes,2,opt,name=label" json:"label"`
	Unit     string                    `protobuf:"bytes,3,opt,name=unit" json:"unit"`
	DataType string                    `protobuf:"bytes,4,opt,name=dataType" json:"dataType"`
	Points   []*TokenProjectTrendPoint `protobuf:"bytes,5,rep,name=points" json:"points"`
}

type TokenProjectTrends struct {
	Range        string                     `protobuf:"bytes,1,opt,name=range" json:"range"`
	ObservedFrom string                     `protobuf:"bytes,2,opt,name=observedFrom" json:"observedFrom"`
	GeneratedAt  string                     `protobuf:"bytes,3,opt,name=generatedAt" json:"generatedAt"`
	Series       []*TokenProjectTrendSeries `protobuf:"bytes,4,rep,name=series" json:"series"`
}

type TokenWalletNormalTransaction struct {
	Wallet           string `protobuf:"bytes,1,opt,name=wallet" json:"wallet"`
	TransactionHash  string `protobuf:"bytes,2,opt,name=transactionHash" json:"transactionHash"`
	BlockNumber      uint64 `protobuf:"varint,3,opt,name=blockNumber" json:"blockNumber"`
	BlockTimestamp   string `protobuf:"bytes,4,opt,name=blockTimestamp" json:"blockTimestamp"`
	TransactionIndex uint64 `protobuf:"varint,5,opt,name=transactionIndex" json:"transactionIndex"`
	Nonce            uint64 `protobuf:"varint,6,opt,name=nonce" json:"nonce"`
	FromAddress      string `protobuf:"bytes,7,opt,name=fromAddress" json:"fromAddress"`
	ToAddress        string `protobuf:"bytes,8,opt,name=toAddress" json:"toAddress"`
	Value            string `protobuf:"bytes,9,opt,name=value" json:"value"`
	Gas              uint64 `protobuf:"varint,10,opt,name=gas" json:"gas"`
	GasPrice         string `protobuf:"bytes,11,opt,name=gasPrice" json:"gasPrice"`
	GasUsed          uint64 `protobuf:"varint,12,opt,name=gasUsed" json:"gasUsed"`
	Input            string `protobuf:"bytes,13,opt,name=input" json:"input"`
	MethodID         string `protobuf:"bytes,14,opt,name=methodId" json:"methodId"`
	FunctionName     string `protobuf:"bytes,15,opt,name=functionName" json:"functionName"`
	ReceiptStatus    string `protobuf:"bytes,16,opt,name=receiptStatus" json:"receiptStatus"`
	IsError          bool   `protobuf:"varint,17,opt,name=isError" json:"isError"`
	CollectedAt      string `protobuf:"bytes,18,opt,name=collectedAt" json:"collectedAt"`
}
