package store

import (
	"encoding/json"
	"time"
)

type SportsHistoryMarketCard struct {
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

type SportsHistoryPricePoint struct {
	TokenID     string
	MarketKey   string
	EventKey    string
	ConditionID string
	Outcome     string
	PriceTs     time.Time
	Price       float64
	FetchedAt   time.Time
}

type SportsHistoryPriceHistorySeries struct {
	MarketKey  string
	TokenID    string
	Outcome    string
	Timestamps []int64
	Prices     []float64
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
	Markets        []SportsHistoryMarketCard
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
