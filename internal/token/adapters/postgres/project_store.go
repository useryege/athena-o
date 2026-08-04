package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	tokensqlc "github.com/useryege/athena/internal/token/adapters/postgres/sqlc"
	"github.com/useryege/athena/internal/token/catalog"
	"github.com/useryege/athena/internal/token/shared"
)

func (s *CatalogRepository) UpsertProject(ctx context.Context, item catalog.Project) (*catalog.Project, error) {
	q, err := s.querier()
	if err != nil {
		return nil, err
	}
	txIndex, err := uint64ToInt64("tx_index", item.TxIndex)
	if err != nil {
		return nil, err
	}
	blockNumber, err := uint64ToInt64("block_number", item.BlockNumber)
	if err != nil {
		return nil, err
	}
	blockTime, err := uint64ToInt64("block_time", item.BlockTime)
	if err != nil {
		return nil, err
	}
	row, err := q.UpsertProject(ctx, tokensqlc.UpsertProjectParams{
		ChainID:     item.ChainID,
		Contract:    item.Contract.Bytes(),
		TxSender:    item.TxSender.Bytes(),
		TxHash:      item.TxHash.Bytes(),
		TxIndex:     txIndex,
		BlockNumber: blockNumber,
		BlockTime:   blockTime,
		CodeHash:    item.CodeHash.Bytes(),
		Name:        item.Name,
		Symbol:      item.Symbol,
		Decimals:    int16(item.Decimals),
		TotalSupply: numericFromBigInt(item.TotalSupply),
		WethPair:    optionalAddressBytes(item.WethPair),
		UsdtPair:    optionalAddressBytes(item.UsdtPair),
	})
	if err != nil {
		return nil, fmt.Errorf("upsert project: %w", err)
	}
	mapped, err := mapProject(row)
	if err != nil {
		return nil, fmt.Errorf("map project: %w", err)
	}
	return mapped, nil
}

func (s *CatalogRepository) GetProject(ctx context.Context, id int64) (*catalog.Project, error) {
	q, err := s.querier()
	if err != nil {
		return nil, err
	}
	row, err := q.GetProject(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get project: %w", err)
	}
	item, err := mapProject(row)
	if err != nil {
		return nil, fmt.Errorf("map project: %w", err)
	}
	return item, nil
}

func (s *CatalogRepository) GetProjectByContract(ctx context.Context, chainID int64, contract shared.Address) (*catalog.Project, error) {
	q, err := s.querier()
	if err != nil {
		return nil, err
	}
	row, err := q.GetProjectByContract(ctx, tokensqlc.GetProjectByContractParams{
		ChainID:  chainID,
		Contract: contract.Bytes(),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get project by contract: %w", err)
	}
	item, err := mapProject(row)
	if err != nil {
		return nil, fmt.Errorf("map project: %w", err)
	}
	return item, nil
}

func (s *CatalogRepository) ListProjects(ctx context.Context, chainID int64) ([]catalog.Project, error) {
	q, err := s.querier()
	if err != nil {
		return nil, err
	}
	rows, err := q.ListProjects(ctx, chainID)
	if err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}
	items, err := mapProjects(rows)
	if err != nil {
		return nil, fmt.Errorf("map projects: %w", err)
	}
	return items, nil
}

func (s *CatalogRepository) DeleteProject(ctx context.Context, id int64) (int64, error) {
	q, err := s.querier()
	if err != nil {
		return 0, err
	}
	rowsAffected, err := q.DeleteProject(ctx, id)
	if err != nil {
		return 0, fmt.Errorf("delete project: %w", err)
	}
	return rowsAffected, nil
}
