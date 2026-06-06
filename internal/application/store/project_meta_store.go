package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/useryege/athena/internal/application/model"
	appsqlc "github.com/useryege/athena/internal/application/store/sqlc"
)

func (s *SQLStore) ListProjectMetas(ctx context.Context, chainID int64) ([]model.Project, error) {
	queries, err := s.querier()
	if err != nil {
		return nil, err
	}
	rows, err := queries.ListProjectMetas(ctx, s.chainIDForProject(chainID))
	if err != nil {
		return nil, fmt.Errorf("list project metas: %w", err)
	}
	return decodeProjectMetaRows(rows, func(i int) projectMetaRow {
		r := rows[i]
		return projectMetaRow{
			chainID: r.ChainID, blockNumber: r.BlockNumber, blockTime: r.BlockTime,
			contract: r.Contract, creator: r.Creator, wethPair: r.WethPair, usdtPair: r.UsdtPair,
			fetchAt: r.FetchAt, txHash: r.TxHash, txIndex: r.TxIndex,
			canMintFromDeadViaTransferFrom:     r.CanMintFromDeadViaTransferFrom,
			canMintFromZeroViaTransferFrom:     r.CanMintFromZeroViaTransferFrom,
			canMintFromWethPairViaTransferFrom: r.CanMintFromWethPairViaTransferFrom,
			canMintFromUsdtPairViaTransferFrom: r.CanMintFromUsdtPairViaTransferFrom,
			canMintViaTransferToWethPair:       r.CanMintViaTransferToWethPair,
			canMintViaTransferToUsdtPair:       r.CanMintViaTransferToUsdtPair,
			genesisWalletsFetchedAt:            r.GenesisWalletsFetchedAt,
			creatorHistoricalProjectsFetchedAt: r.CreatorHistoricalProjectsFetchedAt,
		}
	}, len(rows))
}

func (s *SQLStore) ListProjectMetasByPairAddresses(ctx context.Context, chainID int64, pairs []common.Address) ([]model.Project, error) {
	uniquePairs := uniqueNonZeroAddresses(pairs)
	if len(uniquePairs) == 0 {
		return nil, nil
	}
	queries, err := s.querier()
	if err != nil {
		return nil, err
	}
	rows, err := queries.ListProjectMetasByPairAddresses(ctx, appsqlc.ListProjectMetasByPairAddressesParams{
		ChainID: s.chainIDForProject(chainID),
		Pairs:   addressesToBytes(uniquePairs),
	})
	if err != nil {
		return nil, fmt.Errorf("list project metas by pair addresses: %w", err)
	}
	return decodeProjectMetaRows(rows, func(i int) projectMetaRow {
		r := rows[i]
		return projectMetaRow{
			chainID: r.ChainID, blockNumber: r.BlockNumber, blockTime: r.BlockTime,
			contract: r.Contract, creator: r.Creator, wethPair: r.WethPair, usdtPair: r.UsdtPair,
			fetchAt: r.FetchAt, txHash: r.TxHash, txIndex: r.TxIndex,
			canMintFromDeadViaTransferFrom:     r.CanMintFromDeadViaTransferFrom,
			canMintFromZeroViaTransferFrom:     r.CanMintFromZeroViaTransferFrom,
			canMintFromWethPairViaTransferFrom: r.CanMintFromWethPairViaTransferFrom,
			canMintFromUsdtPairViaTransferFrom: r.CanMintFromUsdtPairViaTransferFrom,
			canMintViaTransferToWethPair:       r.CanMintViaTransferToWethPair,
			canMintViaTransferToUsdtPair:       r.CanMintViaTransferToUsdtPair,
			genesisWalletsFetchedAt:            r.GenesisWalletsFetchedAt,
			creatorHistoricalProjectsFetchedAt: r.CreatorHistoricalProjectsFetchedAt,
		}
	}, len(rows))
}

