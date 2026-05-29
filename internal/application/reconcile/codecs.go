package reconcile

import (
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/useryege/athena/internal/application/model"
	appstore "github.com/useryege/athena/internal/application/store"
	athenacontract "github.com/useryege/athena/pkg/abi/ATHENA"
	"github.com/useryege/athena/util/ave"
)

func buildProjectQueries(projects []*Project) ([]athenacontract.AthenaProjectQuery, []common.Address) {
	return model.BuildProjectQueries(projects)
}

func projectMetaToStore(meta ProjectMeta) appstore.ProjectMeta {
	return appstore.ProjectMeta{
		BlockTime:                          meta.BlockTime,
		BlockNumber:                        meta.BlockNumber,
		Contract:                           meta.Contract,
		Creator:                            meta.Creator,
		WethPair:                           meta.WethPair,
		UsdtPair:                           meta.UsdtPair,
		FetchAt:                            meta.FetchAt,
		TxHash:                             meta.TxHash,
		TxIndex:                            meta.TxIndex,
		CreatorResult:                      simulateResultToStore(meta.CreatorResult),
		GenesisWalletsFetchedAt:            meta.GenesisWalletsFetchedAt,
		CreatorHistoricalProjectsFetchedAt: meta.CreatorHistoricalProjectsFetchedAt,
	}
}

func projectBaseFromMeta(meta ProjectMeta) appstore.ProjectBase {
	return appstore.ProjectBase{
		BlockTime:   meta.BlockTime,
		BlockNumber: meta.BlockNumber,
		Contract:    meta.Contract,
		Creator:     meta.Creator,
		Tx:          meta.GenesisTx,
		TxHash:      meta.TxHash,
		TxIndex:     meta.TxIndex,
	}
}

func projectChainStateFromSnapshot(contract common.Address, snapshot athenacontract.AthenaProject, fetchedAt time.Time) appstore.ProjectChainState {
	if fetchedAt.IsZero() {
		fetchedAt = time.Now().UTC()
	}
	return appstore.ProjectChainState{
		ProjectContract: contract,
		ChainState:      snapshot,
		WethPair:        snapshot.WethPair.ContractAddress,
		UsdtPair:        snapshot.UsdtPair.ContractAddress,
		TokenName:       snapshot.Token.Name,
		TokenSymbol:     snapshot.Token.Symbol,
		FetchedAt:       fetchedAt.UTC(),
	}
}

func projectFromBase(base appstore.ProjectBase) *Project {
	return &Project{Meta: ProjectMeta{
		BlockTime:   base.BlockTime,
		BlockNumber: base.BlockNumber,
		Contract:    base.Contract,
		Creator:     base.Creator,
		TxHash:      base.TxHash,
		TxIndex:     base.TxIndex,
		GenesisTx:   base.Tx,
	}}
}

func projectMetaFromBase(base appstore.ProjectBase) ProjectMeta {
	return projectFromBase(base).Meta
}

func simulateResultToStore(result SimulateResult) appstore.SimulateResult {
	return appstore.SimulateResult{
		CanMintFromDeadViaTransferFrom:     result.CanMintFromDeadViaTransferFrom,
		CanMintFromZeroViaTransferFrom:     result.CanMintFromZeroViaTransferFrom,
		CanMintFromWethPairViaTransferFrom: result.CanMintFromWethPairViaTransferFrom,
		CanMintFromUsdtPairViaTransferFrom: result.CanMintFromUsdtPairViaTransferFrom,
		CanMintViaTransferToWethPair:       result.CanMintViaTransferToWethPair,
		CanMintViaTransferToUsdtPair:       result.CanMintViaTransferToUsdtPair,
	}
}

func projectAveDetailFromAveResponse(resp *ave.TokenDetailResponse, fetchedAt time.Time) *ProjectAveDetail {
	if resp == nil {
		return nil
	}
	if fetchedAt.IsZero() {
		fetchedAt = time.Now().UTC()
	}
	detail := &ProjectAveDetail{
		Status:    resp.Status,
		Msg:       resp.Msg,
		DataType:  resp.DataType,
		IsAudited: resp.Data.IsAudited,
		FetchedAt: fetchedAt.UTC(),
		Token:     projectAveTokenFromAve(resp.Data.Token),
		Pairs:     make([]ProjectAvePair, 0, len(resp.Data.Pairs)),
	}
	for _, pair := range resp.Data.Pairs {
		detail.Pairs = append(detail.Pairs, projectAvePairFromAve(pair))
	}
	return detail
}

