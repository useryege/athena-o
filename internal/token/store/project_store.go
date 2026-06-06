package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/jackc/pgx/v5"
	tokensqlc "github.com/useryege/athena/internal/token/store/sqlc"
)

func (s *SQLStore) UpsertProject(ctx context.Context, item Project) (*Project, error) {
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
		Creator:     item.Creator.Bytes(),
		TxHash:      item.TxHash.Bytes(),
		TxIndex:     txIndex,
		BlockNumber: blockNumber,
		BlockTime:   blockTime,
		CodeHash:    item.CodeHash.Bytes(),
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

func (s *SQLStore) GetProject(ctx context.Context, id int64) (*Project, error) {
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

func (s *SQLStore) GetProjectByContract(ctx context.Context, chainID int64, contract common.Address) (*Project, error) {
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

func (s *SQLStore) CountProjects(ctx context.Context, chainID int64, codeHash common.Hash, contract common.Address) (int64, error) {
	q, err := s.querier()
	if err != nil {
		return 0, err
	}
	total, err := q.CountProjects(ctx, tokensqlc.CountProjectsParams{
		ChainID:  chainID,
		CodeHash: optionalHashBytes(codeHash),
		Contract: optionalAddressBytes(contract),
	})
	if err != nil {
		return 0, fmt.Errorf("count projects: %w", err)
	}
	return total, nil
}

func (s *SQLStore) ListProjects(ctx context.Context, chainID int64) ([]Project, error) {
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

func (s *SQLStore) ListProjectsPage(ctx context.Context, chainID int64, codeHash common.Hash, contract common.Address, page, pageSize int32) (*ProjectPage, error) {
	q, err := s.querier()
	if err != nil {
		return nil, err
	}
	page, pageSize, offset := normalizePage(page, pageSize)
	codeHashFilter := optionalHashBytes(codeHash)
	contractFilter := optionalAddressBytes(contract)
	total, err := q.CountProjects(ctx, tokensqlc.CountProjectsParams{
		ChainID:  chainID,
		CodeHash: codeHashFilter,
		Contract: contractFilter,
	})
	if err != nil {
		return nil, fmt.Errorf("count projects: %w", err)
	}
	rows, err := q.ListProjectsPage(ctx, tokensqlc.ListProjectsPageParams{
		ChainID:  chainID,
		CodeHash: codeHashFilter,
		Contract: contractFilter,
		Offset:   offset,
		Limit:    pageSize,
	})
	if err != nil {
		return nil, fmt.Errorf("list projects page: %w", err)
	}
	items, err := mapProjects(rows)
	if err != nil {
		return nil, fmt.Errorf("map projects: %w", err)
	}
	return &ProjectPage{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func (s *SQLStore) DeleteProject(ctx context.Context, id int64) (int64, error) {
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
