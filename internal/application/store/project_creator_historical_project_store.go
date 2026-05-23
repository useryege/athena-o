package store

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/common"
)

func (s *SQLStore) ReplaceProjectCreatorHistoricalProjects(ctx context.Context, contract common.Address, items []ProjectCreatorHistoricalProject) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin replace project creator historical projects tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	if _, err := tx.ExecContext(ctx, `
DELETE FROM project_creator_historical_project
WHERE project_contract = $1
`, contract.Bytes()); err != nil {
		return fmt.Errorf("delete project creator historical projects: %w", err)
	}

	for _, item := range items {
		if item.HistoricalProjectContract == (common.Address{}) {
			return errors.New("project creator historical project contract must not be zero")
		}
		if item.RankIndex < 0 {
			return fmt.Errorf("project creator historical project %s rank index must be non-negative", item.HistoricalProjectContract.Hex())
		}
		if _, err := tx.ExecContext(ctx, `
INSERT INTO project_creator_historical_project (
  project_contract,
  historical_project_contract,
  rank_index
) VALUES ($1, $2, $3)
`, contract.Bytes(), item.HistoricalProjectContract.Bytes(), item.RankIndex); err != nil {
			return fmt.Errorf("insert project creator historical project %s: %w", item.HistoricalProjectContract.Hex(), err)
		}
	}

	if _, err := tx.ExecContext(ctx, `
UPDATE project
SET creator_historical_projects_fetched_at = now()
WHERE contract = $1
`, contract.Bytes()); err != nil {
		return fmt.Errorf("update project creator historical projects fetched at: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit replace project creator historical projects tx: %w", err)
	}
	return nil
}

func (s *SQLStore) ListProjectCreatorHistoricalProjectsByContract(ctx context.Context, contract common.Address) ([]ProjectCreatorHistoricalProject, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT
  id,
  project_contract,
  historical_project_contract,
  rank_index,
  created_at
FROM project_creator_historical_project
WHERE project_contract = $1
ORDER BY rank_index ASC, id ASC
`, contract.Bytes())
	if err != nil {
		return nil, fmt.Errorf("list project creator historical projects by contract: %w", err)
	}
	defer rows.Close()

	items := make([]ProjectCreatorHistoricalProject, 0)
	for rows.Next() {
		item, err := scanProjectCreatorHistoricalProjectRow(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate project creator historical projects by contract: %w", err)
	}
	return items, nil
}

func (s *SQLStore) ListProjectCreatorHistoricalProjectsByContracts(ctx context.Context, contracts []common.Address) (map[common.Address][]ProjectCreatorHistoricalProject, error) {
	result := make(map[common.Address][]ProjectCreatorHistoricalProject)
	if len(contracts) == 0 {
		return result, nil
	}

	uniqueContracts := make([]common.Address, 0, len(contracts))
	seen := make(map[common.Address]struct{}, len(contracts))
	for _, contract := range contracts {
		if contract == (common.Address{}) {
			continue
		}
		if _, ok := seen[contract]; ok {
			continue
		}
		seen[contract] = struct{}{}
		uniqueContracts = append(uniqueContracts, contract)
	}
	if len(uniqueContracts) == 0 {
		return result, nil
	}

	placeholders := make([]string, 0, len(uniqueContracts))
	args := make([]any, 0, len(uniqueContracts))
	for i, contract := range uniqueContracts {
		placeholders = append(placeholders, fmt.Sprintf("$%d", i+1))
		args = append(args, contract.Bytes())
	}

	query := fmt.Sprintf(`
SELECT
  id,
  project_contract,
  historical_project_contract,
  rank_index,
  created_at
FROM project_creator_historical_project
WHERE project_contract IN (%s)
ORDER BY project_contract ASC, rank_index ASC, id ASC
`, strings.Join(placeholders, ", "))
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list project creator historical projects by contracts: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		item, err := scanProjectCreatorHistoricalProjectRow(rows)
		if err != nil {
			return nil, err
		}
		result[item.ProjectContract] = append(result[item.ProjectContract], item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate project creator historical projects by contracts: %w", err)
	}
	return result, nil
}

func scanProjectCreatorHistoricalProjectRow(scanner rowScanner) (ProjectCreatorHistoricalProject, error) {
	var (
		item                      ProjectCreatorHistoricalProject
		projectContract           []byte
		historicalProjectContract []byte
	)
	if err := scanner.Scan(
		&item.ID,
		&projectContract,
		&historicalProjectContract,
		&item.RankIndex,
		&item.CreatedAt,
	); err != nil {
		return ProjectCreatorHistoricalProject{}, fmt.Errorf("scan project creator historical project: %w", err)
	}
	item.ProjectContract = common.BytesToAddress(projectContract)
	item.HistoricalProjectContract = common.BytesToAddress(historicalProjectContract)
	return item, nil
}
