package application

import (
	appstore "github.com/useryege/athena/internal/application/store"
)

func projectMetaToStore(meta ProjectMeta) appstore.ProjectMeta {
	return appstore.ProjectMeta{
		BlockTime:                    meta.BlockTime,
		BlockNumber:                  meta.BlockNumber,
		Contract:                     meta.Contract,
		Creator:                      meta.Creator,
		TxHash:                       meta.TxHash,
		TxIndex:                      meta.TxIndex,
		SourceCode:                   meta.SourceCode,
		SourceCodeHash:               meta.SourceCodeHash,
		SourceCodeFetchedAt:          meta.SourceCodeFetchedAt,
		CodeBinHash:                  meta.CodeBinHash,
		CodeBinHashFetchedAt:         meta.CodeBinHashFetchedAt,
		SourceQualityReport:          meta.SourceQualityReport,
		SourceQualityReportFetchedAt: meta.SourceQualityReportFetchedAt,
		GenesisWalletsFetchedAt:      meta.GenesisWalletsFetchedAt,
	}
}

func projectMetaFromStore(meta appstore.ProjectMeta) ProjectMeta {
	return ProjectMeta{
		BlockTime:                    meta.BlockTime,
		BlockNumber:                  meta.BlockNumber,
		Contract:                     meta.Contract,
		Creator:                      meta.Creator,
		TxHash:                       meta.TxHash,
		TxIndex:                      meta.TxIndex,
		SourceCode:                   meta.SourceCode,
		SourceCodeHash:               meta.SourceCodeHash,
		SourceCodeFetchedAt:          meta.SourceCodeFetchedAt,
		CodeBinHash:                  meta.CodeBinHash,
		CodeBinHashFetchedAt:         meta.CodeBinHashFetchedAt,
		SourceQualityReport:          meta.SourceQualityReport,
		SourceQualityReportFetchedAt: meta.SourceQualityReportFetchedAt,
		GenesisWalletsFetchedAt:      meta.GenesisWalletsFetchedAt,
	}
}
