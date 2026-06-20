package v1alpha1

type WormMarketItem struct {
	ConditionID      string `protobuf:"bytes,1,opt,name=conditionId" json:"conditionId"`
	Title            string `protobuf:"bytes,2,opt,name=title" json:"title"`
	Description      string `protobuf:"bytes,3,opt,name=description" json:"description"`
	Logo             string `protobuf:"bytes,4,opt,name=logo" json:"logo"`
	LastTradePrice   string `protobuf:"bytes,5,opt,name=lastTradePrice" json:"lastTradePrice"`
	State            string `protobuf:"bytes,6,opt,name=state" json:"state"`
	Category         string `protobuf:"bytes,7,opt,name=category" json:"category"`
	Created          int64  `protobuf:"varint,8,opt,name=created" json:"created"`
	EventTitle       string `protobuf:"bytes,9,opt,name=eventTitle" json:"eventTitle"`
	EventConditionID string `protobuf:"bytes,10,opt,name=eventConditionId" json:"eventConditionId"`
	EventLogo        string `protobuf:"bytes,11,opt,name=eventLogo" json:"eventLogo"`
	MarginEnabled    bool   `protobuf:"varint,12,opt,name=marginEnabled" json:"marginEnabled"`
	LiveState        string `protobuf:"bytes,13,opt,name=liveState" json:"liveState"`
	LiveCheckedAt    int64  `protobuf:"varint,14,opt,name=liveCheckedAt" json:"liveCheckedAt"`
	LivePriceChange  string `protobuf:"bytes,15,opt,name=livePriceChange" json:"livePriceChange"`
}

type WormEventItem struct {
	ConditionID string            `protobuf:"bytes,1,opt,name=conditionId" json:"conditionId"`
	Title       string            `protobuf:"bytes,2,opt,name=title" json:"title"`
	Logo        string            `protobuf:"bytes,3,opt,name=logo" json:"logo"`
	Live        bool              `protobuf:"varint,4,opt,name=live" json:"live"`
	MarketCount int64             `protobuf:"varint,5,opt,name=marketCount" json:"marketCount"`
	Markets     []*WormMarketItem `protobuf:"bytes,6,rep,name=markets" json:"markets"`
	Description string            `protobuf:"bytes,7,opt,name=description" json:"description"`
	Category    string            `protobuf:"bytes,8,opt,name=category" json:"category"`
	Created     int64             `protobuf:"varint,9,opt,name=created" json:"created"`
}
