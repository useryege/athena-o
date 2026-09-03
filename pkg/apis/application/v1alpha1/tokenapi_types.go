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
	UpdatedAt         string `protobuf:"bytes,7,opt,name=updatedAt" json:"updatedAt"`
}

type TokenChainProcessingAttempt struct {
	AttemptID                int64  `protobuf:"varint,1,opt,name=attemptId" json:"attemptId"`
	ChainID                  int64  `protobuf:"varint,2,opt,name=chainId" json:"chainId"`
	BlockNumber              uint64 `protobuf:"varint,3,opt,name=blockNumber" json:"blockNumber"`
	AttemptNumber            int32  `protobuf:"varint,4,opt,name=attemptNumber" json:"attemptNumber"`
	BlockTime                uint64 `protobuf:"varint,5,opt,name=blockTime" json:"blockTime"`
	Status                   string `protobuf:"bytes,6,opt,name=status" json:"status"`
	TerminalStage            string `protobuf:"bytes,7,opt,name=terminalStage" json:"terminalStage"`
	ErrorMessage             string `protobuf:"bytes,8,opt,name=errorMessage" json:"errorMessage"`
	CheckpointReadDurationUS int64  `protobuf:"varint,9,opt,name=checkpointReadDurationUs" json:"checkpointReadDurationUs"`
	DiscoveryDurationUS      int64  `protobuf:"varint,10,opt,name=discoveryDurationUs" json:"discoveryDurationUs"`
	ValidationDurationUS     int64  `protobuf:"varint,11,opt,name=validationDurationUs" json:"validationDurationUs"`
	PersistenceDurationUS    int64  `protobuf:"varint,12,opt,name=persistenceDurationUs" json:"persistenceDurationUs"`
	TotalDurationUS          int64  `protobuf:"varint,13,opt,name=totalDurationUs" json:"totalDurationUs"`
	CandidateCount           int32  `protobuf:"varint,14,opt,name=candidateCount" json:"candidateCount"`
	ValidatedCount           int32  `protobuf:"varint,15,opt,name=validatedCount" json:"validatedCount"`
	RejectedCount            int32  `protobuf:"varint,16,opt,name=rejectedCount" json:"rejectedCount"`
	TimingComplete           bool   `protobuf:"varint,17,opt,name=timingComplete" json:"timingComplete"`
	StartedAt                string `protobuf:"bytes,18,opt,name=startedAt" json:"startedAt"`
	CompletedAt              string `protobuf:"bytes,19,opt,name=completedAt" json:"completedAt"`
	CreatedAt                string `protobuf:"bytes,20,opt,name=createdAt" json:"createdAt"`
	UpdatedAt                string `protobuf:"bytes,21,opt,name=updatedAt" json:"updatedAt"`
}

