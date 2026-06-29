package v1alpha1

type PolymarketStatus struct {
	Started bool   `protobuf:"varint,1,opt,name=started" json:"started"`
	Status  string `protobuf:"bytes,2,opt,name=status" json:"status"`
}

type PolymarketUMAProposalItem struct {
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

type PolymarketUMADisputeItem struct {
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

type PolymarketFIFAMoneylineDirectionItem struct {
	TokenID  string  `protobuf:"bytes,1,opt,name=tokenId" json:"tokenId"`
	MidPrice float64 `protobuf:"fixed64,2,opt,name=midPrice" json:"midPrice"`
	BestBid  float64 `protobuf:"fixed64,3,opt,name=bestBid" json:"bestBid"`
	BestAsk  float64 `protobuf:"fixed64,4,opt,name=bestAsk" json:"bestAsk"`
	Spread   float64 `protobuf:"fixed64,5,opt,name=spread" json:"spread"`
}

type PolymarketFIFAMoneylineOptionItem struct {
	OutcomeKey      string                                `protobuf:"bytes,1,opt,name=outcomeKey" json:"outcomeKey"`
	OutcomeLabel    string                                `protobuf:"bytes,2,opt,name=outcomeLabel" json:"outcomeLabel"`
	MarketID        string                                `protobuf:"bytes,3,opt,name=marketId" json:"marketId"`
	MarketSlug      string                                `protobuf:"bytes,4,opt,name=marketSlug" json:"marketSlug"`
	Question        string                                `protobuf:"bytes,5,opt,name=question" json:"question"`
	ConditionID     string                                `protobuf:"bytes,6,opt,name=conditionId" json:"conditionId"`
	Yes             *PolymarketFIFAMoneylineDirectionItem `protobuf:"bytes,7,opt,name=yes" json:"yes"`
	No              *PolymarketFIFAMoneylineDirectionItem `protobuf:"bytes,8,opt,name=no" json:"no"`
	OrderMinSize    float64                               `protobuf:"fixed64,9,opt,name=orderMinSize" json:"orderMinSize"`
	TickSize        float64                               `protobuf:"fixed64,10,opt,name=tickSize" json:"tickSize"`
	EnableOrderBook bool                                  `protobuf:"varint,11,opt,name=enableOrderBook" json:"enableOrderBook"`
	AcceptingOrders bool                                  `protobuf:"varint,12,opt,name=acceptingOrders" json:"acceptingOrders"`
	NegRisk         bool                                  `protobuf:"varint,13,opt,name=negRisk" json:"negRisk"`
}

type PolymarketFIFAMoneylineEventItem struct {
	EventID       string                               `protobuf:"bytes,1,opt,name=eventId" json:"eventId"`
	EventSlug     string                               `protobuf:"bytes,2,opt,name=eventSlug" json:"eventSlug"`
	Title         string                               `protobuf:"bytes,3,opt,name=title" json:"title"`
	Image         string                               `protobuf:"bytes,4,opt,name=image" json:"image"`
	Sport         string                               `protobuf:"bytes,5,opt,name=sport" json:"sport"`
	Score         string                               `protobuf:"bytes,6,opt,name=score" json:"score"`
	GameStatus    string                               `protobuf:"bytes,7,opt,name=gameStatus" json:"gameStatus"`
	StartTime     string                               `protobuf:"bytes,8,opt,name=startTime" json:"startTime"`
	UpdatedAt     string                               `protobuf:"bytes,9,opt,name=updatedAt" json:"updatedAt"`
	PolymarketURL string                               `protobuf:"bytes,10,opt,name=polymarketUrl" json:"polymarketUrl"`
	Active        bool                                 `protobuf:"varint,11,opt,name=active" json:"active"`
	Closed        bool                                 `protobuf:"varint,12,opt,name=closed" json:"closed"`
	Live          bool                                 `protobuf:"varint,13,opt,name=live" json:"live"`
	Ended         bool                                 `protobuf:"varint,14,opt,name=ended" json:"ended"`
	Teams         []*PolymarketSportsLiveTeamItem      `protobuf:"bytes,15,rep,name=teams" json:"teams"`
	Options       []*PolymarketFIFAMoneylineOptionItem `protobuf:"bytes,16,rep,name=options" json:"options"`
}

type PolymarketFIFAWalletBalanceItem struct {
	Chain         string `protobuf:"bytes,1,opt,name=chain" json:"chain"`
	Label         string `protobuf:"bytes,2,opt,name=label" json:"label"`
	WalletAddress string `protobuf:"bytes,3,opt,name=walletAddress" json:"walletAddress"`
	TokenAddress  string `protobuf:"bytes,4,opt,name=tokenAddress" json:"tokenAddress"`
	TokenSymbol   string `protobuf:"bytes,5,opt,name=tokenSymbol" json:"tokenSymbol"`
	Decimals      int32  `protobuf:"varint,6,opt,name=decimals" json:"decimals"`
	RawAmount     string `protobuf:"bytes,7,opt,name=rawAmount" json:"rawAmount"`
	Amount        string `protobuf:"bytes,8,opt,name=amount" json:"amount"`
	ExplorerURL   string `protobuf:"bytes,9,opt,name=explorerUrl" json:"explorerUrl"`
	OK            bool   `protobuf:"varint,10,opt,name=ok" json:"ok"`
	ErrorMessage  string `protobuf:"bytes,11,opt,name=errorMessage" json:"errorMessage"`
}

type PolymarketFIFAWalletHoldingItem struct {
	WalletID      int64  `protobuf:"varint,1,opt,name=walletId" json:"walletId"`
	Chain         string `protobuf:"bytes,2,opt,name=chain" json:"chain"`
	Type          string `protobuf:"bytes,3,opt,name=type" json:"type"`
	Alias         string `protobuf:"bytes,4,opt,name=alias" json:"alias"`
	WalletAddress string `protobuf:"bytes,5,opt,name=walletAddress" json:"walletAddress"`
	SolRawAmount  string `protobuf:"bytes,6,opt,name=solRawAmount" json:"solRawAmount"`
	SolAmount     string `protobuf:"bytes,7,opt,name=solAmount" json:"solAmount"`
	UsdcRawAmount string `protobuf:"bytes,8,opt,name=usdcRawAmount" json:"usdcRawAmount"`
	UsdcAmount    string `protobuf:"bytes,9,opt,name=usdcAmount" json:"usdcAmount"`
	ExplorerURL   string `protobuf:"bytes,10,opt,name=explorerUrl" json:"explorerUrl"`
	OK            bool   `protobuf:"varint,11,opt,name=ok" json:"ok"`
	ErrorMessage  string `protobuf:"bytes,12,opt,name=errorMessage" json:"errorMessage"`
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

type PolymarketSportsHistorySyncStatus struct {
	State         string `protobuf:"bytes,1,opt,name=state" json:"state"`
	StartedAt     int64  `protobuf:"varint,2,opt,name=startedAt" json:"startedAt"`
	CompletedAt   int64  `protobuf:"varint,3,opt,name=completedAt" json:"completedAt"`
	LastSuccessAt int64  `protobuf:"varint,4,opt,name=lastSuccessAt" json:"lastSuccessAt"`
	ErrorMessage  string `protobuf:"bytes,5,opt,name=errorMessage" json:"errorMessage"`
}
