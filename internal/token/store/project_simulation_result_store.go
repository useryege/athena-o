package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/jackc/pgx/v5"
	tokensqlc "github.com/useryege/athena/internal/token/store/sqlc"
)

func (s *SQLStore) UpsertProjectSimulationResult(ctx context.Context, item ProjectSimulationResult) (*ProjectSimulationResult, error) {
	q, err := s.querier()
	if err != nil {
		return nil, err
	}
	fetchedAt := item.FetchedAt
	if fetchedAt.IsZero() {
		fetchedAt = time.Now().UTC()
	}
	row, err := q.UpsertProjectSimulationResult(ctx, tokensqlc.UpsertProjectSimulationResultParams{
		ProjectID:                          item.ProjectID,
		Wallet:                             item.Wallet.Bytes(),
		CanMintFromDeadViaTransferFrom:     item.CanMintFromDeadViaTransferFrom,
		CanMintFromZeroViaTransferFrom:     item.CanMintFromZeroViaTransferFrom,
		CanMintFromWethPairViaTransferFrom: item.CanMintFromWethPairViaTransferFrom,
		CanMintFromUsdtPairViaTransferFrom: item.CanMintFromUsdtPairViaTransferFrom,
		CanMintViaTransferToWethPair:       item.CanMintViaTransferToWethPair,
		CanMintViaTransferToUsdtPair:       item.CanMintViaTransferToUsdtPair,
		FetchedAt:                          nullableTime(fetchedAt),
	})
	if err != nil {
		return nil, fmt.Errorf("upsert project simulation result: %w", err)
	}
	return mapProjectSimulationResult(row), nil
}

func (s *SQLStore) GetProjectSimulationResult(ctx context.Context, projectID int64, wallet common.Address) (*ProjectSimulationResult, error) {
	q, err := s.querier()
	if err != nil {
		return nil, err
	}
	row, err := q.GetProjectSimulationResult(ctx, tokensqlc.GetProjectSimulationResultParams{
		ProjectID: projectID,
		Wallet:    wallet.Bytes(),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get project simulation result: %w", err)
	}
	return mapProjectSimulationResult(row), nil
}

func (s *SQLStore) ListProjectSimulationResultsByProject(ctx context.Context, projectID int64) ([]ProjectSimulationResult, error) {
	q, err := s.querier()
	if err != nil {
		return nil, err
	}
	rows, err := q.ListProjectSimulationResultsByProject(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("list project simulation results by project: %w", err)
	}
	return mapProjectSimulationResults(rows), nil
}

func (s *SQLStore) ListProjectSimulationResultsByWallet(ctx context.Context, wallet common.Address) ([]ProjectSimulationResult, error) {
	q, err := s.querier()
	if err != nil {
		return nil, err
	}
	rows, err := q.ListProjectSimulationResultsByWallet(ctx, wallet.Bytes())
	if err != nil {
		return nil, fmt.Errorf("list project simulation results by wallet: %w", err)
	}
	return mapProjectSimulationResults(rows), nil
}

func (s *SQLStore) DeleteProjectSimulationResult(ctx context.Context, projectID int64, wallet common.Address) (int64, error) {
	q, err := s.querier()
	if err != nil {
		return 0, err
	}
	rowsAffected, err := q.DeleteProjectSimulationResult(ctx, tokensqlc.DeleteProjectSimulationResultParams{
		ProjectID: projectID,
		Wallet:    wallet.Bytes(),
	})
	if err != nil {
		return 0, fmt.Errorf("delete project simulation result: %w", err)
	}
	return rowsAffected, nil
}

func (s *SQLStore) DeleteProjectSimulationResultsByProject(ctx context.Context, projectID int64) (int64, error) {
	q, err := s.querier()
	if err != nil {
		return 0, err
	}
	rowsAffected, err := q.DeleteProjectSimulationResultsByProject(ctx, projectID)
	if err != nil {
		return 0, fmt.Errorf("delete project simulation results by project: %w", err)
	}
	return rowsAffected, nil
}
