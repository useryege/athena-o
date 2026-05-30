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
	return s.SaveProjectBase(ctx, ProjectBase{
		BlockTime:   meta.BlockTime,
		BlockNumber: meta.BlockNumber,
		Contract:    meta.Contract,
		Creator:     meta.Creator,
		Tx:          meta.Tx,
		TxHash:      meta.TxHash,
		TxIndex:     meta.TxIndex,
	})
}

func (s *SQLStore) GetMaxProjectBlockNumber(ctx context.Context) (uint64, bool, error) {
	queries, err := s.querier()
	if err != nil {
		return 0, false, err
	}
	row, err := queries.GetMaxProjectBlockNumber(ctx)
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

func (s *SQLStore) ListProjectBases(ctx context.Context) ([]ProjectBase, error) {
	queries, err := s.querier()
	if err != nil {
		return nil, err
	}
	rows, err := queries.ListProjectBases(ctx)
	if err != nil {
		return nil, fmt.Errorf("list project bases: %w", err)
	}
	items := make([]ProjectBase, 0, len(rows))
	for _, row := range rows {
		item, err := projectBaseFromFields(row.BlockNumber, row.BlockTime, row.Contract, row.Creator, row.TxHash, row.TxIndex, row.CreatedAt)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func (s *SQLStore) ListProjectBasesPage(ctx context.Context, page int32, pageSize int32) ([]ProjectBase, int64, int32, int32, error) {
	queries, err := s.querier()
	if err != nil {
		return nil, 0, 0, 0, err
	}
	page, pageSize = normalizePage(page, pageSize)
	total, err := queries.CountProjectBases(ctx)
	if err != nil {
		return nil, 0, page, pageSize, fmt.Errorf("count project bases: %w", err)
	}
	offset := int64(page-1) * int64(pageSize)
	if offset > math.MaxInt32 {
		return nil, 0, page, pageSize, fmt.Errorf("project base page offset %d exceeds int32", offset)
	}
	rows, err := queries.ListProjectBasesPage(ctx, appsqlc.ListProjectBasesPageParams{
		Limit:  pageSize,
		Offset: int32(offset),
	})
	if err != nil {
		return nil, 0, page, pageSize, fmt.Errorf("list project bases page: %w", err)
	}
	items := make([]ProjectBase, 0, len(rows))
	for _, row := range rows {
		item, err := projectBaseFromFields(row.BlockNumber, row.BlockTime, row.Contract, row.Creator, row.TxHash, row.TxIndex, row.CreatedAt)
		if err != nil {
			return nil, 0, page, pageSize, err
		}
		items = append(items, item)
	}
	return items, total, page, pageSize, nil
}

func (s *SQLStore) GetProjectBaseByContract(ctx context.Context, contract common.Address) (*ProjectBase, error) {
	queries, err := s.querier()
	if err != nil {
		return nil, err
	}
	row, err := queries.GetProjectBaseByContract(ctx, contract.Bytes())
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get project base by contract: %w", err)
	}
	base, err := projectBaseFromFields(row.BlockNumber, row.BlockTime, row.Contract, row.Creator, row.TxHash, row.TxIndex, row.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &base, nil
}

func (s *SQLStore) ListProjectBasesByCreatorBefore(ctx context.Context, creator common.Address, blockNumber uint64, txIndex uint64) ([]ProjectBase, error) {
	if err := validateProjectOrderNumbers(blockNumber, txIndex); err != nil {
		return nil, err
	}
	queries, err := s.querier()
	if err != nil {
		return nil, err
	}
	rows, err := queries.ListProjectBasesByCreatorBefore(ctx, appsqlc.ListProjectBasesByCreatorBeforeParams{
		Creator:     creator.Bytes(),
		BlockNumber: int64(blockNumber),
		TxIndex:     int64(txIndex),
	})
	if err != nil {
		return nil, fmt.Errorf("list project bases by creator before: %w", err)
	}
	items := make([]ProjectBase, 0, len(rows))
	for _, row := range rows {
		item, err := projectBaseFromFields(row.BlockNumber, row.BlockTime, row.Contract, row.Creator, row.TxHash, row.TxIndex, row.CreatedAt)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func (s *SQLStore) ListProjectMetas(ctx context.Context) ([]ProjectMeta, error) {
	queries, err := s.querier()
	if err != nil {
		return nil, err
	}
	rows, err := queries.ListProjectMetas(ctx)
	if err != nil {
		return nil, fmt.Errorf("list project metas: %w", err)
	}
	items := make([]ProjectMeta, 0, len(rows))
	for _, row := range rows {
		item, err := projectMetaFromFields(
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
			row.IsReportEvaluated,
			row.IsReportComplete,
			row.IsBlacklistedCreatorWallet,
			row.IsBlacklistedGenesisWallet,
			row.IsBlacklistedBytecode,
			row.HasMintRisk,
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

func (s *SQLStore) ListAllProjectMetas(ctx context.Context) ([]ProjectMeta, error) {
	return s.ListProjectMetas(ctx)
}

func (s *SQLStore) ListProjectMetasByPairAddresses(ctx context.Context, pairs []common.Address) ([]ProjectMeta, error) {
	uniquePairs := uniqueNonZeroAddresses(pairs)
	if len(uniquePairs) == 0 {
		return nil, nil
	}
	queries, err := s.querier()
	if err != nil {
		return nil, err
	}
	rows, err := queries.ListProjectMetasByPairAddresses(ctx, addressesToBytes(uniquePairs))
	if err != nil {
		return nil, fmt.Errorf("list project metas by pair addresses: %w", err)
	}
	items := make([]ProjectMeta, 0, len(rows))
	for _, row := range rows {
		item, err := projectMetaFromFields(
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
			row.IsReportEvaluated,
			row.IsReportComplete,
			row.IsBlacklistedCreatorWallet,
			row.IsBlacklistedGenesisWallet,
			row.IsBlacklistedBytecode,
			row.HasMintRisk,
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

func (s *SQLStore) ListProjectMetasByCreator(ctx context.Context, creator common.Address) ([]ProjectMeta, error) {
	queries, err := s.querier()
	if err != nil {
		return nil, err
	}
	rows, err := queries.ListProjectMetasByCreator(ctx, creator.Bytes())
	if err != nil {
		return nil, fmt.Errorf("list project metas by creator: %w", err)
	}
	items := make([]ProjectMeta, 0, len(rows))
	for _, row := range rows {
		item, err := projectMetaFromFields(
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
			row.IsReportEvaluated,
			row.IsReportComplete,
			row.IsBlacklistedCreatorWallet,
			row.IsBlacklistedGenesisWallet,
			row.IsBlacklistedBytecode,
			row.HasMintRisk,
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

func (s *SQLStore) ListProjectMetasByCreatorBefore(ctx context.Context, creator common.Address, blockNumber uint64, txIndex uint64) ([]ProjectMeta, error) {
	if err := validateProjectOrderNumbers(blockNumber, txIndex); err != nil {
		return nil, err
	}
	queries, err := s.querier()
	if err != nil {
		return nil, err
	}
	rows, err := queries.ListProjectMetasByCreatorBefore(ctx, appsqlc.ListProjectMetasByCreatorBeforeParams{
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
			row.IsReportEvaluated,
			row.IsReportComplete,
			row.IsBlacklistedCreatorWallet,
			row.IsBlacklistedGenesisWallet,
			row.IsBlacklistedBytecode,
			row.HasMintRisk,
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

func (s *SQLStore) GetProjectMetaByContract(ctx context.Context, contract common.Address) (*ProjectMeta, error) {
	queries, err := s.querier()
	if err != nil {
		return nil, err
	}
	row, err := queries.GetProjectMetaByContract(ctx, contract.Bytes())
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get project meta by contract: %w", err)
	}
	item, err := projectMetaFromFields(
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
		row.IsReportEvaluated,
		row.IsReportComplete,
		row.IsBlacklistedCreatorWallet,
		row.IsBlacklistedGenesisWallet,
		row.IsBlacklistedBytecode,
		row.HasMintRisk,
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
		ProjectContract: item.ProjectContract.Bytes(),
		Column2:         payload,
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

func (s *SQLStore) GetProjectChainState(ctx context.Context, contract common.Address) (*ProjectChainState, error) {
	queries, err := s.querier()
	if err != nil {
		return nil, err
	}
	row, err := queries.GetProjectChainState(ctx, contract.Bytes())
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get project chain state: %w", err)
	}
	item, err := projectChainStateFromSQLC(row)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (s *SQLStore) ListProjectChainStatesByContracts(ctx context.Context, contracts []common.Address) (map[common.Address]ProjectChainState, error) {
	unique := uniqueNonZeroAddresses(contracts)
	result := make(map[common.Address]ProjectChainState, len(unique))
	if len(unique) == 0 {
		return result, nil
	}
	queries, err := s.querier()
	if err != nil {
		return nil, err
	}
	rows, err := queries.ListProjectChainStatesByContracts(ctx, addressesToBytes(unique))
	if err != nil {
		return nil, fmt.Errorf("list project chain states by contracts: %w", err)
	}
	for _, row := range rows {
		item, err := projectChainStateFromSQLC(row)
		if err != nil {
			return nil, err
		}
		result[item.ProjectContract] = item
	}
	return result, nil
}

func (s *SQLStore) ListProjectChainStatesByPairAddresses(ctx context.Context, pairs []common.Address) ([]ProjectChainState, error) {
	unique := uniqueNonZeroAddresses(pairs)
	if len(unique) == 0 {
		return nil, nil
	}
	queries, err := s.querier()
	if err != nil {
		return nil, err
	}
	rows, err := queries.ListProjectChainStatesByPairAddresses(ctx, addressesToBytes(unique))
	if err != nil {
		return nil, fmt.Errorf("list project chain states by pair addresses: %w", err)
	}
	items := make([]ProjectChainState, 0, len(rows))
	for _, row := range rows {
		item, err := projectChainStateFromSQLC(row)
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

func (s *SQLStore) UpdateProjectCreatorResult(ctx context.Context, contract common.Address, result SimulateResult) error {
	return s.UpsertProjectSimulationResult(ctx, ProjectSimulationResult{ProjectContract: contract, Result: result})
}

func (s *SQLStore) GetProjectSimulationResult(ctx context.Context, contract common.Address) (*ProjectSimulationResult, error) {
	queries, err := s.querier()
	if err != nil {
		return nil, err
	}
	row, err := queries.GetProjectSimulationResult(ctx, contract.Bytes())
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get project simulation result: %w", err)
	}
	item := ProjectSimulationResult{
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

func (s *SQLStore) UpsertProjectReportState(ctx context.Context, item ProjectReportState) error {
	evaluatedAt := item.EvaluatedAt
	if evaluatedAt.IsZero() {
		evaluatedAt = time.Now().UTC()
	}
	r := item.Report
	queries, err := s.querier()
	if err != nil {
		return err
	}
	err = queries.UpsertProjectReportState(ctx, appsqlc.UpsertProjectReportStateParams{
		ProjectContract:            item.ProjectContract.Bytes(),
		IsReportEvaluated:          r.IsReportEvaluated,
		IsReportComplete:           r.IsReportComplete,
		IsBlacklistedCreatorWallet: r.IsBlacklistedCreatorWallet,
		IsBlacklistedGenesisWallet: r.IsBlacklistedGenesisWallet,
		IsBlacklistedBytecode:      r.IsBlacklistedBytecode,
		HasMintRisk:                r.HasMintRisk,
		EvaluatedAt:                pgtype.Timestamptz{Time: evaluatedAt.UTC(), Valid: true},
	})
	if err != nil {
		return fmt.Errorf("upsert project report: %w", err)
	}
	return nil
}

func (s *SQLStore) UpdateProjectReport(ctx context.Context, contract common.Address, report ProjectReport) error {
	return s.UpsertProjectReportState(ctx, ProjectReportState{ProjectContract: contract, Report: report})
}

func (s *SQLStore) GetProjectReportState(ctx context.Context, contract common.Address) (*ProjectReportState, error) {
	queries, err := s.querier()
	if err != nil {
		return nil, err
	}
	row, err := queries.GetProjectReportState(ctx, contract.Bytes())
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get project report state: %w", err)
	}
	item := projectReportStateFromSQLC(row)
	return &item, nil
}

func (s *SQLStore) ListProjectReportStatesByContracts(ctx context.Context, contracts []common.Address) (map[common.Address]ProjectReportState, error) {
	unique := uniqueNonZeroAddresses(contracts)
	result := make(map[common.Address]ProjectReportState, len(unique))
	if len(unique) == 0 {
		return result, nil
	}
	queries, err := s.querier()
	if err != nil {
		return nil, err
	}
	rows, err := queries.ListProjectReportStatesByContracts(ctx, addressesToBytes(unique))
	if err != nil {
		return nil, fmt.Errorf("list project reports by contracts: %w", err)
	}
	for _, row := range rows {
		item := projectReportStateFromSQLC(row)
		result[item.ProjectContract] = item
	}
	return result, nil
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

func (s *SQLStore) GetProjectBytecodeFact(ctx context.Context, contract common.Address) (*ProjectBytecodeFact, error) {
	queries, err := s.querier()
	if err != nil {
		return nil, err
	}
	row, err := queries.GetProjectBytecodeFact(ctx, contract.Bytes())
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get project bytecode fact: %w", err)
	}
	item := ProjectBytecodeFact{
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

func (s *SQLStore) GetProjectComponentState(ctx context.Context, contract common.Address, component string) (*ProjectComponentState, error) {
	queries, err := s.querier()
	if err != nil {
		return nil, err
	}
	row, err := queries.GetProjectComponentState(ctx, appsqlc.GetProjectComponentStateParams{
		ProjectContract: contract.Bytes(),
		Component:       component,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get project component state: %w", err)
	}
	item := projectComponentStateFromSQLC(row)
	return &item, nil
}

func projectBaseFromFields(blockNumber int64, blockTime int64, contract []byte, creator []byte, txHash []byte, txIndex int64, createdAt pgtype.Timestamptz) (ProjectBase, error) {
	if blockNumber < 0 || blockTime < 0 || txIndex < 0 {
		return ProjectBase{}, fmt.Errorf("project base has negative order fields")
	}
	item := ProjectBase{
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
	isReportEvaluated bool,
	isReportComplete bool,
	isBlacklistedCreatorWallet bool,
	isBlacklistedGenesisWallet bool,
	isBlacklistedBytecode bool,
	hasMintRisk bool,
	genesisWalletsFetchedAt pgtype.Timestamptz,
	creatorHistoricalProjectsFetchedAt pgtype.Timestamptz,
) (ProjectMeta, error) {
	if blockNumber < 0 || blockTime < 0 || txIndex < 0 {
		return ProjectMeta{}, fmt.Errorf("project meta has negative order fields")
	}
	item := ProjectMeta{
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
		Report: ProjectReport{
			IsReportEvaluated:          isReportEvaluated,
			IsReportComplete:           isReportComplete,
			IsBlacklistedCreatorWallet: isBlacklistedCreatorWallet,
			IsBlacklistedGenesisWallet: isBlacklistedGenesisWallet,
			IsBlacklistedBytecode:      isBlacklistedBytecode,
			HasMintRisk:                hasMintRisk,
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

func projectChainStateFromSQLC(row appsqlc.ProjectChainState) (ProjectChainState, error) {
	item := ProjectChainState{
		ProjectContract: common.BytesToAddress(row.ProjectContract),
		RawChainState:   append(json.RawMessage(nil), row.ChainState...),
		WethPair:        common.BytesToAddress(row.WethPair),
		UsdtPair:        common.BytesToAddress(row.UsdtPair),
		TokenName:       row.TokenName.String,
		TokenSymbol:     row.TokenSymbol.String,
	}
	_ = json.Unmarshal(row.ChainState, &item.ChainState)
	if row.FetchedAt.Valid {
		item.FetchedAt = row.FetchedAt.Time
	}
	if row.UpdatedAt.Valid {
		item.UpdatedAt = row.UpdatedAt.Time
	}
	return item, nil
}

func projectReportStateFromSQLC(row appsqlc.ProjectReport) ProjectReportState {
	item := ProjectReportState{
		ProjectContract: common.BytesToAddress(row.ProjectContract),
		Report: ProjectReport{
			IsReportEvaluated:          row.IsReportEvaluated,
			IsReportComplete:           row.IsReportComplete,
			IsBlacklistedCreatorWallet: row.IsBlacklistedCreatorWallet,
			IsBlacklistedGenesisWallet: row.IsBlacklistedGenesisWallet,
			IsBlacklistedBytecode:      row.IsBlacklistedBytecode,
			HasMintRisk:                row.HasMintRisk,
		},
	}
	if row.EvaluatedAt.Valid {
		item.EvaluatedAt = row.EvaluatedAt.Time
	}
	if row.UpdatedAt.Valid {
		item.UpdatedAt = row.UpdatedAt.Time
	}
	return item
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
