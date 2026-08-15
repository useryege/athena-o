package v1alpha1

type WormMarketsMarketItem struct {
	ConditionID      string                                 `protobuf:"bytes,1,opt,name=conditionId" json:"conditionId"`
	Title            string                                 `protobuf:"bytes,2,opt,name=title" json:"title"`
	Description      string                                 `protobuf:"bytes,3,opt,name=description" json:"description"`
	Logo             string                                 `protobuf:"bytes,4,opt,name=logo" json:"logo"`
	LastTradePrice   string                                 `protobuf:"bytes,5,opt,name=lastTradePrice" json:"lastTradePrice"`
	State            string                                 `protobuf:"bytes,6,opt,name=state" json:"state"`
	Category         string                                 `protobuf:"bytes,7,opt,name=category" json:"category"`
	Created          int64                                  `protobuf:"varint,8,opt,name=created" json:"created"`
	EventTitle       string                                 `protobuf:"bytes,9,opt,name=eventTitle" json:"eventTitle"`
	EventConditionID string                                 `protobuf:"bytes,10,opt,name=eventConditionId" json:"eventConditionId"`
	EventLogo        string                                 `protobuf:"bytes,11,opt,name=eventLogo" json:"eventLogo"`
	MarginEnabled    bool                                   `protobuf:"varint,12,opt,name=marginEnabled" json:"marginEnabled"`
	LiveState        string                                 `protobuf:"bytes,13,opt,name=liveState" json:"liveState"`
	LiveCheckedAt    int64                                  `protobuf:"varint,14,opt,name=liveCheckedAt" json:"liveCheckedAt"`
	LivePriceChange  string                                 `protobuf:"bytes,15,opt,name=livePriceChange" json:"livePriceChange"`
	ConfigKind       string                                 `protobuf:"bytes,16,opt,name=configKind" json:"configKind"`
	MaxLeverageYes   string                                 `protobuf:"bytes,17,opt,name=maxLeverageYes" json:"maxLeverageYes"`
	MaxLeverageNo    string                                 `protobuf:"bytes,18,opt,name=maxLeverageNo" json:"maxLeverageNo"`
	OpeningFee       string                                 `protobuf:"bytes,19,opt,name=openingFee" json:"openingFee"`
	ClosingFee       string                                 `protobuf:"bytes,20,opt,name=closingFee" json:"closingFee"`
	AnnualFeeRate    string                                 `protobuf:"bytes,21,opt,name=annualFeeRate" json:"annualFeeRate"`
	OrderMinSize     string                                 `protobuf:"bytes,22,opt,name=orderMinSize" json:"orderMinSize"`
	PriceDecimals    int32                                  `protobuf:"varint,23,opt,name=priceDecimals" json:"priceDecimals"`
	SharesDecimals   int32                                  `protobuf:"varint,24,opt,name=sharesDecimals" json:"sharesDecimals"`
	Estimate         *WormMarketsMarginPositionEstimateItem `protobuf:"bytes,25,opt,name=estimate" json:"estimate"`
	TradingDataError string                                 `protobuf:"bytes,26,opt,name=tradingDataError" json:"tradingDataError"`
}

type WormMarketsMarginPositionEstimateItem struct {
	Funds            string `protobuf:"bytes,1,opt,name=funds" json:"funds"`
	IsYes            bool   `protobuf:"varint,2,opt,name=isYes" json:"isYes"`
	Leverage         string `protobuf:"bytes,3,opt,name=leverage" json:"leverage"`
	AveragePrice     string `protobuf:"bytes,4,opt,name=averagePrice" json:"averagePrice"`
	TotalShares      string `protobuf:"bytes,5,opt,name=totalShares" json:"totalShares"`
	TotalCost        string `protobuf:"bytes,6,opt,name=totalCost" json:"totalCost"`
	BestAsk          string `protobuf:"bytes,7,opt,name=bestAsk" json:"bestAsk"`
	WorstFillPrice   string `protobuf:"bytes,8,opt,name=worstFillPrice" json:"worstFillPrice"`
	IsFullyFilled    bool   `protobuf:"varint,9,opt,name=isFullyFilled" json:"isFullyFilled"`
	FeeAmount        string `protobuf:"bytes,10,opt,name=feeAmount" json:"feeAmount"`
	UserFundsNeeded  string `protobuf:"bytes,11,opt,name=userFundsNeeded" json:"userFundsNeeded"`
	LiquidationPrice string `protobuf:"bytes,12,opt,name=liquidationPrice" json:"liquidationPrice"`
}

type WormMarketsEventItem struct {
	ConditionID string                   `protobuf:"bytes,1,opt,name=conditionId" json:"conditionId"`
	Title       string                   `protobuf:"bytes,2,opt,name=title" json:"title"`
	Logo        string                   `protobuf:"bytes,3,opt,name=logo" json:"logo"`
	Live        bool                     `protobuf:"varint,4,opt,name=live" json:"live"`
	MarketCount int64                    `protobuf:"varint,5,opt,name=marketCount" json:"marketCount"`
	Markets     []*WormMarketsMarketItem `protobuf:"bytes,6,rep,name=markets" json:"markets"`
	Description string                   `protobuf:"bytes,7,opt,name=description" json:"description"`
	Category    string                   `protobuf:"bytes,8,opt,name=category" json:"category"`
	Created     int64                    `protobuf:"varint,9,opt,name=created" json:"created"`
}