type TokenChainProcessingSummary struct {
	ChainID                         int64  `protobuf:"varint,1,opt,name=chainId" json:"chainId"`
	RangeStartBlockTime             uint64 `protobuf:"varint,2,opt,name=rangeStartBlockTime" json:"rangeStartBlockTime"`
	RangeEndBlockTime               uint64 `protobuf:"varint,3,opt,name=rangeEndBlockTime" json:"rangeEndBlockTime"`
	AttemptCount                    int64  `protobuf:"varint,4,opt,name=attemptCount" json:"attemptCount"`
	RunningCount                    int64  `protobuf:"varint,5,opt,name=runningCount" json:"runningCount"`
	SucceededCount                  int64  `protobuf:"varint,6,opt,name=succeededCount" json:"succeededCount"`
	FailedCount                     int64  `protobuf:"varint,7,opt,name=failedCount" json:"failedCount"`
	CancelledCount                  int64  `protobuf:"varint,8,opt,name=cancelledCount" json:"cancelledCount"`
	InterruptedCount                int64  `protobuf:"varint,9,opt,name=interruptedCount" json:"interruptedCount"`
	IncompleteSucceededCount        int64  `protobuf:"varint,10,opt,name=incompleteSucceededCount" json:"incompleteSucceededCount"`
	MeasuredSucceededCount          int64  `protobuf:"varint,11,opt,name=measuredSucceededCount" json:"measuredSucceededCount"`
	FailureRateBPS                  int64  `protobuf:"varint,12,opt,name=failureRateBps" json:"failureRateBps"`
	AverageDurationUS               int64  `protobuf:"varint,13,opt,name=averageDurationUs" json:"averageDurationUs"`
	AverageCheckpointReadDurationUS int64  `protobuf:"varint,14,opt,name=averageCheckpointReadDurationUs" json:"averageCheckpointReadDurationUs"`
	AverageDiscoveryDurationUS      int64  `protobuf:"varint,15,opt,name=averageDiscoveryDurationUs" json:"averageDiscoveryDurationUs"`
	AverageValidationDurationUS     int64  `protobuf:"varint,16,opt,name=averageValidationDurationUs" json:"averageValidationDurationUs"`
	AveragePersistenceDurationUS    int64  `protobuf:"varint,17,opt,name=averagePersistenceDurationUs" json:"averagePersistenceDurationUs"`
	FastestBlockNumber              uint64 `protobuf:"varint,18,opt,name=fastestBlockNumber" json:"fastestBlockNumber"`
	FastestDurationUS               int64  `protobuf:"varint,19,opt,name=fastestDurationUs" json:"fastestDurationUs"`
	SlowestBlockNumber              uint64 `protobuf:"varint,20,opt,name=slowestBlockNumber" json:"slowestBlockNumber"`
	SlowestDurationUS               int64  `protobuf:"varint,21,opt,name=slowestDurationUs" json:"slowestDurationUs"`
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
	ProjectID       int64  `protobuf:"varint,1,opt,name=projectId" json:"projectId"`
	ChainID         int64  `protobuf:"varint,2,opt,name=chainId" json:"chainId"`
	Name            string `protobuf:"bytes,3,opt,name=name" json:"name"`
	Symbol          string `protobuf:"bytes,4,opt,name=symbol" json:"symbol"`
	Contract        string `protobuf:"bytes,5,opt,name=contract" json:"contract"`
	TxSender        string `protobuf:"bytes,6,opt,name=txSender" json:"txSender"`
	TxHash          string `protobuf:"bytes,7,opt,name=txHash" json:"txHash"`
	TxIndex         uint64 `protobuf:"varint,8,opt,name=txIndex" json:"txIndex"`
	BlockNumber     uint64 `protobuf:"varint,9,opt,name=blockNumber" json:"blockNumber"`
	BlockTime       uint64 `protobuf:"varint,10,opt,name=blockTime" json:"blockTime"`
	CodeHash        string `protobuf:"bytes,11,opt,name=codeHash" json:"codeHash"`
	CreatedAt       string `protobuf:"bytes,12,opt,name=createdAt" json:"createdAt"`
	Decimals        int32  `protobuf:"varint,13,opt,name=decimals" json:"decimals"`
	TotalSupply     string `protobuf:"bytes,14,opt,name=totalSupply" json:"totalSupply"`
	WethPair        string `protobuf:"bytes,15,opt,name=wethPair" json:"wethPair"`
	UsdtPair        string `protobuf:"bytes,16,opt,name=usdtPair" json:"usdtPair"`
	DeploymentNonce uint64 `protobuf:"varint,17,opt,name=deploymentNonce" json:"deploymentNonce"`
}

