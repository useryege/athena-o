package v1alpha1

type ManagedOOProposalItem struct {
	TxHash              string `protobuf:"bytes,1,opt,name=txHash" json:"txHash"`
	LogIndex            int64  `protobuf:"varint,2,opt,name=logIndex" json:"logIndex"`
	BlockNumber         int64  `protobuf:"varint,3,opt,name=blockNumber" json:"blockNumber"`
	BlockHash           string `protobuf:"bytes,4,opt,name=blockHash" json:"blockHash"`
	TxIndex             int64  `protobuf:"varint,5,opt,name=txIndex" json:"txIndex"`
	ContractAddress     string `protobuf:"bytes,6,opt,name=contractAddress" json:"contractAddress"`
	Topic               string `protobuf:"bytes,7,opt,name=topic" json:"topic"`
	Requester           string `protobuf:"bytes,8,opt,name=requester" json:"requester"`
	Proposer            string `protobuf:"bytes,9,opt,name=proposer" json:"proposer"`
	Identifier          string `protobuf:"bytes,10,opt,name=identifier" json:"identifier"`
	RequestTimestamp    int64  `protobuf:"varint,11,opt,name=requestTimestamp" json:"requestTimestamp"`
	AncillaryDataHex    string `protobuf:"bytes,12,opt,name=ancillaryDataHex" json:"ancillaryDataHex"`
	AncillaryDataText   string `protobuf:"bytes,13,opt,name=ancillaryDataText" json:"ancillaryDataText"`
	MarketID            string `protobuf:"bytes,14,opt,name=marketId" json:"marketId"`
	ProposedPrice       string `protobuf:"bytes,15,opt,name=proposedPrice" json:"proposedPrice"`
	ExpirationTimestamp int64  `protobuf:"varint,16,opt,name=expirationTimestamp" json:"expirationTimestamp"`
	Currency            string `protobuf:"bytes,17,opt,name=currency" json:"currency"`
	RawTopics           string `protobuf:"bytes,18,opt,name=rawTopics" json:"rawTopics"`
	RawData             string `protobuf:"bytes,19,opt,name=rawData" json:"rawData"`
	FetchedAt           string `protobuf:"bytes,20,opt,name=fetchedAt" json:"fetchedAt"`
	ConditionID         string `protobuf:"bytes,21,opt,name=conditionId" json:"conditionId"`
	EventSlug           string `protobuf:"bytes,22,opt,name=eventSlug" json:"eventSlug"`
	MarketSlug          string `protobuf:"bytes,23,opt,name=marketSlug" json:"marketSlug"`
	Question            string `protobuf:"bytes,24,opt,name=question" json:"question"`
	PolymarketURL       string `protobuf:"bytes,25,opt,name=polymarketUrl" json:"polymarketUrl"`
}

type ManagedOODisputeItem struct {
	TxHash            string `protobuf:"bytes,1,opt,name=txHash" json:"txHash"`
	LogIndex          int64  `protobuf:"varint,2,opt,name=logIndex" json:"logIndex"`
	BlockNumber       int64  `protobuf:"varint,3,opt,name=blockNumber" json:"blockNumber"`
	BlockHash         string `protobuf:"bytes,4,opt,name=blockHash" json:"blockHash"`
	TxIndex           int64  `protobuf:"varint,5,opt,name=txIndex" json:"txIndex"`
	ContractAddress   string `protobuf:"bytes,6,opt,name=contractAddress" json:"contractAddress"`
	Topic             string `protobuf:"bytes,7,opt,name=topic" json:"topic"`
	Requester         string `protobuf:"bytes,8,opt,name=requester" json:"requester"`
	Proposer          string `protobuf:"bytes,9,opt,name=proposer" json:"proposer"`
	Disputer          string `protobuf:"bytes,10,opt,name=disputer" json:"disputer"`
	Identifier        string `protobuf:"bytes,11,opt,name=identifier" json:"identifier"`
	RequestTimestamp  int64  `protobuf:"varint,12,opt,name=requestTimestamp" json:"requestTimestamp"`
	AncillaryDataHex  string `protobuf:"bytes,13,opt,name=ancillaryDataHex" json:"ancillaryDataHex"`
	AncillaryDataText string `protobuf:"bytes,14,opt,name=ancillaryDataText" json:"ancillaryDataText"`
	MarketID          string `protobuf:"bytes,15,opt,name=marketId" json:"marketId"`
	ProposedPrice     string `protobuf:"bytes,16,opt,name=proposedPrice" json:"proposedPrice"`
	RawTopics         string `protobuf:"bytes,17,opt,name=rawTopics" json:"rawTopics"`
	RawData           string `protobuf:"bytes,18,opt,name=rawData" json:"rawData"`
	FetchedAt         string `protobuf:"bytes,19,opt,name=fetchedAt" json:"fetchedAt"`
	ConditionID       string `protobuf:"bytes,20,opt,name=conditionId" json:"conditionId"`
	EventSlug         string `protobuf:"bytes,21,opt,name=eventSlug" json:"eventSlug"`
	MarketSlug        string `protobuf:"bytes,22,opt,name=marketSlug" json:"marketSlug"`
	Question          string `protobuf:"bytes,23,opt,name=question" json:"question"`
	PolymarketURL     string `protobuf:"bytes,24,opt,name=polymarketUrl" json:"polymarketUrl"`
}

