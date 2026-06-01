package v1alpha1

type PolymarketStatus struct {
	Started bool   `protobuf:"varint,1,opt,name=started" json:"started"`
	Status  string `protobuf:"bytes,2,opt,name=status" json:"status"`
}

type PolymarketSportsLiveMarketItem struct {
	ConditionID  string  `protobuf:"bytes,1,opt,name=conditionId" json:"conditionId"`
	MarketSlug   string  `protobuf:"bytes,2,opt,name=marketSlug" json:"marketSlug"`
	EventSlug    string  `protobuf:"bytes,3,opt,name=eventSlug" json:"eventSlug"`
	Title        string  `protobuf:"bytes,4,opt,name=title" json:"title"`
	Image        string  `protobuf:"bytes,5,opt,name=image" json:"image"`
	Score        string  `protobuf:"bytes,6,opt,name=score" json:"score"`
	Period       string  `protobuf:"bytes,7,opt,name=period" json:"period"`
	Elapsed      string  `protobuf:"bytes,8,opt,name=elapsed" json:"elapsed"`
	LastUpdate   string  `protobuf:"bytes,9,opt,name=lastUpdate" json:"lastUpdate"`
	LiquidityNum float64 `protobuf:"fixed64,10,opt,name=liquidityNum" json:"liquidityNum"`
	VolumeNum    float64 `protobuf:"fixed64,11,opt,name=volumeNum" json:"volumeNum"`
}
