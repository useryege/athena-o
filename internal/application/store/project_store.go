package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	appsqlc "github.com/useryege/athena/internal/application/store/sqlc"
)

func (s *SQLStore) SaveProjectBase(ctx context.Context, base ProjectBase) error {
	txHash := base.TxHash
	if base.Tx != nil {
		txHash = base.Tx.Hash()
	}
	if txHash == (common.Hash{}) {
		return errors.New("project base transaction is nil")
	}
	if err := validateProjectBaseNumbers(base.BlockNumber, base.BlockTime, base.TxIndex); err != nil {
		return err
	}
	queries, err := s.querier()
	if err != nil {
		return err
	}
	err = queries.InsertProjectBase(ctx, appsqlc.InsertProjectBaseParams{
		ChainID:     s.chainIDForProject(base.ChainID),
		BlockNumber: int64(base.BlockNumber),
		BlockTime:   int64(base.BlockTime),
		Contract:    base.Contract.Bytes(),
		Creator:     base.Creator.Bytes(),
		TxHash:      txHash.Bytes(),
		TxIndex:     int64(base.TxIndex),
	})
	if err != nil {
		return fmt.Errorf("save project base: %w", err)
	}
	return nil
}

func (s *SQLStore) SaveProjectMeta(ctx context.Context, meta ProjectMeta) error {
	txHash := meta.TxHash
	if txHash == (common.Hash{}) && meta.GenesisTx != nil {
		txHash = meta.GenesisTx.Hash()
	}
	return s.SaveProjectBase(ctx, ProjectBase{
		ChainID:     meta.ChainID,
		BlockTime:   meta.BlockTime,
		BlockNumber: meta.BlockNumber,
		Contract:    meta.Contract,
		Creator:     meta.Creator,
		TxHash:      txHash,
		TxIndex:     meta.TxIndex,
	})
}

func (s *SQLStore) GetMaxProjectBlockNumber(ctx context.Context, chainID int64) (uint64, bool, error) {
	queries, err := s.querier()
	if err != nil {
		return 0, false, err
	}
	row, err := queries.GetMaxProjectBlockNumber(ctx, s.chainIDForProject(chainID))
	if err != nil {
		return 0, false, fmt.Errorf("get max project block number: %w", err)
	}
	if !row.HasValue {
		return 0, false, nil
	}
	if row.MaxBlock < 0 {
		return 0, false, fmt.Errorf("project base block number %d is negative", row.MaxBlock)
	}
	return uint64(row.MaxBlock), true, nil
}

