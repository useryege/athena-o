package events

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"
)

const (
	SchemaVersionV1 = 1

	// TOPIC
	TopicContractCreatedV1    = "athena.evm.contract_created.v1"
	TopicDexSwapV1            = "athena.evm.dex_swap.v1"
	TopicApplicationProjectV1 = "athena.application.project_event.v1"

	// EVENT
	EventTypeContractCreated = "evm.contract_created"
	EventTypeDexSwap         = "evm.dex_swap"
	EventTypeProjectEvent    = "application.project_event"
)

type Envelope struct {
	EventID       string          `json:"event_id"`
	EventType     string          `json:"event_type"`
	SchemaVersion int             `json:"schema_version"`
	ChainID       int64           `json:"chain_id"`
	OccurredAt    time.Time       `json:"occurred_at"`
	Payload       json.RawMessage `json:"payload"`
}

func NewEnvelope(eventType string, chainID int64, payload any) (Envelope, error) {
	if eventType == "" {
		return Envelope{}, fmt.Errorf("application event type is required")
	}
	if chainID <= 0 {
		return Envelope{}, fmt.Errorf("application event chain_id must be positive")
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return Envelope{}, fmt.Errorf("marshal application event payload: %w", err)
	}
	return Envelope{
		EventID:       newEventID(),
		EventType:     eventType,
		SchemaVersion: SchemaVersionV1,
		ChainID:       chainID,
		OccurredAt:    time.Now().UTC(),
		Payload:       raw,
	}, nil
}

func (e Envelope) MarshalJSONBytes() ([]byte, error) {
	if e.EventID == "" {
		e.EventID = newEventID()
	}
	if e.SchemaVersion == 0 {
		e.SchemaVersion = SchemaVersionV1
	}
	if e.OccurredAt.IsZero() {
		e.OccurredAt = time.Now().UTC()
	}
	return json.Marshal(e)
}

func newEventID() string {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(buf[:])
}
