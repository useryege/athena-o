package research

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"

	"github.com/useryege/athena/internal/token/shared"
)

func NormalizeObservation(schemaVersion int32, payload json.RawMessage) (json.RawMessage, shared.Hash, error) {
	var value any
	if err := json.Unmarshal(payload, &value); err != nil {
		return nil, shared.Hash{}, err
	}
	canonical, err := json.Marshal(value)
	if err != nil {
		return nil, shared.Hash{}, err
	}
	hashInput, err := json.Marshal(struct {
		SchemaVersion int32           `json:"schemaVersion"`
		Payload       json.RawMessage `json:"payload"`
	}{SchemaVersion: schemaVersion, Payload: canonical})
	if err != nil {
		return nil, shared.Hash{}, fmt.Errorf("marshal observation hash input: %w", err)
	}
	digest := sha256.Sum256(hashInput)
	return canonical, shared.BytesToHash(digest[:]), nil
}
