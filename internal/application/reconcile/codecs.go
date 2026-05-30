package reconcile

import (
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/useryege/athena/internal/application/model"
	appstore "github.com/useryege/athena/internal/application/store"
	athenacontract "github.com/useryege/athena/pkg/abi/ATHENA"
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