type TokenProjectListItem struct {
	ProjectID                int64                           `protobuf:"varint,1,opt,name=projectId" json:"projectId"`
	ChainID                  int64                           `protobuf:"varint,2,opt,name=chainId" json:"chainId"`
	Name                     string                          `protobuf:"bytes,3,opt,name=name" json:"name"`
	Symbol                   string                          `protobuf:"bytes,4,opt,name=symbol" json:"symbol"`
	Contract                 string                          `protobuf:"bytes,5,opt,name=contract" json:"contract"`
	CodeHash                 string                          `protobuf:"bytes,6,opt,name=codeHash" json:"codeHash"`
	BlockNumber              uint64                          `protobuf:"varint,7,opt,name=blockNumber" json:"blockNumber"`
	BlockTime                uint64                          `protobuf:"varint,8,opt,name=blockTime" json:"blockTime"`
	TxHash                   string                          `protobuf:"bytes,9,opt,name=txHash" json:"txHash"`
	CreatedAt                string                          `protobuf:"bytes,10,opt,name=createdAt" json:"createdAt"`
	CollectionStatus         string                          `protobuf:"bytes,11,opt,name=collectionStatus" json:"collectionStatus"`
	CollectionSucceededCount int32                           `protobuf:"varint,12,opt,name=collectionSucceededCount" json:"collectionSucceededCount"`
	CollectionTerminalCount  int32                           `protobuf:"varint,13,opt,name=collectionTerminalCount" json:"collectionTerminalCount"`
	CollectionTotalCount     int32                           `protobuf:"varint,14,opt,name=collectionTotalCount" json:"collectionTotalCount"`
	ProfileState             string                          `protobuf:"bytes,15,opt,name=profileState" json:"profileState"`
	CompletenessStatus       string                          `protobuf:"bytes,16,opt,name=completenessStatus" json:"completenessStatus"`
	ProfileBuiltAt           string                          `protobuf:"bytes,17,opt,name=profileBuiltAt" json:"profileBuiltAt"`
	Market                   *TokenProjectMarketSummary      `protobuf:"bytes,18,opt,name=market" json:"market"`
	WrappedNativePair        *TokenProjectPairProfileSummary `protobuf:"bytes,19,opt,name=wrappedNativePair" json:"wrappedNativePair"`
	UsdtPair                 *TokenProjectPairProfileSummary `protobuf:"bytes,20,opt,name=usdtPair" json:"usdtPair"`
}

type TokenProjectMarketSummary struct {
	LogoURL         string `protobuf:"bytes,1,opt,name=logoUrl" json:"logoUrl"`
	CurrentPriceUSD string `protobuf:"bytes,2,opt,name=currentPriceUsd" json:"currentPriceUsd"`
	MarketCapUSD    string `protobuf:"bytes,3,opt,name=marketCapUsd" json:"marketCapUsd"`
	FDVUSD          string `protobuf:"bytes,4,opt,name=fdvUsd" json:"fdvUsd"`
	TVLUSD          string `protobuf:"bytes,5,opt,name=tvlUsd" json:"tvlUsd"`
	Holders         int64  `protobuf:"varint,6,opt,name=holders" json:"holders"`
}

type TokenProjectPairProfileSummary struct {
	Kind                               string `protobuf:"bytes,1,opt,name=kind" json:"kind"`
	Address                            string `protobuf:"bytes,2,opt,name=address" json:"address"`
	IsCreated                          bool   `protobuf:"varint,3,opt,name=isCreated" json:"isCreated"`
	QuoteUsdtValueInt                  string `protobuf:"bytes,4,opt,name=quoteUsdtValueInt" json:"quoteUsdtValueInt"`
	ReserveUpdatedAt                   uint64 `protobuf:"varint,5,opt,name=reserveUpdatedAt" json:"reserveUpdatedAt"`
	PairTokenBalanceExceedsTotalSupply bool   `protobuf:"varint,6,opt,name=pairTokenBalanceExceedsTotalSupply" json:"pairTokenBalanceExceedsTotalSupply"`
	LPMinimumSupplyOnly                bool   `protobuf:"varint,7,opt,name=lpMinimumSupplyOnly" json:"lpMinimumSupplyOnly"`
	FixedFeeAddressLPShareGte90Percent bool   `protobuf:"varint,8,opt,name=fixedFeeAddressLpShareGte90Percent" json:"fixedFeeAddressLpShareGte90Percent"`
}

