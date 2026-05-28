package persistence

import appstore "github.com/useryege/athena/internal/application/store"

func projectReportToStore(report ProjectReport) appstore.ProjectReport {
	return appstore.ProjectReport{
		IsPolicyEvaluated:          report.IsPolicyEvaluated,
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
