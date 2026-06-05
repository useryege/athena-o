package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/jackc/pgx/v5/pgtype"
	appsqlc "github.com/useryege/athena/internal/application/store/sqlc"
)

func (s *SQLStore) ReplaceProjectCreatorHistoricalProjects(ctx context.Context, chainID int64, contract common.Address, items []ProjectCreatorHistoricalProject) error {
	if s.pool == nil {
		return fmt.Errorf("application postgres database is not configured")
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin replace project creator historical projects tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	queries := appsqlc.New(tx)
	chainID = s.chainIDForProject(chainID)
	if len(items) > 0 && items[0].ChainID > 0 {
		chainID = s.chainIDForProject(items[0].ChainID)
	}
	if err := queries.DeleteProjectCreatorHistoricalProjectsByContract(ctx, appsqlc.DeleteProjectCreatorHistoricalProjectsByContractParams{
		ChainID:         chainID,
		ProjectContract: contract.Bytes(),
	}); err != nil {
		return fmt.Errorf("delete project creator historical projects: %w", err)
	}

	for _, item := range items {
		if item.HistoricalProjectContract == (common.Address{}) {
			return errors.New("project creator historical project contract must not be zero")
		}
		if item.RankIndex < 0 {
			return fmt.Errorf("project creator historical project %s rank index must be non-negative", item.HistoricalProjectContract.Hex())
		}
		err = queries.InsertProjectCreatorHistoricalProject(ctx, appsqlc.InsertProjectCreatorHistoricalProjectParams{
			ChainID:                   chainID,
			ProjectContract:           contract.Bytes(),
			HistoricalProjectContract: item.HistoricalProjectContract.Bytes(),
			RankIndex:                 item.RankIndex,
		})
		if err != nil {
			return fmt.Errorf("insert project creator historical project %s: %w", item.HistoricalProjectContract.Hex(), err)
		}
	}

	if err := queries.MarkProjectComponentSuccessNow(ctx, appsqlc.MarkProjectComponentSuccessNowParams{
		ChainID:         chainID,
		ProjectContract: contract.Bytes(),
		Component:       ProjectComponentCreatorHistory,
	}); err != nil {
		return fmt.Errorf("update project creator history component state: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit replace project creator historical projects tx: %w", err)
	}
	return nil
}

func (s *SQLStore) ListProjectCreatorHistoricalProjectsByContract(ctx context.Context, chainID int64, contract common.Address) ([]ProjectCreatorHistoricalProject, error) {
	queries, err := s.querier()
	if err != nil {
		return nil, err
	}
	rows, err := queries.ListProjectCreatorHistoricalProjectsByContract(ctx, appsqlc.ListProjectCreatorHistoricalProjectsByContractParams{
		ChainID:         s.chainIDForProject(chainID),
		ProjectContract: contract.Bytes(),
	})
	if err != nil {
		return nil, fmt.Errorf("list project creator historical projects by contract: %w", err)
	}

	items := make([]ProjectCreatorHistoricalProject, 0, len(rows))
	for _, row := range rows {
		items = append(items, projectCreatorHistoricalProjectFromFields(row.ID, row.ChainID, row.ProjectContract, row.HistoricalProjectContract, row.RankIndex, row.CreatedAt))
	}
	return items, nil
}

func (s *SQLStore) ListProjectCreatorHistoricalProjectsByContracts(ctx context.Context, chainID int64, contracts []common.Address) (map[common.Address][]ProjectCreatorHistoricalProject, error) {
	result := make(map[common.Address][]ProjectCreatorHistoricalProject)
	uniqueContracts := uniqueNonZeroAddresses(contracts)
	if len(uniqueContracts) == 0 {
		return result, nil
	}
	queries, err := s.querier()
	if err != nil {
		return nil, err
	}
	rows, err := queries.ListProjectCreatorHistoricalProjectsByContracts(ctx, appsqlc.ListProjectCreatorHistoricalProjectsByContractsParams{
		ChainID:          s.chainIDForProject(chainID),
		ProjectContracts: addressesToBytes(uniqueContracts),
	})
	if err != nil {
		return nil, fmt.Errorf("list project creator historical projects by contracts: %w", err)
	}

	for _, row := range rows {
		item := projectCreatorHistoricalProjectFromFields(row.ID, row.ChainID, row.ProjectContract, row.HistoricalProjectContract, row.RankIndex, row.CreatedAt)
		result[item.ProjectContract] = append(result[item.ProjectContract], item)
	}
	return result, nil
}

func projectCreatorHistoricalProjectFromFields(id int64, chainID int64, projectContract []byte, historicalProjectContract []byte, rankIndex int32, createdAt pgtype.Timestamptz) ProjectCreatorHistoricalProject {
	return ProjectCreatorHistoricalProject{
		ID:                        id,
		ChainID:                   chainID,
		ProjectContract:           common.BytesToAddress(projectContract),
		HistoricalProjectContract: common.BytesToAddress(historicalProjectContract),
		RankIndex:                 rankIndex,
		CreatedAt:                 createdAt.Time,
	}
}