type TokenCollectionTask struct {
	TaskID          int64                  `protobuf:"varint,1,opt,name=taskId" json:"taskId"`
	ProjectID       int64                  `protobuf:"varint,2,opt,name=projectId" json:"projectId"`
	DataType        string                 `protobuf:"bytes,3,opt,name=dataType" json:"dataType"`
	Status          string                 `protobuf:"bytes,4,opt,name=status" json:"status"`
	FailureCount    int32                  `protobuf:"varint,5,opt,name=failureCount" json:"failureCount"`
	AvailableAt     string                 `protobuf:"bytes,6,opt,name=availableAt" json:"availableAt"`
	ClaimGeneration int64                  `protobuf:"varint,7,opt,name=claimGeneration" json:"claimGeneration"`
	LockedAt        string                 `protobuf:"bytes,8,opt,name=lockedAt" json:"lockedAt"`
	LeaseExpiresAt  string                 `protobuf:"bytes,9,opt,name=leaseExpiresAt" json:"leaseExpiresAt"`
	LastError       string                 `protobuf:"bytes,10,opt,name=lastError" json:"lastError"`
	FinishedAt      string                 `protobuf:"bytes,11,opt,name=finishedAt" json:"finishedAt"`
	CreatedAt       string                 `protobuf:"bytes,12,opt,name=createdAt" json:"createdAt"`
	UpdatedAt       string                 `protobuf:"bytes,13,opt,name=updatedAt" json:"updatedAt"`
	Result          *TokenCollectionResult `protobuf:"bytes,14,opt,name=result" json:"result"`
}

type TokenCollectionResult struct {
	TaskID        int64   `protobuf:"varint,1,opt,name=taskId" json:"taskId"`
	ProjectID     int64   `protobuf:"varint,2,opt,name=projectId" json:"projectId"`
	DataType      string  `protobuf:"bytes,3,opt,name=dataType" json:"dataType"`
	SchemaVersion int32   `protobuf:"varint,4,opt,name=schemaVersion" json:"schemaVersion"`
	PayloadJSON   string  `protobuf:"bytes,5,opt,name=payloadJson" json:"payloadJson"`
	ContentHash   string  `protobuf:"bytes,6,opt,name=contentHash" json:"contentHash"`
	BlockNumber   *uint64 `protobuf:"varint,7,opt,name=blockNumber" json:"blockNumber,omitempty"`
	CollectedAt   string  `protobuf:"bytes,8,opt,name=collectedAt" json:"collectedAt"`
}

type TokenProjectProfile struct {
	ProjectID          int64                             `protobuf:"varint,1,opt,name=projectId" json:"projectId"`
	SchemaVersion      int32                             `protobuf:"varint,2,opt,name=schemaVersion" json:"schemaVersion"`
	CompletenessStatus string                            `protobuf:"bytes,3,opt,name=completenessStatus" json:"completenessStatus"`
	FailedDataTypes    []string                          `protobuf:"bytes,4,rep,name=failedDataTypes" json:"failedDataTypes"`
	Market             *TokenProjectProfileMarket        `protobuf:"bytes,5,opt,name=market" json:"market"`
	ContractSource     *TokenProjectProfileSource        `protobuf:"bytes,6,opt,name=contractSource" json:"contractSource"`
	WrappedNativePair  *TokenProjectProfilePair          `protobuf:"bytes,7,opt,name=wrappedNativePair" json:"wrappedNativePair"`
	UsdtPair           *TokenProjectProfilePair          `protobuf:"bytes,8,opt,name=usdtPair" json:"usdtPair"`
	WalletSummary      *TokenProjectProfileWalletSummary `protobuf:"bytes,9,opt,name=walletSummary" json:"walletSummary"`
	Transactions       *TokenProjectProfileTransactions  `protobuf:"bytes,10,opt,name=transactions" json:"transactions"`
	Evidence           []*TokenProjectProfileEvidence    `protobuf:"bytes,11,rep,name=evidence" json:"evidence"`
	ProfileJSON        string                            `protobuf:"bytes,12,opt,name=profileJson" json:"profileJson"`
	ContentHash        string                            `protobuf:"bytes,13,opt,name=contentHash" json:"contentHash"`
	BuiltAt            string                            `protobuf:"bytes,14,opt,name=builtAt" json:"builtAt"`
	CreatedAt          string                            `protobuf:"bytes,15,opt,name=createdAt" json:"createdAt"`
	Wallets            []*TokenProjectProfileWallet      `protobuf:"bytes,16,rep,name=wallets" json:"wallets"`
}

