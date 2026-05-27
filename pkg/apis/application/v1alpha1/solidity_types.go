package v1alpha1

type SolidityStatus struct {
	Started bool   `protobuf:"varint,1,opt,name=started" json:"started"`
	Status  string `protobuf:"bytes,2,opt,name=status" json:"status"`
}

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
}

type BytecodeBlacklistEntry struct {
	CodeHash       string `protobuf:"bytes,1,opt,name=codeHash" json:"codeHash"`
	Note           string `protobuf:"bytes,2,opt,name=note" json:"note"`
	SourceChainID  int64  `protobuf:"varint,3,opt,name=sourceChainId" json:"sourceChainId"`
	SourceContract string `protobuf:"bytes,4,opt,name=sourceContract" json:"sourceContract"`
	CreatedAt      string `protobuf:"bytes,5,opt,name=createdAt" json:"createdAt"`
}