func (s *SQLStore) ListProjectMetasByCreator(ctx context.Context, chainID int64, creator common.Address) ([]model.Project, error) {
	queries, err := s.querier()
	if err != nil {
		return nil, err
	}
	rows, err := queries.ListProjectMetasByCreator(ctx, appsqlc.ListProjectMetasByCreatorParams{
		ChainID: s.chainIDForProject(chainID),
		Creator: creator.Bytes(),
	})
	if err != nil {
		return nil, fmt.Errorf("list project metas by creator: %w", err)
	}
	return decodeProjectMetaRows(rows, func(i int) projectMetaRow {
		r := rows[i]
		return projectMetaRow{
			chainID: r.ChainID, blockNumber: r.BlockNumber, blockTime: r.BlockTime,
			contract: r.Contract, creator: r.Creator, wethPair: r.WethPair, usdtPair: r.UsdtPair,
			fetchAt: r.FetchAt, txHash: r.TxHash, txIndex: r.TxIndex,
			canMintFromDeadViaTransferFrom:     r.CanMintFromDeadViaTransferFrom,
			canMintFromZeroViaTransferFrom:     r.CanMintFromZeroViaTransferFrom,
			canMintFromWethPairViaTransferFrom: r.CanMintFromWethPairViaTransferFrom,
			canMintFromUsdtPairViaTransferFrom: r.CanMintFromUsdtPairViaTransferFrom,
			canMintViaTransferToWethPair:       r.CanMintViaTransferToWethPair,
			canMintViaTransferToUsdtPair:       r.CanMintViaTransferToUsdtPair,
			genesisWalletsFetchedAt:            r.GenesisWalletsFetchedAt,
			creatorHistoricalProjectsFetchedAt: r.CreatorHistoricalProjectsFetchedAt,
		}
	}, len(rows))
}

func (s *SQLStore) ListProjectMetasByCreatorBefore(ctx context.Context, chainID int64, creator common.Address, blockNumber uint64, txIndex uint64) ([]model.Project, error) {
	if err := validateProjectOrderNumbers(blockNumber, txIndex); err != nil {
		return nil, err
	}
	queries, err := s.querier()
	if err != nil {
		return nil, err
	}
	rows, err := queries.ListProjectMetasByCreatorBefore(ctx, appsqlc.ListProjectMetasByCreatorBeforeParams{
		ChainID:     s.chainIDForProject(chainID),
		Creator:     creator.Bytes(),
		BlockNumber: int64(blockNumber),
		TxIndex:     int64(txIndex),
	})
	if err != nil {
		return nil, fmt.Errorf("list project metas by creator before: %w", err)
	}
	return decodeProjectMetaRows(rows, func(i int) projectMetaRow {
		r := rows[i]
		return projectMetaRow{
			chainID: r.ChainID, blockNumber: r.BlockNumber, blockTime: r.BlockTime,
			contract: r.Contract, creator: r.Creator, wethPair: r.WethPair, usdtPair: r.UsdtPair,
			fetchAt: r.FetchAt, txHash: r.TxHash, txIndex: r.TxIndex,
			canMintFromDeadViaTransferFrom:     r.CanMintFromDeadViaTransferFrom,
			canMintFromZeroViaTransferFrom:     r.CanMintFromZeroViaTransferFrom,
			canMintFromWethPairViaTransferFrom: r.CanMintFromWethPairViaTransferFrom,
			canMintFromUsdtPairViaTransferFrom: r.CanMintFromUsdtPairViaTransferFrom,
			canMintViaTransferToWethPair:       r.CanMintViaTransferToWethPair,
			canMintViaTransferToUsdtPair:       r.CanMintViaTransferToUsdtPair,
			genesisWalletsFetchedAt:            r.GenesisWalletsFetchedAt,
			creatorHistoricalProjectsFetchedAt: r.CreatorHistoricalProjectsFetchedAt,
		}
	}, len(rows))
}

