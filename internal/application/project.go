package application

import (
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/useryege/athena/internal/application/sourcecode"
	athenacontract "github.com/useryege/athena/pkg/abi/ATHENA"
	"github.com/useryege/athena/pkg/apis/application/v1alpha1"
)

type Project struct {
	Meta    ProjectMeta
	Runtime ProjectRuntime
	Report  ProjectReport
}

type ProjectReport struct {
	IsPolicyEvaluated            bool
	IsBlacklistedCreatorWallet   bool
	IsBlacklistedGenesisWallet   bool
	IsBlacklistedBytecode        bool
	IsBlacklistedSourceCode      bool
	IsBlacklistedSourceCodeField bool
	HasMintRisk                  bool
	ShouldArchive                bool
}

type ProjectMeta struct {
	BlockTime               uint64
	BlockNumber             uint64
	Contract                common.Address
	Creator                 common.Address
	TxHash                  common.Hash
	TxIndex                 uint64
	IsArchived              bool
	ArchivedAt              time.Time
	SourceCode              string
	SourceCodeHash          common.Hash
	CodeBinHash             common.Hash
	SourceQualityReport     string
	SourceQualityReportedAt time.Time
	GenesisWallets          []GenesisWalletMeta
}

type ProjectRuntime struct {
	GenesisTx                      *types.Transaction
	ChainState                     athenacontract.AthenaProject
	CreatorResult                  SimulateResult
	CreatorOtherProjectContracts   []common.Address
	CreatorOtherProjectsResolvedAt time.Time
	SourceCodeBlacklist            sourcecode.BlacklistReport
}

type GenesisWalletMeta struct {
	Wallet    common.Address
	NetAmount *big.Int
	RatioBPS  int64
	RankIndex int32
}

func projectToView(project *Project, includeGenesisWallets bool) *v1alpha1.ProjectView {
	return projectToViewWithOptions(project, includeGenesisWallets, true)
}

func projectToListItem(project *Project) *v1alpha1.ProjectListItem {
	if project == nil {
		return nil
	}

	chainState := project.Runtime.ChainState
	creatorResult := project.Runtime.CreatorResult
	return &v1alpha1.ProjectListItem{
		Contract:                project.Meta.Contract.String(),
		Name:                    chainState.Token.Name,
		Symbol:                  chainState.Token.Symbol,
		IsArchived:              project.Meta.IsArchived,
		HasSourceCodeBlacklist:  project.Runtime.SourceCodeBlacklist.HasBlacklistFields,
		HasMintRisk:             hasMintRisk(creatorResult),
		IsOpenSource:            strings.TrimSpace(project.Meta.SourceCode) != "",
		WethPairQuoteUsdtValue:  bigIntToString(chainState.WethPair.QuoteUsdtValue),
		WethPairRemoveLiquidity: chainState.WethPair.IsRemoveLiquidity,
		UsdtPairQuoteUsdtValue:  bigIntToString(chainState.UsdtPair.QuoteUsdtValue),
		UsdtPairRemoveLiquidity: chainState.UsdtPair.IsRemoveLiquidity,
		CreatorAssetUsdtValue:   bigIntToString(chainState.AssetState.UsdtValue),
		BlockTime:               project.Meta.BlockTime,
		BlockNumber:             project.Meta.BlockNumber,
		TxIndex:                 project.Meta.TxIndex,
	}
}

func projectToViewWithOptions(project *Project, includeGenesisWallets bool, includeDetailFields bool) *v1alpha1.ProjectView {
	if project == nil {
		return nil
	}

	txHash := ""
	if project.Meta.TxHash != (common.Hash{}) {
		txHash = project.Meta.TxHash.Hex()
	} else if project.Runtime.GenesisTx != nil {
		txHash = project.Runtime.GenesisTx.Hash().Hex()
	}

	chainState := project.Runtime.ChainState
	sourceCode := ""
	sourceQualityReport := ""
	sourceQualityReportedAt := ""
	if includeDetailFields {
		sourceCode = project.Meta.SourceCode
		sourceQualityReport = project.Meta.SourceQualityReport
		sourceQualityReportedAt = formatOptionalTime(project.Meta.SourceQualityReportedAt)
	}
	creatorResult := project.Runtime.CreatorResult
	creatorOtherProjectContracts := make([]string, 0, len(project.Runtime.CreatorOtherProjectContracts))
	for _, contract := range project.Runtime.CreatorOtherProjectContracts {
		if contract == (common.Address{}) {
			continue
		}
		creatorOtherProjectContracts = append(creatorOtherProjectContracts, contract.Hex())
	}
	sourceCodeBlacklist := project.Runtime.SourceCodeBlacklist
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
			IsArchived:                   project.Meta.IsArchived,
			GenesisWallets:               genesisWallets,
			CreatorOtherProjectContracts: creatorOtherProjectContracts,
			SourceQualityReport:          sourceQualityReport,
			SourceQualityReportedAt:      sourceQualityReportedAt,
			IsOpenSource:                 strings.TrimSpace(project.Meta.SourceCode) != "",
			SourceCodeHash:               hashToString(project.Meta.SourceCodeHash),
			CodeBinHash:                  hashToString(project.Meta.CodeBinHash),
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

func hasMintRisk(result SimulateResult) bool {
	return result.CanMintViaTransferToWethPair ||
		result.CanMintViaTransferToUsdtPair ||
		result.CanMintFromDeadViaTransferFrom ||
		result.CanMintFromZeroViaTransferFrom ||
		result.CanMintFromWethPairViaTransferFrom ||
		result.CanMintFromUsdtPairViaTransferFrom
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

func hashToString(value common.Hash) string {
	if value == (common.Hash{}) {
		return ""
	}
	return value.Hex()
}

func formatOptionalTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339Nano)
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
