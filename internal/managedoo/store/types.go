package store

import (
	"encoding/json"
	"time"
)

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

type ManagedOODisputePriceLog struct {
	TxHash            string
	LogIndex          uint
	BlockNumber       uint64
	BlockHash         string
	TxIndex           uint
	ContractAddress   string
	Topic             string
	Requester         string
	Proposer          string
	Disputer          string
	Identifier        string
	RequestTimestamp  uint64
	AncillaryDataHex  string
	AncillaryDataText string
	MarketID          string
	ProposedPrice     string
	RawTopics         json.RawMessage
	RawData           string
	FetchedAt         time.Time
}

type ListManagedOOLogsOptions struct {
	Page        int
	PageSize    int
	BlockNumber uint64
}

type ManagedOOMarket struct {
	MarketID         string
	ConditionID      string
	Slug             string
	EventSlug        string
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

type ManagedOOProposePriceAlertCandidate struct {
	TxHash              string
	LogIndex            int64
	BlockNumber         int64
	MarketID            string
	Proposer            string
	ProposedPrice       string
	RequestTimestamp    int64
	ExpirationTimestamp int64
	AncillaryDataText   string
	ConditionID         string
	EventSlug           string
	MarketSlug          string
	Question            string
	MatchedLabels       string
}

type ManagedOODisputePriceAlertCandidate struct {
	TxHash            string
	LogIndex          int64
	BlockNumber       int64
	MarketID          string
	Requester         string
	Proposer          string
	Disputer          string
	ProposedPrice     string
	RequestTimestamp  int64
	AncillaryDataText string
	ConditionID       string
	EventSlug         string
	MarketSlug        string
	Question          string
	MatchedLabels     string
}
