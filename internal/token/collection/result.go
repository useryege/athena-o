package collection

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"

	"github.com/useryege/athena/internal/token/shared"
)

func NormalizeResult(schemaVersion int32, payload json.RawMessage) (json.RawMessage, shared.Hash, error) {
	var value any
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.UseNumber()
	if err := decoder.Decode(&value); err != nil {
		return nil, shared.Hash{}, err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return nil, shared.Hash{}, fmt.Errorf("collection result payload contains multiple JSON values")
		}
		return nil, shared.Hash{}, fmt.Errorf("decode trailing collection result payload: %w", err)
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
		return nil, shared.Hash{}, fmt.Errorf("marshal collection result hash input: %w", err)
	}
	digest := sha256.Sum256(hashInput)
	return canonical, shared.BytesToHash(digest[:]), nil
}
