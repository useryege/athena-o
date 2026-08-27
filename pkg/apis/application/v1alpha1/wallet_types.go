package v1alpha1

type WalletStatus struct {
	Started bool   `protobuf:"varint,1,opt,name=started" json:"started"`
	Status  string `protobuf:"bytes,2,opt,name=status" json:"status"`
}

type WalletItem struct {
	ID             int64  `protobuf:"varint,1,opt,name=id" json:"id"`
	WalletType     string `protobuf:"bytes,2,opt,name=wallet_type,json=walletType" json:"walletType"`
	Address        string `protobuf:"bytes,3,opt,name=address" json:"address"`
	Remark         string `protobuf:"bytes,4,opt,name=remark" json:"remark"`
	Source         string `protobuf:"bytes,5,opt,name=source" json:"source"`
	AvatarKind     string `protobuf:"bytes,6,opt,name=avatar_kind,json=avatarKind" json:"avatarKind"`
	AvatarPresetID string `protobuf:"bytes,7,opt,name=avatar_preset_id,json=avatarPresetId" json:"avatarPresetId"`
	AvatarURL      string `protobuf:"bytes,8,opt,name=avatar_url,json=avatarUrl" json:"avatarUrl"`
	Revision       uint64 `protobuf:"varint,9,opt,name=revision" json:"revision"`
	CreatedAt      string `protobuf:"bytes,10,opt,name=created_at,json=createdAt" json:"createdAt"`
	UpdatedAt      string `protobuf:"bytes,11,opt,name=updated_at,json=updatedAt" json:"updatedAt"`
}
