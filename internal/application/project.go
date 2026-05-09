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

	Token      TokenState
	WethV2Pool PairV2State
}

type ProjectMeta struct {
	ProjectID   uuid.UUID
	BlockTime   uint64
	BlockNumber uint64
	Contract    common.Address
	Creator     common.Address
	Tx          *types.Transaction
	TxIndex     uint64
}

type PerfTrace struct {
	BlockDiscoveredAt time.Time
	TxDiscoveredAt    time.Time
	FilterCompletedAt time.Time
}

type TokenState struct {
	Name        FieldValue[string]
	Symbol      FieldValue[string]
	Decimals    FieldValue[uint8]
	TotalSupply FieldValue[*big.Int]

	SourceCode    FieldValue[string]
	SourceCodeABI FieldValue[string]
}

type PairV2State struct {
	IsContractCreated FieldValue[bool]
	Contract          FieldValue[common.Address]
	Token0            FieldValue[common.Address]
	Token1            FieldValue[common.Address]

	TotalSupply FieldValue[*big.Int]
	// reserves
	Reserve0           FieldValue[*big.Int]
	Reserve1           FieldValue[*big.Int]
	BlockTimestampLast FieldValue[uint32]
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
	if supply, ok := project.Token.TotalSupply.Get(); ok && supply != nil {
		totalSupply = supply.String()
	}

	name, _ := project.Token.Name.Get()
	symbol, _ := project.Token.Symbol.Get()
	decimals, _ := project.Token.Decimals.Get()
	sourceCode, _ := project.Token.SourceCode.Get()
	sourceCodeABI, _ := project.Token.SourceCodeABI.Get()

	isWethV2PoolContractCreated, _ := project.WethV2Pool.IsContractCreated.Get()
	wethV2PoolBlockTimestampLast, _ := project.WethV2Pool.BlockTimestampLast.Get()

	return &v1alpha1.ProjectView{
		Meta: v1alpha1.ProjectMeta{
			ProjectID:   project.Meta.ProjectID.String(),
			BlockTime:   project.Meta.BlockTime,
			BlockNumber: project.Meta.BlockNumber,
			Contract:    project.Meta.Contract.String(),
			Creator:     project.Meta.Creator.String(),
			TxHash:      txHash,
			TxIndex:     project.Meta.TxIndex,
		},
		Token: v1alpha1.TokenState{
			Name:          name,
			Symbol:        symbol,
			Decimals:      uint32(decimals),
			TotalSupply:   totalSupply,
			SourceCode:    sourceCode,
			SourceCodeABI: sourceCodeABI,
		},
		WethV2Pool: v1alpha1.PairV2State{
			IsContractCreated:  isWethV2PoolContractCreated,
			Contract:           addressFieldToString(&project.WethV2Pool.Contract),
			Token0:             addressFieldToString(&project.WethV2Pool.Token0),
			Token1:             addressFieldToString(&project.WethV2Pool.Token1),
			TotalSupply:        bigIntFieldToString(&project.WethV2Pool.TotalSupply),
			Reserve0:           bigIntFieldToString(&project.WethV2Pool.Reserve0),
			Reserve1:           bigIntFieldToString(&project.WethV2Pool.Reserve1),
			BlockTimestampLast: wethV2PoolBlockTimestampLast,
		},
	}
}

func bigIntFieldToString(field *FieldValue[*big.Int]) string {
	value, ok := field.Get()
	if !ok || value == nil {
		return ""
	}
	return value.String()
}

func addressFieldToString(field *FieldValue[common.Address]) string {
	value, ok := field.Get()
	if !ok {
		return ""
	}
	return value.String()
}
