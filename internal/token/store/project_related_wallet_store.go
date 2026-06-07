package store

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	tokensqlc "github.com/useryege/athena/internal/token/store/sqlc"
)

func (s *SQLStore) UpsertProjectRelatedWallet(ctx context.Context, item ProjectRelatedWallet) (*ProjectRelatedWallet, error) {
	q, err := s.querier()
	if err != nil {
		return nil, err
	}
	row, err := q.UpsertProjectRelatedWallet(ctx, tokensqlc.UpsertProjectRelatedWalletParams{
		ProjectID: item.ProjectID,
		Wallet:    item.Wallet.Bytes(),
		Role:      item.Role,
	})
	if err != nil {
		return nil, fmt.Errorf("upsert project related wallet: %w", err)
	}
	return mapProjectRelatedWallet(row), nil
}

func (s *SQLStore) ListProjectRelatedWalletsByProject(ctx context.Context, projectID int64) ([]ProjectRelatedWallet, error) {
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

func (s *SQLStore) ListProjectRelatedWalletsByWallet(ctx context.Context, wallet common.Address) ([]ProjectRelatedWallet, error) {
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

func (s *SQLStore) DeleteProjectRelatedWallet(ctx context.Context, projectID int64, wallet common.Address, role string) (int64, error) {
	q, err := s.querier()
	if err != nil {
		return 0, err
	}
	rowsAffected, err := q.DeleteProjectRelatedWallet(ctx, tokensqlc.DeleteProjectRelatedWalletParams{
		ProjectID: projectID,
		Wallet:    wallet.Bytes(),
		Role:      role,
	})
	if err != nil {
		return 0, fmt.Errorf("delete project related wallet: %w", err)
	}
	return rowsAffected, nil
}

func (s *SQLStore) DeleteProjectRelatedWalletsByProject(ctx context.Context, projectID int64) (int64, error) {
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
