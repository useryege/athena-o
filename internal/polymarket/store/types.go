package store

import (
	"encoding/json"
	"time"
)

type SportsLiveEvent struct {
	EventKey          string
	EventID           string
	Ticker            string
	Slug              string
	Title             string
	Description       string
	ResolutionSource  string
	StartDate         time.Time
	CreationDate      time.Time
	EndDate           time.Time
	StartTime         time.Time
	CreatedAtGamma    time.Time
	UpdatedAtGamma    time.Time
	Image             string
	Icon              string
	Active            bool
	Closed            bool
	Archived          bool
	Featured          bool
	Restricted        bool
	Live              bool
	Ended             bool
	Liquidity         float64
	Volume            float64
	OpenInterest      float64
	Category          string
	Score             string
	Period            string
	Elapsed           string
	FinishedTimestamp string
	GameID            int64
	EventDate         string
	GameStatus        string
	CommentCount      int64
	Sport             json.RawMessage
	Teams             json.RawMessage
	Tags              json.RawMessage
	Raw               json.RawMessage
	FetchedAt         time.Time
	LastSeenAt        time.Time
}

type SportsLiveMarket struct {
	MarketKey        string
	EventKey         string
	EventID          string
	EventSlug        string
	MarketID         string
	ConditionID      string
	Slug             string
	Question         string
	Title            string
	Description      string
	ResolutionSource string
	SportsMarketType string
	GroupItemTitle   string
	Image            string
	Icon             string
	Outcomes         string
	OutcomePrices    string
	ClobTokenIDs     string
	Active           bool
	Closed           bool
	Archived         bool
	Restricted       bool
	EnableOrderBook  bool
	Volume           string
	VolumeNum        float64
	LiquidityNum     float64
	Volume24hr       float64
	Volume1wk        float64
	Volume1mo        float64
	Volume1yr        float64
	Spread           float64
	BestBid          float64
	BestAsk          float64
	LastTradePrice   float64
	StartDate        time.Time
	EndDate          time.Time
	CreatedAtGamma   time.Time
	UpdatedAtGamma   time.Time
	Tags             json.RawMessage
	Raw              json.RawMessage
	FetchedAt        time.Time
	LastSeenAt       time.Time
}

type SportsLiveEventCard struct {
	EventKey       string
	EventID        string
	Slug           string
	Title          string
	Image          string
	Score          string
	Period         string
	Elapsed        string
	GameStatus     string
	StartTime      time.Time
	UpdatedAtGamma time.Time
	Liquidity      float64
	Volume         float64
	MarketCount    int64
	Teams          json.RawMessage
	FetchedAt      time.Time
	LastSeenAt     time.Time
	Markets        []SportsLiveMarketCard
}

type SportsLiveMarketCard struct {
	EventKey         string
	MarketKey        string
	ConditionID      string
	Slug             string
	SportsMarketType string
	Question         string
	Outcomes         string
	OutcomePrices    string
	BestBid          float64
	BestAsk          float64
	LastTradePrice   float64
	Spread           float64
	LiquidityNum     float64
	VolumeNum        float64
	UpdatedAtGamma   time.Time
}

type SportsLivePriceHistoryMarket struct {
	EventKey     string
	MarketKey    string
	ConditionID  string
	Outcomes     string
	ClobTokenIDs string
}

type SportsLivePricePoint struct {
	TokenID     string
	MarketKey   string
	EventKey    string
	ConditionID string
	Outcome     string
	PriceTs     time.Time
	Price       float64
	FetchedAt   time.Time
}

type SportsLivePriceAlertToken struct {
	TokenID       string
	MarketKey     string
	EventKey      string
	ConditionID   string
	Outcome       string
	PriceTs       time.Time
	Price         float64
	EventSlug     string
	EventTitle    string
	MarketTitle   string
	Score         string
	Period        string
	Elapsed       string
	GameStatus    string
	Volume        float64
	Liquidity     float64
	LastAlertBand string
	LastAlertedAt time.Time
}