type MarketRadarHotMarketTokenItem struct {
	TokenID string  `protobuf:"bytes,1,opt,name=tokenId" json:"tokenId"`
	Outcome string  `protobuf:"bytes,2,opt,name=outcome" json:"outcome"`
	Price   float64 `protobuf:"fixed64,3,opt,name=price" json:"price"`
}

type MarketRadarHotMarketItem struct {
	ConditionID    string                           `protobuf:"bytes,1,opt,name=conditionId" json:"conditionId"`
	MarketSlug     string                           `protobuf:"bytes,2,opt,name=marketSlug" json:"marketSlug"`
	Question       string                           `protobuf:"bytes,3,opt,name=question" json:"question"`
	Image          string                           `protobuf:"bytes,4,opt,name=image" json:"image"`
	Volume24hr     float64                          `protobuf:"fixed64,5,opt,name=volume24hr" json:"volume24hr"`
	VolumeNum      float64                          `protobuf:"fixed64,6,opt,name=volumeNum" json:"volumeNum"`
	LiquidityNum   float64                          `protobuf:"fixed64,7,opt,name=liquidityNum" json:"liquidityNum"`
	Spread         float64                          `protobuf:"fixed64,8,opt,name=spread" json:"spread"`
	BestBid        float64                          `protobuf:"fixed64,9,opt,name=bestBid" json:"bestBid"`
	BestAsk        float64                          `protobuf:"fixed64,10,opt,name=bestAsk" json:"bestAsk"`
	LastTradePrice float64                          `protobuf:"fixed64,11,opt,name=lastTradePrice" json:"lastTradePrice"`
	UpdatedAt      string                           `protobuf:"bytes,12,opt,name=updatedAt" json:"updatedAt"`
	Tokens         []*MarketRadarHotMarketTokenItem `protobuf:"bytes,13,rep,name=tokens" json:"tokens"`
	EventSlug      string                           `protobuf:"bytes,14,opt,name=eventSlug" json:"eventSlug"`
}

type MarketRadarRealtimeWindowItem struct {
	Window        string  `protobuf:"bytes,1,opt,name=window" json:"window"`
	PriceChangePp float64 `protobuf:"fixed64,2,opt,name=priceChangePp" json:"priceChangePp"`
	Warmup        bool    `protobuf:"varint,3,opt,name=warmup" json:"warmup"`
	SampleCount   int32   `protobuf:"varint,4,opt,name=sampleCount" json:"sampleCount"`
}

type MarketRadarRealtimeTokenItem struct {
	TokenID        string                           `protobuf:"bytes,1,opt,name=tokenId" json:"tokenId"`
	Outcome        string                           `protobuf:"bytes,2,opt,name=outcome" json:"outcome"`
	Price          float64                          `protobuf:"fixed64,3,opt,name=price" json:"price"`
	BestBid        float64                          `protobuf:"fixed64,4,opt,name=bestBid" json:"bestBid"`
	BestAsk        float64                          `protobuf:"fixed64,5,opt,name=bestAsk" json:"bestAsk"`
	Spread         float64                          `protobuf:"fixed64,6,opt,name=spread" json:"spread"`
	LastTradePrice float64                          `protobuf:"fixed64,7,opt,name=lastTradePrice" json:"lastTradePrice"`
	LastTradeSize  float64                          `protobuf:"fixed64,8,opt,name=lastTradeSize" json:"lastTradeSize"`
	LastTradeSide  string                           `protobuf:"bytes,9,opt,name=lastTradeSide" json:"lastTradeSide"`
	LastEventAt    int64                            `protobuf:"varint,10,opt,name=lastEventAt" json:"lastEventAt"`
	Windows        []*MarketRadarRealtimeWindowItem `protobuf:"bytes,11,rep,name=windows" json:"windows"`
	Warmup         bool                             `protobuf:"varint,12,opt,name=warmup" json:"warmup"`
}

