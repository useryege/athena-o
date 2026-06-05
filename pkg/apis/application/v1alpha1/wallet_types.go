package v1alpha1

type WalletStatus struct {
	Started bool   `protobuf:"varint,1,opt,name=started" json:"started"`
	Status  string `protobuf:"bytes,2,opt,name=status" json:"status"`
}

type WalletItem struct {
	ID             int64  `protobuf:"varint,1,opt,name=id" json:"id"`
	Chain          string `protobuf:"bytes,2,opt,name=chain" json:"chain"`
	Address        string `protobuf:"bytes,3,opt,name=address" json:"address"`
	Alias          string `protobuf:"bytes,4,opt,name=alias" json:"alias"`
	Source         string `protobuf:"bytes,5,opt,name=source" json:"source"`
	DerivationPath string `protobuf:"bytes,6,opt,name=derivation_path,json=derivationPath" json:"derivationPath"`
	CreatedAt      string `protobuf:"bytes,7,opt,name=created_at,json=createdAt" json:"createdAt"`
	UpdatedAt      string `protobuf:"bytes,8,opt,name=updated_at,json=updatedAt" json:"updatedAt"`
}

type WalletDetail struct {
	ID             int64  `protobuf:"varint,1,opt,name=id" json:"id"`
	Chain          string `protobuf:"bytes,2,opt,name=chain" json:"chain"`
	Address        string `protobuf:"bytes,3,opt,name=address" json:"address"`
	Alias          string `protobuf:"bytes,4,opt,name=alias" json:"alias"`
	Source         string `protobuf:"bytes,5,opt,name=source" json:"source"`
	DerivationPath string `protobuf:"bytes,6,opt,name=derivation_path,json=derivationPath" json:"derivationPath"`
	CreatedAt      string `protobuf:"bytes,7,opt,name=created_at,json=createdAt" json:"createdAt"`
	UpdatedAt      string `protobuf:"bytes,8,opt,name=updated_at,json=updatedAt" json:"updatedAt"`
	PrivateKey     string `protobuf:"bytes,9,opt,name=private_key,json=privateKey" json:"privateKey"`
	Mnemonic       string `protobuf:"bytes,10,opt,name=mnemonic" json:"mnemonic"`
}

type WalletBlacklistEntry struct {
	Wallet    string `protobuf:"bytes,1,opt,name=wallet" json:"wallet"`
	Note      string `protobuf:"bytes,2,opt,name=note" json:"note"`
	CreatedAt string `protobuf:"bytes,3,opt,name=createdAt" json:"createdAt"`
}
