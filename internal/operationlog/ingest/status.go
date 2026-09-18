package ingest

import "time"

type Status struct {
	ProducerID             string    `json:"producerId"`
	SnapshotNo             uint64    `json:"snapshotNo,string"`
	ObservedAt             time.Time `json:"observedAt"`
	AttemptedEvents        uint64    `json:"attemptedEvents,string"`
	ConfirmedEvents        uint64    `json:"confirmedEvents,string"`
	UnconfirmedEvents      uint64    `json:"unconfirmedEvents,string"`
	InvalidEvents          uint64    `json:"invalidEvents,string"`
	CapacityRejectedEvents uint64    `json:"capacityRejectedEvents,string"`
	InFlightEvents         uint64    `json:"inFlightEvents,string"`
}
