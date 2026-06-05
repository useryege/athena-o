package workflows

import (
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
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

type CandidateQualificationInput struct {
	Project   ProjectRef     `json:"project"`
	Source    string         `json:"source,omitempty"`
	Candidate CandidateFacts `json:"candidate,omitempty"`
}

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

type CandidateFacts struct {
	ChainID     int64                        `json:"chain_id,omitempty"`
	BlockTime   uint64                       `json:"block_time,omitempty"`
	BlockNumber uint64                       `json:"block_number,omitempty"`
	TxIndex     uint64                       `json:"tx_index,omitempty"`
	Contract    common.Address               `json:"contract,omitempty"`
	Creator     common.Address               `json:"creator,omitempty"`
	TxHash      common.Hash                  `json:"tx_hash,omitempty"`
	Source      model.ProjectDiscoverySource `json:"source,omitempty"`
}

func (c CandidateFacts) DiscoveredProjectCandidate(fallback ProjectRef, source string) model.DiscoveredProjectCandidate {
	if c.ChainID <= 0 {
		c.ChainID = fallback.ChainID
	}
	if c.Contract == (common.Address{}) {
		c.Contract = fallback.Contract
	}
	if c.Source == "" {
		c.Source = model.ProjectDiscoverySource(source)
	}
	return model.DiscoveredProjectCandidate{
		ChainID:     c.ChainID,
		BlockTime:   c.BlockTime,
		BlockNumber: c.BlockNumber,
		TxIndex:     c.TxIndex,
		Contract:    c.Contract,
		Creator:     c.Creator,
		TxHash:      c.TxHash,
		Source:      c.Source,
	}
}

func ChainTaskQueue(chainID int64) string {
	return fmt.Sprintf("application-chain-%d", chainID)
}

func CandidateQualificationWorkflowID(chainID int64, contract string) string {
	return fmt.Sprintf("candidate/%d/%s", chainID, contract)
}

func ProjectCollectionWorkflowID(chainID int64, contract string) string {
	return fmt.Sprintf("project-collection/%d/%s", chainID, contract)
}
