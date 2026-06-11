package v1alpha1

type TokenAPIBytecodeBlacklist struct {
	CodeHash       string `protobuf:"bytes,1,opt,name=codeHash" json:"codeHash"`
	Note           string `protobuf:"bytes,2,opt,name=note" json:"note"`
	SourceChainID  int64  `protobuf:"varint,3,opt,name=sourceChainId" json:"sourceChainId"`
	SourceContract string `protobuf:"bytes,4,opt,name=sourceContract" json:"sourceContract"`
	CreatedAt      string `protobuf:"bytes,5,opt,name=createdAt" json:"createdAt"`
}

type TokenAPIWalletBlacklist struct {
	Wallet    string `protobuf:"bytes,1,opt,name=wallet" json:"wallet"`
	Note      string `protobuf:"bytes,2,opt,name=note" json:"note"`
	CreatedAt string `protobuf:"bytes,3,opt,name=createdAt" json:"createdAt"`
}

type TokenAPIChainIngestCheckpoint struct {
	ChainID           int64  `protobuf:"varint,1,opt,name=chainId" json:"chainId"`
	ChainName         string `protobuf:"bytes,2,opt,name=chainName" json:"chainName"`
	Enabled           bool   `protobuf:"varint,3,opt,name=enabled" json:"enabled"`
	CursorBlockNumber uint64 `protobuf:"varint,4,opt,name=cursorBlockNumber" json:"cursorBlockNumber"`
	Status            string `protobuf:"bytes,5,opt,name=status" json:"status"`
	CreatedAt         string `protobuf:"bytes,6,opt,name=createdAt" json:"createdAt"`
}

type TokenAPIChainOption struct {
	ChainID   int64  `protobuf:"varint,1,opt,name=chainId" json:"chainId"`
	ChainName string `protobuf:"bytes,2,opt,name=chainName" json:"chainName"`
}

type TokenAPIOptions struct {
	Chains []TokenAPIChainOption `protobuf:"bytes,1,rep,name=chains" json:"chains"`
}

