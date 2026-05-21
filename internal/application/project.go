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
	BlockTime       uint64
	BlockNumber     uint64
	Contract        common.Address
	Creator         common.Address
	Tx              *types.Transaction
	TxHash          common.Hash
	TxIndex         uint64
	IsArchived      bool
	ArchivedAt      time.Time
	SourceCode      string
	RuntimeCodeHash common.Hash

	// extra fields
	CreatorResult       SimulateResult
	SourceCodeBlacklist sourcecode.BlacklistReport
	GenesisWallets      []GenesisWalletMeta
}

type GenesisWalletMeta struct {
	Wallet    common.Address
	NetAmount *big.Int
	RatioBPS  int64
	RankIndex int32
}

func projectToView(project *Project, includeGenesisWallets bool) *v1alpha1.ProjectView {
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
	var genesisWallets []v1alpha1.GenesisWalletState
	var genesisWalletAssetStates []v1alpha1.GenesisWalletAssetState
	if includeGenesisWallets && len(project.Meta.GenesisWallets) > 0 {
		genesisWallets = make([]v1alpha1.GenesisWalletState, 0, len(project.Meta.GenesisWallets))
		for _, item := range project.Meta.GenesisWallets {
			genesisWallets = append(genesisWallets, v1alpha1.GenesisWalletState{
				Wallet:    item.Wallet.Hex(),
				NetAmount: bigIntToString(item.NetAmount),
				RatioBps:  item.RatioBPS,
				Rank:      item.RankIndex,
			})
		}
	}
	if includeGenesisWallets && len(chainState.GenesisWalletAssetStates) > 0 {
		genesisWalletAssetStates = make([]v1alpha1.GenesisWalletAssetState, 0, len(chainState.GenesisWalletAssetStates))
		for _, item := range chainState.GenesisWalletAssetStates {
			genesisWalletAssetStates = append(genesisWalletAssetStates, v1alpha1.GenesisWalletAssetState{
				Wallet: addressToString(item.Wallet),
				AssetState: v1alpha1.AssetState{
					TokenBalance:  bigIntToString(item.AssetState.TokenBalance),
					WethBalance:   bigIntToString(item.AssetState.WethBalance),
					UsdtBalance:   bigIntToString(item.AssetState.UsdtBalance),
					NativeBalance: bigIntToString(item.AssetState.NativeBalance),
					UsdtValue:     bigIntToString(item.AssetState.UsdtValue),
				},
			})
		}
	}

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
			IsArchived:     project.Meta.IsArchived,
			GenesisWallets: genesisWallets,
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
			AssetState: v1alpha1.AssetState{
				TokenBalance:  bigIntToString(chainState.AssetState.TokenBalance),
				WethBalance:   bigIntToString(chainState.AssetState.WethBalance),
				UsdtBalance:   bigIntToString(chainState.AssetState.UsdtBalance),
				NativeBalance: bigIntToString(chainState.AssetState.NativeBalance),
				UsdtValue:     bigIntToString(chainState.AssetState.UsdtValue),
			},
			GenesisWalletAssetStates: genesisWalletAssetStates,
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
