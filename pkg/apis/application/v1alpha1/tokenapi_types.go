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

type TokenAPIContractCode struct {
	CodeHash            string `protobuf:"bytes,1,opt,name=codeHash" json:"codeHash"`
	SourceCode          string `protobuf:"bytes,2,opt,name=sourceCode" json:"sourceCode"`
	SourceCodeHash      string `protobuf:"bytes,3,opt,name=sourceCodeHash" json:"sourceCodeHash"`
	SourceCodeFetchedAt string `protobuf:"bytes,4,opt,name=sourceCodeFetchedAt" json:"sourceCodeFetchedAt"`
	CreatedAt           string `protobuf:"bytes,5,opt,name=createdAt" json:"createdAt"`
	DeploymentCount     int64  `protobuf:"varint,6,opt,name=deploymentCount" json:"deploymentCount"`
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
