package v1alpha1

type ContractSourceInfo struct {
	Contract                     string `protobuf:"bytes,1,opt,name=contract" json:"contract"`
	ChainID                      int64  `protobuf:"varint,2,opt,name=chainId" json:"chainId"`
	CodeBinHash                  string `protobuf:"bytes,3,opt,name=codeBinHash" json:"codeBinHash"`
	SourceCode                   string `protobuf:"bytes,4,opt,name=sourceCode" json:"sourceCode"`
	SourceCodeHash               string `protobuf:"bytes,5,opt,name=sourceCodeHash" json:"sourceCodeHash"`
	SourceCodeFetchedAt          string `protobuf:"bytes,6,opt,name=sourceCodeFetchedAt" json:"sourceCodeFetchedAt"`
	SourceCodeOrigin             string `protobuf:"bytes,7,opt,name=sourceCodeOrigin" json:"sourceCodeOrigin"`
	SourceQualityReport          string `protobuf:"bytes,8,opt,name=sourceQualityReport" json:"sourceQualityReport"`
	SourceQualityReportFetchedAt string `protobuf:"bytes,9,opt,name=sourceQualityReportFetchedAt" json:"sourceQualityReportFetchedAt"`
	SourceQualityReportOrigin    string `protobuf:"bytes,10,opt,name=sourceQualityReportOrigin" json:"sourceQualityReportOrigin"`
	IsOpenSource                 bool   `protobuf:"varint,11,opt,name=isOpenSource" json:"isOpenSource"`
	IsBytecodeBlacklisted        bool   `protobuf:"varint,12,opt,name=isBytecodeBlacklisted" json:"isBytecodeBlacklisted"`
	SourceQualityPromptVersion   int64  `protobuf:"varint,13,opt,name=sourceQualityPromptVersion" json:"sourceQualityPromptVersion"`
}

type SourceQualityPrompt struct {
	ID           int64  `protobuf:"varint,1,opt,name=id" json:"id"`
	Version      int64  `protobuf:"varint,2,opt,name=version" json:"version"`
	Name         string `protobuf:"bytes,3,opt,name=name" json:"name"`
	SystemPrompt string `protobuf:"bytes,4,opt,name=systemPrompt" json:"systemPrompt"`
	IsActive     bool   `protobuf:"varint,5,opt,name=isActive" json:"isActive"`
	CreatedAt    string `protobuf:"bytes,6,opt,name=createdAt" json:"createdAt"`
	UpdatedAt    string `protobuf:"bytes,7,opt,name=updatedAt" json:"updatedAt"`
}

type BytecodeBlacklistEntry struct {
	CodeHash       string `protobuf:"bytes,1,opt,name=codeHash" json:"codeHash"`
	Note           string `protobuf:"bytes,2,opt,name=note" json:"note"`
	SourceChainID  int64  `protobuf:"varint,3,opt,name=sourceChainId" json:"sourceChainId"`
	SourceContract string `protobuf:"bytes,4,opt,name=sourceContract" json:"sourceContract"`
	CreatedAt      string `protobuf:"bytes,5,opt,name=createdAt" json:"createdAt"`
}

type BytecodeListItem struct {
	CodeHash              string `protobuf:"bytes,1,opt,name=codeHash" json:"codeHash"`
	RuntimeBytecodeSize   int64  `protobuf:"varint,2,opt,name=runtimeBytecodeSize" json:"runtimeBytecodeSize"`
	DeploymentCount       int64  `protobuf:"varint,3,opt,name=deploymentCount" json:"deploymentCount"`
	IsOpenSource          bool   `protobuf:"varint,4,opt,name=isOpenSource" json:"isOpenSource"`
	IsBytecodeBlacklisted bool   `protobuf:"varint,5,opt,name=isBytecodeBlacklisted" json:"isBytecodeBlacklisted"`
	CreatedAt             string `protobuf:"bytes,6,opt,name=createdAt" json:"createdAt"`
	UpdatedAt             string `protobuf:"bytes,7,opt,name=updatedAt" json:"updatedAt"`
}

type BytecodeDetail struct {
	CodeHash                     string `protobuf:"bytes,1,opt,name=codeHash" json:"codeHash"`
	RuntimeBytecodeSize          int64  `protobuf:"varint,2,opt,name=runtimeBytecodeSize" json:"runtimeBytecodeSize"`
	DeploymentCount              int64  `protobuf:"varint,3,opt,name=deploymentCount" json:"deploymentCount"`
	IsOpenSource                 bool   `protobuf:"varint,4,opt,name=isOpenSource" json:"isOpenSource"`
	IsBytecodeBlacklisted        bool   `protobuf:"varint,5,opt,name=isBytecodeBlacklisted" json:"isBytecodeBlacklisted"`
	CreatedAt                    string `protobuf:"bytes,6,opt,name=createdAt" json:"createdAt"`
	UpdatedAt                    string `protobuf:"bytes,7,opt,name=updatedAt" json:"updatedAt"`
	RuntimeBytecode              string `protobuf:"bytes,8,opt,name=runtimeBytecode" json:"runtimeBytecode"`
	SourceCode                   string `protobuf:"bytes,9,opt,name=sourceCode" json:"sourceCode"`
	SourceCodeHash               string `protobuf:"bytes,10,opt,name=sourceCodeHash" json:"sourceCodeHash"`
	SourceCodeFetchedAt          string `protobuf:"bytes,11,opt,name=sourceCodeFetchedAt" json:"sourceCodeFetchedAt"`
	SourceCodeOrigin             string `protobuf:"bytes,12,opt,name=sourceCodeOrigin" json:"sourceCodeOrigin"`
	SourceQualityReport          string `protobuf:"bytes,13,opt,name=sourceQualityReport" json:"sourceQualityReport"`
	SourceQualityReportFetchedAt string `protobuf:"bytes,14,opt,name=sourceQualityReportFetchedAt" json:"sourceQualityReportFetchedAt"`
	SourceQualityReportOrigin    string `protobuf:"bytes,15,opt,name=sourceQualityReportOrigin" json:"sourceQualityReportOrigin"`
	SourceQualityPromptVersion   int64  `protobuf:"varint,16,opt,name=sourceQualityPromptVersion" json:"sourceQualityPromptVersion"`
}

type BytecodeDeployment struct {
	ChainID     int64  `protobuf:"varint,1,opt,name=chainId" json:"chainId"`
	Contract    string `protobuf:"bytes,2,opt,name=contract" json:"contract"`
	FirstSeenAt string `protobuf:"bytes,3,opt,name=firstSeenAt" json:"firstSeenAt"`
	UpdatedAt   string `protobuf:"bytes,4,opt,name=updatedAt" json:"updatedAt"`
}
