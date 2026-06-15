package v1alpha1

type PolymarketStatus struct {
	Started bool   `protobuf:"varint,1,opt,name=started" json:"started"`
	Status  string `protobuf:"bytes,2,opt,name=status" json:"status"`
}

type PolymarketHotMarketTokenItem struct {
	TokenID string  `protobuf:"bytes,1,opt,name=tokenId" json:"tokenId"`
	Outcome string  `protobuf:"bytes,2,opt,name=outcome" json:"outcome"`
	Price   float64 `protobuf:"fixed64,3,opt,name=price" json:"price"`
}

type PolymarketHotMarketItem struct {
	ConditionID    string                          `protobuf:"bytes,1,opt,name=conditionId" json:"conditionId"`
	MarketSlug     string                          `protobuf:"bytes,2,opt,name=marketSlug" json:"marketSlug"`
	Question       string                          `protobuf:"bytes,3,opt,name=question" json:"question"`
	Image          string                          `protobuf:"bytes,4,opt,name=image" json:"image"`
	Volume24hr     float64                         `protobuf:"fixed64,5,opt,name=volume24hr" json:"volume24hr"`
	VolumeNum      float64                         `protobuf:"fixed64,6,opt,name=volumeNum" json:"volumeNum"`
	LiquidityNum   float64                         `protobuf:"fixed64,7,opt,name=liquidityNum" json:"liquidityNum"`
	Spread         float64                         `protobuf:"fixed64,8,opt,name=spread" json:"spread"`
	BestBid        float64                         `protobuf:"fixed64,9,opt,name=bestBid" json:"bestBid"`
	BestAsk        float64                         `protobuf:"fixed64,10,opt,name=bestAsk" json:"bestAsk"`
	LastTradePrice float64                         `protobuf:"fixed64,11,opt,name=lastTradePrice" json:"lastTradePrice"`
	UpdatedAt      string                          `protobuf:"bytes,12,opt,name=updatedAt" json:"updatedAt"`
	Tokens         []*PolymarketHotMarketTokenItem `protobuf:"bytes,13,rep,name=tokens" json:"tokens"`
	EventSlug      string                          `protobuf:"bytes,14,opt,name=eventSlug" json:"eventSlug"`
}

type PolymarketRealtimeWindowItem struct {
	Window        string  `protobuf:"bytes,1,opt,name=window" json:"window"`
	PriceChangePp float64 `protobuf:"fixed64,2,opt,name=priceChangePp" json:"priceChangePp"`
	Warmup        bool    `protobuf:"varint,3,opt,name=warmup" json:"warmup"`
	SampleCount   int32   `protobuf:"varint,4,opt,name=sampleCount" json:"sampleCount"`
}

type PolymarketRealtimeTokenItem struct {
	TokenID        string                          `protobuf:"bytes,1,opt,name=tokenId" json:"tokenId"`
	Outcome        string                          `protobuf:"bytes,2,opt,name=outcome" json:"outcome"`
	Price          float64                         `protobuf:"fixed64,3,opt,name=price" json:"price"`
	BestBid        float64                         `protobuf:"fixed64,4,opt,name=bestBid" json:"bestBid"`
	BestAsk        float64                         `protobuf:"fixed64,5,opt,name=bestAsk" json:"bestAsk"`
	Spread         float64                         `protobuf:"fixed64,6,opt,name=spread" json:"spread"`
	LastTradePrice float64                         `protobuf:"fixed64,7,opt,name=lastTradePrice" json:"lastTradePrice"`
	LastTradeSize  float64                         `protobuf:"fixed64,8,opt,name=lastTradeSize" json:"lastTradeSize"`
	LastTradeSide  string                          `protobuf:"bytes,9,opt,name=lastTradeSide" json:"lastTradeSide"`
	LastEventAt    int64                           `protobuf:"varint,10,opt,name=lastEventAt" json:"lastEventAt"`
	Windows        []*PolymarketRealtimeWindowItem `protobuf:"bytes,11,rep,name=windows" json:"windows"`
	Warmup         bool                            `protobuf:"varint,12,opt,name=warmup" json:"warmup"`
}

type PolymarketRealtimeMarketItem struct {
	ConditionID  string                         `protobuf:"bytes,1,opt,name=conditionId" json:"conditionId"`
	MarketSlug   string                         `protobuf:"bytes,2,opt,name=marketSlug" json:"marketSlug"`
	Question     string                         `protobuf:"bytes,3,opt,name=question" json:"question"`
	Image        string                         `protobuf:"bytes,4,opt,name=image" json:"image"`
	Volume24hr   float64                        `protobuf:"fixed64,5,opt,name=volume24hr" json:"volume24hr"`
	VolumeNum    float64                        `protobuf:"fixed64,6,opt,name=volumeNum" json:"volumeNum"`
	LiquidityNum float64                        `protobuf:"fixed64,7,opt,name=liquidityNum" json:"liquidityNum"`
	UpdatedAt    string                         `protobuf:"bytes,8,opt,name=updatedAt" json:"updatedAt"`
	Tokens       []*PolymarketRealtimeTokenItem `protobuf:"bytes,9,rep,name=tokens" json:"tokens"`
	EventSlug    string                         `protobuf:"bytes,10,opt,name=eventSlug" json:"eventSlug"`
}

