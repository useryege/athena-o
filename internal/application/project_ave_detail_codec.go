package application

import (
	"strings"
	"time"

	appstore "github.com/useryege/athena/internal/application/store"
	"github.com/useryege/athena/pkg/apis/application/v1alpha1"
	"github.com/useryege/athena/util/ave"
)

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
		pairs = append(pairs, projectAvePairToStore(pair))
	}
	return appstore.ProjectAveDetail{Status: detail.Status, Msg: detail.Msg, DataType: detail.DataType, IsAudited: detail.IsAudited, FetchedAt: detail.FetchedAt, Token: projectAveTokenToStore(detail.Token), Pairs: pairs}
}

func projectAveDetailFromStore(detail appstore.ProjectAveDetail) *ProjectAveDetail {
	pairs := make([]ProjectAvePair, 0, len(detail.Pairs))
	for _, pair := range detail.Pairs {
		pairs = append(pairs, projectAvePairFromStore(pair))
	}
	return &ProjectAveDetail{Status: detail.Status, Msg: detail.Msg, DataType: detail.DataType, IsAudited: detail.IsAudited, FetchedAt: detail.FetchedAt, Token: projectAveTokenFromStore(detail.Token), Pairs: pairs}
}

func projectAveTokenToStore(t ProjectAveTokenDetail) appstore.ProjectAveTokenDetail {
	return appstore.ProjectAveTokenDetail{
		Total: t.Total, LaunchPrice: t.LaunchPrice, CurrentPriceETH: t.CurrentPriceETH, CurrentPriceUSD: t.CurrentPriceUSD, PriceChange1D: t.PriceChange1D, PriceChange24H: t.PriceChange24H, PriceChange1H: t.PriceChange1H,
		LockAmount: t.LockAmount, BurnAmount: t.BurnAmount, OtherAmount: t.OtherAmount, TxAmount24H: t.TxAmount24H, TxVolumeU24H: t.TxVolumeU24H, LockedPercent: t.LockedPercent, MarketCap: t.MarketCap, FDV: t.FDV, TVL: t.TVL, MainPairTVL: t.MainPairTVL,
		TokenPriceChange5M: t.TokenPriceChange5M, TokenPriceChange1H: t.TokenPriceChange1H, TokenPriceChange4H: t.TokenPriceChange4H, TokenPriceChange24H: t.TokenPriceChange24H, TokenTxVolumeUSD5M: t.TokenTxVolumeUSD5M, TokenTxVolumeUSD1H: t.TokenTxVolumeUSD1H, TokenTxVolumeUSD4H: t.TokenTxVolumeUSD4H, TokenTxVolumeUSD24H: t.TokenTxVolumeUSD24H,
		TokenBuyVolumeU5M: t.TokenBuyVolumeU5M, TokenSellVolumeU5M: t.TokenSellVolumeU5M, Token: t.Token, Chain: t.Chain, Decimal: t.Decimal, Name: t.Name, Symbol: t.Symbol, Holders: t.Holders, Appendix: t.Appendix, RiskLevel: t.RiskLevel, LogoURL: t.LogoURL,
		RiskInfo: t.RiskInfo, RiskScore: t.RiskScore, LaunchAt: t.LaunchAt, CreatedAt: t.CreatedAt, TxCount24H: t.TxCount24H, LockPlatform: t.LockPlatform, IsMintable: t.IsMintable, UpdatedAt: t.UpdatedAt, MainPair: t.MainPair,
		HasMintMethod: t.HasMintMethod, IsLPNotLocked: t.IsLPNotLocked, HasNotRenounced: t.HasNotRenounced, HasNotAudited: t.HasNotAudited, HasNotOpenSource: t.HasNotOpenSource, IsInBlacklist: t.IsInBlacklist, IsHoneypot: t.IsHoneypot, AveRiskLevel: t.AveRiskLevel,
	}
}

