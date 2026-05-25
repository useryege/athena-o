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
}

type WormMarketDetail struct {
	Market          WormMarketItem        `protobuf:"bytes,1,opt,name=market" json:"market"`
	YesOutcomeLabel string                `protobuf:"bytes,2,opt,name=yesOutcomeLabel" json:"yesOutcomeLabel"`
	NoOutcomeLabel  string                `protobuf:"bytes,3,opt,name=noOutcomeLabel" json:"noOutcomeLabel"`
	Outcomes        []WormMarketOutcome   `protobuf:"bytes,4,rep,name=outcomes" json:"outcomes"`
	Rules           []string              `protobuf:"bytes,5,rep,name=rules" json:"rules"`
	ResolutionDate  int64                 `protobuf:"varint,6,opt,name=resolutionDate" json:"resolutionDate"`
	MakerFee        string                `protobuf:"bytes,7,opt,name=makerFee" json:"makerFee"`
	TakerFee        string                `protobuf:"bytes,8,opt,name=takerFee" json:"takerFee"`
	Config          WormMarketConfig      `protobuf:"bytes,9,opt,name=config" json:"config"`
	Stats           WormMarketStats       `protobuf:"bytes,10,opt,name=stats" json:"stats"`
	Prices          []WormMarketPrice     `protobuf:"bytes,11,rep,name=prices" json:"prices"`
	OrderBooks      []WormMarketOrderBook `protobuf:"bytes,12,rep,name=orderBooks" json:"orderBooks"`
}

type WormMarketOutcome struct {
	IsYes bool   `protobuf:"varint,1,opt,name=isYes" json:"isYes"`
	Text  string `protobuf:"bytes,2,opt,name=text" json:"text"`
}

type WormMarketConfig struct {
	Kind                string `protobuf:"bytes,1,opt,name=kind" json:"kind"`
	MaxLeverage         string `protobuf:"bytes,2,opt,name=maxLeverage" json:"maxLeverage"`
	OpeningFee          string `protobuf:"bytes,3,opt,name=openingFee" json:"openingFee"`
	ClosingFee          string `protobuf:"bytes,4,opt,name=closingFee" json:"closingFee"`
	AnnualFeeRate       string `protobuf:"bytes,5,opt,name=annualFeeRate" json:"annualFeeRate"`
	OrderMinSize        string `protobuf:"bytes,6,opt,name=orderMinSize" json:"orderMinSize"`
	PriceDecimals       int32  `protobuf:"varint,7,opt,name=priceDecimals" json:"priceDecimals"`
	SharesDecimals      int32  `protobuf:"varint,8,opt,name=sharesDecimals" json:"sharesDecimals"`
	MinPrice            string `protobuf:"bytes,9,opt,name=minPrice" json:"minPrice"`
	MaxPrice            string `protobuf:"bytes,10,opt,name=maxPrice" json:"maxPrice"`
	MinAmount           string `protobuf:"bytes,11,opt,name=minAmount" json:"minAmount"`
	MaxAmount           string `protobuf:"bytes,12,opt,name=maxAmount" json:"maxAmount"`
	MinFunds            string `protobuf:"bytes,13,opt,name=minFunds" json:"minFunds"`
	MaxFunds            string `protobuf:"bytes,14,opt,name=maxFunds" json:"maxFunds"`
	PricePrecision      int32  `protobuf:"varint,15,opt,name=pricePrecision" json:"pricePrecision"`
	AmountPrecision     int32  `protobuf:"varint,16,opt,name=amountPrecision" json:"amountPrecision"`
	FundsPrecision      int32  `protobuf:"varint,17,opt,name=fundsPrecision" json:"fundsPrecision"`
	MakerFeeRate        string `protobuf:"bytes,18,opt,name=makerFeeRate" json:"makerFeeRate"`
	TakerFeeRate        string `protobuf:"bytes,19,opt,name=takerFeeRate" json:"takerFeeRate"`
	DefaultSlippageRate string `protobuf:"bytes,20,opt,name=defaultSlippageRate" json:"defaultSlippageRate"`
}

type WormMarketStats struct {
	TotalVolume    string `protobuf:"bytes,1,opt,name=totalVolume" json:"totalVolume"`
	TotalVolume24H string `protobuf:"bytes,2,opt,name=totalVolume24h" json:"totalVolume24H"`
	MarketCap      string `protobuf:"bytes,3,opt,name=marketCap" json:"marketCap"`
	TradeCount     int64  `protobuf:"varint,4,opt,name=tradeCount" json:"tradeCount"`
}

type WormMarketPrice struct {
	ConditionID string `protobuf:"bytes,1,opt,name=conditionId" json:"conditionId"`
	Price       string `protobuf:"bytes,2,opt,name=price" json:"price"`
	PriceKind   string `protobuf:"bytes,3,opt,name=priceKind" json:"priceKind"`
	IsYes       bool   `protobuf:"varint,4,opt,name=isYes" json:"isYes"`
}

type WormMarketOrderBook struct {
	Market string               `protobuf:"bytes,1,opt,name=market" json:"market"`
	IsYes  bool                 `protobuf:"varint,2,opt,name=isYes" json:"isYes"`
	Bid    []WormOrderBookLevel `protobuf:"bytes,3,rep,name=bid" json:"bid"`
	Ask    []WormOrderBookLevel `protobuf:"bytes,4,rep,name=ask" json:"ask"`
}

type WormOrderBookLevel struct {
	Price       string `protobuf:"bytes,1,opt,name=price" json:"price"`
	TotalAmount string `protobuf:"bytes,2,opt,name=totalAmount" json:"totalAmount"`
}
