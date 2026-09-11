package types

import "time"

// ArrivalEvidence identifies an original owner candidate. Missing monotonic
// elapsed is unknown, never reconstructed from UTC or projection ordering.
type ArrivalEvidence struct {
	SourceID         int64     `json:"sourceId"`
	SubscriptionID   string    `json:"subscriptionId"`
	AttemptID        string    `json:"attemptId"`
	Generation       uint64    `json:"generation"`
	Epoch            uint64    `json:"epoch"`
	Sequence         uint64    `json:"sequence"`
	ReceivedAt       time.Time `json:"receivedAt"`
	ElapsedNS        *int64    `json:"elapsedNs,omitempty"`
	Confirmation     string    `json:"confirmation"`
	Removed          bool      `json:"removed"`
	Formed           bool      `json:"formed"`
	Mode             string    `json:"mode"`
	ChatID           *int64    `json:"chatId,omitempty"`
	BindingRevision  *int64    `json:"bindingRevision,omitempty"`
	DeliveryStatus   string    `json:"deliveryStatus"`
	DeliveryAttempts int32     `json:"deliveryAttempts"`
	DeliveryEligible bool      `json:"deliveryEligible"`
	EverStarted      bool      `json:"everStarted"`
}
type OwnerQueueEvidence struct {
	OrdinaryPending int64  `json:"ordinaryPending"`
	OrdinarySending int64  `json:"ordinarySending"`
	SummaryWaiting  int64  `json:"summaryWaiting"`
	PartPending     int64  `json:"partPending"`
	PartSending     int64  `json:"partSending"`
	ReplyPending    int64  `json:"replyPending"`
	ReplySending    int64  `json:"replySending"`
	Policy          string `json:"policy"`
}
type FormationEvidence struct {
	Processing        *ProjectionTiming  `json:"processing,omitempty"`
	Gate              []GatePhase        `json:"gate"`
	Rule              string             `json:"rule"`
	Cohort            string             `json:"cohort"`
	Reason            string             `json:"reason"`
	ObservedAt        time.Time          `json:"observedAt"`
	Current           ArrivalEvidence    `json:"current"`
	Previous          *ArrivalEvidence   `json:"previous,omitempty"`
	ArrivalIntervalNS *int64             `json:"arrivalIntervalNs,omitempty"`
	Queue             OwnerQueueEvidence `json:"queue"`
}

// ProjectionTiming describes this actual processing round, never first confirmation
// or time since receipt. Durations retain the originating process monotonic clock.
type ProjectionTiming struct {
	ConfirmationRoundNS     int64     `json:"confirmationRoundNs"`
	VersionRoundNS          int64     `json:"versionRoundNs"`
	ConfirmationCompletedAt time.Time `json:"confirmationCompletedAt"`
	MetadataExtraWaitNS     int64     `json:"metadataExtraWaitNs"`
}
type GatePhase struct {
	Kind      string `json:"kind"`
	Phase     string `json:"phase"`
	ElapsedNS int64  `json:"elapsedNs"`
	Succeeded bool   `json:"succeeded"`
}

// ObservationClock belongs to one Projector metrics instance. ElapsedNS is never
// compared across IDs; Valid is false after any lost observation or cutoff failure.
type ObservationClock struct {
	ID        string
	ElapsedNS int64
	Cutoff    int64
	Valid     bool
}
type FinalityTiming struct {
	State            string     `json:"state"`
	Reason           string     `json:"reason,omitempty"`
	ClockID          string     `json:"clockId,omitempty"`
	FirstStartedAt   *time.Time `json:"firstStartedAt,omitempty"`
	FirstStartedNS   *int64     `json:"firstStartedNs,omitempty"`
	FirstRoundNS     *int64     `json:"firstRoundNs,omitempty"`
	FirstConfirmedAt *time.Time `json:"firstConfirmedAt,omitempty"`
	FirstConfirmedNS *int64     `json:"firstConfirmedNs,omitempty"`
}

// FinalityRoundObservation is sampled at the actual call boundary, not CheckedAt.
type FinalityRoundObservation struct {
	ClockID                                string
	StartedAt, ReturnedAt                  time.Time
	StartedNS, ReturnedNS                  int64
	Confirmed, Reliable, SourceAfterCutoff bool
}