func (s *SQLStore) GetProjectMetaByContract(ctx context.Context, chainID int64, contract common.Address) (*model.Project, error) {
	queries, err := s.querier()
	if err != nil {
		return nil, err
	}
	row, err := queries.GetProjectMetaByContract(ctx, appsqlc.GetProjectMetaByContractParams{
		ChainID:  s.chainIDForProject(chainID),
		Contract: contract.Bytes(),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get project meta by contract: %w", err)
	}
	item, err := projectFromMetaFields(
		row.ChainID, row.BlockNumber, row.BlockTime,
		row.Contract, row.Creator, row.WethPair, row.UsdtPair,
		row.FetchAt, row.TxHash, row.TxIndex,
		row.CanMintFromDeadViaTransferFrom,
		row.CanMintFromZeroViaTransferFrom,
		row.CanMintFromWethPairViaTransferFrom,
		row.CanMintFromUsdtPairViaTransferFrom,
		row.CanMintViaTransferToWethPair,
		row.CanMintViaTransferToUsdtPair,
		row.GenesisWalletsFetchedAt,
		row.CreatorHistoricalProjectsFetchedAt,
	)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

// projectMetaRow is a normalized intermediate type used by decodeProjectMetaRows.
type projectMetaRow struct {
	chainID     int64
	blockNumber int64
	blockTime   int64
	contract    []byte
	creator     []byte
	wethPair    []byte
	usdtPair    []byte
	fetchAt     pgtype.Timestamptz
	txHash      []byte
	txIndex     int64

	canMintFromDeadViaTransferFrom     bool
	canMintFromZeroViaTransferFrom     bool
	canMintFromWethPairViaTransferFrom bool
	canMintFromUsdtPairViaTransferFrom bool
	canMintViaTransferToWethPair       bool
	canMintViaTransferToUsdtPair       bool

	genesisWalletsFetchedAt            pgtype.Timestamptz
	creatorHistoricalProjectsFetchedAt pgtype.Timestamptz
}

// decodeProjectMetaRows decodes n rows using the provided accessor into []model.Project.
func decodeProjectMetaRows[T any](rows []T, access func(int) projectMetaRow, n int) ([]model.Project, error) {
	items := make([]model.Project, 0, n)
	for i := range rows {
		r := access(i)
		item, err := projectFromMetaFields(
			r.chainID, r.blockNumber, r.blockTime,
			r.contract, r.creator, r.wethPair, r.usdtPair,
			r.fetchAt, r.txHash, r.txIndex,
			r.canMintFromDeadViaTransferFrom,
			r.canMintFromZeroViaTransferFrom,
			r.canMintFromWethPairViaTransferFrom,
			r.canMintFromUsdtPairViaTransferFrom,
			r.canMintViaTransferToWethPair,
			r.canMintViaTransferToUsdtPair,
			r.genesisWalletsFetchedAt,
			r.creatorHistoricalProjectsFetchedAt,
		)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func projectFromMetaFields(
	chainID int64,
	blockNumber int64,
	blockTime int64,
	contract []byte,
	creator []byte,
	wethPair []byte,
	usdtPair []byte,
	fetchAt pgtype.Timestamptz,
	txHash []byte,
	txIndex int64,
	canMintFromDeadViaTransferFrom bool,
	canMintFromZeroViaTransferFrom bool,
	canMintFromWethPairViaTransferFrom bool,
	canMintFromUsdtPairViaTransferFrom bool,
	canMintViaTransferToWethPair bool,
	canMintViaTransferToUsdtPair bool,
	genesisWalletsFetchedAt pgtype.Timestamptz,
	creatorHistoricalProjectsFetchedAt pgtype.Timestamptz,
) (model.Project, error) {
	if blockNumber < 0 || blockTime < 0 || txIndex < 0 {
		return model.Project{}, fmt.Errorf("project meta has negative order fields")
	}
	item := model.Project{
		ChainID:     chainID,
		BlockNumber: uint64(blockNumber),
		BlockTime:   uint64(blockTime),
		Contract:    common.BytesToAddress(contract),
		Creator:     common.BytesToAddress(creator),
		WethPair:    common.BytesToAddress(wethPair),
		UsdtPair:    common.BytesToAddress(usdtPair),
		TxHash:      common.BytesToHash(txHash),
		TxIndex:     uint64(txIndex),
		CreatorResult: model.SimulateResult{
			CanMintFromDeadViaTransferFrom:     canMintFromDeadViaTransferFrom,
			CanMintFromZeroViaTransferFrom:     canMintFromZeroViaTransferFrom,
			CanMintFromWethPairViaTransferFrom: canMintFromWethPairViaTransferFrom,
			CanMintFromUsdtPairViaTransferFrom: canMintFromUsdtPairViaTransferFrom,
			CanMintViaTransferToWethPair:       canMintViaTransferToWethPair,
			CanMintViaTransferToUsdtPair:       canMintViaTransferToUsdtPair,
		},
	}
	if fetchAt.Valid {
		item.FetchAt = fetchAt.Time
	}
	if genesisWalletsFetchedAt.Valid {
		item.GenesisWalletsFetchedAt = genesisWalletsFetchedAt.Time
	}
	if creatorHistoricalProjectsFetchedAt.Valid {
		item.CreatorHistoricalProjectsFetchedAt = creatorHistoricalProjectsFetchedAt.Time
	}
	return item, nil
}
