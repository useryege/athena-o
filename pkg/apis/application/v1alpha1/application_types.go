package v1alpha1

type ProjectOption struct {
	FactoryContract string `protobuf:"bytes,1,opt,name=factoryContract" json:"factoryContract"`
	WethContract    string `protobuf:"bytes,2,opt,name=wethContract" json:"wethContract"`
	UsdtContract    string `protobuf:"bytes,3,opt,name=usdtContract" json:"usdtContract"`
	WethDecimals    uint32 `protobuf:"varint,4,opt,name=wethDecimals" json:"wethDecimals"`
	UsdtDecimals    uint32 `protobuf:"varint,5,opt,name=usdtDecimals" json:"usdtDecimals"`
}

type ApplicationChain struct {
	ChainID int64  `protobuf:"varint,1,opt,name=chainID" json:"chainID"`
	Name    string `protobuf:"bytes,2,opt,name=name" json:"name"`
	Enabled bool   `protobuf:"varint,3,opt,name=enabled" json:"enabled"`
}

type ChainIngestStatus struct {
	ChainID              int64  `protobuf:"varint,1,opt,name=chainID" json:"chainID"`
	Name                 string `protobuf:"bytes,2,opt,name=name" json:"name"`
	Enabled              bool   `protobuf:"varint,3,opt,name=enabled" json:"enabled"`
	Status               string `protobuf:"bytes,4,opt,name=status" json:"status"`
	FinalizedBlockNumber uint64 `protobuf:"varint,5,opt,name=finalizedBlockNumber" json:"finalizedBlockNumber"`
	FinalizedBlockHash   string `protobuf:"bytes,6,opt,name=finalizedBlockHash" json:"finalizedBlockHash"`
	CursorBlockNumber    uint64 `protobuf:"varint,7,opt,name=cursorBlockNumber" json:"cursorBlockNumber"`
	CursorBlockHash      string `protobuf:"bytes,8,opt,name=cursorBlockHash" json:"cursorBlockHash"`
	UpdatedAt            string `protobuf:"bytes,9,opt,name=updatedAt" json:"updatedAt"`
}

type ProjectComponentStatus struct {
	Component     string `protobuf:"bytes,1,opt,name=component" json:"component"`
	Status        string `protobuf:"bytes,2,opt,name=status" json:"status"`
	LastAttemptAt string `protobuf:"bytes,3,opt,name=lastAttemptAt" json:"lastAttemptAt"`
	LastSuccessAt string `protobuf:"bytes,4,opt,name=lastSuccessAt" json:"lastSuccessAt"`
	NextRunAt     string `protobuf:"bytes,5,opt,name=nextRunAt" json:"nextRunAt"`
	LastError     string `protobuf:"bytes,6,opt,name=lastError" json:"lastError"`
	UpdatedAt     string `protobuf:"bytes,7,opt,name=updatedAt" json:"updatedAt"`
}

type ProjectCollectionStatus struct {
	ChainID         int64                     `protobuf:"varint,1,opt,name=chainID" json:"chainID"`
	Contract        string                    `protobuf:"bytes,2,opt,name=contract" json:"contract"`
	Status          string                    `protobuf:"bytes,3,opt,name=status" json:"status"`
	WorkflowID      string                    `protobuf:"bytes,4,opt,name=workflowID" json:"workflowID"`
	LastRequestedAt string                    `protobuf:"bytes,5,opt,name=lastRequestedAt" json:"lastRequestedAt"`
	LastStartedAt   string                    `protobuf:"bytes,6,opt,name=lastStartedAt" json:"lastStartedAt"`
	LastCompletedAt string                    `protobuf:"bytes,7,opt,name=lastCompletedAt" json:"lastCompletedAt"`
	NextRunAt       string                    `protobuf:"bytes,8,opt,name=nextRunAt" json:"nextRunAt"`
	LastError       string                    `protobuf:"bytes,9,opt,name=lastError" json:"lastError"`
	UpdatedAt       string                    `protobuf:"bytes,10,opt,name=updatedAt" json:"updatedAt"`
	Components      []*ProjectComponentStatus `protobuf:"bytes,11,rep,name=components" json:"components"`
}
