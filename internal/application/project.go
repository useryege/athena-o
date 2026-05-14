package application

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/google/uuid"
	"github.com/useryege/athena/internal/application/sourcecode"
	athenacontract "github.com/useryege/athena/pkg/abi/ATHENA"
	"github.com/useryege/athena/pkg/apis/application/v1alpha1"
)

type Project struct {
	Meta       ProjectMeta
	ChainState athenacontract.AthenaProject
	SourceCode ProjectSourceCodeState
	Simulate   ProjectSimulateState
	Analysis   ProjectAnalysisState
}

type ProjectMeta struct {
	ProjectID   uuid.UUID
	BlockTime   uint64
	BlockNumber uint64
	Contract    common.Address
	Creator     common.Address
	Tx          *types.Transaction
	TxHash      common.Hash
	TxIndex     uint64
}

type ProjectSourceCodeState struct {
	SourceCode    FieldValue[string]
	SourceCodeABI FieldValue[string]
}

type ProjectAnalysisState struct {
	SourceCodeBlacklist FieldValue[sourcecode.BlacklistReport]
}

func (p *Project) IsPairBalanceOverSupply() bool {
	if p == nil || p.ChainState.Pair.TokenReserveBalance == nil || p.ChainState.Token.TotalSupply == nil {
		return false
	}
	return p.ChainState.Pair.TokenReserveBalance.Cmp(p.ChainState.Token.TotalSupply) > 0
}

func (p *Project) IsLowValuePool() bool {
	if p == nil || p.ChainState.Pair.WethReserveBalance == nil {
		return false
	}
	return p.ChainState.Pair.WethReserveBalance.Cmp(big.NewInt(MinWethValue)) < 0
}

func (p *Project) HasLiquidity() bool {
	if p == nil || p.ChainState.Pair.Reserve0 == nil || p.ChainState.Pair.Reserve1 == nil {
		return false
	}
	return p.ChainState.Pair.Reserve0.Sign() != 0 && p.ChainState.Pair.Reserve1.Sign() != 0
}

func (p *Project) HasOnlyMinimumLiquidity() bool {
	if p == nil || p.ChainState.Pair.TotalSupply == nil {
		return false
	}
	return p.ChainState.Pair.TotalSupply.Cmp(big.NewInt(1000)) == 0
}

type ProjectSimulateState struct {
	CreatorResult FieldValue[SimulateResult]
}

func projectToView(project *Project) *v1alpha1.ProjectView {
	if project == nil {
		return nil
	}

	txHash := ""
	if project.Meta.Tx != nil {
		txHash = project.Meta.Tx.Hash().Hex()
	} else if project.Meta.TxHash != (common.Hash{}) {
		txHash = project.Meta.TxHash.Hex()
	}

	chainState := project.ChainState
	sourceCode, _ := project.SourceCode.SourceCode.Get()
	sourceCodeABI, _ := project.SourceCode.SourceCodeABI.Get()
	creatorResult, _ := project.Simulate.CreatorResult.Get()
	sourceCodeBlacklist, _ := project.Analysis.SourceCodeBlacklist.Get()

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
			Name:          chainState.Token.Name,
			Symbol:        chainState.Token.Symbol,
			Decimals:      uint32(chainState.Token.Decimals),
			TotalSupply:   bigIntToString(chainState.Token.TotalSupply),
			SourceCode:    sourceCode,
			SourceCodeABI: sourceCodeABI,
			IsValidERC20:  chainState.Token.IsValidERC20,
		},
		WethV2Pool: v1alpha1.PairV2State{
			IsContractCreated:   chainState.Pair.IsCreated,
			Contract:            addressToString(chainState.Pair.ContractAddress),
			Token0:              addressToString(chainState.Pair.Token0),
			Token1:              addressToString(chainState.Pair.Token1),
			TotalSupply:         bigIntToString(chainState.Pair.TotalSupply),
			Reserve0:            bigIntToString(chainState.Pair.Reserve0),
			Reserve1:            bigIntToString(chainState.Pair.Reserve1),
			BlockTimestampLast:  chainState.Pair.BlockTimestampLast,
			TokenReserveBalance: bigIntToString(chainState.Pair.TokenReserveBalance),
			WethReserveBalance:  bigIntToString(chainState.Pair.WethReserveBalance),
			LockedLiquidity:     bigIntToString(chainState.Pair.LockedLiquidity),
		},
		SourceCode: v1alpha1.ProjectSourceCodeState{
			SourceCode:    sourceCode,
			SourceCodeABI: sourceCodeABI,
		},
		Simulate: v1alpha1.ProjectSimulateState{
			CreatorResult: v1alpha1.SimulateResult{
				CanMintFromDeadViaTransferFrom: creatorResult.CanMintFromDeadViaTransferFrom,
				CanMintFromZeroViaTransferFrom: creatorResult.CanMintFromZeroViaTransferFrom,
				CanMintFromPairViaTransferFrom: creatorResult.CanMintFromPairViaTransferFrom,
				CanMintViaTransfer:             creatorResult.CanMintViaTransfer,
			},
		},
		Analysis: v1alpha1.ProjectAnalysisState{
			SourceCodeBlacklist: v1alpha1.SourceCodeBlacklistState{
				HasBlacklistFields: sourceCodeBlacklist.HasBlacklistFields,
				BlacklistFields:    cloneStringSlice(sourceCodeBlacklist.BlacklistFields),
			},
		},
	}
}

func bigIntToString(value *big.Int) string {
	if value == nil {
		return ""
	}
	return value.String()
}

func addressToString(value common.Address) string {
	if value == (common.Address{}) {
		return ""
	}
	return value.String()
}