type TokenProjectProfileMarket struct {
	LogoURL           string                      `protobuf:"bytes,1,opt,name=logoUrl" json:"logoUrl"`
	CurrentPriceUSD   string                      `protobuf:"bytes,2,opt,name=currentPriceUsd" json:"currentPriceUsd"`
	CurrentPriceETH   string                      `protobuf:"bytes,3,opt,name=currentPriceEth" json:"currentPriceEth"`
	MarketCapUSD      string                      `protobuf:"bytes,4,opt,name=marketCapUsd" json:"marketCapUsd"`
	FDVUSD            string                      `protobuf:"bytes,5,opt,name=fdvUsd" json:"fdvUsd"`
	TVLUSD            string                      `protobuf:"bytes,6,opt,name=tvlUsd" json:"tvlUsd"`
	MainPairTVLUSD    string                      `protobuf:"bytes,7,opt,name=mainPairTvlUsd" json:"mainPairTvlUsd"`
	Holders           int64                       `protobuf:"varint,8,opt,name=holders" json:"holders"`
	LaunchAt          string                      `protobuf:"bytes,9,opt,name=launchAt" json:"launchAt"`
	ProviderUpdatedAt string                      `protobuf:"bytes,10,opt,name=providerUpdatedAt" json:"providerUpdatedAt"`
	AveRisk           *TokenProjectProfileAveRisk `protobuf:"bytes,11,opt,name=aveRisk" json:"aveRisk"`
}

type TokenProjectProfileAveRisk struct {
	RiskLevel             int32  `protobuf:"varint,1,opt,name=riskLevel" json:"riskLevel"`
	RiskScore             string `protobuf:"bytes,2,opt,name=riskScore" json:"riskScore"`
	RiskInfo              string `protobuf:"bytes,3,opt,name=riskInfo" json:"riskInfo"`
	Audited               bool   `protobuf:"varint,4,opt,name=audited" json:"audited"`
	Mintable              *bool  `protobuf:"varint,5,opt,name=mintable" json:"mintable,omitempty"`
	HasMintMethod         bool   `protobuf:"varint,6,opt,name=hasMintMethod" json:"hasMintMethod"`
	LiquidityPoolUnlocked bool   `protobuf:"varint,7,opt,name=liquidityPoolUnlocked" json:"liquidityPoolUnlocked"`
	OwnershipNotRenounced bool   `protobuf:"varint,8,opt,name=ownershipNotRenounced" json:"ownershipNotRenounced"`
	NotAudited            bool   `protobuf:"varint,9,opt,name=notAudited" json:"notAudited"`
	NotOpenSource         bool   `protobuf:"varint,10,opt,name=notOpenSource" json:"notOpenSource"`
	InBlacklist           bool   `protobuf:"varint,11,opt,name=inBlacklist" json:"inBlacklist"`
	Honeypot              bool   `protobuf:"varint,12,opt,name=honeypot" json:"honeypot"`
}

type TokenProjectProfileSource struct {
	CodeHash           string `protobuf:"bytes,1,opt,name=codeHash" json:"codeHash"`
	VerificationStatus string `protobuf:"bytes,2,opt,name=verificationStatus" json:"verificationStatus"`
	ArtifactReference  string `protobuf:"bytes,3,opt,name=artifactReference" json:"artifactReference"`
}

type TokenProjectProfilePair struct {
	Kind       string                             `protobuf:"bytes,1,opt,name=kind" json:"kind"`
	Address    string                             `protobuf:"bytes,2,opt,name=address" json:"address"`
	ChainState *TokenProjectProfilePairChainState `protobuf:"bytes,3,opt,name=chainState" json:"chainState"`
	Market     *TokenProjectProfilePairMarket     `protobuf:"bytes,4,opt,name=market" json:"market"`
}

type TokenProjectProfilePairChainState struct {
	IsCreated         bool                              `protobuf:"varint,1,opt,name=isCreated" json:"isCreated"`
	BaseBalance       string                            `protobuf:"bytes,2,opt,name=baseBalance" json:"baseBalance"`
	QuoteBalance      string                            `protobuf:"bytes,3,opt,name=quoteBalance" json:"quoteBalance"`
	QuoteUsdtValue    string                            `protobuf:"bytes,4,opt,name=quoteUsdtValue" json:"quoteUsdtValue"`
	QuoteUsdtValueInt string                            `protobuf:"bytes,5,opt,name=quoteUsdtValueInt" json:"quoteUsdtValueInt"`
	ReserveUpdatedAt  uint64                            `protobuf:"varint,6,opt,name=reserveUpdatedAt" json:"reserveUpdatedAt"`
	Liquidity         *TokenProjectProfilePairLiquidity `protobuf:"bytes,7,opt,name=liquidity" json:"liquidity"`
	Signals           *TokenProjectProfilePairSignals   `protobuf:"bytes,8,opt,name=signals" json:"signals"`
}