func projectAveTokenFromAve(t ave.Token) ProjectAveTokenDetail {
	return ProjectAveTokenDetail{
		Total: t.Total, LaunchPrice: t.LaunchPrice, CurrentPriceETH: t.CurrentPriceETH, CurrentPriceUSD: t.CurrentPriceUSD, PriceChange1D: t.PriceChange1D, PriceChange24H: t.PriceChange24H, PriceChange1H: t.PriceChange1H,
		LockAmount: t.LockAmount, BurnAmount: t.BurnAmount, OtherAmount: t.OtherAmount, TxAmount24H: t.TxAmount24H, TxVolumeU24H: t.TxVolumeU24H, LockedPercent: t.LockedPercent, MarketCap: t.MarketCap, FDV: t.FDV, TVL: t.TVL, MainPairTVL: t.MainPairTVL,
		TokenPriceChange5M: t.TokenPriceChange5M, TokenPriceChange1H: t.TokenPriceChange1H, TokenPriceChange4H: t.TokenPriceChange4H, TokenPriceChange24H: t.TokenPriceChange24H, TokenTxVolumeUSD5M: t.TokenTxVolumeUSD5M, TokenTxVolumeUSD1H: t.TokenTxVolumeUSD1H, TokenTxVolumeUSD4H: t.TokenTxVolumeUSD4H, TokenTxVolumeUSD24H: t.TokenTxVolumeUSD24H,
		TokenBuyVolumeU5M: t.TokenBuyVolumeU5M, TokenSellVolumeU5M: t.TokenSellVolumeU5M, Token: t.Token, Chain: t.Chain, Decimal: t.Decimal, Name: t.Name, Symbol: t.Symbol, Holders: t.Holders, Appendix: t.Appendix, RiskLevel: t.RiskLevel, LogoURL: strings.TrimSpace(t.LogoURL),
		RiskInfo: t.RiskInfo, RiskScore: t.RiskScore, LaunchAt: t.LaunchAt, CreatedAt: t.CreatedAt, TxCount24H: t.TxCount24H, LockPlatform: t.LockPlatform, IsMintable: t.IsMintable, UpdatedAt: t.UpdatedAt, MainPair: t.MainPair,
		HasMintMethod: t.HasMintMethod, IsLPNotLocked: t.IsLPNotLocked, HasNotRenounced: t.HasNotRenounced, HasNotAudited: t.HasNotAudited, HasNotOpenSource: t.HasNotOpenSource, IsInBlacklist: t.IsInBlacklist, IsHoneypot: t.IsHoneypot, AveRiskLevel: t.AveRiskLevel,
	}
}

func projectAvePairFromAve(p ave.Pair) ProjectAvePair {
	return ProjectAvePair{
		Reserve0: p.Reserve0, Reserve1: p.Reserve1, Token0PriceETH: p.Token0PriceETH, Token0PriceUSD: p.Token0PriceUSD, Token1PriceETH: p.Token1PriceETH, Token1PriceUSD: p.Token1PriceUSD, PriceChange: p.PriceChange, PriceChange24H: p.PriceChange24H, PriceChange1H: p.PriceChange1H,
		VolumeU: p.VolumeU, LowU: p.LowU, HighU: p.HighU, Fee: p.Fee, TotalSupply: p.TotalSupply, TxAmount: p.TxAmount, Pair: p.Pair, Chain: p.Chain, AMM: p.AMM, Token0Address: p.Token0Address, Token0Symbol: p.Token0Symbol, Token0Decimal: p.Token0Decimal,
		Token1Address: p.Token1Address, Token1Symbol: p.Token1Symbol, Token1Decimal: p.Token1Decimal, TargetToken: p.TargetToken, PriceChange1D: p.PriceChange1D, CreatedAt: p.CreatedAt, TxCount: p.TxCount, UpdatedAt: p.UpdatedAt, MarketCap: p.MarketCap, FDV: p.FDV, IsFake: p.IsFake,
	}
}

func projectAveDetailToStore(detail *ProjectAveDetail) appstore.ProjectAveDetail {
	if detail == nil {
		return appstore.ProjectAveDetail{}
	}
	pairs := make([]appstore.ProjectAvePair, 0, len(detail.Pairs))
	for _, pair := range detail.Pairs {
		pairs = append(pairs, appstore.ProjectAvePair(pair))
	}
	return appstore.ProjectAveDetail{Status: detail.Status, Msg: detail.Msg, DataType: detail.DataType, IsAudited: detail.IsAudited, FetchedAt: detail.FetchedAt, Token: appstore.ProjectAveTokenDetail(detail.Token), Pairs: pairs}
}
