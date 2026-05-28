package persistence

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	appstore "github.com/useryege/athena/internal/application/store"
)

type ProjectEventType int16

const (
	projectEventTypeCreated          ProjectEventType = 1
	projectEventTypeOpenSource       ProjectEventType = 2
	projectEventTypePolicyMatchAudit ProjectEventType = 5
)

const (
	projectEventIdempotencyCreated    = "project_created"
	projectEventIdempotencyOpenSource = "project_source_code_opened"
)

func projectEventIdempotencyPolicyMatch(rule string) string {
	return "project_policy_matched:" + rule
}

type projectPolicyMatchedEventPayload struct {
	Rule     string         `json:"rule"`
	Evidence map[string]any `json:"evidence"`
	Source   string         `json:"source"`
}

func NewProjectPolicyMatchedEvent(contract common.Address, ruleName string, evidence map[string]any, occurredAt time.Time) (appstore.ProjectEventLog, error) {
	payload, err := json.Marshal(projectPolicyMatchedEventPayload{
		Rule:     ruleName,
		Evidence: evidence,
		Source:   "policy_engine",
	})
	if err != nil {
		return appstore.ProjectEventLog{}, fmt.Errorf("marshal project policy matched event payload: %w", err)
	}
	return appstore.ProjectEventLog{
		Contract:       contract,
		EventType:      int16(projectEventTypePolicyMatchAudit),
		OccurredAt:     occurredAt,
		Message:        fmt.Sprintf("Policy rule %s matched project", ruleName),
		Payload:        string(payload),
		IdempotencyKey: projectEventIdempotencyPolicyMatch(ruleName),
	}, nil
}