func projectAveTokenFromStore(t appstore.ProjectAveTokenDetail) ProjectAveTokenDetail {
	return ProjectAveTokenDetail(projectAveTokenFromAve(ave.Token{
		Total: t.Total, LaunchPrice: t.LaunchPrice, CurrentPriceETH: t.CurrentPriceETH, CurrentPriceUSD: t.CurrentPriceUSD, PriceChange1D: t.PriceChange1D, PriceChange24H: t.PriceChange24H, PriceChange1H: t.PriceChange1H,
		LockAmount: t.LockAmount, BurnAmount: t.BurnAmount, OtherAmount: t.OtherAmount, TxAmount24H: t.TxAmount24H, TxVolumeU24H: t.TxVolumeU24H, LockedPercent: t.LockedPercent, MarketCap: t.MarketCap, FDV: t.FDV, TVL: t.TVL, MainPairTVL: t.MainPairTVL,
		TokenPriceChange5M: t.TokenPriceChange5M, TokenPriceChange1H: t.TokenPriceChange1H, TokenPriceChange4H: t.TokenPriceChange4H, TokenPriceChange24H: t.TokenPriceChange24H, TokenTxVolumeUSD5M: t.TokenTxVolumeUSD5M, TokenTxVolumeUSD1H: t.TokenTxVolumeUSD1H, TokenTxVolumeUSD4H: t.TokenTxVolumeUSD4H, TokenTxVolumeUSD24H: t.TokenTxVolumeUSD24H,
		TokenBuyVolumeU5M: t.TokenBuyVolumeU5M, TokenSellVolumeU5M: t.TokenSellVolumeU5M, Token: t.Token, Chain: t.Chain, Decimal: t.Decimal, Name: t.Name, Symbol: t.Symbol, Holders: t.Holders, Appendix: t.Appendix, RiskLevel: t.RiskLevel, LogoURL: t.LogoURL,
		RiskInfo: t.RiskInfo, RiskScore: t.RiskScore, LaunchAt: t.LaunchAt, CreatedAt: t.CreatedAt, TxCount24H: t.TxCount24H, LockPlatform: t.LockPlatform, IsMintable: t.IsMintable, UpdatedAt: t.UpdatedAt, MainPair: t.MainPair,
		HasMintMethod: t.HasMintMethod, IsLPNotLocked: t.IsLPNotLocked, HasNotRenounced: t.HasNotRenounced, HasNotAudited: t.HasNotAudited, HasNotOpenSource: t.HasNotOpenSource, IsInBlacklist: t.IsInBlacklist, IsHoneypot: t.IsHoneypot, AveRiskLevel: t.AveRiskLevel,
	}))
}

func projectAvePairToStore(p ProjectAvePair) appstore.ProjectAvePair {
	return appstore.ProjectAvePair(p)
}

func projectAvePairFromStore(p appstore.ProjectAvePair) ProjectAvePair {
	return ProjectAvePair(p)
}

func projectAveDetailToView(detail *ProjectAveDetail, includeDetailFields bool) v1alpha1.AveDetail {
	if detail == nil {
		return v1alpha1.AveDetail{}
	}
	fetchedAt := ""
	if includeDetailFields {
		fetchedAt = formatOptionalTime(detail.FetchedAt)
	}
	pairs := make([]v1alpha1.AvePair, 0, len(detail.Pairs))
	for _, pair := range detail.Pairs {
		pairs = append(pairs, projectAvePairToView(pair))
	}
	return v1alpha1.AveDetail{Status: int32(detail.Status), Msg: detail.Msg, DataType: int32(detail.DataType), IsAudited: detail.IsAudited, FetchedAt: fetchedAt, Token: projectAveTokenToView(detail.Token), Pairs: pairs}
}