type TokenAPINodeStatus struct {
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

type TokenAPIContractCode struct {
	CodeHash            string `protobuf:"bytes,1,opt,name=codeHash" json:"codeHash"`
	SourceCode          string `protobuf:"bytes,2,opt,name=sourceCode" json:"sourceCode"`
	SourceCodeFetchedAt string `protobuf:"bytes,3,opt,name=sourceCodeFetchedAt" json:"sourceCodeFetchedAt"`
	CreatedAt           string `protobuf:"bytes,4,opt,name=createdAt" json:"createdAt"`
	DeploymentCount     int64  `protobuf:"varint,5,opt,name=deploymentCount" json:"deploymentCount"`
}

type TokenAPIProject struct {
	ProjectID   int64  `protobuf:"varint,1,opt,name=projectId" json:"projectId"`
	ChainID     int64  `protobuf:"varint,2,opt,name=chainId" json:"chainId"`
	Name        string `protobuf:"bytes,3,opt,name=name" json:"name"`
	Symbol      string `protobuf:"bytes,4,opt,name=symbol" json:"symbol"`
	Contract    string `protobuf:"bytes,5,opt,name=contract" json:"contract"`
	Creator     string `protobuf:"bytes,6,opt,name=creator" json:"creator"`
	TxHash      string `protobuf:"bytes,7,opt,name=txHash" json:"txHash"`
	TxIndex     uint64 `protobuf:"varint,8,opt,name=txIndex" json:"txIndex"`
	BlockNumber uint64 `protobuf:"varint,9,opt,name=blockNumber" json:"blockNumber"`
	BlockTime   uint64 `protobuf:"varint,10,opt,name=blockTime" json:"blockTime"`
	CodeHash    string `protobuf:"bytes,11,opt,name=codeHash" json:"codeHash"`
	CreatedAt   string `protobuf:"bytes,12,opt,name=createdAt" json:"createdAt"`
}

type TokenAPIProjectReport struct {
	ProjectID                 int64  `protobuf:"varint,1,opt,name=projectId" json:"projectId"`
	ChainID                   int64  `protobuf:"varint,2,opt,name=chainId" json:"chainId"`
	Name                      string `protobuf:"bytes,3,opt,name=name" json:"name"`
	Symbol                    string `protobuf:"bytes,4,opt,name=symbol" json:"symbol"`
	Contract                  string `protobuf:"bytes,5,opt,name=contract" json:"contract"`
	EvaluationStatus          string `protobuf:"bytes,6,opt,name=evaluationStatus" json:"evaluationStatus"`
	EvaluationAttempts        int32  `protobuf:"varint,7,opt,name=evaluationAttempts" json:"evaluationAttempts"`
	EvaluationLastError       string `protobuf:"bytes,8,opt,name=evaluationLastError" json:"evaluationLastError"`
	EvaluationUpdatedAt       string `protobuf:"bytes,9,opt,name=evaluationUpdatedAt" json:"evaluationUpdatedAt"`
	ReportDataAvailable       bool   `protobuf:"varint,10,opt,name=reportDataAvailable" json:"reportDataAvailable"`
	WethPairIsCreated         bool   `protobuf:"varint,11,opt,name=wethPairIsCreated" json:"wethPairIsCreated"`
	WethPairIsRemoveLiquidity bool   `protobuf:"varint,12,opt,name=wethPairIsRemoveLiquidity" json:"wethPairIsRemoveLiquidity"`
	WethPairIsMint            bool   `protobuf:"varint,13,opt,name=wethPairIsMint" json:"wethPairIsMint"`
	WethPairQuoteUsdtValueInt string `protobuf:"bytes,14,opt,name=wethPairQuoteUsdtValueInt" json:"wethPairQuoteUsdtValueInt"`
	WethPairLastSwapAt        string `protobuf:"bytes,15,opt,name=wethPairLastSwapAt" json:"wethPairLastSwapAt"`
	UsdtPairIsCreated         bool   `protobuf:"varint,16,opt,name=usdtPairIsCreated" json:"usdtPairIsCreated"`
	UsdtPairIsRemoveLiquidity bool   `protobuf:"varint,17,opt,name=usdtPairIsRemoveLiquidity" json:"usdtPairIsRemoveLiquidity"`
	UsdtPairIsMint            bool   `protobuf:"varint,18,opt,name=usdtPairIsMint" json:"usdtPairIsMint"`
	UsdtPairQuoteUsdtValueInt string `protobuf:"bytes,19,opt,name=usdtPairQuoteUsdtValueInt" json:"usdtPairQuoteUsdtValueInt"`
	UsdtPairLastSwapAt        string `protobuf:"bytes,20,opt,name=usdtPairLastSwapAt" json:"usdtPairLastSwapAt"`
	SourceUpdatedAt           string `protobuf:"bytes,21,opt,name=sourceUpdatedAt" json:"sourceUpdatedAt"`
	EvaluatedAt               string `protobuf:"bytes,22,opt,name=evaluatedAt" json:"evaluatedAt"`
	CreatedAt                 string `protobuf:"bytes,23,opt,name=createdAt" json:"createdAt"`
}

type TokenAPIProjectDataCollectionTask struct {
	ProjectID     int64  `protobuf:"varint,1,opt,name=projectId" json:"projectId"`
	DataType      string `protobuf:"bytes,2,opt,name=dataType" json:"dataType"`
	Status        string `protobuf:"bytes,3,opt,name=status" json:"status"`
	Attempts      int32  `protobuf:"varint,4,opt,name=attempts" json:"attempts"`
	NextAttemptAt string `protobuf:"bytes,5,opt,name=nextAttemptAt" json:"nextAttemptAt"`
	LastError     string `protobuf:"bytes,6,opt,name=lastError" json:"lastError"`
	CreatedAt     string `protobuf:"bytes,7,opt,name=createdAt" json:"createdAt"`
}
