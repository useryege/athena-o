package api

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
		CreatorResult:                      simulateResultFromStore(meta.CreatorResult),
		GenesisWalletsFetchedAt:            meta.GenesisWalletsFetchedAt,
		CreatorHistoricalProjectsFetchedAt: meta.CreatorHistoricalProjectsFetchedAt,
	}
}

func projectReportToStore(report ProjectReport) appstore.ProjectReport {
	return appstore.ProjectReport{
		IsReportEvaluated:          report.IsReportEvaluated,
		IsReportComplete:           report.IsReportComplete,
		IsBlacklistedCreatorWallet: report.IsBlacklistedCreatorWallet,
		IsBlacklistedGenesisWallet: report.IsBlacklistedGenesisWallet,
		IsBlacklistedBytecode:      report.IsBlacklistedBytecode,
		HasMintRisk:                report.HasMintRisk,
	}
}

func projectReportFromStore(report appstore.ProjectReport) ProjectReport {
	return ProjectReport{
		IsReportEvaluated:          report.IsReportEvaluated,
		IsReportComplete:           report.IsReportComplete,
		IsBlacklistedCreatorWallet: report.IsBlacklistedCreatorWallet,
		IsBlacklistedGenesisWallet: report.IsBlacklistedGenesisWallet,
		IsBlacklistedBytecode:      report.IsBlacklistedBytecode,
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