type PolymarketMoverWindowItem struct {
	Window        string  `protobuf:"bytes,1,opt,name=window" json:"window"`
	PriceChangePp float64 `protobuf:"fixed64,2,opt,name=priceChangePp" json:"priceChangePp"`
	Warmup        bool    `protobuf:"varint,3,opt,name=warmup" json:"warmup"`
	SampleCount   int32   `protobuf:"varint,4,opt,name=sampleCount" json:"sampleCount"`
}

type PolymarketMoverTokenItem struct {
	TokenID        string                       `protobuf:"bytes,1,opt,name=tokenId" json:"tokenId"`
	Outcome        string                       `protobuf:"bytes,2,opt,name=outcome" json:"outcome"`
	Price          float64                      `protobuf:"fixed64,3,opt,name=price" json:"price"`
	BestBid        float64                      `protobuf:"fixed64,4,opt,name=bestBid" json:"bestBid"`
	BestAsk        float64                      `protobuf:"fixed64,5,opt,name=bestAsk" json:"bestAsk"`
	Spread         float64                      `protobuf:"fixed64,6,opt,name=spread" json:"spread"`
	LastTradePrice float64                      `protobuf:"fixed64,7,opt,name=lastTradePrice" json:"lastTradePrice"`
	LastTradeSize  float64                      `protobuf:"fixed64,8,opt,name=lastTradeSize" json:"lastTradeSize"`
	LastTradeSide  string                       `protobuf:"bytes,9,opt,name=lastTradeSide" json:"lastTradeSide"`
	LastEventAt    int64                        `protobuf:"varint,10,opt,name=lastEventAt" json:"lastEventAt"`
	Windows        []*PolymarketMoverWindowItem `protobuf:"bytes,11,rep,name=windows" json:"windows"`
	Warmup         bool                         `protobuf:"varint,12,opt,name=warmup" json:"warmup"`
	Score          float64                      `protobuf:"fixed64,13,opt,name=score" json:"score"`
	Direction      string                       `protobuf:"bytes,14,opt,name=direction" json:"direction"`
}

type PolymarketMoverMarketItem struct {
	ConditionID  string                      `protobuf:"bytes,1,opt,name=conditionId" json:"conditionId"`
	MarketSlug   string                      `protobuf:"bytes,2,opt,name=marketSlug" json:"marketSlug"`
	Question     string                      `protobuf:"bytes,3,opt,name=question" json:"question"`
	Image        string                      `protobuf:"bytes,4,opt,name=image" json:"image"`
	Volume24hr   float64                     `protobuf:"fixed64,5,opt,name=volume24hr" json:"volume24hr"`
	VolumeNum    float64                     `protobuf:"fixed64,6,opt,name=volumeNum" json:"volumeNum"`
	LiquidityNum float64                     `protobuf:"fixed64,7,opt,name=liquidityNum" json:"liquidityNum"`
	UpdatedAt    string                      `protobuf:"bytes,8,opt,name=updatedAt" json:"updatedAt"`
	Tokens       []*PolymarketMoverTokenItem `protobuf:"bytes,9,rep,name=tokens" json:"tokens"`
	EventSlug    string                      `protobuf:"bytes,10,opt,name=eventSlug" json:"eventSlug"`
	Leader       *PolymarketMoverTokenItem   `protobuf:"bytes,11,opt,name=leader" json:"leader"`
	Score        float64                     `protobuf:"fixed64,12,opt,name=score" json:"score"`
	Direction    string                      `protobuf:"bytes,13,opt,name=direction" json:"direction"`
}

type PolymarketSportsLiveMarketCardItem struct {
	MarketKey        string  `protobuf:"bytes,1,opt,name=marketKey" json:"marketKey"`
	ConditionID      string  `protobuf:"bytes,2,opt,name=conditionId" json:"conditionId"`
	Slug             string  `protobuf:"bytes,3,opt,name=slug" json:"slug"`
	Question         string  `protobuf:"bytes,4,opt,name=question" json:"question"`
	Outcomes         string  `protobuf:"bytes,5,opt,name=outcomes" json:"outcomes"`
	OutcomePrices    string  `protobuf:"bytes,6,opt,name=outcomePrices" json:"outcomePrices"`
	BestBid          float64 `protobuf:"fixed64,7,opt,name=bestBid" json:"bestBid"`
	BestAsk          float64 `protobuf:"fixed64,8,opt,name=bestAsk" json:"bestAsk"`
	LastTradePrice   float64 `protobuf:"fixed64,9,opt,name=lastTradePrice" json:"lastTradePrice"`
	Spread           float64 `protobuf:"fixed64,10,opt,name=spread" json:"spread"`
	LiquidityNum     float64 `protobuf:"fixed64,11,opt,name=liquidityNum" json:"liquidityNum"`
	VolumeNum        float64 `protobuf:"fixed64,12,opt,name=volumeNum" json:"volumeNum"`
	UpdatedAt        string  `protobuf:"bytes,13,opt,name=updatedAt" json:"updatedAt"`
	SportsMarketType string  `protobuf:"bytes,14,opt,name=sportsMarketType" json:"sportsMarketType"`
}

