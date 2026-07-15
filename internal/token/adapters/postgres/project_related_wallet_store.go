package postgres

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	tokensqlc "github.com/useryege/athena/internal/token/adapters/postgres/sqlc"
	"github.com/useryege/athena/internal/token/catalog"
)

func (s *Database) UpsertProjectRelatedWallet(ctx context.Context, item catalog.ProjectRelatedWallet) (*catalog.ProjectRelatedWallet, error) {
	q, err := s.querier()
	if err != nil {
		return nil, err
	}
	row, err := q.UpsertProjectRelatedWallet(ctx, tokensqlc.UpsertProjectRelatedWalletParams{
		ProjectID: item.ProjectID,
		Wallet:    item.Wallet.Bytes(),
		Role:      string(item.Role),
	})
	if err != nil {
		return nil, fmt.Errorf("upsert project related wallet: %w", err)
	}
	return mapProjectRelatedWallet(row), nil
}

func (s *Database) ListProjectRelatedWalletsByProject(ctx context.Context, projectID int64) ([]catalog.ProjectRelatedWallet, error) {
	q, err := s.querier()
	if err != nil {
		return nil, err
	}
	rows, err := q.ListProjectRelatedWalletsByProject(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("list project related wallets by project: %w", err)
	}
	return mapProjectRelatedWallets(rows), nil
}

func (s *Database) ListProjectRelatedWalletsByWallet(ctx context.Context, wallet common.Address) ([]catalog.ProjectRelatedWallet, error) {
	q, err := s.querier()
	if err != nil {
		return nil, err
	}
	rows, err := q.ListProjectRelatedWalletsByWallet(ctx, wallet.Bytes())
	if err != nil {
		return nil, fmt.Errorf("list project related wallets by wallet: %w", err)
	}
	return mapProjectRelatedWallets(rows), nil
}

func (s *Database) DeleteProjectRelatedWallet(ctx context.Context, projectID int64, wallet common.Address, role catalog.RelatedWalletRole) (int64, error) {
	q, err := s.querier()
	if err != nil {
		return 0, err
	}
	rowsAffected, err := q.DeleteProjectRelatedWallet(ctx, tokensqlc.DeleteProjectRelatedWalletParams{
		ProjectID: projectID,
		Wallet:    wallet.Bytes(),
		Role:      string(role),
	})
	if err != nil {
		return 0, fmt.Errorf("delete project related wallet: %w", err)
	}
	return rowsAffected, nil
}

func (s *Database) DeleteProjectRelatedWalletsByProject(ctx context.Context, projectID int64) (int64, error) {
	q, err := s.querier()
	if err != nil {
		return 0, err
	}
	rowsAffected, err := q.DeleteProjectRelatedWalletsByProject(ctx, projectID)
	if err != nil {
		return 0, fmt.Errorf("delete project related wallets by project: %w", err)
	}
	return rowsAffected, nil
}
