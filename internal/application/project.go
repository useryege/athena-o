package application

import (
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/google/uuid"
	"github.com/useryege/athena/pkg/apis/application/v1alpha1"
)

type Project struct {
	Meta      ProjectMeta
	PerfTrace PerfTrace

	InitState    ProjectInitState
	DelayedState ProjectDelayedState
	DynamicState ProjectDynamicState
}

type ProjectMeta struct {
	ProjectID   uuid.UUID
	BlockTime   uint64
	BlockNumber uint64
	Contract    common.Address
	Creator     common.Address
	Tx          *types.Transaction
}

type PerfTrace struct {
	BlockDiscoveredAt time.Time
	TxDiscoveredAt    time.Time
	FilterCompletedAt time.Time
}

type ProjectInitState struct {
	Name        OnceValue[string]
	Symbol      OnceValue[string]
	Decimals    OnceValue[uint8]
	TotalSupply OnceValue[*big.Int]
}

func (s *ProjectInitState) IsReady() bool {
	return s.Name.IsReady() && s.Symbol.IsReady() && s.Decimals.IsReady() && s.TotalSupply.IsReady()
}

type ProjectDelayedState struct {
	SourceCode    RetryUntilReadyValue[string]
	SourceCodeABI RetryUntilReadyValue[string]
}

type ProjectDynamicState struct {
	HolderCount DynamicValue[uint64]
}

func projectToView(project *Project) *v1alpha1.ProjectView {
	if project == nil {
		return nil
	}

	txHash := ""
	if project.Meta.Tx != nil {
		txHash = project.Meta.Tx.Hash().Hex()
	}

	totalSupply := ""
	if supply := project.InitState.TotalSupply.Get(); supply != nil {
		totalSupply = supply.String()
	}

	return &v1alpha1.ProjectView{
		Meta: v1alpha1.ProjectMeta{
			ProjectID:   project.Meta.ProjectID.String(),
			BlockTime:   project.Meta.BlockTime,
			BlockNumber: project.Meta.BlockNumber,
			Contract:    project.Meta.Contract.String(),
			Creator:     project.Meta.Creator.String(),
			TxHash:      txHash,
		},
		InitState: v1alpha1.ProjectInitState{
			Name:        project.InitState.Name.Get(),
			Symbol:      project.InitState.Symbol.Get(),
			Decimals:    uint32(project.InitState.Decimals.Get()),
			TotalSupply: totalSupply,
		},
	}
}
