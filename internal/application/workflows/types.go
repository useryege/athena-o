package workflows

import (
	"fmt"
	"time"

	"github.com/useryege/athena/internal/application/model"
)

const (
	TaskQueueApplicationControl  = "application-control"
	TaskQueueApplicationExternal = "application-external"

	DefaultTemporalAddress   = "localhost:7233"
	DefaultTemporalNamespace = "default"

	ProjectCollectionReasonDexSwap = "dex_swap"
)

type ProjectRef = model.ProjectRef

type ProjectCollectionInput struct {
	Project ProjectRef `json:"project"`
	Reason  string     `json:"reason,omitempty"`
}

type ProjectCollectionLifecycleInput struct {
	Project    ProjectRef `json:"project"`
	WorkflowID string     `json:"workflow_id,omitempty"`
	NextRunAt  time.Time  `json:"next_run_at,omitempty"`
	LastError  string     `json:"last_error,omitempty"`
}

func ChainTaskQueue(chainID int64) string {
	return fmt.Sprintf("application-chain-%d", chainID)
}

func ProjectCollectionWorkflowID(chainID int64, contract string) string {
	return fmt.Sprintf("project-collection/%d/%s", chainID, contract)
}
