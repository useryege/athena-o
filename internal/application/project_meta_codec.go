package application

import (
	appstore "github.com/useryege/athena/internal/application/store"
)

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
		SourceCode:                         meta.SourceCode,
		SourceCodeHash:                     meta.SourceCodeHash,
		SourceCodeFetchedAt:                meta.SourceCodeFetchedAt,
		CodeBinHash:                        meta.CodeBinHash,
		CodeBinHashFetchedAt:               meta.CodeBinHashFetchedAt,
		SourceQualityReport:                meta.SourceQualityReport,
		SourceQualityReportFetchedAt:       meta.SourceQualityReportFetchedAt,
		AveLogo:                            meta.AveLogo,
		AveLogoFetchedAt:                   meta.AveLogoFetchedAt,
		CreatorResult:                      simulateResultToStore(meta.CreatorResult),
		GenesisWalletsFetchedAt:            meta.GenesisWalletsFetchedAt,
		CreatorHistoricalProjectsFetchedAt: meta.CreatorHistoricalProjectsFetchedAt,
	}
}

func projectMetaFromStore(meta appstore.ProjectMeta) ProjectMeta {
	return ProjectMeta{
		BlockTime:                          meta.BlockTime,
		BlockNumber:                        meta.BlockNumber,
		Contract:                           meta.Contract,
		Creator:                            meta.Creator,
		WethPair:                           meta.WethPair,
		UsdtPair:                           meta.UsdtPair,
		FetchAt:                            meta.FetchAt,
		TxHash:                             meta.TxHash,
		TxIndex:                            meta.TxIndex,
		SourceCode:                         meta.SourceCode,
		SourceCodeHash:                     meta.SourceCodeHash,
		SourceCodeFetchedAt:                meta.SourceCodeFetchedAt,
		CodeBinHash:                        meta.CodeBinHash,
		CodeBinHashFetchedAt:               meta.CodeBinHashFetchedAt,
		SourceQualityReport:                meta.SourceQualityReport,
		SourceQualityReportFetchedAt:       meta.SourceQualityReportFetchedAt,
		AveLogo:                            meta.AveLogo,
		AveLogoFetchedAt:                   meta.AveLogoFetchedAt,
		CreatorResult:                      simulateResultFromStore(meta.CreatorResult),
		GenesisWalletsFetchedAt:            meta.GenesisWalletsFetchedAt,
		CreatorHistoricalProjectsFetchedAt: meta.CreatorHistoricalProjectsFetchedAt,
	}
}

func projectReportToStore(report ProjectReport) appstore.ProjectReport {
	return appstore.ProjectReport{
		IsPolicyEvaluated:          report.IsPolicyEvaluated,
		IsBlacklistedCreatorWallet: report.IsBlacklistedCreatorWallet,
		IsBlacklistedGenesisWallet: report.IsBlacklistedGenesisWallet,
		IsBlacklistedBytecode:      report.IsBlacklistedBytecode,
		IsBlacklistedSourceCode:    report.IsBlacklistedSourceCode,
		HasMintRisk:                report.HasMintRisk,
	}
}

func projectReportFromStore(report appstore.ProjectReport) ProjectReport {
	return ProjectReport{
		IsPolicyEvaluated:          report.IsPolicyEvaluated,
		IsBlacklistedCreatorWallet: report.IsBlacklistedCreatorWallet,
		IsBlacklistedGenesisWallet: report.IsBlacklistedGenesisWallet,
		IsBlacklistedBytecode:      report.IsBlacklistedBytecode,
		IsBlacklistedSourceCode:    report.IsBlacklistedSourceCode,
		HasMintRisk:                report.HasMintRisk,
	}
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

func simulateResultFromStore(result appstore.SimulateResult) SimulateResult {
	return SimulateResult{
		CanMintFromDeadViaTransferFrom:     result.CanMintFromDeadViaTransferFrom,
		CanMintFromZeroViaTransferFrom:     result.CanMintFromZeroViaTransferFrom,
		CanMintFromWethPairViaTransferFrom: result.CanMintFromWethPairViaTransferFrom,
		CanMintFromUsdtPairViaTransferFrom: result.CanMintFromUsdtPairViaTransferFrom,
		CanMintViaTransferToWethPair:       result.CanMintViaTransferToWethPair,
		CanMintViaTransferToUsdtPair:       result.CanMintViaTransferToUsdtPair,
	}
}