type SportsLivePriceAlertState struct {
	TokenID       string
	MarketKey     string
	EventKey      string
	ConditionID   string
	Outcome       string
	AlertBand     string
	LastAlertedAt time.Time
	LastPriceTs   time.Time
	LastPrice     float64
}

type SportsLivePriceHistorySeries struct {
	MarketKey  string
	TokenID    string
	Outcome    string
	Timestamps []int64
	Prices     []float64
}

type ChainLogCursor struct {
	SyncName        string
	ContractAddress string
	Topic           string
	LastBlockNumber uint64
	LastPolledAt    time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type ManagedOOProposePriceLog struct {
	TxHash              string
	LogIndex            uint
	BlockNumber         uint64
	BlockHash           string
	TxIndex             uint
	ContractAddress     string
	Topic               string
	Requester           string
	Proposer            string
	Identifier          string
	RequestTimestamp    uint64
	AncillaryDataHex    string
	AncillaryDataText   string
	MarketID            string
	ProposedPrice       string
	ExpirationTimestamp uint64
	Currency            string
	RawTopics           json.RawMessage
	RawData             string
	FetchedAt           time.Time
}

type ManagedOOMarket struct {
	MarketID         string
	ConditionID      string
	Slug             string
	Question         string
	Description      string
	ResolutionSource string
	QuestionID       string
	SportsMarketType string
	GroupItemTitle   string
	Image            string
	Icon             string
	Outcomes         string
	OutcomePrices    string
	ClobTokenIDs     string
	Active           bool
	Closed           bool
	Archived         bool
	Restricted       bool
	EnableOrderBook  bool
	AcceptingOrders  bool
	Volume           string
	VolumeNum        float64
	LiquidityNum     float64
	Volume24hr       float64
	Volume1wk        float64
	Volume1mo        float64
	Volume1yr        float64
	Spread           float64
	BestBid          float64
	BestAsk          float64
	LastTradePrice   float64
	StartDate        time.Time
	EndDate          time.Time
	CreatedAtGamma   time.Time
	UpdatedAtGamma   time.Time
	Tags             json.RawMessage
	Raw              json.RawMessage
	FetchedAt        time.Time
	Labels           []ManagedOOMarketLabel
}

type ManagedOOMarketLabel struct {
	MarketID  string
	Label     string
	TagID     string
	Slug      string
	Position  int64
	FetchedAt time.Time
}

type SportsHistoryEvent struct {
	EventKey       string
	EventID        string
	League         string
	Slug           string
	Title          string
	Image          string
	Icon           string
	Score          string
	Period         string
	Elapsed        string
	GameStatus     string
	StartTime      time.Time
	FinishedAt     time.Time
	UpdatedAtGamma time.Time
	Liquidity      float64
	Volume         float64
	Teams          json.RawMessage
	Raw            json.RawMessage
	FetchedAt      time.Time
	LastSeenAt     time.Time
}

type SportsHistoryMarket struct {
	MarketKey        string
	EventKey         string
	ConditionID      string
	Slug             string
	Question         string
	SportsMarketType string
	Outcomes         string
	OutcomePrices    string
	ClobTokenIDs     string
	BestBid          float64
	BestAsk          float64
	LastTradePrice   float64
	Spread           float64
	LiquidityNum     float64
	VolumeNum        float64
	UpdatedAtGamma   time.Time
	Raw              json.RawMessage
	FetchedAt        time.Time
	LastSeenAt       time.Time
}

type SportsHistoryEventCard struct {
	EventKey       string
	EventID        string
	League         string
	Slug           string
	Title          string
	Image          string
	Score          string
	Period         string
	Elapsed        string
	GameStatus     string
	StartTime      time.Time
	FinishedAt     time.Time
	UpdatedAtGamma time.Time
	Liquidity      float64
	Volume         float64
	MarketCount    int64
	Teams          json.RawMessage
	FetchedAt      time.Time
	LastSeenAt     time.Time
	Markets        []SportsLiveMarketCard
}

type SportsHistoryPriceHistoryMarket struct {
	EventKey     string
	MarketKey    string
	ConditionID  string
	Outcomes     string
	ClobTokenIDs string
	StartTime    time.Time
	FinishedAt   time.Time
}
