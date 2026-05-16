package application

import (
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/useryege/athena/internal/application/sourcecode"
	athenacontract "github.com/useryege/athena/pkg/abi/ATHENA"
	"github.com/useryege/athena/pkg/apis/application/v1alpha1"
)

type Project struct {
	Meta       ProjectMeta
	ChainState athenacontract.AthenaProject
}

type ProjectMeta struct {
	BlockTime   uint64
	BlockNumber uint64
	Contract    common.Address
	Creator     common.Address
	Tx          *types.Transaction
	TxHash      common.Hash
	TxIndex     uint64
	IsArchived  bool
	ArchivedAt  time.Time
	SourceCode  string

	// extra fields
	CreatorResult       SimulateResult
	SourceCodeBlacklist sourcecode.BlacklistReport
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
	sourceCode := project.Meta.SourceCode
	creatorResult := project.Meta.CreatorResult
	sourceCodeBlacklist := project.Meta.SourceCodeBlacklist

	return &v1alpha1.ProjectView{
		Meta: v1alpha1.ProjectMeta{
			BlockTime:   project.Meta.BlockTime,
			BlockNumber: project.Meta.BlockNumber,
			Contract:    project.Meta.Contract.String(),
			Creator:     project.Meta.Creator.String(),
			TxHash:      txHash,
			TxIndex:     project.Meta.TxIndex,
			SourceCode:  sourceCode,
			CreatorResult: v1alpha1.SimulateResult{
				CanMintFromDeadViaTransferFrom:     creatorResult.CanMintFromDeadViaTransferFrom,
				CanMintFromZeroViaTransferFrom:     creatorResult.CanMintFromZeroViaTransferFrom,
				CanMintFromWethPairViaTransferFrom: creatorResult.CanMintFromWethPairViaTransferFrom,
				CanMintViaTransferToWethPair:       creatorResult.CanMintViaTransferToWethPair,
				CanMintViaTransferToUsdtPair:       creatorResult.CanMintViaTransferToUsdtPair,
				CanMintFromUsdtPairViaTransferFrom: creatorResult.CanMintFromUsdtPairViaTransferFrom,
			},
			SourceCodeBlacklist: v1alpha1.SourceCodeBlacklistState{
				HasBlacklistFields: sourceCodeBlacklist.HasBlacklistFields,
				BlacklistFields:    sourceCodeBlacklist.BlacklistFields,
			},
		},
		ChainState: v1alpha1.ProjectChainState{
			Token: v1alpha1.TokenState{
				Name:         chainState.Token.Name,
				Symbol:       chainState.Token.Symbol,
				Decimals:     uint32(chainState.Token.Decimals),
				TotalSupply:  bigIntToString(chainState.Token.TotalSupply),
				IsValidERC20: chainState.Token.IsValidERC20,
			},
			WethPair: pairToView(chainState.WethPair),
			UsdtPair: pairToView(chainState.UsdtPair),
			CreatorState: v1alpha1.CreatorState{
				TokenBalance:  bigIntToString(chainState.CreatorState.TokenBalance),
				WethBalance:   bigIntToString(chainState.CreatorState.WethBalance),
				UsdtBalance:   bigIntToString(chainState.CreatorState.UsdtBalance),
				NativeBalance: bigIntToString(chainState.CreatorState.NativeBalance),
			},
		},
	}
}

func pairToView(pair athenacontract.AthenaPair) v1alpha1.PairV2State {
	return v1alpha1.PairV2State{
		IsCreated:                      pair.IsCreated,
		Contract:                       addressToString(pair.ContractAddress),
		Token0:                         addressToString(pair.Token0),
		Token1:                         addressToString(pair.Token1),
		TotalSupply:                    bigIntToString(pair.TotalSupply),
		Reserve0:                       bigIntToString(pair.Reserve0),
		Reserve1:                       bigIntToString(pair.Reserve1),
		BlockTimestampLast:             pair.BlockTimestampLast,
		BaseBalance:                    bigIntToString(pair.BaseBalance),
		QuoteBalance:                   bigIntToString(pair.QuoteBalance),
		QuoteUsdtValue:                 bigIntToString(pair.QuoteUsdtValue),
		LockedLiquidity:                bigIntToString(pair.LockedLiquidity),
		FeeAddressHoldLiquidityBalance: bigIntToString(pair.FeeAddressHoldLiquidityBalance),
		IsRemoveLiquidity:              pair.IsRemoveLiquidity,
		FeeAddressHoldLiquidityRatio:   bigIntToString(pair.FeeAddressHoldLiquidityRatio),
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
