package store

import "time"

type SportsLiveMarket struct {
	ConditionID    string
	MarketSlug     string
	EventSlug      string
	Title          string
	Image          string
	Score          string
	Period         string
	Elapsed        string
	GammaUpdatedAt time.Time
	LiquidityNum   float64
	VolumeNum      float64
	FetchedAt      time.Time
	LastSeenAt     time.Time
}
