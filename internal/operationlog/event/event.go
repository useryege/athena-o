// Package event defines the versioned, allowlisted operation log envelope.
package event

import (
	"encoding/json"
	"time"
	"unicode/utf8"
)

type Phase string
type Outcome string
type Observation string

const (
	Start          Phase       = "START"
	Finish         Phase       = "FINISH"
	Unknown        Outcome     = "UNKNOWN"
	Succeeded      Outcome     = "SUCCEEDED"
	Accepted       Outcome     = "ACCEPTED"
	Failed         Outcome     = "FAILED"
	Denied         Outcome     = "DENIED"
	Partial        Outcome     = "PARTIAL"
	ActionRequired Outcome     = "ACTION_REQUIRED"
	Cancelled      Outcome     = "CANCELLED"
	StartOnly      Observation = "START_ONLY"
	FinishOnly     Observation = "FINISH_ONLY"
	Complete       Observation = "COMPLETE"
)

type Actor struct {
	AccountID                *string `json:"accountId"`
	UsernameSnapshot         *string `json:"usernameSnapshot"`
	Role                     string  `json:"role"`
	Realm                    string  `json:"realm"`
	CredentialKind           string  `json:"credentialKind"`
	IdentityVerified         bool    `json:"identityVerified"`
	IdentitySnapshotComplete bool    `json:"identitySnapshotComplete"`
	Provider                 *string `json:"provider"`
}
type Resource struct {
	Type              string `json:"type"`
	ID                string `json:"id"`
	ReferenceVerified bool   `json:"referenceVerified"`
	Primary           bool   `json:"primary"`
}
type ModuleAccess struct {
	Module string `json:"module"`
	Access string `json:"access"`
}
type Access struct {
	LoginEnabled         bool           `json:"loginEnabled"`
	APIKeyEnabled        bool           `json:"apiKeyEnabled"`
	ProfitSharingEnabled bool           `json:"profitSharingEnabled"`
	Revision             string         `json:"revision"`
	ModuleAccess         []ModuleAccess `json:"moduleAccess"`
}

// Value has a closed set of constructors. Validate additionally checks the type
// and allowed field name for the action; presence is represented by map membership.
type Value struct{ raw json.RawMessage }

func Bool(v bool) Value { b, _ := json.Marshal(v); return Value{b} }
func String(v string) Value {
	if !utf8.ValidString(v) {
		return Value{raw: json.RawMessage{255}}
	}
	b, _ := json.Marshal(v)
	return Value{b}
}
func Strings(v []string) Value {
	for _, s := range v {
		if !utf8.ValidString(s) {
			return Value{raw: json.RawMessage{255}}
		}
	}
	b, _ := json.Marshal(v)
	return Value{b}
}
func AccessValue(v Access) Value              { b, _ := json.Marshal(v); return Value{b} }
func (v Value) MarshalJSON() ([]byte, error)  { return v.raw.MarshalJSON() }
func (v *Value) UnmarshalJSON(b []byte) error { v.raw = append(json.RawMessage(nil), b...); return nil }

type Details map[string]Value

type Event struct {
	SchemaVersion       int         `json:"schemaVersion"`
	EventID             string      `json:"eventId"`
	OperationID         string      `json:"operationId"`
	RequestID           string      `json:"requestId"`
	ParentOperationID   *string     `json:"parentOperationId"`
	Phase               Phase       `json:"phase"`
	ProducerID          string      `json:"producerId"`
	StartedAt           time.Time   `json:"startedAt"`
	OccurredAt          time.Time   `json:"occurredAt"`
	Actor               Actor       `json:"actor"`
	ActionCode          string      `json:"actionCode"`
	ModuleCode          string      `json:"moduleCode"`
	Resources           []Resource  `json:"resources"`
	TargetAccountID     *string     `json:"targetAccountId"`
	BusinessRequestID   *string     `json:"businessRequestId"`
	Outcome             Outcome     `json:"outcome"`
	Observation         Observation `json:"observation"`
	BusinessState       *string     `json:"businessState"`
	Effect              []string    `json:"effect"`
	DurationMs          *int64      `json:"durationMs,string"`
	GRPCCode            *string     `json:"grpcCode"`
	HTTPStatus          *int        `json:"httpStatus"`
	ReasonCode          *string     `json:"reasonCode"`
	ResponseWriteFailed bool        `json:"responseWriteFailed"`
	Details             Details     `json:"details"`
	ResourceCount       *uint64     `json:"resourceCount,string"`
	ResourcesComplete   bool        `json:"resourcesComplete"`
}