func (s *SQLStore) ListProjectBases(ctx context.Context, chainID int64) ([]ProjectBase, error) {
	queries, err := s.querier()
	if err != nil {
		return nil, err
	}
	rows, err := queries.ListProjectBases(ctx, s.chainIDForProject(chainID))
	if err != nil {
		return nil, fmt.Errorf("list project bases: %w", err)
	}
	items := make([]ProjectBase, 0, len(rows))
	for _, row := range rows {
		item, err := projectBaseFromFields(row.ChainID, row.BlockNumber, row.BlockTime, row.Contract, row.Creator, row.TxHash, row.TxIndex, row.CreatedAt)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func (s *SQLStore) ListProjectBasesPage(ctx context.Context, chainID int64, page int32, pageSize int32) ([]ProjectBase, int64, int32, int32, error) {
	queries, err := s.querier()
	if err != nil {
		return nil, 0, 0, 0, err
	}
	page, pageSize = normalizePage(page, pageSize)
	chainID = s.chainIDForProject(chainID)
	total, err := queries.CountProjectBases(ctx, chainID)
	if err != nil {
		return nil, 0, page, pageSize, fmt.Errorf("count project bases: %w", err)
	}
	offset := int64(page-1) * int64(pageSize)
	if offset > math.MaxInt32 {
		return nil, 0, page, pageSize, fmt.Errorf("project base page offset %d exceeds int32", offset)
	}
	rows, err := queries.ListProjectBasesPage(ctx, appsqlc.ListProjectBasesPageParams{
		ChainID:     chainID,
		LimitCount:  pageSize,
		OffsetCount: int32(offset),
	})
	if err != nil {
		return nil, 0, page, pageSize, fmt.Errorf("list project bases page: %w", err)
	}
	items := make([]ProjectBase, 0, len(rows))
	for _, row := range rows {
		item, err := projectBaseFromFields(row.ChainID, row.BlockNumber, row.BlockTime, row.Contract, row.Creator, row.TxHash, row.TxIndex, row.CreatedAt)
		if err != nil {
			return nil, 0, page, pageSize, err
		}
		items = append(items, item)
	}
	return items, total, page, pageSize, nil
}

func (s *SQLStore) GetProjectBaseByContract(ctx context.Context, chainID int64, contract common.Address) (*ProjectBase, error) {
	queries, err := s.querier()
	if err != nil {
		return nil, err
	}
	row, err := queries.GetProjectBaseByContract(ctx, appsqlc.GetProjectBaseByContractParams{
		ChainID:  s.chainIDForProject(chainID),
		Contract: contract.Bytes(),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get project base by contract: %w", err)
	}
	base, err := projectBaseFromFields(row.ChainID, row.BlockNumber, row.BlockTime, row.Contract, row.Creator, row.TxHash, row.TxIndex, row.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &base, nil
}

func (s *SQLStore) ListProjectBasesByCreatorBefore(ctx context.Context, chainID int64, creator common.Address, blockNumber uint64, txIndex uint64) ([]ProjectBase, error) {
	if err := validateProjectOrderNumbers(blockNumber, txIndex); err != nil {
		return nil, err
	}
	queries, err := s.querier()
	if err != nil {
		return nil, err
	}
	rows, err := queries.ListProjectBasesByCreatorBefore(ctx, appsqlc.ListProjectBasesByCreatorBeforeParams{
		ChainID:     s.chainIDForProject(chainID),
		Creator:     creator.Bytes(),
		BlockNumber: int64(blockNumber),
		TxIndex:     int64(txIndex),
	})
	if err != nil {
		return nil, fmt.Errorf("list project bases by creator before: %w", err)
	}
	items := make([]ProjectBase, 0, len(rows))
	for _, row := range rows {
		item, err := projectBaseFromFields(row.ChainID, row.BlockNumber, row.BlockTime, row.Contract, row.Creator, row.TxHash, row.TxIndex, row.CreatedAt)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func (s *SQLStore) ListProjectMetas(ctx context.Context, chainID int64) ([]ProjectMeta, error) {
	queries, err := s.querier()
	if err != nil {
		return nil, err
	}
	rows, err := queries.ListProjectMetas(ctx, s.chainIDForProject(chainID))
	if err != nil {
		return nil, fmt.Errorf("list project metas: %w", err)
	}
	items := make([]ProjectMeta, 0, len(rows))
	for _, row := range rows {
		item, err := projectMetaFromFields(
			row.ChainID,
			row.BlockNumber,
			row.BlockTime,
			row.Contract,
			row.Creator,
			row.WethPair,
			row.UsdtPair,
			row.FetchAt,
			row.TxHash,
			row.TxIndex,
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
		items = append(items, item)
	}
	return items, nil
}

func (s *SQLStore) ListAllProjectMetas(ctx context.Context, chainID int64) ([]ProjectMeta, error) {
	return s.ListProjectMetas(ctx, chainID)
}

func (s *SQLStore) ListProjectMetasByPairAddresses(ctx context.Context, chainID int64, pairs []common.Address) ([]ProjectMeta, error) {
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
	items := make([]ProjectMeta, 0, len(rows))
	for _, row := range rows {
		item, err := projectMetaFromFields(
			row.ChainID,
			row.BlockNumber,
			row.BlockTime,
			row.Contract,
			row.Creator,
			row.WethPair,
			row.UsdtPair,
			row.FetchAt,
			row.TxHash,
			row.TxIndex,
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
		items = append(items, item)
	}
	return items, nil
}

func (s *SQLStore) ListProjectMetasByCreator(ctx context.Context, chainID int64, creator common.Address) ([]ProjectMeta, error) {
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
	items := make([]ProjectMeta, 0, len(rows))
	for _, row := range rows {
		item, err := projectMetaFromFields(
			row.ChainID,
			row.BlockNumber,
			row.BlockTime,
			row.Contract,
			row.Creator,
			row.WethPair,
			row.UsdtPair,
			row.FetchAt,
			row.TxHash,
			row.TxIndex,
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
		items = append(items, item)
	}
	return items, nil
}

func (s *SQLStore) ListProjectMetasByCreatorBefore(ctx context.Context, chainID int64, creator common.Address, blockNumber uint64, txIndex uint64) ([]ProjectMeta, error) {
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
	items := make([]ProjectMeta, 0, len(rows))
	for _, row := range rows {
		item, err := projectMetaFromFields(
			row.ChainID,
			row.BlockNumber,
			row.BlockTime,
			row.Contract,
			row.Creator,
			row.WethPair,
			row.UsdtPair,
			row.FetchAt,
			row.TxHash,
			row.TxIndex,
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
		items = append(items, item)
	}
	return items, nil
}

func (s *SQLStore) GetProjectMetaByContract(ctx context.Context, chainID int64, contract common.Address) (*ProjectMeta, error) {
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
	item, err := projectMetaFromFields(
		row.ChainID,
		row.BlockNumber,
		row.BlockTime,
		row.Contract,
		row.Creator,
		row.WethPair,
		row.UsdtPair,
		row.FetchAt,
		row.TxHash,
		row.TxIndex,
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

func (s *SQLStore) UpsertProjectChainState(ctx context.Context, item ProjectChainState) error {
	payload := item.RawChainState
	if len(payload) == 0 {
		data, err := json.Marshal(item.ChainState)
		if err != nil {
			return fmt.Errorf("marshal project chain state: %w", err)
		}
		payload = data
	}
	if item.WethPair == (common.Address{}) {
		item.WethPair = item.ChainState.WethPair.ContractAddress
	}
	if item.UsdtPair == (common.Address{}) {
		item.UsdtPair = item.ChainState.UsdtPair.ContractAddress
	}
	if item.TokenName == "" {
		item.TokenName = item.ChainState.Token.Name
	}
	if item.TokenSymbol == "" {
		item.TokenSymbol = item.ChainState.Token.Symbol
	}
	fetchedAt := item.FetchedAt
	if fetchedAt.IsZero() {
		fetchedAt = time.Now().UTC()
	}

	queries, err := s.querier()
	if err != nil {
		return err
	}
	err = queries.UpsertProjectChainState(ctx, appsqlc.UpsertProjectChainStateParams{
		ChainID:         s.chainIDForProject(item.ChainID),
		ProjectContract: item.ProjectContract.Bytes(),
		ChainState:      payload,
		WethPair:        item.WethPair.Bytes(),
		UsdtPair:        item.UsdtPair.Bytes(),
		FetchedAt:       pgtype.Timestamptz{Time: fetchedAt.UTC(), Valid: true},
		TokenName:       optionalPgText(item.TokenName),
		TokenSymbol:     optionalPgText(item.TokenSymbol),
	})
	if err != nil {
		return fmt.Errorf("upsert project chain state: %w", err)
	}
	return nil
}

func (s *SQLStore) GetProjectChainState(ctx context.Context, chainID int64, contract common.Address) (*ProjectChainState, error) {
	queries, err := s.querier()
	if err != nil {
		return nil, err
	}
	row, err := queries.GetProjectChainState(ctx, appsqlc.GetProjectChainStateParams{
		ChainID:         s.chainIDForProject(chainID),
		ProjectContract: contract.Bytes(),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get project chain state: %w", err)
	}
	item, err := projectChainStateFromFields(row.ChainID, row.ProjectContract, row.ChainState, row.WethPair, row.UsdtPair, row.TokenName, row.TokenSymbol, row.FetchedAt, row.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (s *SQLStore) ListProjectChainStatesByContracts(ctx context.Context, chainID int64, contracts []common.Address) (map[common.Address]ProjectChainState, error) {
	unique := uniqueNonZeroAddresses(contracts)
	result := make(map[common.Address]ProjectChainState, len(unique))
	if len(unique) == 0 {
		return result, nil
	}
	queries, err := s.querier()
	if err != nil {
		return nil, err
	}
	rows, err := queries.ListProjectChainStatesByContracts(ctx, appsqlc.ListProjectChainStatesByContractsParams{
		ChainID:          s.chainIDForProject(chainID),
		ProjectContracts: addressesToBytes(unique),
	})
	if err != nil {
		return nil, fmt.Errorf("list project chain states by contracts: %w", err)
	}
	for _, row := range rows {
		item, err := projectChainStateFromFields(row.ChainID, row.ProjectContract, row.ChainState, row.WethPair, row.UsdtPair, row.TokenName, row.TokenSymbol, row.FetchedAt, row.UpdatedAt)
		if err != nil {
			return nil, err
		}
		result[item.ProjectContract] = item
	}
	return result, nil
}

func (s *SQLStore) ListProjectChainStatesByPairAddresses(ctx context.Context, chainID int64, pairs []common.Address) ([]ProjectChainState, error) {
	unique := uniqueNonZeroAddresses(pairs)
	if len(unique) == 0 {
		return nil, nil
	}
	queries, err := s.querier()
	if err != nil {
		return nil, err
	}
	rows, err := queries.ListProjectChainStatesByPairAddresses(ctx, appsqlc.ListProjectChainStatesByPairAddressesParams{
		ChainID: s.chainIDForProject(chainID),
		Pairs:   addressesToBytes(unique),
	})
	if err != nil {
		return nil, fmt.Errorf("list project chain states by pair addresses: %w", err)
	}
	items := make([]ProjectChainState, 0, len(rows))
	for _, row := range rows {
		item, err := projectChainStateFromFields(row.ChainID, row.ProjectContract, row.ChainState, row.WethPair, row.UsdtPair, row.TokenName, row.TokenSymbol, row.FetchedAt, row.UpdatedAt)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func (s *SQLStore) UpsertProjectSimulationResult(ctx context.Context, item ProjectSimulationResult) error {
	fetchedAt := item.FetchedAt
	if fetchedAt.IsZero() {
		fetchedAt = time.Now().UTC()
	}
	r := item.Result
	queries, err := s.querier()
	if err != nil {
		return err
	}
	err = queries.UpsertProjectSimulationResult(ctx, appsqlc.UpsertProjectSimulationResultParams{
		ChainID:                            s.chainIDForProject(item.ChainID),
		ProjectContract:                    item.ProjectContract.Bytes(),
		CanMintFromDeadViaTransferFrom:     r.CanMintFromDeadViaTransferFrom,
		CanMintFromZeroViaTransferFrom:     r.CanMintFromZeroViaTransferFrom,
		CanMintFromWethPairViaTransferFrom: r.CanMintFromWethPairViaTransferFrom,
		CanMintFromUsdtPairViaTransferFrom: r.CanMintFromUsdtPairViaTransferFrom,
		CanMintViaTransferToWethPair:       r.CanMintViaTransferToWethPair,
		CanMintViaTransferToUsdtPair:       r.CanMintViaTransferToUsdtPair,
		FetchedAt:                          pgtype.Timestamptz{Time: fetchedAt.UTC(), Valid: true},
	})
	if err != nil {
		return fmt.Errorf("upsert project simulation result: %w", err)
	}
	return nil
}

func (s *SQLStore) UpdateProjectCreatorResult(ctx context.Context, chainID int64, contract common.Address, result SimulateResult) error {
	return s.UpsertProjectSimulationResult(ctx, ProjectSimulationResult{ChainID: chainID, ProjectContract: contract, Result: result})
}

func (s *SQLStore) GetProjectSimulationResult(ctx context.Context, chainID int64, contract common.Address) (*ProjectSimulationResult, error) {
	queries, err := s.querier()
	if err != nil {
		return nil, err
	}
	row, err := queries.GetProjectSimulationResult(ctx, appsqlc.GetProjectSimulationResultParams{
		ChainID:         s.chainIDForProject(chainID),
		ProjectContract: contract.Bytes(),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get project simulation result: %w", err)
	}
	item := ProjectSimulationResult{
		ChainID:         row.ChainID,
		ProjectContract: common.BytesToAddress(row.ProjectContract),
		Result: SimulateResult{
			CanMintFromDeadViaTransferFrom:     row.CanMintFromDeadViaTransferFrom,
			CanMintFromZeroViaTransferFrom:     row.CanMintFromZeroViaTransferFrom,
			CanMintFromWethPairViaTransferFrom: row.CanMintFromWethPairViaTransferFrom,
			CanMintFromUsdtPairViaTransferFrom: row.CanMintFromUsdtPairViaTransferFrom,
			CanMintViaTransferToWethPair:       row.CanMintViaTransferToWethPair,
			CanMintViaTransferToUsdtPair:       row.CanMintViaTransferToUsdtPair,
		},
		FetchedAt: row.FetchedAt.Time,
		UpdatedAt: row.UpdatedAt.Time,
	}
	return &item, nil
}

func (s *SQLStore) UpsertProjectBytecodeFact(ctx context.Context, item ProjectBytecodeFact) error {
	fetchedAt := item.FetchedAt
	if fetchedAt.IsZero() {
		fetchedAt = time.Now().UTC()
	}
	queries, err := s.querier()
	if err != nil {
		return err
	}
	err = queries.UpsertProjectBytecodeFact(ctx, appsqlc.UpsertProjectBytecodeFactParams{
		ChainID:               s.chainIDForProject(item.ChainID),
		ProjectContract:       item.ProjectContract.Bytes(),
		IsBytecodeBlacklisted: item.IsBytecodeBlacklisted,
		FetchedAt:             pgtype.Timestamptz{Time: fetchedAt.UTC(), Valid: true},
		CodeHash:              hashBytesOrNil(item.CodeHash),
	})
	if err != nil {
		return fmt.Errorf("upsert project bytecode fact: %w", err)
	}
	return nil
}

func (s *SQLStore) GetProjectBytecodeFact(ctx context.Context, chainID int64, contract common.Address) (*ProjectBytecodeFact, error) {
	queries, err := s.querier()
	if err != nil {
		return nil, err
	}
	row, err := queries.GetProjectBytecodeFact(ctx, appsqlc.GetProjectBytecodeFactParams{
		ChainID:         s.chainIDForProject(chainID),
		ProjectContract: contract.Bytes(),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get project bytecode fact: %w", err)
	}
	item := ProjectBytecodeFact{
		ChainID:               row.ChainID,
		ProjectContract:       common.BytesToAddress(row.ProjectContract),
		IsBytecodeBlacklisted: row.IsBytecodeBlacklisted,
		FetchedAt:             row.FetchedAt.Time,
		UpdatedAt:             row.UpdatedAt.Time,
	}
	if len(row.CodeHash) > 0 {
		item.CodeHash = common.BytesToHash(row.CodeHash)
	}
	return &item, nil
}

func (s *SQLStore) UpsertProjectComponentState(ctx context.Context, item ProjectComponentState) error {
	queries, err := s.querier()
	if err != nil {
		return err
	}
	err = queries.UpsertProjectComponentState(ctx, appsqlc.UpsertProjectComponentStateParams{
		ChainID:         s.chainIDForProject(item.ChainID),
		ProjectContract: item.ProjectContract.Bytes(),
		Component:       item.Component,
		Status:          item.Status,
		LastAttemptAt:   pgTime(item.LastAttemptAt),
		LastSuccessAt:   pgTime(item.LastSuccessAt),
		NextRunAt:       pgTime(item.NextRunAt),
		LastError:       optionalPgText(item.LastError),
	})
	if err != nil {
		return fmt.Errorf("upsert project component state: %w", err)
	}
	return nil
}

func (s *SQLStore) GetProjectComponentState(ctx context.Context, chainID int64, contract common.Address, component string) (*ProjectComponentState, error) {
	queries, err := s.querier()
	if err != nil {
		return nil, err
	}
	row, err := queries.GetProjectComponentState(ctx, appsqlc.GetProjectComponentStateParams{
		ChainID:         s.chainIDForProject(chainID),
		ProjectContract: contract.Bytes(),
		Component:       component,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get project component state: %w", err)
	}
	item := projectComponentStateFromFields(row.ChainID, row.ProjectContract, row.Component, row.Status, row.LastAttemptAt, row.LastSuccessAt, row.NextRunAt, row.LastError, row.UpdatedAt)
	return &item, nil
}

func projectBaseFromFields(chainID int64, blockNumber int64, blockTime int64, contract []byte, creator []byte, txHash []byte, txIndex int64, createdAt pgtype.Timestamptz) (ProjectBase, error) {
	if blockNumber < 0 || blockTime < 0 || txIndex < 0 {
		return ProjectBase{}, fmt.Errorf("project base has negative order fields")
	}
	item := ProjectBase{
		ChainID:     chainID,
		BlockNumber: uint64(blockNumber),
		BlockTime:   uint64(blockTime),
		Contract:    common.BytesToAddress(contract),
		Creator:     common.BytesToAddress(creator),
		TxHash:      common.BytesToHash(txHash),
		TxIndex:     uint64(txIndex),
	}
	if createdAt.Valid {
		item.CreatedAt = createdAt.Time
	}
	return item, nil
}

func projectMetaFromFields(
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
) (ProjectMeta, error) {
	if blockNumber < 0 || blockTime < 0 || txIndex < 0 {
		return ProjectMeta{}, fmt.Errorf("project meta has negative order fields")
	}
	item := ProjectMeta{
		ChainID:     chainID,
		BlockNumber: uint64(blockNumber),
		BlockTime:   uint64(blockTime),
		Contract:    common.BytesToAddress(contract),
		Creator:     common.BytesToAddress(creator),
		WethPair:    common.BytesToAddress(wethPair),
		UsdtPair:    common.BytesToAddress(usdtPair),
		TxHash:      common.BytesToHash(txHash),
		TxIndex:     uint64(txIndex),
		CreatorResult: SimulateResult{
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

func projectChainStateFromFields(chainID int64, projectContract []byte, chainState []byte, wethPair []byte, usdtPair []byte, tokenName pgtype.Text, tokenSymbol pgtype.Text, fetchedAt pgtype.Timestamptz, updatedAt pgtype.Timestamptz) (ProjectChainState, error) {
	item := ProjectChainState{
		ChainID:         chainID,
		ProjectContract: common.BytesToAddress(projectContract),
		RawChainState:   append(json.RawMessage(nil), chainState...),
		WethPair:        common.BytesToAddress(wethPair),
		UsdtPair:        common.BytesToAddress(usdtPair),
		TokenName:       tokenName.String,
		TokenSymbol:     tokenSymbol.String,
	}
	_ = json.Unmarshal(chainState, &item.ChainState)
	if fetchedAt.Valid {
		item.FetchedAt = fetchedAt.Time
	}
	if updatedAt.Valid {
		item.UpdatedAt = updatedAt.Time
	}
	return item, nil
}

func validateProjectBaseNumbers(blockNumber, blockTime, txIndex uint64) error {
	if blockTime > math.MaxInt64 {
		return fmt.Errorf("project base block time %d exceeds postgres BIGINT", blockTime)
	}
	return validateProjectOrderNumbers(blockNumber, txIndex)
}

func validateProjectOrderNumbers(blockNumber, txIndex uint64) error {
	if blockNumber > math.MaxInt64 {
		return fmt.Errorf("project base block number %d exceeds postgres BIGINT", blockNumber)
	}
	if txIndex > math.MaxInt64 {
		return fmt.Errorf("project base tx index %d exceeds postgres BIGINT", txIndex)
	}
	return nil
}

func uniqueNonZeroAddresses(items []common.Address) []common.Address {
	seen := make(map[common.Address]struct{}, len(items))
	unique := make([]common.Address, 0, len(items))
	for _, item := range items {
		if item == (common.Address{}) {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		unique = append(unique, item)
	}
	return unique
}

func addressesToBytes(items []common.Address) [][]byte {
	result := make([][]byte, 0, len(items))
	for _, item := range items {
		result = append(result, item.Bytes())
	}
	return result
}

func optionalPgText(value string) pgtype.Text {
	if value == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: value, Valid: true}
}

func hashBytesOrNil(value common.Hash) []byte {
	if value == (common.Hash{}) {
		return nil
	}
	return value.Bytes()
}

func normalizePage(page int32, pageSize int32) (int32, int32) {
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 200 {
		pageSize = 200
	}
	return page, pageSize
}
