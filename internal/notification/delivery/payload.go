package delivery

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"
)

// Payload is the exact immutable Telegram body and explicit rendering mode.
// Routing remains frozen in the work row and the send permit.
type Payload struct {
	Format          string `json:"format"`
	Text            string `json:"text"`
	MessageThreadID int    `json:"messageThreadId"`
}

func (p Payload) validate() error {
	if (p.Format != "plain" && p.Format != "html") || strings.TrimSpace(p.Text) == "" || p.MessageThreadID < 0 {
		return fmt.Errorf("invalid frozen notification payload")
	}
	return nil
}
func EncodePayload(p Payload) ([]byte, error) {
	if err := p.validate(); err != nil {
		return nil, err
	}
	return json.Marshal(p)
}
func DecodePayload(b []byte) (Payload, error) {
	var p Payload
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if err := d.Decode(&p); err != nil {
		return p, err
	}
	if d.Decode(new(any)) != io.EOF {
		return p, fmt.Errorf("trailing notification payload")
	}
	return p, p.validate()
}
func PayloadDigest(b []byte) []byte { sum := sha256.Sum256(b); return append([]byte(nil), sum[:]...) }

type AccountEnqueue struct {
	OwnerID, Source string
	ActivityID      int64
	BindingRevision uint64
	ChatID          int64
	Payload         []byte
	RecordedAt      time.Time
}

// SummaryPartEnqueue identifies a complete frozen summary part, never a single activity.
type SummaryPartEnqueue struct {
	OwnerID         string
	BatchID         int64
	Index           int
	BindingRevision uint64
	ChatID          int64
	Payload         []byte
	FrozenAt        time.Time
}
