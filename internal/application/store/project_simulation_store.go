package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/useryege/athena/internal/application/model"
	appsqlc "github.com/useryege/athena/internal/application/store/sqlc"
)

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

func (s *SQLStore) UpdateProjectCreatorResult(ctx context.Context, chainID int64, contract common.Address, result model.SimulateResult) error {
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
		Result: model.SimulateResult{
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
