package store

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/ethereum/go-ethereum/common"
	"github.com/jackc/pgx/v5"
	appsqlc "github.com/useryege/athena/internal/application/store/sqlc"
)

func (s *SQLStore) SaveProject(ctx context.Context, project ProjectRecord) error {
	txHash := project.TxHash
	if err := validateProjectNumbers(project.BlockNumber, project.BlockTime, project.TxIndex); err != nil {
		return err
	}
	queries, err := s.querier()
	if err != nil {
		return err
	}
	err = queries.InsertProject(ctx, appsqlc.InsertProjectParams{
		ChainID:     s.chainIDForProject(project.ChainID),
		BlockNumber: int64(project.BlockNumber),
		BlockTime:   int64(project.BlockTime),
		Contract:    project.Contract.Bytes(),
		Creator:     project.Creator.Bytes(),
		TxHash:      txHash.Bytes(),
		TxIndex:     int64(project.TxIndex),
	})
	if err != nil {
		return fmt.Errorf("save project: %w", err)
	}
	return nil
}

func (s *SQLStore) GetMaxProjectBlockNumber(ctx context.Context, chainID int64) (uint64, bool, error) {
	queries, err := s.querier()
	if err != nil {
		return 0, false, err
	}
	row, err := queries.GetMaxProjectBlockNumber(ctx, s.chainIDForProject(chainID))
	if err != nil {
		return 0, false, fmt.Errorf("get max project block number: %w", err)
	}
	if !row.HasValue {
		return 0, false, nil
	}
	if row.MaxBlock < 0 {
		return 0, false, fmt.Errorf("project block number %d is negative", row.MaxBlock)
	}
	return uint64(row.MaxBlock), true, nil
}

func (s *SQLStore) ListProjects(ctx context.Context, chainID int64) ([]ProjectRecord, error) {
	queries, err := s.querier()
	if err != nil {
		return nil, err
	}
	rows, err := queries.ListProjects(ctx, s.chainIDForProject(chainID))
	if err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}
	items := make([]ProjectRecord, 0, len(rows))
	for _, row := range rows {
		item, err := projectRecordFromFields(row.ChainID, row.BlockNumber, row.BlockTime, row.Contract, row.Creator, row.TxHash, row.TxIndex, row.CreatedAt)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func (s *SQLStore) ListProjectsPage(ctx context.Context, chainID int64, page int32, pageSize int32) ([]ProjectRecord, int64, int32, int32, error) {
	queries, err := s.querier()
	if err != nil {
		return nil, 0, 0, 0, err
	}
	page, pageSize = normalizePage(page, pageSize)
	chainID = s.chainIDForProject(chainID)
	total, err := queries.CountProjects(ctx, chainID)
	if err != nil {
		return nil, 0, page, pageSize, fmt.Errorf("count projects: %w", err)
	}
	offset := int64(page-1) * int64(pageSize)
	if offset > math.MaxInt32 {
		return nil, 0, page, pageSize, fmt.Errorf("project page offset %d exceeds int32", offset)
	}
	rows, err := queries.ListProjectsPage(ctx, appsqlc.ListProjectsPageParams{
		ChainID:     chainID,
		LimitCount:  pageSize,
		OffsetCount: int32(offset),
	})
	if err != nil {
		return nil, 0, page, pageSize, fmt.Errorf("list projects page: %w", err)
	}
	items := make([]ProjectRecord, 0, len(rows))
	for _, row := range rows {
		item, err := projectRecordFromFields(row.ChainID, row.BlockNumber, row.BlockTime, row.Contract, row.Creator, row.TxHash, row.TxIndex, row.CreatedAt)
		if err != nil {
			return nil, 0, page, pageSize, err
		}
		items = append(items, item)
	}
	return items, total, page, pageSize, nil
}

func (s *SQLStore) GetProjectByContract(ctx context.Context, chainID int64, contract common.Address) (*ProjectRecord, error) {
	queries, err := s.querier()
	if err != nil {
		return nil, err
	}
	row, err := queries.GetProjectByContract(ctx, appsqlc.GetProjectByContractParams{
		ChainID:  s.chainIDForProject(chainID),
		Contract: contract.Bytes(),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get project by contract: %w", err)
	}
	project, err := projectRecordFromFields(row.ChainID, row.BlockNumber, row.BlockTime, row.Contract, row.Creator, row.TxHash, row.TxIndex, row.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &project, nil
}

func (s *SQLStore) ListProjectsByCreatorBefore(ctx context.Context, chainID int64, creator common.Address, blockNumber uint64, txIndex uint64) ([]ProjectRecord, error) {
	if err := validateProjectOrderNumbers(blockNumber, txIndex); err != nil {
		return nil, err
	}
	queries, err := s.querier()
	if err != nil {
		return nil, err
	}
	rows, err := queries.ListProjectsByCreatorBefore(ctx, appsqlc.ListProjectsByCreatorBeforeParams{
		ChainID:     s.chainIDForProject(chainID),
		Creator:     creator.Bytes(),
		BlockNumber: int64(blockNumber),
		TxIndex:     int64(txIndex),
	})
	if err != nil {
		return nil, fmt.Errorf("list projects by creator before: %w", err)
	}
	items := make([]ProjectRecord, 0, len(rows))
	for _, row := range rows {
		item, err := projectRecordFromFields(row.ChainID, row.BlockNumber, row.BlockTime, row.Contract, row.Creator, row.TxHash, row.TxIndex, row.CreatedAt)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}
