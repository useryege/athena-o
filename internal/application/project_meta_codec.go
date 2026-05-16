package application

import (
	appstore "github.com/useryege/athena/internal/application/store"
)

func projectMetaToStore(meta ProjectMeta) appstore.ProjectMeta {
	txHash := meta.TxHash
	if meta.Tx != nil {
		txHash = meta.Tx.Hash()
	}
	return appstore.ProjectMeta{
		BlockTime:   meta.BlockTime,
		BlockNumber: meta.BlockNumber,
		Contract:    meta.Contract,
		Creator:     meta.Creator,
		Tx:          meta.Tx,
		TxHash:      txHash,
		TxIndex:     meta.TxIndex,
		SourceCode:  meta.SourceCode,
		IsArchived:  meta.IsArchived,
		ArchivedAt:  meta.ArchivedAt,
	}
}

func projectMetaFromStore(meta appstore.ProjectMeta) ProjectMeta {
	return ProjectMeta{
		BlockTime:   meta.BlockTime,
		BlockNumber: meta.BlockNumber,
		Contract:    meta.Contract,
		Creator:     meta.Creator,
		Tx:          meta.Tx,
		TxHash:      meta.TxHash,
		TxIndex:     meta.TxIndex,
		SourceCode:  meta.SourceCode,
		IsArchived:  meta.IsArchived,
		ArchivedAt:  meta.ArchivedAt,
	}
}
