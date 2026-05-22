package application

import (
	appstore "github.com/useryege/athena/internal/application/store"
)

func projectMetaToStore(meta ProjectMeta) appstore.ProjectMeta {
	return appstore.ProjectMeta{
		BlockTime:               meta.BlockTime,
		BlockNumber:             meta.BlockNumber,
		Contract:                meta.Contract,
		Creator:                 meta.Creator,
		TxHash:                  meta.TxHash,
		TxIndex:                 meta.TxIndex,
		SourceCode:              meta.SourceCode,
		SourceQualityReport:     meta.SourceQualityReport,
		SourceQualityReportedAt: meta.SourceQualityReportedAt,
		IsArchived:              meta.IsArchived,
		ArchivedAt:              meta.ArchivedAt,
	}
}

func projectMetaFromStore(meta appstore.ProjectMeta) ProjectMeta {
	return ProjectMeta{
		BlockTime:               meta.BlockTime,
		BlockNumber:             meta.BlockNumber,
		Contract:                meta.Contract,
		Creator:                 meta.Creator,
		TxHash:                  meta.TxHash,
		TxIndex:                 meta.TxIndex,
		SourceCode:              meta.SourceCode,
		SourceQualityReport:     meta.SourceQualityReport,
		SourceQualityReportedAt: meta.SourceQualityReportedAt,
		IsArchived:              meta.IsArchived,
		ArchivedAt:              meta.ArchivedAt,
	}
}