type PolymarketSportsLivePriceHistorySeriesItem struct {
	MarketKey  string    `protobuf:"bytes,1,opt,name=marketKey" json:"marketKey"`
	TokenID    string    `protobuf:"bytes,2,opt,name=tokenId" json:"tokenId"`
	Outcome    string    `protobuf:"bytes,3,opt,name=outcome" json:"outcome"`
	Timestamps []int64   `protobuf:"varint,4,rep,name=timestamps" json:"timestamps"`
	Prices     []float64 `protobuf:"fixed64,5,rep,name=prices" json:"prices"`
}

type PolymarketSportsLiveTeamItem struct {
	Name         string `protobuf:"bytes,1,opt,name=name" json:"name"`
	Logo         string `protobuf:"bytes,2,opt,name=logo" json:"logo"`
	Abbreviation string `protobuf:"bytes,3,opt,name=abbreviation" json:"abbreviation"`
	Alias        string `protobuf:"bytes,4,opt,name=alias" json:"alias"`
}

type PolymarketSportsLiveEventCardItem struct {
	EventKey    string                                `protobuf:"bytes,1,opt,name=eventKey" json:"eventKey"`
	EventID     string                                `protobuf:"bytes,2,opt,name=eventId" json:"eventId"`
	Slug        string                                `protobuf:"bytes,3,opt,name=slug" json:"slug"`
	Title       string                                `protobuf:"bytes,4,opt,name=title" json:"title"`
	Image       string                                `protobuf:"bytes,5,opt,name=image" json:"image"`
	Score       string                                `protobuf:"bytes,6,opt,name=score" json:"score"`
	Period      string                                `protobuf:"bytes,7,opt,name=period" json:"period"`
	Elapsed     string                                `protobuf:"bytes,8,opt,name=elapsed" json:"elapsed"`
	GameStatus  string                                `protobuf:"bytes,9,opt,name=gameStatus" json:"gameStatus"`
	StartTime   string                                `protobuf:"bytes,10,opt,name=startTime" json:"startTime"`
	UpdatedAt   string                                `protobuf:"bytes,11,opt,name=updatedAt" json:"updatedAt"`
	Liquidity   float64                               `protobuf:"fixed64,12,opt,name=liquidity" json:"liquidity"`
	Volume      float64                               `protobuf:"fixed64,13,opt,name=volume" json:"volume"`
	MarketCount int32                                 `protobuf:"varint,14,opt,name=marketCount" json:"marketCount"`
	Markets     []*PolymarketSportsLiveMarketCardItem `protobuf:"bytes,15,rep,name=markets" json:"markets"`
	Teams       []*PolymarketSportsLiveTeamItem       `protobuf:"bytes,16,rep,name=teams" json:"teams"`
}

type PolymarketSportsHistoryEventCardItem struct {
	EventKey    string                                `protobuf:"bytes,1,opt,name=eventKey" json:"eventKey"`
	EventID     string                                `protobuf:"bytes,2,opt,name=eventId" json:"eventId"`
	League      string                                `protobuf:"bytes,3,opt,name=league" json:"league"`
	Slug        string                                `protobuf:"bytes,4,opt,name=slug" json:"slug"`
	Title       string                                `protobuf:"bytes,5,opt,name=title" json:"title"`
	Image       string                                `protobuf:"bytes,6,opt,name=image" json:"image"`
	Score       string                                `protobuf:"bytes,7,opt,name=score" json:"score"`
	Period      string                                `protobuf:"bytes,8,opt,name=period" json:"period"`
	Elapsed     string                                `protobuf:"bytes,9,opt,name=elapsed" json:"elapsed"`
	GameStatus  string                                `protobuf:"bytes,10,opt,name=gameStatus" json:"gameStatus"`
	StartTime   string                                `protobuf:"bytes,11,opt,name=startTime" json:"startTime"`
	FinishedAt  string                                `protobuf:"bytes,12,opt,name=finishedAt" json:"finishedAt"`
	UpdatedAt   string                                `protobuf:"bytes,13,opt,name=updatedAt" json:"updatedAt"`
	Liquidity   float64                               `protobuf:"fixed64,14,opt,name=liquidity" json:"liquidity"`
	Volume      float64                               `protobuf:"fixed64,15,opt,name=volume" json:"volume"`
	MarketCount int32                                 `protobuf:"varint,16,opt,name=marketCount" json:"marketCount"`
	Markets     []*PolymarketSportsLiveMarketCardItem `protobuf:"bytes,17,rep,name=markets" json:"markets"`
	Teams       []*PolymarketSportsLiveTeamItem       `protobuf:"bytes,18,rep,name=teams" json:"teams"`
}