type TokenProjectProfilePairLiquidity struct {
	TotalSupply            string `protobuf:"bytes,1,opt,name=totalSupply" json:"totalSupply"`
	LockedLiquidity        string `protobuf:"bytes,2,opt,name=lockedLiquidity" json:"lockedLiquidity"`
	FixedFeeAddressBalance string `protobuf:"bytes,3,opt,name=fixedFeeAddressBalance" json:"fixedFeeAddressBalance"`
	FixedFeeAddressShare   string `protobuf:"bytes,4,opt,name=fixedFeeAddressShare" json:"fixedFeeAddressShare"`
}

type TokenProjectProfilePairSignals struct {
	PairTokenBalanceExceedsTotalSupply bool `protobuf:"varint,1,opt,name=pairTokenBalanceExceedsTotalSupply" json:"pairTokenBalanceExceedsTotalSupply"`
	LPMinimumSupplyOnly                bool `protobuf:"varint,2,opt,name=lpMinimumSupplyOnly" json:"lpMinimumSupplyOnly"`
	FixedFeeAddressLPShareGte90Percent bool `protobuf:"varint,3,opt,name=fixedFeeAddressLpShareGte90Percent" json:"fixedFeeAddressLpShareGte90Percent"`
}

type TokenProjectProfilePairMarket struct {
	AMM           string `protobuf:"bytes,1,opt,name=amm" json:"amm"`
	Token0Address string `protobuf:"bytes,2,opt,name=token0Address" json:"token0Address"`
	Token0Symbol  string `protobuf:"bytes,3,opt,name=token0Symbol" json:"token0Symbol"`
	Token1Address string `protobuf:"bytes,4,opt,name=token1Address" json:"token1Address"`
	Token1Symbol  string `protobuf:"bytes,5,opt,name=token1Symbol" json:"token1Symbol"`
	Reserve0      string `protobuf:"bytes,6,opt,name=reserve0" json:"reserve0"`
	Reserve1      string `protobuf:"bytes,7,opt,name=reserve1" json:"reserve1"`
	VolumeUSD     string `protobuf:"bytes,8,opt,name=volumeUsd" json:"volumeUsd"`
	MarketCapUSD  string `protobuf:"bytes,9,opt,name=marketCapUsd" json:"marketCapUsd"`
	FDVUSD        string `protobuf:"bytes,10,opt,name=fdvUsd" json:"fdvUsd"`
	IsFake        bool   `protobuf:"varint,11,opt,name=isFake" json:"isFake"`
	CreatedAt     string `protobuf:"bytes,12,opt,name=createdAt" json:"createdAt"`
	UpdatedAt     string `protobuf:"bytes,13,opt,name=updatedAt" json:"updatedAt"`
}

type TokenProjectProfileWalletSummary struct {
	WalletCount                   int32                                 `protobuf:"varint,1,opt,name=walletCount" json:"walletCount"`
	NativeBalanceTotal            string                                `protobuf:"bytes,2,opt,name=nativeBalanceTotal" json:"nativeBalanceTotal"`
	WrappedNativeBalanceTotal     string                                `protobuf:"bytes,3,opt,name=wrappedNativeBalanceTotal" json:"wrappedNativeBalanceTotal"`
	UsdtBalanceTotal              string                                `protobuf:"bytes,4,opt,name=usdtBalanceTotal" json:"usdtBalanceTotal"`
	TrackedAssetUsdtValueTotal    string                                `protobuf:"bytes,5,opt,name=trackedAssetUsdtValueTotal" json:"trackedAssetUsdtValueTotal"`
	InitialRecipientCount         int32                                 `protobuf:"varint,6,opt,name=initialRecipientCount" json:"initialRecipientCount"`
	InitialRecipientAllocationBPS uint64                                `protobuf:"varint,7,opt,name=initialRecipientAllocationBps" json:"initialRecipientAllocationBps"`
	WalletsWithSimulationSignals  int32                                 `protobuf:"varint,8,opt,name=walletsWithSimulationSignals" json:"walletsWithSimulationSignals"`
	RoleCounts                    []*TokenProjectProfileWalletRoleCount `protobuf:"bytes,9,rep,name=roleCounts" json:"roleCounts"`
}

