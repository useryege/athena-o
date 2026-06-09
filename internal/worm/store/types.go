package store

import (
	"encoding/json"
	"time"
)

type WormMarket struct {
	ConditionID      string
	Title            string
	Description      string
	Logo             string
	LastTradePrice   string
	State            string
	Category         string
	SortOption       string
	Created          int64
	EventTitle       string
	EventConditionID string
	EventLogo        string
	MarginEnabled    bool
	LiveState        string
	LiveCheckedAt    time.Time
	LivePriceChange  string
	GetMarketData    json.RawMessage
	Raw              json.RawMessage
	FetchedAt        time.Time
	LastSeenAt       time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type WormMarketPage struct {
	Items    []WormMarket
	Total    int64
	Page     int32
	PageSize int32
}

type WormMarketPriceSample struct {
	ConditionID string
	Price       string
	SampledAt   time.Time
}

type WormMarketLivePriceChange struct {
	ConditionID string
	SampleCount int64
	PriceChange string
	IsLive      bool
}
