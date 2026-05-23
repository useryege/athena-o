package application

import (
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	athenacontract "github.com/useryege/athena/pkg/abi/ATHENA"
	"github.com/useryege/athena/pkg/apis/application/v1alpha1"
)

type Project struct {
	Meta   ProjectMeta
	Report ProjectReport
}

type ProjectReport struct {
	IsPolicyEvaluated          bool
	IsBlacklistedCreatorWallet bool
	IsBlacklistedGenesisWallet bool
	IsBlacklistedBytecode      bool
	IsBlacklistedSourceCode    bool
	HasMintRisk                bool
}

type ProjectMeta struct {
	BlockTime                          uint64
	BlockNumber                        uint64
	Contract                           common.Address
	Creator                            common.Address
	TxHash                             common.Hash
	TxIndex                            uint64
	GenesisTx                          *types.Transaction
	ChainState                         athenacontract.AthenaProject
	SourceCode                         string
	SourceCodeHash                     common.Hash
	SourceCodeFetchedAt                time.Time
	CodeBinHash                        common.Hash
	CodeBinHashFetchedAt               time.Time
	SourceQualityReport                string
	SourceQualityReportFetchedAt       time.Time
	GenesisWallets                     []GenesisWalletMeta
	GenesisWalletsFetchedAt            time.Time
	CreatorResult                      SimulateResult
	CreatorResultFetchedAt             time.Time
	CreatorHistoricalProjects          []common.Address
	CreatorHistoricalProjectsFetchedAt time.Time
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

	chainState := project.Meta.ChainState
	creatorResult := project.Meta.CreatorResult
	return &v1alpha1.ProjectListItem{
		Contract:                project.Meta.Contract.String(),
		Name:                    chainState.Token.Name,
		Symbol:                  chainState.Token.Symbol,
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
	} else if project.Meta.GenesisTx != nil {
		txHash = project.Meta.GenesisTx.Hash().Hex()
	}

	chainState := project.Meta.ChainState
	sourceCode := ""
	sourceCodeFetchedAt := ""
	sourceQualityReport := ""
	sourceQualityReportFetchedAt := ""
	codeBinHashFetchedAt := ""
	genesisWalletsFetchedAt := ""
	creatorHistoricalProjectsFetchedAt := ""
	creatorResultFetchedAt := ""
	if includeDetailFields {
		sourceCode = project.Meta.SourceCode
		sourceCodeFetchedAt = formatOptionalTime(project.Meta.SourceCodeFetchedAt)
		sourceQualityReport = project.Meta.SourceQualityReport
		sourceQualityReportFetchedAt = formatOptionalTime(project.Meta.SourceQualityReportFetchedAt)
		codeBinHashFetchedAt = formatOptionalTime(project.Meta.CodeBinHashFetchedAt)
		genesisWalletsFetchedAt = formatOptionalTime(project.Meta.GenesisWalletsFetchedAt)
		creatorHistoricalProjectsFetchedAt = formatOptionalTime(project.Meta.CreatorHistoricalProjectsFetchedAt)
		creatorResultFetchedAt = formatOptionalTime(project.Meta.CreatorResultFetchedAt)
	}
	creatorResult := project.Meta.CreatorResult
	creatorHistoricalProjects := make([]string, 0, len(project.Meta.CreatorHistoricalProjects))
	for _, contract := range project.Meta.CreatorHistoricalProjects {
		if contract == (common.Address{}) {
			continue
		}
		creatorHistoricalProjects = append(creatorHistoricalProjects, contract.Hex())
	}
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
			GenesisWallets:                     genesisWallets,
			CreatorHistoricalProjects:          creatorHistoricalProjects,
			SourceQualityReport:                sourceQualityReport,
			SourceCodeFetchedAt:                sourceCodeFetchedAt,
			SourceQualityReportFetchedAt:       sourceQualityReportFetchedAt,
			IsOpenSource:                       strings.TrimSpace(project.Meta.SourceCode) != "",
			SourceCodeHash:                     hashToString(project.Meta.SourceCodeHash),
			CodeBinHash:                        hashToString(project.Meta.CodeBinHash),
			CodeBinHashFetchedAt:               codeBinHashFetchedAt,
			GenesisWalletsFetchedAt:            genesisWalletsFetchedAt,
			CreatorHistoricalProjectsFetchedAt: creatorHistoricalProjectsFetchedAt,
			CreatorResultFetchedAt:             creatorResultFetchedAt,
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
