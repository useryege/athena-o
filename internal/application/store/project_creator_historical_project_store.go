package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	appsqlc "github.com/useryege/athena/internal/application/store/sqlc"
)

func (s *SQLStore) ReplaceProjectCreatorHistoricalProjects(ctx context.Context, contract common.Address, items []ProjectCreatorHistoricalProject) error {
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
	if err := queries.DeleteProjectCreatorHistoricalProjectsByContract(ctx, contract.Bytes()); err != nil {
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
			ProjectContract:           contract.Bytes(),
			HistoricalProjectContract: item.HistoricalProjectContract.Bytes(),
			RankIndex:                 item.RankIndex,
		})
		if err != nil {
			return fmt.Errorf("insert project creator historical project %s: %w", item.HistoricalProjectContract.Hex(), err)
		}
	}

	if err := queries.MarkProjectComponentSuccessNow(ctx, appsqlc.MarkProjectComponentSuccessNowParams{
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

func (s *SQLStore) ListProjectCreatorHistoricalProjectsByContract(ctx context.Context, contract common.Address) ([]ProjectCreatorHistoricalProject, error) {
	queries, err := s.querier()
	if err != nil {
		return nil, err
	}
	rows, err := queries.ListProjectCreatorHistoricalProjectsByContract(ctx, contract.Bytes())
	if err != nil {
		return nil, fmt.Errorf("list project creator historical projects by contract: %w", err)
	}

	items := make([]ProjectCreatorHistoricalProject, 0, len(rows))
	for _, row := range rows {
		items = append(items, projectCreatorHistoricalProjectFromSQLC(row))
	}
	return items, nil
}

func (s *SQLStore) ListProjectCreatorHistoricalProjectsByContracts(ctx context.Context, contracts []common.Address) (map[common.Address][]ProjectCreatorHistoricalProject, error) {
	result := make(map[common.Address][]ProjectCreatorHistoricalProject)
	uniqueContracts := uniqueNonZeroAddresses(contracts)
	if len(uniqueContracts) == 0 {
		return result, nil
	}
	queries, err := s.querier()
	if err != nil {
		return nil, err
	}
	rows, err := queries.ListProjectCreatorHistoricalProjectsByContracts(ctx, addressesToBytes(uniqueContracts))
	if err != nil {
		return nil, fmt.Errorf("list project creator historical projects by contracts: %w", err)
	}

	for _, row := range rows {
		item := projectCreatorHistoricalProjectFromSQLC(row)
		result[item.ProjectContract] = append(result[item.ProjectContract], item)
	}
	return result, nil
}

func projectCreatorHistoricalProjectFromSQLC(row appsqlc.ProjectCreatorHistoricalProject) ProjectCreatorHistoricalProject {
	return ProjectCreatorHistoricalProject{
		ID:                        row.ID,
		ProjectContract:           common.BytesToAddress(row.ProjectContract),
		HistoricalProjectContract: common.BytesToAddress(row.HistoricalProjectContract),
		RankIndex:                 row.RankIndex,
		CreatedAt:                 row.CreatedAt.Time,
	}
}