type TokenProjectProfileWalletRoleCount struct {
	Role  string `protobuf:"bytes,1,opt,name=role" json:"role"`
	Count int32  `protobuf:"varint,2,opt,name=count" json:"count"`
}

type TokenProjectProfileWallet struct {
	Address                 string                               `protobuf:"bytes,1,opt,name=address" json:"address"`
	Roles                   []string                             `protobuf:"bytes,2,rep,name=roles" json:"roles"`
	InitialRecipient        *TokenProjectProfileInitialRecipient `protobuf:"bytes,3,opt,name=initialRecipient" json:"initialRecipient"`
	Assets                  *TokenProjectProfileWalletAssets     `protobuf:"bytes,4,opt,name=assets" json:"assets"`
	Simulation              *TokenProjectProfileWalletSimulation `protobuf:"bytes,5,opt,name=simulation" json:"simulation"`
	TransactionSampleCapped bool                                 `protobuf:"varint,6,opt,name=transactionSampleCapped" json:"transactionSampleCapped"`
}

type TokenProjectProfileInitialRecipient struct {
	Rank     int32  `protobuf:"varint,1,opt,name=rank" json:"rank"`
	RatioBPS uint64 `protobuf:"varint,2,opt,name=ratioBps" json:"ratioBps"`
}

type TokenProjectProfileWalletAssets struct {
	NativeBalance         string `protobuf:"bytes,1,opt,name=nativeBalance" json:"nativeBalance"`
	WrappedNativeBalance  string `protobuf:"bytes,2,opt,name=wrappedNativeBalance" json:"wrappedNativeBalance"`
	UsdtBalance           string `protobuf:"bytes,3,opt,name=usdtBalance" json:"usdtBalance"`
	TrackedAssetUsdtValue string `protobuf:"bytes,4,opt,name=trackedAssetUsdtValue" json:"trackedAssetUsdtValue"`
}

type TokenProjectProfileWalletSimulation struct {
	TransferFromDeadToWalletCallSucceeded     bool `protobuf:"varint,1,opt,name=transferFromDeadToWalletCallSucceeded" json:"transferFromDeadToWalletCallSucceeded"`
	TransferFromZeroToWalletCallSucceeded     bool `protobuf:"varint,2,opt,name=transferFromZeroToWalletCallSucceeded" json:"transferFromZeroToWalletCallSucceeded"`
	TransferFromWethPairToWalletCallSucceeded bool `protobuf:"varint,3,opt,name=transferFromWethPairToWalletCallSucceeded" json:"transferFromWethPairToWalletCallSucceeded"`
	TransferFromUsdtPairToWalletCallSucceeded bool `protobuf:"varint,4,opt,name=transferFromUsdtPairToWalletCallSucceeded" json:"transferFromUsdtPairToWalletCallSucceeded"`
	TransferFromWalletToWethPairCallSucceeded bool `protobuf:"varint,5,opt,name=transferFromWalletToWethPairCallSucceeded" json:"transferFromWalletToWethPairCallSucceeded"`
	TransferFromWalletToUsdtPairCallSucceeded bool `protobuf:"varint,6,opt,name=transferFromWalletToUsdtPairCallSucceeded" json:"transferFromWalletToUsdtPairCallSucceeded"`
}

