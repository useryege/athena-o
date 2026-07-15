package selection

import (
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/useryege/athena/internal/token/reporting"
	"github.com/useryege/athena/internal/token/research"
)

type SelectionOutcome string

const (
	SelectionOutcomeSelected SelectionOutcome = "selected"
	SelectionOutcomeRejected SelectionOutcome = "rejected"
	SelectionOutcomeDeferred SelectionOutcome = "deferred"
)

type ProjectSelectionEvaluationTask struct {
	ID             int64
	ProjectID      int64
	ReportRevision int64
	Status         research.TaskStatus
	Attempts       int32
	AvailableAt    time.Time
	LeaseExpiresAt time.Time
	LastError      string
}

type ProjectSelection struct {
	ID              int64
	ProjectID       int64
	ChainID         int64
	Contract        common.Address
	Outcome         SelectionOutcome
	StrategyKey     string
	StrategyVersion string
	ReportRevision  int64
	ReasonCodes     []string
	ReasonDetail    string
	DecidedAt       time.Time
	CreatedAt       time.Time
}

type StrategyInput struct {
	ProjectID      int64
	ReportRevision int64
	Report         reporting.ResearchReportV1
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
