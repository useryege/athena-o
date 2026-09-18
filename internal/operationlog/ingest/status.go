package ingest

import "time"

type Status struct {
	StartedAt              time.Time  `json:"startedAt"`
	StoppedAt              *time.Time `json:"stoppedAt"`
	LastFailureAt          *time.Time `json:"lastFailureAt"`
	LastFailureCode        *string    `json:"lastFailureCode"`
	LastRecoveredAt        *time.Time `json:"lastRecoveredAt"`
	LastConfirmedAt        *time.Time `json:"lastConfirmedAt"`
	PersistenceReachable   *bool      `json:"persistenceReachable"`
	ProducerID             string     `json:"producerId"`
	SnapshotNo             uint64     `json:"snapshotNo,string"`
	ObservedAt             time.Time  `json:"observedAt"`
	AttemptedEvents        uint64     `json:"attemptedEvents,string"`
	ConfirmedEvents        uint64     `json:"confirmedEvents,string"`
	UnconfirmedEvents      uint64     `json:"unconfirmedEvents,string"`
	InvalidEvents          uint64     `json:"invalidEvents,string"`
	CapacityRejectedEvents uint64     `json:"capacityRejectedEvents,string"`
	InFlightEvents         uint64     `json:"inFlightEvents,string"`
}
