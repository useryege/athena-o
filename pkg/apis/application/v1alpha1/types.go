package v1alpha1

type ProjectView struct {
	Meta       ProjectMeta            `protobuf:"bytes,1,opt,name=meta" json:"meta"`
	Token      TokenState             `protobuf:"bytes,2,opt,name=token" json:"token"`
	WethV2Pool PairV2State            `protobuf:"bytes,3,opt,name=wethV2Pool" json:"wethV2Pool"`
	SourceCode ProjectSourceCodeState `protobuf:"bytes,4,opt,name=sourceCode" json:"sourceCode"`
	Simulate   ProjectSimulateState   `protobuf:"bytes,5,opt,name=simulate" json:"simulate"`
	Analysis   ProjectAnalysisState   `protobuf:"bytes,6,opt,name=analysis" json:"analysis"`
}

type ProjectMeta struct {
	ProjectID   string `protobuf:"bytes,1,opt,name=projectID" json:"projectID"`
	BlockTime   uint64 `protobuf:"varint,2,opt,name=blockTime" json:"blockTime"`
	BlockNumber uint64 `protobuf:"varint,3,opt,name=blockNumber" json:"blockNumber"`
	Contract    string `protobuf:"bytes,4,opt,name=contract" json:"contract"`
	Creator     string `protobuf:"bytes,5,opt,name=creator" json:"creator"`
	TxHash      string `protobuf:"bytes,6,opt,name=txHash" json:"txHash"`
	TxIndex     uint64 `protobuf:"varint,7,opt,name=txIndex" json:"txIndex"`
}

type TokenState struct {
	Name          string `protobuf:"bytes,1,opt,name=name" json:"name"`
	Symbol        string `protobuf:"bytes,2,opt,name=symbol" json:"symbol"`
	Decimals      uint32 `protobuf:"varint,3,opt,name=decimals" json:"decimals"`
	TotalSupply   string `protobuf:"bytes,4,opt,name=totalSupply" json:"totalSupply"`
	SourceCode    string `protobuf:"bytes,5,opt,name=sourceCode" json:"sourceCode"`
	SourceCodeABI string `protobuf:"bytes,6,opt,name=sourceCodeABI" json:"sourceCodeABI"`
	IsValidERC20  bool   `protobuf:"varint,7,opt,name=isValidERC20" json:"isValidERC20"`
}

type PairV2State struct {
	IsContractCreated   bool   `protobuf:"varint,1,opt,name=isContractCreated" json:"isContractCreated"`
	Contract            string `protobuf:"bytes,2,opt,name=contract" json:"contract"`
	Token0              string `protobuf:"bytes,3,opt,name=token0" json:"token0"`
	Token1              string `protobuf:"bytes,4,opt,name=token1" json:"token1"`
	TotalSupply         string `protobuf:"bytes,5,opt,name=totalSupply" json:"totalSupply"`
	Reserve0            string `protobuf:"bytes,6,opt,name=reserve0" json:"reserve0"`
	Reserve1            string `protobuf:"bytes,7,opt,name=reserve1" json:"reserve1"`
	BlockTimestampLast  uint32 `protobuf:"varint,8,opt,name=blockTimestampLast" json:"blockTimestampLast"`
	TokenReserveBalance string `protobuf:"bytes,9,opt,name=tokenReserveBalance" json:"tokenReserveBalance"`
	WethReserveBalance  string `protobuf:"bytes,10,opt,name=wethReserveBalance" json:"wethReserveBalance"`
	LockedLiquidity     string `protobuf:"bytes,11,opt,name=lockedLiquidity" json:"lockedLiquidity"`
}

type ProjectSourceCodeState struct {
	SourceCode    string `protobuf:"bytes,1,opt,name=sourceCode" json:"sourceCode"`
	SourceCodeABI string `protobuf:"bytes,2,opt,name=sourceCodeABI" json:"sourceCodeABI"`
}

type ProjectSimulateState struct {
	CreatorResult SimulateResult `protobuf:"bytes,1,opt,name=creatorResult" json:"creatorResult"`
}

type SimulateResult struct {
	CanMintFromDeadViaTransferFrom bool `protobuf:"varint,1,opt,name=canMintFromDeadViaTransferFrom" json:"canMintFromDeadViaTransferFrom"`
	CanMintFromZeroViaTransferFrom bool `protobuf:"varint,2,opt,name=canMintFromZeroViaTransferFrom" json:"canMintFromZeroViaTransferFrom"`
	CanMintFromPairViaTransferFrom bool `protobuf:"varint,3,opt,name=canMintFromPairViaTransferFrom" json:"canMintFromPairViaTransferFrom"`
	CanMintViaTransfer             bool `protobuf:"varint,4,opt,name=canMintViaTransfer" json:"canMintViaTransfer"`
}

type ProjectAnalysisState struct {
	SourceCodeBlacklist SourceCodeBlacklistState `protobuf:"bytes,1,opt,name=sourceCodeBlacklist" json:"sourceCodeBlacklist"`
}

type SourceCodeBlacklistState struct {
	HasBlacklistFields bool     `protobuf:"varint,1,opt,name=hasBlacklistFields" json:"hasBlacklistFields"`
	BlacklistFields    []string `protobuf:"bytes,2,rep,name=blacklistFields" json:"blacklistFields"`
}
