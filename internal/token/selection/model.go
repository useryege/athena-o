package selection

import (
	"encoding/json"
	"time"

	"github.com/useryege/athena/internal/token/shared"
)

type SelectionOutcome string

const (
	SelectionOutcomeSelected SelectionOutcome = "selected"
	SelectionOutcomeRejected SelectionOutcome = "rejected"
	SelectionOutcomeDeferred SelectionOutcome = "deferred"
)

type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"
	TaskStatusRunning   TaskStatus = "running"
	TaskStatusSucceeded TaskStatus = "succeeded"
	TaskStatusFailed    TaskStatus = "failed"
)

type ProjectSelectionEvaluationTask struct {
	ID             int64
	ProjectID      int64
	ReportRevision int64
	Status         TaskStatus
	Attempts       int32
	AvailableAt    time.Time
	LeaseExpiresAt time.Time
	LastError      string
}

type ProjectSelection struct {
	ID              int64
	ProjectID       int64
	ChainID         int64
	Contract        shared.Address
	Outcome         SelectionOutcome
	StrategyKey     string
	StrategyVersion string
	ReportRevision  int64
	ReasonCodes     []string
	ReasonDetail    string
	DecidedAt       time.Time
	CreatedAt       time.Time
}

type ReportSnapshot struct {
	ProjectID          int64
	Revision           int64
	SchemaVersion      int32
	CompletenessStatus string
	Report             json.RawMessage
}

type StrategyInput struct {
	ProjectID      int64
	ReportRevision int64
	Report         ReportSnapshot
}

type SelectionDecision struct {
	Outcome      SelectionOutcome
	ReasonCodes  []string
	ReasonDetail string
}

type Page struct {
	Items    []ProjectSelection
	Total    int64
	Page     int32
	PageSize int32
}
