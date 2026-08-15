package store

import "time"

type FIFAEventConfig struct {
	WormEventID string
	EventRef    string
	UpdatedAt   time.Time
}