func projectAveTokenToView(t ProjectAveTokenDetail) v1alpha1.AveTokenDetail {
	return v1alpha1.AveTokenDetail{
		Total: t.Total, LaunchPrice: t.LaunchPrice, CurrentPriceETH: t.CurrentPriceETH, CurrentPriceUSD: t.CurrentPriceUSD, PriceChange1D: t.PriceChange1D, PriceChange24H: t.PriceChange24H, PriceChange1H: t.PriceChange1H,
		LockAmount: t.LockAmount, BurnAmount: t.BurnAmount, OtherAmount: t.OtherAmount, TxAmount24H: t.TxAmount24H, TxVolumeU24H: t.TxVolumeU24H, LockedPercent: t.LockedPercent, MarketCap: t.MarketCap, FDV: t.FDV, TVL: t.TVL, MainPairTVL: t.MainPairTVL,
		TokenPriceChange5M: t.TokenPriceChange5M, TokenPriceChange1H: t.TokenPriceChange1H, TokenPriceChange4H: t.TokenPriceChange4H, TokenPriceChange24H: t.TokenPriceChange24H, TokenTxVolumeUSD5M: t.TokenTxVolumeUSD5M, TokenTxVolumeUSD1H: t.TokenTxVolumeUSD1H, TokenTxVolumeUSD4H: t.TokenTxVolumeUSD4H, TokenTxVolumeUSD24H: t.TokenTxVolumeUSD24H,
		TokenBuyVolumeU5M: t.TokenBuyVolumeU5M, TokenSellVolumeU5M: t.TokenSellVolumeU5M, Token: t.Token, Chain: t.Chain, Decimal: int32(t.Decimal), Name: t.Name, Symbol: t.Symbol, Holders: int32(t.Holders), Appendix: t.Appendix, RiskLevel: int32(t.RiskLevel), LogoURL: t.LogoURL,
		RiskInfo: t.RiskInfo, RiskScore: t.RiskScore, LaunchAt: t.LaunchAt, CreatedAt: t.CreatedAt, TxCount24H: int32(t.TxCount24H), LockPlatform: t.LockPlatform, IsMintable: t.IsMintable, UpdatedAt: t.UpdatedAt, MainPair: t.MainPair,
		HasMintMethod: t.HasMintMethod, IsLPNotLocked: t.IsLPNotLocked, HasNotRenounced: t.HasNotRenounced, HasNotAudited: t.HasNotAudited, HasNotOpenSource: t.HasNotOpenSource, IsInBlacklist: t.IsInBlacklist, IsHoneypot: t.IsHoneypot, AveRiskLevel: int32(t.AveRiskLevel),
	}
}

func projectAvePairToView(p ProjectAvePair) v1alpha1.AvePair {
	return v1alpha1.AvePair{
		Reserve0: p.Reserve0, Reserve1: p.Reserve1, Token0PriceETH: p.Token0PriceETH, Token0PriceUSD: p.Token0PriceUSD, Token1PriceETH: p.Token1PriceETH, Token1PriceUSD: p.Token1PriceUSD, PriceChange: p.PriceChange, PriceChange24H: p.PriceChange24H, PriceChange1H: p.PriceChange1H,
		VolumeU: p.VolumeU, LowU: p.LowU, HighU: p.HighU, Fee: p.Fee, TotalSupply: p.TotalSupply, TxAmount: p.TxAmount, Pair: p.Pair, Chain: p.Chain, AMM: p.AMM, Token0Address: p.Token0Address, Token0Symbol: p.Token0Symbol, Token0Decimal: int32(p.Token0Decimal),
		Token1Address: p.Token1Address, Token1Symbol: p.Token1Symbol, Token1Decimal: int32(p.Token1Decimal), TargetToken: p.TargetToken, PriceChange1D: p.PriceChange1D, CreatedAt: p.CreatedAt, TxCount: int32(p.TxCount), UpdatedAt: p.UpdatedAt, MarketCap: p.MarketCap, FDV: p.FDV, IsFake: p.IsFake,
	}
}