type MarketRadarRealtimeMarketItem struct {
	ConditionID  string                          `protobuf:"bytes,1,opt,name=conditionId" json:"conditionId"`
	MarketSlug   string                          `protobuf:"bytes,2,opt,name=marketSlug" json:"marketSlug"`
	Question     string                          `protobuf:"bytes,3,opt,name=question" json:"question"`
	Image        string                          `protobuf:"bytes,4,opt,name=image" json:"image"`
	Volume24hr   float64                         `protobuf:"fixed64,5,opt,name=volume24hr" json:"volume24hr"`
	VolumeNum    float64                         `protobuf:"fixed64,6,opt,name=volumeNum" json:"volumeNum"`
	LiquidityNum float64                         `protobuf:"fixed64,7,opt,name=liquidityNum" json:"liquidityNum"`
	UpdatedAt    string                          `protobuf:"bytes,8,opt,name=updatedAt" json:"updatedAt"`
	Tokens       []*MarketRadarRealtimeTokenItem `protobuf:"bytes,9,rep,name=tokens" json:"tokens"`
	EventSlug    string                          `protobuf:"bytes,10,opt,name=eventSlug" json:"eventSlug"`
}

type MarketRadarMoverWindowItem struct {
	Window        string  `protobuf:"bytes,1,opt,name=window" json:"window"`
	PriceChangePp float64 `protobuf:"fixed64,2,opt,name=priceChangePp" json:"priceChangePp"`
	Warmup        bool    `protobuf:"varint,3,opt,name=warmup" json:"warmup"`
	SampleCount   int32   `protobuf:"varint,4,opt,name=sampleCount" json:"sampleCount"`
}

type MarketRadarMoverTokenItem struct {
	TokenID        string                        `protobuf:"bytes,1,opt,name=tokenId" json:"tokenId"`
	Outcome        string                        `protobuf:"bytes,2,opt,name=outcome" json:"outcome"`
	Price          float64                       `protobuf:"fixed64,3,opt,name=price" json:"price"`
	BestBid        float64                       `protobuf:"fixed64,4,opt,name=bestBid" json:"bestBid"`
	BestAsk        float64                       `protobuf:"fixed64,5,opt,name=bestAsk" json:"bestAsk"`
	Spread         float64                       `protobuf:"fixed64,6,opt,name=spread" json:"spread"`
	LastTradePrice float64                       `protobuf:"fixed64,7,opt,name=lastTradePrice" json:"lastTradePrice"`
	LastTradeSize  float64                       `protobuf:"fixed64,8,opt,name=lastTradeSize" json:"lastTradeSize"`
	LastTradeSide  string                        `protobuf:"bytes,9,opt,name=lastTradeSide" json:"lastTradeSide"`
	LastEventAt    int64                         `protobuf:"varint,10,opt,name=lastEventAt" json:"lastEventAt"`
	Windows        []*MarketRadarMoverWindowItem `protobuf:"bytes,11,rep,name=windows" json:"windows"`
	Warmup         bool                          `protobuf:"varint,12,opt,name=warmup" json:"warmup"`
	Score          float64                       `protobuf:"fixed64,13,opt,name=score" json:"score"`
	Direction      string                        `protobuf:"bytes,14,opt,name=direction" json:"direction"`
}

type MarketRadarMoverMarketItem struct {
	ConditionID  string                       `protobuf:"bytes,1,opt,name=conditionId" json:"conditionId"`
	MarketSlug   string                       `protobuf:"bytes,2,opt,name=marketSlug" json:"marketSlug"`
	Question     string                       `protobuf:"bytes,3,opt,name=question" json:"question"`
	Image        string                       `protobuf:"bytes,4,opt,name=image" json:"image"`
	Volume24hr   float64                      `protobuf:"fixed64,5,opt,name=volume24hr" json:"volume24hr"`
	VolumeNum    float64                      `protobuf:"fixed64,6,opt,name=volumeNum" json:"volumeNum"`
	LiquidityNum float64                      `protobuf:"fixed64,7,opt,name=liquidityNum" json:"liquidityNum"`
	UpdatedAt    string                       `protobuf:"bytes,8,opt,name=updatedAt" json:"updatedAt"`
	Tokens       []*MarketRadarMoverTokenItem `protobuf:"bytes,9,rep,name=tokens" json:"tokens"`
	EventSlug    string                       `protobuf:"bytes,10,opt,name=eventSlug" json:"eventSlug"`
	Leader       *MarketRadarMoverTokenItem   `protobuf:"bytes,11,opt,name=leader" json:"leader"`
	Score        float64                      `protobuf:"fixed64,12,opt,name=score" json:"score"`
	Direction    string                       `protobuf:"bytes,13,opt,name=direction" json:"direction"`
}
