package api

import (
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	athenacontract "github.com/useryege/athena/pkg/abi/ATHENA"
	"github.com/useryege/athena/pkg/apis/application/v1alpha1"

	"github.com/useryege/athena/internal/application/model"
)

func projectToView(project *model.Project, includeGenesisWallets bool) *v1alpha1.ProjectView {
	return projectToViewWithOptions(project, includeGenesisWallets, true)
}

func projectToListItem(project *model.Project) *v1alpha1.ProjectListItem {
	if project == nil {
		return nil
	}

	chainState := project.Meta.ChainState
	creatorResult := project.Meta.CreatorResult
	aveToken := model.ProjectAveTokenDetail{}
	aveDetailAvailable := project.AveDetail != nil
	if aveDetailAvailable {
		aveToken = project.AveDetail.Token
	}
	return &v1alpha1.ProjectListItem{
		Contract:                project.Meta.Contract.String(),
		Name:                    chainState.Token.Name,
		Symbol:                  chainState.Token.Symbol,
		HasMintRisk:             creatorResult.HasMintRisk(),
		IsOpenSource:            false,
		WethPairQuoteUsdtValue:  bigIntToString(chainState.WethPair.QuoteUsdtValue),
		WethPairRemoveLiquidity: chainState.WethPair.IsRemoveLiquidity,
		UsdtPairQuoteUsdtValue:  bigIntToString(chainState.UsdtPair.QuoteUsdtValue),
		UsdtPairRemoveLiquidity: chainState.UsdtPair.IsRemoveLiquidity,
		CreatorAssetUsdtValue:   bigIntToString(chainState.AssetState.UsdtValue),
		BlockTime:               project.Meta.BlockTime,
		BlockNumber:             project.Meta.BlockNumber,
		TxIndex:                 project.Meta.TxIndex,
		AveLogo:                 projectAveLogo(project),
		AveDetailAvailable:      aveDetailAvailable,
		AveIsHoneypot:           aveToken.IsHoneypot,
		AveHasMintMethod:        aveToken.HasMintMethod,
		AveIsMintable:           aveToken.IsMintable,
		AveHolders:              int32(aveToken.Holders),
		AveMarketCap:            aveToken.MarketCap,
	}
}

func projectToViewWithOptions(project *model.Project, includeGenesisWallets bool, includeDetailFields bool) *v1alpha1.ProjectView {
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
	fetchAt := ""
	genesisWalletsFetchedAt := ""
	creatorHistoricalProjectsFetchedAt := ""
	if includeDetailFields {
		fetchAt = model.FormatOptionalTime(project.Meta.FetchAt)
		genesisWalletsFetchedAt = model.FormatOptionalTime(project.Meta.GenesisWalletsFetchedAt)
		creatorHistoricalProjectsFetchedAt = model.FormatOptionalTime(project.Meta.CreatorHistoricalProjectsFetchedAt)
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
			GenesisWalletsFetchedAt:            genesisWalletsFetchedAt,
			CreatorHistoricalProjectsFetchedAt: creatorHistoricalProjectsFetchedAt,
			FetchAt:                            fetchAt,
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
		AveDetail: projectAveDetailToView(project.AveDetail, includeDetailFields),
	}
}

func projectAveLogo(project *model.Project) string {
	if project == nil || project.AveDetail == nil {
		return ""
	}
	return strings.TrimSpace(project.AveDetail.Token.LogoURL)
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

func formatOptionalTime(value time.Time) string {
	return model.FormatOptionalTime(value)
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

func normalizeCachePage(page int32, pageSize int32) (int32, int32) {
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 200 {
		pageSize = 200
	}
	return page, pageSize
}
