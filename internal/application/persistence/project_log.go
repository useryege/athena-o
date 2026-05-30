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
	projectEventTypeReportMatchAudit ProjectEventType = 5
)

const (
	projectEventIdempotencyCreated    = "project_created"
	projectEventIdempotencyOpenSource = "project_source_code_opened"
)

func projectEventIdempotencyReportMatch(rule string) string {
	return "project_report_matched:" + rule
}

type projectReportMatchedEventPayload struct {
	Rule     string         `json:"rule"`
	Evidence map[string]any `json:"evidence"`
	Source   string         `json:"source"`
}

func NewProjectReportMatchedEvent(contract common.Address, ruleName string, evidence map[string]any, occurredAt time.Time) (appstore.ProjectEventLog, error) {
	payload, err := json.Marshal(projectReportMatchedEventPayload{
		Rule:     ruleName,
		Evidence: evidence,
		Source:   "report_component",
	})
	if err != nil {
		return appstore.ProjectEventLog{}, fmt.Errorf("marshal project report matched event payload: %w", err)
	}
	return appstore.ProjectEventLog{
		Contract:       contract,
		EventType:      int16(projectEventTypeReportMatchAudit),
		OccurredAt:     occurredAt,
		Message:        fmt.Sprintf("Report rule %s matched project", ruleName),
		Payload:        string(payload),
		IdempotencyKey: projectEventIdempotencyReportMatch(ruleName),
	}, nil
}