type TokenProjectProfileTransactions struct {
	WalletCount                 int32                                         `protobuf:"varint,1,opt,name=walletCount" json:"walletCount"`
	TransactionAssociationCount int64                                         `protobuf:"varint,2,opt,name=transactionAssociationCount" json:"transactionAssociationCount"`
	UniqueTransactionCount      int64                                         `protobuf:"varint,3,opt,name=uniqueTransactionCount" json:"uniqueTransactionCount"`
	SucceededTransactionCount   int64                                         `protobuf:"varint,4,opt,name=succeededTransactionCount" json:"succeededTransactionCount"`
	FailedTransactionCount      int64                                         `protobuf:"varint,5,opt,name=failedTransactionCount" json:"failedTransactionCount"`
	TotalInflowNativeValue      string                                        `protobuf:"bytes,6,opt,name=totalInflowNativeValue" json:"totalInflowNativeValue"`
	TotalOutflowNativeValue     string                                        `protobuf:"bytes,7,opt,name=totalOutflowNativeValue" json:"totalOutflowNativeValue"`
	CappedWallets               []string                                      `protobuf:"bytes,8,rep,name=cappedWallets" json:"cappedWallets"`
	TopMethods                  []*TokenProjectProfileTransactionMethod       `protobuf:"bytes,9,rep,name=topMethods" json:"topMethods"`
	TopCounterparties           []*TokenProjectProfileTransactionCounterparty `protobuf:"bytes,10,rep,name=topCounterparties" json:"topCounterparties"`
}

type TokenProjectProfileTransactionMethod struct {
	MethodID     string `protobuf:"bytes,1,opt,name=methodId" json:"methodId"`
	FunctionName string `protobuf:"bytes,2,opt,name=functionName" json:"functionName"`
	Count        int64  `protobuf:"varint,3,opt,name=count" json:"count"`
}

type TokenProjectProfileTransactionCounterparty struct {
	Address string `protobuf:"bytes,1,opt,name=address" json:"address"`
	Count   int64  `protobuf:"varint,2,opt,name=count" json:"count"`
}

type TokenProjectProfileEvidence struct {
	TaskID              int64   `protobuf:"varint,1,opt,name=taskId" json:"taskId"`
	DataType            string  `protobuf:"bytes,2,opt,name=dataType" json:"dataType"`
	Status              string  `protobuf:"bytes,3,opt,name=status" json:"status"`
	FailureCount        int32   `protobuf:"varint,4,opt,name=failureCount" json:"failureCount"`
	LastError           string  `protobuf:"bytes,5,opt,name=lastError" json:"lastError"`
	ResultSchemaVersion int32   `protobuf:"varint,6,opt,name=resultSchemaVersion" json:"resultSchemaVersion"`
	ResultContentHash   string  `protobuf:"bytes,7,opt,name=resultContentHash" json:"resultContentHash"`
	BlockNumber         *uint64 `protobuf:"varint,8,opt,name=blockNumber" json:"blockNumber,omitempty"`
	CollectedAt         string  `protobuf:"bytes,9,opt,name=collectedAt" json:"collectedAt"`
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

type TokenWalletTransactionCount struct {
	Wallet           string `protobuf:"bytes,1,opt,name=wallet" json:"wallet"`
	TransactionCount int64  `protobuf:"varint,2,opt,name=transactionCount" json:"transactionCount"`
}

type TokenProjectDetail struct {
	Project                 *TokenProject                   `protobuf:"bytes,1,opt,name=project" json:"project"`
	Profile                 *TokenProjectProfile            `protobuf:"bytes,2,opt,name=profile" json:"profile"`
	CollectionTasks         []*TokenCollectionTask          `protobuf:"bytes,3,rep,name=collectionTasks" json:"collectionTasks"`
	RelatedWallets          []*TokenProjectRelatedWallet    `protobuf:"bytes,4,rep,name=relatedWallets" json:"relatedWallets"`
	InitialRecipients       []*TokenProjectInitialRecipient `protobuf:"bytes,5,rep,name=initialRecipients" json:"initialRecipients"`
	WalletTransactionCounts []*TokenWalletTransactionCount  `protobuf:"bytes,6,rep,name=walletTransactionCounts" json:"walletTransactionCounts"`
	TransactionCount        int64                           `protobuf:"varint,7,opt,name=transactionCount" json:"transactionCount"`
	GeneratedAt             string                          `protobuf:"bytes,8,opt,name=generatedAt" json:"generatedAt"`
	CollectionStatus        string                          `protobuf:"bytes,9,opt,name=collectionStatus" json:"collectionStatus"`
	ProfileState            string                          `protobuf:"bytes,10,opt,name=profileState" json:"profileState"`
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
