package v1alpha1

type ProjectDiscoveryStatus struct {
	Started bool   `protobuf:"varint,1,opt,name=started" json:"started"`
	Status  string `protobuf:"bytes,2,opt,name=status" json:"status"`
}

type ProjectOption struct {
	FactoryContract string `protobuf:"bytes,1,opt,name=factoryContract" json:"factoryContract"`
	WethContract    string `protobuf:"bytes,2,opt,name=wethContract" json:"wethContract"`
	UsdtContract    string `protobuf:"bytes,3,opt,name=usdtContract" json:"usdtContract"`
	WethDecimals    uint32 `protobuf:"varint,4,opt,name=wethDecimals" json:"wethDecimals"`
	UsdtDecimals    uint32 `protobuf:"varint,5,opt,name=usdtDecimals" json:"usdtDecimals"`
}
